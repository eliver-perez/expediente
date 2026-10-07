package libraries

import (
	"context"
	"database/sql"
	"os"

	"gestor-documental/internal/domain"
)

// ReindexDocument verifies the original outside the writer transaction, then
// queues the ordinary extraction pipeline. That pipeline verifies the snapshot
// again and atomically publishes its index; failed attempts retain the old text.
func (service *Service) ReindexDocument(ctx context.Context, principal domain.Principal, documentID, requestID string, metadata domain.RequestMetadata) (string, error) {
	if len(requestID) < 16 || len(requestID) > 100 {
		return "", invalid("Se requiere una clave de idempotencia de 16 a 100 caracteres.")
	}
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return "", err
	}
	var identifier string
	// Repeat authorization both before reading bytes and before committing work.
	authorize := func(tx *sql.Tx, current domain.Principal) error {
		if err := service.require(ctx, tx, current, document.LibraryID, "documents.read"); err != nil {
			return err
		}
		var approval string
		err := tx.QueryRowContext(ctx, "SELECT d.approval_status FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=? AND d.deleted_at IS NULL AND "+visibleDocumentSQL, documentID, current.User.ID, current.User.ID).Scan(&approval)
		if err == sql.ErrNoRows {
			return notFound()
		}
		if err != nil {
			return err
		}
		if approval == "materializing" || approval == "cancelled" {
			return invalid("Espera a que termine el guardado; los documentos cancelados no se reindexan.")
		}
		if err = service.jobLicense(ctx, tx, Job{LibraryID: document.LibraryID, Kind: "extract"}); err != nil {
			return err
		}
		var digest string
		err = tx.QueryRowContext(ctx, "SELECT request_digest,resource_id FROM idempotency_requests WHERE actor_user_id=? AND operation='document.reindex' AND idempotency_key=?", current.User.ID, requestID).Scan(&digest, &identifier)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		if digest != domain.Digest(documentID) {
			return domain.Failure("IDEMPOTENCY_CONFLICT", "La clave ya se utilizó para otra solicitud.", 409)
		}
		return nil
	}
	if err = service.write(ctx, principal, document.LibraryID, "indexing.run", authorize); err != nil || identifier != "" {
		return identifier, err
	}
	configuration, err := service.fileConfiguration(ctx, service.Database.Reader, document.LibraryID)
	if err != nil {
		return "", err
	}
	var root Root
	var file *os.File
	if document.Availability == "staged" {
		file, err = service.openUpload(ctx, document.FileID)
	} else {
		root, err = service.root(ctx, document.RootID)
		if err == nil && !verifiableRoot(root) {
			err = scanFailure("ROOT_DISABLED")
		}
		if err == nil {
			file, err = openLinked(root, document.RelativePath)
		}
	}
	if err != nil {
		return "", domain.Failure("DOCUMENT_UNAVAILABLE", "No se puede leer el original. Comprueba la carpeta y sus permisos antes de reindexar.", 409)
	}
	defer file.Close()
	observed, err := observeFile(ctx, file, document.Filename, int64(configuration.Effective.MaximumIndexMB)<<20, false)
	if err != nil {
		return "", err
	}
	if observed.Identity != document.Identity {
		return "", domain.Failure("REINDEX_SOURCE_CHANGED", "El archivo original fue reemplazado. Verifica la biblioteca antes de reindexar este documento.", 409)
	}
	if document.Availability == "staged" && observed.Hash != document.Hash {
		return "", domain.Failure("REINDEX_SOURCE_CHANGED", "La carga temporal cambió y no coincide con el original registrado.", 409)
	}
	err = service.write(ctx, principal, document.LibraryID, "indexing.run", func(tx *sql.Tx, current domain.Principal) error {
		if err := authorize(tx, current); err != nil || identifier != "" {
			return err
		}
		live, err := service.fileConfiguration(ctx, tx, document.LibraryID)
		if err != nil {
			return err
		}
		if live.Effective.IndexReason(observed.Detection, observed.Size) != "" {
			return domain.Failure("REINDEX_NOT_ALLOWED", "La configuración actual no permite indexar este formato o tamaño de archivo.", 409)
		}
		if root.ID != "" {
			if err = service.checkRootRevision(ctx, tx, root); err != nil {
				return err
			}
		}
		var version, digest, identity, availability string
		var revision int64
		err = tx.QueryRowContext(ctx, `SELECT f.current_content_version_id,v.sha256,coalesce(f.os_identity_key,''),f.availability,d.revision FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN content_versions v ON v.id=f.current_content_version_id WHERE d.id=? AND f.id=?`, documentID, document.FileID).Scan(&version, &digest, &identity, &availability, &revision)
		if err != nil {
			return err
		}
		if digest != document.Hash || identity != document.Identity || revision != document.Revision || (availability == "staged") != (document.Availability == "staged") {
			return domain.Failure("REVISION_CONFLICT", "El documento cambió durante la comprobación. Actualiza la ficha e inténtalo de nuevo.", 409)
		}
		if observed.Hash != digest {
			origin := "scan"
			if document.Source == "managed" {
				origin = "external"
			}
			version = domain.NewID()
			_, err = tx.ExecContext(ctx, `INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,sha256,observed_change_key,observed_at,change_origin) SELECT ?,?,coalesce(max(generation),0)+1,?,?,?,?,?,? FROM content_versions WHERE physical_file_id=?`, version, document.FileID, observed.Size, observed.Modified, observed.Hash, version, now(), origin, document.FileID)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE physical_files SET current_content_version_id=? WHERE id=?", version, document.FileID); err != nil {
				return err
			}
			kind := "linked_changed"
			if document.Source == "managed" {
				kind = "managed_changed"
				if _, err = tx.ExecContext(ctx, "UPDATE physical_files SET integrity_status='changed' WHERE id=?", document.FileID); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, "UPDATE documents SET approval_status=CASE WHEN approval_status='approved' THEN 'needs_review' ELSE approval_status END WHERE id=?", documentID); err != nil {
					return err
				}
			}
			if err = managedObservation(ctx, tx, document.LibraryID, documentID, version, kind); err != nil {
				return err
			}
		}
		if err = saveDetection(ctx, tx, version, observed.Detection, ""); err != nil {
			return err
		}
		err = tx.QueryRowContext(ctx, `SELECT id FROM jobs WHERE physical_file_id=? AND job_type='extract' AND target_version=? AND status IN ('queued','running','retry_wait','paused') ORDER BY created_at,id LIMIT 1`, document.FileID, version).Scan(&identifier)
		reused := err == nil
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if !reused {
			identifier = domain.NewID()
			_, err = tx.ExecContext(ctx, `INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{"manual_reindex":true}','queued',?,?)`, identifier, document.LibraryID, document.FileID, version, identifier, now(), now())
			if err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE physical_files SET availability=CASE WHEN availability='staged' THEN 'staged' ELSE 'available' END,extraction_freshness=CASE WHEN indexed_extraction_id IS NULL THEN 'none' ELSE 'stale' END WHERE id=?`, document.FileID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO idempotency_requests VALUES(?,'document.reindex',?,?,?,'complete',?)`, current.User.ID, requestID, domain.Digest(documentID), identifier, now()); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE jobs SET payload_json=json_set(payload_json,'$.manual',json('true')) WHERE id=?`, identifier); err != nil {
			return err
		}
		return record(ctx, tx, current, metadata, "document.reindex_requested", document.LibraryID, documentID, map[string]any{"job_id": identifier, "content_version_id": version, "sha256": observed.Hash, "format": observed.Detection.Format, "reused_job": reused})
	})
	return identifier, err
}
