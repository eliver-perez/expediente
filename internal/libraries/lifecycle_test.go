//go:build development

package libraries

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"gestor-documental/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func completeDeletionForTest(t *testing.T, s *Service, p domain.Principal, plan DeletionPlan) DeletionPlan {
	t.Helper()
	result, err := s.ConfirmDeletion(context.Background(), p, plan.ID, plan.Name, domain.RequestMetadata{})
	requireNoError(t, err)
	if result.State != "pending" {
		t.Fatalf("deletion was not queued: %+v", result)
	}
	requireNoError(t, s.processPendingDeletion(context.Background()))
	result, err = s.DeletionStatus(context.Background(), p, plan.ID)
	requireNoError(t, err)
	return result
}

func TestLifecycleLibraryIsolationAndArchive(t *testing.T) {
	s, p, directory := fixture(t)
	ctx := context.Background()
	a := managedLibrary(t, s, p, "hybrid")
	b := managedLibrary(t, s, p, "hybrid")
	c := managedLibrary(t, s, p, "hybrid")
	uploads := map[string]UploadItem{}
	locators := map[string]string{}
	for _, lib := range []string{a, b, c} {
		uploads[lib] = uploadFixture(t, s, p, lib)
		var locator string
		requireNoError(t, s.Database.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE document_id=?", uploads[lib].DocumentID).Scan(&locator))
		locators[lib] = filepath.Join(s.Identity.Config.PrivateUploadDirectory(), locator)
		path := filepath.Join(directory, lib, "original.pdf")
		copyFixture(t, path, "native.pdf")
		root := addRoot(t, s, p, lib, filepath.Dir(path))
		scan(t, s, root)
	}
	// Classified staged uploads are the only copy: deletion must preserve them.
	classified := uploadFixture(t, s, p, b)
	category, kind := organizationCatalog(t, s, p, b, true, false)
	doc, err := s.Document(ctx, p, classified.DocumentID)
	requireNoError(t, err)
	requireNoError(t, s.Classify(ctx, p, doc.ID, "classification", Classification{CategoryID: category, TypeID: kind}, doc.Revision, domain.RequestMetadata{}))
	var kept string
	requireNoError(t, s.Database.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE document_id=?", classified.DocumentID).Scan(&kept))
	// Archive preserves all rows/files and blocks operational writes/worker claims.
	var revision int64
	requireNoError(t, s.Database.Reader.QueryRow("SELECT revision FROM libraries WHERE id=?", b).Scan(&revision))
	requireNoError(t, s.ArchiveLibrary(ctx, p, b, revision, false, domain.RequestMetadata{}))
	active, err := s.Libraries(ctx, p)
	requireNoError(t, err)
	if len(active) != 2 {
		t.Fatal("archive visible among active")
	}
	archived, err := s.ListLibraries(ctx, p, true)
	requireNoError(t, err)
	if len(archived) != 1 || !archived[0].Archived {
		t.Fatal(archived)
	}
	if _, err = s.CreateBatch(ctx, p, b, domain.NewID(), domain.RequestMetadata{}); err == nil {
		t.Fatal("archived upload allowed")
	}
	if _, err = os.Stat(locators[b]); err != nil {
		t.Fatal("archive cleaned uploads")
	}
	requireNoError(t, s.ArchiveLibrary(ctx, p, b, revision+1, true, domain.RequestMetadata{}))
	drain(t, s)
	plan, err := s.PlanDeletion(ctx, p, "library", b, domain.RequestMetadata{})
	requireNoError(t, err)
	if _, err = s.ConfirmDeletion(ctx, p, plan.ID, "incorrect", domain.RequestMetadata{}); err == nil {
		t.Fatal("name not enforced")
	}
	manager := member(t, s, p, a, "manager-only", "library_manager")
	if _, err = s.PlanDeletion(ctx, manager, "library", a, domain.RequestMetadata{}); failureCode(err) != "FORBIDDEN" {
		t.Fatal("manager permitted", err)
	}
	result := completeDeletionForTest(t, s, p, plan)
	if result.State != "complete" {
		t.Fatalf("cleanup incomplete: %+v", result)
	}
	for _, lib := range []string{a, b, c} {
		if _, err = os.Stat(filepath.Join(directory, lib, "original.pdf")); err != nil {
			t.Fatal("original removed", lib, err)
		}
		_, err = os.Stat(locators[lib])
		if lib == b {
			if !os.IsNotExist(err) {
				t.Fatal("exclusive temporary not removed")
			}
		} else if err != nil {
			t.Fatal("foreign temporary removed", lib, err)
		}
	}
	if _, err = os.Stat(filepath.Join(s.Identity.Config.PrivateUploadDirectory(), kept)); err != nil {
		t.Fatal("classified original removed", err)
	}
	var foreign int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM libraries WHERE id IN (?,?)", a, c).Scan(&foreign))
	if foreign != 2 {
		t.Fatal("foreign libraries damaged")
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM pragma_foreign_key_check").Scan(&foreign))
	if foreign != 0 {
		t.Fatal("dangling FKs")
	}
	var integrity string
	requireNoError(t, s.Database.Reader.QueryRow("PRAGMA integrity_check").Scan(&integrity))
	if integrity != "ok" {
		t.Fatal(integrity)
	}
	_, err = s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{})
	requireNoError(t, err)
}

func TestLifecycleLegacyResumePreservesIndividualWithdrawal(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Legacy retirement")
	folder := filepath.Join(dir, "files")
	for _, name := range []string{"a.pdf", "b.pdf"} {
		copyFixture(t, filepath.Join(folder, name), "native.pdf")
	}
	root := addRoot(t, s, p, lib, folder)
	scan(t, s, root)
	var withdrawn string
	requireNoError(t, s.Database.Reader.QueryRow("SELECT id FROM documents WHERE original_filename='a.pdf'").Scan(&withdrawn))
	doc, err := s.Document(ctx, p, withdrawn)
	requireNoError(t, err)
	requireNoError(t, s.RemoveIndex(ctx, p, doc.ID, "individual", doc.Revision, domain.RequestMetadata{}))
	plan, err := s.RetirementPlan(ctx, p, root)
	requireNoError(t, err)
	requireNoError(t, s.Retire(ctx, p, root, plan["plan_id"].(string), "remove_index_references", domain.RequestMetadata{}))
	_, err = s.Database.Writer.Exec("DELETE FROM root_retired_documents WHERE root_id=?", root)
	requireNoError(t, err) // Pre-upgrade evidence is only plan + audit.
	live, err := s.root(ctx, root)
	requireNoError(t, err)
	requireNoError(t, s.ResumeRoot(ctx, p, root, live.Revision, domain.RequestMetadata{}))
	var removed int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE deleted_at IS NOT NULL AND id=?", withdrawn).Scan(&removed))
	if removed != 1 {
		t.Fatal("individual withdrawal revived")
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE deleted_at IS NULL").Scan(&removed))
	if removed != 1 {
		t.Fatal("legacy retirement not restored")
	}
}
func TestLifecycleConcurrentDeletionAndTransactionRollback(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	lib := managedLibrary(t, s, p, "managed")
	uploadFixture(t, s, p, lib)
	drain(t, s)
	plan, err := s.PlanDeletion(ctx, p, "library", lib, domain.RequestMetadata{})
	requireNoError(t, err)
	secondPlan, err := s.PlanDeletion(ctx, p, "library", lib, domain.RequestMetadata{})
	requireNoError(t, err)
	// An error after preparing the durable journal must restore ALL entity rows.
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("CREATE TRIGGER fail_lifecycle_delete BEFORE DELETE ON libraries BEGIN SELECT RAISE(ABORT,'synthetic interruption'); END")
		return err
	}))
	queued, err := s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{})
	requireNoError(t, err)
	if queued.State != "pending" {
		t.Fatal("operation was not queued", queued)
	}
	if _, err = s.PlanDeletion(ctx, p, "library", lib, domain.RequestMetadata{}); failureCode(err) != "DELETION_ALREADY_RUNNING" {
		t.Fatal("second pending operation planned", err)
	}
	if _, err = s.ConfirmDeletion(ctx, p, secondPlan.ID, secondPlan.Name, domain.RequestMetadata{}); failureCode(err) != "DELETION_ALREADY_RUNNING" {
		t.Fatal("second pending operation confirmed", err)
	}
	if err = s.processPendingDeletion(ctx); err == nil {
		t.Fatal("interrupted transaction committed")
	}
	failed, err := s.DeletionStatus(ctx, p, plan.ID)
	requireNoError(t, err)
	if failed.State != "failed" {
		t.Fatal("failure state not persisted", failed)
	}
	var n int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM upload_items WHERE library_id=?", lib).Scan(&n))
	if n != 1 {
		t.Fatal("rollback lost ownership")
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM deletion_cleanup WHERE operation_id=?", plan.ID).Scan(&n))
	if n != 0 {
		t.Fatal("uncommitted journal remained")
	}
	_, err = s.Database.Writer.Exec("DROP TRIGGER fail_lifecycle_delete")
	requireNoError(t, err)
	_, err = s.RetryDeletionCleanup(ctx, p, plan.ID)
	requireNoError(t, err)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{})
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		requireNoError(t, err)
	}
	requireNoError(t, s.processPendingDeletion(ctx))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE event_type='library.deleted' AND library_id=?", lib).Scan(&n))
	if n != 1 {
		t.Fatal("duplicate deletion", n)
	}
}

func TestPendingDeletionPreventsWatcherAndManualRescan(t *testing.T) {
	s, p, directory := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Delete versus watcher")
	path := filepath.Join(directory, "originals", "one.pdf")
	copyFixture(t, path, "native.pdf")
	root := addRoot(t, s, p, lib, filepath.Dir(path))
	drain(t, s)
	_, err := s.Database.Writer.Exec("UPDATE storage_roots SET last_scan_at=? WHERE id=?", domain.Timestamp(time.Now().Add(-48*time.Hour)), root)
	requireNoError(t, err)
	plan, err := s.PlanDeletion(ctx, p, "root", root, domain.RequestMetadata{})
	requireNoError(t, err)
	queued, err := s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{})
	requireNoError(t, err)
	if queued.State != "pending" {
		t.Fatal("confirmation was not durable", queued)
	}
	runtime := &Runtime{Service: s, directories: map[string]string{}, dirty: map[string]time.Time{}, degraded: map[string]string{}, watchInitialized: map[string]bool{root: true}}
	for i := 0; i < 3; i++ {
		runtime.scheduleOnce(ctx)
	}
	if err = s.Verify(ctx, p, lib, root, domain.RequestMetadata{}); failureCode(err) != "DELETION_ALREADY_RUNNING" {
		t.Fatal("manual verification raced with confirmed deletion", err)
	}
	var pending int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE target_version=? AND status IN ('queued','running','retry_wait')", root).Scan(&pending))
	if pending != 0 {
		t.Fatal("watcher scheduled during deletion", pending)
	}
	requireNoError(t, s.processPendingDeletion(ctx))
	result, err := s.DeletionStatus(ctx, p, plan.ID)
	requireNoError(t, err)
	if result.State != "complete" {
		t.Fatal("deletion did not complete", result)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("original document was removed", err)
	}
}

func lifecyclePreview(t *testing.T, s *Service, p domain.Principal, doc string) string {
	t.Helper()
	id := domain.NewID()
	_, err := s.Database.Writer.Exec(`INSERT INTO preview_cache(id,document_id,content_version_id,source_hash,generator_version,requested_by,status,created_at,last_accessed_at) SELECT ?,d.id,f.current_content_version_id,v.sha256,'synthetic',?,'ready',?,? FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN content_versions v ON v.id=f.current_content_version_id WHERE d.id=?`, id, p.User.ID, now(), now(), doc)
	requireNoError(t, err)
	requireNoError(t, os.MkdirAll(s.previewDirectory(), 0700))
	requireNoError(t, os.WriteFile(s.previewPath(id), []byte("synthetic cache "+id), 0600))
	return id
}
func TestLifecycleRootTemporaryIsolation(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	lib := managedLibrary(t, s, p, "hybrid")
	upload := uploadFixture(t, s, p, lib)
	var locator string
	requireNoError(t, s.Database.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE document_id=?", upload.DocumentID).Scan(&locator))
	roots := []string{}
	previews := []string{}
	for _, name := range []string{"a", "b", "c"} {
		path := filepath.Join(dir, name, "original.pdf")
		copyFixture(t, path, "native.pdf")
		r := addRoot(t, s, p, lib, filepath.Dir(path))
		scan(t, s, r)
		roots = append(roots, r)
		var doc string
		requireNoError(t, s.Database.Reader.QueryRow("SELECT d.id FROM documents d JOIN physical_file_locations l ON l.physical_file_id=d.physical_file_id WHERE l.root_id=?", r).Scan(&doc))
		previews = append(previews, lifecyclePreview(t, s, p, doc))
	}
	drain(t, s)
	plan, err := s.PlanDeletion(ctx, p, "root", roots[1], domain.RequestMetadata{})
	requireNoError(t, err)
	completeDeletionForTest(t, s, p, plan)
	for i, id := range previews {
		_, err = os.Stat(s.previewPath(id))
		if i == 1 {
			if !os.IsNotExist(err) {
				t.Fatal("exclusive preview not removed", err)
			}
		} else if err != nil {
			t.Fatal("foreign preview removed", err)
		}
	}
	if _, err = os.Stat(filepath.Join(s.Identity.Config.PrivateUploadDirectory(), locator)); err != nil {
		t.Fatal("unbound library upload removed", err)
	}
	for _, name := range []string{"a", "b", "c"} {
		original, err := os.ReadFile(filepath.Join(dir, name, "original.pdf"))
		requireNoError(t, err)
		want, err := os.ReadFile("../../testdata/documents/native.pdf")
		requireNoError(t, err)
		if !bytes.Equal(original, want) {
			t.Fatal("original changed")
		}
	}
	scan(t, s, roots[0])
	scan(t, s, roots[2])
}
func TestLifecycleSharedRootReferencePreservesDocumentAndPreview(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Shared reference")
	a := filepath.Join(dir, "a", "same.pdf")
	b := filepath.Join(dir, "b", "alias.pdf")
	copyFixture(t, a, "native.pdf")
	requireNoError(t, os.MkdirAll(filepath.Dir(b), 0700))
	requireNoError(t, os.Link(a, b))
	rootA := addRoot(t, s, p, lib, filepath.Dir(a))
	rootB := addRoot(t, s, p, lib, filepath.Dir(b))
	scan(t, s, rootA)
	scan(t, s, rootB)
	var doc string
	requireNoError(t, s.Database.Reader.QueryRow("SELECT id FROM documents WHERE library_id=?", lib).Scan(&doc))
	preview := lifecyclePreview(t, s, p, doc)
	drain(t, s)
	plan, err := s.PlanDeletion(ctx, p, "root", rootA, domain.RequestMetadata{})
	requireNoError(t, err)
	if plan.Counts["documents"] != 0 || plan.Counts["shared_references"] != 1 {
		t.Fatal(plan)
	}
	completeDeletionForTest(t, s, p, plan)
	if _, err = os.Stat(s.previewPath(preview)); err != nil {
		t.Fatal("shared preview removed", err)
	}
	d, err := s.Document(ctx, p, doc)
	requireNoError(t, err)
	if d.RootID != rootB {
		t.Fatal("primary reference not redirected", d)
	}
	for _, path := range []string{a, b} {
		if _, err = os.Stat(path); err != nil {
			t.Fatal("shared original removed", err)
		}
	}
	var records int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE library_id=?", lib).Scan(&records))
	if records != 1 {
		t.Fatal("document duplicated or lost")
	}
}

// Commit the same durable boundary as ConfirmDeletion, without running the cleaner.
// This simulates a process exiting after commit and before (or during) file cleanup.
func lifecycleInterrupted(t *testing.T, s *Service, p domain.Principal, plan DeletionPlan) {
	t.Helper()
	requireNoError(t, s.Database.Write(context.Background(), func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM deletion_documents WHERE operation_id=?", plan.ID); err != nil {
			return err
		}
		if err := s.journalDeletionFiles(context.Background(), tx, plan, true); err != nil {
			return err
		}
		if err := s.removeEntityRecords(context.Background(), tx, plan); err != nil {
			return err
		}
		_, err := tx.Exec("UPDATE entity_deletions SET state='cleanup_pending' WHERE id=?", plan.ID)
		return err
	}))
}
func TestLifecycleCleanupRecoveryAndUnsafeFiles(t *testing.T) {
	for _, scenario := range []string{"commit", "renamed", "removed", "changed", "symlink", "permission", "container_replaced", "shared"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, dir := fixture(t)
			ctx := context.Background()
			lib := managedLibrary(t, s, p, "managed")
			item := uploadFixture(t, s, p, lib)
			drain(t, s)
			var locator string
			requireNoError(t, s.Database.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE document_id=?", item.DocumentID).Scan(&locator))
			plan, err := s.PlanDeletion(ctx, p, "library", lib, domain.RequestMetadata{})
			requireNoError(t, err)
			lifecycleInterrupted(t, s, p, plan)
			private := s.Identity.Config.PrivateUploadDirectory()
			path := filepath.Join(private, locator)
			var quarantine string
			requireNoError(t, s.Database.Reader.QueryRow("SELECT quarantine FROM deletion_cleanup WHERE operation_id=? AND kind='upload'", plan.ID).Scan(&quarantine))
			target := filepath.Join(dir, "external-original.pdf")
			requireNoError(t, os.WriteFile(target, []byte("never remove"), 0600))
			switch scenario {
			case "renamed":
				requireNoError(t, os.Rename(path, filepath.Join(private, quarantine)))
			case "removed":
				requireNoError(t, os.Remove(path))
			case "changed":
				requireNoError(t, os.WriteFile(path, []byte("unexpected different content"), 0600))
			case "symlink":
				requireNoError(t, os.Remove(path))
				requireNoError(t, os.Symlink(target, path))
			case "permission":
				requireNoError(t, os.Chmod(private, 0500))
				defer os.Chmod(private, 0700)
			case "container_replaced":
				requireNoError(t, os.Rename(private, private+"-original"))
				requireNoError(t, os.Mkdir(private, 0700))
				requireNoError(t, os.WriteFile(path, []byte("different container"), 0600))
			case "shared":
				other := managedLibrary(t, s, p, "managed")
				foreign := uploadFixture(t, s, p, other)
				_, err = s.Database.Writer.Exec("UPDATE upload_items SET private_temporary_locator=? WHERE document_id=?", locator, foreign.DocumentID)
				requireNoError(t, err)
			}
			resumed := New(s.Identity)
			requireNoError(t, resumed.cleanDeletionFiles(ctx, plan.ID, 100))
			state, err := resumed.DeletionStatus(ctx, p, plan.ID)
			requireNoError(t, err)
			shouldComplete := scenario == "commit" || scenario == "renamed" || scenario == "removed"
			if shouldComplete && state.State != "complete" {
				t.Fatal(state)
			}
			if !shouldComplete && (state.State != "cleanup_pending" || state.Issues != 1) {
				t.Fatal("unsafe cleanup not reported", state)
			}
			if !shouldComplete {
				if _, err = os.Lstat(path); err != nil {
					t.Fatal("unsafe file removed", err)
				}
			}
			if scenario == "permission" {
				requireNoError(t, os.Chmod(private, 0700))
				state, err = resumed.RetryDeletionCleanup(ctx, p, plan.ID)
				requireNoError(t, err)
				if state.State != "complete" {
					t.Fatal("permission retry failed", state)
				}
			}
			preserved, err := os.ReadFile(target)
			requireNoError(t, err)
			if string(preserved) != "never remove" {
				t.Fatal("external changed")
			}
			requireNoError(t, resumed.cleanDeletionFiles(ctx, plan.ID, 100))
		})
	}
}
func TestLifecycleCleanupPreparationRejectsUnsafeOwnership(t *testing.T) {
	for _, scenario := range []string{"traversal", "symlink", "replaced", "container_symlink"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, dir := fixture(t)
			ctx := context.Background()
			library := managedLibrary(t, s, p, "managed")
			item := uploadFixture(t, s, p, library)
			drain(t, s)
			var locator string
			requireNoError(t, s.Database.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE document_id=?", item.DocumentID).Scan(&locator))
			private := s.Identity.Config.PrivateUploadDirectory()
			path := filepath.Join(private, locator)
			target := filepath.Join(dir, "original fuera del almacén.pdf")
			requireNoError(t, os.WriteFile(target, []byte("original preserved"), 0600))
			switch scenario {
			case "traversal":
				_, err := s.Database.Writer.Exec("UPDATE upload_items SET private_temporary_locator=? WHERE document_id=?", "../"+filepath.Base(target), item.DocumentID)
				requireNoError(t, err)
			case "symlink":
				requireNoError(t, os.Remove(path))
				requireNoError(t, os.Symlink(target, path))
			case "replaced":
				requireNoError(t, os.Remove(path))
				requireNoError(t, os.WriteFile(path, []byte("different identity or content"), 0600))
			case "container_symlink":
				requireNoError(t, os.Rename(private, private+"-saved"))
				requireNoError(t, os.Symlink(private+"-saved", private))
			}
			plan, err := s.PlanDeletion(ctx, p, "library", library, domain.RequestMetadata{})
			requireNoError(t, err)
			result := completeDeletionForTest(t, s, p, plan)
			if result.State != "cleanup_pending" || result.Issues != 1 {
				t.Fatal("unsafe ownership was accepted", result)
			}
			if _, err = os.Lstat(path); err != nil {
				t.Fatal("unverified file was removed", err)
			}
			content, err := os.ReadFile(target)
			requireNoError(t, err)
			if string(content) != "original preserved" {
				t.Fatal("external document modified")
			}
		})
	}
}

func TestLifecycleArchiveQueuesValidationAndStalePlan(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	lib := managedLibrary(t, s, p, "hybrid")
	item := uploadFixture(t, s, p, lib)
	preview := lifecyclePreview(t, s, p, item.DocumentID)
	_, err := s.Database.Writer.Exec("UPDATE preview_cache SET status='queued' WHERE id=?", preview)
	requireNoError(t, err)
	path := filepath.Join(dir, "root")
	requireNoError(t, os.Mkdir(path, 0700))
	root := addRoot(t, s, p, lib, path)
	var revision int64
	requireNoError(t, s.Database.Reader.QueryRow("SELECT revision FROM libraries WHERE id=?", lib).Scan(&revision))
	requireNoError(t, s.ArchiveLibrary(ctx, p, lib, revision, false, domain.RequestMetadata{}))
	job, err := s.claim(ctx)
	if err != nil && err != sql.ErrNoRows {
		t.Fatal(err)
	}
	if job.ID != "" {
		t.Fatal("archived job claimed")
	}
	pj, _, err := s.claimPreview(ctx)
	requireNoError(t, err)
	if pj.ID != "" {
		t.Fatal("archived preview claimed")
	}
	requireNoError(t, os.Rename(path, path+"-old"))
	if err = s.ArchiveLibrary(ctx, p, lib, revision+1, true, domain.RequestMetadata{}); failureCode(err) != "ROOT_NOT_FOUND" {
		t.Fatal("unavailable restore", err)
	}
	requireNoError(t, os.Rename(path+"-old", path))
	requireNoError(t, s.ArchiveLibrary(ctx, p, lib, revision+1, true, domain.RequestMetadata{}))
	drain(t, s)
	plan, err := s.PlanDeletion(ctx, p, "library", lib, domain.RequestMetadata{})
	requireNoError(t, err)
	uploadFixture(t, s, p, lib)
	drain(t, s)
	if _, err = s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{}); failureCode(err) != "DELETION_PLAN_STALE" {
		t.Fatal("stale plan permitted", err)
	}
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error { return enqueueScan(ctx, tx, lib, root) }))
	_, err = s.claimLane(ctx, "scan")
	requireNoError(t, err)
	if _, err = s.PlanDeletion(ctx, p, "root", root, domain.RequestMetadata{}); failureCode(err) != "ENTITY_BUSY" {
		t.Fatal("running scan not protected", err)
	}
}
func TestLifecycleManagedOriginalAndDependencies(t *testing.T) {
	f := workflowSetup(t, false)
	s := f.service
	ctx := context.Background()
	if _, err := s.PlanDeletion(ctx, f.author, "root", f.root, domain.RequestMetadata{}); failureCode(err) != "MANAGED_ROOT_IN_USE" {
		t.Fatal(err)
	}
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	if _, err = s.PlanDeletion(ctx, f.author, "library", f.library, domain.RequestMetadata{}); failureCode(err) != "ENTITY_BUSY" {
		t.Fatal("pending publication not protected", err)
	}
	drain(t, s)
	doc := workflowDocumentFor(t, f)
	before, err := os.ReadFile(doc.OriginalPath)
	requireNoError(t, err)
	plan, err := s.PlanDeletion(ctx, f.author, "library", f.library, domain.RequestMetadata{})
	requireNoError(t, err)
	if plan.Counts["cases"] != 1 || plan.Counts["preserved_documents"] != 1 {
		t.Fatal(plan.Counts)
	}
	var manifest strings.Builder
	requireNoError(t, s.DeletionManifest(ctx, f.author, plan.ID, func(line string) error { manifest.WriteString(line); return nil }))
	if !strings.Contains(manifest.String(), doc.Hash) {
		t.Fatal("pre-confirmation manifest missing")
	}
	completeDeletionForTest(t, s, f.author, plan)
	after, err := os.ReadFile(doc.OriginalPath)
	requireNoError(t, err)
	if !bytes.Equal(before, after) {
		t.Fatal("managed original changed")
	}
	var n int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM pragma_foreign_key_check").Scan(&n))
	if n != 0 {
		t.Fatal("dangling relationships")
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE library_id=?", f.library).Scan(&n))
	if n == 0 {
		t.Fatal("audit lost")
	}
}
func TestLifecycleDirectoryDiagnostics(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Paths")
	path := filepath.Join(dir, "protected")
	requireNoError(t, os.Mkdir(path, 0700))
	r := addRoot(t, s, p, lib, path)
	plan, err := s.PlanRoot(ctx, p, lib, path, domain.RequestMetadata{})
	requireNoError(t, err)
	if plan.Relation != "equal" || plan.Notice == "" {
		t.Fatal(plan)
	}
	retirement, err := s.RetirementPlan(ctx, p, r)
	requireNoError(t, err)
	requireNoError(t, s.Retire(ctx, p, r, retirement["plan_id"].(string), "retain_index", domain.RequestMetadata{}))
	plan, err = s.PlanRoot(ctx, p, lib, path, domain.RequestMetadata{})
	requireNoError(t, err)
	if !strings.Contains(plan.Notice, "reanud") {
		t.Fatal(plan.Notice)
	}
	requireNoError(t, os.Chmod(path, 0000))
	defer os.Chmod(path, 0700)
	if _, err = s.PlanRoot(ctx, p, lib, path, domain.RequestMetadata{}); failureCode(err) != "ROOT_PERMISSION_DENIED" {
		t.Fatal("permission diagnosis", err)
	}
	requireNoError(t, os.Chmod(path, 0700))
	if _, err = s.PlanRoot(ctx, p, lib, filepath.Join(dir, "missing"), domain.RequestMetadata{}); failureCode(err) != "ROOT_NOT_FOUND" {
		t.Fatal(err)
	}
	other := addLibrary(t, s, p, "Other")
	if _, err = s.PlanRoot(ctx, p, other, path, domain.RequestMetadata{}); failureCode(err) != "ROOT_AUTHORIZATION_CONFLICT" {
		t.Fatal(err)
	}
}
func TestLifecycleProcessingTablesBeyond100AndLiteralSearch(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Tables")
	path := filepath.Join(dir, "files")
	requireNoError(t, os.Mkdir(path, 0700))
	r := addRoot(t, s, p, lib, path)
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error {
		for i := 0; i < 137; i++ {
			_, err := tx.Exec("INSERT INTO document_file_skips VALUES(?,?,?,?,?)", r, fmt.Sprintf("odd_%%_%03d.xlsx", i), "FILE_FORMAT_DISABLED", "{}", now())
			if err != nil {
				return err
			}
		}
		return nil
	}))
	f := TableFilter{Size: 25, Page: 6, Sort: "path", Direction: "asc", Query: "odd_%_"}
	page, err := s.FileSkips(ctx, p, lib, f)
	requireNoError(t, err)
	if page.Total != 137 || len(page.Items) != 12 {
		t.Fatal(page)
	}
	f.State = "FILE_TYPE_BLOCKED"
	page, err = s.FileSkips(ctx, p, lib, f)
	requireNoError(t, err)
	if page.Total != 0 || page.Page != 1 {
		t.Fatal(page)
	}
	f = TableFilter{Size: 10, Page: 1, Sort: "date", Direction: "desc", Kind: "scan"}
	jobs, err := s.JobHistory(ctx, p, lib, f)
	requireNoError(t, err)
	if jobs.Total != 1 || len(jobs.Items) != 1 || jobs.Items[0].RootID != r {
		t.Fatal(jobs)
	}
	f.Sort = "created_at; DROP TABLE jobs"
	if _, err = s.JobHistory(ctx, p, lib, f); err == nil {
		t.Fatal("unsafe sorting accepted")
	}
	outsider := member(t, s, p, lib, "table-reader", "library_reader")
	if _, err = s.FileSkips(ctx, outsider, lib, TableFilter{Size: 10, Page: 1, Sort: "path", Direction: "asc"}); err == nil {
		t.Fatal("unauthorized diagnostics")
	}
}
func TestLifecycleRootResumeAndIsolation(t *testing.T) {
	s, p, directory := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Raíces")
	ids := []string{}
	for _, name := range []string{"a", "b"} {
		path := filepath.Join(directory, name, "original.pdf")
		copyFixture(t, path, "native.pdf")
		id := addRoot(t, s, p, lib, filepath.Dir(path))
		scan(t, s, id)
		ids = append(ids, id)
	}
	plan, err := s.RetirementPlan(ctx, p, ids[0])
	requireNoError(t, err)
	requireNoError(t, s.Retire(ctx, p, ids[0], plan["plan_id"].(string), "remove_index_references", domain.RequestMetadata{}))
	root, err := s.root(ctx, ids[0])
	requireNoError(t, err)
	requireNoError(t, s.ResumeRoot(ctx, p, root.ID, root.Revision, domain.RequestMetadata{}))
	var removed int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE deleted_at IS NOT NULL").Scan(&removed))
	if removed != 0 {
		t.Fatal("references not restored")
	}
	drain(t, s)
	deletion, err := s.PlanDeletion(ctx, p, "root", root.ID, domain.RequestMetadata{})
	requireNoError(t, err)
	result := completeDeletionForTest(t, s, p, deletion)
	if result.State != "complete" {
		t.Fatal(result)
	}
	for _, name := range []string{"a", "b"} {
		if _, err = os.Stat(filepath.Join(directory, name, "original.pdf")); err != nil {
			t.Fatal("original changed", err)
		}
	}
	roots, err := s.Roots(ctx, p, lib)
	requireNoError(t, err)
	if len(roots) != 1 || roots[0].ID != ids[1] {
		t.Fatal("wrong root removed", roots)
	}
	if err = s.ResumeRoot(ctx, p, root.ID, root.Revision+1, domain.RequestMetadata{}); err == nil {
		t.Fatal("deleted root revived")
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM pragma_foreign_key_check").Scan(&removed))
	if removed != 0 {
		t.Fatal("dangling FKs")
	}
}
