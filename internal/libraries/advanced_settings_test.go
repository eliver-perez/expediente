//go:build development

package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gestor-documental/internal/domain"
)

func TestAdvancedInheritanceValidationAndPersistence(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "Reglas")
	global, err := s.AdvancedConfiguration(ctx, p, "")
	requireNoError(t, err)
	if !global.Effective.OCREnabled || global.Effective.OCRMinimumCharacters != 32 {
		t.Fatal(global)
	}
	global.Global.OCRLanguages = "spa+eng"
	global.Global.MaximumPages = 25
	requireNoError(t, s.ConfigureAdvanced(ctx, p, "", global, domain.RequestMetadata{}))
	config, err := s.AdvancedConfiguration(ctx, p, library)
	requireNoError(t, err)
	if config.Effective.MaximumPages != 25 || config.Effective.OCRLanguages != "spa+eng" {
		t.Fatal(config)
	}
	config.Overrides["maximum_pages"] = json.RawMessage(`10`)
	config.Overrides["ocr_enabled"] = json.RawMessage(`false`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, library, config, domain.RequestMetadata{}))
	if err = s.ConfigureAdvanced(ctx, p, library, config, domain.RequestMetadata{}); failureCode(err) != "REVISION_CONFLICT" {
		t.Fatal(err)
	}
	restarted := New(s.Identity)
	config, err = restarted.AdvancedConfiguration(ctx, p, library)
	requireNoError(t, err)
	if config.Effective.MaximumPages != 10 || config.Effective.OCREnabled {
		t.Fatal(config)
	}
	config.Overrides["maximum_pages"] = json.RawMessage(`null`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, library, config, domain.RequestMetadata{}))
	config, err = s.AdvancedConfiguration(ctx, p, library)
	requireNoError(t, err)
	if config.Effective.MaximumPages != 25 {
		t.Fatal(config)
	}
	for _, invalid := range []string{`0`, `10001`, `"10"`} {
		config.Overrides["maximum_pages"] = json.RawMessage(invalid)
		if s.ConfigureAdvanced(ctx, p, library, config, domain.RequestMetadata{}) == nil {
			t.Fatal("invalid accepted", invalid)
		}
	}
	config.Overrides = map[string]json.RawMessage{"unknown": json.RawMessage(`true`)}
	if s.ConfigureAdvanced(ctx, p, library, config, domain.RequestMetadata{}) == nil {
		t.Fatal("unknown accepted")
	}
	if _, err = s.AdvancedConfiguration(ctx, domain.Principal{}, ""); err == nil {
		t.Fatal("anonymous read")
	}
	if err = s.ConfigureAdvanced(ctx, domain.Principal{}, "", global, domain.RequestMetadata{}); err == nil {
		t.Fatal("anonymous write")
	}
}
func TestAdvancedQueueAutomaticAndManualPriority(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	lib := managedLibrary(t, s, p, "managed")
	allowContentFormats(t, s, p)
	first := uploadFormat(t, s, p, lib, "primero.txt", []byte("Documento primero"))
	second := uploadFormat(t, s, p, lib, "manual.txt", []byte("Documento urgente"))
	config, err := s.AdvancedConfiguration(ctx, p, lib)
	requireNoError(t, err)
	config.Overrides["automatic_processing"] = json.RawMessage(`false`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, lib, config, domain.RequestMetadata{}))
	if _, err = s.claim(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("automatic claim", err)
	}
	id, err := s.ReindexDocument(ctx, p, second.DocumentID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	job, err := s.claim(ctx)
	requireNoError(t, err)
	if job.ID != id || !job.Manual {
		t.Fatal(job)
	}
	requireNoError(t, s.Extract(ctx, job, s.Identity.Config.Indexing))
	requireNoError(t, s.finish(ctx, job, nil))
	if _, err = s.claim(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("automatic escaped", err)
	}
	config, err = s.AdvancedConfiguration(ctx, p, lib)
	requireNoError(t, err)
	config.Overrides["automatic_processing"] = json.RawMessage(`null`)
	config.Overrides["reprocess_changes"] = json.RawMessage(`false`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, lib, config, domain.RequestMetadata{}))
	job, err = s.claim(ctx)
	requireNoError(t, err)
	requireNoError(t, s.Extract(ctx, job, s.Identity.Config.Indexing))
	requireNoError(t, s.finish(ctx, job, nil))
	document, err := s.Document(ctx, p, first.DocumentID)
	requireNoError(t, err)
	_, err = s.Database.Writer.Exec(`INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) SELECT ?,library_id,id,'extract',current_content_version_id,?,'{}','queued',?,? FROM physical_files WHERE id=?`, domain.NewID(), domain.NewID(), now(), now(), document.FileID)
	requireNoError(t, err)
	if _, err = s.claim(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("reprocessed automatically", err)
	}
	_, err = s.ReindexDocument(ctx, p, first.DocumentID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	job, err = s.claim(ctx)
	requireNoError(t, err)
	if !job.Manual {
		t.Fatal("manual reprocessing lost")
	}
}
func TestAdvancedWatcherIgnoresWithoutMarkingDocumentsMissing(t *testing.T) {
	s, p, directory := fixture(t)
	ctx := context.Background()
	lib := addLibrary(t, s, p, "Watcher")
	rootPath := filepath.Join(directory, "origen")
	copyFixture(t, filepath.Join(rootPath, ".oculta", "documento.pdf"), "native.pdf")
	root := addRoot(t, s, p, lib, rootPath)
	scan(t, s, root)
	copyFixture(t, filepath.Join(rootPath, "~$documento.pdf"), "native.pdf")
	copyFixture(t, filepath.Join(rootPath, "temporal.tmp"), "native.pdf")
	config, err := s.AdvancedConfiguration(ctx, p, lib)
	requireNoError(t, err)
	config.Overrides["ignore_hidden"] = json.RawMessage(`true`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, lib, config, domain.RequestMetadata{}))
	scan(t, s, root)
	var files, missing, skips, failures int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*),sum(availability='missing') FROM physical_files WHERE library_id=?", lib).Scan(&files, &missing))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM document_file_skips WHERE root_id=?", root).Scan(&skips))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM scan_errors WHERE root_id=?", root).Scan(&failures))
	if files != 1 || missing != 0 || skips != 3 || failures != 0 {
		t.Fatal(files, missing, skips, failures)
	}
	if _, err = os.Stat(filepath.Join(rootPath, "~$documento.pdf")); err != nil {
		t.Fatal("original removed", err)
	}
	requireNoError(t, s.Verify(ctx, p, lib, root, domain.RequestMetadata{}))
	job, err := s.claimLane(ctx, "scan")
	requireNoError(t, err)
	if !job.Manual {
		t.Fatal("verification not manual")
	}
}
func TestAdvancedPreviewLibraryOverride(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	lib := managedLibrary(t, s, p, "managed")
	content, err := os.ReadFile("../../testdata/documents/sample.docx")
	requireNoError(t, err)
	item := uploadFormat(t, s, p, lib, "sample.docx", content)
	config, err := s.AdvancedConfiguration(ctx, p, lib)
	requireNoError(t, err)
	config.Overrides["preview_enabled"] = json.RawMessage(`false`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, lib, config, domain.RequestMetadata{}))
	preview, err := s.Preview(ctx, p, item.DocumentID)
	requireNoError(t, err)
	if preview.Error != "PREVIEW_DISABLED" {
		t.Fatal(preview)
	}
	if _, err = s.RequestPreview(ctx, p, item.DocumentID, false, domain.RequestMetadata{}); failureCode(err) != "PREVIEW_DISABLED" {
		t.Fatal(err)
	}
	config, err = s.AdvancedConfiguration(ctx, p, lib)
	requireNoError(t, err)
	config.Overrides["preview_enabled"] = json.RawMessage(`null`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, lib, config, domain.RequestMetadata{}))
	preview, err = s.Preview(ctx, p, item.DocumentID)
	requireNoError(t, err)
	if preview.Error != "" {
		t.Fatal(preview)
	}
}

func TestAdvancedQueueDoesNotStarveAnotherLibrary(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	blocked := managedLibrary(t, s, p, "managed")
	allowContentFormats(t, s, p)
	item := uploadFormat(t, s, p, blocked, "bloqueado.txt", []byte("Pendiente por configuración"))
	document, err := s.Document(ctx, p, item.DocumentID)
	requireNoError(t, err)
	// More disabled candidates than the old claim window.
	for n := 0; n < 70; n++ {
		_, err = s.Database.Writer.Exec(`INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) SELECT ?,library_id,id,'extract',current_content_version_id,?,'{}','queued',?,? FROM physical_files WHERE id=?`, domain.NewID(), domain.NewID(), now(), now(), document.FileID)
		requireNoError(t, err)
	}
	rules, err := s.AdvancedConfiguration(ctx, p, blocked)
	requireNoError(t, err)
	rules.Overrides["automatic_processing"] = json.RawMessage(`false`)
	requireNoError(t, s.ConfigureAdvanced(ctx, p, blocked, rules, domain.RequestMetadata{}))
	active, err := s.Create(ctx, p, "Biblioteca activa", "managed", p.User.ID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	earlier := uploadFormat(t, s, p, active, "normal.txt", []byte("Trabajo normal"))
	urgent := uploadFormat(t, s, p, active, "urgente.txt", []byte("Trabajo prioritario"))
	manual, err := s.ReindexDocument(ctx, p, urgent.DocumentID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	job, err := s.claim(ctx)
	requireNoError(t, err)
	if job.ID != manual || !job.Manual || job.LibraryID != active {
		t.Fatal(job)
	}
	requireNoError(t, s.Extract(ctx, job, s.Identity.Config.Indexing))
	requireNoError(t, s.finish(ctx, job, nil))
	job, err = s.claim(ctx)
	requireNoError(t, err)
	next, err := s.Document(ctx, p, earlier.DocumentID)
	requireNoError(t, err)
	if job.FileID != next.FileID || job.Manual {
		t.Fatal(job)
	}
}
