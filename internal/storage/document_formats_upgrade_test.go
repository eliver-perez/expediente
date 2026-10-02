package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"gestor-documental/db"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentFormatsUpgradeFromVersionEight(t *testing.T) {
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
	entries, err := db.Migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") || name >= "0009" {
			continue
		}
		migration, err := db.Migrations.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = connection.Exec(string(migration)); err != nil {
			t.Fatal(name, err)
		}
		if _, err = connection.Exec("INSERT INTO schema_migrations VALUES(?,?,'2026-10-01')", name, fmt.Sprintf("%x", sha256.Sum256(migration))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = connection.Exec(`
 INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('library','Existing r7','hybrid','2026-09-24','2026-09-24');
 INSERT INTO cases(id,library_id,identifier,identifier_key,created_at) VALUES('case','library','PR-008','pr-008','2026-09-24');
 INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,current_content_version_id,indexed_extraction_id,extraction_freshness,created_at) VALUES('file','library','linked','missing','verified','version','extraction','current','2026-09-24');
 INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,sha256,os_modified_at,observed_change_key,observed_at,change_origin) VALUES('version','file',1,123,'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-09-24','observation','2026-09-24','scan');
 INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count) VALUES('extraction','file','version','poppler-tesseract-v2','spa','complete',1);
 INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text) VALUES('extraction',1,'ocr','Evidencia original indexada');
 INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) VALUES('file','extraction',1,'Evidencia original indexada');
 INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at,case_id,revision) VALUES('document','library','file','Conservado.pdf','conservado.pdf','2026-09-24','case',12);
 INSERT INTO processing_settings VALUES(1,'manual',2,1,2,4,'2026-09-24');
 `); err != nil {
		t.Fatal(err)
	}
	connection.Close()
	database, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	var unchanged int
	err = database.Reader.QueryRow(`SELECT count(*) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN content_versions v ON v.id=f.current_content_version_id WHERE d.id='document' AND d.case_id='case' AND d.revision=12 AND f.indexed_extraction_id='extraction' AND f.extraction_freshness='current' AND v.sha256='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' AND v.document_format='pdf' AND v.detected_mime='application/pdf' AND v.extension_mismatch=0 AND v.index_block_reason=''`).Scan(&unchanged)
	if err != nil || unchanged != 1 {
		t.Fatal("upgrade changed PDF identity/hash/association/index", err)
	}
	var text string
	if err = database.Reader.QueryRow("SELECT page_text FROM pages_fts WHERE pages_fts MATCH 'evidencia'").Scan(&text); err != nil || text != "Evidencia original indexada" {
		t.Fatal("lost original FTS", err)
	}
	var revision int
	if err = database.Reader.QueryRow("SELECT revision FROM processing_settings").Scan(&revision); err != nil || revision != 4 {
		t.Fatal("lost processing configuration", err)
	}
	if err = database.Reader.QueryRow("PRAGMA integrity_check").Scan(&text); err != nil || text != "ok" {
		t.Fatal("integrity", err, text)
	}
	database.Close()
	backups, err := filepath.Glob(filepath.Join(directory, "upgrade-backups", "before-processing-*.db"))
	if err != nil || len(backups) != 1 {
		t.Fatal("missing pre-v2 backup", err, backups)
	}
	backupURL := sqliteFileURL(filepath.ToSlash(backups[0]))
	backupURL.RawQuery = "mode=ro"
	backup, err := sql.Open("sqlite", backupURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err = backup.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&revision); err != nil || revision != 8 {
		t.Fatal("backup does not contain old schema", err, revision)
	}
	database, err = Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	database.Close()
	backups, err = filepath.Glob(filepath.Join(directory, "upgrade-backups", "before-processing-*.db"))
	if err != nil || len(backups) != 1 {
		t.Fatal("backup repeated unnecessarily", err)
	}
}
