package libraries

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

type Review struct {
	ID          string `json:"id"`
	DocumentID  string `json:"document_id"`
	LibraryID   string `json:"library_id"`
	Revision    int64  `json:"document_revision"`
	VersionID   string `json:"-"`
	RequestedBy string `json:"requested_by"`
	AssignedTo  string `json:"assigned_reviewer_id"`
	Status      string `json:"status"`
	RequestedAt string `json:"requested_at"`
	DecidedBy   string `json:"decided_by"`
	Reason      string `json:"reason"`
	Title       string `json:"title"`
	AuthorName  string `json:"author_name"`
}

const reviewColumns = "r.id,r.document_id,r.library_id,r.document_revision,r.content_version_id,r.requested_by,coalesce(r.assigned_reviewer_id,''),r.status,r.requested_at,coalesce(r.decided_by,''),r.reason,coalesce(nullif(d.title,''),d.original_filename),u.display_name"
const reviewJoins = " FROM review_requests r JOIN documents d ON d.id=r.document_id JOIN users u ON u.id=r.requested_by "

func scanReview(row rowScanner) (Review, error) {
	var review Review
	err := row.Scan(&review.ID, &review.DocumentID, &review.LibraryID, &review.Revision, &review.VersionID, &review.RequestedBy, &review.AssignedTo, &review.Status, &review.RequestedAt, &review.DecidedBy, &review.Reason, &review.Title, &review.AuthorName)
	return review, err
}
func requiresReview(settings Settings, source string) bool {
	if source == "managed" {
		return settings.ReviewManaged
	}
	return settings.ReviewLinked
}
func workflowDocument(ctx context.Context, query storage.Querier, principal domain.Principal, documentID string) (Document, error) {
	document, err := scanDocument(query.QueryRowContext(ctx, "SELECT "+documentColumns+documentJoins+" WHERE d.id=? AND d.deleted_at IS NULL AND "+visibleDocumentSQL, documentID, principal.User.ID, principal.User.ID))
	if err == sql.ErrNoRows {
		return document, notFound()
	}
	return document, err
}
func workflowEvent(ctx context.Context, transaction *sql.Tx, principal domain.Principal, document Document, metadata domain.RequestMetadata, kind string, details map[string]any) error {
	eventID := domain.NewID()
	if err := audit.Append(ctx, transaction, time.Now(), audit.Event{ID: eventID, Type: kind, ActorUserID: principal.User.ID, SystemActor: principal.User.ID == "", SessionID: principal.SessionID, LibraryID: document.LibraryID, CaseID: document.CaseID, DocumentID: document.ID, Metadata: metadata, Details: details}); err != nil {
		return err
	}
	_, err := transaction.ExecContext(ctx, "INSERT INTO document_history(id,document_id,revision,event_id,snapshot_json) SELECT ?,id,revision,?,? FROM documents WHERE id=?", domain.NewID(), eventID, encode(details), document.ID)
	return err
}
func validateWorkflowClassification(ctx context.Context, query storage.Querier, document Document, settings Settings) error {
	if document.CaseID != "" {
		if err := licensing.CheckFeatures("expedientes"); err != nil {
			return err
		}
	}
	if document.CategoryID == "" || document.TypeID == "" || (settings.CasesEnabled && document.CaseID == "") {
		return domain.Failure("CLASSIFICATION_REQUIRED", "Completa expediente, categoría y tipo antes de continuar.", 422)
	}
	var valid int
	if err := query.QueryRowContext(ctx, "SELECT count(*) FROM document_types t JOIN categories c ON c.id=t.category_id WHERE t.id=? AND c.id=? AND t.library_id=? AND t.archived_at IS NULL AND c.archived_at IS NULL AND (t.requires_descriptive_title=0 OR length(trim(?))>0)", document.TypeID, document.CategoryID, document.LibraryID, document.Title).Scan(&valid); err != nil {
		return err
	}
	if valid != 1 {
		return invalid("Revisa la clasificación y el título del documento.")
	}
	if document.Availability != "staged" && document.Availability != "available" {
		return domain.Failure("DOCUMENT_UNAVAILABLE", "Verifica el original antes de continuar.", 409)
	}
	return nil
}
func (service *Service) SubmitReview(ctx context.Context, principal domain.Principal, documentID, assignedTo string, revision int64, metadata domain.RequestMetadata) (string, error) {
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return "", err
	}
	reviewID := domain.NewID()
	err = service.write(ctx, principal, document.LibraryID, "documents.submit", func(transaction *sql.Tx, current domain.Principal) error {
		document, err := workflowDocument(ctx, transaction, current, documentID)
		if err != nil {
			return err
		}
		existing, lookup := scanReview(transaction.QueryRowContext(ctx, "SELECT "+reviewColumns+reviewJoins+" WHERE r.document_id=? AND r.status='pending'", documentID))
		if lookup == nil && existing.RequestedBy == current.User.ID && existing.AssignedTo == assignedTo && existing.Revision == document.Revision && (revision == existing.Revision || revision+1 == existing.Revision) {
			reviewID = existing.ID
			return nil
		}
		if lookup != nil && lookup != sql.ErrNoRows {
			return lookup
		}
		if document.Revision != revision {
			return conflict()
		}
		if document.Approval != "draft" && document.Approval != "rejected" && document.Approval != "needs_review" && document.Approval != "pending_review" {
			return invalid("Este documento no admite envío a revisión.")
		}
		settings, _, err := settingsFor(ctx, transaction, document.LibraryID)
		if err != nil {
			return err
		}
		if !requiresReview(settings, document.Source) {
			return domain.Failure("REVIEW_NOT_REQUIRED", "Esta biblioteca utiliza confirmación directa para este origen.", 409)
		}
		if err = licensing.CheckFeatures("review_workflow"); err != nil {
			return err
		}
		if err = validateWorkflowClassification(ctx, transaction, document, settings); err != nil {
			return err
		}
		if assignedTo != "" {
			if (assignedTo == current.User.ID || assignedTo == document.CreatedBy) && service.require(ctx, transaction, domain.Principal{User: domain.User{ID: assignedTo}}, document.LibraryID, "documents.approve_own") != nil {
				return domain.Failure("SELF_APPROVAL_FORBIDDEN", "Selecciona otra persona para revisar.", 403)
			}
			if err = service.require(ctx, transaction, domain.Principal{User: domain.User{ID: assignedTo}}, document.LibraryID, "documents.approve"); err != nil {
				return invalid("Selecciona un revisor autorizado de la biblioteca.")
			}
			var active bool
			if err = transaction.QueryRowContext(ctx, "SELECT disabled_at IS NULL FROM users WHERE id=?", assignedTo).Scan(&active); err != nil {
				return err
			}
			if !active {
				return invalid("El revisor está deshabilitado.")
			}
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE review_requests SET status='superseded',decided_at=? WHERE document_id=? AND status='pending'", now(), documentID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE documents SET approval_status='pending_review',revision=revision+1 WHERE id=?", documentID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE upload_items SET retain_until=NULL WHERE document_id=?", documentID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO review_requests(id,document_id,library_id,document_revision,content_version_id,requested_by,assigned_reviewer_id,status,requested_at) SELECT ?,d.id,d.library_id,d.revision,f.current_content_version_id,?,?,'pending',? FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=?", reviewID, current.User.ID, nullableID(assignedTo), now(), documentID); err != nil {
			return err
		}
		return workflowEvent(ctx, transaction, current, document, metadata, "document.review_submitted", map[string]any{"review_id": reviewID, "assigned_reviewer_id": assignedTo})
	})
	return reviewID, err
}
func (service *Service) PendingReviews(ctx context.Context, principal domain.Principal, libraryID, cursor string) (Page[Review], error) {
	result := Page[Review]{Items: []Review{}}
	if err := service.Read(ctx, principal, libraryID, "documents.review"); err != nil {
		return result, err
	}
	scope := principal.User.ID + ":reviews:" + libraryID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT "+reviewColumns+reviewJoins+" WHERE r.library_id=? AND r.status='pending' AND d.approval_status='pending_review' AND d.deleted_at IS NULL AND (r.assigned_reviewer_id IS NULL OR r.assigned_reviewer_id=?) AND (?='' OR r.requested_at<? OR (r.requested_at=? AND r.id<?)) ORDER BY r.requested_at DESC,r.id DESC LIMIT 51", libraryID, principal.User.ID, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, review)
	}
	if len(result.Items) > 50 {
		result.Items = result.Items[:50]
		last := result.Items[49]
		result.Next = nextListCursor(scope, last.RequestedAt, last.ID)
	}
	return result, rows.Err()
}
func (service *Service) Reviewers(ctx context.Context, principal domain.Principal, libraryID string) ([]map[string]string, error) {
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT DISTINCT u.id,u.display_name FROM users u JOIN library_role_assignments a ON a.user_id=u.id JOIN role_permissions p USING(role_id) WHERE a.library_id=? AND p.permission_key='documents.approve' AND u.disabled_at IS NULL ORDER BY u.display_name,u.id", libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []map[string]string{}
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result = append(result, map[string]string{"id": id, "display_name": name})
	}
	return result, rows.Err()
}
func (service *Service) DecideReview(ctx context.Context, principal domain.Principal, reviewID string, revision int64, approve bool, reason string, metadata domain.RequestMetadata) (string, error) {
	review, err := scanReview(service.Database.Reader.QueryRowContext(ctx, "SELECT "+reviewColumns+reviewJoins+" WHERE r.id=?", reviewID))
	if err == sql.ErrNoRows {
		return "", notFound()
	}
	if err != nil {
		return "", err
	}
	document, err := service.Document(ctx, principal, review.DocumentID)
	if err != nil {
		return "", err
	}
	permission := "documents.reject"
	if approve {
		permission = "documents.approve"
	}
	if !approve && (strings.TrimSpace(reason) == "" || len(reason) > 1000) {
		return "", invalid("Indica el motivo del rechazo.")
	}
	if approve && document.Approval == "pending_review" {
		if err = service.verifyDecisionContent(ctx, document); err != nil {
			return "", err
		}
	}
	verifiedRevision := document.Revision
	operationID := ""
	err = service.write(ctx, principal, review.LibraryID, permission, func(transaction *sql.Tx, current domain.Principal) error {
		review, err := scanReview(transaction.QueryRowContext(ctx, "SELECT "+reviewColumns+reviewJoins+" WHERE r.id=?", reviewID))
		if err != nil {
			return err
		}
		if review.AssignedTo != "" && review.AssignedTo != current.User.ID {
			return notFound()
		}
		document, err := workflowDocument(ctx, transaction, current, review.DocumentID)
		if err != nil {
			return err
		}
		if current.User.ID == document.CreatedBy || current.User.ID == review.RequestedBy {
			if service.require(ctx, transaction, current, review.LibraryID, "documents.approve_own") != nil {
				return domain.Failure("SELF_APPROVAL_FORBIDDEN", "La revisión debe realizarla otra persona autorizada.", 403)
			}
		}
		if review.Revision == revision {
			if approve {
				op, lookup := scanMaterialization(transaction.QueryRowContext(ctx, "SELECT "+materializationColumns+" FROM materializations WHERE review_id=? AND approved_by=? AND document_revision=?", reviewID, current.User.ID, revision))
				if lookup == nil && (document.Approval == "materializing" && document.Revision == revision+1 || document.Approval == "approved" && document.Revision == revision+2) {
					operationID = op.ID
					return nil
				}
				if lookup != nil && lookup != sql.ErrNoRows {
					return lookup
				}
			}
			if review.DecidedBy == current.User.ID && document.Revision == revision+1 && (approve && review.Status == "approved" && document.Approval == "approved" || !approve && review.Status == "rejected" && document.Approval == "rejected" && review.Reason == strings.TrimSpace(reason)) {
				return nil
			}
		}
		if review.Status != "pending" || document.Approval != "pending_review" {
			return domain.Failure("REVIEW_ALREADY_DECIDED", "La solicitud ya fue resuelta o cambió de estado.", 409)
		}
		if review.Revision != revision || document.Revision != revision || approve && document.Revision != verifiedRevision {
			return conflict()
		}
		var versionID string
		if err = transaction.QueryRowContext(ctx, "SELECT current_content_version_id FROM physical_files WHERE id=?", document.FileID).Scan(&versionID); err != nil {
			return err
		}
		if versionID != review.VersionID {
			return conflict()
		}
		settings, _, err := settingsFor(ctx, transaction, review.LibraryID)
		if err != nil {
			return err
		}
		if !requiresReview(settings, document.Source) {
			return domain.Failure("REVIEW_POLICY_CHANGED", "Cambió la política de revisión. Actualiza el documento.", 409)
		}
		if err = licensing.CheckFeatures("review_workflow"); err != nil {
			return err
		}
		if current.User.ID == document.CreatedBy || current.User.ID == review.RequestedBy {
			if err = record(ctx, transaction, current, metadata, "document.self_review_exception", document.LibraryID, document.ID, map[string]any{"review_id": reviewID, "approve": approve}); err != nil {
				return err
			}
		}
		if !approve {
			if _, err = transaction.ExecContext(ctx, "UPDATE documents SET approval_status='rejected',revision=revision+1 WHERE id=?", document.ID); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE review_requests SET status='rejected',decided_by=?,decided_at=?,reason=? WHERE id=?", current.User.ID, now(), strings.TrimSpace(reason), reviewID); err != nil {
				return err
			}
			if err = setUploadRetention(ctx, transaction, document.ID, settings.RetentionDays); err != nil {
				return err
			}
			return workflowEvent(ctx, transaction, current, document, metadata, "document.rejected", map[string]any{"review_id": reviewID, "reason": strings.TrimSpace(reason)})
		}
		if err = validateWorkflowClassification(ctx, transaction, document, settings); err != nil {
			return err
		}
		operationID, err = service.beginApproval(ctx, transaction, current, document, settings, "review", reviewID, metadata)
		return err
	})
	return operationID, err
}
func (service *Service) FinalizeDocument(ctx context.Context, principal domain.Principal, documentID string, revision int64, metadata domain.RequestMetadata) (string, error) {
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return "", err
	}
	if document.Approval != "materializing" && document.Approval != "approved" {
		if err = service.verifyDecisionContent(ctx, document); err != nil {
			return "", err
		}
	}
	verifiedRevision := document.Revision
	operationID := ""
	err = service.write(ctx, principal, document.LibraryID, "documents.finalize", func(transaction *sql.Tx, current domain.Principal) error {
		document, err := workflowDocument(ctx, transaction, current, documentID)
		if err != nil {
			return err
		}
		op, lookup := scanMaterialization(transaction.QueryRowContext(ctx, "SELECT "+materializationColumns+" FROM materializations WHERE document_id=? AND decision_kind='direct' AND approved_by=? AND document_revision=?", documentID, current.User.ID, revision))
		if lookup == nil && (document.Approval == "materializing" && document.Revision == revision+1 || document.Approval == "approved" && document.Revision == revision+2) {
			operationID = op.ID
			return nil
		}
		if lookup != nil && lookup != sql.ErrNoRows {
			return lookup
		}
		if document.Approval == "approved" && document.Revision == revision+1 {
			var actor string
			if err = transaction.QueryRowContext(ctx, "SELECT coalesce(approved_by,'') FROM documents WHERE id=?", documentID).Scan(&actor); err != nil {
				return err
			}
			if actor == current.User.ID {
				return nil
			}
		}
		if document.Revision != revision || document.Revision != verifiedRevision {
			return conflict()
		}
		if document.Approval != "draft" && document.Approval != "rejected" && document.Approval != "needs_review" {
			return invalid("El documento no admite confirmación directa en su estado actual.")
		}
		settings, _, err := settingsFor(ctx, transaction, document.LibraryID)
		if err != nil {
			return err
		}
		if requiresReview(settings, document.Source) {
			return domain.Failure("REVIEW_REQUIRED", "Envía el documento a revisión.", 409)
		}
		if err = validateWorkflowClassification(ctx, transaction, document, settings); err != nil {
			return err
		}
		operationID, err = service.beginApproval(ctx, transaction, current, document, settings, "direct", "", metadata)
		return err
	})
	return operationID, err
}
func setUploadRetention(ctx context.Context, transaction *sql.Tx, documentID string, days int) error {
	var until any
	if days > 0 {
		until = domain.Timestamp(time.Now().Add(time.Duration(days) * 24 * time.Hour))
	}
	_, err := transaction.ExecContext(ctx, "UPDATE upload_items SET retain_until=? WHERE document_id=? AND status IN ('staged','retained')", until, documentID)
	return err
}
