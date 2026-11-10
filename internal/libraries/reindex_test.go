//go:build development

package libraries

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

func TestManualReindexRetainsIdentityHistoryAndPublishesCurrentExtractor(t *testing.T) {
	service, admin, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, admin, "Reindexación")
	allowContentFormats(t, service, admin)
	path := filepath.Join(directory, "originals", "sample.txt")
	copyFixture(t, path, "sample.txt")
	addRoot(t, service, admin, library, filepath.Dir(path))
	drainDocumentContent(t, service)
	first := query(t, service, admin, library, "").Items[0]
	before, err := service.Document(ctx, admin, first.ID)
	requireNoError(t, err)
	if !before.CanReindex {
		t.Fatal("manager cannot reindex")
	}
	reader := member(t, service, admin, library, "reindex-reader", "library_reader")
	denied, err := service.Document(ctx, reader, first.ID)
	requireNoError(t, err)
	if denied.CanReindex {
		t.Fatal("reader reindex button")
	}
	if _, err = service.ReindexDocument(ctx, reader, first.ID, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "NOT_FOUND" {
		t.Fatal(err)
	}

	// Concurrent clicks and different requests must share one extraction.
	var group sync.WaitGroup
	jobs := make(chan string, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			job, err := service.ReindexDocument(ctx, admin, first.ID, domain.NewID(), domain.RequestMetadata{})
			jobs <- job
			failures <- err
		}()
	}
	group.Wait()
	close(jobs)
	close(failures)
	for err := range failures {
		requireNoError(t, err)
	}
	var jobID string
	for job := range jobs {
		if jobID != "" && job != jobID {
			t.Fatal("duplicate manual jobs")
		}
		jobID = job
	}
	key := domain.NewID()
	replay, err := service.ReindexDocument(ctx, admin, first.ID, key, domain.RequestMetadata{})
	requireNoError(t, err)
	if replay != jobID {
		t.Fatal("active job not reused")
	}
	waiting, err := service.Document(ctx, admin, first.ID)
	requireNoError(t, err)
	if waiting.Freshness != "stale" || waiting.Processing.ID != before.Processing.ID {
		t.Fatal("old text not retained", waiting)
	}
	drainDocumentContent(t, service)
	after, err := service.Document(ctx, admin, first.ID)
	requireNoError(t, err)
	if after.ID != before.ID || after.FileID != before.FileID || after.Hash != before.Hash || after.Processing.ID == before.Processing.ID || after.Freshness != "current" || after.Processing.Extractor.Version != "1" {
		t.Fatalf("bad reindex: %+v", after)
	}
	replay, err = service.ReindexDocument(ctx, admin, first.ID, key, domain.RequestMetadata{})
	requireNoError(t, err)
	if replay != jobID {
		t.Fatal("idempotent replay created another job")
	}
	if _, err = service.claim(ctx); err != sql.ErrNoRows {
		t.Fatal("unexpected work", err)
	}

	// A changed original is another content version, never another document.
	requireNoError(t, os.WriteFile(path, []byte("Contenido actualizado marcadornuevofasecinco"), 0600))
	_, err = service.ReindexDocument(ctx, admin, first.ID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	drainDocumentContent(t, service)
	found := query(t, service, admin, library, "marcadornuevofasecinco")
	if found.Count != 1 || found.Items[0].ID != first.ID {
		t.Fatalf("new content missing/duplicated: %+v", found)
	}
	if query(t, service, admin, library, "cambios").Count != 0 {
		t.Fatal("old FTS rows retained")
	}
	var docs, versions, runs, events int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE library_id=?", library).Scan(&docs))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM content_versions WHERE physical_file_id=?", first.FileID).Scan(&versions))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM extraction_runs WHERE physical_file_id=?", first.FileID).Scan(&runs))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE document_id=? AND event_type='document.reindex_requested'", first.ID).Scan(&events))
	if docs != 1 || versions != 2 || runs != 3 || events != 10 {
		t.Fatal(docs, versions, runs, events)
	}
	original, err := os.ReadFile(path)
	requireNoError(t, err)
	if string(original) != "Contenido actualizado marcadornuevofasecinco" {
		t.Fatal("original rewritten")
	}

	// Missing or rejected originals do not erase the published text.
	requireNoError(t, os.Remove(path))
	if _, err = service.ReindexDocument(ctx, admin, first.ID, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "DOCUMENT_UNAVAILABLE" {
		t.Fatal(err)
	}
	if query(t, service, admin, library, "marcadornuevofasecinco").Count != 1 {
		t.Fatal("missing original erased text")
	}
}

func TestManualReindexRulesPrivateVisibilityAndFailedAttemptRetainText(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "managed")
	contents, err := os.ReadFile("../../testdata/documents/sample.txt")
	requireNoError(t, err)
	item := uploadFormat(t, service, admin, library, "Texto.txt", contents)
	if _, err = service.ReindexDocument(ctx, admin, item.DocumentID, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "REINDEX_NOT_ALLOWED" {
		t.Fatal(err)
	}
	allowContentFormats(t, service, admin)
	drainDocumentContent(t, service)
	other := member(t, service, admin, library, "other-manager", "library_manager")
	if _, err = service.ReindexDocument(ctx, other, item.DocumentID, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "NOT_FOUND" {
		t.Fatal("private upload visible", err)
	}
	before, err := service.Document(ctx, admin, item.DocumentID)
	requireNoError(t, err)
	key := domain.NewID()
	_, err = service.ReindexDocument(ctx, admin, item.DocumentID, key, domain.RequestMetadata{})
	requireNoError(t, err)
	job, err := service.claim(ctx)
	requireNoError(t, err)
	requireNoError(t, service.CancelJob(ctx, admin, job.ID, domain.RequestMetadata{}))
	err = service.Extract(ctx, job, service.Identity.Config.Indexing)
	if err == nil {
		t.Fatal("cancelled extraction published")
	}
	requireNoError(t, service.finish(ctx, job, err))
	after, err := service.Document(ctx, admin, item.DocumentID)
	requireNoError(t, err)
	if after.Freshness != "stale" {
		t.Fatal(after.Freshness)
	}
	page, err := service.Page(ctx, admin, after.ID, 1)
	requireNoError(t, err)
	if !strings.Contains(page.Text, "cambios") {
		t.Fatal("failed reindex erased retained text", page)
	}
	var indexed string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT indexed_extraction_id FROM physical_files WHERE id=?", after.FileID).Scan(&indexed))
	if indexed != before.Processing.ID {
		t.Fatal("failed run replaced published index")
	}
	second := uploadFormat(t, service, admin, library, "Otra.txt", contents)
	if _, err = service.ReindexDocument(ctx, admin, second.DocumentID, key, domain.RequestMetadata{}); failureCode(err) != "IDEMPOTENCY_CONFLICT" {
		t.Fatal(err)
	}
}

func TestManualReindexManagedChangePreservesCaseAndRequiresReview(t *testing.T) {
	f := workflowSetup(t, false)
	ctx := context.Background()
	s := f.service
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	before := workflowDocumentFor(t, f)
	original, err := os.ReadFile(before.OriginalPath)
	requireNoError(t, err)
	changed := append(original, []byte("\n% changed for manual reindex\n")...)
	requireNoError(t, os.WriteFile(before.OriginalPath, changed, 0600))
	_, err = s.ReindexDocument(ctx, f.author, before.ID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	pending := workflowDocumentFor(t, f)
	if pending.CaseID != before.CaseID || pending.ID != before.ID || pending.Approval != "needs_review" || pending.Integrity != "changed" || pending.Hash == before.Hash {
		t.Fatalf("lost external change evidence: %+v", pending)
	}
	drain(t, s)
	after := workflowDocumentFor(t, f)
	if after.Processing.ID == before.Processing.ID || after.Freshness != "current" || after.Approval != "needs_review" {
		t.Fatal(after)
	}
	claims := attachLicense(t, s)
	claims.Revision++
	claims.Type = "subscription"
	claims.GraceDays = 15
	expired := time.Now().Add(-16 * 24 * time.Hour).UTC().Format(time.RFC3339)
	claims.ExpiresAt = &expired
	applyLicense(t, s, claims)
	if _, err = s.ReindexDocument(ctx, f.author, before.ID, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "LICENSE_READ_ONLY" {
		t.Fatal("license bypass", err)
	}
}
