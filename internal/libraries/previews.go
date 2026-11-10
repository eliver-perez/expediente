package libraries

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/previews"
)

type Preview struct {
	ID         string `json:"id,omitempty"`
	State      string `json:"state"`
	Error      string `json:"error_code,omitempty"`
	SourceHash string `json:"source_hash,omitempty"`
	Generator  string `json:"generator_version,omitempty"`
	Generated  string `json:"generated_at,omitempty"`
	MediaType  string `json:"media_type,omitempty"`
	URL        string `json:"url,omitempty"`
	Paused     bool   `json:"paused"`
}

func previewEligibility(document Document, settings PreviewSettings) string {
	if !settings.Enabled {
		return "PREVIEW_DISABLED"
	}
	if document.Format != "docx" && document.Format != "xlsx" {
		return "PREVIEW_UNSUPPORTED"
	}
	if document.Size == nil || *document.Size > int64(settings.MaximumSourceMB)<<20 {
		return "PREVIEW_SOURCE_LIMIT"
	}
	if (document.Availability != "available" && document.Availability != "staged") || document.IndexReason == "content_rejected" {
		return "DOCUMENT_UNAVAILABLE"
	}
	return ""
}
func (service *Service) Preview(ctx context.Context, principal domain.Principal, documentID string) (Preview, error) {
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return Preview{}, err
	}
	settings, err := service.libraryPreviewSettings(ctx, service.Database.Reader, document.LibraryID)
	if err != nil {
		return Preview{}, err
	}
	result := Preview{State: "not_generated"}
	if reason := previewEligibility(document, settings); reason != "" {
		result.State = "unavailable"
		result.Error = reason
		return result, nil
	}
	generator := previews.Generator(settings.ConverterPath)
	err = service.Database.Reader.QueryRowContext(ctx, `SELECT id,status,error_code,source_hash,generator_version,generated_at,media_type FROM preview_cache WHERE document_id=? AND source_hash=? AND generator_version=?`, documentID, document.Hash, generator).Scan(&result.ID, &result.State, &result.Error, &result.SourceHash, &result.Generator, &result.Generated, &result.MediaType)
	if errors.Is(err, sql.ErrNoRows) {
		var old int
		err = service.Database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM preview_cache WHERE document_id=?", documentID).Scan(&old)
		if old > 0 {
			result.State = "obsolete"
		}
		return result, err
	}
	if err != nil {
		return result, err
	}
	policy, err := service.ProcessingPolicy(ctx)
	if err != nil {
		return result, err
	}
	result.Paused = policy.Paused && result.State == "queued"
	if result.State == "ready" {
		info, err := os.Lstat(service.previewPath(result.ID))
		if err != nil || !info.Mode().IsRegular() {
			result.State = "obsolete"
			return result, nil
		}
		result.URL = "/api/v1/documents/" + url.PathEscape(documentID) + "/preview/content?id=" + url.QueryEscape(result.ID)
	}
	return result, nil
}
func (service *Service) RequestPreview(ctx context.Context, principal domain.Principal, documentID string, retry bool, metadata domain.RequestMetadata) (Preview, error) {
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return Preview{}, err
	}
	err = service.Identity.AuthorizedWrite(ctx, principal, "", func(tx *sql.Tx, current domain.Principal) error {
		if err := service.previewRead(ctx, tx, current, documentID, document.Hash); err != nil {
			return err
		}
		settings, err := service.libraryPreviewSettings(ctx, tx, document.LibraryID)
		if err != nil {
			return err
		}
		if reason := previewEligibility(document, settings); reason != "" {
			return previews.Failure(reason)
		}
		generator := previews.Generator(settings.ConverterPath)
		var id, status string
		err = tx.QueryRowContext(ctx, `SELECT id,status FROM preview_cache WHERE document_id=? AND source_hash=? AND generator_version=?`, documentID, document.Hash, generator).Scan(&id, &status)
		if err == nil {
			info, fileError := os.Lstat(service.previewPath(id))
			missing := status == "ready" && (fileError != nil || !info.Mode().IsRegular())
			if !missing && (status != "error" || !retry) {
				return nil
			}
			if err = service.deletePreview(ctx, tx, id); err != nil {
				return err
			}
		} else if err != sql.ErrNoRows {
			return err
		}
		var pending int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM preview_cache WHERE status IN ('queued','generating')").Scan(&pending); err != nil {
			return err
		}
		if pending >= 32 {
			return previews.Failure("PREVIEW_QUEUE_FULL")
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO preview_cache(id,document_id,content_version_id,source_hash,generator_version,requested_by,status,created_at,last_accessed_at) SELECT ?,d.id,f.current_content_version_id,?,?,?,'queued',?,? FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=?`, domain.NewID(), document.Hash, generator, current.User.ID, now(), now(), documentID)
		if err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "document.preview_requested", document.LibraryID, documentID, nil)
	})
	if err != nil {
		return Preview{}, err
	}
	return service.Preview(ctx, principal, documentID)
}

// Every delivery rechecks current permissions, private-upload visibility, license,
// current hash and generator. A cached representation grants no independent access.
func (service *Service) previewRead(ctx context.Context, tx *sql.Tx, principal domain.Principal, documentID, hash string) error {
	if err := service.Identity.License.Check(ctx, licensing.ReadDocuments); err != nil {
		return err
	}
	var libraryID string
	err := tx.QueryRowContext(ctx, `SELECT d.library_id FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN content_versions v ON v.id=f.current_content_version_id JOIN users u ON u.id=? AND u.disabled_at IS NULL WHERE d.id=? AND v.sha256=? AND d.deleted_at IS NULL AND `+visibleDocumentSQL, principal.User.ID, documentID, hash, principal.User.ID, principal.User.ID).Scan(&libraryID)
	if err == sql.ErrNoRows {
		return notFound()
	}
	if err != nil {
		return err
	}
	return service.require(ctx, tx, principal, libraryID, "documents.read")
}
func (service *Service) OpenPreview(ctx context.Context, principal domain.Principal, documentID, id string, metadata domain.RequestMetadata) (*os.File, string, error) {
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return nil, "", err
	}
	var file *os.File
	var media string
	err = service.Identity.AuthorizedWrite(ctx, principal, "", func(tx *sql.Tx, current domain.Principal) error {
		if err := service.previewRead(ctx, tx, current, documentID, document.Hash); err != nil {
			return err
		}
		settings, err := service.libraryPreviewSettings(ctx, tx, document.LibraryID)
		if err != nil {
			return err
		}
		if reason := previewEligibility(document, settings); reason != "" {
			return previews.Failure(reason)
		}
		var size int64
		err = tx.QueryRowContext(ctx, `SELECT media_type,size_bytes FROM preview_cache WHERE id=? AND document_id=? AND source_hash=? AND generator_version=? AND status='ready'`, id, documentID, document.Hash, previews.Generator(settings.ConverterPath)).Scan(&media, &size)
		if err == sql.ErrNoRows {
			return previews.Failure("PREVIEW_OBSOLETE")
		}
		if err != nil {
			return err
		}
		info, err := os.Lstat(service.previewPath(id))
		if err != nil || !info.Mode().IsRegular() || info.Size() != size {
			return previews.Failure("PREVIEW_OBSOLETE")
		}
		file, err = os.Open(service.previewPath(id))
		if err != nil {
			return previews.Failure("PREVIEW_STORAGE_ERROR")
		}
		if _, err = tx.ExecContext(ctx, "UPDATE preview_cache SET last_accessed_at=? WHERE id=?", now(), id); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "document.preview_opened", document.LibraryID, documentID, map[string]any{"preview_id": id})
	})
	if err != nil && file != nil {
		file.Close()
		file = nil
	}
	return file, media, err
}
