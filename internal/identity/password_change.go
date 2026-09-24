package identity

import (
	"context"
	"database/sql"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
)

func (service *Service) ChangePassword(ctx context.Context, principal domain.Principal, currentPassword, newPassword string, metadata domain.RequestMetadata) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	if err := service.reserveLoginAttempt(ctx, principal.User.Username, metadata); err != nil {
		return err
	}
	var passwordHash string
	var version int64
	if err := service.Database.Reader.QueryRowContext(ctx, "SELECT password_hash,credential_version FROM users WHERE id=?", principal.User.ID).Scan(&passwordHash, &version); err != nil {
		return err
	}
	valid, err := service.Hasher.Verify(ctx, currentPassword, passwordHash)
	if err != nil {
		return err
	}
	if !valid {
		if err := service.Database.Write(ctx, func(transaction *sql.Tx) error {
			if err := service.penalizeLogin(ctx, transaction, principal.User.Username); err != nil {
				return err
			}
			return service.recordAttempt(ctx, transaction, principal.User.Username, principal.User.ID, "failed", metadata)
		}); err != nil {
			return err
		}
		return domain.Failure("CURRENT_PASSWORD_INCORRECT", "La contraseña actual no es correcta.", 403)
	}
	newHash, err := service.Hasher.Hash(ctx, newPassword)
	if err != nil {
		return err
	}
	return service.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		if current.CredentialVersion != version {
			return domain.Failure("SESSION_REVOKED", "Tu contraseña cambió. Inicia sesión nuevamente.", 401)
		}
		if err := service.releaseSuccessfulAttempt(ctx, transaction, principal.User.Username, metadata); err != nil {
			return err
		}
		return service.setPassword(ctx, transaction, current.User.ID, newHash, current, metadata, "password.changed")
	})
}

func (service *Service) ResetPassword(ctx context.Context, principal domain.Principal, userID, newPassword string, metadata domain.RequestMetadata) error {
	if !principal.Can("users.reset_password") {
		return domain.Failure("FORBIDDEN", "No tienes permiso para restablecer contraseñas.", 403)
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	newHash, err := service.Hasher.Hash(ctx, newPassword)
	if err != nil {
		return err
	}
	return service.AuthorizedWrite(ctx, principal, "users.reset_password", func(transaction *sql.Tx, current domain.Principal) error {
		if _, err := UserByID(ctx, transaction, userID); err != nil {
			return err
		}
		return service.setPassword(ctx, transaction, userID, newHash, current, metadata, "password.reset")
	})
}

func (service *Service) setPassword(ctx context.Context, transaction *sql.Tx, userID, passwordHash string, actor domain.Principal, metadata domain.RequestMetadata, eventType string) error {
	if _, err := transaction.ExecContext(ctx, "UPDATE users SET password_hash=?,credential_version=credential_version+1,revision=revision+1,updated_at=? WHERE id=?", passwordHash, domain.Timestamp(service.Now()), userID); err != nil {
		return err
	}
	if err := service.closeUserSessions(ctx, transaction, userID, "password_changed", metadata, actor); err != nil {
		return err
	}
	return audit.Append(ctx, transaction, service.Now(), audit.Event{Type: eventType, ActorUserID: actor.User.ID, SessionID: actor.SessionID, Metadata: metadata, Details: map[string]any{"user_id": userID}})
}

// RecoverAdministrator is available only from the local CLI while holding the state lock.
func (service *Service) RecoverAdministrator(ctx context.Context, username, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	passwordHash, err := service.Hasher.Hash(ctx, newPassword)
	if err != nil {
		return err
	}
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		var userID string
		if err := transaction.QueryRowContext(ctx, `SELECT users.id FROM users JOIN global_role_assignments ON users.id=user_id
			WHERE username_key=? AND role_id='installation_admin'`, normalizeUsername(username)).Scan(&userID); err != nil {
			return domain.Failure("ADMIN_NOT_FOUND", "No se encontró ese administrador.", 404)
		}
		if _, err := transaction.ExecContext(ctx, "UPDATE users SET disabled_at=NULL WHERE id=?", userID); err != nil {
			return err
		}
		return service.setPassword(ctx, transaction, userID, passwordHash, domain.Principal{}, domain.RequestMetadata{RequestID: domain.NewID(), ObservedIP: "local"}, "password.local_recovery")
	})
}
