//go:build development

package libraries

import (
	"context"
	"database/sql"
	"testing"

	"gestor-documental/internal/storage"
	_ "modernc.org/sqlite"
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

func TestDeletionFTSRebuildPreservesOtherDocumentsAndTriggers(t *testing.T) {
	ctx := context.Background()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	for _, statement := range []string{
		"CREATE TABLE indexed_pages(id INTEGER PRIMARY KEY, page_text TEXT NOT NULL)",
		"CREATE VIRTUAL TABLE pages_fts USING fts5(page_text,content='indexed_pages',content_rowid='id')",
		"CREATE TRIGGER indexed_pages_insert AFTER INSERT ON indexed_pages BEGIN INSERT INTO pages_fts(rowid,page_text) VALUES(new.id,new.page_text); END",
		"CREATE TRIGGER indexed_pages_delete AFTER DELETE ON indexed_pages BEGIN INSERT INTO pages_fts(pages_fts,rowid,page_text) VALUES('delete',old.id,old.page_text); END",
		"INSERT INTO indexed_pages VALUES(1,'eliminar')",
		"INSERT INTO indexed_pages VALUES(2,'conservar')",
		"INSERT INTO indexed_pages VALUES(3,'eliminar')",
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var triggerSQL string
	if err := tx.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='trigger' AND name='indexed_pages_delete'").Scan(&triggerSQL); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"DROP TRIGGER indexed_pages_delete",
		"DELETE FROM indexed_pages WHERE id IN (1,3)",
		"INSERT INTO pages_fts(pages_fts) VALUES('rebuild')",
		triggerSQL,
		"INSERT INTO indexed_pages VALUES(4,'nuevo')",
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for term, want := range map[string]int{"eliminar": 0, "conservar": 1, "nuevo": 1} {
		var got int
		if err := database.QueryRowContext(ctx, "SELECT count(*) FROM pages_fts WHERE pages_fts MATCH ?", term).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s: got %d, want %d", term, got, want)
		}
	}
	if _, err := database.ExecContext(ctx, "INSERT INTO pages_fts(pages_fts,rank) VALUES('integrity-check',1)"); err != nil {
		t.Fatal(err)
	}
}
