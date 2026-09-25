package libraries

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

func verifiableRoot(root Root) bool { return root.Status == "active" || root.Status == "inaccessible" }
func rootJobKind(root Root) string {
	if root.Source == "managed" {
		return "verify_managed"
	}
	return "scan"
}
func enqueueVerification(ctx context.Context, transaction *sql.Tx, root Root) error {
	if root.Source != "managed" {
		return enqueueScan(ctx, transaction, root.LibraryID, root.ID)
	}
	identifier := domain.NewID()
	_, err := transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,'verify_managed',?,?,'{}','queued',?,?)", identifier, root.LibraryID, root.ID, identifier, now(), now())
	return err
}

// Verify only recorded managed locations; unrelated files in a destination are
// never imported, and private uploads do not participate in root reconciliation.
func (service *Service) VerifyManagedRoot(ctx context.Context, rootID string) error {
	if err := licensing.CheckFeatures("managed_libraries"); err != nil {
		return err
	}
	root, err := service.root(ctx, rootID)
	if err != nil {
		return err
	}
	if root.Source != "managed" || !verifiableRoot(root) {
		return scanFailure("ROOT_DISABLED")
	}
	_, mode, err := settingsFor(ctx, service.Database.Reader, root.LibraryID)
	if err != nil {
		return err
	}
	if err = licensing.CheckFeatures(modeCapabilities(mode)...); err != nil {
		return err
	}
	inspected, err := inspectDirectory(root.Path)
	if err != nil || inspected.Identity != root.Identity {
		return service.scanError(ctx, root, scanFailure("ROOT_UNAVAILABLE"))
	}
	after := ""
	for {
		rows, err := service.Database.Reader.QueryContext(ctx, "SELECT f.id,l.relative_path FROM physical_files f JOIN physical_file_locations l ON l.id=f.primary_location_id JOIN documents d ON d.physical_file_id=f.id WHERE l.root_id=? AND f.storage_source='managed' AND d.deleted_at IS NULL AND f.id>? ORDER BY f.id LIMIT 100", root.ID, after)
		if err != nil {
			return err
		}
		type target struct{ ID, Relative string }
		targets := []target{}
		for rows.Next() {
			var item target
			if err = rows.Scan(&item.ID, &item.Relative); err != nil {
				rows.Close()
				return err
			}
			targets = append(targets, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			break
		}
		for _, item := range targets {
			if err = service.verifyManagedFile(ctx, root, item.ID, item.Relative); err != nil {
				return service.scanError(ctx, root, err)
			}
			after = item.ID
		}
	}
	inspected, err = inspectDirectory(root.Path)
	if err != nil || inspected.Identity != root.Identity {
		return service.scanError(ctx, root, scanFailure("ROOT_UNAVAILABLE"))
	}
	_, err = service.Database.Writer.ExecContext(ctx, "UPDATE storage_roots SET status='active',last_verified_at=?,last_scan_at=?,last_error_code='' WHERE id=? AND configuration_revision=?", now(), now(), root.ID, root.Revision)
	return err
}
func (service *Service) verifyManagedFile(ctx context.Context, root Root, fileID, relative string) error {
	file, err := openLinked(root, relative)
	if err != nil {
		// An inaccessible mount/directory is never evidence of deletion.
		inspected, rootErr := inspectDirectory(root.Path)
		if rootErr != nil || inspected.Identity != root.Identity {
			return scanFailure("ROOT_UNAVAILABLE")
		}
		if !os.IsNotExist(err) {
			return scanFailure("SOURCE_UNAVAILABLE")
		}
		return service.Database.Write(ctx, func(transaction *sql.Tx) error {
			if err := checkRootRevision(ctx, transaction, root); err != nil {
				return err
			}
			var documentID, versionID, availability string
			if err := transaction.QueryRowContext(ctx, "SELECT d.id,f.current_content_version_id,f.availability FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE f.id=?", fileID).Scan(&documentID, &versionID, &availability); err != nil {
				return err
			}
			if availability == "missing" {
				return nil
			}
			if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET availability='missing' WHERE id=?", fileID); err != nil {
				return err
			}
			return managedObservation(ctx, transaction, root.LibraryID, documentID, versionID, "managed_missing")
		})
	}
	defer file.Close()
	digest, size, err := hashFile(ctx, file, int64(service.Identity.Config.Indexing.MaximumFileMB)<<20)
	if err != nil {
		return err
	}
	identity, _, err := physicalIdentity(file)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if err := checkRootRevision(ctx, transaction, root); err != nil {
			return err
		}
		var documentID, versionID, oldHash, oldIdentity, availability, approval string
		if err := transaction.QueryRowContext(ctx, "SELECT d.id,f.current_content_version_id,v.sha256,coalesce(f.os_identity_key,''),f.availability,d.approval_status FROM physical_files f JOIN documents d ON d.physical_file_id=f.id JOIN content_versions v ON v.id=f.current_content_version_id WHERE f.id=?", fileID).Scan(&documentID, &versionID, &oldHash, &oldIdentity, &availability, &approval); err != nil {
			return err
		}
		if oldHash == digest && oldIdentity == identity {
			if availability == "missing" {
				if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET availability='available' WHERE id=?", fileID); err != nil {
					return err
				}
				return managedObservation(ctx, transaction, root.LibraryID, documentID, versionID, "managed_reappeared")
			}
			return nil
		}
		newVersion := domain.NewID()
		var generation int
		if err := transaction.QueryRowContext(ctx, "SELECT coalesce(max(generation),0)+1 FROM content_versions WHERE physical_file_id=?", fileID).Scan(&generation); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,sha256,observed_change_key,observed_at,change_origin) VALUES(?,?,?,?,?,?,?,?,'external')", newVersion, fileID, generation, size, domain.Timestamp(info.ModTime()), digest, newVersion, now()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE physical_files SET current_content_version_id=?,os_identity_key=?,availability='available',integrity_status='changed',extraction_freshness=CASE WHEN indexed_extraction_id IS NULL THEN 'none' ELSE 'stale' END WHERE id=?", newVersion, identity, fileID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE documents SET approval_status=CASE WHEN approval_status='approved' THEN 'needs_review' ELSE approval_status END WHERE id=?", documentID); err != nil {
			return err
		}
		jobID := domain.NewID()
		if _, err := transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','queued',?,?)", jobID, root.LibraryID, fileID, newVersion, "extract:"+newVersion, now(), now()); err != nil {
			return err
		}
		return managedObservation(ctx, transaction, root.LibraryID, documentID, newVersion, "managed_changed")
	})
}
func managedObservation(ctx context.Context, transaction *sql.Tx, libraryID, documentID, versionID, kind string) error {
	eventID := domain.NewID()
	if err := appendDocumentObservation(ctx, transaction, eventID, "document."+kind, libraryID, documentID, versionID); err != nil {
		return err
	}
	_, err := transaction.ExecContext(ctx, "INSERT INTO document_notifications VALUES(?,?,?,?,?,?,?)", domain.NewID(), documentID, versionID, eventID, kind, eventID, now())
	return err
}
func (service *Service) cleanExpiredUploads(ctx context.Context) error {
	if err := licensing.CheckFeatures("managed_libraries"); err != nil {
		return err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT u.id,u.document_id,u.private_temporary_locator,d.library_id,coalesce(d.case_id,'') FROM upload_items u JOIN documents d ON d.id=u.document_id JOIN physical_files f ON f.id=d.physical_file_id WHERE u.retain_until IS NOT NULL AND u.retain_until<=? AND u.status IN ('staged','retained') AND d.approval_status IN ('rejected','cancelled') AND f.primary_location_id IS NULL LIMIT 100", now())
	if err != nil {
		return err
	}
	type expired struct{ ID, DocumentID, Locator, LibraryID, CaseID string }
	items := []expired{}
	for rows.Next() {
		var item expired
		if err = rows.Scan(&item.ID, &item.DocumentID, &item.Locator, &item.LibraryID, &item.CaseID); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range items {
		if !relativeSafe(item.Locator) || strings.ContainsAny(item.Locator, "/\\") {
			return privateUploadError()
		}
		// Hold the writer while rechecking and unlinking this one owned filename;
		// submission/cancellation cannot race retention and the operation is idempotent.
		err = service.Database.Write(ctx, func(transaction *sql.Tx) error {
			_, mode, err := settingsFor(ctx, transaction, item.LibraryID)
			if err != nil {
				return err
			}
			if err = licensing.CheckFeatures(modeCapabilities(mode)...); err != nil {
				return err
			}
			var eligible int
			if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM upload_items u JOIN documents d ON d.id=u.document_id JOIN physical_files f ON f.id=d.physical_file_id WHERE f.primary_location_id IS NULL AND u.id=? AND u.retain_until<=? AND u.status IN ('staged','retained') AND d.approval_status IN ('rejected','cancelled')", item.ID, now()).Scan(&eligible); err != nil {
				return err
			}
			if eligible == 0 {
				return nil
			}
			temporary, err := os.OpenRoot(service.Identity.Config.PrivateUploadDirectory())
			if err != nil {
				return privateUploadError()
			}
			defer temporary.Close()
			if err = temporary.Remove(item.Locator); err != nil && !os.IsNotExist(err) {
				return privateUploadError()
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE upload_items SET status='cleaned' WHERE id=?", item.ID); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_files SET availability='missing' WHERE id=(SELECT physical_file_id FROM documents WHERE id=?)", item.DocumentID); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE documents SET revision=revision+1 WHERE id=?", item.DocumentID); err != nil {
				return err
			}
			return workflowEvent(ctx, transaction, domain.Principal{}, Document{ID: item.DocumentID, LibraryID: item.LibraryID, CaseID: item.CaseID}, domain.RequestMetadata{}, "document.temporary_cleaned", map[string]any{"reason": "retention_expired"})
		})
		if err != nil {
			return err
		}
	}
	return nil
}
