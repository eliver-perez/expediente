package licensing

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Redacted diagnostics for the explicitly authorized live test only.
type providerTraceTransport struct {
	next http.RoundTripper
	test *testing.T
}

func (trace providerTraceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := trace.next.RoundTrip(request)
	if err != nil {
		trace.test.Log("HTTPS transport failed (no request contents logged)")
		return response, err
	}
	contents, readErr := io.ReadAll(io.LimitReader(response.Body, MaximumArtifactBytes+1))
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(contents))
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(contents, &envelope)
	trace.test.Log("provider", request.URL.Path, "HTTP", response.StatusCode, "code", envelope.Error.Code)
	return response, readErr
}

// Explicit opt-in: this test consumes and releases a real commercial activation.
// The key, signed licenses and installation secret remain in private files only.
func TestLiveProvider(t *testing.T) {
	keyPath, stateDirectory := os.Getenv("AIDEX_TEST_LICENSE_FILE"), os.Getenv("AIDEX_TEST_LICENSE_STATE")
	if keyPath == "" || stateDirectory == "" {
		t.Skip("live test requires isolated state and an authorized test license")
	}
	contents, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	productKey := strings.TrimSpace(string(contents))
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	database, err := storage.Open(ctx, stateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if database != nil {
			database.Close()
		}
	}()
	var client *Client
	openClient := func(options Options) {
		t.Helper()
		client, err = New(database, stateDirectory, options)
		if err != nil {
			t.Fatal(err)
		}
		client.httpClient.Transport = providerTraceTransport{next: client.httpClient.Transport, test: t}
	}
	openClient(ProviderOptions())
	metadata := func() domain.RequestMetadata { return domain.RequestMetadata{RequestID: domain.NewID()} }
	online := func(action, id, key string) (Status, error) {
		return client.Online(ctx, action, id, key, nil, metadata())
	}
	artifact := func() string {
		t.Helper()
		var compact string
		if err := database.Reader.QueryRowContext(ctx, "SELECT current_jws FROM license_state WHERE singleton=1").Scan(&compact); err != nil {
			t.Fatal(err)
		}
		return compact
	}
	status := func() Status {
		t.Helper()
		result, err := client.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	assertActive := func(state Status) {
		t.Helper()
		if state.State != "active" || !state.ReadAllowed || !state.WriteAllowed || state.Development || state.Diagnostic != "" {
			t.Fatal("production activation did not authorize this installation")
		}
		for _, feature := range FeatureNames {
			if _, ok := state.Features[feature]; !ok {
				t.Fatal("missing feature policy:", feature)
			}
		}
	}
	// A previously accepted activation is cleaned up even if a subsequent check fails.
	// Failed/unconfirmed activation requests stay in the private journal for recovery.
	needsRelease := status().WriteAllowed
	defer func() {
		if !needsRelease {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cleanupCancel()
		result, err := client.Online(cleanupCtx, "deactivate", domain.NewID(), "", nil, metadata())
		if err != nil || result.State != "revoked" || result.WriteAllowed || result.Diagnostic != "ACTIVATION_DEACTIVATED" {
			t.Error("temporary installation requires confirmed release; error code:", errorCode(err))
		} else {
			t.Log("PASS cleanup: temporary activation deactivated and slot released")
		}
	}()
	requestIDPath := filepath.Join(stateDirectory, "activation-request-id")
	writeRequestID := func(id string) {
		t.Helper()
		if err := os.WriteFile(requestIDPath, []byte(id), 0600); err != nil {
			t.Fatal(err)
		}
	}
	idBytes, err := os.ReadFile(requestIDPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	id := strings.TrimSpace(string(idBytes))
	initial := status()
	if id == "" || initial.State == "revoked" {
		id = domain.NewID()
		writeRequestID(id)
	}
	state := initial
	if !needsRelease {
		state, err = online("activate", id, productKey)
		// Only a definitive rejection retires the old request. A temporary error must
		// retain the same request/identity to avoid an ambiguous duplicate activation.
		if errorCode(err) == "INVALID_PROOF" {
			t.Log("saved challenge definitively rejected; requesting fresh proof with the same installation identity")
			id = domain.NewID()
			writeRequestID(id)
			state, err = online("activate", id, productKey)
		}
		if err != nil {
			t.Fatal("activation failed; error code:", errorCode(err))
		}
		needsRelease = true
	} else {
		t.Log("resuming confirmed activation from private persistent state")
	}
	assertActive(state)
	original := state
	compact := artifact()
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		t.Fatal("noncompact license")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatal("invalid header encoding")
	}
	var header map[string]string
	if json.Unmarshal(headerBytes, &header) != nil || header["kid"] != ProviderKey().Kid || header["alg"] != "EdDSA" {
		t.Fatal("unexpected production KID/algorithm")
	}
	claims, err := Verify(compact, ProviderOptions().TrustedKeys, client.binding)
	if err != nil || claims.ActivationID != state.ActivationID || claims.LicenseID != state.LicenseID {
		t.Fatal("production signature/binding rejected")
	}
	t.Log("PASS online activation and Ed25519 signature; KID:", header["kid"], "type:", state.LicenseType, "revision:", state.Revision)
	if replay, err := online("activate", id, productKey); err != nil || replay.ActivationID != state.ActivationID {
		t.Fatal("activation idempotency failed; error code:", errorCode(err))
	}
	t.Log("PASS activation replay: same installation and activation")
	refreshID := domain.NewID()
	state, err = online("refresh", refreshID, "")
	if err != nil {
		t.Fatal("refresh failed; error code:", errorCode(err))
	}
	assertActive(state)
	if state.ActivationID != original.ActivationID || state.LicenseID != original.LicenseID || state.Revision < original.Revision || state.LastContact == "" {
		t.Fatal("refresh identity/revision/contact inconsistent")
	}
	if replay, err := online("refresh", refreshID, ""); err != nil || replay.Revision != state.Revision {
		t.Fatal("refresh replay failed")
	}
	compact = artifact()
	if claims, err = Verify(compact, ProviderOptions().TrustedKeys, client.binding); err != nil || claims.Revision != state.Revision || !reflect.DeepEqual(claims.Features, state.Features) {
		t.Fatal("refreshed artifact/state mismatch")
	}
	t.Log("PASS online refresh, updated signed artifact, contact and idempotent replay; revision:", state.Revision)
	// Close both SQLite pools and reload identity + license. Unreachable HTTPS URL
	// proves local validation/import do not require a provider connection.
	reopen := func(options Options) {
		t.Helper()
		if err := database.Close(); err != nil {
			t.Fatal(err)
		}
		database = nil
		database, err = storage.Open(ctx, stateDirectory)
		if err != nil {
			t.Fatal(err)
		}
		openClient(options)
	}
	offlineOptions := ProviderOptions()
	offlineOptions.ServerURL = "https://127.0.0.1:1"
	reopen(offlineOptions)
	restored := status()
	assertActive(restored)
	if restored.Binding != original.Binding || restored.ActivationID != original.ActivationID || restored.Revision != state.Revision || !reflect.DeepEqual(restored.Features, state.Features) || artifact() != compact {
		t.Fatal("persistent license/identity changed across restart")
	}
	if _, err := client.Import(ctx, []byte(compact), nil, metadata()); err != nil {
		t.Fatal("offline import failed; error code:", errorCode(err))
	}
	request, err := client.OfflineRequest(ctx, "renew", domain.NewID(), nil, metadata())
	if err != nil {
		t.Fatal("offline renewal request failed")
	}
	if _, payload, err := VerifyOfflineRequest([]byte(request.Contents)); err != nil || payload.ActivationID != state.ActivationID {
		t.Fatal("offline renewal proof invalid")
	}
	t.Log("PASS closed/reopened database, private installation identity, persisted state/features, offline signature/import and renewal proof")
	// Mutate only local copies. Each rejected import must leave the trusted license
	// and revision floor intact. Never print either the product key or JWS.
	rejectImport := func(name, altered, expectedCode string) {
		t.Helper()
		before, trusted := status(), artifact()
		if _, err := client.Import(ctx, []byte(altered), nil, metadata()); errorCode(err) != expectedCode {
			t.Fatal(name, "unexpected rejection:", errorCode(err))
		}
		after := status()
		if artifact() != trusted || before.State != after.State || before.Revision != after.Revision || before.ActivationID != after.ActivationID || !reflect.DeepEqual(before.Features, after.Features) {
			t.Fatal(name, "rejection changed the trusted state")
		}
		t.Log("PASS rejection:", name, expectedCode, "trusted state preserved")
	}
	refreshedParts := strings.Split(compact, ".")
	mutated := append([]string(nil), refreshedParts...)
	signature, _ := base64.RawURLEncoding.DecodeString(mutated[2])
	signature[0] ^= 1
	mutated[2] = base64.RawURLEncoding.EncodeToString(signature)
	rejectImport("tampered signature", strings.Join(mutated, "."), "LICENSE_INVALID_SIGNATURE")
	mutated = append([]string(nil), refreshedParts...)
	payload, _ := base64.RawURLEncoding.DecodeString(mutated[1])
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatal("invalid signed payload")
	}
	object["license_revision"] = float64(state.Revision + 1)
	payload, err = json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	mutated[1] = base64.RawURLEncoding.EncodeToString(payload)
	rejectImport("tampered claims", strings.Join(mutated, "."), "LICENSE_INVALID_SIGNATURE")
	mutated = append([]string(nil), refreshedParts...)
	header["kid"] = "unknown-production-key"
	headerBytes, _ = json.Marshal(header)
	mutated[0] = base64.RawURLEncoding.EncodeToString(headerBytes)
	rejectImport("unknown KID", strings.Join(mutated, "."), "LICENSE_UNKNOWN_KEY")
	rejectImport("malformed license", "invalid-license", "LICENSE_INVALID")
	wrongBinding := client.binding
	wrongBinding.InstallationID = domain.NewID()
	if _, err := Verify(compact, ProviderOptions().TrustedKeys, wrongBinding); errorCode(err) != "LICENSE_BINDING_MISMATCH" {
		t.Fatal("foreign installation license accepted")
	}
	t.Log("PASS foreign installation binding rejected")
	openClient(ProviderOptions())
	deactivateID := domain.NewID()
	state, err = online("deactivate", deactivateID, "")
	if err != nil {
		t.Fatal("deactivation failed; error code:", errorCode(err))
	}
	needsRelease = false
	if state.State != "revoked" || state.WriteAllowed || !state.ReadAllowed || state.Diagnostic != "ACTIVATION_DEACTIVATED" {
		t.Fatal("deactivation state inconsistent")
	}
	if replay, err := online("deactivate", deactivateID, ""); err != nil || replay.WriteAllowed {
		t.Fatal("deactivation replay failed")
	}
	rejectImport("deactivated signed license", compact, "ACTIVATION_DEACTIVATED")
	reopen(ProviderOptions())
	if state = status(); state.State != "revoked" || state.WriteAllowed {
		t.Fatal("deactivation not persistent")
	}
	t.Log("PASS confirmed deactivation, read-only state, idempotent replay, anti-reactivation floor and restart")
	// Invalid key test is made only while this installation is deactivated.
	if _, err := online("activate", domain.NewID(), "AIDEX-INVALID-QA-"+domain.NewID()); errorCode(err) != "LICENSE_NOT_FOUND" {
		t.Fatal("invalid product key unexpected result:", errorCode(err))
	}
	if status().WriteAllowed {
		t.Fatal("invalid product key enabled installation")
	}
	t.Log("PASS provider rejected invalid product key with LICENSE_NOT_FOUND")
	id = domain.NewID()
	writeRequestID(id)
	state, err = online("activate", id, productKey)
	if err != nil {
		t.Fatal("reactivation failed; error code:", errorCode(err))
	}
	needsRelease = true
	assertActive(state)
	if state.Binding != original.Binding || state.LicenseID != original.LicenseID || state.ActivationID == original.ActivationID || state.Revision <= original.Revision {
		t.Fatal("reactivation identity/revision inconsistent")
	}
	if _, err := Verify(artifact(), ProviderOptions().TrustedKeys, client.binding); err != nil {
		t.Fatal("new activation signature invalid")
	}
	t.Log("PASS new activation after release: same installation/license, new activation, production signature; revision:", state.Revision)
	// A valid signature from the retired activation cannot overwrite the new one.
	rejectImport("previous activation artifact", compact, "LICENSE_ACTIVATION_MISMATCH")
	state, err = online("refresh", domain.NewID(), "")
	if err != nil {
		t.Fatal("new activation refresh failed; error code:", errorCode(err))
	}
	assertActive(state)
	// Confirm release explicitly, and re-open to verify persisted final read-only state.
	state, err = online("deactivate", domain.NewID(), "")
	if err != nil {
		t.Fatal("final release failed; error code:", errorCode(err))
	}
	needsRelease = false
	reopen(ProviderOptions())
	final := status()
	if final.State != "revoked" || final.WriteAllowed || !final.ReadAllowed || final.Diagnostic != "ACTIVATION_DEACTIVATED" {
		t.Fatal("final deactivation not persistent")
	}
	// Journal/artifacts must not retain the commercial credential.
	var leaked int
	if err := database.Reader.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM license_operations WHERE instr(request_json,?)>0) + (SELECT count(*) FROM license_artifacts WHERE instr(contents,?)>0)", productKey, productKey).Scan(&leaked); err != nil || leaked != 0 {
		t.Fatal("credential persistence check failed")
	}
	t.Log("PASS final release confirmed and persisted; product key absent from journal/artifacts; temporary installation left deactivated")
}
