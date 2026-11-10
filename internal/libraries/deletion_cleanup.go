package libraries

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type deletionUpload struct {
	ID, DocumentID, Locator, Hash, Identity string
	Cleanable                               bool
}

func (s *Service) deletionUploads(ctx context.Context, q storage.Querier, root bool, id string) ([]deletionUpload, error) {
	selector := "u.library_id=?"
	args := []any{id}
	if root {
		selector = "u.document_id IN (" + deletionDocs(root) + ")"
		args = scopeArgs(root, id)
	}
	rows, err := q.QueryContext(ctx, `SELECT u.id,coalesce(u.document_id,''),u.private_temporary_locator,u.sha256,coalesce(f.os_identity_key,''),
 (u.status<>'receiving' AND NOT EXISTS(SELECT 1 FROM physical_file_locations WHERE physical_file_id=f.id)
 AND NOT EXISTS(SELECT 1 FROM materializations WHERE document_id=d.id)
 AND (d.id IS NULL OR (d.approval_status IN ('draft','cancelled','rejected') AND d.case_id IS NULL AND d.category_id IS NULL AND d.document_type_id IS NULL AND d.title='' AND d.metadata_json='{}' AND d.approved_content_version_id IS NULL)))
 FROM upload_items u LEFT JOIN documents d ON d.id=u.document_id LEFT JOIN physical_files f ON f.id=d.physical_file_id WHERE `+selector+` ORDER BY u.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []deletionUpload{}
	for rows.Next() {
		var item deletionUpload
		if err = rows.Scan(&item.ID, &item.DocumentID, &item.Locator, &item.Hash, &item.Identity, &item.Cleanable); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Service) journalDeletionFiles(ctx context.Context, tx *sql.Tx, p DeletionPlan, prepare bool) error {
	root := p.Kind == "root"
	args := scopeArgs(root, p.EntityID)
	manifestDocs, manifestArgs := deletionDocs(root), args
	if root {
		manifestDocs = "SELECT id FROM documents WHERE physical_file_id IN (SELECT physical_file_id FROM physical_file_locations WHERE root_id=?)"
		manifestArgs = []any{p.EntityID}
	}
	// SQLite streams the manifest directly; even large libraries do not require
	// a second in-memory copy of all document paths.
	_, err := tx.ExecContext(ctx, `INSERT INTO deletion_documents(operation_id,document_id,filename,sha256,locations_json) SELECT ?,d.id,d.original_filename,coalesce(v.sha256,''),coalesce((SELECT json_group_array(canonical_path) FROM physical_file_locations WHERE physical_file_id=f.id),'[]') FROM documents d JOIN physical_files f ON f.id=d.physical_file_id LEFT JOIN content_versions v ON v.id=f.current_content_version_id WHERE d.id IN (`+manifestDocs+`)`, append([]any{p.ID}, manifestArgs...)...)
	if err != nil {
		return err
	}
	uploads, err := s.deletionUploads(ctx, tx, root, p.EntityID)
	if err != nil {
		return err
	}
	privatePaths := map[string]string{}
	for _, u := range uploads {
		// Files published in a managed destination may have a now-unused private copy.
		// Preserve any ambiguous copy: publishing must have cleaned it ordinarily.
		if !u.Cleanable {
			privatePaths[u.DocumentID] = filepath.Join(s.Identity.Config.PrivateUploadDirectory(), u.Locator)
			continue
		}
		if !prepare {
			continue
		}
		if err = s.prepareDeletionCleanup(ctx, tx, p.ID, "upload", u.Locator, u.ID+".document", u.Identity, u.Hash); err != nil {
			return err
		}
		// A failed upload may leave its individually owned partial transfer.
		if u.DocumentID == "" && u.Locator == u.ID+".document" {
			if err = s.prepareDeletionCleanup(ctx, tx, p.ID, "upload", u.Locator+".part", u.ID+".document.part", "", ""); err != nil {
				return err
			}
		}
	}
	for doc, path := range privatePaths {
		if _, err = tx.ExecContext(ctx, "UPDATE deletion_documents SET locations_json=json_insert(locations_json,'$[#]',?) WHERE operation_id=? AND document_id=?", path, p.ID, doc); err != nil {
			return err
		}
	}
	if !prepare {
		return nil
	}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM preview_cache WHERE document_id IN ("+deletionDocs(root)+")", args...)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !prepare {
		return nil
	}
	for _, id := range ids {
		if err = s.prepareDeletionCleanup(ctx, tx, p.ID, "preview", id+".preview", id+".preview", "", ""); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) cleanupDirectory(kind string) string {
	if kind == "upload" {
		return s.Identity.Config.PrivateUploadDirectory()
	}
	return s.previewDirectory()
}

// Never resolve a substituted component, including Windows reparse points.
func privateCleanupRoot(path string) (*os.Root, string, error) {
	if !filepath.IsAbs(path) {
		return nil, "", scanFailure("CLEANUP_UNSAFE_PATH")
	}
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return nil, "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, "", scanFailure("CLEANUP_UNSAFE_PATH")
		}
		handle, err := os.Open(current)
		if err != nil {
			return nil, "", err
		}
		_, _, err = physicalIdentity(handle)
		handle.Close()
		if err != nil {
			return nil, "", scanFailure("CLEANUP_UNSAFE_PATH")
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	container, err := os.OpenRoot(path)
	if err != nil {
		return nil, "", err
	}
	file, err := container.Open(".")
	if err != nil {
		container.Close()
		return nil, "", err
	}
	identity, _, err := physicalIdentity(file)
	file.Close()
	if err != nil {
		container.Close()
		return nil, "", err
	}
	return container, identity, nil
}
func cleanupFile(container *os.Root, name string) (*os.File, string, error) {
	if !relativeSafe(name) || strings.ContainsAny(name, "/\\") {
		return nil, "", scanFailure("CLEANUP_UNSAFE_PATH")
	}
	info, err := container.Lstat(name)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", scanFailure("CLEANUP_UNSAFE_PATH")
	}
	file, err := container.Open(name)
	if err != nil {
		return nil, "", err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(info, after) {
		file.Close()
		return nil, "", scanFailure("CLEANUP_CHANGED")
	}
	identity, _, err := physicalIdentity(file)
	if err != nil {
		file.Close()
		return nil, "", err
	}
	return file, identity, nil
}
func (s *Service) prepareDeletionCleanup(ctx context.Context, tx *sql.Tx, op, kind, locator, expected, expectedID, expectedHash string) error {
	id := domain.NewID()
	path := s.cleanupDirectory(kind)
	directoryID, fileID, digest := "", "", ""
	var size int64
	status, code := "pending", ""
	if locator != expected || !relativeSafe(locator) || strings.ContainsAny(locator, "/\\") {
		status, code = "attention", "CLEANUP_UNSAFE_PATH"
	} else {
		roots, err := rootsQuery(ctx, tx, "")
		if err != nil {
			return err
		}
		for _, root := range roots {
			if within(root.Path, path, root.CaseSensitive) || within(path, root.Path, root.CaseSensitive) {
				status, code = "attention", "CLEANUP_UNSAFE_PATH"
				break
			}
		}
		if status == "pending" {
			container, identity, err := privateCleanupRoot(path)
			if os.IsNotExist(err) {
				status = "complete"
			} else if err != nil {
				status, code = "attention", "CLEANUP_STORAGE_UNAVAILABLE"
			} else {
				directoryID = identity
				file, identity, err := cleanupFile(container, locator)
				if os.IsNotExist(err) {
					status = "complete"
				} else if err != nil {
					status, code = "attention", "CLEANUP_UNSAFE_PATH"
				} else {
					fileID = identity
					info, err := file.Stat()
					if err == nil {
						digest, size, err = hashFile(ctx, file, info.Size()+1)
					}
					file.Close()
					if err != nil {
						status, code = "attention", "CLEANUP_STORAGE_UNAVAILABLE"
					} else if (expectedID != "" && expectedID != fileID) || (expectedHash != "" && expectedHash != digest) {
						status, code = "attention", "CLEANUP_CHANGED"
						fileID = ""
					}
				}
				container.Close()
			}
		}
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO deletion_cleanup(id,operation_id,kind,locator,container_path,container_identity,file_identity,sha256,size_bytes,quarantine,status,error_code) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", id, op, kind, locator, path, directoryID, fileID, digest, size, "cleanup-"+id+".pending", status, code)
	return err
}

// A durable, exact-file cleanup journal is committed with metadata deletion.
// Cleanup never enumerates or removes a documentary directory. A private random
// quarantine name separates removal from names that normal operations can use.
func (s *Service) cleanDeletionFiles(ctx context.Context, operation string, limit int) error {
	rows, err := s.Database.Reader.QueryContext(ctx, "SELECT id FROM deletion_cleanup WHERE status<>'complete' AND error_code<>'CLEANUP_RESTORE_REVIEW' AND (?='' OR operation_id=?) AND operation_id IN(SELECT id FROM entity_deletions WHERE state='cleanup_pending') ORDER BY attempts,id LIMIT ?", operation, operation, limit)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = s.Database.Write(ctx, func(tx *sql.Tx) error {
			var item struct {
				op, kind, locator, path, dirID, fileID, hash, quarantine, status string
				size                                                             int64
			}
			if err := tx.QueryRowContext(ctx, "SELECT operation_id,kind,locator,container_path,container_identity,file_identity,sha256,size_bytes,quarantine,status FROM deletion_cleanup WHERE id=?", id).Scan(&item.op, &item.kind, &item.locator, &item.path, &item.dirID, &item.fileID, &item.hash, &item.size, &item.quarantine, &item.status); err != nil {
				return err
			}
			if item.status == "complete" {
				return nil
			}
			code := ""
			container, dirID, openErr := privateCleanupRoot(item.path)
			if openErr != nil {
				code = "CLEANUP_STORAGE_UNAVAILABLE"
			} else {
				defer container.Close()
				if item.dirID == "" || dirID != item.dirID || item.fileID == "" || len(item.hash) != 64 || item.path != s.cleanupDirectory(item.kind) {
					code = "CLEANUP_CHANGED"
				}
				if code == "" {
					var referenced int
					if err := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM physical_files WHERE os_identity_key=?)+(SELECT count(*) FROM upload_items WHERE private_temporary_locator=? OR private_temporary_locator||'.part'=?)+(SELECT count(*) FROM preview_cache WHERE id=?)+(SELECT count(*) FROM deletion_documents d JOIN entity_deletions op ON op.id=d.operation_id,json_each(d.locations_json) p WHERE op.state<>'planned' AND p.value=?)`, item.fileID, item.locator, item.locator, strings.TrimSuffix(item.locator, ".preview"), filepath.Join(item.path, item.locator)).Scan(&referenced); err != nil {
						return err
					}
					if referenced > 0 {
						code = "CLEANUP_REFERENCED"
					}
				}
				if code == "" {
					name := item.quarantine
					file, identity, err := cleanupFile(container, name)
					if os.IsNotExist(err) {
						name = item.locator
						file, identity, err = cleanupFile(container, name)
					}
					if os.IsNotExist(err) { /* Already removed: completion may have been interrupted. */
					} else if err != nil {
						code = "CLEANUP_UNSAFE_PATH"
					} else {
						hash, size, hashErr := hashFile(ctx, file, item.size+1)
						file.Close()
						if hashErr != nil {
							code = "CLEANUP_STORAGE_UNAVAILABLE"
						} else if identity != item.fileID || hash != item.hash || size != item.size {
							code = "CLEANUP_CHANGED"
						} else {
							if name == item.locator {
								if _, err = container.Lstat(item.quarantine); !os.IsNotExist(err) {
									code = "CLEANUP_CHANGED"
								} else if err = container.Rename(name, item.quarantine); err != nil {
									code = "CLEANUP_STORAGE_UNAVAILABLE"
								}
							}
							if code == "" {
								file, identity, err = cleanupFile(container, item.quarantine)
								if err != nil {
									code = "CLEANUP_STORAGE_UNAVAILABLE"
								} else {
									hash, size, err = hashFile(ctx, file, item.size+1)
									file.Close()
									if err != nil || identity != item.fileID || hash != item.hash || size != item.size {
										code = "CLEANUP_CHANGED"
									} else if err = container.Remove(item.quarantine); err != nil && !os.IsNotExist(err) {
										code = "CLEANUP_STORAGE_UNAVAILABLE"
									}
								}
							}
						}
					}
				}
			}
			state := "complete"
			if code != "" {
				state = "attention"
			}
			if _, err := tx.ExecContext(ctx, "UPDATE deletion_cleanup SET status=?,error_code=?,attempts=attempts+1 WHERE id=?", state, code, id); err != nil {
				return err
			}
			if code != "" {
				return diagnostics.RecordTx(ctx, tx, "system", code, diagnostics.Context{Operation: "selective_cleanup", OperationID: item.op})
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	_, err = s.Database.Writer.ExecContext(ctx, "UPDATE entity_deletions SET state='complete',completed_at=? WHERE state='cleanup_pending' AND NOT EXISTS(SELECT 1 FROM deletion_cleanup WHERE operation_id=entity_deletions.id AND status<>'complete')", now())
	return err
}
