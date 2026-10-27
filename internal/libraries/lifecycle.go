package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

func lifecycleBusy() error {
	return domain.Failure("ENTITY_BUSY", "Hay cargas, recorridos, vistas previas o guardados definitivos pendientes de terminar. Revisa sus diagnósticos en Procesamiento; espera a que terminen, reintenta los fallidos o cancela los trabajos que lo permitan antes de continuar.", 409)
}
func (s *Service) lifecycleAccess(ctx context.Context, q storage.Querier, p domain.Principal, library string, deleting bool) error {
	if deleting {
		if !p.Can("libraries.delete") {
			return domain.Failure("FORBIDDEN", "Solo un Administrador de instalación puede eliminar bibliotecas definitivamente.", 403)
		}
	} else if !p.Can("permissions.manage_global") {
		var n int
		if err := q.QueryRowContext(ctx, "SELECT count(*) FROM library_role_assignments a JOIN role_permissions r USING(role_id) WHERE library_id=? AND user_id=? AND permission_key='libraries.configure'", library, p.User.ID).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return notFound()
		}
	}
	return s.Identity.License.Check(ctx, licensing.WriteDocuments)
}
func activeLibrary(ctx context.Context, q storage.Querier, id string) error {
	var archived bool
	if err := q.QueryRowContext(ctx, "SELECT disabled_at IS NOT NULL FROM libraries WHERE id=?", id).Scan(&archived); err == sql.ErrNoRows {
		return notFound()
	} else if err != nil {
		return err
	}
	if archived {
		return domain.Failure("LIBRARY_ARCHIVED", "La biblioteca está archivada. Restáurala antes de realizar operaciones.", 409)
	}
	return nil
}
func (s *Service) libraryBusy(ctx context.Context, q storage.Querier, library string) error {
	var n int
	err := q.QueryRowContext(ctx, `SELECT
 (SELECT count(*) FROM jobs WHERE library_id=? AND status='running')+
 (SELECT count(*) FROM upload_items WHERE library_id=? AND status='receiving')+
 (SELECT count(*) FROM preview_cache p JOIN documents d ON d.id=p.document_id WHERE d.library_id=? AND p.status='generating')+
 (SELECT count(*) FROM root_scans s JOIN storage_roots r ON r.id=s.root_id WHERE r.library_id=? AND s.status='running')+
 (SELECT count(*) FROM materializations WHERE library_id=? AND state NOT IN ('cleaned'))`, library, library, library, library, library).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return lifecycleBusy()
	}
	return nil
}
func (s *Service) ArchiveLibrary(ctx context.Context, p domain.Principal, id string, revision int64, restore bool, meta domain.RequestMetadata) error {
	return s.Identity.AuthorizedWrite(ctx, p, "", func(tx *sql.Tx, current domain.Principal) error {
		if err := s.lifecycleAccess(ctx, tx, current, id, false); err != nil {
			return err
		}
		var archived bool
		var live int64
		var name, mode string
		if err := tx.QueryRowContext(ctx, "SELECT name,mode,revision,disabled_at IS NOT NULL FROM libraries WHERE id=?", id).Scan(&name, &mode, &live, &archived); err == sql.ErrNoRows {
			return notFound()
		} else if err != nil {
			return err
		}
		if live != revision {
			return conflict()
		}
		if restore != archived {
			return domain.Failure("LIBRARY_STATE_CHANGED", "El estado de la biblioteca cambió. Actualiza el listado.", 409)
		}
		if err := s.Identity.License.CheckFeatures(ctx, modeCapabilities(mode)...); err != nil {
			return err
		}
		if err := s.libraryBusy(ctx, tx, id); err != nil {
			return err
		}
		if restore {
			var raw string
			var settings Settings
			if err := tx.QueryRowContext(ctx, "SELECT settings_json FROM libraries WHERE id=?", id).Scan(&raw); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(raw), &settings); err != nil {
				return err
			}
			if err := validSettings(settings); err != nil {
				return err
			}
			roots, err := rootsQuery(ctx, tx, id)
			if err != nil {
				return err
			}
			for _, root := range roots {
				if root.Status == "retired" || root.Status == "superseded" {
					continue
				}
				dir, _, related, err := s.inspect(ctx, tx, id, root.Path)
				if err != nil {
					return err
				}
				if dir.Identity != root.Identity {
					return domain.Failure("ROOT_IDENTITY_CHANGED", "La carpeta ya no es la misma que se registró. Revisa su ubicación antes de restaurar la biblioteca.", 409)
				}
				for _, other := range related {
					if other != root.ID {
						return domain.Failure("ROOT_STORAGE_OVERLAP", "Hay carpetas superpuestas que deben resolverse antes de restaurar.", 409)
					}
				}
				if root.Source == "managed" {
					if err := probeManagedRoot(root.Path); err != nil {
						return err
					}
				}
			}
		}
		action := "library.archived"
		var disabled any = now()
		if restore {
			disabled = nil
			action = "library.restored"
		}
		if _, err := tx.ExecContext(ctx, "UPDATE libraries SET disabled_at=?,revision=revision+1,updated_at=? WHERE id=?", disabled, now(), id); err != nil {
			return err
		}
		if restore {
			if err := s.queueNewlyIndexable(ctx, tx, id); err != nil {
				return err
			}
			roots, err := rootsQuery(ctx, tx, id)
			if err != nil {
				return err
			}
			for _, root := range roots {
				if verifiableRoot(root) {
					if err := enqueueVerification(ctx, tx, root); err != nil {
						return err
					}
				}
			}
		}
		return record(ctx, tx, current, meta, action, id, "", map[string]any{"name": name, "documents_preserved": true})
	})
}
func (s *Service) ResumeRoot(ctx context.Context, p domain.Principal, id string, revision int64, meta domain.RequestMetadata) error {
	root, err := s.root(ctx, id)
	if err != nil {
		return err
	}
	return s.write(ctx, p, root.LibraryID, "storage.manage_roots", func(tx *sql.Tx, current domain.Principal) error {
		live, err := scanRoot(tx.QueryRowContext(ctx, "SELECT "+rootColumns+" FROM storage_roots WHERE id=?", id))
		if err != nil {
			return err
		}
		if live.Revision != revision {
			return conflict()
		}
		if live.Status != "retired" {
			return invalid("La carpeta no está retirada. Actualiza su estado.")
		}
		dir, _, related, err := s.inspect(ctx, tx, live.LibraryID, live.Path)
		if err != nil {
			return err
		}
		for _, other := range related {
			if other != id {
				return domain.Failure("ROOT_STORAGE_OVERLAP", "La carpeta entra en conflicto con otra raíz. Resuelve la superposición antes de reanudar.", 409)
			}
		}
		if dir.Identity != live.Identity {
			return domain.Failure("ROOT_IDENTITY_CHANGED", "La ubicación corresponde a una carpeta distinta de la registrada. No se reanudó la supervisión.", 409)
		}
		if live.Source == "managed" {
			if err = probeManagedRoot(live.Path); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE storage_roots SET status='active',watch_mode='polling',configuration_revision=configuration_revision+1,last_error_code='',last_scan_at=NULL WHERE id=?", id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM root_scan_controls WHERE root_id=?", id); err != nil {
			return err
		}
		// Restore only the references this retirement removed, never individual withdrawals.
		// Pre-phase-1 retirements have no per-document marker. Recover only records
		// supported by the completed retirement plan and its audit event; an
		// individual withdrawal always wins. Ambiguous records remain withdrawn.
		if _, err = tx.ExecContext(ctx, `INSERT INTO root_retired_documents(root_id,document_id,removed_revision)
 SELECT ?,d.id,d.revision FROM documents d WHERE d.deleted_at IS NOT NULL
 AND EXISTS(SELECT 1 FROM physical_file_locations l WHERE l.physical_file_id=d.physical_file_id AND l.root_id=? AND l.retired_at IS NULL)
 AND NOT EXISTS(SELECT 1 FROM audit_events a WHERE a.document_id=d.id AND a.event_type='document.index_removed')
 AND EXISTS(SELECT 1 FROM root_operations op WHERE op.library_id=d.library_id AND op.operation_kind='retire' AND op.state='complete' AND json_extract(op.private_plan_json,'$.root_id')=? AND d.deleted_at>=op.created_at AND d.deleted_at<=op.updated_at)
 AND (SELECT json_extract(a.details_json,'$.reference_policy') FROM audit_events a WHERE a.event_type='storage.root_retired' AND json_extract(a.details_json,'$.root_id')=? ORDER BY a.occurred_at DESC,a.id DESC LIMIT 1)='remove_index_references'
 ON CONFLICT(root_id,document_id) DO NOTHING`, id, id, id, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE documents SET deleted_at=NULL,revision=revision+1 WHERE id IN (SELECT document_id FROM root_retired_documents WHERE root_id=? AND removed_revision=documents.revision)`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO indexed_pages(physical_file_id,extraction_id,page_number,page_text) SELECT f.id,e.extraction_id,e.page_number,e.page_text FROM physical_files f JOIN documents d ON d.physical_file_id=f.id JOIN extraction_pages e ON e.extraction_id=f.indexed_extraction_id WHERE d.id IN (SELECT document_id FROM root_retired_documents WHERE root_id=?) AND d.deleted_at IS NULL ON CONFLICT(physical_file_id,page_number) DO NOTHING`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM root_retired_documents WHERE root_id=?", id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE libraries SET revision=revision+1,current_configuration_revision=current_configuration_revision+1 WHERE id=?", root.LibraryID); err != nil {
			return err
		}
		if err = configurationSnapshot(ctx, tx, root.LibraryID, current.User.ID); err != nil {
			return err
		}
		live.Status = "active"
		if err = enqueueVerification(ctx, tx, live); err != nil {
			return err
		}
		return record(ctx, tx, current, meta, "storage.root_resumed", root.LibraryID, "", map[string]any{"root_id": id, "verification": "queued"})
	})
}
