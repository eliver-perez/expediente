//go:build development

package libraries

import (
	"context"
	"path/filepath"
	"testing"

	"gestor-documental/internal/storage"
)

func TestDeletionProgressSharedBetweenRuntimeAndHTTPServices(t *testing.T) {
	database := &storage.Database{}
	runtime := &Service{Database: database}
	httpService := &Service{Database: database}
	runtime.setDeletionProgress("operation", DeletionProgress{Phase: "removing_index", Done: 12, Total: 20, Unit: "pages"})
	t.Cleanup(func() { runtime.clearDeletionProgress("operation") })
	progress, ok := httpService.getDeletionProgress("operation")
	if !ok || progress.Phase != "removing_index" || progress.Done != 12 || progress.Total != 20 {
		t.Fatalf("HTTP service could not see runtime progress: %+v, %v", progress, ok)
	}
}

func TestDeletionGINPreservesOtherDocuments(t *testing.T) {
	ctx := context.Background()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := storage.Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for _, statement := range []string{
		"CREATE TABLE search_projection_fixture(id bigint PRIMARY KEY,page_text text NOT NULL,search_vector tsvector GENERATED ALWAYS AS (aidex_search_vector(page_text)) STORED)",
		"CREATE INDEX search_projection_gin ON search_projection_fixture USING gin(search_vector)",
		"INSERT INTO search_projection_fixture(id,page_text) VALUES(1,'eliminar'),(2,'conservar'),(3,'eliminar')",
		"DELETE FROM search_projection_fixture WHERE id IN (1,3)",
		"INSERT INTO search_projection_fixture(id,page_text) VALUES(4,'nuevo')",
	} {
		if _, err = database.Writer.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	for term, want := range map[string]int{"eliminar": 0, "conservar": 1, "nuevo": 1} {
		var got int
		if err = database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM search_projection_fixture WHERE search_vector @@ aidex_search_query(ARRAY[?]::text[])", term).Scan(&got); err != nil || got != want {
			t.Fatal(term, got, want, err)
		}
	}
}
