//go:build development

package libraries

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/storage"
)

func TestScanCachePartialErrorsAndTargetedRetry(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Incremental")
	path := filepath.Join(directory, "files")
	for _, name := range []string{"one.pdf", "copy.pdf", "$RECYCLE.BIN/trash.pdf", "System Volume Information/private.pdf"} {
		copyFixture(t, filepath.Join(path, name), "native.pdf")
	}
	root := addRoot(t, service, principal, library, path)
	scan(t, service, root)
	first, err := service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if first.Stats.Total != 2 || first.Scans[0].Hashed != 2 || first.Scans[0].Excluded != 2 {
		t.Fatalf("bad first counters: %+v", first)
	}
	scan(t, service, root)
	second, err := service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if second.Scans[0].Unchanged != 2 || second.Scans[0].Hashed != 0 || second.Stats.Pending != 2 {
		t.Fatalf("cache not used: %+v", second)
	}
	var jobs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE job_type='extract'").Scan(&jobs))
	if jobs != 2 {
		t.Fatal("unchanged files requeued", jobs)
	}
	// One failed branch must not retire missing originals or block other discoveries.
	requireNoError(t, os.Remove(filepath.Join(path, "one.pdf")))
	requireNoError(t, os.WriteFile(filepath.Join(path, "bad.pdf"), []byte("not a PDF"), 0600))
	past := time.Now().Add(-time.Minute)
	requireNoError(t, os.Chtimes(filepath.Join(path, "bad.pdf"), past, past))
	if err = service.Scan(ctx, root, 256<<20, nil); failureCode(err) != "SCAN_PARTIAL" {
		t.Fatalf("expected partial: %v", err)
	}
	var missing int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM physical_files WHERE availability='missing'").Scan(&missing))
	if missing != 0 {
		t.Fatal("partial scan inferred absence")
	}
	report, err := service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if len(report.Errors) != 1 || report.Errors[0].Path != "bad.pdf" || report.Errors[0].Code != "INVALID_PDF" {
		t.Fatalf("missing actionable error %+v", report.Errors)
	}
	copyFixture(t, filepath.Join(path, "bad.pdf"), "native.pdf")
	// Finish the initial queued scan, then use the same bounded retry payload as the worker.
	_, err = service.Database.Writer.Exec("UPDATE jobs SET status='succeeded' WHERE job_type='scan'")
	requireNoError(t, err)
	requireNoError(t, service.RetryScanErrors(ctx, principal, root, domain.RequestMetadata{}))
	var payload string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT payload_json FROM jobs WHERE job_type='scan' AND status='queued'").Scan(&payload))
	if payload != "{\"paths\":[\"bad.pdf\"]}" {
		t.Fatal(payload)
	}
	requireNoError(t, service.Scan(context.WithValue(ctx, scanPathsKey{}, []string{"bad.pdf"}), root, 256<<20, nil))
	report, err = service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if len(report.Errors) != 0 || report.Scans[0].Files != 1 {
		t.Fatalf("retry did not limit scope %+v", report)
	}
	groups, err := service.DuplicateGroups(ctx, principal, library, "")
	requireNoError(t, err)
	if groups.Groups != 1 || groups.Files != 3 {
		t.Fatalf("wrong duplicate totals %+v", groups)
	}
	members, err := service.DuplicateDocuments(ctx, principal, library, groups.Items[0].Hash, "")
	requireNoError(t, err)
	if len(members.Items) != 3 {
		t.Fatal("missing duplicate members")
	}
}

func TestExtractionStartsWhileScanStillRunning(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("PDF tools unavailable")
	}
	service, principal, directory := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	library := addLibrary(t, service, principal, "Concurrent")
	path := filepath.Join(directory, "files")
	copyFixture(t, filepath.Join(path, "one.pdf"), "native.pdf")
	requireNoError(t, os.Mkdir(filepath.Join(path, "hold"), 0700))
	root := addRoot(t, service, principal, library, path)
	scanJob, err := service.claimLane(ctx, "scan")
	requireNoError(t, err)
	reached := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- service.Scan(ctx, root, 256<<20, func(path string) error {
			if filepath.Base(path) == "hold" {
				close(reached)
				select {
				case <-release:
				case <-ctx.Done():
				}
			}
			return nil
		})
	}()
	defer func() { cancel(); <-done }()
	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("scan never reached barrier")
	}
	job, err := service.claimLane(ctx, "content")
	requireNoError(t, err)
	if job.Kind != "extract" || job.Attempts != 1 {
		t.Fatalf("extraction starved: %+v", job)
	}
	requireNoError(t, service.Extract(ctx, job, service.Identity.Config.Indexing))
	requireNoError(t, service.finish(ctx, job, nil))
	var status string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT status FROM jobs WHERE id=?", scanJob.ID).Scan(&status))
	if status != "running" {
		t.Fatal("scan ended before content")
	}
	report, err := service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if report.Stats.Processed != 1 || report.Stats.Native != 1 {
		t.Fatalf("content not published concurrently %+v", report.Stats)
	}
	close(release)
}
func TestCancelledScanStaysPausedAndCanResume(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Cancel")
	path := filepath.Join(directory, "files")
	copyFixture(t, filepath.Join(path, "one.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	job, err := service.claimLane(ctx, "scan")
	requireNoError(t, err)
	requireNoError(t, service.CancelJob(ctx, principal, job.ID, domain.RequestMetadata{}))
	requireNoError(t, service.finish(ctx, job, scanFailure("USER_CANCELLED")))
	runtime := &Runtime{Service: service, directories: map[string]string{}, dirty: map[string]time.Time{}, degraded: map[string]string{}}
	runtime.scheduleOnce(ctx)
	_, err = service.claimLane(ctx, "scan")
	if err != sql.ErrNoRows {
		t.Fatal("cancelled scan restarted automatically", err)
	}
	requireNoError(t, service.Verify(ctx, principal, library, root, domain.RequestMetadata{}))
	_, err = service.claimLane(ctx, "scan")
	requireNoError(t, err)
}

// Opt-in corpus test: actual files, actual SQLite commits, no external 9,000-file corpus required.
func TestLargeLibrary9000(t *testing.T) {
	if os.Getenv("AIBID_LARGE_TEST") != "1" {
		t.Skip("set AIBID_LARGE_TEST=1 for the 9,000-file regression")
	}
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "9000")
	path := filepath.Join(directory, "corpus")
	requireNoError(t, os.Mkdir(path, 0700))
	past := time.Now().Add(-time.Minute)
	for index := 0; index < 9000; index++ {
		name := filepath.Join(path, fmt.Sprintf("document-%05d.pdf", index))
		requireNoError(t, os.WriteFile(name, []byte(fmt.Sprintf("%%PDF-1.7\nfixture %d", index)), 0600))
		requireNoError(t, os.Chtimes(name, past, past))
	}
	root := addRoot(t, service, principal, library, path)
	interrupted, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- service.Scan(interrupted, root, 256<<20, nil) }()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE library_id=?", library).Scan(&count))
		if count >= 200 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	err := <-done
	if err == nil {
		t.Fatal("expected interruption")
	}
	var before int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE library_id=?", library).Scan(&before))
	if before < 200 || before >= 9000 {
		t.Fatal("bad interruption point", before)
	}
	// Close and reopen SQLite; the cache/checkpoint must survive a real connection restart.
	state := service.Identity.Config.StateDirectory
	requireNoError(t, service.Database.Close())
	database, err := storage.Open(ctx, state)
	requireNoError(t, err)
	defer database.Close()
	service.Database = database
	service.Identity, err = identity.New(database, service.Identity.Config)
	requireNoError(t, err)
	requireNoError(t, service.recoverJobs(ctx))
	scan(t, service, root)
	firstDuration := time.Since(start)
	report, err := service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if report.Stats.Total != 9000 || report.Scans[0].Files != 9000 || report.Scans[0].Hashed != 9000 {
		t.Fatalf("lost/duplicated resume counters %+v", report)
	}
	start = time.Now()
	scan(t, service, root)
	incrementalDuration := time.Since(start)
	report, err = service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if report.Scans[0].Unchanged != 9000 || report.Scans[0].Hashed != 0 {
		t.Fatalf("incremental reread %+v", report.Scans)
	}
	var extractionJobs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE job_type='extract'").Scan(&extractionJobs))
	if extractionJobs != 9000 {
		t.Fatal("duplicated extraction jobs", extractionJobs)
	}
	var integrity string
	requireNoError(t, service.Database.Reader.QueryRow("PRAGMA integrity_check").Scan(&integrity))
	if integrity != "ok" {
		t.Fatal(integrity)
	}
	t.Logf("9,000 small PDF-header fixtures: first+interrupt+reopen=%s; incremental=%s; resumed after %d files; 9000 unique extraction jobs; no OCR in this load test", firstDuration, incrementalDuration, before)
}

func TestDuplicateGroupsRespectLibraryPermissions(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	first := addLibrary(t, service, principal, "First")
	second := addLibrary(t, service, principal, "Second")
	pathA := filepath.Join(directory, "first")
	pathB := filepath.Join(directory, "second")
	copyFixture(t, filepath.Join(pathA, "a.pdf"), "native.pdf")
	copyFixture(t, filepath.Join(pathB, "b.pdf"), "native.pdf")
	scan(t, service, addRoot(t, service, principal, first, pathA))
	scan(t, service, addRoot(t, service, principal, second, pathB))
	groups, err := service.DuplicateGroups(ctx, principal, first, "")
	requireNoError(t, err)
	if groups.Groups != 1 || groups.Files != 2 {
		t.Fatalf("cross-library copies not grouped %+v", groups)
	}
	hash := groups.Items[0].Hash
	_, err = service.Database.Writer.Exec("DELETE FROM library_role_assignments WHERE library_id=? AND user_id=?", second, principal.User.ID)
	requireNoError(t, err)
	groups, err = service.DuplicateGroups(ctx, principal, first, "")
	requireNoError(t, err)
	if groups.Groups != 0 || groups.Files != 0 {
		t.Fatal("unauthorized library leaked in totals")
	}
	members, err := service.DuplicateDocuments(ctx, principal, first, hash, "")
	requireNoError(t, err)
	if len(members.Items) != 1 || members.Items[0].LibraryID != first {
		t.Fatal("unauthorized duplicate exposed")
	}
}

func TestCancelledExtractionDoesNotReturnOnPeriodicScan(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Cancel content")
	path := filepath.Join(directory, "files")
	copyFixture(t, filepath.Join(path, "one.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	scan(t, service, root)
	jobs, err := service.Jobs(ctx, principal, library, "")
	requireNoError(t, err)
	var cancelledID string
	for _, job := range jobs.Items {
		if job.Kind == "extract" {
			cancelledID = job.ID
			requireNoError(t, service.CancelJob(ctx, principal, job.ID, domain.RequestMetadata{}))
		}
	}
	scan(t, service, root)
	_, err = service.claimLane(ctx, "content")
	if err != sql.ErrNoRows {
		t.Fatal("periodic scan undid cancellation", err)
	}
	_, err = service.Retry(ctx, principal, cancelledID, "Explicit resume", domain.RequestMetadata{})
	requireNoError(t, err)
	_, err = service.claimLane(ctx, "content")
	requireNoError(t, err)
}
