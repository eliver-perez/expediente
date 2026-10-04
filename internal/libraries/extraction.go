package libraries

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/extraction"
	"gestor-documental/internal/storage"
)

func appendDocumentObservation(ctx context.Context, transaction *sql.Tx, eventID, eventType, libraryID, documentID, versionID string) error {
	if err := audit.Append(ctx, transaction, time.Now(), audit.Event{ID: eventID, Type: eventType, LibraryID: libraryID, DocumentID: documentID, SystemActor: true, Metadata: domain.RequestMetadata{RequestID: domain.NewID()}, Details: map[string]any{"content_version_id": versionID}}); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, "UPDATE documents SET revision=revision+1 WHERE id=?", documentID); err != nil {
		return err
	}
	_, err := transaction.ExecContext(ctx, "INSERT INTO document_history(id,document_id,revision,event_id,snapshot_json) SELECT ?,id,revision,?,? FROM documents WHERE id=?", domain.NewID(), eventID, encode(map[string]string{"content_version_id": versionID}), documentID)
	return err
}

type preparedExtraction struct {
	service                    *Service
	job                        Job
	root                       Root
	file                       *os.File
	before                     os.FileInfo
	path, temporary, languages string
	relative                   string
	options                    extraction.Options
	pages                      []extraction.Page
	result                     extraction.Result
	started                    string
}

func (p *preparedExtraction) cleanup() { p.file.Close(); os.RemoveAll(p.temporary) }
func (p *preparedExtraction) complete(ctx context.Context) (resultError error) {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	defer func() {
		if recover() != nil {
			resultError = scanFailure("PROCESSING_INTERNAL_ERROR")
		}
		if resultError != nil {
			resultError = p.service.extractionFailure(ctx, p.job, p.languages, p.started, p.result, resultError)
		}
	}()
	p.options.Progress = func(operation string, completed, total int) {
		_ = p.service.reportProgress(ctx, p.job, operation, p.relative, completed, total)
	}
	if err := extraction.CompleteDocument(ctx, p.path, p.languages, p.options, &p.result); err != nil {
		return err
	}
	after, err := p.file.Stat()
	if err != nil || after.Size() != p.before.Size() || !after.ModTime().Equal(p.before.ModTime()) {
		return scanFailure("FILE_UNSTABLE")
	}
	_ = p.service.reportProgress(ctx, p.job, "publishing", "", len(p.result.Units), len(p.result.Units))
	return p.service.publish(ctx, p.job, p.root, p.result, p.languages, p.started)
}
func (service *Service) extractionFailure(ctx context.Context, job Job, languages, started string, result extraction.Result, cause error) error {
	// Persist the attempt's outcome even when its worker was cancelled. The short
	// independent deadline prevents journal writes delaying shutdown indefinitely.
	journalContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	err := service.Database.Write(journalContext, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(journalContext, `INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,error_code,extractor_id,extractor_version,document_format,started_at,completed_at,result_code,summary_json,warnings_json) VALUES(?,?,?,?,?,'failed',?,?,?,?,?,?,'failed',?,?)`, domain.NewID(), job.FileID, job.Version, result.Extractor.ID+"-v"+result.Extractor.Version, languages, failureCode(cause), result.Extractor.ID, result.Extractor.Version, result.Extractor.Format, started, now(), encode(result.Summary), encode(result.Warnings))
		return err
	})
	if err != nil {
		return err
	}
	return cause
}
func (service *Service) Extract(ctx context.Context, job Job, options extraction.Options) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout(job))
	defer cancel()
	prepared, err := service.prepareExtraction(ctx, job, options)
	if err != nil {
		return err
	}
	defer prepared.cleanup()
	return prepared.complete(ctx)
}
func (service *Service) prepareExtraction(ctx context.Context, job Job, options extraction.Options) (prepared *preparedExtraction, resultError error) {
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	if err := service.jobLicense(ctx, service.Database.Reader, job); err != nil {
		return nil, err
	}
	configuration, err := service.fileConfiguration(ctx, service.Database.Reader, job.LibraryID)
	if err != nil {
		return nil, err
	}
	options.MaximumFileMB = configuration.Effective.MaximumIndexMB
	if reason, err := service.versionIndexReason(ctx, service.Database.Reader, job.LibraryID, job.Version); err != nil {
		return nil, err
	} else if reason != "" {
		_, _ = service.Database.Writer.ExecContext(ctx, "UPDATE content_versions SET index_block_reason=? WHERE id=?", reason, job.Version)
		return nil, scanFailure("FILE_INDEX_DISABLED")
	}
	var rootID, relative, expectedHash, identity, languages, availability, libraryID, format string
	var size int64
	err = service.Database.Reader.QueryRowContext(ctx, "SELECT coalesce(l.root_id,''),coalesce(l.relative_path,''),v.sha256,v.size_bytes,f.os_identity_key,b.ocr_languages,f.availability,f.library_id,v.document_format FROM physical_files f LEFT JOIN physical_file_locations l ON l.id=f.primary_location_id JOIN content_versions v ON v.id=f.current_content_version_id JOIN libraries b ON b.id=f.library_id JOIN documents d ON d.physical_file_id=f.id WHERE f.id=? AND v.id=? AND d.deleted_at IS NULL", job.FileID, job.Version).Scan(&rootID, &relative, &expectedHash, &size, &identity, &languages, &availability, &libraryID, &format)
	if err == sql.ErrNoRows {
		return nil, scanFailure("VERSION_SUPERSEDED")
	}
	if err != nil {
		return nil, err
	}
	descriptor, available := extraction.ExtractorFor(format)
	if !available {
		return nil, scanFailure("EXTRACTOR_UNAVAILABLE")
	}
	started := now()
	result := extraction.Result{Extractor: descriptor, Summary: map[string]any{}, Warnings: []string{}}
	defer func() {
		if recover() != nil {
			resultError = scanFailure("PROCESSING_INTERNAL_ERROR")
		}
		if resultError != nil {
			resultError = service.extractionFailure(ctx, job, languages, started, result, resultError)
		}
	}()
	root := Root{LibraryID: libraryID}
	var file *os.File
	if availability == "staged" {
		file, err = service.openUpload(ctx, job.FileID)
	} else {
		root, err = service.root(ctx, rootID)
		if err != nil {
			return nil, err
		}
		if !verifiableRoot(root) {
			return nil, scanFailure("ROOT_DISABLED")
		}
		file, err = openLinked(root, relative)
	}
	if err != nil {
		return nil, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	defer func() {
		if prepared == nil {
			file.Close()
		}
	}()
	currentIdentity, _, err := physicalIdentity(file)
	if err != nil || currentIdentity != identity {
		return nil, scanFailure("VERSION_SUPERSEDED")
	}
	temporaryRoot := filepath.Join(service.Identity.Config.StateDirectory, "extraction")
	if err = storage.PreparePrivateDirectory(temporaryRoot); err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp(temporaryRoot, "job-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if prepared == nil {
			os.RemoveAll(temporary)
		}
	}()
	if err = storage.ProtectPrivatePath(temporary, true); err != nil {
		return nil, err
	}
	documentPath := filepath.Join(temporary, "source."+format)
	snapshot, err := os.OpenFile(documentPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer snapshot.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	_ = service.reportProgress(ctx, job, "snapshot", relative, 0, 0)
	written, copyError := io.Copy(io.MultiWriter(snapshot, hash), contextReader{ctx, io.LimitReader(file, (int64(options.MaximumFileMB)<<20)+1)})
	closeError := snapshot.Close()
	if copyError != nil {
		return nil, copyError
	}
	if closeError != nil {
		return nil, closeError
	}
	if written > int64(options.MaximumFileMB)<<20 {
		return nil, scanFailure("FILE_SIZE_LIMIT")
	}
	if written != size || fmt.Sprintf("%x", hash.Sum(nil)) != expectedHash {
		return nil, scanFailure("FILE_UNSTABLE")
	}
	options.Progress = func(operation string, completed, total int) {
		_ = service.reportProgress(ctx, job, operation, relative, completed, total)
	}
	result, err = extraction.PrepareDocument(ctx, documentPath, format, options)
	if err != nil {
		return nil, err
	}
	prepared = &preparedExtraction{service: service, job: job, root: root, file: file, before: before, path: documentPath, temporary: temporary, languages: languages, relative: relative, options: options, pages: result.Units, result: result, started: started}
	return prepared, nil
}
func (service *Service) publish(ctx context.Context, job Job, root Root, result extraction.Result, languages, started string) error {
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if reason, err := service.versionIndexReason(ctx, transaction, root.LibraryID, job.Version); err != nil {
			return err
		} else if reason != "" {
			return scanFailure("FILE_INDEX_DISABLED")
		}
		if err := service.jobLicense(ctx, transaction, Job{Kind: "extract", LibraryID: root.LibraryID}); err != nil {
			return err
		}
		if root.ID != "" {
			if err := service.checkRootRevision(ctx, transaction, root); err != nil {
				return err
			}
		}
		var currentLanguages string
		if err := transaction.QueryRowContext(ctx, "SELECT ocr_languages FROM libraries WHERE id=?", root.LibraryID).Scan(&currentLanguages); err != nil {
			return err
		}
		if result.Extractor.Format == "pdf" && currentLanguages != languages {
			return scanFailure("VERSION_SUPERSEDED")
		}
		var permitted int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM jobs j JOIN physical_files f ON f.id=j.physical_file_id JOIN documents d ON d.physical_file_id=f.id WHERE j.id=? AND j.status='running' AND j.fencing_token=? AND NOT EXISTS(SELECT 1 FROM job_controls WHERE job_id=j.id AND cancel_requested=1) AND f.current_content_version_id=? AND d.deleted_at IS NULL AND d.approval_status<>'cancelled'", job.ID, job.Fence, job.Version).Scan(&permitted); err != nil {
			return err
		}
		if permitted != 1 {
			return scanFailure("VERSION_SUPERSEDED")
		}
		extractionID := domain.NewID()
		var pageCount any
		if result.Extractor.Format == "pdf" {
			pageCount = len(result.Units)
		}
		outcome := "complete"
		if len(result.Warnings) > 0 {
			outcome = "complete_with_warnings"
		}
		if _, err := transaction.ExecContext(ctx, `INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count,completed_at,extractor_id,extractor_version,document_format,started_at,result_code,unit_count,summary_json,warnings_json) VALUES(?,?,?,?,?,'complete',?,?,?,?,?,?,?,?,?,?)`, extractionID, job.FileID, job.Version, result.Extractor.ID+"-v"+result.Extractor.Version, languages, pageCount, now(), result.Extractor.ID, result.Extractor.Version, result.Extractor.Format, started, outcome, len(result.Units), encode(result.Summary), encode(result.Warnings)); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "DELETE FROM indexed_pages WHERE physical_file_id=?", job.FileID); err != nil {
			return err
		}
		for _, page := range result.Units {
			if _, err := transaction.ExecContext(ctx, "INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text,unit_kind,context_label,context_json) VALUES(?,?,?,?,?,?,?)", extractionID, page.Number, page.Method, page.Text, page.Kind, page.Label, encode(page.Context)); err != nil {
				return err
			}
			if _, err := transaction.ExecContext(ctx, "INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) VALUES(?,?,?,?)", job.FileID, extractionID, page.Number, page.Text); err != nil {
				return err
			}
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE content_versions SET index_block_reason='' WHERE id=?", job.Version); err != nil {
			return err
		}
		_, err := transaction.ExecContext(ctx, "UPDATE physical_files SET indexed_extraction_id=?,indexed_at=?,extraction_freshness='current' WHERE id=?", extractionID, now(), job.FileID)
		return err
	})
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}
