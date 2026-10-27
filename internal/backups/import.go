package backups

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
)

func stringValue(v any) string {
	if value, ok := v.(string); ok {
		return value
	}
	return ""
}
func intValue(v any) int64 {
	if n, ok := v.(json.Number); ok {
		x, _ := n.Int64()
		return x
	}
	return 0
}

// Decode bounded JSON-lines without trusting archive-provided column names.
func decodeTable(pkg *Package, name string, visit func(map[string]any) error) error {
	if !tableSet[name] {
		return fmt.Errorf("unlisted table")
	}
	entry := pkg.Files["tables/"+name+".jsonl"]
	if entry == nil {
		return fmt.Errorf("missing table")
	}
	file, err := entry.Open()
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 64<<20)
	if !scanner.Scan() {
		return fmt.Errorf("missing table header")
	}
	var columns []string
	if err = json.Unmarshal(scanner.Bytes(), &columns); err != nil {
		return err
	}
	if len(columns) == 0 || len(columns) > 128 {
		return fmt.Errorf("invalid table header")
	}
	seen := map[string]bool{}
	for _, column := range columns {
		if column == "" || seen[column] {
			return fmt.Errorf("invalid table column")
		}
		seen[column] = true
	}
	var count int64
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			return fmt.Errorf("empty table row")
		}
		decoder := json.NewDecoder(strings.NewReader(string(line)))
		decoder.UseNumber()
		var values []any
		if err = decoder.Decode(&values); err != nil {
			return err
		}
		if len(values) != len(columns) {
			return fmt.Errorf("invalid table row")
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = values[i]
		}
		if err = visit(row); err != nil {
			return err
		}
		count++
		if count > 5000000 {
			return fmt.Errorf("too many table rows")
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if count != pkg.Manifest.Tables[name] {
		return fmt.Errorf("table row count mismatch: %s", name)
	}
	return nil
}

func (s *Service) importLogical(ctx context.Context, op *Operation, pkg *Package, idx *restoreIndex) error {
	// Only the writer connection changes its FK mode. The entire logical replace
	// remains one transaction; FK and SQLite structural checks precede commit.
	if _, err := s.DB.Writer.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	defer s.DB.Writer.ExecContext(context.Background(), "PRAGMA foreign_keys=ON")
	tx, err := s.DB.Writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var username, display, hash string
	err = tx.QueryRowContext(ctx, `SELECT username,display_name,password_hash FROM users WHERE id=? AND EXISTS(SELECT 1 FROM global_role_assignments WHERE user_id=users.id AND role_id='installation_admin')`, op.Actor).Scan(&username, &display, &hash)
	if err != nil {
		return fmt.Errorf("destination administrator unavailable: %w", err)
	}
	// Delete only recoverable data and transient state. Never touch license_*.
	transient := []string{"sessions", "authentication_attempts", "authentication_throttles", "idempotency_requests", "file_scan_cache", "root_watch_recovery", "preview_cache", "diagnostic_events"}
	for _, table := range transient {
		if _, err = tx.ExecContext(ctx, "DELETE FROM \""+table+"\""); err != nil {
			return err
		}
	}
	for i := len(tables) - 1; i >= 0; i-- {
		table := tables[i]
		if table == "audit_events" {
			continue
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM \""+table+"\""); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	for _, table := range tables {
		if err = ctx.Err(); err != nil {
			return err
		}
		columns, err := tableColumns(ctx, tx, table)
		if err != nil {
			return err
		}
		entry := pkg.Files["tables/"+table+".jsonl"]
		if entry == nil {
			return fmt.Errorf("missing required table")
		}
		if err = importTable(ctx, tx, pkg, entry, table, columns, idx); err != nil {
			return fmt.Errorf("restore %s: %w", table, err)
		}
	}
	// Preserve a working local administrator even if the source came from another
	// machine or was captured before this administrator changed credentials.
	newID := domain.NewID()
	newUser := "restore_" + strings.ToLower(strings.ReplaceAll(op.ID[:12], "-", ""))
	_, err = tx.ExecContext(ctx, `INSERT INTO users(id,username,username_key,display_name,password_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, newID, newUser, newUser, display+" (recuperación)", hash, domain.Timestamp(time.Now()), domain.Timestamp(time.Now()))
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO role_permissions(role_id,permission_key) VALUES ('installation_admin','backup.manage'),('installation_admin','restore.manage')"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO global_role_assignments(user_id,role_id) VALUES(?,'installation_admin')", newID); err != nil {
		return err
	}
	op.RecoveryUser = newUser
	if err = postImport(ctx, tx, op, idx, pkg.Manifest.Kind, newID); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	if rows.Next() {
		rows.Close()
		return fmt.Errorf("restored data violate foreign keys")
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var integrity string
	if err = tx.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fmt.Errorf("SQLite integrity check failed")
	}
	if err = audit.Append(ctx, tx, time.Now(), audit.Event{Type: "backup.restore_committed", SystemActor: true, Metadata: domain.RequestMetadata{RequestID: domain.NewID(), ObservedIP: "local"}, Details: map[string]any{"operation_id": op.ID, "kind": pkg.Manifest.Kind, "requested_by": op.Actor, "recovery_user": newUser}}); err != nil {
		return err
	}
	return tx.Commit()
}

func importTable(ctx context.Context, tx *sql.Tx, pkg *Package, entry *zip.File, table string, columns []string, idx *restoreIndex) error {
	// Validate against the destination schema, not the archive's self-description.
	file, err := entry.Open()
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 64<<20)
	if !scanner.Scan() {
		return fmt.Errorf("table header missing")
	}
	var header []string
	if err = json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return err
	}
	// Pre-description AIDEX archives contain the original library columns.
	// A missing trailing description uses the destination's empty default.
	legacyLibrary := table == "libraries" && len(header)+1 == len(columns) && columns[len(columns)-1] == "description"
	if len(header) != len(columns) && !legacyLibrary {
		return fmt.Errorf("column count mismatch")
	}
	for i := range header {
		if header[i] != columns[i] {
			return fmt.Errorf("column mismatch")
		}
	}
	placeholders := make([]string, len(header))
	quoted := make([]string, len(header))
	for i, name := range header {
		placeholders[i] = "?"
		quoted[i] = "\"" + name + "\""
	}
	verb := "INSERT"
	if table == "audit_events" {
		verb = "INSERT OR IGNORE"
	}
	statement, err := tx.PrepareContext(ctx, verb+" INTO \""+table+"\"("+strings.Join(quoted, ",")+") VALUES("+strings.Join(placeholders, ",")+")")
	if err != nil {
		return err
	}
	defer statement.Close()
	var count int64
	for scanner.Scan() {
		if err = ctx.Err(); err != nil {
			return err
		}
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		decoder.UseNumber()
		var values []any
		if err = decoder.Decode(&values); err != nil {
			return err
		}
		if len(values) != len(header) {
			return fmt.Errorf("invalid row")
		}
		row := map[string]any{}
		for i, name := range header {
			row[name] = values[i]
		}
		if table == "audit_events" && strings.HasPrefix(stringValue(row["event_type"]), "license.") {
			return fmt.Errorf("license evidence forbidden")
		}
		normalizeRow(table, row, idx, pkg.Manifest.Kind)
		for i, name := range header {
			value := row[name]
			if number, ok := value.(json.Number); ok {
				if integer, e := number.Int64(); e == nil {
					value = integer
				} else if floating, e := number.Float64(); e == nil {
					value = floating
				} else {
					return e
				}
			}
			values[i] = value
		}
		if _, err = statement.ExecContext(ctx, values...); err != nil {
			return err
		}
		count++
		if count > 5000000 {
			return fmt.Errorf("table row limit exceeded")
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if count != pkg.Manifest.Tables[table] {
		return fmt.Errorf("table count mismatch")
	}
	return nil
}

func normalizeRow(table string, row map[string]any, idx *restoreIndex, kind string) {
	switch table {
	case "storage_roots":
		root := idx.roots[stringValue(row["id"])]
		if root == nil {
			return
		}
		row["directory_identity"] = ""
		row["volume_identity"] = nil
		row["last_verified_at"] = nil
		row["last_scan_at"] = nil
		row["last_error_code"] = "RESTORE_REVERIFY"
		if root.Verified {
			row["canonical_path"] = root.NewPath
			row["comparison_key"] = comparison(root.NewPath, root.Sensitive)
			row["directory_identity"] = root.Identity
			row["case_sensitive"] = boolInt(root.Sensitive)
			if root.NewPath != root.OldPath {
				row["configuration_revision"] = root.Revision + 1
			}
			if root.Status == "active" || root.Status == "inaccessible" {
				row["status"] = "active"
				row["last_error_code"] = nil
				if stringValue(row["watch_mode"]) == "paused" {
					row["watch_mode"] = "polling"
				}
			}
		}
		if !root.Verified && (root.Status == "active" || root.Status == "inaccessible") {
			row["status"] = "inaccessible"
			row["watch_mode"] = "paused"
		}
	case "physical_files":
		file := idx.files[stringValue(row["id"])]
		if file != nil && file.Confirmed {
			row["availability"] = "available"
			row["os_identity_key"] = file.Identity
			row["integrity_status"] = "verified"
		} else {
			row["availability"] = "missing"
			row["os_identity_key"] = nil
			row["integrity_status"] = "unknown"
		}
	case "physical_file_locations":
		file := idx.files[stringValue(row["physical_file_id"])]
		if kind == "full" && file != nil && file.NewPath != "" {
			if stringValue(row["id"]) != file.PrimaryLocation {
				if row["retired_at"] == nil {
					row["retired_at"] = domain.Timestamp(time.Now())
				}
				return
			}
			if stringValue(row["root_id"]) != file.Root {
				return
			}
			row["canonical_path"] = file.NewPath
			row["comparison_key"] = comparison(file.NewPath, idx.roots[file.Root].Sensitive)
			row["relative_path"] = file.Relative
			row["root_configuration_revision"] = idx.roots[file.Root].Revision + 1
			return
		}
		root := idx.roots[stringValue(row["root_id"])]
		if root != nil && root.Verified && root.NewPath != root.OldPath {
			relative := filepath.FromSlash(stringValue(row["relative_path"]))
			row["canonical_path"] = filepath.Join(root.NewPath, relative)
			row["comparison_key"] = comparison(stringValue(row["canonical_path"]), root.Sensitive)
			row["root_configuration_revision"] = root.Revision + 1
		}
	case "upload_items":
		upload := idx.uploads[stringValue(row["id"])]
		if upload == nil {
			return
		}
		if kind == "full" && upload.NewLocator != "" {
			row["private_temporary_locator"] = upload.NewLocator
		} else {
			row["status"] = "failed"
			row["error_code"] = "RESTORE_FILE_MISSING"
		}
	case "jobs":
		if status := stringValue(row["status"]); status == "queued" || status == "running" || status == "retry_wait" {
			row["status"] = "cancelled"
			row["lease_owner"] = nil
			row["lease_expires_at"] = nil
			row["last_error_code"] = "RESTORE_REVIEW"
		}
	case "root_scans":
		if stringValue(row["status"]) == "running" {
			row["status"] = "failed"
			row["completed_at"] = domain.Timestamp(time.Now())
			row["can_confirm_absence"] = int64(0)
		}
	case "root_operations":
		if status := stringValue(row["state"]); status == "running" || status == "confirmed" || status == "planned" {
			row["state"] = "failed"
		}
	case "materializations":
		if stringValue(row["state"]) == "planned" {
			row["state"] = "failed"
			row["error_code"] = "RESTORE_REVIEW"
		}
	case "deletion_cleanup":
		row["status"] = "attention"
		row["error_code"] = "CLEANUP_RESTORE_REVIEW"
		row["container_path"] = ""
		row["container_identity"] = ""
		row["file_identity"] = ""
		row["sha256"] = ""
		row["quarantine"] = ""
	}
}
func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
func comparison(path string, sensitive bool) string {
	if sensitive {
		return path
	}
	return strings.ToLower(path)
}

func postImport(ctx context.Context, tx *sql.Tx, op *Operation, idx *restoreIndex, kind, actor string) error {
	for _, root := range idx.roots {
		if !root.Verified || root.NewPath == root.OldPath {
			continue
		}
		var configJSON string
		err := tx.QueryRowContext(ctx, "SELECT configuration_json FROM storage_root_configuration_versions WHERE root_id=? ORDER BY revision DESC LIMIT 1", root.ID).Scan(&configJSON)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO storage_root_configuration_versions(root_id,revision,canonical_path,comparison_key,configuration_json,created_at,created_by) VALUES(?,?,?,?,?,?,?)", root.ID, root.Revision+1, root.NewPath, comparison(root.NewPath, root.Sensitive), configJSON, domain.Timestamp(time.Now()), actor); err != nil {
			return err
		}
	}
	if kind == "full" {
		for _, file := range idx.files {
			if file.NewPath == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, "UPDATE physical_files SET availability='available',os_identity_key=?,integrity_status='verified' WHERE id=?", file.Identity, file.ID); err != nil {
				return err
			}
		}
		for _, upload := range idx.uploads {
			if upload.NewLocator == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, "UPDATE physical_files SET availability='staged' WHERE id=(SELECT physical_file_id FROM documents WHERE id=(SELECT document_id FROM upload_items WHERE id=?)) AND primary_location_id IS NULL", upload.ID); err != nil {
				return err
			}
		}
	}
	// No old leases or source OS identities are allowed to execute automatically.
	if _, err := tx.ExecContext(ctx, "UPDATE job_attempts SET finished_at=?,error_code='RESTORE_REVIEW' WHERE finished_at IS NULL", domain.Timestamp(time.Now())); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE root_plans SET expires_at=? WHERE committed_root_id IS NULL", domain.Timestamp(time.Unix(0, 0))); err != nil {
		return err
	}
	return nil
}
