package identity

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

const fixturePassword = "Only-a-test-password-2026"

var fixtureMetadata = domain.RequestMetadata{RequestID: "test-request", ObservedIP: "192.0.2.10", UserAgent: "Test browser"}

func fixtureService(t *testing.T) *Service {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	database, err := storage.Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	hasher, err := newPasswordHasher(8192, 1)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Database: database, Config: config.Defaults(directory), Hasher: hasher, Now: time.Now}
	if err := service.Bootstrap(context.Background(), "admin", "Administrador de prueba", fixturePassword); err != nil {
		t.Fatal(err)
	}
	return service
}

func loginFixture(t *testing.T, service *Service) (LoginResult, domain.Principal) {
	t.Helper()
	result, err := service.Login(context.Background(), "admin", fixturePassword, fixtureMetadata)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := service.Authenticate(context.Background(), result.SessionToken, false, fixtureMetadata)
	if err != nil {
		t.Fatal(err)
	}
	return result, principal
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var failure *domain.Error
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func TestConcurrentLoginsReplaceAtomically(t *testing.T) {
	service := fixtureService(t)
	ctx := context.Background()
	var wait sync.WaitGroup
	wait.Add(2)
	results := make(chan LoginResult, 2)
	failures := make(chan error, 2)
	for range 2 {
		go func() {
			defer wait.Done()
			result, err := service.Login(ctx, "admin", fixturePassword, fixtureMetadata)
			if err != nil {
				failures <- err
			} else {
				results <- result
			}
		}()
	}
	wait.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	active, replaced := 0, 0
	for result := range results {
		_, err := service.Authenticate(ctx, result.SessionToken, false, fixtureMetadata)
		if err == nil {
			active++
		} else {
			requireCode(t, err, "SESSION_REPLACED")
			replaced++
		}
	}
	if active != 1 || replaced != 1 {
		t.Fatalf("active=%d replaced=%d", active, replaced)
	}
	var count int
	if err := service.Database.Reader.QueryRow("SELECT count(*) FROM sessions WHERE closed_at IS NULL").Scan(&count); err != nil || count != 1 {
		t.Fatalf("open sessions=%d error=%v", count, err)
	}
}

func TestLoginRollbackPreservesPreviousSession(t *testing.T) {
	service := fixtureService(t)
	previous, _ := loginFixture(t, service)
	_, err := service.Database.Writer.Exec(`CREATE TRIGGER simulate_audit_failure BEFORE INSERT ON audit_events WHEN NEW.event_type='session.started' BEGIN SELECT RAISE(ABORT,'test failure'); END;`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(context.Background(), "admin", fixturePassword, fixtureMetadata); err == nil {
		t.Fatal("expected audit failure")
	}
	if _, err := service.Authenticate(context.Background(), previous.SessionToken, false, fixtureMetadata); err != nil {
		t.Fatalf("previous session lost: %v", err)
	}
}

func TestPasswordChangeAndResetInvalidateSessions(t *testing.T) {
	service := fixtureService(t)
	result, principal := loginFixture(t, service)
	ctx := context.Background()
	if err := service.ChangePassword(ctx, principal, fixturePassword, "A-new-test-password-2026", fixtureMetadata); err != nil {
		t.Fatal(err)
	}
	_, err := service.Authenticate(ctx, result.SessionToken, false, fixtureMetadata)
	requireCode(t, err, "SESSION_REVOKED")
	updated, err := service.Login(ctx, "admin", "A-new-test-password-2026", fixtureMetadata)
	if err != nil {
		t.Fatal(err)
	}
	principal, err = service.Authenticate(ctx, updated.SessionToken, false, fixtureMetadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(ctx, principal, principal.User.ID, fixturePassword, fixtureMetadata); err != nil {
		t.Fatal(err)
	}
	_, err = service.Authenticate(ctx, updated.SessionToken, false, fixtureMetadata)
	requireCode(t, err, "SESSION_REVOKED")
}

func TestConcurrentResetNeverLeavesOldCredentialSessionActive(t *testing.T) {
	service := fixtureService(t)
	_, admin := loginFixture(t, service)
	ctx := context.Background()
	user, err := service.CreateUser(ctx, admin, "other", "Otra persona", fixturePassword, fixtureMetadata)
	if err != nil {
		t.Fatal(err)
	}
	var result LoginResult
	var loginError, resetError error
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		result, loginError = service.Login(ctx, "other", fixturePassword, fixtureMetadata)
	}()
	go func() {
		defer wait.Done()
		resetError = service.ResetPassword(ctx, admin, user.ID, "Reset-test-password-2026", fixtureMetadata)
	}()
	wait.Wait()
	if resetError != nil {
		t.Fatal(resetError)
	}
	if loginError == nil {
		_, err = service.Authenticate(ctx, result.SessionToken, false, fixtureMetadata)
		requireCode(t, err, "SESSION_REVOKED")
	} else {
		requireCode(t, loginError, "AUTHENTICATION_FAILED")
	}
}

func TestFailedLoginPersistsWithoutSessionOrSecretsAndRateLimitSurvivesRestart(t *testing.T) {
	service := fixtureService(t)
	ctx := context.Background()
	now := time.Now()
	service.Now = func() time.Time { return now }
	for attempt := 0; attempt < 5; attempt++ {
		_, err := service.Login(ctx, "missing-user", "Sensitive-wrong-password", fixtureMetadata)
		requireCode(t, err, "AUTHENTICATION_FAILED")
		now = now.Add(time.Minute)
	}
	// A new service shares persistent counters; no in-memory limit can hide the restart.
	restarted := &Service{Database: service.Database, Hasher: service.Hasher, Config: service.Config, Now: service.Now}
	_, err := restarted.Login(ctx, "missing-user", "Sensitive-wrong-password", fixtureMetadata)
	requireCode(t, err, "RATE_LIMITED")
	var count int
	if err := service.Database.Reader.QueryRow("SELECT count(*) FROM authentication_attempts WHERE user_id IS NULL").Scan(&count); err != nil || count != 6 {
		t.Fatalf("attempts=%d %v", count, err)
	}
	rows, err := service.Database.Reader.Query("SELECT details_json FROM audit_events")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var details string
		if err := rows.Scan(&details); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(details, "Sensitive-wrong-password") || strings.Contains(details, "$argon2id$") {
			t.Fatal("secret in audit")
		}
	}
}

func TestPollingDoesNotExtendIdleSessionAndExpiryIsRecorded(t *testing.T) {
	service := fixtureService(t)
	now := time.Now()
	service.Now = func() time.Time { return now }
	result, _ := loginFixture(t, service)
	now = now.Add(29 * time.Minute)
	if _, err := service.Authenticate(context.Background(), result.SessionToken, false, fixtureMetadata); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	_, err := service.Authenticate(context.Background(), result.SessionToken, false, fixtureMetadata)
	requireCode(t, err, "SESSION_EXPIRED")
	var reason string
	if err := service.Database.Reader.QueryRow("SELECT close_reason FROM sessions WHERE token_digest=?", domain.Digest(result.SessionToken)).Scan(&reason); err != nil || reason != "expired" {
		t.Fatalf("expiry not recorded: %s %v", reason, err)
	}
}

func TestBootstrapLastAdministratorAndRevisionGuards(t *testing.T) {
	service := fixtureService(t)
	_, principal := loginFixture(t, service)
	ctx := context.Background()
	requireCode(t, service.Bootstrap(ctx, "again", "Otro", fixturePassword), "BOOTSTRAP_COMPLETE")
	disabled := true
	_, err := service.UpdateUser(ctx, principal, principal.User.ID, principal.User.Revision, nil, &disabled, fixtureMetadata)
	requireCode(t, err, "LAST_ADMIN_REQUIRED")
	_, err = service.SetGlobalRoles(ctx, principal, principal.User.ID, principal.User.Revision, []string{}, fixtureMetadata)
	requireCode(t, err, "LAST_ADMIN_REQUIRED")
	_, err = service.UpdateUser(ctx, principal, principal.User.ID, 999, nil, &disabled, fixtureMetadata)
	requireCode(t, err, "VERSION_CONFLICT")
}

func TestEveryWriteRechecksPermissionsAndSession(t *testing.T) {
	service := fixtureService(t)
	_, principal := loginFixture(t, service)
	ctx := context.Background()
	if err := service.Database.Write(ctx, func(transaction *sql.Tx) error {
		_, err := transaction.Exec("DELETE FROM global_role_assignments WHERE user_id=?", principal.User.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	_, err := service.CreateUser(ctx, principal, "forbidden", "No permitido", fixturePassword, fixtureMetadata)
	requireCode(t, err, "FORBIDDEN")
}

func TestProductionPasswordParametersAndValidation(t *testing.T) {
	hasher, err := NewPasswordHasher()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := hasher.Hash(context.Background(), fixturePassword)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=1$") {
		t.Fatal("unexpected parameters")
	}
	valid, err := hasher.Verify(context.Background(), fixturePassword, encoded)
	if err != nil || !valid {
		t.Fatalf("password verification: %v", err)
	}
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
	for _, password := range []string{"abcdef", "áéíóúñ"} {
		if err := ValidatePassword(password); err != nil {
			t.Fatalf("six-character password rejected: %v", err)
		}
	}
	if _, err := hasher.Verify(context.Background(), fixturePassword, strings.Replace(encoded, "m=65536", "m=4294967295", 1)); err == nil {
		t.Fatal("unbounded memory accepted")
	}
}
