package libraries

import (
	"context"
	"database/sql"
	"strings"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

type Workflow struct {
	Processing      bool             `json:"processing"`
	RequiresReview  bool             `json:"requires_review"`
	CanSubmit       bool             `json:"can_submit"`
	CanFinalize     bool             `json:"can_finalize"`
	CanApprove      bool             `json:"can_approve"`
	CanReject       bool             `json:"can_reject"`
	CanRetry        bool             `json:"can_retry"`
	Review          *Review          `json:"review"`
	Materialization *Materialization `json:"materialization"`
}

func (service *Service) Workflow(ctx context.Context, principal domain.Principal, documentID string) (Workflow, error) {
	result := Workflow{}
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return result, err
	}
	settings, _, err := settingsFor(ctx, service.Database.Reader, document.LibraryID)
	if err != nil {
		return result, err
	}
	result.RequiresReview = requiresReview(settings, document.Source)
	editable := document.Approval == "draft" || document.Approval == "rejected" || document.Approval == "needs_review" || document.Approval == "pending_review"
	readable := document.Availability == "staged" || document.Availability == "available"
	can := func(permission string) bool {
		return service.require(ctx, service.Database.Reader, principal, document.LibraryID, permission) == nil
	}
	result.CanSubmit = editable && readable && result.RequiresReview && can("documents.submit")
	result.CanFinalize = editable && document.Approval != "pending_review" && readable && !result.RequiresReview && can("documents.finalize")
	review, err := scanReview(service.Database.Reader.QueryRowContext(ctx, "SELECT "+reviewColumns+reviewJoins+" WHERE r.document_id=? ORDER BY r.requested_at DESC,r.id DESC LIMIT 1", documentID))
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	if err == nil {
		result.Review = &review
		assigned := review.AssignedTo == "" || review.AssignedTo == principal.User.ID
		separated := (document.CreatedBy != principal.User.ID && review.RequestedBy != principal.User.ID) || can("documents.approve_own")
		eligible := result.RequiresReview && assigned && separated && document.Approval == "pending_review" && review.Status == "pending" && review.Revision == document.Revision
		result.CanApprove = eligible && readable && can("documents.approve")
		result.CanReject = eligible && can("documents.reject")
	}
	operation, err := scanMaterialization(service.Database.Reader.QueryRowContext(ctx, "SELECT "+materializationColumns+" FROM materializations WHERE document_id=?", documentID))
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	if err == nil {
		result.Materialization = &operation
		if err = service.Database.Reader.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM jobs WHERE job_type='materialize' AND target_version=? AND status IN ('queued','running','retry_wait','paused'))", operation.ID).Scan(&result.Processing); err != nil {
			return result, err
		}
		result.CanRetry = !result.Processing && operation.State != "cleaned" && (can("indexing.retry") || operation.ApprovedBy == principal.User.ID && can("documents.approve"))
	}
	return result, nil
}
func (service *Service) Materialization(ctx context.Context, principal domain.Principal, identifier string) (Materialization, error) {
	operation, err := scanMaterialization(service.Database.Reader.QueryRowContext(ctx, "SELECT "+materializationColumns+" FROM materializations WHERE id=?", identifier))
	if err == sql.ErrNoRows {
		return operation, notFound()
	}
	if err != nil {
		return operation, err
	}
	if _, err = service.Document(ctx, principal, operation.DocumentID); err != nil {
		return Materialization{}, err
	}
	return operation, nil
}
func (service *Service) RetryMaterialization(ctx context.Context, principal domain.Principal, identifier, reason string, metadata domain.RequestMetadata) (string, error) {
	operation, err := service.Materialization(ctx, principal, identifier)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return "", invalid("Indica el motivo del reintento.")
	}
	jobID := domain.NewID()
	err = service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.Identity.License.Check(ctx, licensing.WriteDocuments); err != nil {
			return err
		}
		_, mode, err := settingsFor(ctx, transaction, operation.LibraryID)
		if err != nil {
			return err
		}
		if err = service.Identity.License.CheckFeatures(ctx, modeCapabilities(mode)...); err != nil {
			return err
		}
		if operation.DecisionKind == "review" {
			if err = service.Identity.License.CheckFeatures(ctx, "review_workflow"); err != nil {
				return err
			}
		}
		if err := service.require(ctx, transaction, current, operation.LibraryID, "indexing.retry"); err != nil {
			if operation.ApprovedBy != current.User.ID {
				return err
			}
			if err = service.require(ctx, transaction, current, operation.LibraryID, "documents.approve"); err != nil {
				return err
			}
		}
		if _, err := workflowDocument(ctx, transaction, current, operation.DocumentID); err != nil {
			return err
		}
		var state string
		if err := transaction.QueryRowContext(ctx, "SELECT state FROM materializations WHERE id=?", identifier).Scan(&state); err != nil {
			return err
		}
		if state == "cleaned" {
			return invalid("El guardado definitivo ya terminó.")
		}
		var active string
		err = transaction.QueryRowContext(ctx, "SELECT id FROM jobs WHERE job_type='materialize' AND target_version=? AND status IN ('queued','running','retry_wait','paused') ORDER BY created_at DESC LIMIT 1", identifier).Scan(&active)
		if err == nil {
			jobID = active
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		var prior string
		if err = transaction.QueryRowContext(ctx, "SELECT id FROM jobs WHERE job_type='materialize' AND target_version=? ORDER BY created_at DESC,id DESC LIMIT 1", identifier).Scan(&prior); err != nil {
			return err
		}
		root, err := scanRoot(transaction.QueryRowContext(ctx, "SELECT "+rootColumns+" FROM storage_roots WHERE id=?", operation.RootID))
		if err != nil {
			return err
		}
		if root.Status != "active" {
			return scanFailure("ROOT_UNAVAILABLE")
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE materializations SET root_revision=?,state=CASE WHEN state='failed' THEN 'planned' ELSE state END,error_code='',updated_at=? WHERE id=?", root.Revision, now(), identifier); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,retry_of_job_id,created_at) VALUES(?,?,?,'materialize',?,?,'{}','queued',?,?,?)", jobID, operation.LibraryID, operation.FileID, identifier, jobID, now(), prior, now()); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "materialization.retry_requested", operation.LibraryID, operation.DocumentID, map[string]any{"operation_id": identifier, "reason": reason})
	})
	return jobID, err
}
