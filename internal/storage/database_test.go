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
	if err = database.Reader.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&migrations); err != nil || migrations != 4 {
		t.Fatal("partial down migration", err)
	}
	if err = database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE id='old-audit'").Scan(&evidence); err != nil || evidence != 1 {
		t.Fatal("upgrade lost audit", err)
	}
}

func TestUpgradePreservesH3IdentityOCRAndReferences(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	connection, err := sql.Open("sqlite", filepath.Join(directory, "documental.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err = connection.Exec("CREATE TABLE schema_migrations(name TEXT PRIMARY KEY,checksum TEXT NOT NULL,applied_at TEXT NOT NULL) STRICT"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"0001_identity.up.sql", "0002_linked_libraries.up.sql"} {
		migration, err := db.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = connection.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
		if _, err = connection.Exec("INSERT INTO schema_migrations VALUES(?,?,'2026-09-24T00:00:00.000000Z')", name, fmt.Sprintf("%x", sha256.Sum256(migration))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = connection.Exec(`
INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('old-library','Existente H3','linked','2026-09-24','2026-09-24');
INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,current_content_version_id,indexed_extraction_id,extraction_freshness,created_at) VALUES('old-file','old-library','linked','missing','verified','old-version','old-extraction','current','2026-09-24');
INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,observed_change_key,observed_at,change_origin) VALUES('old-version','old-file',1,123,'2026-09-24','old-observation','2026-09-24','scan');
INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count) VALUES('old-extraction','old-file','old-version','h3','spa','complete',1);
INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text) VALUES('old-extraction',1,'ocr','conservación metálica H3');
INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) VALUES('old-file','old-extraction',1,'conservación metálica H3');
INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES('old-document','old-library','old-file','H3.pdf','h3.pdf','2026-09-24');
`); err != nil {
		t.Fatal(err)
	}
	connection.Close()
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var fileID, versionID, extractionID, availability, metadata string
	var unclassified bool
	if err = database.Reader.QueryRow("SELECT d.physical_file_id,f.current_content_version_id,f.indexed_extraction_id,f.availability,d.metadata_json,d.case_id IS NULL AND d.category_id IS NULL AND d.document_type_id IS NULL FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id='old-document'").Scan(&fileID, &versionID, &extractionID, &availability, &metadata, &unclassified); err != nil {
		t.Fatal(err)
	}
	if fileID != "old-file" || versionID != "old-version" || extractionID != "old-extraction" || availability != "missing" || metadata != "{}" || !unclassified {
		t.Fatal("H4 changed H3 identity/classification")
	}
	var text string
	if err = database.Reader.QueryRow("SELECT page_text FROM pages_fts WHERE pages_fts MATCH 'metalica'").Scan(&text); err != nil || text != "conservación metálica H3" {
		t.Fatal("H4 lost searchable retained OCR", err)
	}
	if err = database.RollbackEmpty(context.Background()); err == nil {
		t.Fatal("rollback destroyed H3 content")
	}
	var count int
	if err = database.Reader.QueryRow("SELECT count(*) FROM library_configuration_versions WHERE library_id='old-library'").Scan(&count); err != nil || count != 1 {
		t.Fatal("configuration snapshot missing", err)
	}
	rows, err := database.Reader.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("upgrade broke references")
	}
}

func TestUpgradePreservesH4PrivateClassifiedUpload(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	connection, err := sql.Open("sqlite", filepath.Join(directory, "documental.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err = connection.Exec("CREATE TABLE schema_migrations(name TEXT PRIMARY KEY,checksum TEXT NOT NULL,applied_at TEXT NOT NULL) STRICT"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"0001_identity.up.sql", "0002_linked_libraries.up.sql", "0003_managed_organization.up.sql"} {
		migration, err := db.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = connection.Exec(string(migration)); err != nil {
			t.Fatal(err)
		}
		if _, err = connection.Exec("INSERT INTO schema_migrations VALUES(?,?,'2026-09-24')", name, fmt.Sprintf("%x", sha256.Sum256(migration))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = connection.Exec(`
 INSERT INTO users(id,username,username_key,display_name,password_hash,created_at,updated_at) VALUES('owner','owner','owner','Owner','unchanged','2026-09-24','2026-09-24');
 INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('library','Existing H4','hybrid','2026-09-24','2026-09-24');
 INSERT INTO categories(id,library_id,name) VALUES('category','library','Planos');
 INSERT INTO document_types(id,library_id,category_id,name,allows_multiple,requires_descriptive_title) VALUES('type','library','category','Plano',1,0);
 INSERT INTO cases(id,library_id,identifier,identifier_key,created_at) VALUES('case','library','PR-008','pr-008','2026-09-24');
 INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,current_content_version_id,indexed_extraction_id,extraction_freshness,created_at) VALUES('file','library','managed','staged','verified','version','extraction','current','2026-09-24');
 INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,sha256,os_modified_at,observed_change_key,observed_at,change_origin) VALUES('version','file',1,123,'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-09-24','observation','2026-09-24','upload');
 INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count) VALUES('extraction','file','version','h4','spa','complete',1);
 INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text) VALUES('extraction',1,'ocr','evidencia privada H4');
 INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) VALUES('file','extraction',1,'evidencia privada H4');
 INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at,case_id,category_id,document_type_id,created_by,revision) VALUES('document','library','file','H4.pdf','h4.pdf','2026-09-24','case','category','type','owner',7);
 INSERT INTO upload_batches VALUES('batch','library','owner','client-batch','2026-09-24');
 INSERT INTO upload_items(id,batch_id,library_id,client_file_id,original_filename,document_id,private_temporary_locator,status,sha256,created_at) VALUES('upload','batch','library','client-file','H4.pdf','document','private.pdf','staged','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-09-24');
 `); err != nil {
		t.Fatal(err)
	}
	connection.Close()
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var unchanged int
	if err = database.Reader.QueryRow(`SELECT count(*) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN upload_items u ON u.document_id=d.id WHERE d.id='document' AND d.revision=7 AND d.case_id='case' AND d.category_id='category' AND d.document_type_id='type' AND d.created_by='owner' AND d.approval_status='draft' AND f.current_content_version_id='version' AND f.indexed_extraction_id='extraction' AND f.availability='staged' AND u.status='staged' AND u.retain_until IS NULL AND u.private_temporary_locator='private.pdf' AND u.sha256='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'`).Scan(&unchanged); err != nil || unchanged != 1 {
		t.Fatal("H5 changed H4 evidence", err)
	}
	var text string
	if err = database.Reader.QueryRow("SELECT page_text FROM pages_fts WHERE pages_fts MATCH 'privada'").Scan(&text); err != nil || text != "evidencia privada H4" {
		t.Fatal("lost retained text", err)
	}
	if err = database.RollbackEmpty(context.Background()); err == nil {
		t.Fatal("rollback destroyed H4 evidence")
	}
	rows, err := database.Reader.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("upgrade broke scope references")
	}
}
