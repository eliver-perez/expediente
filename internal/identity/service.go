package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"regexp"
	"strings"
	"time"
	"unicode"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

const SessionReplacedMessage = "Tu sesión fue cerrada porque se inició sesión con tu cuenta desde otro equipo o navegador"

type Service struct {
	Database *storage.Database
	Hasher   *PasswordHasher
	Config   config.Config
	Now      func() time.Time
}

func New(database *storage.Database, configuration config.Config) (*Service, error) {
	hasher, err := NewPasswordHasher()
	if err != nil {
		return nil, err
	}
	return &Service{Database: database, Hasher: hasher, Config: configuration, Now: time.Now}, nil
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._@-]{2,79}$`)

func normalizeUsername(username string) string { return strings.ToLower(strings.TrimSpace(username)) }
func validateUsername(username string) error {
	if !usernamePattern.MatchString(normalizeUsername(username)) {
		return domain.Failure("INVALID_USERNAME", "El usuario debe tener entre 3 y 80 caracteres: letras, números, punto, guion, @ o guion bajo.", 422)
	}
	return nil
}

func sanitizeText(value string, maximum int) string {
	characters := make([]rune, 0, maximum)
	for _, character := range value {
		if !unicode.IsControl(character) {
			characters = append(characters, character)
		}
		if len(characters) >= maximum {
			break
		}
	}
	return string(characters)
}

func CSRFToken(sessionToken string) string {
	mac := hmac.New(sha256.New, []byte(sessionToken))
	mac.Write([]byte("gestor-documental:csrf:v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyCSRF(principal domain.Principal, value string) bool {
	return value != "" && subtle.ConstantTimeCompare([]byte(domain.Digest(value)), []byte(principal.CSRFTokenDigest)) == 1
}

func (service *Service) Bootstrap(ctx context.Context, username, displayName, password string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	if strings.TrimSpace(displayName) == "" {
		return domain.Failure("INVALID_REQUEST", "Escribe un nombre.", 422)
	}
	passwordHash, err := service.Hasher.Hash(ctx, password)
	if err != nil {
		return err
	}
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		var count int
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM bootstrap_state").Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return domain.Failure("BOOTSTRAP_COMPLETE", "El administrador inicial ya fue creado.", 409)
		}
		if err := transaction.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return domain.Failure("BOOTSTRAP_CONFLICT", "Existen usuarios; usa recuperación local.", 409)
		}
		now := service.Now()
		userID := domain.NewID()
		if _, err := transaction.ExecContext(ctx, `INSERT INTO users (id,username,username_key,display_name,password_hash,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
			userID, normalizeUsername(username), normalizeUsername(username), sanitizeText(displayName, 120), passwordHash, domain.Timestamp(now), domain.Timestamp(now)); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO global_role_assignments(user_id,role_id) VALUES (?, 'installation_admin')", userID); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, "INSERT INTO bootstrap_state VALUES (1,?,?)", userID, domain.Timestamp(now)); err != nil {
			return err
		}
		return audit.Append(ctx, transaction, now, audit.Event{Type: "installation.bootstrapped", Metadata: domain.RequestMetadata{RequestID: domain.NewID(), ObservedIP: "local"}, Details: map[string]any{"user_id": userID}})
	})
}

func (service *Service) LoadPrincipal(ctx context.Context, query storage.Querier, tokenDigest string) (domain.Principal, error) {
	var principal domain.Principal
	var startedAt, lastActivity, closedAt, closeReason, disabledAt string
	var userCredentialVersion int64
	err := query.QueryRowContext(ctx, `SELECT sessions.id,sessions.user_id,sessions.credential_version,sessions.csrf_token_digest,
		sessions.expires_at,sessions.started_at,sessions.last_activity_at,coalesce(sessions.closed_at,''),coalesce(sessions.close_reason,''),
		users.username,users.display_name,users.revision,users.created_at,coalesce(users.disabled_at,''),users.credential_version
		FROM sessions JOIN users ON users.id=sessions.user_id WHERE sessions.token_digest=?`, tokenDigest).Scan(
		&principal.SessionID, &principal.User.ID, &principal.CredentialVersion, &principal.CSRFTokenDigest, &principal.ExpiresAt,
		&startedAt, &lastActivity, &closedAt, &closeReason, &principal.User.Username, &principal.User.DisplayName, &principal.User.Revision, &principal.User.CreatedAt, &disabledAt, &userCredentialVersion)
	if err == sql.ErrNoRows {
		return principal, domain.Failure("SESSION_REQUIRED", "Inicia sesión para continuar.", 401)
	}
	if err != nil {
		return principal, err
	}
	principal.TokenDigest = tokenDigest
	if closedAt != "" {
		if closeReason == "new_login" {
			return principal, domain.Failure("SESSION_REPLACED", SessionReplacedMessage, 401)
		}
		if closeReason == "expired" {
			return principal, domain.Failure("SESSION_EXPIRED", "Tu sesión expiró. Inicia sesión nuevamente.", 401)
		}
		return principal, domain.Failure("SESSION_REVOKED", "Tu sesión fue cerrada. Inicia sesión nuevamente.", 401)
	}
	if disabledAt != "" || userCredentialVersion != principal.CredentialVersion {
		return principal, domain.Failure("SESSION_REVOKED", "Tu sesión fue cerrada. Inicia sesión nuevamente.", 401)
	}
	now := service.Now()
	if principal.ExpiresAt <= domain.Timestamp(now) || lastActivity <= domain.Timestamp(now.Add(-time.Duration(service.Config.SessionIdleMinutes)*time.Minute)) {
		return principal, domain.Failure("SESSION_EXPIRED", "Tu sesión expiró. Inicia sesión nuevamente.", 401)
	}
	principal.User.RoleIDs, principal.User.Permissions, err = loadRoles(ctx, query, principal.User.ID)
	return principal, err
}

func loadRoles(ctx context.Context, query storage.Querier, userID string) ([]string, []string, error) {
	roleIDs := []string{}
	permissions := []string{}
	rows, err := query.QueryContext(ctx, "SELECT role_id FROM global_role_assignments WHERE user_id=? ORDER BY role_id", userID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			rows.Close()
			return nil, nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	rows, err = query.QueryContext(ctx, `SELECT DISTINCT permission_key FROM role_permissions JOIN global_role_assignments USING(role_id) WHERE user_id=? ORDER BY permission_key`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, nil, err
		}
		permissions = append(permissions, permission)
	}
	return roleIDs, permissions, rows.Err()
}

func (service *Service) Authenticate(ctx context.Context, token string, touch bool, metadata domain.RequestMetadata) (domain.Principal, error) {
	if len(token) != 43 {
		return domain.Principal{}, domain.Failure("SESSION_REQUIRED", "Inicia sesión para continuar.", 401)
	}
	principal, err := service.LoadPrincipal(ctx, service.Database.Reader, domain.Digest(token))
	if failure, ok := err.(*domain.Error); ok && failure.Code == "SESSION_EXPIRED" {
		closeError := service.Database.Write(ctx, func(transaction *sql.Tx) error {
			return service.closeSession(ctx, transaction, principal.SessionID, "expired", metadata, domain.Principal{})
		})
		if closeError != nil {
			return principal, closeError
		}
	}
	if err != nil || !touch {
		return principal, err
	}
	err = service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if _, err := service.LoadPrincipal(ctx, transaction, principal.TokenDigest); err != nil {
			return err
		}
		_, err := transaction.ExecContext(ctx, "UPDATE sessions SET last_activity_at=? WHERE id=? AND last_activity_at<?", domain.Timestamp(service.Now()), principal.SessionID, domain.Timestamp(service.Now().Add(-time.Minute)))
		return err
	})
	return principal, err
}

func (service *Service) closeSession(ctx context.Context, transaction *sql.Tx, sessionID, reason string, metadata domain.RequestMetadata, actor domain.Principal) error {
	var userID string
	err := transaction.QueryRowContext(ctx, "SELECT user_id FROM sessions WHERE id=? AND closed_at IS NULL", sessionID).Scan(&userID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, "UPDATE sessions SET closed_at=?,close_reason=? WHERE id=? AND closed_at IS NULL", domain.Timestamp(service.Now()), reason, sessionID); err != nil {
		return err
	}
	return audit.Append(ctx, transaction, service.Now(), audit.Event{Type: "session.closed", ActorUserID: actor.User.ID, SystemActor: actor.User.ID == "", SessionID: actor.SessionID, Metadata: metadata, Details: map[string]any{"reason": reason, "user_id": userID, "closed_session_id": sessionID}})
}

func (service *Service) AuthorizedWrite(ctx context.Context, principal domain.Principal, permission string, operation func(*sql.Tx, domain.Principal) error) error {
	return service.Database.Write(ctx, func(transaction *sql.Tx) error {
		current, err := service.LoadPrincipal(ctx, transaction, principal.TokenDigest)
		if err != nil {
			return err
		}
		if permission != "" && !current.Can(permission) {
			return domain.Failure("FORBIDDEN", "No tienes permiso para esta operación.", 403)
		}
		return operation(transaction, current)
	})
}
