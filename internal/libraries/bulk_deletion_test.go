//go:build development

package libraries

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
)

func TestBulkRootDeletionKeepsOriginals(t *testing.T) {
	s, principal, directory := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, s, principal, "hybrid")
	path := filepath.Join(directory, "originales")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(path, "original.txt")
	if err := os.WriteFile(marker, []byte("conservar original"), 0600); err != nil {
		t.Fatal(err)
	}
	root := addRoot(t, s, principal, library, path)
	var source string
	var revision int
	if err := s.Database.Reader.QueryRowContext(ctx, "SELECT storage_source,configuration_revision FROM storage_roots WHERE id=?", root).Scan(&source, &revision); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Database.Writer.ExecContext(ctx, "UPDATE jobs SET status='succeeded' WHERE job_type IN ('scan','verify_managed') AND target_version=?", root); err != nil {
		t.Fatal(err)
	}
	const count = 10000
	if err := s.Database.Write(ctx, func(tx *sql.Tx) error {
		file, err := tx.PrepareContext(ctx, "INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,created_at) VALUES(?,?,?,'available','verified',?)")
		if err != nil {
			return err
		}
		defer file.Close()
		location, err := tx.PrepareContext(ctx, "INSERT INTO physical_file_locations(id,physical_file_id,library_id,storage_source,root_id,relative_path,canonical_path,comparison_key,root_configuration_revision,first_seen_at,last_seen_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)")
		if err != nil {
			return err
		}
		defer location.Close()
		document, err := tx.PrepareContext(ctx, "INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES(?,?,?,?,?,?)")
		if err != nil {
			return err
		}
		defer document.Close()
		job, err := tx.PrepareContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','succeeded',?,?)")
		if err != nil {
			return err
		}
		defer job.Close()
		version, err := tx.PrepareContext(ctx, "INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,observed_change_key,observed_at,change_origin) VALUES(?,?,1,1,?,?,?,'scan')")
		if err != nil {
			return err
		}
		defer version.Close()
		extraction, err := tx.PrepareContext(ctx, "INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count,completed_at) VALUES(?,?,?,'v1','es','complete',1,?)")
		if err != nil {
			return err
		}
		defer extraction.Close()
		extractionPage, err := tx.PrepareContext(ctx, "INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text) VALUES(?,1,'native',?)")
		if err != nil {
			return err
		}
		defer extractionPage.Close()
		indexedPage, err := tx.PrepareContext(ctx, "INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) VALUES(?,?,1,?)")
		if err != nil {
			return err
		}
		defer indexedPage.Close()
		for i := 0; i < count; i++ {
			id := fmt.Sprintf("bulk-%05d", i)
			name := id + ".pdf"
			full := filepath.Join(path, name)
			if _, err = file.ExecContext(ctx, id, library, source, now()); err != nil {
				return err
			}
			if _, err = location.ExecContext(ctx, id+"-location", id, library, source, root, name, full, full, revision, now(), now()); err != nil {
				return err
			}
			if _, err = document.ExecContext(ctx, id+"-document", library, id, name, name, now()); err != nil {
				return err
			}
			if i%10 == 0 {
				if _, err = job.ExecContext(ctx, id+"-job", library, id, "version-"+id, id+"-job", now(), now()); err != nil {
					return err
				}
			}
			if i%5 == 0 {
				versionID, extractionID := id+"-version", id+"-extraction"
				if _, err = version.ExecContext(ctx, versionID, id, now(), versionID, now()); err != nil {
					return err
				}
				if _, err = extraction.ExecContext(ctx, extractionID, id, versionID, now()); err != nil {
					return err
				}
				if _, err = extractionPage.ExecContext(ctx, extractionID, "página de prueba"); err != nil {
					return err
				}
				if _, err = indexedPage.ExecContext(ctx, id, extractionID, "página de prueba"); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	plan, err := s.PlanDeletion(ctx, principal, "root", root, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Counts["documents"] != count {
		t.Fatalf("planned %d documents, want %d", plan.Counts["documents"], count)
	}
	result := completeDeletionForTest(t, s, principal, plan)
	if result.State != "complete" {
		t.Fatalf("root deletion did not complete: %+v", result)
	}
	for _, query := range []string{"SELECT count(*) FROM documents WHERE library_id=?", "SELECT count(*) FROM physical_files WHERE library_id=?", "SELECT count(*) FROM storage_roots WHERE library_id=?", "SELECT count(*) FROM indexed_pages", "SELECT count(*) FROM extraction_pages", "SELECT count(*) FROM indexed_pages WHERE search_vector IS NOT NULL"} {
		var n int
		args := []any{}
		if strings.Contains(query, "library_id=?") {
			args = append(args, library)
		}
		if err := s.Database.Reader.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("records remained for %s: %d", query, n)
		}
	}
	contents, err := os.ReadFile(marker)
	if err != nil || string(contents) != "conservar original" {
		t.Fatalf("original changed: %q, %v", contents, err)
	}
}
