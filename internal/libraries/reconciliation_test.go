//go:build development

package libraries

import (
	"context"
	"fmt"
	"gestor-documental/internal/domain"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestRootRequestsCoalesceAndRecoverAfterRestart(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Coordination")
	path := filepath.Join(directory, "files")
	requireNoError(t, os.Mkdir(path, 0700))
	root := addRoot(t, service, principal, library, path)
	configuration, err := service.root(ctx, root)
	requireNoError(t, err)
	if configuration.Interval != 86400 {
		t.Fatal("new default", configuration.Interval)
	}
	var group sync.WaitGroup
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			failures <- service.Verify(ctx, principal, library, root, domain.RequestMetadata{})
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		requireNoError(t, err)
	}
	job, err := service.claimLane(ctx, "scan")
	requireNoError(t, err)
	requireNoError(t, service.Verify(ctx, principal, library, root, domain.RequestMetadata{}))
	if other, err := service.claimLane(ctx, "scan"); err == nil {
		t.Fatal("second scan claimed", other)
	}
	requireNoError(t, service.recoverJobs(ctx))
	recovered, err := service.claimLane(ctx, "scan")
	requireNoError(t, err)
	if recovered.ID != job.ID || recovered.Attempts != 2 || recovered.Fence <= job.Fence {
		t.Fatal("lost recovery fence", recovered)
	}
	var count int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE job_type='scan' AND target_version=? AND status IN ('queued','running','retry_wait','paused')", root).Scan(&count))
	if count != 1 {
		t.Fatal("duplicate root requests", count)
	}
	requireNoError(t, service.ConfigureRoot(ctx, principal, root, true, 1800, configuration.Revision, domain.RequestMetadata{}))
	saved, err := service.root(ctx, root)
	requireNoError(t, err)
	if saved.Interval != 1800 {
		t.Fatal("interval not preserved")
	}
}

func TestOverflowGenerationNeedsCompleteBarrier(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Watcher recovery")
	path := filepath.Join(directory, "files")
	copyFixture(t, filepath.Join(path, "one.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	runtime := &Runtime{Service: service, directories: map[string]string{path: root}, dirty: map[string]time.Time{}, degraded: map[string]string{}}
	runtime.eventsLost(ctx)
	once := false
	requireNoError(t, service.Scan(ctx, root, 256<<20, func(string) error {
		if !once {
			once = true
			runtime.eventsLost(ctx)
		}
		return nil
	}))
	pending, err := service.root(ctx, root)
	requireNoError(t, err)
	if pending.LastError != "WATCH_EVENTS_LOST" {
		t.Fatal("newer loss was cleared", pending.LastError)
	}
	partial := context.WithValue(ctx, scanPathsKey{}, []string{"one.pdf"})
	requireNoError(t, service.Scan(partial, root, 256<<20, nil))
	pending, err = service.root(ctx, root)
	requireNoError(t, err)
	if pending.LastError != "WATCH_EVENTS_LOST" {
		t.Fatal("partial scan acknowledged loss")
	}
	requireNoError(t, service.Scan(ctx, root, 256<<20, nil))
	pending, err = service.root(ctx, root)
	requireNoError(t, err)
	if pending.LastError != "" {
		t.Fatal("successful full scan left stale warning", pending.LastError)
	}
}

func TestIncompleteFullScanRespectsIntervalWithoutAcknowledgingLoss(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Partial cadence")
	path := filepath.Join(directory, "files")
	copyFixture(t, filepath.Join(path, "valid.pdf"), "native.pdf")
	invalid := filepath.Join(path, "invalid.pdf")
	requireNoError(t, os.WriteFile(invalid, []byte("not a PDF"), 0600))
	old := time.Now().Add(-5 * time.Second)
	requireNoError(t, os.Chtimes(invalid, old, old))
	root := addRoot(t, service, principal, library, path)
	runtime := &Runtime{Service: service, directories: map[string]string{path: root}, dirty: map[string]time.Time{}, degraded: map[string]string{}}
	runtime.eventsLost(ctx)
	if err := service.Scan(ctx, root, 256<<20, nil); failureCode(err) != "SCAN_PARTIAL" {
		t.Fatal(err)
	}
	configured, err := service.root(ctx, root)
	requireNoError(t, err)
	scanned, err := time.Parse(domain.TimeLayout, configured.LastScan)
	requireNoError(t, err)
	if time.Since(scanned) > time.Minute || configured.LastError != "WATCH_EVENTS_LOST" {
		t.Fatal("partial traversal lost cadence or warning", configured)
	}
	partial := context.WithValue(ctx, scanPathsKey{}, []string{"valid.pdf"})
	_ = service.Scan(partial, root, 256<<20, nil)
	after, err := service.root(ctx, root)
	requireNoError(t, err)
	if after.LastScan != configured.LastScan {
		t.Fatal("partial retry postponed full reconciliation")
	}
}

func TestRetirementAndIndexRemovalPreserveOriginalAndHistory(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Retirement semantics")
	path := filepath.Join(directory, "files")
	original := filepath.Join(path, "original.pdf")
	copyFixture(t, original, "native.pdf")
	root := addRoot(t, service, principal, library, path)
	drain(t, service)
	bytes, err := os.ReadFile(original)
	requireNoError(t, err)
	result := query(t, service, principal, library, "estructura")
	if result.Count != 1 {
		t.Fatal(result)
	}
	doc := result.Items[0]
	plan, err := service.RetirementPlan(ctx, principal, root)
	requireNoError(t, err)
	before, err := service.root(ctx, root)
	requireNoError(t, err)
	if before.Status != "active" {
		t.Fatal("preview retired root")
	}
	requireNoError(t, service.Retire(ctx, principal, root, plan["plan_id"].(string), "retain_index", domain.RequestMetadata{}))
	retired, err := service.root(ctx, root)
	requireNoError(t, err)
	if retired.Status != "retired" || retired.WatchMode != "paused" || query(t, service, principal, library, "estructura").Count != 1 {
		t.Fatal("retirement lost index or kept watcher", retired)
	}
	requireNoError(t, service.ConfigureRoot(ctx, principal, root, true, 86400, retired.Revision, domain.RequestMetadata{}))
	requireNoError(t, service.RemoveIndex(ctx, principal, doc.ID, "Prueba", doc.Revision, domain.RequestMetadata{}))
	requireNoError(t, service.Scan(ctx, root, 256<<20, nil))
	if query(t, service, principal, library, "estructura").Count != 0 {
		t.Fatal("rediscovery undid index removal")
	}
	current, err := os.ReadFile(original)
	requireNoError(t, err)
	if string(current) != string(bytes) {
		t.Fatal("original was changed")
	}
	var records, texts, auditCount, indexed int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE id=? AND deleted_at IS NOT NULL", doc.ID).Scan(&records))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM extraction_pages p JOIN extraction_runs e ON e.id=p.extraction_id WHERE e.physical_file_id=?", doc.FileID).Scan(&texts))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE document_id=?", doc.ID).Scan(&auditCount))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM indexed_pages WHERE physical_file_id=?", doc.FileID).Scan(&indexed))
	if records != 1 || texts == 0 || auditCount < 2 || indexed != 0 {
		t.Fatal("incorrect retained data", records, texts, auditCount, indexed)
	}
}

func TestUpgradeCoalescesLegacyRequestsAndPreservesConfiguredInterval(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Upgrade")
	path := filepath.Join(directory, "files")
	requireNoError(t, os.Mkdir(path, 0700))
	root := addRoot(t, service, principal, library, path)
	configured, err := service.root(ctx, root)
	requireNoError(t, err)
	requireNoError(t, service.ConfigureRoot(ctx, principal, root, true, 900, configured.Revision, domain.RequestMetadata{}))
	down, err := os.ReadFile("../../db/migrations/0008_reconciliation.down.sql")
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec(string(down))
	requireNoError(t, err)
	job, err := service.claimLane(ctx, "scan")
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec("INSERT INTO jobs(id,library_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,'scan',?,?,'{}','queued',?,?)", domain.NewID(), library, root, domain.NewID(), now(), now())
	requireNoError(t, err)
	up, err := os.ReadFile("../../db/migrations/0008_reconciliation.up.sql")
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec(string(up))
	requireNoError(t, err)
	var active string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT id FROM jobs WHERE job_type='scan' AND target_version=? AND status IN ('queued','running','retry_wait','paused')", root).Scan(&active))
	if active != job.ID {
		t.Fatal("upgrade discarded running checkpoint owner")
	}
	saved, err := service.root(ctx, root)
	requireNoError(t, err)
	if saved.Interval != 900 {
		t.Fatal("upgrade overwrote interval")
	}
}

func TestRootWatchRegistrationDoesNotHideIncompleteCoverage(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Watch coverage")
	path := filepath.Join(directory, "files")
	requireNoError(t, os.Mkdir(path, 0700))
	root := addRoot(t, service, principal, library, path)
	watcher, err := fsnotify.NewWatcher()
	requireNoError(t, err)
	defer watcher.Close()
	_, err = service.Database.Writer.Exec("UPDATE storage_roots SET watch_mode='polling',last_error_code='WATCH_LIMIT',last_scan_at=? WHERE id=?", now(), root)
	requireNoError(t, err)
	runtime := &Runtime{Service: service, watcher: watcher, directories: map[string]string{}, dirty: map[string]time.Time{}, degraded: map[string]string{}, watchInitialized: map[string]bool{root: true}}
	runtime.scheduleOnce(ctx)
	saved, err := service.root(ctx, root)
	requireNoError(t, err)
	if saved.WatchMode != "polling" || saved.LastError != "WATCH_LIMIT" {
		t.Fatal("registering only the root hid incomplete coverage", saved)
	}
}

// Hold the watcher coordinator to keep claimed scans alive without slow disks,
// large fixtures or test-only hooks. This exercises the production Start pool.
func TestRuntimeReconcilesDifferentRootsConcurrentlyWithPerRootExclusion(t *testing.T) {
	service, admin, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, admin, "Concurrent roots")
	allowContentFormats(t, service, admin)
	path := filepath.Join(directory, "first")
	copyFixture(t, filepath.Join(path, "first.txt"), "sample.txt")
	first := addRoot(t, service, admin, library, path)
	policy, err := service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.Paused = true
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	runtime, err := service.Start(ctx)
	requireNoError(t, err)
	defer runtime.Close()
	runtime.mutex.Lock()
	locked := true
	defer func() {
		if locked {
			runtime.mutex.Unlock()
		}
	}()
	policy, err = service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.Paused = false
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	awaitCount := func(query string, expected int) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for {
			var count int
			requireNoError(t, service.Database.Reader.QueryRow(query).Scan(&count))
			if count == expected {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("count %d, want %d: %s", count, expected, query)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	awaitCount("SELECT count(*) FROM jobs WHERE job_type='scan' AND status='running'", 1)
	for i := 0; i < 4; i++ {
		folder := filepath.Join(directory, fmt.Sprintf("added-%d", i))
		copyFixture(t, filepath.Join(folder, "added.txt"), "sample.txt")
		addRoot(t, service, admin, library, folder)
		requireNoError(t, service.Verify(ctx, admin, library, first, domain.RequestMetadata{}))
	}
	awaitCount("SELECT count(*) FROM jobs WHERE job_type='scan' AND status='running'", 4)
	awaitCount("SELECT count(*) FROM jobs WHERE job_type='scan' AND status='queued'", 1)
	var firstJobs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE job_type='scan' AND target_version=?", first).Scan(&firstJobs))
	if firstJobs != 1 {
		t.Fatal("same root scheduled twice", firstJobs)
	}
	runtime.mutex.Unlock()
	locked = false
	awaitCount("SELECT count(DISTINCT root_id) FROM root_scans WHERE status='complete'", 5)
	awaitCount("SELECT count(*) FROM physical_files WHERE extraction_freshness='current'", 5)
}
