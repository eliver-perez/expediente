package libraries

import (
	"context"
	"database/sql"
	"strings"
	"time"
	"unicode/utf8"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
)

type Classification struct {
	CaseID     string            `json:"case_id"`
	CategoryID string            `json:"category_id"`
	TypeID     string            `json:"document_type_id"`
	Title      string            `json:"title"`
	Metadata   map[string]string `json:"metadata"`
	Reason     string            `json:"reason"`
}

func (service *Service) Classify(ctx context.Context, principal domain.Principal, documentID, operation string, input Classification, revision int64, metadata domain.RequestMetadata) error {
	input.Title = strings.TrimSpace(input.Title)
	if utf8.RuneCountInString(input.Title) > 250 || len(input.Metadata) > 20 {
		return invalid("Título o metadatos demasiado extensos.")
	}
	if input.Metadata == nil {
		input.Metadata = map[string]string{}
	}
	for key, value := range input.Metadata {
		if strings.TrimSpace(key) == "" || utf8.RuneCountInString(key) > 60 || utf8.RuneCountInString(value) > 500 {
			return invalid("Metadatos inválidos.")
		}
	}
	if len(encode(input.Metadata)) > 8192 {
		return invalid("Los metadatos superan el límite de 8 KiB.")
	}
	permission := "documents.classify"
	switch operation {
	case "associate":
		permission = "documents.associate"
	case "reassign":
		permission = "documents.reassign"
		if strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 1000 {
			return invalid("Indica el motivo de reasignación.")
		}
	case "classification":
	default:
		return invalid("Operación inválida.")
	}
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, document.LibraryID, permission, func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.requireManaged(ctx, transaction, document.LibraryID); err != nil {
			return err
		}
		document, err := scanDocument(transaction.QueryRowContext(ctx, "SELECT "+documentColumns+documentJoins+" WHERE d.id=? AND d.deleted_at IS NULL AND "+visibleDocumentSQL, documentID, current.User.ID, current.User.ID))
		if err == sql.ErrNoRows {
			return notFound()
		}
		if err != nil {
			return err
		}
		if document.Revision != revision {
			return conflict()
		}
		if document.Approval != "draft" && document.Approval != "rejected" && document.Approval != "pending_review" && document.Approval != "needs_review" {
			return invalid("No se puede editar un documento aprobado o en materialización.")
		}
		if operation == "associate" && (document.Source != "linked" || document.CaseID != "" || input.CaseID == "") {
			return domain.Failure("DOCUMENT_ALREADY_ASSOCIATED", "Selecciona un archivo vinculado sin expediente y un expediente de destino.", 409)
		}
		if operation == "reassign" && (document.CaseID == "" || input.CaseID == "" || input.CaseID == document.CaseID) {
			return invalid("Selecciona otro expediente para reasignar.")
		}
		if operation == "classification" && ((document.CaseID != "" && input.CaseID != document.CaseID) || (document.Source == "linked" && input.CaseID != document.CaseID)) {
			return invalid("Utiliza Asociar o Reasignar para cambiar el expediente de este archivo.")
		}
		if input.CaseID != "" {
			if _, err := service.requireOrganization(ctx, transaction, document.LibraryID); err != nil {
				return err
			}
			var count int
			if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM cases WHERE id=? AND library_id=? AND deleted_at IS NULL", input.CaseID, document.LibraryID).Scan(&count); err != nil {
				return err
			}
			if count != 1 {
				return invalid("Selecciona un expediente de esta biblioteca.")
			}
			if input.CategoryID == "" || input.TypeID == "" {
				return invalid("El expediente requiere categoría y tipo de documento.")
			}
		}
		if input.CategoryID != "" {
			var count int
			if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM categories WHERE id=? AND library_id=? AND archived_at IS NULL", input.CategoryID, document.LibraryID).Scan(&count); err != nil {
				return err
			}
			if count != 1 {
				return invalid("Selecciona una categoría activa de esta biblioteca.")
			}
		}
		if input.TypeID != "" {
			var multiple, titleRequired bool
			if err = transaction.QueryRowContext(ctx, "SELECT allows_multiple,requires_descriptive_title FROM document_types WHERE id=? AND category_id=? AND library_id=? AND archived_at IS NULL", input.TypeID, input.CategoryID, document.LibraryID).Scan(&multiple, &titleRequired); err == sql.ErrNoRows {
				return invalid("Selecciona un tipo activo de la categoría elegida.")
			} else if err != nil {
				return err
			}
			if titleRequired && input.Title == "" {
				return invalid("Este tipo requiere un título descriptivo.")
			}
			if input.CaseID != "" {
				// A case owns its template snapshot; later catalog changes do not rewrite it.
				err = transaction.QueryRowContext(ctx, "SELECT allows_multiple FROM case_requirements WHERE case_id=? AND document_type_id=?", input.CaseID, input.TypeID).Scan(&multiple)
				if err != nil && err != sql.ErrNoRows {
					return err
				}
				if !multiple {
					var count int
					if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM documents WHERE case_id=? AND document_type_id=? AND id<>? AND deleted_at IS NULL AND approval_status<>'cancelled'", input.CaseID, input.TypeID, documentID).Scan(&count); err != nil {
						return err
					}
					if count > 0 {
						return domain.Failure("DOCUMENT_TYPE_OCCUPIED", "El expediente ya tiene un archivo de este tipo único.", 409)
					}
				}
			}
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE documents SET case_id=?,category_id=?,document_type_id=?,title=?,metadata_json=?,created_by=coalesce(created_by,?),approval_status='draft',revision=revision+1 WHERE id=?", nullableID(input.CaseID), nullableID(input.CategoryID), nullableID(input.TypeID), input.Title, encode(input.Metadata), current.User.ID, documentID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE review_requests SET status='superseded',decided_at=? WHERE document_id=? AND status='pending'", now(), documentID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE upload_items SET retain_until=NULL WHERE document_id=?", documentID); err != nil {
			return err
		}
		eventID := domain.NewID()
		details := map[string]any{"previous_case_id": document.CaseID, "case_id": input.CaseID, "category_id": input.CategoryID, "document_type_id": input.TypeID, "title": input.Title, "reason": input.Reason}
		if err = audit.Append(ctx, transaction, time.Now(), audit.Event{ID: eventID, Type: "document." + operation, ActorUserID: current.User.ID, SessionID: current.SessionID, LibraryID: document.LibraryID, CaseID: input.CaseID, DocumentID: documentID, Metadata: metadata, Details: details}); err != nil {
			return err
		}
		_, err = transaction.ExecContext(ctx, "INSERT INTO document_history VALUES(?,?,?,?,?)", domain.NewID(), documentID, revision+1, eventID, encode(details))
		return err
	})
}
func (service *Service) CancelUpload(ctx context.Context, principal domain.Principal, documentID, reason string, revision int64, metadata domain.RequestMetadata) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return invalid("Indica el motivo de cancelación.")
	}
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, document.LibraryID, "documents.cancel_own", func(transaction *sql.Tx, current domain.Principal) error {
		var owner, status, availability string
		var version int64
		if err := transaction.QueryRowContext(ctx, "SELECT d.created_by,d.approval_status,f.availability,d.revision FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=?", documentID).Scan(&owner, &status, &availability, &version); err != nil {
			return err
		}
		if owner != current.User.ID || availability != "staged" {
			return notFound()
		}
		if version != revision {
			return conflict()
		}
		if status != "draft" && status != "rejected" && status != "pending_review" {
			return invalid("Solo puedes cancelar cargas en borrador, rechazadas o pendientes de revisión.")
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE documents SET approval_status='cancelled',revision=revision+1 WHERE id=?", documentID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE upload_items SET status='retained' WHERE document_id=?", documentID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE jobs SET status='cancelled',fencing_token=fencing_token+1 WHERE physical_file_id=? AND status IN ('queued','running','retry_wait')", document.FileID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE review_requests SET status='cancelled',decided_at=?,reason=? WHERE document_id=? AND status='pending'", now(), reason, documentID); err != nil {
			return err
		}
		settings, _, err := settingsFor(ctx, transaction, document.LibraryID)
		if err != nil {
			return err
		}
		if err = setUploadRetention(ctx, transaction, documentID, settings.RetentionDays); err != nil {
			return err
		}
		return workflowEvent(ctx, transaction, current, document, metadata, "document.cancelled", map[string]any{"reason": reason, "retention_days": settings.RetentionDays})
	})
}
