package licensing

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

type Authorization func(context.Context, *sql.Tx) (domain.Principal, error)
type Client struct {
	Database       *storage.Database
	Options        Options
	Now            func() time.Time
	stateDirectory string
	binding        Binding
	privateKey     ed25519.PrivateKey
	identityError  string
	httpClient     *http.Client
	operationMutex sync.Mutex
	mutex          sync.Mutex
	lastEffective  time.Time
	lastSample     time.Time
	cachedJWS      string
	cachedClaims   Claims
	cachedBinding  Binding
}
type localState struct {
	JWS                 string
	Deactivated         bool
	DeactivationPending bool
	Trusted             string
	Observed            string
	LastContact         string
	LastError           string
	ClockWarning        bool
}

func readLocal(ctx context.Context, query storage.Querier) (localState, error) {
	var state localState
	err := query.QueryRowContext(ctx, "SELECT current_jws,deactivated,offline_deactivation_pending,last_trusted_at,last_observed_at,last_contact_at,last_error_code,clock_warning FROM license_state WHERE singleton=1").Scan(&state.JWS, &state.Deactivated, &state.DeactivationPending, &state.Trusted, &state.Observed, &state.LastContact, &state.LastError, &state.ClockWarning)
	return state, err
}
func New(database *storage.Database, stateDirectory string, options Options) (*Client, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}
	client := &Client{Database: database, Options: options, Now: time.Now, stateDirectory: stateDirectory}
	fingerprint, err := MachineFingerprint()
	if err == nil {
		client.binding, client.privateKey, err = loadInstallation(context.Background(), database, stateDirectory, fingerprint)
	}
	if err != nil {
		client.identityError = errorCode(err)
		if client.identityError == "LICENSE_INTERNAL" {
			client.identityError = "LICENSE_IDENTITY_UNAVAILABLE"
		}
	}
	transport, err := licenseTransport(options)
	if err != nil {
		return nil, err
	}
	client.httpClient = &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("license redirects forbidden") }}
	return client, nil
}
func errorCode(err error) string {
	var typed *domain.Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return "LICENSE_INTERNAL"
}
func (client *Client) effectiveTime(state localState) (time.Time, bool) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	sample := client.Now()
	current := sample.UTC()
	effective := current
	for _, value := range []string{state.Trusted, state.Observed} {
		if instant, err := utcInstant(value); err == nil && instant.After(effective) {
			effective = instant
		}
	}
	if !client.lastSample.IsZero() {
		elapsed := sample.Sub(client.lastSample)
		if elapsed < 0 {
			elapsed = 0
		}
		advanced := client.lastEffective.Add(elapsed)
		if advanced.After(effective) {
			effective = advanced
		}
	}
	client.lastEffective = effective
	client.lastSample = sample
	return effective, current.Add(5 * time.Minute).Before(effective)
}
func (client *Client) verify(compact string) (Claims, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if client.cachedJWS == compact && client.cachedBinding == client.binding {
		return client.cachedClaims, nil
	}
	claims, err := Verify(compact, client.Options.TrustedKeys, client.binding)
	if err != nil {
		return claims, err
	}
	client.cachedJWS = compact
	client.cachedClaims = claims
	client.cachedBinding = client.binding
	return claims, nil
}

type Status struct {
	State        string `json:"state"`
	LicenseType  string `json:"license_type"`
	LicenseID    string `json:"license_id"`
	ActivationID string `json:"activation_id"`
	Revision     int64  `json:"license_revision"`
	Binding
	Features             map[string]bool `json:"features"`
	ExpiresAt            *string         `json:"expires_at"`
	GraceUntil           *string         `json:"grace_until"`
	MaintenanceUntil     *string         `json:"maintenance_until"`
	EntitledReleaseUntil string          `json:"entitled_release_until"`
	ReadAllowed          bool            `json:"read_allowed"`
	WriteAllowed         bool            `json:"write_allowed"`
	Development          bool            `json:"development"`
	OnlineConfigured     bool            `json:"online_configured"`
	ClockWarning         bool            `json:"clock_warning"`
	LastContact          string          `json:"last_contact_at"`
	LastError            string          `json:"last_error_code"`
	Diagnostic           string          `json:"diagnostic"`
	DeactivationPending  bool            `json:"offline_deactivation_pending"`
}

func (client *Client) Status(ctx context.Context) (Status, error) {
	result := Status{State: "unactivated", Binding: client.binding, Features: map[string]bool{}, OnlineConfigured: client.Options.ServerURL != "", Development: client.Options.DevelopmentBypass && developmentEnabled}
	snapshot, err := client.Database.Reader.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer snapshot.Rollback()
	state, err := readLocal(ctx, snapshot)
	if err != nil {
		return result, err
	}
	result.LastContact = state.LastContact
	result.LastError = state.LastError
	result.DeactivationPending = state.DeactivationPending
	effective, warning := client.effectiveTime(state)
	result.ClockWarning = warning
	if result.Development && state.JWS == "" {
		result.State = "development"
		result.ReadAllowed = true
		result.WriteAllowed = true
		for _, name := range FeatureNames {
			result.Features[name] = true
		}
		return result, nil
	}
	result.Development = false
	if client.identityError != "" {
		result.State = "invalid"
		result.Diagnostic = client.identityError
		return result, nil
	}
	if state.JWS == "" {
		return result, nil
	}
	claims, err := client.verify(state.JWS)
	if err != nil {
		result.State = "invalid"
		result.Diagnostic = errorCode(err)
		return result, nil
	}
	var revision int64
	var digest string
	var retired bool
	if err = snapshot.QueryRowContext(ctx, "SELECT highest_revision,jws_sha256,deactivated FROM license_revision_floors WHERE activation_id=? AND license_id=?", claims.ActivationID, claims.LicenseID).Scan(&revision, &digest, &retired); err != nil || revision != claims.Revision || digest != domain.Digest(state.JWS) {
		result.State = "invalid"
		result.Diagnostic = "LICENSE_REVISION_CONFLICT"
		return result, nil
	}
	result.LicenseID = claims.LicenseID
	result.ActivationID = claims.ActivationID
	result.Revision = claims.Revision
	result.LicenseType = claims.Type
	// Never expose the cached mutable map to callers.
	for name, value := range claims.Features {
		result.Features[name] = value
	}
	result.ExpiresAt = claims.ExpiresAt
	result.MaintenanceUntil = claims.MaintenanceUntil
	result.EntitledReleaseUntil = claims.EntitledReleaseUntil
	result.State = "active"
	result.ReadAllowed = true
	result.WriteAllowed = true
	if claims.Type == "subscription" {
		expires, _ := utcInstant(*claims.ExpiresAt)
		grace := expires.Add(15 * 24 * time.Hour)
		formatted := grace.Format(time.RFC3339Nano)
		result.GraceUntil = &formatted
		if !effective.Before(grace) {
			result.State = "expired"
			result.WriteAllowed = false
		} else if !effective.Before(expires) {
			result.State = "grace"
		}
	}
	if claims.Status == "revoked" || state.Deactivated || retired {
		result.State = "revoked"
		result.WriteAllowed = false
		if state.Deactivated || retired {
			result.Diagnostic = "ACTIVATION_DEACTIVATED"
		}
	}
	return result, nil
}
func authorize(ctx context.Context, transaction *sql.Tx, authorization Authorization) (domain.Principal, error) {
	if authorization == nil {
		return domain.Principal{}, nil
	}
	return authorization(ctx, transaction)
}
func licenseEvent(ctx context.Context, transaction *sql.Tx, principal domain.Principal, kind string, details map[string]any, metadata domain.RequestMetadata) error {
	return audit.Append(ctx, transaction, time.Now(), audit.Event{Type: kind, ActorUserID: principal.User.ID, SessionID: principal.SessionID, SystemActor: principal.User.ID == "", Metadata: metadata, Details: details})
}
func (client *Client) priorClaims(compact string) (Claims, error) {
	parts := splitPayload(compact)
	var claims Claims
	if len(parts) == 0 || json.Unmarshal(parts, &claims) != nil {
		return claims, contractFailure()
	}
	binding := client.binding
	binding.FingerprintHash = claims.FingerprintHash
	return Verify(compact, client.Options.TrustedKeys, binding)
}
func splitPayload(compact string) []byte {
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return nil
	}
	contents, err := decodeBase64(parts[1], 0)
	if err != nil {
		return nil
	}
	return contents
}
