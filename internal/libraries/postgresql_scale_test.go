//go:build development

package libraries

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

// This corpus has 20,000 documents/jobs/extractions and 200,000 indexed pages.
// All disk originals are independent markers and remain unchanged after deletion.
func TestPostgresLargeIndexConcurrentDeletion(t *testing.T) {
	if os.Getenv("AIDEX_PERF_POSTGRES") != "1" {
		t.Skip("set AIDEX_PERF_POSTGRES=1")
	}
	s, p, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "Corpus PostgreSQL")
	path := filepath.Join(directory, "originals")
	requireNoError(t, os.MkdirAll(path, 0700))
	marker := filepath.Join(path, "original.txt")
	requireNoError(t, os.WriteFile(marker, []byte("original inmutable"), 0600))
	root := addRoot(t, s, p, library, path)
	_, err := s.Database.Writer.Exec("UPDATE jobs SET status='succeeded' WHERE target_version=?", root)
	requireNoError(t, err)
	started := time.Now()
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error {
		statements := []struct {
			query string
			args  []any
		}{
			{`INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,created_at) SELECT 'pg-file-'||n,?,'linked','available','verified',? FROM generate_series(1,20000)n`, []any{library, now()}},
			{`INSERT INTO physical_file_locations(id,physical_file_id,library_id,storage_source,root_id,relative_path,canonical_path,comparison_key,root_configuration_revision,first_seen_at,last_seen_at) SELECT 'pg-location-'||n,'pg-file-'||n,?,'linked',?,'documento-'||n||'.txt',?||'/documento-'||n||'.txt',?||'/documento-'||n||'.txt',1,?,? FROM generate_series(1,20000)n`, []any{library, root, path, path, now(), now()}},
			{`INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,observed_change_key,observed_at,change_origin) SELECT 'pg-version-'||n,'pg-file-'||n,1,42,?,'version-'||n,?,'scan' FROM generate_series(1,20000)n`, []any{now(), now()}},
			{`INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count,unit_count,summary_json) SELECT 'pg-extraction-'||n,'pg-file-'||n,'pg-version-'||n,'fixture','spa','complete',10,10,'{"ocr_pages":5}' FROM generate_series(1,20000)n`, nil},
			{`INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) SELECT '00000000-0000-4000-8000-'||lpad(n::text,12,'0'),?,'pg-file-'||n,'Documento-'||n||'.txt','documento-'||n||'.txt',? FROM generate_series(1,20000)n`, []any{library, now()}},
			{`INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text) SELECT 'pg-extraction-'||n,page,CASE WHEN page<=5 THEN 'ocr' ELSE 'native' END,'Evidencia metálica documento '||n||' página '||page FROM generate_series(1,20000)n CROSS JOIN generate_series(1,10)page`, nil},
			{`INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) SELECT 'pg-file-'||n,'pg-extraction-'||n,page,'Evidencia metálica documento '||n||' página '||page FROM generate_series(1,20000)n CROSS JOIN generate_series(1,10)page`, nil},
			{`INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) SELECT 'pg-job-'||n,?,'pg-file-'||n,'extract','pg-version-'||n,'pg-job-'||n,'{}','succeeded',?,? FROM generate_series(1,20000)n`, []any{library, now(), now()}},
			{`UPDATE physical_files SET primary_location_id='pg-location-'||substring(id from 9),current_content_version_id='pg-version-'||substring(id from 9),indexed_extraction_id='pg-extraction-'||substring(id from 9),extraction_freshness='current' WHERE library_id=?`, []any{library}},
		}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
				return err
			}
		}
		return nil
	}))
	t.Logf("corpus: documents=20000 jobs=20000 extractions=20000 pages=200000 insert=%s", time.Since(started))
	_, err = s.Database.Writer.Exec("ANALYZE")
	requireNoError(t, err)
	result, err := s.Search(ctx, p, SearchInput{Query: `"evidencia metalica"`, Type: "ocr", Libraries: []string{library}, Limit: 10}, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if result.Count != 20000 || len(result.Items) != 10 {
		t.Fatal("large index search changed", result.Count, len(result.Items))
	}
	// Keep an operational library available throughout metadata and index deletion.
	other := addLibrary(t, s, p, "Biblioteca en uso")
	copyFixture(t, filepath.Join(directory, "other", "native.pdf"), "native.pdf")
	otherRoot := addRoot(t, s, p, other, filepath.Join(directory, "other"))
	scan(t, s, otherRoot)
	drain(t, s)
	plan, err := s.PlanDeletion(ctx, p, "library", library, domain.RequestMetadata{})
	requireNoError(t, err)
	_, err = s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{})
	requireNoError(t, err)
	done := make(chan error, 1)
	go func() { done <- s.processPendingDeletion(ctx) }()
	probes := 0
	maxLatency := time.Duration(0)
	started = time.Now()
	for {
		select {
		case err = <-done:
			requireNoError(t, err)
			goto finished
		default:
		}
		begin := time.Now()
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = s.Search(probeCtx, p, SearchInput{Libraries: []string{other}, Limit: 1}, domain.RequestMetadata{}, true)
		cancel()
		requireNoError(t, err)
		if elapsed := time.Since(begin); elapsed > maxLatency {
			maxLatency = elapsed
		}
		probes++
		time.Sleep(10 * time.Millisecond)
	}
finished:
	if probes == 0 {
		t.Fatal("no concurrent search/audit probe executed")
	}
	for _, table := range []string{"indexed_pages", "extraction_pages"} {
		var count int
		requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM "+table+" WHERE "+map[string]string{"indexed_pages": "physical_file_id", "extraction_pages": "extraction_id"}[table]+" LIKE 'pg-%'").Scan(&count))
		if count != 0 {
			t.Fatal("large deletion left pages", table, count)
		}
	}
	contents, err := os.ReadFile(marker)
	requireNoError(t, err)
	if string(contents) != "original inmutable" {
		t.Fatal("original changed")
	}
	t.Logf("delete=%s concurrent_searches_with_audit=%d max_latency=%s", time.Since(started), probes, maxLatency)
}
