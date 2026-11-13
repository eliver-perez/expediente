// Temporary local maintenance helper for the disposable Windows benchmark installation.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) != 4 || (os.Args[1] != "create" && os.Args[1] != "disable" && os.Args[1] != "target" && os.Args[1] != "migrations" && os.Args[1] != "stats" && os.Args[1] != "eula" && os.Args[1] != "ocrcheck" && os.Args[1] != "jobbench" && os.Args[1] != "folderbench" && os.Args[1] != "deletiondiag" && os.Args[1] != "deletionstate" && os.Args[1] != "deletionsample" && os.Args[1] != "deletionplanquery" && os.Args[1] != "deletionverify") {
		panic("usage: aidex-benchmark-user create|disable|target|migrations|stats|eula|ocrcheck|jobbench|folderbench|deletiondiag DB_PATH ARG")
	}
	timeout := 45 * time.Second
	if os.Args[1] == "deletionverify" {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	path, err := filepath.Abs(os.Args[2])
	if err != nil {
		panic(err)
	}
	path = filepath.ToSlash(path)
	if len(path) > 2 && path[1] == ':' {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path}
	u.RawQuery = "_pragma=busy_timeout(30000)&_pragma=foreign_keys(1)&_txlock=immediate"
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	username := strings.ToLower(strings.TrimSpace(os.Args[3]))
	if os.Args[1] == "deletionverify" {
		var library, state, execution, executionError string
		if err := db.QueryRowContext(ctx, "SELECT library_id,state,execution_status,execution_error FROM entity_deletions WHERE id=?", os.Args[3]).Scan(&library, &state, &execution, &executionError); err != nil {
			panic(err)
		}
		counts := map[string]int{}
		for key, query := range map[string]string{
			"library":          "SELECT count(*) FROM libraries WHERE id=?",
			"documents":        "SELECT count(*) FROM documents WHERE library_id=?",
			"physical_files":   "SELECT count(*) FROM physical_files WHERE library_id=?",
			"roots":            "SELECT count(*) FROM storage_roots WHERE library_id=?",
			"jobs":             "SELECT count(*) FROM jobs WHERE library_id=?",
			"cases":            "SELECT count(*) FROM cases WHERE library_id=?",
			"versions":         "SELECT count(*) FROM content_versions WHERE physical_file_id IN (SELECT id FROM physical_files WHERE library_id=?)",
			"extractions":      "SELECT count(*) FROM extraction_runs WHERE physical_file_id IN (SELECT id FROM physical_files WHERE library_id=?)",
			"indexed_pages":    "SELECT count(*) FROM indexed_pages WHERE physical_file_id IN (SELECT id FROM physical_files WHERE library_id=?)",
			"extraction_pages": "SELECT count(*) FROM extraction_pages WHERE extraction_id IN (SELECT id FROM extraction_runs WHERE physical_file_id IN (SELECT id FROM physical_files WHERE library_id=?))",
			"manifest":         "SELECT count(*) FROM deletion_documents WHERE operation_id=?",
			"cleanup_pending":  "SELECT count(*) FROM deletion_cleanup WHERE operation_id=? AND status<>'complete'",
		} {
			arg := library
			if key == "manifest" || key == "cleanup_pending" {
				arg = os.Args[3]
			}
			var n int
			if err := db.QueryRowContext(ctx, query, arg).Scan(&n); err != nil {
				panic(err)
			}
			counts[key] = n
		}
		var ftsCount, indexedCount int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM pages_fts").Scan(&ftsCount); err != nil {
			panic(err)
		}
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM indexed_pages").Scan(&indexedCount); err != nil {
			panic(err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO pages_fts(pages_fts,rank) VALUES('integrity-check',1)"); err != nil {
			panic(err)
		}
		var fkIssues int
		rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
		if err != nil {
			panic(err)
		}
		for rows.Next() {
			var table, parent string
			var rowID, foreignKeyID int64
			if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
				panic(err)
			}
			fkIssues++
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		rows.Close()
		result := map[string]any{"state": state, "execution": execution, "execution_error": executionError, "counts": counts, "fts_rows": ftsCount, "indexed_rows": indexedCount, "fts_integrity": "ok", "foreign_key_issues": fkIssues}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "deletionplanquery" {
		if _, err := db.ExecContext(ctx, "CREATE TEMP TABLE delete_extractions(id TEXT PRIMARY KEY)"); err != nil {
			panic(err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO delete_extractions SELECT id FROM extraction_runs WHERE physical_file_id IN (SELECT id FROM physical_files WHERE library_id=?)", os.Args[3]); err != nil {
			panic(err)
		}
		rows, err := db.QueryContext(ctx, "EXPLAIN QUERY PLAN DELETE FROM extraction_pages WHERE extraction_id IN (SELECT id FROM delete_extractions)")
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		plan := []string{}
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				panic(err)
			}
			plan = append(plan, detail)
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(plan); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "deletionstate" {
		library := os.Args[3]
		result := map[string]int{}
		for key, query := range map[string]string{
			"documents":       "SELECT count(*) FROM documents WHERE library_id=?",
			"jobs_active":     "SELECT count(*) FROM jobs WHERE library_id=? AND status IN ('queued','running','retry_wait','paused')",
			"scans_active":    "SELECT count(*) FROM root_scans WHERE root_id IN (SELECT id FROM storage_roots WHERE library_id=?) AND status IN ('running','paused')",
			"previews_active": "SELECT count(*) FROM preview_cache WHERE document_id IN (SELECT id FROM documents WHERE library_id=?) AND status='generating'",
		} {
			var n int
			if err := db.QueryRowContext(ctx, query, library).Scan(&n); err != nil {
				panic(err)
			}
			result[key] = n
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "deletionsample" {
		rows, err := db.QueryContext(ctx, `SELECT filename,sha256,locations_json FROM deletion_documents WHERE operation_id=? AND length(sha256)=64 ORDER BY document_id LIMIT 20`, os.Args[3])
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		result := []map[string]string{}
		for rows.Next() {
			var filename, hash, locations string
			if err := rows.Scan(&filename, &hash, &locations); err != nil {
				panic(err)
			}
			result = append(result, map[string]string{"filename": filename, "sha256": hash, "locations_json": locations})
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "deletiondiag" {
		rows, err := db.QueryContext(ctx, `SELECT e.id,e.kind,e.entity_id,e.library_id,e.name,e.state,e.execution_status,e.execution_error,e.created_at,coalesce(json_extract(e.summary_json,'$.counts.documents'),0),coalesce((SELECT count(*) FROM deletion_documents dd WHERE dd.operation_id=e.id),0) FROM entity_deletions e ORDER BY e.created_at DESC LIMIT 12`)
		if err != nil {
			panic(err)
		}
		type deletion struct {
			ID                string `json:"id"`
			Kind              string `json:"kind"`
			EntityID          string `json:"entity_id"`
			LibraryID         string `json:"library_id"`
			Name              string `json:"name"`
			State             string `json:"state"`
			Execution         string `json:"execution"`
			Error             string `json:"error"`
			Created           string `json:"created"`
			PlannedDocuments  int    `json:"planned_documents"`
			ManifestDocuments int    `json:"manifest_documents"`
		}
		items := []deletion{}
		for rows.Next() {
			var v deletion
			if err := rows.Scan(&v.ID, &v.Kind, &v.EntityID, &v.LibraryID, &v.Name, &v.State, &v.Execution, &v.Error, &v.Created, &v.PlannedDocuments, &v.ManifestDocuments); err != nil {
				panic(err)
			}
			items = append(items, v)
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		rows.Close()
		if err := json.NewEncoder(os.Stdout).Encode(items); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "folderbench" {
		rootID := os.Args[3]
		var libraryID string
		if err := db.QueryRowContext(ctx, "SELECT library_id FROM storage_roots WHERE id=?", rootID).Scan(&libraryID); err != nil {
			panic(err)
		}
		oldQuery := `SELECT DISTINCT substr(l.relative_path,1,instr(substr(l.relative_path,1),'/')-1) AS folder FROM physical_file_locations l JOIN physical_files f ON f.primary_location_id=l.id JOIN documents d ON d.physical_file_id=f.id WHERE d.library_id=? AND d.deleted_at IS NULL AND l.root_id=? AND instr(substr(l.relative_path,1),'/')>0 AND substr(l.relative_path,1,instr(substr(l.relative_path,1),'/')-1)>'' ORDER BY folder LIMIT 51`
		newQuery := `WITH folders AS MATERIALIZED (SELECT DISTINCT substr(relative_path,1,instr(relative_path,'/')-1) AS folder FROM physical_file_locations INDEXED BY physical_locations_explorer WHERE root_id=? AND instr(relative_path,'/')>0) SELECT folder FROM folders WHERE folder>'' AND EXISTS(SELECT 1 FROM physical_file_locations l INDEXED BY physical_locations_explorer CROSS JOIN physical_files f CROSS JOIN documents d WHERE l.root_id=? AND l.relative_path>=folder||'/' AND l.relative_path<folder||'0' AND f.primary_location_id=l.id AND d.physical_file_id=f.id AND d.library_id=? AND d.deleted_at IS NULL) ORDER BY folder LIMIT 51`
		read := func(query string, args ...any) ([]string, int64) {
			start := time.Now()
			rows, err := db.QueryContext(ctx, query, args...)
			if err != nil {
				panic(err)
			}
			defer rows.Close()
			names := []string{}
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					panic(err)
				}
				names = append(names, name)
			}
			if err := rows.Err(); err != nil {
				panic(err)
			}
			return names, time.Since(start).Milliseconds()
		}
		before, beforeMS := read(oldQuery, libraryID, rootID)
		after, afterMS := read(newQuery, rootID, rootID, libraryID)
		result := map[string]any{"before_ms": beforeMS, "after_ms": afterMS, "before_count": len(before), "after_count": len(after), "same": strings.Join(before, "\x00") == strings.Join(after, "\x00")}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "jobbench" {
		queries := map[string]string{
			"correlated": `SELECT count(*),coalesce(sum(j.status='failed'),0) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id LEFT JOIN jobs j ON j.id=(SELECT id FROM jobs WHERE physical_file_id=f.id AND target_version=f.current_content_version_id AND job_type='extract' ORDER BY created_at DESC,id DESC LIMIT 1) WHERE d.deleted_at IS NULL`,
			"window":     `WITH latest AS MATERIALIZED (SELECT physical_file_id,target_version,status FROM (SELECT physical_file_id,target_version,status,row_number() OVER (PARTITION BY physical_file_id,target_version ORDER BY created_at DESC,id DESC) rn FROM jobs WHERE job_type='extract') WHERE rn=1) SELECT count(*),coalesce(sum(latest.status='failed'),0) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id LEFT JOIN latest ON latest.physical_file_id=f.id AND latest.target_version=f.current_content_version_id WHERE d.deleted_at IS NULL`,
		}
		results := map[string]any{}
		for index, name := range []string{"correlated", "correlated"} {
			start := time.Now()
			var total, failed int64
			if err := db.QueryRowContext(ctx, queries[name]).Scan(&total, &failed); err != nil {
				panic(err)
			}
			results[fmt.Sprintf("%d_%s", index, name)] = map[string]any{"ms": time.Since(start).Milliseconds(), "total": total, "failed": failed}
		}
		if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "ocrcheck" {
		var result struct {
			Indexed  int64 `json:"indexed"`
			Missing  int64 `json:"missing_summary"`
			Mismatch int64 `json:"mismatch"`
		}
		err := db.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(json_type(e.summary_json,'$.ocr_pages') IS NULL),0),coalesce(sum((coalesce(json_extract(e.summary_json,'$.ocr_pages'),0)>0) <> EXISTS(SELECT 1 FROM extraction_pages ep WHERE ep.extraction_id=e.id AND ep.extraction_method IN ('ocr','empty'))),0) FROM physical_files f JOIN extraction_runs e ON e.id=f.indexed_extraction_id WHERE f.extraction_freshness='current'`).Scan(&result.Indexed, &result.Missing, &result.Mismatch)
		if err != nil {
			panic(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "eula" {
		var result struct {
			Version string `json:"version"`
			Hash    string `json:"hash"`
		}
		if err := db.QueryRowContext(ctx, "SELECT version,text_sha256 FROM eula_acceptances ORDER BY accepted_at DESC LIMIT 1").Scan(&result.Version, &result.Hash); err != nil {
			panic(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "stats" {
		counts := map[string]int64{}
		for _, table := range []string{"documents", "physical_files", "extraction_pages", "jobs", "job_attempts", "extraction_runs", "scan_errors", "document_file_skips"} {
			var count int64
			if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
				panic(err)
			}
			counts[table] = count
		}
		rows, err := db.QueryContext(ctx, "SELECT status,count(*) FROM jobs GROUP BY status")
		if err != nil {
			panic(err)
		}
		for rows.Next() {
			var status string
			var count int64
			if err := rows.Scan(&status, &count); err != nil {
				panic(err)
			}
			counts["jobs_"+status] = count
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		rows.Close()
		if err := json.NewEncoder(os.Stdout).Encode(counts); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "migrations" {
		rows, err := db.QueryContext(ctx, "SELECT name,checksum FROM schema_migrations ORDER BY name")
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		entries := []struct {
			Name     string `json:"name"`
			Checksum string `json:"checksum"`
		}{}
		for rows.Next() {
			var entry struct {
				Name     string `json:"name"`
				Checksum string `json:"checksum"`
			}
			if err := rows.Scan(&entry.Name, &entry.Checksum); err != nil {
				panic(err)
			}
			entries = append(entries, entry)
		}
		if err := rows.Err(); err != nil {
			panic(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(entries); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "target" {
		var target struct {
			LibraryID string `json:"library_id"`
			RootID    string `json:"root_id"`
			Documents int    `json:"documents"`
		}
		err = db.QueryRowContext(ctx, "SELECT l.id,count(d.id) FROM libraries l LEFT JOIN documents d ON d.library_id=l.id AND d.deleted_at IS NULL WHERE l.disabled_at IS NULL GROUP BY l.id ORDER BY count(d.id) DESC LIMIT 1").Scan(&target.LibraryID, &target.Documents)
		if err != nil {
			panic(err)
		}
		err = db.QueryRowContext(ctx, "SELECT id FROM storage_roots WHERE library_id=? AND status='active' ORDER BY created_at LIMIT 1", target.LibraryID).Scan(&target.RootID)
		if err != nil {
			panic(err)
		}
		if err = json.NewEncoder(os.Stdout).Encode(target); err != nil {
			panic(err)
		}
		return
	}
	if os.Args[1] == "disable" {
		_, err = db.ExecContext(ctx, "UPDATE users SET disabled_at=?,updated_at=? WHERE username_key=?", domain.Timestamp(time.Now()), domain.Timestamp(time.Now()), username)
		if err != nil {
			panic(err)
		}
		fmt.Println("benchmark user disabled")
		return
	}
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		panic(err)
	}
	password = strings.TrimRight(password, "\r\n")
	hasher, err := identity.NewPasswordHasher()
	if err != nil {
		panic(err)
	}
	hash, err := hasher.Hash(ctx, password)
	if err != nil {
		panic(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()
	id, now := domain.NewID(), domain.Timestamp(time.Now())
	_, err = tx.ExecContext(ctx, "INSERT INTO users(id,username,username_key,display_name,password_hash,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", id, username, username, "Medición local de rendimiento", hash, now, now)
	if err != nil {
		panic(err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO global_role_assignments(user_id,role_id) VALUES (?, 'installation_admin')", id)
	if err != nil {
		panic(err)
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO library_role_assignments(user_id,library_id,role_id) SELECT ?,id,'library_manager' FROM libraries WHERE disabled_at IS NULL", id)
	if err != nil {
		panic(err)
	}
	if err = tx.Commit(); err != nil {
		panic(err)
	}
	fmt.Println("benchmark user created")
}
