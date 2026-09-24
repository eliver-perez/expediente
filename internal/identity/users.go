package identity

import (
	"context"
	"database/sql"
	"strings"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

func UserByID(ctx context.Context, query storage.Querier, userID string) (domain.User, error) {
	var user domain.User
	err := query.QueryRowContext(ctx, "SELECT id,username,display_name,disabled_at IS NOT NULL,revision,created_at FROM users WHERE id=?", userID).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Disabled, &user.Revision, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return user, domain.Failure("NOT_FOUND", "No se encontró el usuario.", 404)
	}
	if err != nil {
		return user, err
	}
	user.RoleIDs, user.Permissions, err = loadRoles(ctx, query, userID)
	return user, err
}

func (service *Service) CreateUser(ctx context.Context, principal domain.Principal, username, displayName, password string, metadata domain.RequestMetadata) (domain.User, error) {
	if !principal.Can("users.manage") {
		return domain.User{}, domain.Failure("FORBIDDEN", "No tienes permiso para crear usuarios.", 403)
	}
	if err := validateUsername(username); err != nil {
		return domain.User{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return domain.User{}, err
	}
	displayName = strings.TrimSpace(sanitizeText(displayName, 120))
	if displayName == "" {
		return domain.User{}, domain.Failure("INVALID_REQUEST", "Escribe un nombre.", 422)
	}
	passwordHash, err := service.Hasher.Hash(ctx, password)
	if err != nil {
		return domain.User{}, err
	}
	var created domain.User
	err = service.AuthorizedWrite(ctx, principal, "users.manage", func(transaction *sql.Tx, current domain.Principal) error {
		var count int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE username_key=?", normalizeUsername(username)).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return domain.Failure("USERNAME_EXISTS", "El nombre de usuario ya está registrado.", 409)
		}
		userID := domain.NewID()
		now := domain.Timestamp(service.Now())
		if _, err := transaction.ExecContext(ctx, "INSERT INTO users(id,username,username_key,display_name,password_hash,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", userID, normalizeUsername(username), normalizeUsername(username), displayName, passwordHash, now, now); err != nil {
			return err
		}
		if err := audit.Append(ctx, transaction, service.Now(), audit.Event{Type: "user.created", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata, Details: map[string]any{"user_id": userID}}); err != nil {
			return err
		}
		created, err = UserByID(ctx, transaction, userID)
		return err
	})
	return created, err
}

func checkRevision(user domain.User, expected int64) error {
	if expected < 1 {
		return domain.Failure("PRECONDITION_REQUIRED", "Falta la revisión del usuario.", 428)
	}
	if user.Revision != expected {
		return domain.Failure("VERSION_CONFLICT", "El usuario cambió. Actualiza la página.", 412)
	}
	return nil
}

func (service *Service) UpdateUser(ctx context.Context, principal domain.Principal, userID string, expected int64, displayName *string, disabled *bool, metadata domain.RequestMetadata) (domain.User, error) {
	var updated domain.User
	err := service.AuthorizedWrite(ctx, principal, "users.manage", func(transaction *sql.Tx, current domain.Principal) error {
		user, err := UserByID(ctx, transaction, userID)
		if err != nil {
			return err
		}
		if err := checkRevision(user, expected); err != nil {
			return err
		}
		if displayName != nil {
			user.DisplayName = strings.TrimSpace(sanitizeText(*displayName, 120))
			if user.DisplayName == "" {
				return domain.Failure("INVALID_REQUEST", "Escribe un nombre.", 422)
			}
		}
		if disabled != nil {
			user.Disabled = *disabled
		}
		var disabledAt any
		if user.Disabled {
			disabledAt = domain.Timestamp(service.Now())
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE users SET display_name=?,disabled_at=?,revision=revision+1,updated_at=? WHERE id=?", user.DisplayName, disabledAt, domain.Timestamp(service.Now()), userID); err != nil {
			return err
		}
		if err := ensureAdministrator(ctx, transaction); err != nil {
			return err
		}
		if user.Disabled {
			if err := service.closeUserSessions(ctx, transaction, userID, "admin_revoked", metadata, current); err != nil {
				return err
			}
		}
		if err := audit.Append(ctx, transaction, service.Now(), audit.Event{Type: "user.updated", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata, Details: map[string]any{"user_id": userID, "disabled": user.Disabled}}); err != nil {
			return err
		}
		updated, err = UserByID(ctx, transaction, userID)
		return err
	})
	return updated, err
}

func ensureAdministrator(ctx context.Context, transaction *sql.Tx) error {
	var count int
	if err := transaction.QueryRowContext(ctx, `SELECT count(*) FROM users JOIN global_role_assignments ON users.id=user_id WHERE role_id='installation_admin' AND disabled_at IS NULL`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return domain.Failure("LAST_ADMIN_REQUIRED", "Debe conservarse al menos un administrador habilitado.", 409)
	}
	return nil
}

func (service *Service) SetGlobalRoles(ctx context.Context, principal domain.Principal, userID string, expected int64, roleIDs []string, metadata domain.RequestMetadata) (domain.User, error) {
	var updated domain.User
	err := service.AuthorizedWrite(ctx, principal, "permissions.manage_global", func(transaction *sql.Tx, current domain.Principal) error {
		user, err := UserByID(ctx, transaction, userID)
		if err != nil {
			return err
		}
		if err := checkRevision(user, expected); err != nil {
			return err
		}
		if len(roleIDs) > 2 {
			return domain.Failure("INVALID_REQUEST", "Roles no válidos.", 400)
		}
		seen := map[string]bool{}
		for _, roleID := range roleIDs {
			if (roleID != "installation_admin" && roleID != "access_auditor") || seen[roleID] {
				return domain.Failure("INVALID_REQUEST", "Roles no válidos.", 400)
			}
			seen[roleID] = true
		}
		if _, err := transaction.ExecContext(ctx, "DELETE FROM global_role_assignments WHERE user_id=?", userID); err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			if _, err := transaction.ExecContext(ctx, "INSERT INTO global_role_assignments(user_id,role_id) VALUES (?,?)", userID, roleID); err != nil {
				return err
			}
		}
		if err := ensureAdministrator(ctx, transaction); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE users SET revision=revision+1,updated_at=? WHERE id=?", domain.Timestamp(service.Now()), userID); err != nil {
			return err
		}
		if err := audit.Append(ctx, transaction, service.Now(), audit.Event{Type: "permissions.global_changed", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata, Details: map[string]any{"user_id": userID, "role_ids": roleIDs}}); err != nil {
			return err
		}
		updated, err = UserByID(ctx, transaction, userID)
		return err
	})
	return updated, err
}

func (service *Service) closeUserSessions(ctx context.Context, transaction *sql.Tx, userID, reason string, metadata domain.RequestMetadata, actor domain.Principal) error {
	rows, err := transaction.QueryContext(ctx, "SELECT id FROM sessions WHERE user_id=? AND closed_at IS NULL", userID)
	if err != nil {
		return err
	}
	identifiers := []string{}
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			rows.Close()
			return err
		}
		identifiers = append(identifiers, sessionID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, sessionID := range identifiers {
		if err := service.closeSession(ctx, transaction, sessionID, reason, metadata, actor); err != nil {
			return err
		}
	}
	return nil
}

func (service *Service) RevokeSessions(ctx context.Context, principal domain.Principal, userID, reason string, metadata domain.RequestMetadata) error {
	return service.AuthorizedWrite(ctx, principal, "sessions.revoke", func(transaction *sql.Tx, current domain.Principal) error {
		if _, err := UserByID(ctx, transaction, userID); err != nil {
			return err
		}
		if err := service.closeUserSessions(ctx, transaction, userID, "admin_revoked", metadata, current); err != nil {
			return err
		}
		return audit.Append(ctx, transaction, service.Now(), audit.Event{Type: "user.sessions_revoked", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata, Details: map[string]any{"user_id": userID, "reason": sanitizeText(reason, 500)}})
	})
}
