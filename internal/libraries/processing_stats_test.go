//go:build development

package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

func TestProcessingStatisticsSnapshots(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "Processing snapshots")
	empty, err := s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if !empty.Available || empty.Stats == nil || *empty.Stats != (ProcessingStats{}) || empty.UpdatedAt == "" {
		t.Fatalf("empty library is a valid zero snapshot: %+v", empty)
	}
	path := filepath.Join(dir, "root")
	copyFixture(t, filepath.Join(path, "one.pdf"), "native.pdf")
	copyFixture(t, filepath.Join(path, "copy.pdf"), "native.pdf")
	scan(t, s, addRoot(t, s, p, library, path))
	first, err := s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if !first.Available || first.Stats.Total != 2 || first.Stats.Linked != 2 || first.Stats.Pending != 2 {
		t.Fatalf("physical file counts: %+v", first)
	}
	// Latest job changes classification, without changing the registered total.
	_, err = s.Database.Writer.Exec("UPDATE jobs SET status='failed' WHERE job_type='extract' AND physical_file_id=(SELECT id FROM physical_files WHERE library_id=? LIMIT 1)", library)
	requireNoError(t, err)
	next, err := s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if next.Stats.Total != 2 || next.Stats.Errors != 1 || next.Stats.Pending != 1 {
		t.Fatalf("new complete snapshot: %+v", next)
	}
	// Record the actual PostgreSQL plan without running a corpus/load test.
	rows, err := s.Database.AnalyticsReader.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+processingStatsSQL, library, p.User.ID, p.User.ID)
	requireNoError(t, err)
	for rows.Next() {
		var line string
		requireNoError(t, rows.Scan(&line))
		if strings.Contains(line, "Execution Time") || strings.Contains(line, "SubPlan") || strings.Contains(line, "loops=") {
			t.Log(line)
		}
	}
	requireNoError(t, rows.Err())
	rows.Close()
	_, err = s.Database.Writer.Exec("DELETE FROM library_role_assignments WHERE library_id=? AND user_id=?", library, p.User.ID)
	requireNoError(t, err)
	if _, err = s.ProcessingStatistics(ctx, p, library); err == nil {
		t.Fatal("statistics bypassed library authorization")
	}
}

func TestProcessingStatisticsTimeoutDoesNotBlockActivity(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "Isolated statistics")
	path := filepath.Join(dir, "root")
	copyFixture(t, filepath.Join(path, "one.pdf"), "native.pdf")
	root := addRoot(t, s, p, library, path)
	scan(t, s, root)
	job, err := s.claimLane(ctx, "scan")
	requireNoError(t, err)
	// Hold every analytics connection, simulating contention or a long count.
	var held []*sql.Conn
	for i := 0; i < s.Database.AnalyticsReader.Stats().MaxOpenConnections; i++ {
		connection, err := s.Database.AnalyticsReader.Conn(ctx)
		requireNoError(t, err)
		held = append(held, connection)
	}
	defer func() {
		for _, connection := range held {
			connection.Close()
		}
	}()
	done := make(chan ProcessingStatsReport, 1)
	failure := make(chan error, 1)
	go func() { report, err := s.ProcessingStatistics(ctx, p, library); done <- report; failure <- err }()
	for n := 0; n < 2; n++ {
		started := time.Now()
		activity, err := s.ProcessingActivity(ctx, p, library)
		requireNoError(t, err)
		if time.Since(started) > 2*time.Second || len(activity.Scans) != 1 || len(activity.Active) != 1 || activity.Active[0].ID != job.ID {
			t.Fatalf("statistics blocked live progress: %+v", activity)
		}
		_, err = s.Database.Writer.Exec("UPDATE scan_progress SET current_path=? WHERE scan_id=?", "progress-continues.pdf", activity.Scans[0].ID)
		requireNoError(t, err)
		if n == 1 && activity.Scans[0].Current != "progress-continues.pdf" {
			t.Fatal("scan progress stopped during pending statistics")
		}
	}
	start := time.Now()
	timedOut := <-done
	requireNoError(t, <-failure)
	if time.Since(start) > 6*time.Second || timedOut.Available || timedOut.Stats != nil || timedOut.UpdatedAt != "" || timedOut.Error == "" {
		t.Fatalf("invalid timeout response: %+v", timedOut)
	}
	raw, err := json.Marshal(timedOut)
	requireNoError(t, err)
	if strings.Contains(string(raw), `"stats":`) {
		t.Fatal("timeout exposed zero/partial counts", string(raw))
	}
	for _, connection := range held {
		connection.Close()
	}
	held = nil
	recovered, err := s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if !recovered.Available || recovered.Stats.Total != 1 {
		t.Fatalf("did not recover: %+v", recovered)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.ProcessingStatistics(cancelled, domain.Principal{}, library); err == nil {
		t.Fatal("cancelled request reported success")
	}
}

func TestProcessingStatisticsContentClassifications(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "Content counters")
	allowContentFormats(t, s, p)
	path := filepath.Join(dir, "root")
	for _, name := range []string{"native.txt", "summary.txt", "pages.txt", "stored.txt"} {
		copyFixture(t, filepath.Join(path, name), "sample.txt")
	}
	scan(t, s, addRoot(t, s, p, library, path))
	var files []string
	for n := 0; n < 3; n++ {
		job, err := s.claimLane(ctx, "content")
		requireNoError(t, err)
		requireNoError(t, s.Extract(ctx, job, s.Identity.Config.Indexing))
		requireNoError(t, s.finish(ctx, job, nil))
		files = append(files, job.FileID)
	}
	// Exercise both supported OCR count sources using published TXT fixtures;
	// this tests classification without invoking OCR or changing its mechanism.
	_, err := s.Database.Writer.Exec(`UPDATE extraction_runs SET summary_json='{"ocr_pages":1}' WHERE id=(SELECT indexed_extraction_id FROM physical_files WHERE id=?)`, files[1])
	requireNoError(t, err)
	_, err = s.Database.Writer.Exec(`UPDATE extraction_runs SET summary_json='{}' WHERE id=(SELECT indexed_extraction_id FROM physical_files WHERE id=?)`, files[2])
	requireNoError(t, err)
	_, err = s.Database.Writer.Exec("UPDATE extraction_pages SET extraction_method='ocr' WHERE extraction_id=(SELECT indexed_extraction_id FROM physical_files WHERE id=?)", files[2])
	requireNoError(t, err)
	_, err = s.Database.Writer.Exec("UPDATE content_versions SET index_block_reason='policy' WHERE id IN (SELECT current_content_version_id FROM physical_files WHERE library_id=? AND extraction_freshness<>'current')", library)
	requireNoError(t, err)
	report, err := s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if !report.Available || *report.Stats != (ProcessingStats{Total: 4, Linked: 4, Processed: 3, Native: 1, OCR: 2, StoredOnly: 1}) {
		t.Fatalf("content categories: %+v", report.Stats)
	}
	_, err = s.Database.Writer.Exec("UPDATE physical_files SET extraction_freshness='stale' WHERE id=?", files[2])
	requireNoError(t, err)
	report, err = s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if report.Stats.Processed != 2 || report.Stats.OCR != 1 || report.Stats.Pending != 1 {
		t.Fatalf("stale content counted as current: %+v", report.Stats)
	}
}

func TestProcessingStatisticsPostgresWaitBudget(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "PostgreSQL wait budget")
	lock := func() *sql.Tx {
		tx, err := s.Database.Writer.BeginTx(ctx, nil)
		requireNoError(t, err)
		_, err = tx.Exec("LOCK TABLE extraction_pages IN ACCESS EXCLUSIVE MODE")
		requireNoError(t, err)
		t.Cleanup(func() { tx.Rollback() })
		return tx
	}
	// A real server-side wait longer than the former 750 ms must succeed.
	tx := lock()
	timer := time.AfterFunc(1100*time.Millisecond, func() { tx.Rollback() })
	defer timer.Stop()
	started := time.Now()
	report, err := s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if !report.Available || time.Since(started) < time.Second {
		t.Fatalf("PostgreSQL budget still too short: %+v", report)
	}
	// A longer server-side wait is cancelled, with activity still independent.
	tx = lock()
	done := make(chan ProcessingStatsReport, 1)
	failure := make(chan error, 1)
	go func() { report, err := s.ProcessingStatistics(ctx, p, library); done <- report; failure <- err }()
	activityCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err = s.ProcessingActivity(activityCtx, p, library)
	requireNoError(t, err)
	report = <-done
	requireNoError(t, <-failure)
	if report.Available || report.Stats != nil || report.Error == "" {
		t.Fatalf("blocked PostgreSQL query exposed a snapshot: %+v", report)
	}
	requireNoError(t, tx.Rollback())
	report, err = s.ProcessingStatistics(ctx, p, library)
	requireNoError(t, err)
	if !report.Available {
		t.Fatal("PostgreSQL did not recover after cancelled query")
	}
}
