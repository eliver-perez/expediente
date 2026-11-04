package licensing

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

type scheduledCached struct {
	request  onlineRequest
	response map[string]string
}
type scheduledAuthority struct {
	responses             map[string]scheduledCached
	mutex                 sync.Mutex
	now                   *atomic.Int64
	claims                Claims
	key                   ed25519.PrivateKey
	public                ed25519.PublicKey
	challenges            map[string]string
	unavailable, tampered bool
	refreshes             int
}

func automaticFixture(t *testing.T) (*Client, *scheduledAuthority, *atomic.Int64, time.Time) {
	t.Helper()
	client, key := testClient(t)
	initial := client.Now()
	clock := &atomic.Int64{}
	clock.Store(initial.UnixNano())
	client.Now = func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	claims := testClaims(client)
	importClaims(t, client, key, claims)
	if err := client.Database.Write(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE license_state SET last_contact_at=?", domain.Timestamp(initial))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	public, err := base64.RawURLEncoding.DecodeString(client.binding.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	authority := &scheduledAuthority{now: clock, claims: claims, key: key, public: public, challenges: map[string]string{}, responses: map[string]scheduledCached{}}
	server := httptest.NewTLSServer(http.HandlerFunc(authority.handle))
	t.Cleanup(server.Close)
	client.Options.ServerURL = server.URL
	client.Options.RefreshHours = 24
	client.httpClient = server.Client()
	client.httpClient.Timeout = 5 * time.Second
	return client, authority, clock, initial
}
func (authority *scheduledAuthority) handle(w http.ResponseWriter, r *http.Request) {
	authority.mutex.Lock()
	defer authority.mutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if authority.unavailable {
		w.WriteHeader(503)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "TEMPORARY_UNAVAILABLE"}})
		return
	}
	now := time.Unix(0, authority.now.Load()).UTC()
	if r.URL.Path == "/v1/activations/challenge" {
		id, nonce := domain.NewID(), domain.NewID()
		authority.challenges[id] = nonce
		_ = json.NewEncoder(w).Encode(map[string]string{"challenge_id": id, "nonce": nonce, "expires_at": now.Add(3 * time.Minute).Format(time.RFC3339)})
		return
	}
	var request onlineRequest
	if r.URL.Path != "/v1/activations/refresh" || json.NewDecoder(r.Body).Decode(&request) != nil {
		w.WriteHeader(400)
		return
	}
	if previous, ok := authority.responses[request.RequestID]; ok {
		if previous.request != request {
			w.WriteHeader(400)
			return
		}
		authority.reply(w, previous.response)
		return
	}
	nonce, exists := authority.challenges[request.ChallengeID]
	signature, err := base64.RawURLEncoding.DecodeString(request.Proof)
	if !exists || err != nil || request.InstallationID != authority.claims.InstallationID || request.ActivationID != authority.claims.ActivationID || !ed25519.Verify(authority.public, ProofMessage("refresh", request.ChallengeID, nonce, request.InstallationID, request.ActivationID), signature) {
		w.WriteHeader(401)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "INVALID_PROOF"}})
		return
	}
	delete(authority.challenges, request.ChallengeID)
	authority.refreshes++
	claims := authority.claims
	claims.Revision++
	claims.IssuedAt = now.Format(time.RFC3339)
	response := map[string]string{"license_jws": signed(authority.key, claims), "server_time": now.Format(time.RFC3339), "request_id": request.RequestID}
	authority.responses[request.RequestID] = scheduledCached{request: request, response: response}
	authority.reply(w, response)
}
func (authority *scheduledAuthority) reply(w http.ResponseWriter, response map[string]string) {
	copy := map[string]string{}
	for key, value := range response {
		copy[key] = value
	}
	if authority.tampered {
		jws := copy["license_jws"]
		copy["license_jws"] = jws[:len(jws)-6] + "AAAAAA"
	}
	_ = json.NewEncoder(w).Encode(copy)
}

func licenseLoop(t *testing.T, client *Client) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); client.Run(ctx) }()
	stop := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(6 * time.Second):
			t.Error("el verificador no terminó dentro del límite")
		}
	}
	t.Cleanup(stop)
	return stop
}
func waitLicenseCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("la validación automática no alcanzó el estado esperado dentro del límite")
}

func TestAutomaticValidationKeeps24HoursAndDoesNotPollAtTwoHours(t *testing.T) {
	client, authority, clock, initial := automaticFixture(t)
	clock.Store(initial.Add(2 * time.Hour).UnixNano())
	stop := licenseLoop(t, client)
	time.Sleep(150 * time.Millisecond)
	stop()
	authority.mutex.Lock()
	defer authority.mutex.Unlock()
	if authority.refreshes != 0 {
		t.Fatal("se acortó el intervalo aprobado")
	}
	status, err := client.Status(context.Background())
	if err != nil || !status.WriteAllowed || status.NextValidation != initial.Add(24*time.Hour).UTC().Format(time.RFC3339Nano) {
		t.Fatal(status.NextValidation, err)
	}
}

func TestAutomaticValidationAppliesSignedRevocationWithoutGraceAndPersists(t *testing.T) {
	client, authority, clock, initial := automaticFixture(t)
	authority.claims.Status = "revoked"
	clock.Store(initial.Add(24 * time.Hour).UnixNano())
	stop := licenseLoop(t, client)
	waitLicenseCondition(t, 4*time.Second, func() bool {
		state, err := client.Status(context.Background())
		return err == nil && state.State == "revoked"
	})
	stop()
	if client.Check(context.Background(), WriteDocuments) == nil || client.CheckFeatures(context.Background(), "ocr") == nil {
		t.Fatal("revocación firmada no suspendió permisos")
	}
	if client.Check(context.Background(), ReadDocuments) != nil || client.Check(context.Background(), BackupExport) != nil {
		t.Fatal("se bloqueó el acceso conservado por el contrato")
	}
	restarted, err := New(client.Database, client.stateDirectory, client.Options)
	if err != nil {
		t.Fatal(err)
	}
	restarted.Now = client.Now
	state, err := restarted.Status(context.Background())
	if err != nil || state.State != "revoked" || state.WriteAllowed {
		t.Fatal("no persistió el retiro", err)
	}
}

func TestAutomaticValidationOutageRecoveryAndInvalidSignature(t *testing.T) {
	for _, variant := range []string{"outage", "signature"} {
		t.Run(variant, func(t *testing.T) {
			client, authority, clock, initial := automaticFixture(t)
			authority.claims.Status = "revoked"
			authority.unavailable = variant == "outage"
			authority.tampered = variant == "signature"
			clock.Store(initial.Add(24 * time.Hour).UnixNano())
			stop := licenseLoop(t, client)
			waitLicenseCondition(t, 4*time.Second, func() bool {
				client.mutex.Lock()
				defer client.mutex.Unlock()
				return client.nextAttempt.After(client.Now())
			})
			stop()
			state, err := client.Status(context.Background())
			if err != nil || !state.WriteAllowed || state.Revision != 1 || state.LastContact != domain.Timestamp(initial) {
				t.Fatal("fallo temporal o firma inválida sustituyó estado auténtico", err)
			}
			client.mutex.Lock()
			retry := client.nextAttempt
			client.mutex.Unlock()
			if !retry.Equal(client.Now().Add(15 * time.Minute)) {
				t.Fatal("cambió el backoff aprobado")
			}
			// Reabrir el cliente conserva la licencia durante la desconexión; al volver
			// la conexión se verifica y aplica una revocación, sin una tolerancia adicional.
			restarted, err := New(client.Database, client.stateDirectory, client.Options)
			if err != nil {
				t.Fatal(err)
			}
			restarted.Now = client.Now
			restarted.httpClient = client.httpClient
			state, err = restarted.Status(context.Background())
			if err != nil || !state.WriteAllowed {
				t.Fatal("perpetua bloqueada tras reinicio", err)
			}
			authority.mutex.Lock()
			authority.unavailable = false
			authority.tampered = false
			authority.claims.Status = "revoked"
			authority.mutex.Unlock()
			stop = licenseLoop(t, restarted)
			waitLicenseCondition(t, 4*time.Second, func() bool {
				s, err := restarted.Status(context.Background())
				return err == nil && s.State == "revoked"
			})
			stop()
		})
	}
}

func TestAutomaticValidationContinuesWhileServiceRemainsRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("espera el tick real de un minuto del servicio")
	}
	client, authority, clock, initial := automaticFixture(t)
	clock.Store(initial.Add(2 * time.Hour).UnixNano())
	stop := licenseLoop(t, client)
	defer stop()
	time.Sleep(150 * time.Millisecond)
	authority.mutex.Lock()
	authority.claims.Status = "revoked"
	authority.mutex.Unlock()
	clock.Store(initial.Add(24 * time.Hour).UnixNano())
	t.Log("esperando el tick real de un minuto; no se reinicia Run ni se invoca Online desde la prueba")
	waitLicenseCondition(t, 70*time.Second, func() bool {
		state, err := client.Status(context.Background())
		return err == nil && state.State == "revoked"
	})
}
