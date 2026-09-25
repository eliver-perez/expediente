package libraries

import (
	"context"
	"database/sql"
	"strings"
	"time"
	"unicode/utf8"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type Case struct {
	ID                string `json:"id"`
	LibraryID         string `json:"library_id"`
	Identifier        string `json:"identifier"`
	Exercise          string `json:"exercise"`
	TemplateVersionID string `json:"template_version_id"`
	Initialized       bool   `json:"requirements_initialized"`
	Revision          int64  `json:"revision"`
	Created           string `json:"created_at"`
}

const caseColumns = "id,library_id,identifier,coalesce(exercise,''),coalesce(template_version_id,''),requirements_initialized_at IS NOT NULL,revision,created_at"

func scanCase(row rowScanner) (Case, error) {
	var item Case
	err := row.Scan(&item.ID, &item.LibraryID, &item.Identifier, &item.Exercise, &item.TemplateVersionID, &item.Initialized, &item.Revision, &item.Created)
	return item, err
}
func (service *Service) Case(ctx context.Context, principal domain.Principal, identifier string) (Case, error) {
	item, err := scanCase(service.Database.Reader.QueryRowContext(ctx, "SELECT "+caseColumns+" FROM cases WHERE id=? AND deleted_at IS NULL", identifier))
	if err == sql.ErrNoRows {
		return item, notFound()
	}
	if err != nil {
		return item, err
	}
	if err = service.Read(ctx, principal, item.LibraryID, "documents.read"); err != nil {
		return Case{}, err
	}
	return item, nil
}
func (service *Service) Cases(ctx context.Context, principal domain.Principal, libraryID, query, cursor string) (Page[Case], error) {
	result := Page[Case]{Items: []Case{}}
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return result, err
	}
	if utf8.RuneCountInString(query) > 160 {
		return result, invalid("Identificador demasiado largo.")
	}
	scope := principal.User.ID + ":cases:" + libraryID + ":" + query
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT "+caseColumns+" FROM cases WHERE library_id=? AND deleted_at IS NULL AND instr(identifier_key,?)>0 AND (?='' OR created_at<? OR (created_at=? AND id<?)) ORDER BY created_at DESC,id DESC LIMIT 51", libraryID, searchKey(query), after.Time, after.Time, after.Time, after.ID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanCase(rows)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if len(result.Items) > 50 {
		result.Items = result.Items[:50]
		last := result.Items[49]
		result.Next = nextListCursor(scope, last.Created, last.ID)
	}
	return result, rows.Err()
}
func nullableID(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func caseEvent(ctx context.Context, transaction *sql.Tx, principal domain.Principal, metadata domain.RequestMetadata, kind, libraryID, caseID string, details map[string]any) error {
	return audit.Append(ctx, transaction, time.Now(), audit.Event{Type: kind, LibraryID: libraryID, CaseID: caseID, ActorUserID: principal.User.ID, SessionID: principal.SessionID, Metadata: metadata, Details: details})
}
func initializeRequirements(ctx context.Context, transaction *sql.Tx, caseID, libraryID, versionID string) error {
	var exists int
	if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM template_versions WHERE id=? AND library_id=?", versionID, libraryID).Scan(&exists); err != nil {
		return err
	}
	if exists != 1 {
		return invalid("Selecciona una versión de plantilla de esta biblioteca.")
	}
	rows, err := transaction.QueryContext(ctx, "SELECT id,category_id,document_type_id,required_count,mandatory,allows_multiple FROM template_requirements WHERE template_version_id=?", versionID)
	if err != nil {
		return err
	}
	items := []TemplateRequirement{}
	for rows.Next() {
		var item TemplateRequirement
		if err = rows.Scan(&item.ID, &item.CategoryID, &item.TypeID, &item.Count, &item.Mandatory, &item.Multiple); err != nil {
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
		if !item.Multiple {
			var count int
			if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM documents WHERE case_id=? AND document_type_id=? AND deleted_at IS NULL AND approval_status<>'cancelled'", caseID, item.TypeID).Scan(&count); err != nil {
				return err
			}
			if count > 1 {
				return invalid("El expediente ya contiene varios archivos de un tipo único.")
			}
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO case_requirements(id,case_id,library_id,category_id,document_type_id,source_template_requirement_id,required_count,mandatory,allows_multiple) VALUES(?,?,?,?,?,?,?,?,?)", domain.NewID(), caseID, libraryID, item.CategoryID, item.TypeID, item.ID, item.Count, item.Mandatory, item.Multiple); err != nil {
			return err
		}
	}
	_, err = transaction.ExecContext(ctx, "UPDATE cases SET template_version_id=?,requirements_initialized_at=?,revision=revision+1 WHERE id=?", versionID, now(), caseID)
	return err
}
func (service *Service) SaveCase(ctx context.Context, principal domain.Principal, libraryID, caseID string, input Case, revision int64, metadata domain.RequestMetadata) (string, error) {
	input.Identifier = strings.TrimSpace(input.Identifier)
	input.Exercise = strings.TrimSpace(input.Exercise)
	if !validName(input.Identifier) || utf8.RuneCountInString(input.Exercise) > 40 {
		return "", invalid("Identificador o ejercicio inválido.")
	}
	creating := caseID == ""
	if creating {
		caseID = domain.NewID()
	}
	err := service.write(ctx, principal, libraryID, "cases.manage", func(transaction *sql.Tx, current domain.Principal) error {
		settings, err := service.requireOrganization(ctx, transaction, libraryID)
		if err != nil {
			return err
		}
		if !settings.ExerciseEnabled && input.Exercise != "" {
			return invalid("Esta biblioteca no utiliza ejercicio.")
		}
		if settings.ExerciseEnabled && input.Exercise == "" {
			return invalid("Indica el ejercicio de este expediente.")
		}
		var duplicates int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM cases WHERE library_id=? AND identifier_key=? AND id<>?", libraryID, searchKey(input.Identifier), caseID).Scan(&duplicates); err != nil {
			return err
		}
		if duplicates > 0 {
			return domain.Failure("CASE_IDENTIFIER_EXISTS", "Ya existe este identificador en la biblioteca.", 409)
		}
		if creating {
			if _, err = transaction.ExecContext(ctx, "INSERT INTO cases(id,library_id,identifier,identifier_key,exercise,created_at) VALUES(?,?,?,?,?,?)", caseID, libraryID, input.Identifier, searchKey(input.Identifier), nullableID(input.Exercise), now()); err != nil {
				return err
			}
			if input.TemplateVersionID != "" {
				if err = initializeRequirements(ctx, transaction, caseID, libraryID, input.TemplateVersionID); err != nil {
					return err
				}
			}
		} else {
			result, err := transaction.ExecContext(ctx, "UPDATE cases SET identifier=?,identifier_key=?,exercise=?,revision=revision+1 WHERE id=? AND library_id=? AND revision=? AND deleted_at IS NULL", input.Identifier, searchKey(input.Identifier), nullableID(input.Exercise), caseID, libraryID, revision)
			if err != nil {
				return err
			}
			count, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if count != 1 {
				return conflict()
			}
		}
		return caseEvent(ctx, transaction, current, metadata, "case.saved", libraryID, caseID, map[string]any{"identifier": input.Identifier, "exercise": input.Exercise, "created": creating})
	})
	return caseID, err
}
func (service *Service) InitializeCase(ctx context.Context, principal domain.Principal, caseID, versionID string, metadata domain.RequestMetadata) error {
	item, err := service.Case(ctx, principal, caseID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, item.LibraryID, "requirements.edit", func(transaction *sql.Tx, current domain.Principal) error {
		if _, err := service.requireOrganization(ctx, transaction, item.LibraryID); err != nil {
			return err
		}
		var initialized bool
		if err := transaction.QueryRowContext(ctx, "SELECT requirements_initialized_at IS NOT NULL FROM cases WHERE id=?", caseID).Scan(&initialized); err != nil {
			return err
		}
		if initialized {
			return nil
		}
		if err := initializeRequirements(ctx, transaction, caseID, item.LibraryID, versionID); err != nil {
			return err
		}
		return caseEvent(ctx, transaction, current, metadata, "case.requirements_initialized", item.LibraryID, caseID, map[string]any{"template_version_id": versionID})
	})
}

type CaseRequirement struct {
	TemplateRequirement
	Revision     int64          `json:"revision"`
	CategoryName string         `json:"category_name"`
	TypeName     string         `json:"document_type_name"`
	Counts       map[string]int `json:"counts"`
	Available    int            `json:"available_count"`
	Missing      int            `json:"missing_count"`
}
type RequirementsResult struct {
	Items       []CaseRequirement `json:"items"`
	Progress    *float64          `json:"progress_percent"`
	Initialized bool              `json:"initialized"`
}

// Unmaterialized uploads stay private even when their availability is damaged.
// Staged bytes/text/metadata are visible only to their author or library reviewers.
// This predicate is applied BEFORE totals, pagination, snippets and derived counts.
const visibleDocumentSQL = "((f.availability<>'staged' AND NOT (f.storage_source='managed' AND f.primary_location_id IS NULL) AND NOT EXISTS(SELECT 1 FROM upload_items private_upload WHERE private_upload.document_id=d.id AND private_upload.status IN ('staged','retained'))) OR d.created_by=? OR EXISTS(SELECT 1 FROM library_role_assignments va JOIN role_permissions vp USING(role_id) WHERE va.user_id=? AND va.library_id=d.library_id AND vp.permission_key='documents.review'))"

func (service *Service) Requirements(ctx context.Context, principal domain.Principal, caseID string) (RequirementsResult, error) {
	result := RequirementsResult{Items: []CaseRequirement{}}
	item, err := service.Case(ctx, principal, caseID)
	if err != nil {
		return result, err
	}
	result.Initialized = item.Initialized
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT r.id,r.category_id,r.document_type_id,r.required_count,r.mandatory,r.allows_multiple,r.revision,c.name,t.name FROM case_requirements r JOIN categories c ON c.id=r.category_id JOIN document_types t ON t.id=r.document_type_id WHERE case_id=? ORDER BY c.name,t.name,r.id", caseID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var requirement CaseRequirement
		if err = rows.Scan(&requirement.ID, &requirement.CategoryID, &requirement.TypeID, &requirement.Count, &requirement.Mandatory, &requirement.Multiple, &requirement.Revision, &requirement.CategoryName, &requirement.TypeName); err != nil {
			rows.Close()
			return result, err
		}
		requirement.Counts = map[string]int{}
		result.Items = append(result.Items, requirement)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	required, fulfilled := 0, 0
	for index := range result.Items {
		requirement := &result.Items[index]
		rows, err = service.Database.Reader.QueryContext(ctx, "SELECT d.approval_status,count(*),sum(CASE WHEN f.availability='available' AND coalesce(sr.status,'')<>'inaccessible' THEN 1 ELSE 0 END),sum(CASE WHEN f.availability IN ('missing','unknown') OR sr.status='inaccessible' THEN 1 ELSE 0 END),sum(CASE WHEN d.approval_status='approved' AND (f.storage_source='linked' OR (f.availability='available' AND coalesce(sr.status,'')<>'inaccessible' AND f.integrity_status='verified')) THEN 1 ELSE 0 END) FROM documents d JOIN physical_files f ON f.id=d.physical_file_id LEFT JOIN physical_file_locations fl ON fl.id=f.primary_location_id LEFT JOIN storage_roots sr ON sr.id=fl.root_id WHERE d.case_id=? AND d.document_type_id=? AND d.deleted_at IS NULL AND d.approval_status<>'cancelled' AND "+visibleDocumentSQL+" GROUP BY d.approval_status", caseID, requirement.TypeID, principal.User.ID, principal.User.ID)
		if err != nil {
			return result, err
		}
		validApproved := 0
		for rows.Next() {
			var status string
			var count, available, missing, approved int
			if err = rows.Scan(&status, &count, &available, &missing, &approved); err != nil {
				rows.Close()
				return result, err
			}
			requirement.Counts[status] = count
			requirement.Available += available
			requirement.Missing += missing
			validApproved += approved
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return result, err
		}
		requirement.Counts["valid_approved"] = validApproved
		if requirement.Mandatory {
			required += requirement.Count
			fulfilled += min(requirement.Count, validApproved)
		}
	}
	if required > 0 {
		percent := 100 * float64(fulfilled) / float64(required)
		result.Progress = &percent
	}
	return result, nil
}
func (service *Service) UpdateRequirement(ctx context.Context, principal domain.Principal, caseID, requirementID string, count int, mandatory bool, revision int64, metadata domain.RequestMetadata) error {
	if count < 1 || count > 1000 {
		return invalid("La cantidad debe estar entre 1 y 1000.")
	}
	item, err := service.Case(ctx, principal, caseID)
	if err != nil {
		return err
	}
	return service.write(ctx, principal, item.LibraryID, "requirements.edit", func(transaction *sql.Tx, current domain.Principal) error {
		if _, err := service.requireOrganization(ctx, transaction, item.LibraryID); err != nil {
			return err
		}
		var multiple bool
		var currentRevision int64
		if err := transaction.QueryRowContext(ctx, "SELECT allows_multiple,revision FROM case_requirements WHERE id=? AND case_id=?", requirementID, caseID).Scan(&multiple, &currentRevision); err == sql.ErrNoRows {
			return notFound()
		} else if err != nil {
			return err
		}
		if currentRevision != revision {
			return conflict()
		}
		if !multiple && count != 1 {
			return invalid("Un tipo único requiere cantidad uno.")
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE case_requirements SET required_count=?,mandatory=?,revision=revision+1 WHERE id=?", count, mandatory, requirementID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE cases SET revision=revision+1 WHERE id=?", caseID); err != nil {
			return err
		}
		return caseEvent(ctx, transaction, current, metadata, "case.requirement_changed", item.LibraryID, caseID, map[string]any{"requirement_id": requirementID, "required_count": count, "mandatory": mandatory})
	})
}
func caseLibrary(ctx context.Context, query storage.Querier, caseID string) (string, error) {
	var libraryID string
	err := query.QueryRowContext(ctx, "SELECT library_id FROM cases WHERE id=? AND deleted_at IS NULL", caseID).Scan(&libraryID)
	if err == sql.ErrNoRows {
		return "", notFound()
	}
	return libraryID, err
}
