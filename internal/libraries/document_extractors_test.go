//go:build development

package libraries

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
)

func allowContentFormats(t *testing.T, service *Service, admin domain.Principal) {
	t.Helper()
	ctx := context.Background()
	settings, err := service.FileConfiguration(ctx, admin, "")
	requireNoError(t, err)
	settings.Effective.Index = []string{"pdf", "docx", "xlsx", "txt", "csv"}
	requireNoError(t, service.ConfigureFiles(ctx, admin, "", settings.Effective, documentformat.Overrides{}, settings.Revision, settings.GlobalRevision, domain.RequestMetadata{}))
}
func drainDocumentContent(t *testing.T, service *Service) {
	t.Helper()
	ctx := context.Background()
	for count := 0; count < 20; count++ {
		job, err := service.claim(ctx)
		if err == sql.ErrNoRows {
			return
		}
		requireNoError(t, err)
		if job.Kind == "scan" {
			requireNoError(t, service.Scan(ctx, job.Version, 256<<20, nil))
		} else if job.Kind == "verify_managed" {
			requireNoError(t, service.VerifyManagedRoot(ctx, job.Version))
		} else {
			options := service.Identity.Config.Indexing
			options.PDFInfo = "nonexistent-pdf-program"
			options.Tesseract = "nonexistent-ocr-program"
			requireNoError(t, service.Extract(ctx, job, options))
		}
		requireNoError(t, service.finish(ctx, job, nil))
	}
	t.Fatal("document queue did not drain")
}
func TestNewExtractorsManagedSearchAndFirstIndexAfterPolicyChange(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "hybrid")
	documents := map[string]string{}
	for _, format := range []string{"docx", "xlsx", "txt", "csv"} {
		contents, err := os.ReadFile("../../testdata/documents/sample." + format)
		requireNoError(t, err)
		item := uploadFormat(t, service, admin, library, "Original."+format, contents)
		documents[format] = item.DocumentID
	}
	allowContentFormats(t, service, admin)
	drainDocumentContent(t, service)
	for format, term := range map[string]string{"docx": "conservación", "xlsx": "Prueba", "txt": "cambios", "csv": "Arena"} {
		document, err := service.Document(ctx, admin, documents[format])
		requireNoError(t, err)
		if document.Freshness != "current" || document.Pages != 0 || document.Units < 1 || document.IndexReason != "" || document.Processing == nil || !strings.HasPrefix(document.Processing.Status, "complete") || document.Processing.Extractor.Format != format || document.Processing.Extractor.Version != map[string]string{"docx": "2", "xlsx": "1", "txt": "1", "csv": "1"}[format] || document.Processing.Started == "" || document.Processing.Completed == "" {
			t.Fatalf("%+v / %+v", document, document.Processing)
		}
		found := query(t, service, admin, library, term)
		if found.Count != 1 || found.Items[0].ID != document.ID || len(found.Items[0].Matches) < 1 || found.Items[0].Matches[0].Label == "" || found.Items[0].Matches[0].Kind == "page" || found.Items[0].Preview || !found.Items[0].Download {
			t.Fatalf("search lost format/context: %+v", found)
		}
		unit, err := service.Page(ctx, admin, document.ID, found.Items[0].Matches[0].Page)
		requireNoError(t, err)
		if unit.Kind == "page" || unit.Method != "native" || unit.Context == nil {
			t.Fatalf("%+v", unit)
		}
		file, _, err := service.OpenDocument(ctx, admin, document.ID, true, domain.RequestMetadata{})
		requireNoError(t, err)
		download, err := io.ReadAll(file)
		file.Close()
		requireNoError(t, err)
		original, err := os.ReadFile("../../testdata/documents/sample." + format)
		requireNoError(t, err)
		if !bytes.Equal(download, original) {
			t.Fatal("source changed")
		}
	}
	// Saving the same effective settings does not reprocess already current text.
	allowContentFormats(t, service, admin)
	var count int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM extraction_runs").Scan(&count))
	if count != 4 {
		t.Fatal(count)
	}
	job, err := service.claim(ctx)
	if err != sql.ErrNoRows {
		t.Fatal("unexpected repeat", job, err)
	}
}
func TestLinkedTextChangesReplaceFTSAndRetainEvidence(t *testing.T) {
	service, admin, directory := fixture(t)
	library := addLibrary(t, service, admin, "Texto vinculado")
	allowContentFormats(t, service, admin)
	folder := filepath.Join(directory, "linked-text")
	requireNoError(t, os.Mkdir(folder, 0700))
	file := filepath.Join(folder, "Texto.txt")
	write := func(contents string) {
		requireNoError(t, os.WriteFile(file, []byte(contents), 0600))
		past := time.Now().Add(-5 * time.Second)
		requireNoError(t, os.Chtimes(file, past, past))
	}
	write("Primer contenido irrepetibleoriginal")
	root := addRoot(t, service, admin, library, folder)
	scan(t, service, root)
	drainDocumentContent(t, service)
	first := query(t, service, admin, library, "irrepetibleoriginal")
	if first.Count != 1 {
		t.Fatal(first)
	}
	scan(t, service, root)
	drainDocumentContent(t, service)
	var unchangedRuns int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM extraction_runs").Scan(&unchangedRuns))
	if unchangedRuns != 1 {
		t.Fatal("unchanged hash was reprocessed", unchangedRuns)
	}
	write("Segundo contenido irrepetiblenuevo")
	scan(t, service, root)
	drainDocumentContent(t, service)
	second := query(t, service, admin, library, "irrepetiblenuevo")
	if second.Count != 1 || second.Items[0].ID != first.Items[0].ID || second.Items[0].Hash == first.Items[0].Hash {
		t.Fatal(second)
	}
	if query(t, service, admin, library, "irrepetibleoriginal").Count != 0 {
		t.Fatal("old text remained in FTS")
	}
	var runs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM extraction_runs").Scan(&runs))
	if runs != 2 {
		t.Fatal(runs)
	}
	requireNoError(t, os.Remove(file))
	scan(t, service, root)
	if query(t, service, admin, library, "irrepetiblenuevo").Count != 1 {
		t.Fatal("missing source lost retained text")
	}
}

func TestLegacyWordRunOpensFirstNonemptyUnit(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "managed")
	allowContentFormats(t, service, admin)
	original, err := os.ReadFile("../../testdata/documents/sample.docx")
	requireNoError(t, err)
	item := uploadFormat(t, service, admin, library, "Legacy.docx", original)
	drainDocumentContent(t, service)
	document, err := service.Document(ctx, admin, item.DocumentID)
	requireNoError(t, err)
	extractionID := document.Processing.ID
	// Reproduce the v1 storage layout: its first paragraph was retained empty.
	_, err = service.Database.Writer.Exec(`INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text,unit_kind,context_label,context_json) SELECT extraction_id,2,extraction_method,page_text,unit_kind,context_label,context_json FROM extraction_pages WHERE extraction_id=? AND page_number=1`, extractionID)
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec("UPDATE indexed_pages SET page_number=2 WHERE extraction_id=?", extractionID)
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec("UPDATE extraction_pages SET page_text='' WHERE extraction_id=? AND page_number=1", extractionID)
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec("UPDATE extraction_runs SET unit_count=2,extractor_version='1' WHERE id=?", extractionID)
	requireNoError(t, err)
	current, err := service.Document(ctx, admin, item.DocumentID)
	requireNoError(t, err)
	if current.FirstTextUnit != 2 || current.Processing.Extractor.Version != "1" || current.Hash != document.Hash {
		t.Fatal(current)
	}
	unit, err := service.Page(ctx, admin, item.DocumentID, current.FirstTextUnit)
	requireNoError(t, err)
	if !strings.Contains(unit.Text, "conservación") {
		t.Fatal(unit)
	}
}
func TestExtractorFailureRecordedWithoutPartialIndex(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "managed")
	allowContentFormats(t, service, admin)
	item := uploadFormat(t, service, admin, library, "Long.txt", []byte(strings.Repeat("x", (4<<20)+10)))
	job, err := service.claim(ctx)
	requireNoError(t, err)
	if err = service.Extract(ctx, job, service.Identity.Config.Indexing); err == nil {
		t.Fatal("oversized line accepted")
	}
	requireNoError(t, service.finish(ctx, job, err))
	document, err := service.Document(ctx, admin, item.DocumentID)
	requireNoError(t, err)
	if document.Processing == nil || document.Processing.Status != "failed" || document.Processing.Error == "" || document.Processing.Extractor.ID != "text-unicode" || document.Freshness == "current" {
		t.Fatalf("%+v", document.Processing)
	}
	var count int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM indexed_pages").Scan(&count))
	if count != 0 {
		t.Fatal("partial index published")
	}
}
