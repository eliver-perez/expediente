//go:build development

package libraries

import (
	"context"
	"gestor-documental/internal/domain"
	"path/filepath"
	"testing"
)

func TestGroupedSearchTotalsScopeAndDirectChildren(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	first := addLibrary(t, service, principal, "First")
	second := addLibrary(t, service, principal, "Second")
	var root string
	for i, library := range []string{first, second} {
		path := filepath.Join(directory, library)
		copyFixture(t, filepath.Join(path, "Match.pdf"), "native.pdf")
		copyFixture(t, filepath.Join(path, "Child", "Match-child.pdf"), "native.pdf")
		if i == 0 {
			copyFixture(t, filepath.Join(path, "Child-other", "Match-other.pdf"), "native.pdf")
		}
		id := addRoot(t, service, principal, library, path)
		scan(t, service, id)
		if i == 0 {
			root = id
		}
	}
	summary, err := service.Search(ctx, principal, SearchInput{Query: "Match", Type: "name", SummaryOnly: true}, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if len(summary.Items) != 0 || summary.Count != 5 || len(summary.Groups) != 2 || summary.Groups[0].Count != 3 || summary.Groups[1].Count != 2 {
		t.Fatalf("bad group totals: %+v", summary)
	}
	input := SearchInput{Query: "Match", Type: "name", Libraries: []string{first}, Limit: 1}
	page, err := service.Search(ctx, principal, input, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if page.Count != 3 || page.Cursor == "" || len(page.Items) != 1 {
		t.Fatal("incorrect bounded page")
	}
	input.Libraries = []string{second}
	input.Cursor = page.Cursor
	if _, err = service.Search(ctx, principal, input, domain.RequestMetadata{}, true); err == nil {
		t.Fatal("cursor crossed library boundary")
	}
	result, err := service.Search(ctx, principal, SearchInput{Libraries: []string{first}, Filters: Filters{RootID: root, DirectChildren: true}}, domain.RequestMetadata{}, false)
	requireNoError(t, err)
	if result.Count != 1 || result.Items[0].Filename != "Match.pdf" {
		t.Fatal("file explorer included descendants")
	}
	result, err = service.Search(ctx, principal, SearchInput{Query: "Match", Type: "name", Libraries: []string{first}, Filters: Filters{RootID: root, Prefix: "Child", Availability: "available"}}, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if result.Count != 1 || result.Items[0].Filename != "Match-child.pdf" {
		t.Fatal("combined folder/name/availability filters failed")
	}
	_, err = service.Database.Writer.Exec("DELETE FROM library_role_assignments WHERE library_id=? AND user_id=?", second, principal.User.ID)
	requireNoError(t, err)
	summary, err = service.Search(ctx, principal, SearchInput{Query: "Match", SummaryOnly: true}, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if summary.Count != 3 || len(summary.Groups) != 1 {
		t.Fatal("group summary leaked inaccessible library")
	}
	// More concurrent searches than reader connections must not nest a license
	// read transaction inside another reader and exhaust the pool.
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, err := service.Search(ctx, principal, SearchInput{Query: "Match", Limit: 1}, domain.RequestMetadata{}, true)
			results <- err
		}()
	}
	for i := 0; i < 8; i++ {
		requireNoError(t, <-results)
	}
}
