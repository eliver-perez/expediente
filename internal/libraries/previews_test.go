//go:build development

package libraries

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/previews"
)

func previewFixture(t *testing.T) (*Service, domain.Principal, string, string) {
	t.Helper()
	service, admin, _ := fixture(t)
	library := managedLibrary(t, service, admin, "managed")
	allowContentFormats(t, service, admin)
	contents, err := os.ReadFile("../../testdata/documents/sample.docx")
	requireNoError(t, err)
	item := uploadFormat(t, service, admin, library, "Documento.docx", contents)
	service.previewConvert = func(ctx context.Context, source, format, output, program string) error {
		return os.WriteFile(output, []byte("%PDF-1.7\nsynthetic preview"), 0600)
	}
	return service, admin, library, item.DocumentID
}
func generateRequestedPreview(t *testing.T, service *Service) {
	t.Helper()
	ctx := context.Background()
	job, settings, err := service.claimPreview(ctx)
	requireNoError(t, err)
	if job.ID == "" {
		t.Fatal("preview missing")
	}
	requireNoError(t, service.generatePreview(ctx, job, settings))
}
func TestPreviewOnDemandReuseClearAndPrivateAccess(t *testing.T) {
	service, admin, library, id := previewFixture(t)
	ctx := context.Background()
	drainDocumentContent(t, service)
	before, err := service.Document(ctx, admin, id)
	requireNoError(t, err)
	state, err := service.Preview(ctx, admin, id)
	requireNoError(t, err)
	if state.State != "not_generated" {
		t.Fatal(state)
	}
	_, err = service.RequestPreview(ctx, domain.Principal{}, id, false, domain.RequestMetadata{})
	if err == nil {
		t.Fatal("anonymous preview")
	}
	first, err := service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	second, err := service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	if first.ID != second.ID {
		t.Fatal("duplicate generation")
	}
	generateRequestedPreview(t, service)
	ready, err := service.Preview(ctx, admin, id)
	requireNoError(t, err)
	if ready.State != "ready" || ready.URL == "" {
		t.Fatal(ready)
	}
	file, media, err := service.OpenPreview(ctx, admin, id, ready.ID, domain.RequestMetadata{})
	requireNoError(t, err)
	file.Close()
	if media != "application/pdf" {
		t.Fatal(media)
	}
	reader := member(t, service, admin, library, "preview-reader", "library_reader")
	if _, _, err = service.OpenPreview(ctx, reader, id, ready.ID, domain.RequestMetadata{}); err == nil {
		t.Fatal("reader saw another user's private upload preview")
	}
	if err = service.ClearPreviews(ctx, reader, domain.RequestMetadata{}); err == nil {
		t.Fatal("reader cleared global cache")
	}
	if _, _, err = service.OpenPreview(ctx, admin, "unrelated-document", ready.ID, domain.RequestMetadata{}); err == nil {
		t.Fatal("preview ID bypassed document scope")
	}
	requireNoError(t, service.ClearPreviews(ctx, admin, domain.RequestMetadata{}))
	if _, err = os.Stat(service.previewPath(ready.ID)); !os.IsNotExist(err) {
		t.Fatal("cache survived", err)
	}
	after, err := service.Document(ctx, admin, id)
	requireNoError(t, err)
	if after.Hash != before.Hash || after.FileID != before.FileID || after.Processing.ID != before.Processing.ID {
		t.Fatal("clear changed original/index")
	}
	if query(t, service, admin, library, "sintético").Count == 0 {
		t.Fatal("clear changed searchable text")
	}
	if _, _, err = service.OpenPreview(ctx, admin, id, ready.ID, domain.RequestMetadata{}); err == nil {
		t.Fatal("cleared resource served")
	}
	_, err = service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	// Revoking library access also blocks a representation which is already cached.
	_, err = service.Database.Writer.Exec("DELETE FROM library_role_assignments WHERE user_id=? AND library_id=?", admin.User.ID, library)
	requireNoError(t, err)
	if _, err = service.Preview(ctx, admin, id); err == nil {
		t.Fatal("cached preview bypassed authorization")
	}
}
func TestPreviewLRUExpiryAndClearDuringGeneration(t *testing.T) {
	service, admin, library, firstID := previewFixture(t)
	ctx := context.Background()
	service.previewConvert = func(ctx context.Context, source, format, output, program string) error {
		data := make([]byte, 600<<10)
		copy(data, "%PDF-")
		return os.WriteFile(output, data, 0600)
	}
	contents, err := os.ReadFile("../../testdata/documents/sample.xlsx")
	requireNoError(t, err)
	second := uploadFormat(t, service, admin, library, "Segundo.xlsx", contents)
	settings, err := service.PreviewSettings(ctx)
	requireNoError(t, err)
	settings.MaximumCacheMB = 1
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	first, err := service.RequestPreview(ctx, admin, firstID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	_, err = service.RequestPreview(ctx, admin, second.DocumentID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	if _, err = os.Stat(service.previewPath(first.ID)); !os.IsNotExist(err) {
		t.Fatal("LRU not evicted")
	}
	settings, err = service.PreviewSettings(ctx)
	requireNoError(t, err)
	if settings.Entries != 1 || settings.UsedBytes != 600<<10 {
		t.Fatal(settings)
	}
	_, err = service.Database.Writer.Exec("UPDATE preview_cache SET last_accessed_at=?", domain.Timestamp(time.Now().Add(-40*24*time.Hour)))
	requireNoError(t, err)
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	settings, err = service.PreviewSettings(ctx)
	requireNoError(t, err)
	if settings.Entries != 0 {
		t.Fatal("age cleanup failed", settings)
	}
	_, err = service.RequestPreview(ctx, admin, firstID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	job, settings, err := service.claimPreview(ctx)
	requireNoError(t, err)
	service.previewConvert = func(ctx context.Context, source, format, output, program string) error {
		requireNoError(t, service.ClearPreviews(ctx, admin, domain.RequestMetadata{}))
		return os.WriteFile(output, []byte("%PDF-test"), 0600)
	}
	if err = service.generatePreview(ctx, job, settings); failureCode(err) != "PREVIEW_OBSOLETE" {
		t.Fatal("in-flight publication after clear", err)
	}
}
func TestPreviewInvalidationLimitsAndPolicy(t *testing.T) {
	service, admin, _, id := previewFixture(t)
	ctx := context.Background()
	first, err := service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	// A converter upgrade invalidates representations even when the original is unchanged.
	settings, err := service.PreviewSettings(ctx)
	requireNoError(t, err)
	settings.ConverterPath = filepath.Join(t.TempDir(), "different-converter")
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	state, err := service.Preview(ctx, admin, id)
	requireNoError(t, err)
	if state.State != "obsolete" {
		t.Fatal(state)
	}
	if _, _, err = service.OpenPreview(ctx, admin, id, first.ID, domain.RequestMetadata{}); err == nil {
		t.Fatal("old generator accepted")
	}
	settings, err = service.PreviewSettings(ctx)
	requireNoError(t, err)
	settings.Enabled = false
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	_, err = service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	if failureCode(err) != "PREVIEW_DISABLED" {
		t.Fatal(err)
	}
	settings, err = service.PreviewSettings(ctx)
	requireNoError(t, err)
	settings.Enabled = true
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	_, err = service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	policy, err := service.ProcessingPolicy(ctx)
	requireNoError(t, err)
	policy.Paused = true
	requireNoError(t, service.ConfigureProcessingPolicy(ctx, admin, policy, domain.RequestMetadata{}))
	job, _, err := service.claimPreview(ctx)
	requireNoError(t, err)
	if job.ID != "" {
		t.Fatal("preview ignored global pause")
	}
	// Source version/hash changes cannot reuse either existing generation.
	requireNoError(t, service.Database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE content_versions SET sha256=? WHERE id=(SELECT f.current_content_version_id FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE d.id=?)", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", id)
		return err
	}))
	state, err = service.Preview(ctx, admin, id)
	requireNoError(t, err)
	if state.State != "obsolete" {
		t.Fatal(state)
	}
	if _, _, err = service.OpenPreview(ctx, admin, id, first.ID, domain.RequestMetadata{}); err == nil {
		t.Fatal("stale hash served")
	}
}
func TestPreviewFailureDoesNotPoisonIndex(t *testing.T) {
	service, admin, library, id := previewFixture(t)
	ctx := context.Background()
	service.previewConvert = func(context.Context, string, string, string, string) error {
		return previews.Failure("PREVIEW_TIMEOUT")
	}
	_, err := service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	runtime, err := service.Start(ctx)
	requireNoError(t, err)
	defer runtime.Close()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state, err := service.Preview(ctx, admin, id)
		requireNoError(t, err)
		document, err := service.Document(ctx, admin, id)
		requireNoError(t, err)
		if state.State == "error" && document.Freshness == "current" {
			if state.Error != "PREVIEW_TIMEOUT" || query(t, service, admin, library, "sintético").Count == 0 {
				t.Fatal(state)
			}
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("preview failure blocked indexing or was not reported")
}

func TestPreviewLRUTouchesAndMissingFileRecovery(t *testing.T) {
	service, admin, library, firstID := previewFixture(t)
	ctx := context.Background()
	service.previewConvert = func(ctx context.Context, source, format, output, program string) error {
		data := make([]byte, 400<<10)
		copy(data, "%PDF-")
		return os.WriteFile(output, data, 0600)
	}
	settings, err := service.PreviewSettings(ctx)
	requireNoError(t, err)
	settings.MaximumCacheMB = 1
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	contents, err := os.ReadFile("../../testdata/documents/sample.docx")
	requireNoError(t, err)
	second := uploadFormat(t, service, admin, library, "Segundo.docx", contents)
	third := uploadFormat(t, service, admin, library, "Tercero.docx", contents)
	first, err := service.RequestPreview(ctx, admin, firstID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	secondPreview, err := service.RequestPreview(ctx, admin, second.DocumentID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	_, err = service.Database.Writer.Exec("UPDATE preview_cache SET last_accessed_at=?", domain.Timestamp(time.Now().Add(-time.Hour)))
	requireNoError(t, err)
	file, _, err := service.OpenPreview(ctx, admin, firstID, first.ID, domain.RequestMetadata{})
	requireNoError(t, err)
	file.Close()
	_, err = service.RequestPreview(ctx, admin, third.DocumentID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	generateRequestedPreview(t, service)
	if _, err = os.Stat(service.previewPath(first.ID)); err != nil {
		t.Fatal("recent access was evicted", err)
	}
	if _, err = os.Stat(service.previewPath(secondPreview.ID)); !os.IsNotExist(err) {
		t.Fatal("least recently used entry survived")
	}
	requireNoError(t, os.Remove(service.previewPath(first.ID)))
	requireNoError(t, service.recoverPreviews(ctx))
	settings, err = service.PreviewSettings(ctx)
	requireNoError(t, err)
	if settings.Entries != 1 || settings.UsedBytes != 400<<10 {
		t.Fatal("missing file still counted", settings)
	}
}

func TestPreviewSourceAndOutputLimits(t *testing.T) {
	service, admin, _, id := previewFixture(t)
	ctx := context.Background()
	settings, err := service.PreviewSettings(ctx)
	requireNoError(t, err)
	settings.MaximumSourceMB = 1
	requireNoError(t, service.ConfigurePreviews(ctx, admin, settings, domain.RequestMetadata{}))
	_, err = service.Database.Writer.Exec("UPDATE content_versions SET size_bytes=2097152 WHERE id=(SELECT f.current_content_version_id FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE d.id=?)", id)
	requireNoError(t, err)
	_, err = service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	if failureCode(err) != "PREVIEW_SOURCE_LIMIT" {
		t.Fatal("source limit not applied", err)
	}
	_, err = service.Database.Writer.Exec("UPDATE content_versions SET size_bytes=100 WHERE id=(SELECT f.current_content_version_id FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE d.id=?)", id)
	requireNoError(t, err)
	_, err = service.RequestPreview(ctx, admin, id, false, domain.RequestMetadata{})
	requireNoError(t, err)
	job, settings, err := service.claimPreview(ctx)
	requireNoError(t, err)
	service.previewConvert = func(ctx context.Context, source, format, output, program string) error {
		file, err := os.Create(output)
		if err != nil {
			return err
		}
		defer file.Close()
		return file.Truncate(previews.MaximumOutput + 1)
	}
	if err = service.generatePreview(ctx, job, settings); failureCode(err) != "PREVIEW_SIZE_LIMIT" {
		t.Fatal("output limit not applied", err)
	}
	settings, err = service.PreviewSettings(ctx)
	requireNoError(t, err)
	if settings.Entries != 0 || settings.UsedBytes != 0 {
		t.Fatal("oversize preview published", settings)
	}
}
