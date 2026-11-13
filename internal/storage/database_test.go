package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestPostgreSQLSearchAndReadOnlyPools(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version int
	if err = database.Reader.QueryRow("SHOW server_version_num").Scan(&version); err != nil || version < 180000 {
		t.Fatal(version, err)
	}
	for _, pool := range []*sql.DB{database.Reader, database.ControlReader, database.StatusReader, database.AnalyticsReader} {
		if _, err = pool.Exec("INSERT INTO roles VALUES('forbidden','forbidden','global')"); err == nil {
			t.Fatal("read pool accepted a write")
		}
	}
	for _, entry := range []struct {
		text, query string
		matches     bool
	}{
		{"estructura metálica", "metalica", true}, {"a de la casa", "de la", true},
		{"gato gatos", "gatos", true}, {"gato", "gatos", false}, {"uno dos tres", "uno tres", false},
		{"uno-dos tres", "uno dos", true}, {"niño café", "nino cafe", true}, {"договор 東京 العربية", "договор 東京", true}, {"foo", "-", false},
	} {
		var matches bool
		if err = database.Reader.QueryRow("SELECT aidex_search_vector(?) @@ aidex_search_query(ARRAY[?]::text[])", entry.text, entry.query).Scan(&matches); err != nil || matches != entry.matches {
			t.Fatalf("%q / %q: %v %v", entry.text, entry.query, matches, err)
		}
	}
}

func TestMigrationChecksumAndStateLock(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := LockState(directory)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate, err := LockState(directory); err == nil {
		duplicate()
		t.Fatal("second lock accepted")
	}
	unlock()
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Writer.Exec("UPDATE schema_migrations SET checksum='tampered'"); err != nil {
		t.Fatal(err)
	}
	database.Close()
	if reopened, err := Open(context.Background(), directory); err == nil {
		reopened.Close()
		t.Fatal("changed migration accepted")
	}
}

func TestAuditAndFilesystemProtections(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Writer.Exec("INSERT INTO audit_events(id,occurred_at,actor_kind,event_type,details_json) VALUES ('fixture','2026-09-23T00:00:00.000000Z','system','fixture','{}')"); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"DELETE FROM audit_events", "UPDATE audit_events SET event_type='tampered'"} {
		if _, err := database.Writer.Exec(statement); err == nil {
			t.Fatal("audit mutation accepted")
		}
	}
	alias := filepath.Join(directory, "alias")
	if err := os.Symlink(directory, alias); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if err := PreparePrivateDirectory(alias); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestEmptyMigrationRollbackAndRefusalWithEvidence(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.RollbackEmpty(context.Background()); err == nil {
		t.Fatal("rollback accepted for a schema newer than the initial H2 schema")
	}
	database.Close()
	database, err = Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Writer.Exec("INSERT INTO audit_events(id,occurred_at,actor_kind,event_type,details_json) VALUES ('evidence','2026-09-23T00:00:00.000000Z','system','fixture','{}')"); err != nil {
		t.Fatal(err)
	}
	if err := database.RollbackEmpty(context.Background()); err == nil {
		t.Fatal("rollback destroyed audit evidence")
	}
	var count int
	if err := database.Reader.QueryRow("SELECT count(*) FROM audit_events").Scan(&count); err != nil || count != 1 {
		t.Fatal("rollback did not preserve database", err)
	}
}
