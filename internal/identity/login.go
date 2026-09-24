package identity

import (
	"context"
	"database/sql"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
)

type LoginResult struct {
	User         domain.User `json:"user"`
	ExpiresAt    string      `json:"session_expires_at"`
	CSRFToken    string      `json:"csrf_token"`
	SessionToken string      `json:"-"`
}

func (service *Service) Login(ctx context.Context, username, password string, metadata domain.RequestMetadata) (LoginResult, error) {
	result := LoginResult{}
	identifier := normalizeUsername(sanitizeText(username, 80))
	if err := service.reserveLoginAttempt(ctx, identifier, metadata); err != nil {
		return result, err
	}
	var userID, passwordHash, disabledAt string
	var credentialVersion int64
	err := service.Database.Reader.QueryRowContext(ctx, "SELECT id,password_hash,credential_version,coalesce(disabled_at,'') FROM users WHERE username_key=?", normalizeUsername(username)).Scan(&userID, &passwordHash, &credentialVersion, &disabledAt)
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	valid, err := service.Hasher.Verify(ctx, password, passwordHash)
	if err != nil {
		return result, err
	}
	valid = valid && userID != "" && disabledAt == ""
	var outcome error
	err = service.Database.Write(ctx, func(transaction *sql.Tx) error {
		if valid {
			var currentVersion int64
			var currentDisabled string
			if err := transaction.QueryRowContext(ctx, "SELECT credential_version,coalesce(disabled_at,'') FROM users WHERE id=?", userID).Scan(&currentVersion, &currentDisabled); err != nil {
				return err
			}
			valid = currentVersion == credentialVersion && currentDisabled == ""
		}
		if !valid {
			outcome = domain.Failure("AUTHENTICATION_FAILED", "Usuario o contraseña incorrectos.", 401)
			if err := service.penalizeLogin(ctx, transaction, identifier); err != nil {
				return err
			}
			return service.recordAttempt(ctx, transaction, identifier, userID, "failed", metadata)
		}
		var previousSessionID, previousExpires, previousActivity string
		err := transaction.QueryRowContext(ctx, "SELECT id,expires_at,last_activity_at FROM sessions WHERE user_id=? AND closed_at IS NULL", userID).Scan(&previousSessionID, &previousExpires, &previousActivity)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			reason := "new_login"
			if previousExpires <= domain.Timestamp(service.Now()) || previousActivity <= domain.Timestamp(service.Now().Add(-time.Duration(service.Config.SessionIdleMinutes)*time.Minute)) {
				reason = "expired"
			}
			if err := service.closeSession(ctx, transaction, previousSessionID, reason, metadata, domain.Principal{User: domain.User{ID: userID}}); err != nil {
				return err
			}
		}
		now := service.Now()
		sessionID := domain.NewID()
		result.SessionToken = domain.RandomToken()
		result.CSRFToken = CSRFToken(result.SessionToken)
		result.ExpiresAt = domain.Timestamp(now.Add(time.Duration(service.Config.SessionAbsoluteHours) * time.Hour))
		if _, err := transaction.ExecContext(ctx, `INSERT INTO sessions
			(id,user_id,token_digest,csrf_token_digest,credential_version,started_at,last_activity_at,expires_at,observed_ip,user_agent)
			VALUES (?,?,?,?,?,?,?,?,?,?)`, sessionID, userID, domain.Digest(result.SessionToken), domain.Digest(result.CSRFToken), credentialVersion,
			domain.Timestamp(now), domain.Timestamp(now), result.ExpiresAt, metadata.ObservedIP, sanitizeText(metadata.UserAgent, 512)); err != nil {
			return err
		}
		if err := service.releaseSuccessfulAttempt(ctx, transaction, identifier, metadata); err != nil {
			return err
		}
		if err := service.recordAttempt(ctx, transaction, identifier, userID, "success", metadata); err != nil {
			return err
		}
		if err := audit.Append(ctx, transaction, now, audit.Event{Type: "session.started", ActorUserID: userID, SessionID: sessionID, Metadata: metadata}); err != nil {
			return err
		}
		principal, err := service.LoadPrincipal(ctx, transaction, domain.Digest(result.SessionToken))
		if err != nil {
			return err
		}
		result.User = principal.User
		return nil
	})
	if err != nil {
		return LoginResult{}, err
	}
	if outcome != nil {
		return LoginResult{}, outcome
	}
	return result, nil
}

func (service *Service) recordAttempt(ctx context.Context, transaction *sql.Tx, identifier, userID, outcome string, metadata domain.RequestMetadata) error {
	var nullableUserID any
	if userID != "" {
		nullableUserID = userID
	}
	now := service.Now()
	if _, err := transaction.ExecContext(ctx, `INSERT INTO authentication_attempts
		(id,attempted_identifier,user_id,occurred_at,observed_ip,user_agent,outcome,request_id) VALUES (?,?,?,?,?,?,?,?)`,
		domain.NewID(), identifier, nullableUserID, domain.Timestamp(now), metadata.ObservedIP, sanitizeText(metadata.UserAgent, 512), outcome, metadata.RequestID); err != nil {
		return err
	}
	actorUserID := ""
	if outcome == "success" {
		actorUserID = userID
	}
	return audit.Append(ctx, transaction, now, audit.Event{Type: "authentication." + outcome, ActorUserID: actorUserID, Metadata: metadata,
		Details: map[string]any{"attempted_identifier": identifier, "outcome": outcome}})
}

// Reserve both buckets before hashing so simultaneous failures cannot bypass limits.
// Reservations survive crashes; successful authentication releases only its own charge.
func (service *Service) reserveLoginAttempt(ctx context.Context, identifier string, metadata domain.RequestMetadata) error {
	var denied bool
	err := service.Database.Write(ctx, func(transaction *sql.Tx) error {
		now := service.Now()
		cutoff := domain.Timestamp(now.Add(-time.Duration(service.Config.LoginWindowMinutes) * time.Minute))
		buckets := []struct {
			scope, key string
			limit      int
		}{
			{"ip", domain.Digest(metadata.ObservedIP), service.Config.LoginIPLimit},
			{"identifier", domain.Digest(identifier), service.Config.LoginIdentifierLimit},
		}
		for _, bucket := range buckets {
			if _, err := transaction.ExecContext(ctx, `INSERT INTO authentication_throttles(scope,key_digest,attempt_count,window_started_at,next_allowed_at)
				VALUES (?,?,0,?,?) ON CONFLICT(scope,key_digest) DO UPDATE SET attempt_count=0,window_started_at=excluded.window_started_at,next_allowed_at=excluded.next_allowed_at
				WHERE authentication_throttles.window_started_at<=?`, bucket.scope, bucket.key, domain.Timestamp(now), domain.Timestamp(now), cutoff); err != nil {
				return err
			}
			var count int
			var nextAllowed string
			if err := transaction.QueryRowContext(ctx, "SELECT attempt_count,next_allowed_at FROM authentication_throttles WHERE scope=? AND key_digest=?", bucket.scope, bucket.key).Scan(&count, &nextAllowed); err != nil {
				return err
			}
			if count >= bucket.limit || nextAllowed > domain.Timestamp(now) {
				denied = true
			}
		}
		if denied {
			return service.recordAttempt(ctx, transaction, identifier, "", "rate_limited", metadata)
		}
		for _, bucket := range buckets {
			if _, err := transaction.ExecContext(ctx, "UPDATE authentication_throttles SET attempt_count=attempt_count+1 WHERE scope=? AND key_digest=?", bucket.scope, bucket.key); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if denied {
		return domain.Failure("RATE_LIMITED", "Demasiados intentos. Espera antes de volver a intentar.", 429)
	}
	return nil
}

func (service *Service) penalizeLogin(ctx context.Context, transaction *sql.Tx, identifier string) error {
	var count int
	if err := transaction.QueryRowContext(ctx, "SELECT attempt_count FROM authentication_throttles WHERE scope='identifier' AND key_digest=?", domain.Digest(identifier)).Scan(&count); err != nil {
		return err
	}
	delay := time.Second * time.Duration(1<<min(max(count-1, 0), 6))
	_, err := transaction.ExecContext(ctx, "UPDATE authentication_throttles SET next_allowed_at=? WHERE scope='identifier' AND key_digest=?", domain.Timestamp(service.Now().Add(delay)), domain.Digest(identifier))
	return err
}

func (service *Service) releaseSuccessfulAttempt(ctx context.Context, transaction *sql.Tx, identifier string, metadata domain.RequestMetadata) error {
	for scope, key := range map[string]string{"identifier": identifier, "ip": metadata.ObservedIP} {
		if _, err := transaction.ExecContext(ctx, "UPDATE authentication_throttles SET attempt_count=max(0,attempt_count-1) WHERE scope=? AND key_digest=?", scope, domain.Digest(key)); err != nil {
			return err
		}
	}
	return nil
}

func (service *Service) Logout(ctx context.Context, principal domain.Principal, metadata domain.RequestMetadata) error {
	return service.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		return service.closeSession(ctx, transaction, current.SessionID, "logout", metadata, current)
	})
}
