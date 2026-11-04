package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"gestor-documental/internal/domain"
)

type Correction struct {
	ID             string         `json:"id"`
	DocumentID     string         `json:"document_id"`
	Operation      string         `json:"operation"`
	RequestedBy    string         `json:"requested_by"`
	Revision       int64          `json:"document_revision"`
	Proposed       Classification `json:"proposed"`
	Previous       Classification `json:"previous"`
	Reason         string         `json:"reason"`
	Status         string         `json:"status"`
	RequestedAt    string         `json:"requested_at"`
	DecidedBy      string         `json:"decided_by"`
	DecisionReason string         `json:"decision_reason"`
	CanDecide      bool           `json:"can_decide"`
}

const correctionColumns = "id,document_id,operation,requested_by,document_revision,proposed_json,previous_json,reason,status,requested_at,coalesce(decided_by,''),decision_reason"

func classificationSnapshot(document Document) Classification {
	return Classification{CaseIdentifier: document.CaseIdentifier, CategoryName: document.CategoryName, TypeName: document.TypeName, CaseID: document.CaseID, CategoryID: document.CategoryID, TypeID: document.TypeID, Title: document.Title, Metadata: document.Metadata}
}

func scanCorrection(row rowScanner) (Correction, error) {
	var item Correction
	var proposed, previous string
	err := row.Scan(&item.ID, &item.DocumentID, &item.Operation, &item.RequestedBy, &item.Revision, &proposed, &previous, &item.Reason, &item.Status, &item.RequestedAt, &item.DecidedBy, &item.DecisionReason)
	if err == nil {
		err = json.Unmarshal([]byte(proposed), &item.Proposed)
	}
	if err == nil {
		err = json.Unmarshal([]byte(previous), &item.Previous)
	}
	return item, err
}

func (service *Service) proposeClassificationCorrection(ctx context.Context, tx *sql.Tx, p domain.Principal, document Document, operation string, input Classification, meta domain.RequestMetadata) error {
	if err := service.Identity.License.CheckFeatures(ctx, "review_workflow"); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM document_corrections WHERE document_id=? AND operation<>'request' AND status='pending'", document.ID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return domain.Failure("CORRECTION_PENDING", "Ya existe una corrección de metadatos pendiente. Resuélvela antes de proponer otra.", 409)
	}
	id := domain.NewID()
	_, err := tx.ExecContext(ctx, `INSERT INTO document_corrections(id,document_id,library_id,operation,requested_by,document_revision,proposed_json,previous_json,reason,requested_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, id, document.ID, document.LibraryID, operation, p.User.ID, document.Revision, encode(input), encode(classificationSnapshot(document)), strings.TrimSpace(input.Reason), now())
	if err != nil {
		return err
	}
	return record(ctx, tx, p, meta, "document.classification_correction_requested", document.LibraryID, document.ID, map[string]any{"correction_id": id, "previous": classificationSnapshot(document), "proposed": input, "reason": input.Reason})
}

func (service *Service) RequestCorrection(ctx context.Context, p domain.Principal, documentID, reason string, revision int64, meta domain.RequestMetadata) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1000 {
		return invalid("Indica qué debe corregirse (máximo 1000 caracteres).")
	}
	document, err := service.Document(ctx, p, documentID)
	if err != nil {
		return err
	}
	return service.write(ctx, p, document.LibraryID, "documents.request_correction", func(tx *sql.Tx, current domain.Principal) error {
		d, err := workflowDocument(ctx, tx, current, documentID)
		if err != nil {
			return err
		}
		if d.CreatedBy != current.User.ID || d.Approval != "approved" {
			return notFound()
		}
		if d.Revision != revision {
			return conflict()
		}
		var pending int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM document_corrections WHERE document_id=? AND operation='request' AND status='pending'", documentID).Scan(&pending); err != nil {
			return err
		}
		if pending > 0 {
			return domain.Failure("CORRECTION_PENDING", "Ya hay una solicitud tuya pendiente para este documento.", 409)
		}
		id := domain.NewID()
		_, err = tx.ExecContext(ctx, `INSERT INTO document_corrections(id,document_id,library_id,operation,requested_by,document_revision,proposed_json,previous_json,reason,requested_at) VALUES(?,?,?,'request',?,?,?, ?,?,?)`, id, d.ID, d.LibraryID, current.User.ID, d.Revision, "{}", encode(classificationSnapshot(d)), reason, now())
		if err != nil {
			return err
		}
		return record(ctx, tx, current, meta, "document.correction_requested", d.LibraryID, d.ID, map[string]any{"correction_id": id, "reason": reason, "previous": classificationSnapshot(d)})
	})
}

func (service *Service) Corrections(ctx context.Context, p domain.Principal, documentID string) ([]Correction, error) {
	d, err := service.Document(ctx, p, documentID)
	if err != nil {
		return nil, err
	}
	manager := service.require(ctx, service.Database.Reader, p, d.LibraryID, "documents.correct_classification") == nil
	reviewer := service.require(ctx, service.Database.Reader, p, d.LibraryID, "documents.approve") == nil
	approveOwn := service.require(ctx, service.Database.Reader, p, d.LibraryID, "documents.approve_own") == nil
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT "+correctionColumns+" FROM document_corrections WHERE document_id=? AND (? OR requested_by=?) ORDER BY requested_at DESC,id DESC LIMIT 100", documentID, manager || reviewer, p.User.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Correction{}
	for rows.Next() {
		item, err := scanCorrection(rows)
		if err != nil {
			return nil, err
		}
		item.CanDecide = item.Status == "pending" && (item.Operation == "request" && manager || item.Operation != "request" && reviewer && (item.RequestedBy != p.User.ID || approveOwn))
		items = append(items, item)
	}
	return items, rows.Err()
}

func (service *Service) DecideCorrection(ctx context.Context, p domain.Principal, id string, approve bool, reason string, meta domain.RequestMetadata) error {
	var documentID string
	if err := service.Database.Reader.QueryRowContext(ctx, "SELECT document_id FROM document_corrections WHERE id=?", id).Scan(&documentID); err != nil {
		return notFound()
	}
	d, err := service.Document(ctx, p, documentID)
	if err != nil {
		return err
	}
	return service.write(ctx, p, d.LibraryID, "documents.read", func(tx *sql.Tx, current domain.Principal) error {
		item, err := scanCorrection(tx.QueryRowContext(ctx, "SELECT "+correctionColumns+" FROM document_corrections WHERE id=?", id))
		if err != nil {
			return err
		}
		if item.Status != "pending" {
			return domain.Failure("CORRECTION_ALREADY_DECIDED", "Esta solicitud ya fue resuelta.", 409)
		}
		permission := "documents.approve"
		if item.Operation == "request" {
			permission = "documents.correct_classification"
		}
		if err = service.require(ctx, tx, current, d.LibraryID, permission); err != nil {
			return err
		}
		if item.Operation != "request" {
			if err = service.Identity.License.CheckFeatures(ctx, "review_workflow"); err != nil {
				return err
			}
			if item.RequestedBy == current.User.ID && service.require(ctx, tx, current, d.LibraryID, "documents.approve_own") != nil {
				return domain.Failure("SELF_APPROVAL_FORBIDDEN", "La corrección debe autorizarla otra persona.", 403)
			}
		}
		reason = strings.TrimSpace(reason)
		if (!approve || item.Operation == "request") && reason == "" || len(reason) > 1000 {
			return invalid("Indica el resultado o motivo de la decisión.")
		}
		if approve && item.Operation != "request" {
			if err = service.require(ctx, tx, domain.Principal{User: domain.User{ID: item.RequestedBy}}, d.LibraryID, "documents.correct_classification"); err != nil {
				return err
			}
			if err = service.classifyTransaction(ctx, tx, current, d.ID, item.Operation, item.Proposed, item.Revision, meta, true); err != nil {
				return err
			}
		}
		status := "rejected"
		if approve {
			status = "approved"
		}
		if _, err = tx.ExecContext(ctx, "UPDATE document_corrections SET status=?,decided_by=?,decided_at=?,decision_reason=? WHERE id=?", status, current.User.ID, now(), reason, id); err != nil {
			return err
		}
		return record(ctx, tx, current, meta, "document.correction_decided", d.LibraryID, d.ID, map[string]any{"correction_id": id, "requested_by": item.RequestedBy, "status": status, "reason": reason, "previous": item.Previous, "new": item.Proposed})
	})
}

func (service *Service) PendingCorrections(ctx context.Context, p domain.Principal, libraryID, cursor string) (Page[Correction], error) {
	result := Page[Correction]{Items: []Correction{}}
	if err := service.Read(ctx, p, libraryID, "documents.read"); err != nil {
		return result, err
	}
	manager := service.require(ctx, service.Database.Reader, p, libraryID, "documents.correct_classification") == nil
	reviewer := service.require(ctx, service.Database.Reader, p, libraryID, "documents.approve") == nil
	if !manager && !reviewer {
		return result, notFound()
	}
	scope := p.User.ID + ":corrections:" + libraryID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT "+correctionColumns+" FROM document_corrections WHERE library_id=? AND status='pending' AND ((operation='request' AND ?) OR (operation<>'request' AND ?)) AND (?='' OR requested_at<? OR (requested_at=? AND id<?)) ORDER BY requested_at DESC,id DESC LIMIT 51", libraryID, manager, reviewer, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanCorrection(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if len(result.Items) > 50 {
		result.Items = result.Items[:50]
		last := result.Items[49]
		result.Next = nextListCursor(scope, last.RequestedAt, last.ID)
	}
	return result, rows.Err()
}
