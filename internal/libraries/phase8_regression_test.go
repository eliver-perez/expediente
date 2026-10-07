//go:build development

package libraries

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"gestor-documental/internal/domain"
)

// Exercise the Office dispatch during reindexing, in addition to the existing
// text-file tests of concurrency, identity and failed-publication recovery.
func TestPhase8OfficeReindexPreservesOriginalAndSearch(t *testing.T) {
	for _, format := range []string{"docx", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			service, admin, directory := fixture(t)
			ctx := context.Background()
			library := addLibrary(t, service, admin, "Office reindex")
			allowContentFormats(t, service, admin)
			path := filepath.Join(directory, "originals", "sample."+format)
			copyFixture(t, path, "sample."+format)
			original, err := os.ReadFile(path)
			requireNoError(t, err)
			addRoot(t, service, admin, library, filepath.Dir(path))
			drainDocumentContent(t, service)
			found := query(t, service, admin, library, "sintético")
			if found.Count != 1 {
				t.Fatal("Office text was not searchable", found.Count)
			}
			before, err := service.Document(ctx, admin, found.Items[0].ID)
			requireNoError(t, err)
			_, err = service.ReindexDocument(ctx, admin, before.ID, domain.NewID(), domain.RequestMetadata{})
			requireNoError(t, err)
			drainDocumentContent(t, service)
			after, err := service.Document(ctx, admin, before.ID)
			requireNoError(t, err)
			if after.Format != format || after.Freshness != "current" || after.Hash != before.Hash || after.FileID != before.FileID || after.Processing.ID == before.Processing.ID {
				t.Fatal("Office reindex lost identity, format or current publication")
			}
			found = query(t, service, admin, library, "sintético")
			if found.Count != 1 || found.Items[0].ID != before.ID {
				t.Fatal("Office reindex duplicated or lost search results")
			}
			current, err := os.ReadFile(path)
			requireNoError(t, err)
			if !bytes.Equal(original, current) {
				t.Fatal("Office reindex changed original bytes")
			}
		})
	}
}

func TestPhase8PreviewCacheCannotBecomeLinkedRoot(t *testing.T) {
	service, admin, _, id := previewFixture(t)
	ctx := context.Background()
	_, err := service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	ready, err := service.Preview(ctx, admin, id)
	requireNoError(t, err)
	if ready.State != "ready" {
		t.Fatal("preview not generated")
	}
	library := addLibrary(t, service, admin, "Cache boundary")
	for _, path := range []string{service.previewDirectory(), service.Identity.Config.StateDirectory, filepath.Dir(service.Identity.Config.StateDirectory)} {
		if _, err = service.PlanRoot(ctx, admin, library, path, domain.RequestMetadata{}); failureCode(err) != "INVALID_REQUEST" {
			t.Fatal("private preview directory or ancestor accepted as library", err)
		}
	}
	file, _, err := service.OpenPreview(ctx, admin, id, ready.ID, domain.RequestMetadata{})
	requireNoError(t, err)
	file.Close()
}
