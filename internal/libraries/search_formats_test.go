//go:build development

package libraries

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
)

func TestSearchFormatFiltersCountsCursorAndVisibility(t *testing.T) {
	service, admin, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, admin, "Formatos")
	folder := filepath.Join(directory, "formatos")
	for _, format := range []string{"pdf", "docx", "xlsx", "txt", "csv"} {
		sample := "sample." + format
		if format == "pdf" {
			sample = "native.pdf"
		}
		copyFixture(t, filepath.Join(folder, "Archivo."+format), sample)
	}
	addRoot(t, service, admin, library, folder)
	// Scanning suffices: format filtering must work before text extraction.
	roots, err := service.root(ctx, queryRoot(t, service, library))
	requireNoError(t, err)
	scan(t, service, roots.ID)
	other := addLibrary(t, service, admin, "Otra biblioteca")
	copyFixture(t, filepath.Join(directory, "otros", "Otra.txt"), "sample.txt")
	root := addRoot(t, service, admin, other, filepath.Join(directory, "otros"))
	scan(t, service, root)
	for _, format := range []string{"", "pdf", "docx", "xlsx", "txt", "csv"} {
		input := SearchInput{Libraries: []string{library}, Filters: Filters{Format: format}, Limit: 50}
		result, err := service.Search(ctx, admin, input, domain.RequestMetadata{}, true)
		requireNoError(t, err)
		expected := 1
		if format == "" {
			expected = 5
		}
		if result.Count != expected || len(result.Items) != expected || len(result.Groups) != 1 || result.Groups[0].Count != expected {
			t.Fatalf("%s: %+v", format, result)
		}
		for _, doc := range result.Items {
			if format != "" && doc.Format != format {
				t.Fatal(doc.Format)
			}
		}
	}
	input := SearchInput{Filters: Filters{Format: "txt"}, Limit: 1}
	first, err := service.Search(ctx, admin, input, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if first.Count != 2 || len(first.Groups) != 2 || first.Cursor == "" {
		t.Fatal(first)
	}
	input.Cursor = first.Cursor
	second, err := service.Search(ctx, admin, input, domain.RequestMetadata{}, false)
	requireNoError(t, err)
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatal(second)
	}
	input.Filters.Format = "pdf"
	if _, err = service.Search(ctx, admin, input, domain.RequestMetadata{}, false); err == nil {
		t.Fatal("cursor crossed format scope")
	}
	input = SearchInput{Filters: Filters{Format: "exe"}}
	if _, err = service.Search(ctx, admin, input, domain.RequestMetadata{}, false); err == nil {
		t.Fatal("invalid format accepted")
	}
	var filters string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT applied_filters_json FROM search_audit_details s JOIN audit_events a ON a.id=s.event_id ORDER BY a.occurred_at DESC LIMIT 1").Scan(&filters))
	if !strings.Contains(filters, `"format":"txt"`) {
		t.Fatal("format missing in audit", filters)
	}

	managed := managedLibrary(t, service, admin, "managed")
	contents, err := os.ReadFile("../../testdata/documents/sample.txt")
	requireNoError(t, err)
	uploadFormat(t, service, admin, managed, "Privado.txt", contents)
	reader := member(t, service, admin, managed, "format-reader-five", "library_reader")
	hidden, err := service.Search(ctx, reader, SearchInput{Libraries: []string{managed}, Filters: Filters{Format: "txt"}}, domain.RequestMetadata{}, false)
	requireNoError(t, err)
	if hidden.Count != 0 || len(hidden.Items) != 0 {
		t.Fatal("format filtering leaked private upload")
	}
}
func queryRoot(t *testing.T, service *Service, library string) string {
	t.Helper()
	var root string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT id FROM storage_roots WHERE library_id=?", library).Scan(&root))
	return root
}
