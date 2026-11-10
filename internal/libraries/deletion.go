package libraries

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

type DeletionPlan struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	EntityID    string         `json:"entity_id"`
	LibraryID   string         `json:"library_id"`
	Name        string         `json:"name"`
	LibraryName string         `json:"library_name"`
	State       string         `json:"state"`
	Counts      map[string]int `json:"counts"`
	Locations   []string       `json:"locations"`
	Pending     int            `json:"pending_cleanup"`
	Diagnostics []string       `json:"cleanup_diagnostics"`
	Issues      int            `json:"cleanup_issues"`
}

func (s *Service) Deletions(ctx context.Context, p domain.Principal, cursor string) (Page[DeletionPlan], error) {
	result := Page[DeletionPlan]{Items: []DeletionPlan{}}
	current, err := s.Identity.LoadPrincipal(ctx, s.Database.Reader, p.TokenDigest)
	if err != nil {
		return result, err
	}
	scope := current.User.ID + ":deletions"
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := s.Database.Reader.QueryContext(ctx, `SELECT id,created_at FROM entity_deletions e WHERE execution_status<>'unconfirmed' AND (? OR (kind='root' AND actor_user_id=? AND EXISTS(SELECT 1 FROM library_role_assignments a JOIN role_permissions rp USING(role_id) JOIN libraries b ON b.id=a.library_id WHERE a.library_id=e.library_id AND a.user_id=? AND rp.permission_key='storage.manage_roots' AND b.disabled_at IS NULL))) AND (?='' OR created_at<? OR (created_at=? AND id<?)) ORDER BY created_at DESC,id DESC LIMIT 51`, current.Can("libraries.delete"), current.User.ID, current.User.ID, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	type key struct{ id, at string }
	keys := []key{}
	for rows.Next() {
		var k key
		if err = rows.Scan(&k.id, &k.at); err != nil {
			rows.Close()
			return result, err
		}
		keys = append(keys, k)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(keys) > 50 {
		keys = keys[:50]
		last := keys[49]
		result.Next = nextListCursor(scope, last.at, last.id)
	}
	for _, k := range keys {
		item, err := s.DeletionStatus(ctx, p, k.id)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// Every selector is internal SQL, never request text. Root scope removes files
// only when no other root (including historical locations) references them.
func deletionFiles(root bool) string {
	if !root {
		return `SELECT id FROM physical_files WHERE library_id=?`
	}
	return `SELECT id FROM physical_files WHERE id IN (SELECT physical_file_id FROM physical_file_locations WHERE root_id=?) AND NOT EXISTS(SELECT 1 FROM physical_file_locations other WHERE other.physical_file_id=physical_files.id AND other.root_id<>?)`
}
func scopeArgs(root bool, id string) []any {
	if root {
		return []any{id, id}
	}
	return []any{id}
}
func deletionDocs(root bool) string {
	return `SELECT id FROM documents WHERE physical_file_id IN (` + deletionFiles(root) + `)`
}
func deletionJobs(root bool) string {
	if !root {
		return `SELECT id FROM jobs WHERE library_id=?`
	}
	return `SELECT id FROM jobs WHERE physical_file_id IN (` + deletionFiles(true) + `) OR (job_type IN ('scan','verify_managed') AND target_version=?) OR (job_type='materialize' AND target_version IN (SELECT id FROM materializations WHERE target_root_id=?))`
}
func jobScopeArgs(root bool, id string) []any {
	if root {
		return []any{id, id, id, id}
	}
	return []any{id}
}
func (s *Service) deletionAccess(ctx context.Context, q storage.Querier, p domain.Principal, kind, id string) (DeletionPlan, error) {
	plan := DeletionPlan{Kind: kind, EntityID: id, Counts: map[string]int{}, Locations: []string{}}
	var err error
	if kind == "library" {
		plan.LibraryID = id
		if err = s.lifecycleAccess(ctx, q, p, id, true); err != nil {
			return plan, err
		}
		err = q.QueryRowContext(ctx, "SELECT name FROM libraries WHERE id=?", id).Scan(&plan.Name)
		plan.LibraryName = plan.Name
	} else if kind == "root" {
		if err := s.Identity.License.Check(ctx, licensing.WriteDocuments); err != nil {
			return plan, err
		}
		err = q.QueryRowContext(ctx, "SELECT r.library_id,r.canonical_path,l.name FROM storage_roots r JOIN libraries l ON l.id=r.library_id WHERE r.id=?", id).Scan(&plan.LibraryID, &plan.Name, &plan.LibraryName)
		if err == nil {
			err = s.require(ctx, q, p, plan.LibraryID, "storage.manage_roots")
		}
		if err == nil {
			err = s.require(ctx, q, p, plan.LibraryID, "documents.remove_index")
		}
	} else {
		return plan, invalid("Tipo de eliminación inválido.")
	}
	if err == sql.ErrNoRows {
		err = notFound()
	}
	return plan, err
}
func (s *Service) deletionSnapshot(ctx context.Context, q storage.Querier, plan *DeletionPlan) (string, error) {
	root := plan.Kind == "root"
	id := plan.EntityID
	args := scopeArgs(root, id)
	files := deletionFiles(root)
	docs := deletionDocs(root)
	jobs := deletionJobs(root)
	if root {
		var blocked int
		err := q.QueryRowContext(ctx, `SELECT count(*) FROM storage_roots r JOIN libraries b ON b.id=r.library_id WHERE r.id=? AND r.storage_source='managed' AND (r.status<>'retired' OR json_extract(b.settings_json,'$.default_managed_root_id')=r.id)`, id).Scan(&blocked)
		if err != nil {
			return "", err
		}
		if blocked > 0 {
			return "", domain.Failure("MANAGED_ROOT_IN_USE", "Retira primero el destino administrado y comprueba que ya no sea el destino configurado. Sus archivos se conservarán.", 409)
		}
	}
	specs := []struct {
		name, query string
		args        []any
	}{
		{"documents", "SELECT count(*) FROM documents WHERE id IN (" + docs + ")", args},
		{"jobs", "SELECT count(*) FROM jobs WHERE id IN (" + jobs + ")", jobScopeArgs(root, id)},
		{"active_jobs", "SELECT count(*) FROM jobs WHERE status IN ('queued','running','retry_wait','paused') AND id IN (" + jobs + ")", jobScopeArgs(root, id)},
		{"previews", "SELECT count(*) FROM preview_cache WHERE document_id IN (" + docs + ")", args},
		{"active_previews", "SELECT count(*) FROM preview_cache WHERE status='generating' AND document_id IN (" + docs + ")", args},
		{"affected_cases", "SELECT count(DISTINCT case_id) FROM documents WHERE id IN (" + docs + ")", args},
	}
	scope := "library_id=?"
	if root {
		scope = "root_id=?"
	}
	specs = append(specs, struct {
		name, query string
		args        []any
	}{"views", "SELECT count(*) FROM logical_folder_views WHERE " + scope, []any{id}})
	if root {
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"materializations_pending", "SELECT count(*) FROM materializations WHERE target_root_id=? AND state<>'cleaned'", []any{id}})
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"active_scans", "SELECT count(*) FROM root_scans WHERE root_id=? AND status IN ('running','paused')", []any{id}})
		plan.Counts["roots"] = 1
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"shared_references", "SELECT count(DISTINCT physical_file_id) FROM physical_file_locations WHERE root_id=? AND physical_file_id NOT IN (" + files + ")", append([]any{id}, args...)})
	} else {
		for _, table := range []string{"storage_roots", "cases", "categories", "document_types", "templates", "upload_items"} {
			specs = append(specs, struct {
				name, query string
				args        []any
			}{table, "SELECT count(*) FROM " + table + " WHERE library_id=?", []any{id}})
		}
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"receiving_uploads", "SELECT count(*) FROM upload_items WHERE library_id=? AND status='receiving'", []any{id}})
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"pending_uploads", "SELECT count(*) FROM upload_items WHERE library_id=? AND status IN ('receiving','staged','retained')", []any{id}})
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"materializations_pending", "SELECT count(*) FROM materializations WHERE library_id=? AND state<>'cleaned'", []any{id}})
		specs = append(specs, struct {
			name, query string
			args        []any
		}{"active_scans", "SELECT count(*) FROM root_scans WHERE root_id IN(SELECT id FROM storage_roots WHERE library_id=?) AND status IN ('running','paused')", []any{id}})
	}
	for _, spec := range specs {
		var count int
		if err := q.QueryRowContext(ctx, spec.query, spec.args...).Scan(&count); err != nil {
			return "", err
		}
		plan.Counts[spec.name] = count
	}
	if !root {
		plan.Counts["roots"] = plan.Counts["storage_roots"]
	}
	if plan.Counts["active_jobs"]+plan.Counts["active_previews"]+plan.Counts["receiving_uploads"]+plan.Counts["materializations_pending"]+plan.Counts["active_scans"] > 0 {
		return "", lifecycleBusy()
	}
	// The manifest includes every affected document, even withdrawn references.
	// Classified private uploads are preserved, despite being stored under uploads.
	uploads, err := s.deletionUploads(ctx, q, root, id)
	if err != nil {
		return "", err
	}
	temporaryDocuments := 0
	for _, upload := range uploads {
		if upload.Cleanable {
			plan.Counts["temporary_uploads"]++
			if upload.DocumentID != "" {
				temporaryDocuments++
			}
		} else {
			plan.Counts["preserved_private_uploads"]++
		}
	}
	plan.Counts["preserved_documents"] = plan.Counts["documents"] - temporaryDocuments
	plan.Counts["temporary_candidates"] = plan.Counts["temporary_uploads"] + plan.Counts["previews"]
	locationsQuery := "SELECT canonical_path FROM storage_roots WHERE library_id=? ORDER BY id"
	if root {
		locationsQuery = "SELECT canonical_path FROM storage_roots WHERE id=?"
	}
	rows, err := q.QueryContext(ctx, locationsQuery, id)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var path string
		if err = rows.Scan(&path); err != nil {
			rows.Close()
			return "", err
		}
		plan.Locations = append(plan.Locations, path)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return "", err
	}
	if plan.Counts["preserved_private_uploads"] > 0 {
		plan.Locations = append(plan.Locations, s.Identity.Config.PrivateUploadDirectory())
	}
	h := sha256.New()
	fmt.Fprintln(h, encode(plan.Counts), encode(plan.Locations))
	// Stream stable identities/revisions into the plan digest; no unbounded group_concat.
	queries := []struct {
		sql  string
		args []any
	}{
		{"SELECT json_array(id,revision,mode,disabled_at,settings_json,current_configuration_revision) FROM libraries WHERE id=?", []any{plan.LibraryID}},
		{"SELECT json_array(d.id,d.revision,d.deleted_at,f.current_content_version_id,f.primary_location_id) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE f.id IN (" + files + ") ORDER BY d.id", args},
		{"SELECT json_array(id,status,attempt_count,fencing_token) FROM jobs WHERE id IN (" + jobs + ") ORDER BY id", jobScopeArgs(root, id)},
		{"SELECT json_array(id,configuration_revision,status) FROM storage_roots WHERE library_id=? ORDER BY id", []any{plan.LibraryID}},
		{"SELECT json_array(id,status,private_temporary_locator,sha256) FROM upload_items WHERE library_id=? ORDER BY id", []any{plan.LibraryID}},
		{"SELECT json_array(id,root_id,relative_prefix) FROM logical_folder_views WHERE library_id=? ORDER BY id", []any{plan.LibraryID}},
		{"SELECT json_array(id,status,source_hash,generator_version) FROM preview_cache WHERE document_id IN (" + docs + ") ORDER BY id", args},
	}
	if !root {
		for _, table := range []string{"cases", "categories", "document_types"} {
			queries = append(queries, struct {
				sql  string
				args []any
			}{"SELECT json_array(id,revision) FROM " + table + " WHERE library_id=? ORDER BY id", []any{id}})
		}
	}
	for _, query := range queries {
		rows, err := q.QueryContext(ctx, query.sql, query.args...)
		if err != nil {
			return "", err
		}
		for rows.Next() {
			var value string
			if err = rows.Scan(&value); err != nil {
				rows.Close()
				return "", err
			}
			fmt.Fprintln(h, value)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
func (s *Service) PlanDeletion(ctx context.Context, p domain.Principal, kind, id string, meta domain.RequestMetadata) (DeletionPlan, error) {
	var plan DeletionPlan
	err := s.Identity.AuthorizedWrite(ctx, p, "", func(tx *sql.Tx, current domain.Principal) error {
		var err error
		plan, err = s.deletionAccess(ctx, tx, current, kind, id)
		if err != nil {
			return err
		}
		var activeID string
		err = tx.QueryRowContext(ctx, "SELECT id FROM entity_deletions WHERE kind=? AND entity_id=? AND execution_status IN ('pending','running') LIMIT 1", kind, id).Scan(&activeID)
		if err == nil {
			return domain.Failure("DELETION_ALREADY_RUNNING", "Ya hay una eliminación pendiente o en proceso para esta entidad. Consulta su estado en Eliminaciones y limpieza.", 409)
		}
		if err != sql.ErrNoRows {
			return err
		}
		digest, err := s.deletionSnapshot(ctx, tx, &plan)
		if err != nil {
			return err
		}
		plan.ID = domain.NewID()
		plan.State = "planned"
		_, err = tx.ExecContext(ctx, "INSERT INTO entity_deletions(id,kind,entity_id,library_id,actor_user_id,name,library_name,fingerprint,summary_json,state,created_at) VALUES(?,?,?,?,?,?,?,?,?,'planned',?)", plan.ID, kind, id, plan.LibraryID, current.User.ID, plan.Name, plan.LibraryName, digest, encode(plan), now())
		if err != nil {
			return err
		}
		return s.journalDeletionFiles(ctx, tx, plan, false)
	})
	return plan, err
}
func (s *Service) ConfirmDeletion(ctx context.Context, p domain.Principal, planID, name string, meta domain.RequestMetadata) (DeletionPlan, error) {
	var result DeletionPlan
	err := s.Identity.AuthorizedWrite(ctx, p, "", func(tx *sql.Tx, current domain.Principal) error {
		var raw, actor, digest, created, state, execution string
		err := tx.QueryRowContext(ctx, "SELECT summary_json,actor_user_id,fingerprint,created_at,state,execution_status FROM entity_deletions WHERE id=?", planID).Scan(&raw, &actor, &digest, &created, &state, &execution)
		if err == sql.ErrNoRows {
			return notFound()
		}
		if err != nil {
			return err
		}
		if err = json.Unmarshal([]byte(raw), &result); err != nil {
			return err
		}
		if actor != current.User.ID {
			return notFound()
		}
		if result.Kind == "library" && !current.Can("libraries.delete") {
			return domain.Failure("FORBIDDEN", "Se requiere Administrador de instalación.", 403)
		}
		if name != result.Name {
			return invalid("Escribe exactamente el nombre de la biblioteca o confirma la ruta de la carpeta.")
		}
		if execution != "unconfirmed" {
			result.State = execution
			return nil
		}
		var activeID string
		err = tx.QueryRowContext(ctx, "SELECT id FROM entity_deletions WHERE kind=? AND entity_id=? AND id<>? AND execution_status IN ('pending','running') LIMIT 1", result.Kind, result.EntityID, planID).Scan(&activeID)
		if err == nil {
			return domain.Failure("DELETION_ALREADY_RUNNING", "Ya hay una eliminación pendiente o en proceso para esta entidad. Consulta su estado en Eliminaciones y limpieza.", 409)
		}
		if err != sql.ErrNoRows {
			return err
		}
		live, err := s.deletionAccess(ctx, tx, current, result.Kind, result.EntityID)
		if err != nil {
			return err
		}
		newDigest, err := s.deletionSnapshot(ctx, tx, &live)
		if err != nil {
			return err
		}
		if newDigest != digest || created < domain.Timestamp(time.Now().Add(-15*time.Minute)) {
			return domain.Failure("DELETION_PLAN_STALE", "El contenido o la configuración cambió. Vuelve a revisar el alcance antes de eliminar.", 409)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE entity_deletions SET execution_status='pending',execution_error='' WHERE id=? AND execution_status='unconfirmed'", planID); err != nil {
			return err
		}
		result.State = "pending"
		return record(ctx, tx, current, meta, result.Kind+".deletion_queued", result.LibraryID, "", map[string]any{"operation_id": planID, "name": result.Name, "counts": result.Counts, "documents_preserved": true})
	})
	if err != nil {
		return result, err
	}
	return s.DeletionStatus(ctx, p, planID)
}

// Execute one confirmed deletion outside the HTTP request. The bulk database
// changes either commit together or roll back together. A retry uses the same
// operation ID, so an ambiguous client response cannot authorize a second
// destructive action.
func (s *Service) processPendingDeletion(ctx context.Context) error {
	var id string
	err := s.Database.Reader.QueryRowContext(ctx, "SELECT id FROM entity_deletions WHERE execution_status='pending' ORDER BY created_at,id LIMIT 1").Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if err = s.Database.Write(ctx, func(tx *sql.Tx) error {
		result, e := tx.ExecContext(ctx, "UPDATE entity_deletions SET execution_status='running' WHERE id=? AND execution_status='pending'", id)
		if e != nil {
			return e
		}
		count, e := result.RowsAffected()
		if e != nil {
			return e
		}
		if count == 0 {
			return sql.ErrNoRows
		}
		return nil
	}); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	err = s.Database.Write(ctx, func(tx *sql.Tx) error {
		var raw, digest, actor string
		if e := tx.QueryRowContext(ctx, "SELECT summary_json,fingerprint,actor_user_id FROM entity_deletions WHERE id=? AND execution_status='running' AND state='planned'", id).Scan(&raw, &digest, &actor); e != nil {
			return e
		}
		var plan DeletionPlan
		if e := json.Unmarshal([]byte(raw), &plan); e != nil {
			return e
		}
		live := DeletionPlan{Kind: plan.Kind, EntityID: plan.EntityID, LibraryID: plan.LibraryID, Counts: map[string]int{}, Locations: []string{}}
		if current, e := s.deletionSnapshot(ctx, tx, &live); e != nil {
			return e
		} else if current != digest {
			return domain.Failure("DELETION_PLAN_STALE", "El contenido cambió antes de la eliminación. Prepara un plan nuevo.", 409)
		}
		if _, e := tx.ExecContext(ctx, "DELETE FROM deletion_documents WHERE operation_id=?", id); e != nil {
			return e
		}
		if e := s.journalDeletionFiles(ctx, tx, plan, true); e != nil {
			return e
		}
		if e := s.removeEntityRecords(ctx, tx, plan); e != nil {
			return e
		}
		if plan.Kind == "root" {
			if e := configurationSnapshot(ctx, tx, plan.LibraryID, actor); e != nil {
				return e
			}
		}
		if _, e := tx.ExecContext(ctx, "UPDATE entity_deletions SET state='cleanup_pending',execution_status='completed' WHERE id=?", id); e != nil {
			return e
		}
		return audit.Append(ctx, tx, time.Now(), audit.Event{Type: plan.Kind + ".deleted", LibraryID: plan.LibraryID, ActorUserID: actor, Details: map[string]any{"operation_id": id, "documents_preserved": true, "cleanup": "pending"}})
	})
	if err != nil {
		code := failureCode(err)
		slog.Error("deletion execution failed", "operation_id", id, "code", code)
		failureCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if persistErr := s.Database.Write(failureCtx, func(tx *sql.Tx) error {
			_, e := tx.ExecContext(failureCtx, "UPDATE entity_deletions SET execution_status='failed',execution_error=? WHERE id=? AND execution_status='running'", code, id)
			return e
		}); persistErr != nil {
			slog.Error("deletion failure state could not be persisted", "operation_id", id, "code", failureCode(persistErr))
		}
		return err
	}
	cleanup, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_ = s.cleanDeletionFiles(cleanup, id, 25)
	return nil
}
func (s *Service) DeletionStatus(ctx context.Context, p domain.Principal, id string) (DeletionPlan, error) {
	var result DeletionPlan
	current, err := s.Identity.LoadPrincipal(ctx, s.Database.Reader, p.TokenDigest)
	if err != nil {
		return result, err
	}
	var raw, actor string
	var execution, executionError string
	err = s.Database.Reader.QueryRowContext(ctx, "SELECT summary_json,actor_user_id,state,execution_status,execution_error FROM entity_deletions WHERE id=?", id).Scan(&raw, &actor, &result.State, &execution, &executionError)
	if err == sql.ErrNoRows {
		return result, notFound()
	}
	if err != nil {
		return result, err
	}
	state := result.State
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		return result, err
	}
	result.State = state
	if execution != "unconfirmed" && execution != "completed" {
		result.State = execution
	}
	if executionError != "" {
		result.Diagnostics = []string{executionError}
	}
	if result.Kind == "library" {
		if !current.Can("libraries.delete") {
			return DeletionPlan{}, notFound()
		}
	} else if !current.Can("libraries.delete") {
		if actor != current.User.ID {
			return DeletionPlan{}, notFound()
		}
		if err = s.require(ctx, s.Database.Reader, current, result.LibraryID, "storage.manage_roots"); err != nil {
			return DeletionPlan{}, err
		}
	}
	err = s.Database.Reader.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(status='attention'),0) FROM deletion_cleanup WHERE operation_id=? AND status<>'complete'", id).Scan(&result.Pending, &result.Issues)
	if err != nil {
		return result, err
	}
	if result.Diagnostics == nil {
		result.Diagnostics = []string{}
	}
	rows, err := s.Database.Reader.QueryContext(ctx, "SELECT DISTINCT error_code FROM deletion_cleanup WHERE operation_id=? AND status='attention' ORDER BY error_code", id)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err = rows.Scan(&code); err != nil {
			return result, err
		}
		result.Diagnostics = append(result.Diagnostics, code)
	}
	return result, rows.Err()
}
func (s *Service) RetryDeletionCleanup(ctx context.Context, p domain.Principal, id string) (DeletionPlan, error) {
	status, err := s.DeletionStatus(ctx, p, id)
	if err != nil {
		return DeletionPlan{}, err
	}
	if status.State == "failed" {
		_, err = s.Database.Writer.ExecContext(ctx, "UPDATE entity_deletions SET execution_status='pending',execution_error='' WHERE id=? AND execution_status='failed'", id)
		if err != nil {
			return DeletionPlan{}, err
		}
		return s.DeletionStatus(ctx, p, id)
	}
	cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.cleanDeletionFiles(cleanupCtx, id, 100)
	return s.DeletionStatus(ctx, p, id)
}
func (s *Service) removeEntityRecords(ctx context.Context, tx *sql.Tx, p DeletionPlan) error {
	root := p.Kind == "root"
	id := p.EntityID
	files := deletionFiles(root)
	docs := deletionDocs(root)
	args := scopeArgs(root, id)
	// Freeze IDs in transaction-local tables before removing their source relations.
	for _, table := range []string{"delete_files", "delete_documents", "delete_jobs", "delete_roots"} {
		if _, err := tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS "+table+"(id TEXT PRIMARY KEY)"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}
	for _, entry := range []struct {
		sql  string
		args []any
	}{{"INSERT INTO delete_files " + files, args}, {"INSERT INTO delete_documents " + docs, args}, {"INSERT INTO delete_jobs " + deletionJobs(root), jobScopeArgs(root, id)}} {
		if _, err := tx.ExecContext(ctx, entry.sql, entry.args...); err != nil {
			return err
		}
	}
	scope := "library_id=?"
	if root {
		scope = "id=?"
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO delete_roots SELECT id FROM storage_roots WHERE "+scope, id); err != nil {
		return err
	}
	// Shared files remain, moving only their primary database reference to another location.
	if _, err := tx.ExecContext(ctx, `UPDATE physical_files SET availability=CASE WHEN EXISTS(SELECT 1 FROM physical_file_locations WHERE physical_file_id=physical_files.id AND root_id NOT IN (SELECT id FROM delete_roots) AND retired_at IS NULL) THEN availability ELSE 'missing' END,primary_location_id=(SELECT id FROM physical_file_locations WHERE physical_file_id=physical_files.id AND root_id NOT IN (SELECT id FROM delete_roots) ORDER BY retired_at IS NOT NULL,id LIMIT 1) WHERE id NOT IN (SELECT id FROM delete_files) AND primary_location_id IN (SELECT id FROM physical_file_locations WHERE root_id IN (SELECT id FROM delete_roots))`); err != nil {
		return err
	}
	statements := []string{
		"UPDATE jobs SET retry_of_job_id=NULL WHERE retry_of_job_id IN (SELECT id FROM delete_jobs)",
		"DELETE FROM job_controls WHERE job_id IN (SELECT id FROM delete_jobs)", "DELETE FROM job_progress WHERE job_id IN (SELECT id FROM delete_jobs)", "DELETE FROM job_attempts WHERE job_id IN (SELECT id FROM delete_jobs)", "DELETE FROM jobs WHERE id IN (SELECT id FROM delete_jobs)",
		"DELETE FROM preview_cache WHERE document_id IN (SELECT id FROM delete_documents)",
		"DELETE FROM materializations WHERE document_id IN (SELECT id FROM delete_documents) OR target_root_id IN (SELECT id FROM delete_roots)",
		"DELETE FROM review_requests WHERE document_id IN (SELECT id FROM delete_documents)",
		"DELETE FROM notification_receipts WHERE notification_id IN (SELECT id FROM document_notifications WHERE document_id IN (SELECT id FROM delete_documents))",
		"DELETE FROM document_notifications WHERE document_id IN (SELECT id FROM delete_documents)",
		"DELETE FROM document_history WHERE document_id IN (SELECT id FROM delete_documents)",
		"DELETE FROM root_retired_documents WHERE document_id IN (SELECT id FROM delete_documents) OR root_id IN (SELECT id FROM delete_roots)",
		"DELETE FROM upload_items WHERE document_id IN (SELECT id FROM delete_documents)",
		"DELETE FROM documents WHERE id IN (SELECT id FROM delete_documents)",
		"UPDATE physical_files SET primary_location_id=NULL,current_content_version_id=NULL,indexed_extraction_id=NULL WHERE id IN (SELECT id FROM delete_files)",
		"DELETE FROM indexed_pages WHERE physical_file_id IN (SELECT id FROM delete_files)",
		"DELETE FROM extraction_pages WHERE extraction_id IN (SELECT id FROM extraction_runs WHERE physical_file_id IN (SELECT id FROM delete_files))",
		"DELETE FROM extraction_runs WHERE physical_file_id IN (SELECT id FROM delete_files)",
		"DELETE FROM content_versions WHERE physical_file_id IN (SELECT id FROM delete_files)",
		"DELETE FROM scan_file_observations WHERE scan_id IN (SELECT id FROM root_scans WHERE root_id IN (SELECT id FROM delete_roots)) OR location_id IN (SELECT id FROM physical_file_locations WHERE root_id IN (SELECT id FROM delete_roots))",
		"DELETE FROM physical_file_locations WHERE root_id IN (SELECT id FROM delete_roots)",
		"DELETE FROM physical_files WHERE id IN (SELECT id FROM delete_files)",
		"DELETE FROM scan_progress WHERE scan_id IN (SELECT id FROM root_scans WHERE root_id IN (SELECT id FROM delete_roots))",
		"DELETE FROM root_scans WHERE root_id IN (SELECT id FROM delete_roots)",
		"DELETE FROM logical_folder_views WHERE root_id IN (SELECT id FROM delete_roots)",
		"DELETE FROM root_plans WHERE committed_root_id IN (SELECT id FROM delete_roots)",
	}
	for _, table := range []string{"file_scan_cache", "document_file_skips", "scan_errors", "root_scan_controls", "root_watch_recovery", "storage_root_configuration_versions"} {
		statements = append(statements, "DELETE FROM "+table+" WHERE root_id IN (SELECT id FROM delete_roots)")
	}
	statements = append(statements, "DELETE FROM storage_roots WHERE id IN (SELECT id FROM delete_roots)")
	for step, statement := range statements {
		started := time.Now()
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
		if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
			slog.Warn("slow deletion SQL step", "step", step, "duration_ms", elapsed.Milliseconds())
		}
	}
	if root {
		if _, err := tx.ExecContext(ctx, "UPDATE libraries SET revision=revision+1,current_configuration_revision=current_configuration_revision+1 WHERE id=?", p.LibraryID); err != nil {
			return err
		}
	} else {
		for _, table := range []string{"upload_items", "upload_batches", "case_requirements", "cases", "template_requirements", "template_versions", "templates", "document_types", "categories", "library_document_sequences", "root_operations", "root_plans", "library_configuration_versions", "library_role_assignments"} {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE library_id=?", id); err != nil {
				return err
			}
		}
		for _, table := range []string{"document_file_settings", "advanced_settings"} {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE scope=?", "library:"+id); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM libraries WHERE id=?", id); err != nil {
			return err
		}
	}
	// All audit_events and independent deletion evidence deliberately survive.
	return nil
}

// Manifest is JSON Lines: stream to avoid building large libraries in memory.
func (s *Service) DeletionManifest(ctx context.Context, p domain.Principal, id string, emit func(string) error) error {
	if _, err := s.DeletionStatus(ctx, p, id); err != nil {
		return err
	}
	rows, err := s.Database.Reader.QueryContext(ctx, "SELECT filename,sha256,locations_json FROM deletion_documents WHERE operation_id=? ORDER BY document_id", id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name, hash, locations string
		if err = rows.Scan(&name, &hash, &locations); err != nil {
			return err
		}
		var paths []string
		if err = json.Unmarshal([]byte(locations), &paths); err != nil {
			return err
		}
		if err = emit(encode(map[string]any{"filename": name, "sha256": hash, "locations": paths}) + "\n"); err != nil {
			return err
		}
	}
	return rows.Err()
}
