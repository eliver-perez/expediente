package libraries

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/previews"
	"gestor-documental/internal/storage"
)

type previewJob struct{ ID, DocumentID, Version, Hash, Generator, RequestedBy string }

// One independent worker bounds converter concurrency. Opening a document only
// queues a request; HTTP and indexing never wait for conversion. Failures stay in
// preview_cache and do not modify extraction, document or job records.
func (runtime *Runtime) previewWorker(ctx context.Context) {
	defer runtime.workers.Done()
	service := runtime.Service
	_ = service.recoverPreviews(ctx)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastCleanup := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if time.Since(lastCleanup) > time.Hour {
			_ = service.Database.Write(ctx, func(tx *sql.Tx) error {
				settings, err := previewSettings(ctx, tx)
				if err != nil {
					return err
				}
				if settings.AutomaticCleanup {
					return service.trimPreviews(ctx, tx, settings, 0, true)
				}
				return nil
			})
			lastCleanup = time.Now()
		}
		job, settings, err := service.claimPreview(ctx)
		if err != nil || job.ID == "" {
			continue
		}
		operation, cancel := context.WithTimeout(ctx, 120*time.Second)
		err = isolatedOperation(func() error { return service.generatePreview(operation, job, settings) })
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			code := failureCode(err)
			if code == "PROCESS_TIMEOUT" {
				code = "PREVIEW_TIMEOUT"
			}
			_, _ = service.Database.Writer.ExecContext(ctx, "UPDATE preview_cache SET status='error',error_code=? WHERE id=? AND status='generating'", code, job.ID)
			_ = diagnostics.Record(ctx, service.Database, "preview", code, diagnostics.Context{DocumentID: job.DocumentID, Operation: "preview"})
		}
	}
}
func (service *Service) claimPreview(ctx context.Context) (previewJob, PreviewSettings, error) {
	var job previewJob
	var settings PreviewSettings
	err := service.Database.Write(ctx, func(tx *sql.Tx) error {
		var err error
		settings, err = previewSettings(ctx, tx)
		if err != nil {
			return err
		}
		if !settings.Enabled {
			return nil
		}
		var paused bool
		if err = tx.QueryRowContext(ctx, "SELECT paused FROM processing_policy WHERE singleton=1").Scan(&paused); err != nil {
			return err
		}
		if paused {
			return nil
		}
		err = tx.QueryRowContext(ctx, `SELECT id,document_id,content_version_id,source_hash,generator_version,requested_by FROM preview_cache WHERE status='queued' ORDER BY created_at,id LIMIT 1`).Scan(&job.ID, &job.DocumentID, &job.Version, &job.Hash, &job.Generator, &job.RequestedBy)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE preview_cache SET status='generating' WHERE id=?", job.ID)
		return err
	})
	return job, settings, err
}
func (service *Service) recoverPreviews(ctx context.Context) error {
	for _, directory := range []string{service.previewDirectory(), service.previewWorkDirectory()} {
		if err := storage.PreparePrivateDirectory(directory); err != nil {
			return err
		}
	}
	// The exclusive application state lock excludes a surviving previous worker.
	entries, err := os.ReadDir(service.previewWorkDirectory())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "preview-") {
			if err = os.RemoveAll(filepath.Join(service.previewWorkDirectory(), entry.Name())); err != nil {
				return err
			}
		}
	}
	entries, err = os.ReadDir(service.previewDirectory())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".preview") {
			continue
		}
		var count int
		id := strings.TrimSuffix(entry.Name(), ".preview")
		if err = service.Database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM preview_cache WHERE id=? AND status='ready'", id).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			if err = os.Remove(filepath.Join(service.previewDirectory(), entry.Name())); err != nil {
				return err
			}
		}
	}
	return service.Database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE preview_cache SET status='error',error_code='PROCESS_RESTARTED' WHERE status='generating'"); err != nil {
			return err
		}
		settings, err := previewSettings(ctx, tx)
		if err != nil {
			return err
		}
		return service.trimPreviews(ctx, tx, settings, 0, settings.AutomaticCleanup)
	})
}
func (service *Service) generatePreview(ctx context.Context, job previewJob, settings PreviewSettings) error {
	principal := domain.Principal{User: domain.User{ID: job.RequestedBy}}
	// Work was authorized by a request, but permissions and visibility may have
	// changed while queued. Recheck them before reading any private original.
	if err := service.Database.Write(ctx, func(tx *sql.Tx) error { return service.previewRead(ctx, tx, principal, job.DocumentID, job.Hash) }); err != nil {
		return err
	}
	document, err := scanDocument(service.Database.Reader.QueryRowContext(ctx, "SELECT "+documentColumns+documentJoins+" WHERE d.id=?", job.DocumentID))
	if err != nil {
		return err
	}
	settings, err = service.libraryPreviewSettings(ctx, service.Database.Reader, document.LibraryID)
	if err != nil {
		return err
	}
	if reason := previewEligibility(document, settings); reason != "" {
		return previews.Failure(reason)
	}
	if job.Generator != previews.Generator(settings.ConverterPath) {
		return previews.Failure("PREVIEW_OBSOLETE")
	}
	for _, directory := range []string{service.previewDirectory(), service.previewWorkDirectory()} {
		if err = storage.PreparePrivateDirectory(directory); err != nil {
			return previews.Failure("PREVIEW_STORAGE_ERROR")
		}
	}
	directory, err := os.MkdirTemp(service.previewWorkDirectory(), "preview-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	if err = storage.ProtectPrivatePath(directory, true); err != nil {
		return err
	}
	var original *os.File
	if document.Availability == "staged" {
		original, err = service.openUpload(ctx, document.FileID)
	} else {
		var root Root
		root, err = service.root(ctx, document.RootID)
		if err == nil {
			original, err = openLinked(root, document.RelativePath)
		}
	}
	if err != nil {
		return previews.Failure("DOCUMENT_UNAVAILABLE")
	}
	defer original.Close()
	originalIdentity, _, err := physicalIdentity(original)
	if err != nil || originalIdentity != document.Identity {
		return previews.Failure("PREVIEW_OBSOLETE")
	}
	before, err := original.Stat()
	if err != nil {
		return err
	}
	if before.Size() > int64(settings.MaximumSourceMB)<<20 {
		return previews.Failure("PREVIEW_SOURCE_LIMIT")
	}
	source := filepath.Join(directory, "source."+document.Format)
	snapshot, err := os.OpenFile(source, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyError := io.Copy(io.MultiWriter(snapshot, hash), contextReader{ctx, io.LimitReader(original, (int64(settings.MaximumSourceMB)<<20)+1)})
	closeError := snapshot.Close()
	if copyError != nil {
		return copyError
	}
	if closeError != nil {
		return closeError
	}
	if written > int64(settings.MaximumSourceMB)<<20 {
		return previews.Failure("PREVIEW_SOURCE_LIMIT")
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != job.Hash {
		return previews.Failure("PREVIEW_OBSOLETE")
	}
	output := filepath.Join(directory, "preview.pdf")
	convert := previews.Convert
	if service.previewConvert != nil {
		convert = service.previewConvert
	}
	if err = convert(ctx, source, document.Format, output, settings.ConverterPath); err != nil {
		return err
	}
	after, err := original.Stat()
	if err != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return previews.Failure("PREVIEW_OBSOLETE")
	}
	info, err := os.Lstat(output)
	if err != nil || !info.Mode().IsRegular() {
		return previews.Failure("PREVIEW_CONVERSION_FAILED")
	}
	if info.Size() > previews.MaximumOutput {
		return previews.Failure("PREVIEW_SIZE_LIMIT")
	}
	if err = storage.ProtectPrivatePath(output, false); err != nil {
		return err
	}
	return service.Database.Write(ctx, func(tx *sql.Tx) error {
		if err := service.previewRead(ctx, tx, principal, job.DocumentID, job.Hash); err != nil {
			return err
		}
		current, err := service.libraryPreviewSettings(ctx, tx, document.LibraryID)
		if err != nil {
			return err
		}
		if !current.Enabled || job.Generator != previews.Generator(current.ConverterPath) {
			return previews.Failure("PREVIEW_OBSOLETE")
		}
		if written > int64(current.MaximumSourceMB)<<20 {
			return previews.Failure("PREVIEW_SOURCE_LIMIT")
		}
		var pending int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM preview_cache WHERE id=? AND status='generating' AND content_version_id=(SELECT current_content_version_id FROM physical_files WHERE id=?)", job.ID, document.FileID).Scan(&pending); err != nil {
			return err
		}
		if pending != 1 {
			return previews.Failure("PREVIEW_OBSOLETE")
		}
		if err = service.trimPreviews(ctx, tx, current, info.Size(), current.AutomaticCleanup); err != nil {
			return err
		}
		if err = os.Rename(output, service.previewPath(job.ID)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE preview_cache SET status='ready',media_type='application/pdf',size_bytes=?,generated_at=?,last_accessed_at=?,error_code='' WHERE id=?", info.Size(), now(), now(), job.ID)
		return err
	})
}
