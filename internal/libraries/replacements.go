package libraries

import (
	"context"
	"database/sql"
	"strings"

	"gestor-documental/internal/domain"
)

// RegisterReplacement enlaza una carga nueva con el documento incorporado.
// Cada versión conserva su archivo e identidad; nunca se copia sobre el original.
func (service *Service) RegisterReplacement(ctx context.Context, p domain.Principal, previousID, newID, reason string, revision int64, meta domain.RequestMetadata) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1000 || previousID == newID {
		return invalid("Selecciona una carga nueva e indica el motivo de sustitución.")
	}
	document, err := service.Document(ctx, p, previousID)
	if err != nil {
		return err
	}
	return service.write(ctx, p, document.LibraryID, "documents.replace", func(tx *sql.Tx, current domain.Principal) error {
		old, err := workflowDocument(ctx, tx, current, previousID)
		if err != nil {
			return err
		}
		next, err := workflowDocument(ctx, tx, current, newID)
		if err != nil {
			return err
		}
		if old.Revision != revision {
			return conflict()
		}
		if old.Approval != "approved" || old.Source != "managed" || next.LibraryID != old.LibraryID || next.Availability != "staged" || next.Approval != "draft" || next.CreatedBy != current.User.ID || next.CaseID != "" || next.CategoryID != "" || next.TypeID != "" {
			return invalid("Utiliza una carga propia nueva, sin clasificar, de la misma biblioteca para sustituir un documento administrado aprobado.")
		}
		var pending int
		if err = tx.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM document_replacements WHERE (previous_document_id=? OR previous_document_id=?) AND status<>'cancelled')+(SELECT count(*) FROM document_replacements WHERE new_document_id=?)+(SELECT count(*) FROM document_corrections WHERE document_id=? AND operation<>'request' AND status='pending')", old.ID, next.ID, next.ID, old.ID).Scan(&pending); err != nil {
			return err
		}
		if pending > 0 || old.Classification == "historical" {
			return domain.Failure("REPLACEMENT_PENDING", "Ya hay una sustitución o corrección pendiente, o esta es una versión histórica.", 409)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO document_replacements(id,previous_document_id,previous_revision,new_document_id,requested_by,reason,previous_json,requested_at) VALUES(?,?,?,?,?,?,?,?)`, domain.NewID(), old.ID, old.Revision, next.ID, current.User.ID, reason, encode(classificationSnapshot(old)), now())
		if err != nil {
			return err
		}
		input := classificationSnapshot(old)
		input.Reason = reason
		if err = service.classifyTransaction(ctx, tx, current, next.ID, "classification", input, next.Revision, meta, false); err != nil {
			return err
		}
		return record(ctx, tx, current, meta, "document.replacement_requested", old.LibraryID, old.ID, map[string]any{"new_document_id": next.ID, "reason": reason, "previous": classificationSnapshot(old)})
	})
}

func (service *Service) completeReplacement(ctx context.Context, tx *sql.Tx, p domain.Principal, newID string) error {
	var previousID, reason, libraryID string
	var revision int64
	err := tx.QueryRowContext(ctx, "SELECT previous_document_id,previous_revision,reason FROM document_replacements WHERE new_document_id=? AND status='pending'", newID).Scan(&previousID, &revision, &reason)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var currentRevision int64
	var status string
	if err = tx.QueryRowContext(ctx, "SELECT revision,approval_status,library_id FROM documents WHERE id=? AND deleted_at IS NULL", previousID).Scan(&currentRevision, &status, &libraryID); err != nil {
		return err
	}
	if currentRevision != revision || status != "approved" {
		return domain.Failure("REPLACEMENT_STALE", "Cambió la versión anterior durante la sustitución; se conservan ambos archivos y se requiere revisión.", 409)
	}
	// La versión histórica conserva su aprobación física. Su antigua pertenencia
	// al expediente queda en previous_json y auditoría; solo la versión actual
	// cuenta para el cumplimiento, sin duplicar requisitos de un tipo único.
	if _, err = tx.ExecContext(ctx, "UPDATE documents SET case_id=NULL,revision=revision+1 WHERE id=?", previousID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE document_replacements SET status='completed',completed_at=?,authorized_by=? WHERE new_document_id=?", now(), p.User.ID, newID); err != nil {
		return err
	}
	return workflowEvent(ctx, tx, p, Document{ID: previousID, LibraryID: libraryID, Revision: revision + 1}, domain.RequestMetadata{}, "document.version_preserved", map[string]any{"new_document_id": newID, "reason": reason, "original_preserved": true})
}

type DocumentVersion struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Filename string `json:"original_filename"`
	Approval string `json:"approval_status"`
	Reason   string `json:"reason"`
	Status   string `json:"replacement_status"`
}

func (service *Service) DocumentVersions(ctx context.Context, p domain.Principal, documentID string) ([]DocumentVersion, error) {
	if _, err := service.Document(ctx, p, documentID); err != nil {
		return nil, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, `WITH RECURSIVE family(id) AS (
 SELECT ? UNION SELECT CASE WHEN r.previous_document_id=f.id THEN r.new_document_id ELSE r.previous_document_id END
 FROM document_replacements r JOIN family f ON r.previous_document_id=f.id OR r.new_document_id=f.id WHERE r.status IN ('pending','completed')
) SELECT d.id,coalesce(nullif(d.title,''),d.original_filename),d.original_filename,d.approval_status,coalesce(r.reason,''),coalesce(r.status,'')
 FROM family f JOIN documents d ON d.id=f.id LEFT JOIN document_replacements r ON r.new_document_id=d.id AND r.status IN ('pending','completed')
 WHERE d.deleted_at IS NULL ORDER BY d.created_at,d.id LIMIT 100`, documentID)
	if err != nil {
		return nil, err
	}
	items := []DocumentVersion{}
	for rows.Next() {
		var item DocumentVersion
		if err = rows.Scan(&item.ID, &item.Title, &item.Filename, &item.Approval, &item.Reason, &item.Status); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	// Revalidar permiso de cada versión: una relación no concede acceso implícito.
	allowed := []DocumentVersion{}
	for _, item := range items {
		if _, err := service.Document(ctx, p, item.ID); err == nil {
			allowed = append(allowed, item)
		}
	}
	return allowed, nil
}
