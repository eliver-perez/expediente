//go:build development

package libraries

import (
	"context"
	"os"
	"testing"

	"gestor-documental/internal/domain"
)

func TestManagedIntegrityContinuesAfterUnreadableFile(t *testing.T) {
	f := workflowSetup(t, false)
	s, ctx := f.service, context.Background()
	item := uploadFixture(t, s, f.author, f.library)
	requireNoError(t, s.Classify(ctx, f.author, item.DocumentID, "classification", Classification{CaseID: f.document.CaseID, CategoryID: f.document.CategoryID, TypeID: f.document.TypeID}, 1, domain.RequestMetadata{}))
	drain(t, s)
	for _, id := range []string{f.document.ID, item.DocumentID} {
		d, err := s.Document(ctx, f.author, id)
		requireNoError(t, err)
		_, err = s.FinalizeDocument(ctx, f.author, id, d.Revision, domain.RequestMetadata{})
		requireNoError(t, err)
	}
	drain(t, s)
	a, err := s.Document(ctx, f.author, f.document.ID)
	requireNoError(t, err)
	b, err := s.Document(ctx, f.author, item.DocumentID)
	requireNoError(t, err)
	if a.FileID > b.FileID {
		a, b = b, a
	}
	// A directory in place of one recorded file produces a portable read failure.
	// The second document must still be checked and its external change detected.
	requireNoError(t, os.Rename(a.OriginalPath, a.OriginalPath+".saved"))
	requireNoError(t, os.Mkdir(a.OriginalPath, 0700))
	file, err := os.OpenFile(b.OriginalPath, os.O_APPEND|os.O_WRONLY, 0600)
	requireNoError(t, err)
	_, err = file.WriteString("\n% QA external change\n")
	requireNoError(t, err)
	requireNoError(t, file.Close())
	if err = s.VerifyManagedRoot(ctx, f.root); failureCode(err) != "SCAN_PARTIAL" {
		t.Fatalf("expected partial verification, got %v", err)
	}
	b, err = s.Document(ctx, f.author, b.ID)
	requireNoError(t, err)
	if b.Approval != "needs_review" || b.Integrity != "changed" {
		t.Fatal("remaining file was not verified", b.Approval, b.Integrity)
	}
	var count int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM scan_errors WHERE root_id=? AND relative_path=?", f.root, a.RelativePath).Scan(&count))
	if count != 1 {
		t.Fatal("missing per-file diagnostic")
	}
	requireNoError(t, os.Remove(a.OriginalPath))
	requireNoError(t, os.Rename(a.OriginalPath+".saved", a.OriginalPath))
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM scan_errors WHERE root_id=?", f.root).Scan(&count))
	if count != 0 {
		t.Fatal("resolved file error was not cleared")
	}
}
