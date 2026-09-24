// Package libraries owns linked libraries and their authorization boundaries.
package libraries

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

type Service struct {
	Identity *identity.Service
	Database *storage.Database
}

func New(identityService *identity.Service) *Service {
	return &Service{Identity: identityService, Database: identityService.Database}
}
func invalid(message string) error { return domain.Failure("INVALID_REQUEST", message, 422) }
func notFound() error              { return domain.Failure("NOT_FOUND", "No se encontró el recurso.", 404) }
func now() string                  { return domain.Timestamp(time.Now()) }
func (service *Service) require(ctx context.Context, query storage.Querier, principal domain.Principal, libraryID, permission string) error {
	var permitted int
	err := query.QueryRowContext(ctx, "SELECT count(*) FROM library_role_assignments a JOIN role_permissions p USING(role_id) JOIN libraries l ON l.id=a.library_id WHERE a.user_id=? AND a.library_id=? AND p.permission_key=? AND l.disabled_at IS NULL", principal.User.ID, libraryID, permission).Scan(&permitted)
	if err != nil {
		return err
	}
	if permitted == 0 {
		return notFound()
	}
	return nil
}
func (service *Service) Read(ctx context.Context, principal domain.Principal, libraryID, permission string) error {
	current, err := service.Identity.LoadPrincipal(ctx, service.Database.Reader, principal.TokenDigest)
	if err != nil {
		return err
	}
	if err = licensing.Check(licensing.ReadDocuments); err != nil {
		return err
	}
	return service.require(ctx, service.Database.Reader, current, libraryID, permission)
}
func (service *Service) write(ctx context.Context, principal domain.Principal, libraryID, permission string, operation func(*sql.Tx, domain.Principal) error) error {
	return service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		if err := licensing.Check(licensing.WriteDocuments); err != nil {
			return err
		}
		if libraryID != "" {
			if err := service.require(ctx, transaction, current, libraryID, permission); err != nil {
				return err
			}
		}
		return operation(transaction, current)
	})
}
func record(ctx context.Context, transaction *sql.Tx, principal domain.Principal, metadata domain.RequestMetadata, kind, libraryID, documentID string, details map[string]any) error {
	return audit.Append(ctx, transaction, time.Now(), audit.Event{Type: kind, LibraryID: libraryID, DocumentID: documentID, ActorUserID: principal.User.ID, SystemActor: principal.User.ID == "", SessionID: principal.SessionID, Metadata: metadata, Details: details})
}

type Library struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Mode         string   `json:"mode"`
	Revision     int64    `json:"revision"`
	OCRLanguages string   `json:"ocr_languages"`
	Permissions  []string `json:"permissions"`
}

func (service *Service) Libraries(ctx context.Context, principal domain.Principal) ([]Library, error) {
	result := []Library{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,name,mode,revision,ocr_languages FROM libraries WHERE disabled_at IS NULL AND (? OR EXISTS(SELECT 1 FROM library_role_assignments WHERE library_id=libraries.id AND user_id=?)) ORDER BY name,id", principal.Can("permissions.manage_global"), principal.User.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var library Library
		if err = rows.Scan(&library.ID, &library.Name, &library.Mode, &library.Revision, &library.OCRLanguages); err != nil {
			rows.Close()
			return nil, err
		}
		result = append(result, library)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for index := range result {
		permissions, err := service.permissions(ctx, service.Database.Reader, principal.User.ID, result[index].ID)
		if err != nil {
			return nil, err
		}
		result[index].Permissions = permissions
	}
	return result, nil
}
func (service *Service) permissions(ctx context.Context, query storage.Querier, userID, libraryID string) ([]string, error) {
	permissions := []string{}
	rows, err := query.QueryContext(ctx, "SELECT DISTINCT permission_key FROM role_permissions JOIN library_role_assignments USING(role_id) WHERE user_id=? AND library_id=? ORDER BY permission_key", userID, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var permission string
		if err = rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	return permissions, rows.Err()
}
func (service *Service) Create(ctx context.Context, principal domain.Principal, name, mode, managerID, requestID string, metadata domain.RequestMetadata) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 160 || mode != "linked" || managerID == "" {
		return "", invalid("Escribe un nombre y selecciona gestor. H3 admite bibliotecas vinculadas.")
	}
	if len(requestID) < 16 || len(requestID) > 100 {
		return "", invalid("Se requiere una clave de idempotencia de 16 a 100 caracteres.")
	}
	identifier := domain.NewID()
	digest := domain.Digest(name + "\n" + mode + "\n" + managerID)
	err := service.write(ctx, principal, "", "", func(transaction *sql.Tx, current domain.Principal) error {
		if !current.Can("libraries.create") || !current.Can("permissions.manage_global") {
			return domain.Failure("FORBIDDEN", "No tienes permiso para crear bibliotecas y asignar gestores.", 403)
		}
		var existingDigest, existingID string
		err := transaction.QueryRowContext(ctx, "SELECT request_digest,resource_id FROM idempotency_requests WHERE actor_user_id=? AND operation='library.create' AND idempotency_key=?", current.User.ID, requestID).Scan(&existingDigest, &existingID)
		if err == nil {
			if digest != existingDigest {
				return domain.Failure("IDEMPOTENCY_CONFLICT", "La clave ya se utilizó para otra solicitud.", 409)
			}
			identifier = existingID
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		var enabled int
		if err = transaction.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id=? AND disabled_at IS NULL", managerID).Scan(&enabled); err != nil {
			return err
		}
		if enabled == 0 {
			return invalid("Selecciona un usuario habilitado.")
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES(?,?,'linked',?,?)", identifier, name, now(), now()); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO library_role_assignments(user_id,library_id,role_id) VALUES(?,?,'library_manager')", managerID, identifier); err != nil {
			return err
		}
		if _, err = transaction.ExecContext(ctx, "INSERT INTO idempotency_requests VALUES(?,'library.create',?,?,?,'complete',?)", current.User.ID, requestID, digest, identifier, now()); err != nil {
			return err
		}
		return record(ctx, transaction, current, metadata, "library.created", identifier, "", map[string]any{"name": name, "manager_user_id": managerID})
	})
	return identifier, err
}
func (service *Service) Update(ctx context.Context, principal domain.Principal, libraryID, name, languages string, revision int64, metadata domain.RequestMetadata) error {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 160 || (languages != "spa" && languages != "eng" && languages != "spa+eng") {
		return invalid("Nombre o idioma OCR inválido.")
	}
	return service.write(ctx, principal, libraryID, "libraries.configure", func(transaction *sql.Tx, current domain.Principal) error {
		var previousLanguages string
		if err := transaction.QueryRowContext(ctx, "SELECT ocr_languages FROM libraries WHERE id=?", libraryID).Scan(&previousLanguages); err != nil {
			return err
		}
		result, err := transaction.ExecContext(ctx, "UPDATE libraries SET name=?,ocr_languages=?,revision=revision+1,current_configuration_revision=current_configuration_revision+1,updated_at=? WHERE id=? AND revision=?", name, languages, now(), libraryID, revision)
		if err != nil {
			return err
		}
		count, _ := result.RowsAffected()
		if count != 1 {
			return domain.Failure("VERSION_CONFLICT", "La biblioteca cambió. Recarga la página.", 412)
		}
		if previousLanguages != languages {
			rows, err := transaction.QueryContext(ctx, "SELECT f.id,f.current_content_version_id FROM physical_files f JOIN documents d ON d.physical_file_id=f.id WHERE f.library_id=? AND f.current_content_version_id IS NOT NULL AND d.deleted_at IS NULL", libraryID)
			if err != nil {
				return err
			}
			for rows.Next() {
				var fileID, versionID string
				if err = rows.Scan(&fileID, &versionID); err != nil {
					rows.Close()
					return err
				}
				jobID := domain.NewID()
				if _, err = transaction.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','queued',?,?)", jobID, libraryID, fileID, versionID, jobID, now(), now()); err != nil {
					rows.Close()
					return err
				}
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			if _, err = transaction.ExecContext(ctx, "UPDATE physical_files SET extraction_freshness='stale' WHERE library_id=? AND indexed_extraction_id IS NOT NULL", libraryID); err != nil {
				return err
			}
		}
		return record(ctx, transaction, current, metadata, "library.updated", libraryID, "", map[string]any{"name": name, "ocr_languages": languages})
	})
}
func (service *Service) SetMember(ctx context.Context, principal domain.Principal, libraryID, userID string, roles []string, metadata domain.RequestMetadata) error {
	unique := map[string]bool{}
	for _, role := range roles {
		if (role != "library_reader" && role != "library_manager" && role != "library_auditor") || unique[role] {
			return invalid("Perfil de biblioteca inválido o repetido.")
		}
		unique[role] = true
	}
	return service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		// Security administrators may restore access, but never inherit document permissions.
		if !current.Can("permissions.manage_global") {
			if err := service.require(ctx, transaction, current, libraryID, "permissions.manage_library"); err != nil {
				return err
			}
		}
		var existing int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM libraries WHERE id=?", libraryID).Scan(&existing); err != nil {
			return err
		}
		if existing == 0 {
			return notFound()
		}
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id=? AND disabled_at IS NULL", userID).Scan(&existing); err != nil {
			return err
		}
		if existing == 0 {
			return invalid("Usuario inexistente o deshabilitado.")
		}
		if !unique["library_manager"] {
			var remaining int
			if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM library_role_assignments a JOIN users u ON u.id=a.user_id WHERE library_id=? AND role_id='library_manager' AND user_id<>? AND u.disabled_at IS NULL", libraryID, userID).Scan(&remaining); err != nil {
				return err
			}
			if remaining == 0 {
				return invalid("Conserva al menos un gestor habilitado.")
			}
		}
		if _, err := transaction.ExecContext(ctx, "DELETE FROM library_role_assignments WHERE library_id=? AND user_id=?", libraryID, userID); err != nil {
			return err
		}
		for _, role := range roles {
			if _, err := transaction.ExecContext(ctx, "INSERT INTO library_role_assignments(user_id,library_id,role_id) VALUES(?,?,?)", userID, libraryID, role); err != nil {
				return err
			}
		}
		return record(ctx, transaction, current, metadata, "library.permissions_changed", libraryID, "", map[string]any{"user_id": userID, "role_ids": roles})
	})
}
func encode(value any) string { contents, _ := json.Marshal(value); return string(contents) }
