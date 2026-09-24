package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"gestor-documental/db"
	"os"
	"path/filepath"
	"testing"

	"modernc.org/sqlite"
)

func TestEngineWALFTSForeignKeysAndOnlineBackup(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	database, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var version, journal string
	var foreignKeys int
	if err := database.Reader.QueryRow("SELECT sqlite_version()").Scan(&version); err != nil {
		t.Fatal(err)
	}
	t.Log("embedded SQLite", version)
	if err := database.Writer.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
		t.Fatal(journal, err)
	}
	if err := database.Reader.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatal(foreignKeys, err)
	}
	if _, err := database.Writer.Exec("CREATE VIRTUAL TABLE fixture_fts USING fts5(body, tokenize='unicode61 remove_diacritics 2'); INSERT INTO fixture_fts VALUES ('estructura metálica');"); err != nil {
		t.Fatal(err)
	}
	var matches int
	if err := database.Reader.QueryRow("SELECT count(*) FROM fixture_fts WHERE fixture_fts MATCH ?", "metalica").Scan(&matches); err != nil || matches != 1 {
		t.Fatal(matches, err)
	}
	connection, err := database.Writer.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(directory, "backup.sqlite")
	err = connection.Raw(func(driverConnection any) error {
		backup, err := driverConnection.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		}).NewBackup(backupPath)
		if err != nil {
			return err
		}
		defer backup.Finish()
		_, err = backup.Step(-1)
		return err
	})
	connection.Close()
	if err != nil {
		t.Fatal(err)
	}
	copyDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	if err := copyDB.QueryRow("SELECT count(*) FROM fixture_fts WHERE fixture_fts MATCH 'metalica'").Scan(&matches); err != nil || matches != 1 {
		t.Fatal("backup lost committed WAL content", err)
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
	if err := database.RollbackEmpty(context.Background()); err != nil {
		t.Fatal(err)
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

func TestUpgradePreservesPopulatedH2AndRefusesDestructiveRollback(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	connection, err := sql.Open("sqlite", filepath.Join(directory, "documental.db"))
	if err != nil {
		t.Fatal(err)
	}
	migration, err := db.Migrations.ReadFile("migrations/0001_identity.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec("CREATE TABLE schema_migrations(name TEXT PRIMARY KEY,checksum TEXT NOT NULL,applied_at TEXT NOT NULL) STRICT"); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec("INSERT INTO schema_migrations VALUES('0001_identity.up.sql',?,'2026-09-23T00:00:00.000000Z')", fmt.Sprintf("%x", sha256.Sum256(migration))); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec("INSERT INTO users(id,username,username_key,display_name,password_hash,created_at,updated_at) VALUES('old-user','old-user','old-user','Existing H2','unchanged-phc','2026-09-23T00:00:00.000000Z','2026-09-23T00:00:00.000000Z'); INSERT INTO audit_events(id,occurred_at,actor_kind,event_type,details_json) VALUES('old-audit','2026-09-23T00:00:00.000000Z','system','fixture','{}')"); err != nil {
		t.Fatal(err)
	}
	connection.Close()
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var password string
	if err = database.Reader.QueryRow("SELECT password_hash FROM users WHERE id='old-user'").Scan(&password); err != nil || password != "unchanged-phc" {
		t.Fatal("upgrade changed user", err)
	}
	if err = database.RollbackEmpty(context.Background()); err == nil {
		t.Fatal("rollback destroyed populated H2")
	}
	var migrations, evidence int
	if err = database.Reader.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&migrations); err != nil || migrations != 2 {
		t.Fatal("partial down migration", err)
	}
	if err = database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE id='old-audit'").Scan(&evidence); err != nil || evidence != 1 {
		t.Fatal("upgrade lost audit", err)
	}
}
