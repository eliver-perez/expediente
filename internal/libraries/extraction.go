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
func (service *Service) Extract(ctx context.Context, job Job, options extraction.Options) error {
	if err := service.jobLicense(ctx, service.Database.Reader, job); err != nil {
		return err
	}
	var rootID, relative, expectedHash, identity, languages, availability, libraryID string
	var size int64
	err := service.Database.Reader.QueryRowContext(ctx, "SELECT coalesce(l.root_id,''),coalesce(l.relative_path,''),v.sha256,v.size_bytes,f.os_identity_key,b.ocr_languages,f.availability,f.library_id FROM physical_files f LEFT JOIN physical_file_locations l ON l.id=f.primary_location_id JOIN content_versions v ON v.id=f.current_content_version_id JOIN libraries b ON b.id=f.library_id JOIN documents d ON d.physical_file_id=f.id WHERE f.id=? AND v.id=? AND d.deleted_at IS NULL", job.FileID, job.Version).Scan(&rootID, &relative, &expectedHash, &size, &identity, &languages, &availability, &libraryID)
	if err == sql.ErrNoRows {
		return scanFailure("VERSION_SUPERSEDED")
	}
	if err != nil {
		return err
	}
	root := Root{LibraryID: libraryID}
	var file *os.File
	if availability == "staged" {
		file, err = service.openUpload(ctx, job.FileID)
	} else {
		root, err = service.root(ctx, rootID)
		if err != nil {
			return err
		}
		if !verifiableRoot(root) {
			return scanFailure("ROOT_DISABLED")
		}
		file, err = openLinked(root, relative)
	}
	if err != nil {
		return scanFailure("DOCUMENT_UNAVAILABLE")
	}
	defer file.Close()
	currentIdentity, _, err := physicalIdentity(file)
	if err != nil || currentIdentity != identity {
		return scanFailure("VERSION_SUPERSEDED")
	}
	temporaryRoot := filepath.Join(service.Identity.Config.StateDirectory, "extraction")
	if err = storage.PreparePrivateDirectory(temporaryRoot); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(temporaryRoot, "job-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if err = storage.ProtectPrivatePath(temporary, true); err != nil {
		return err
	}
	documentPath := filepath.Join(temporary, "source.pdf")
	snapshot, err := os.OpenFile(documentPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	before, err := file.Stat()
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyError := io.Copy(io.MultiWriter(snapshot, hash), io.LimitReader(file, (int64(options.MaximumFileMB)<<20)+1))
	closeError := snapshot.Close()
	if copyError != nil {
		return copyError
	}
	if closeError != nil {
		return closeError
	}
	if written != size || fmt.Sprintf("%x", hash.Sum(nil)) != expectedHash {
		return scanFailure("FILE_UNSTABLE")
	}
	pages, err := options.Extract(ctx, documentPath, languages)
	if err != nil {
		recordError := service.Database.Write(ctx, func(transaction *sql.Tx) error {
			_, insertError := transaction.ExecContext(ctx, "INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,error_code) VALUES(?,?,?,'poppler-tesseract-v1',?,'failed',?)", domain.NewID(), job.FileID, job.Version, languages, failureCode(err))
			return insertError
		})
		if recordError != nil {
			return recordError
		}
		return err
	}
	after, err := file.Stat()
	if err != nil || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return scanFailure("FILE_UNSTABLE")
	}
	return service.publish(ctx, job, root, pages, languages)
}
func (service *Service) publish(ctx context.Context, job Job, root Root, pages []extraction.Page, languages string) error {
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
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
		if currentLanguages != languages {
			return scanFailure("VERSION_SUPERSEDED")
		}
		var permitted int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM jobs j JOIN physical_files f ON f.id=j.physical_file_id JOIN documents d ON d.physical_file_id=f.id WHERE j.id=? AND j.status='running' AND j.fencing_token=? AND f.current_content_version_id=? AND d.deleted_at IS NULL AND d.approval_status<>'cancelled'", job.ID, job.Fence, job.Version).Scan(&permitted); err != nil {
			return err
		}
		if permitted != 1 {
			return scanFailure("VERSION_SUPERSEDED")
		}
		extractionID := domain.NewID()
		if _, err := transaction.ExecContext(ctx, "INSERT INTO extraction_runs(id,physical_file_id,content_version_id,extractor_revision,language_codes,status,page_count,completed_at) VALUES(?,?,?,'poppler-tesseract-v1',?,'complete',?,?)", extractionID, job.FileID, job.Version, languages, len(pages), now()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "DELETE FROM indexed_pages WHERE physical_file_id=?", job.FileID); err != nil {
			return err
		}
		for _, page := range pages {
			if _, err := transaction.ExecContext(ctx, "INSERT INTO extraction_pages(extraction_id,page_number,extraction_method,page_text) VALUES(?,?,?,?)", extractionID, page.Number, page.Method, page.Text); err != nil {
				return err
			}
			if _, err := transaction.ExecContext(ctx, "INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) VALUES(?,?,?,?)", job.FileID, extractionID, page.Number, page.Text); err != nil {
				return err
			}
		}
		_, err := transaction.ExecContext(ctx, "UPDATE physical_files SET indexed_extraction_id=?,indexed_at=?,extraction_freshness='current' WHERE id=?", extractionID, now(), job.FileID)
		return err
	})
}
