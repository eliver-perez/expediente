package libraries

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"os"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/extraction"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

func (service *Service) Document(ctx context.Context, principal domain.Principal, documentID string) (Document, error) {
	document, err := scanDocument(service.Database.Reader.QueryRowContext(ctx, "SELECT "+documentColumns+documentJoins+" WHERE d.id=? AND d.deleted_at IS NULL AND "+visibleDocumentSQL, documentID, principal.User.ID, principal.User.ID))
	if err == sql.ErrNoRows {
		return document, notFound()
	}
	if err != nil {
		return document, err
	}
	if err = service.Read(ctx, principal, document.LibraryID, "documents.read"); err != nil {
		return Document{}, err
	}
	document.Preview = document.Availability == "available" || document.Availability == "staged"
	document.CanCancel = document.Availability == "staged" && document.CreatedBy == principal.User.ID && (document.Approval == "draft" || document.Approval == "rejected" || document.Approval == "pending_review") && service.require(ctx, service.Database.Reader, principal, document.LibraryID, "documents.cancel_own") == nil
	document.CanClassify = service.require(ctx, service.Database.Reader, principal, document.LibraryID, "documents.classify") == nil
	document.CanAssociate = document.Source == "linked" && service.require(ctx, service.Database.Reader, principal, document.LibraryID, "documents.associate") == nil
	document.CanReassign = service.require(ctx, service.Database.Reader, principal, document.LibraryID, "documents.reassign") == nil
	document.Download = document.Preview && service.require(ctx, service.Database.Reader, principal, document.LibraryID, "documents.download") == nil
	if document.Availability == "staged" || service.require(ctx, service.Database.Reader, principal, document.LibraryID, "storage.view_paths") != nil {
		document.OriginalPath = ""
	}
	return document, nil
}
func (service *Service) OpenDocument(ctx context.Context, principal domain.Principal, documentID string, download bool, metadata domain.RequestMetadata) (*os.File, Document, error) {
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return nil, document, err
	}
	permission := "documents.read"
	event := "document.opened"
	if download {
		permission = "documents.download"
		event = "document.downloaded"
	}
	if err = service.Read(ctx, principal, document.LibraryID, permission); err != nil {
		return nil, Document{}, err
	}
	if document.Availability != "available" && document.Availability != "staged" {
		return nil, document, domain.Failure("DOCUMENT_UNAVAILABLE", "El original no está disponible; puedes consultar el texto retenido.", 409)
	}
	var file *os.File
	if document.Availability == "staged" {
		file, err = service.openUpload(ctx, document.FileID)
	} else {
		root, rootErr := service.root(ctx, document.RootID)
		if rootErr != nil {
			return nil, document, rootErr
		}
		file, err = openLinked(root, document.RelativePath)
	}
	if err != nil {
		return nil, document, domain.Failure("DOCUMENT_UNAVAILABLE", "El original no está disponible; puedes consultar el texto retenido.", 409)
	}
	header := make([]byte, 5)
	if _, err = io.ReadFull(file, header); err != nil || string(header) != "%PDF-" {
		file.Close()
		return nil, document, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return nil, document, err
	}
	identity, _, err := physicalIdentity(file)
	if err != nil || identity != document.Identity {
		file.Close()
		return nil, document, scanFailure("DOCUMENT_UNAVAILABLE")
	}
	err = service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		if err := service.require(ctx, transaction, current, document.LibraryID, permission); err != nil {
			return err
		}
		if err := service.Identity.License.Check(ctx, licensing.ReadDocuments); err != nil {
			return err
		}
		var visible int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE d.id=? AND d.deleted_at IS NULL AND "+visibleDocumentSQL, document.ID, current.User.ID, current.User.ID).Scan(&visible); err != nil {
			return err
		}
		if visible != 1 {
			return notFound()
		}
		return record(ctx, transaction, current, metadata, event, document.LibraryID, document.ID, nil)
	})
	if err != nil {
		file.Close()
		return nil, document, err
	}
	return file, document, nil
}
func (service *Service) Page(ctx context.Context, principal domain.Principal, documentID string, number int) (extraction.Page, error) {
	var page extraction.Page
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return page, err
	}
	err = service.Database.Reader.QueryRowContext(ctx, "SELECT p.page_number,p.extraction_method,p.page_text FROM extraction_pages p JOIN physical_files f ON f.indexed_extraction_id=p.extraction_id WHERE f.id=? AND p.page_number=?", document.FileID, number).Scan(&page.Number, &page.Method, &page.Text)
	if err == sql.ErrNoRows {
		return page, notFound()
	}
	return page, err
}
func (service *Service) RemoveIndex(ctx context.Context, principal domain.Principal, documentID, reason string, revision int64, metadata domain.RequestMetadata) error {
	return service.RemoveIndexConfirmed(ctx, principal, documentID, reason, "", revision, metadata)
}
func (service *Service) RemoveIndexConfirmed(ctx context.Context, principal domain.Principal, documentID, reason, expectedCaseID string, revision int64, metadata domain.RequestMetadata) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return invalid("Escribe un motivo de hasta 1000 caracteres.")
	}
	document, err := service.Document(ctx, principal, documentID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, document.LibraryID, "documents.remove_index", func(transaction *sql.Tx, current domain.Principal) error {
		var approval string
		if err := transaction.QueryRowContext(ctx, "SELECT approval_status FROM documents WHERE id=?", documentID).Scan(&approval); err != nil {
			return err
		}
		if approval == "materializing" {
			return invalid("Espera a que termine el guardado definitivo antes de retirar el documento.")
		}
		var currentCase string
		if err := transaction.QueryRowContext(ctx, "SELECT coalesce(case_id,'') FROM documents WHERE id=?", documentID).Scan(&currentCase); err != nil {
			return err
		}
		if currentCase != expectedCaseID {
			return domain.Failure("CASE_CONFIRMATION_REQUIRED", "Confirma el retiro del documento de su expediente actual.", 409)
		}
		result, err := transaction.ExecContext(ctx, "UPDATE documents SET deleted_at=?,revision=revision+1 WHERE id=? AND revision=? AND deleted_at IS NULL", now(), documentID, revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("VERSION_CONFLICT", "El documento cambió. Recarga su ficha.", 412)
		}
		if _, err = transaction.ExecContext(ctx, "DELETE FROM indexed_pages WHERE physical_file_id=?", document.FileID); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "document.index_removed", document.LibraryID, documentID, map[string]any{"reason": reason})
	})
}

type Notice struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	Kind       string `json:"kind"`
	Created    string `json:"created_at"`
}

func (service *Service) Notifications(ctx context.Context, principal domain.Principal) ([]Notice, error) {
	if err := service.Identity.License.Check(ctx, licensing.ReadDocuments); err != nil {
		return nil, err
	}
	notices := []Notice{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT n.id,n.document_id,n.notification_kind,n.created_at FROM document_notifications n JOIN documents d ON d.id=n.document_id WHERE d.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM notification_receipts WHERE notification_id=n.id AND user_id=?) AND EXISTS(SELECT 1 FROM library_role_assignments a JOIN role_permissions p USING(role_id) WHERE a.user_id=? AND a.library_id=d.library_id AND p.permission_key='documents.read') ORDER BY n.created_at DESC,n.id LIMIT 100", principal.User.ID, principal.User.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var notice Notice
		if err = rows.Scan(&notice.ID, &notice.DocumentID, &notice.Kind, &notice.Created); err != nil {
			return nil, err
		}
		notices = append(notices, notice)
	}
	return notices, rows.Err()
}
func (service *Service) Acknowledge(ctx context.Context, principal domain.Principal, noticeID string) error {
	if err := service.Identity.License.Check(ctx, licensing.ReadDocuments); err != nil {
		return err
	}
	return service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		var libraryID string
		err := transaction.QueryRowContext(ctx, "SELECT library_id FROM documents d JOIN document_notifications n ON n.document_id=d.id WHERE n.id=?", noticeID).Scan(&libraryID)
		if err == sql.ErrNoRows {
			return notFound()
		}
		if err != nil {
			return err
		}
		if err = service.require(ctx, transaction, current, libraryID, "documents.read"); err != nil {
			return err
		}
		_, err = transaction.ExecContext(ctx, "INSERT INTO notification_receipts VALUES(?,?,?) ON CONFLICT DO NOTHING", noticeID, current.User.ID, now())
		return err
	})
}
func (service *Service) Jobs(ctx context.Context, principal domain.Principal, libraryID, cursor string) (Page[Job], error) {
	page := Page[Job]{Items: []Job{}}
	if err := service.Read(ctx, principal, libraryID, "indexing.run"); err != nil {
		return page, err
	}
	scope := principal.User.ID + ":jobs:" + libraryID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return page, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,library_id,coalesce(physical_file_id,''),job_type,status,attempt_count,coalesce(last_error_code,''),created_at FROM jobs WHERE library_id=? AND (physical_file_id IS NULL OR EXISTS(SELECT 1 FROM documents d JOIN physical_files f ON f.id=d.physical_file_id WHERE f.id=jobs.physical_file_id AND "+visibleDocumentSQL+")) AND (?='' OR created_at<? OR (created_at=? AND id<?)) ORDER BY created_at DESC,id DESC LIMIT 51", libraryID, principal.User.ID, principal.User.ID, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var job Job
		if err = rows.Scan(&job.ID, &job.LibraryID, &job.FileID, &job.Kind, &job.Status, &job.Attempts, &job.Error, &job.Created); err != nil {
			return page, err
		}
		page.Items = append(page.Items, job)
	}
	if len(page.Items) > 50 {
		page.Items = page.Items[:50]
		last := page.Items[49]
		page.Next = nextListCursor(scope, last.Created, last.ID)
	}
	return page, rows.Err()
}
func (service *Service) Job(ctx context.Context, principal domain.Principal, identifier string) (Job, error) {
	var job Job
	err := service.Database.Reader.QueryRowContext(ctx, "SELECT id,library_id,coalesce(physical_file_id,''),job_type,status,attempt_count,coalesce(last_error_code,''),created_at FROM jobs WHERE id=?", identifier).Scan(&job.ID, &job.LibraryID, &job.FileID, &job.Kind, &job.Status, &job.Attempts, &job.Error, &job.Created)
	if err == sql.ErrNoRows {
		return job, notFound()
	}
	if err != nil {
		return job, err
	}
	if job.FileID != "" {
		var documentID string
		if err = service.Database.Reader.QueryRowContext(ctx, "SELECT id FROM documents WHERE physical_file_id=?", job.FileID).Scan(&documentID); err != nil {
			return Job{}, notFound()
		}
		if _, err = service.Document(ctx, principal, documentID); err != nil {
			return Job{}, err
		}
	}
	if err = service.Read(ctx, principal, job.LibraryID, "indexing.run"); err != nil {
		return Job{}, err
	}
	return job, nil
}
func (service *Service) Verify(ctx context.Context, principal domain.Principal, libraryID, rootID string, metadata domain.RequestMetadata) error {
	return service.write(ctx, principal, libraryID, "indexing.run", func(transaction *sql.Tx, current domain.Principal) error {
		roots, err := rootsQuery(ctx, transaction, libraryID)
		if err != nil {
			return err
		}
		found := false
		for _, root := range roots {
			if verifiableRoot(root) && (rootID == "" || rootID == root.ID) {
				found = true
				if err = enqueueVerification(ctx, transaction, root); err != nil {
					return err
				}
			}
		}
		if !found {
			return notFound()
		}
		return record(ctx, transaction, current, metadata, "indexing.requested", libraryID, "", map[string]any{"root_id": rootID})
	})
}
func (service *Service) Retry(ctx context.Context, principal domain.Principal, jobID, reason string, metadata domain.RequestMetadata) (string, error) {
	job, err := service.Job(ctx, principal, jobID)
	if err != nil {
		return "", err
	}
	if job.Kind == "materialize" {
		var operationID string
		if err = service.Database.Reader.QueryRowContext(ctx, "SELECT target_version FROM jobs WHERE id=?", jobID).Scan(&operationID); err != nil {
			return "", err
		}
		return service.RetryMaterialization(ctx, principal, operationID, reason, metadata)
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 1000 {
		return "", invalid("Escribe el motivo del reintento.")
	}
	identifier := domain.NewID()
	err = service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		var libraryID, status string
		if err := transaction.QueryRowContext(ctx, "SELECT library_id,status FROM jobs WHERE id=?", jobID).Scan(&libraryID, &status); err == sql.ErrNoRows {
			return notFound()
		} else if err != nil {
			return err
		}
		if err := service.require(ctx, transaction, current, libraryID, "indexing.retry"); err != nil {
			return err
		}
		if err := service.Identity.License.Check(ctx, licensing.WriteDocuments); err != nil {
			return err
		}
		if status != "failed" && status != "cancelled" {
			return invalid("Solo se reintentan tareas fallidas o canceladas.")
		}
		var existing string
		err := transaction.QueryRowContext(ctx, "SELECT id FROM jobs WHERE retry_of_job_id=? ORDER BY created_at LIMIT 1", jobID).Scan(&existing)
		if err == nil {
			identifier = existing
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,retry_of_job_id,created_at) SELECT ?,library_id,physical_file_id,job_type,target_version,?,payload_json,'queued',?,?,? FROM jobs WHERE id=?", identifier, identifier, now(), jobID, now(), jobID); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "indexing.retry_requested", libraryID, "", map[string]any{"job_id": identifier, "retry_of": jobID, "reason": reason})
	})
	return identifier, err
}

type View struct {
	ID     string `json:"id"`
	RootID string `json:"root_id"`
	Name   string `json:"name"`
	Prefix string `json:"relative_prefix"`
}

func (service *Service) Views(ctx context.Context, principal domain.Principal, libraryID string) ([]View, error) {
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, err
	}
	views := []View{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,root_id,name,relative_prefix FROM logical_folder_views WHERE library_id=? ORDER BY name,id", libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var view View
		if err = rows.Scan(&view.ID, &view.RootID, &view.Name, &view.Prefix); err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, rows.Err()
}

type Member struct {
	ID    string   `json:"user_id"`
	Name  string   `json:"display_name"`
	Roles []string `json:"role_ids"`
}

func (service *Service) Members(ctx context.Context, principal domain.Principal, libraryID string) ([]Member, error) {
	if !principal.Can("permissions.manage_global") {
		if err := service.Read(ctx, principal, libraryID, "permissions.manage_library"); err != nil {
			return nil, err
		}
	}
	members := []Member{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT u.id,u.display_name,a.role_id FROM library_role_assignments a JOIN users u ON u.id=a.user_id WHERE library_id=? ORDER BY u.id,a.role_id", libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var identifier, name, role string
		if err = rows.Scan(&identifier, &name, &role); err != nil {
			return nil, err
		}
		if len(members) == 0 || members[len(members)-1].ID != identifier {
			members = append(members, Member{ID: identifier, Name: name, Roles: []string{}})
		}
		members[len(members)-1].Roles = append(members[len(members)-1].Roles, role)
	}
	return members, rows.Err()
}

type Event struct {
	ID         string `json:"id"`
	Kind       string `json:"event_type"`
	DocumentID string `json:"document_id"`
	At         string `json:"occurred_at"`
	Details    string `json:"details_json"`
}

func (service *Service) Events(ctx context.Context, principal domain.Principal, libraryID, documentID, cursor string) (Page[Event], error) {
	page := Page[Event]{Items: []Event{}}
	permission := "audit.read_library"
	if documentID != "" {
		document, err := service.Document(ctx, principal, documentID)
		if err != nil {
			return page, err
		}
		libraryID = document.LibraryID
		permission = "documents.read"
	}
	if err := service.Read(ctx, principal, libraryID, permission); err != nil {
		return page, err
	}
	scope := principal.User.ID + ":events:" + libraryID + ":" + documentID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return page, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,event_type,coalesce(document_id,''),occurred_at,details_json FROM audit_events WHERE library_id=? AND (?='' OR document_id=?) AND (?='' OR occurred_at<? OR (occurred_at=? AND id<?)) ORDER BY occurred_at DESC,id DESC LIMIT 51", libraryID, documentID, documentID, after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var event Event
		if err = rows.Scan(&event.ID, &event.Kind, &event.DocumentID, &event.At, &event.Details); err != nil {
			return page, err
		}
		page.Items = append(page.Items, event)
	}
	if len(page.Items) > 50 {
		page.Items = page.Items[:50]
		last := page.Items[49]
		page.Next = nextListCursor(scope, last.At, last.ID)
	}
	return page, rows.Err()
}

type Duplicate struct {
	Hash  string `json:"sha256"`
	Count int    `json:"count"`
}

func (service *Service) Duplicates(ctx context.Context, principal domain.Principal, libraryID string) ([]Duplicate, error) {
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, err
	}
	duplicates := []Duplicate{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT v.sha256,count(*) FROM physical_files f JOIN content_versions v ON v.id=f.current_content_version_id JOIN documents d ON d.physical_file_id=f.id WHERE f.library_id=? AND d.deleted_at IS NULL AND "+visibleDocumentSQL+" GROUP BY v.sha256 HAVING count(*)>1 ORDER BY v.sha256 LIMIT 100", libraryID, principal.User.ID, principal.User.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var duplicate Duplicate
		if err = rows.Scan(&duplicate.Hash, &duplicate.Count); err != nil {
			return nil, err
		}
		duplicates = append(duplicates, duplicate)
	}
	return duplicates, rows.Err()
}
func (service *Service) ConfigureRoot(ctx context.Context, principal domain.Principal, rootID string, enabled bool, interval int, revision int64, metadata domain.RequestMetadata) error {
	if interval < 10 || interval > 86400 {
		return invalid("El intervalo debe estar entre 10 segundos y 24 horas.")
	}
	root, err := service.root(ctx, rootID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, root.LibraryID, "storage.manage_roots", func(transaction *sql.Tx, current domain.Principal) error {
		status, mode := "disabled", "paused"
		if enabled {
			status = "active"
			mode = "polling"
		}
		result, err := transaction.ExecContext(ctx, "UPDATE storage_roots SET status=?,watch_mode=?,reconcile_interval_seconds=?,configuration_revision=configuration_revision+1 WHERE id=? AND configuration_revision=? AND status<>'superseded'", status, mode, interval, rootID, revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("VERSION_CONFLICT", "La raíz cambió. Recarga la página.", 412)
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE libraries SET current_configuration_revision=current_configuration_revision+1,revision=revision+1 WHERE id=?", root.LibraryID); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "storage.root_configured", root.LibraryID, "", map[string]any{"root_id": rootID, "enabled": enabled, "interval": interval})
	})
}
func retirementImpact(ctx context.Context, query storage.Querier, rootID string) (string, int, error) {
	rows, err := query.QueryContext(ctx, "SELECT DISTINCT d.id,coalesce(d.case_id,'') FROM documents d JOIN physical_file_locations l ON l.physical_file_id=d.physical_file_id WHERE l.root_id=? AND l.retired_at IS NULL AND d.deleted_at IS NULL ORDER BY d.id", rootID)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	digest := sha256.New()
	cases := map[string]bool{}
	for rows.Next() {
		var documentID, caseID string
		if err := rows.Scan(&documentID, &caseID); err != nil {
			return "", 0, err
		}
		_, _ = io.WriteString(digest, encode([]string{documentID, caseID})+"\n")
		if caseID != "" {
			cases[caseID] = true
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), len(cases), rows.Err()
}

func (service *Service) RetirementPlan(ctx context.Context, principal domain.Principal, rootID string) (map[string]any, error) {
	root, err := service.root(ctx, rootID)
	if err != nil {
		return nil, err
	}
	identifier := domain.NewID()
	count := 0
	caseCount := 0
	err = service.write(ctx, principal, root.LibraryID, "storage.manage_roots", func(transaction *sql.Tx, current domain.Principal) error {
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM physical_file_locations WHERE root_id=? AND retired_at IS NULL", rootID).Scan(&count); err != nil {
			return err
		}
		impact, affected, err := retirementImpact(ctx, transaction, rootID)
		if err != nil {
			return err
		}
		caseCount = affected
		_, err = transaction.ExecContext(ctx, "INSERT INTO root_operations VALUES(?,?,?,'retire','planned',?,?,'{}',?,?)", identifier, root.LibraryID, current.User.ID, encode(map[string]any{"revision": root.Revision}), encode(map[string]any{"root_id": rootID, "count": count, "impact": impact, "case_count": caseCount}), now(), now())
		return err
	})
	return map[string]any{"plan_id": identifier, "reference_count": count, "affected_case_count": caseCount, "revision": root.Revision, "root_id": rootID}, err
}
func (service *Service) Retire(ctx context.Context, principal domain.Principal, rootID, planID, policy string, metadata domain.RequestMetadata) error {
	if policy != "retain_index" && policy != "remove_index_references" {
		return invalid("Elige conservar o retirar referencias del índice.")
	}
	root, err := service.root(ctx, rootID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, root.LibraryID, "storage.manage_roots", func(transaction *sql.Tx, current domain.Principal) error {
		var state, configuration, plan, created string
		var pending int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM materializations WHERE target_root_id=? AND state<>'cleaned'", rootID).Scan(&pending); err != nil {
			return err
		}
		if pending > 0 {
			return invalid("Termina los guardados definitivos pendientes antes de retirar esta carpeta.")
		}
		err := transaction.QueryRowContext(ctx, "SELECT state,expected_configuration_json,private_plan_json,created_at FROM root_operations WHERE id=? AND library_id=? AND requested_by=? AND operation_kind='retire'", planID, root.LibraryID, current.User.ID).Scan(&state, &configuration, &plan, &created)
		if err == sql.ErrNoRows {
			return notFound()
		}
		if err != nil {
			return err
		}
		if state == "complete" {
			return nil
		}
		var plannedRoot string
		var expectedImpact string
		var expected int64
		var count int
		if err = transaction.QueryRowContext(ctx, "SELECT json_extract(?,'$.revision'),json_extract(?,'$.root_id'),json_extract(?,'$.count'),coalesce(json_extract(?,'$.impact'),'')", configuration, plan, plan, plan).Scan(&expected, &plannedRoot, &count, &expectedImpact); err != nil {
			return err
		}
		var currentCount int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM physical_file_locations WHERE root_id=? AND retired_at IS NULL", rootID).Scan(&currentCount); err != nil {
			return err
		}
		var liveRevision int64
		if err = transaction.QueryRowContext(ctx, "SELECT configuration_revision FROM storage_roots WHERE id=?", rootID).Scan(&liveRevision); err != nil {
			return err
		}
		impact, caseCount, err := retirementImpact(ctx, transaction, rootID)
		if err != nil {
			return err
		}
		if plannedRoot != rootID || liveRevision != expected || count != currentCount || impact != expectedImpact || created < domain.Timestamp(time.Now().Add(-15*time.Minute)) {
			return scanFailure("ROOT_PLAN_STALE")
		}
		if policy == "remove_index_references" {
			if err = service.require(ctx, transaction, current, root.LibraryID, "documents.remove_index"); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE documents SET deleted_at=?,revision=revision+1 WHERE physical_file_id IN (SELECT physical_file_id FROM physical_file_locations WHERE root_id=? AND retired_at IS NULL)", now(), rootID); err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "DELETE FROM indexed_pages WHERE physical_file_id IN (SELECT physical_file_id FROM documents WHERE library_id=? AND deleted_at IS NOT NULL)", root.LibraryID); err != nil {
				return err
			}
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE storage_roots SET status='retired',watch_mode='paused',configuration_revision=configuration_revision+1 WHERE id=?", rootID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE root_operations SET state='complete',updated_at=? WHERE id=?", now(), planID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE libraries SET current_configuration_revision=current_configuration_revision+1,revision=revision+1 WHERE id=?", root.LibraryID); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "UPDATE libraries SET settings_json=json_set(settings_json,'$.default_managed_root_id','') WHERE id=? AND json_extract(settings_json,'$.default_managed_root_id')=?", root.LibraryID, rootID); err != nil {
			return err
		}
		if err = configurationSnapshot(ctx, transaction, root.LibraryID, current.User.ID); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "storage.root_retired", root.LibraryID, "", map[string]any{"root_id": rootID, "reference_policy": policy, "reference_count": count, "affected_case_count": caseCount})
	})
}
