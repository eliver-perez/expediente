package licensing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

func testClient(t *testing.T) (*Client, ed25519.PrivateKey) {
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
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(database, directory, Options{TrustedKeys: []TrustKey{{Kid: "license-test-random", PublicKey: base64.RawURLEncoding.EncodeToString(public), Environment: "production", Purpose: "license"}}})
	if err != nil {
		t.Fatal(err)
	}
	if client.identityError != "" {
		t.Fatal(client.identityError)
	}
	instant, _ := time.Parse(time.RFC3339, "2026-09-22T18:30:00Z")
	client.Now = func() time.Time { return instant }
	return client, private
}
func testClaims(client *Client) Claims {
	one := int64(1)
	features := map[string]bool{}
	for _, name := range FeatureNames {
		features[name] = true
	}
	return Claims{SchemaVersion: "1.0", ProductID: ProductID, LicenseID: domain.NewID(), ActivationID: domain.NewID(), Binding: client.binding, Type: "perpetual", Status: "active", Revision: 1, IssuedAt: client.Now().Format(time.RFC3339), EntitledReleaseUntil: client.Now().Format(time.RFC3339), Features: features, Limits: map[string]*int64{"max_installations": &one, "max_users": nil, "max_libraries": nil, "max_documents": nil}}
}
func signRaw(private ed25519.PrivateKey, header, payload []byte) string {
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte(input)))
}
func signed(private ed25519.PrivateKey, claims Claims) string {
	payload, _ := json.Marshal(claims)
	return signRaw(private, []byte(`{"alg":"EdDSA","kid":"license-test-random","typ":"lic+jws"}`), payload)
}
func expectCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || errorCode(err) != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}
func importClaims(t *testing.T, client *Client, private ed25519.PrivateKey, claims Claims) Status {
	t.Helper()
	state, err := client.Import(context.Background(), []byte(signed(private, claims)), nil, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	return state
}
func TestJWSStrictContractSignatureBindingAndEnvironment(t *testing.T) {
	client, private := testClient(t)
	claims := testClaims(client)
	payload, _ := json.Marshal(claims)
	if _, err := Verify(signed(private, claims), client.Options.TrustedKeys, client.binding); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*Claims)
	}{
		{"product", func(c *Claims) { c.ProductID = "wrong" }}, {"installation", func(c *Claims) { c.InstallationID = domain.NewID() }}, {"public-key", func(c *Claims) { c.PublicKey = base64.RawURLEncoding.EncodeToString(make([]byte, 32)) }}, {"fingerprint", func(c *Claims) { c.FingerprintHash = "sha256:" + strings.Repeat("b", 64) }}, {"schema", func(c *Claims) { c.SchemaVersion = "2.0" }}, {"revision-zero", func(c *Claims) { c.Revision = 0 }}, {"features-dependent", func(c *Claims) { c.Features["expedientes"] = false }}, {"missing-feature", func(c *Claims) { delete(c.Features, "ocr") }}, {"perpetual-expiry", func(c *Claims) { value := c.IssuedAt; c.ExpiresAt = &value }}, {"subscription-grace", func(c *Claims) { c.Type = "subscription"; value := c.IssuedAt; c.ExpiresAt = &value; c.GraceDays = 14 }}, {"non-UTC", func(c *Claims) { c.IssuedAt = "2026-09-22T12:30:00-06:00" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := testClaims(client)
			tc.change(&value)
			if _, err := Verify(signed(private, value), client.Options.TrustedKeys, client.binding); err == nil {
				t.Fatal("invalid contract accepted")
			}
		})
	}
	for _, header := range []string{`{"alg":"none","kid":"license-test-random","typ":"lic+jws"}`, `{"alg":"EdDSA","kid":"unknown","typ":"lic+jws"}`, `{"alg":"EdDSA","kid":"license-test-random","typ":"JWT"}`, `{"alg":"EdDSA","ALG":"EdDSA","kid":"license-test-random","typ":"lic+jws"}`, `{"alg":"EdDSA","kid":"license-test-random","typ":"lic+jws","crit":["b64"]}`} {
		if _, err := Verify(signRaw(private, []byte(header), payload), client.Options.TrustedKeys, client.binding); err == nil {
			t.Fatal("invalid header accepted")
		}
	}
	validHeader := []byte(`{"alg":"EdDSA","kid":"license-test-random","typ":"lic+jws"}`)
	for _, changed := range []string{strings.Replace(string(payload), `"schema_version":"1.0"`, `"schema_version":"1.0","SCHEMA_VERSION":"1.0"`, 1), strings.Replace(string(payload), `"ocr":true`, `"ocr":null`, 1), string(payload) + `{}`, strings.Replace(string(payload), `"grace_days":0`, `"grace_days":null`, 1)} {
		if _, err := Verify(signRaw(private, validHeader, []byte(changed)), client.Options.TrustedKeys, client.binding); err == nil {
			t.Fatal("invalid JSON semantics accepted", changed)
		}
	}
	compact := signed(private, claims)
	parts := strings.Split(compact, ".")
	signature, _ := decodeBase64(parts[2], 64)
	signature[4] ^= 1
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	_, err := Verify(strings.Join(parts, "."), client.Options.TrustedKeys, client.binding)
	expectCode(t, err, "LICENSE_INVALID_SIGNATURE")
	public, _ := hex.DecodeString(developmentPublicKeyHex)
	key := TrustKey{Kid: "renamed-production-key", PublicKey: base64.RawURLEncoding.EncodeToString(public), Environment: "production", Purpose: "license"}
	if err := (Options{TrustedKeys: []TrustKey{key}}).Validate(); (err == nil) != developmentEnabled {
		t.Fatal("public development key accepted under production alias")
	}
	key = client.Options.TrustedKeys[0]
	key.Purpose = "updates"
	if err := (Options{TrustedKeys: []TrustKey{key}}).Validate(); err == nil {
		t.Fatal("update key accepted for license")
	}
}
func TestSubscriptionExclusiveBoundariesClockAndPerpetualOffline(t *testing.T) {
	client, private := testClient(t)
	claims := testClaims(client)
	expiry := client.Now().Add(time.Hour)
	text := expiry.Format(time.RFC3339)
	claims.Type = "subscription"
	claims.ExpiresAt = &text
	claims.GraceDays = 15
	importClaims(t, client, private, claims)
	for _, tc := range []struct {
		time  time.Time
		state string
	}{{expiry.Add(-time.Nanosecond), "active"}, {expiry, "grace"}, {expiry.Add(15*24*time.Hour - time.Nanosecond), "grace"}, {expiry.Add(15 * 24 * time.Hour), "expired"}} {
		client.Now = func() time.Time { return tc.time }
		status, err := client.Status(context.Background())
		if err != nil || status.State != tc.state {
			t.Fatalf("%s: %+v %v", tc.state, status, err)
		}
	}
	client.ObserveClock(context.Background())
	client.Now = func() time.Time { return expiry.Add(-time.Hour) }
	status, _ := client.Status(context.Background())
	if status.State != "expired" || !status.ClockWarning {
		t.Fatal("clock rollback extended grace", status)
	}
	for _, op := range []Operation{ReadDocuments, VerifyIntegrity, BackupExport, SecurityAdministration} {
		if err := client.Check(context.Background(), op); err != nil {
			t.Fatal(err)
		}
	}
	expectCode(t, client.Check(context.Background(), WriteDocuments), "LICENSE_READ_ONLY")
	restarted, err := New(client.Database, client.stateDirectory, client.Options)
	if err != nil {
		t.Fatal(err)
	}
	restarted.Now = client.Now
	status, _ = restarted.Status(context.Background())
	if status.State != "expired" || !status.ClockWarning {
		t.Fatal("restart lost clock high-water")
	}
	perpetual, key := testClient(t)
	forever := testClaims(perpetual)
	importClaims(t, perpetual, key, forever)
	perpetual.Now = func() time.Time { return expiry.AddDate(50, 0, 0) }
	status, _ = perpetual.Status(context.Background())
	if status.State != "active" || !status.WriteAllowed || status.MaintenanceUntil != nil {
		t.Fatal("perpetual use expired")
	}
	if err = perpetual.AllowsRelease(context.Background(), forever.EntitledReleaseUntil); err != nil {
		t.Fatal(err)
	}
	expectCode(t, perpetual.AllowsRelease(context.Background(), expiry.Add(time.Hour).Format(time.RFC3339)), "LICENSE_RELEASE_NOT_ENTITLED")
}
func TestRevisionFloorsImportAtomicityAndOfflineProof(t *testing.T) {
	ctx := context.Background()
	client, private := testClient(t)
	claims := testClaims(client)
	original := signed(private, claims)
	importClaims(t, client, private, claims)
	importClaims(t, client, private, claims)
	var count int
	client.Database.Reader.QueryRow("SELECT count(*) FROM license_artifacts").Scan(&count)
	if count != 1 {
		t.Fatal("idempotent import duplicated evidence")
	}
	claims.Features["ocr"] = false
	_, err := client.Import(ctx, []byte(signed(private, claims)), nil, domain.RequestMetadata{})
	expectCode(t, err, "LICENSE_REVISION_CONFLICT")
	claims.Revision = 3
	importClaims(t, client, private, claims)
	_, err = client.Import(ctx, []byte(original), nil, domain.RequestMetadata{})
	expectCode(t, err, "LICENSE_REVISION_CONFLICT")
	denied := func(context.Context, *sql.Tx) (domain.Principal, error) {
		return domain.Principal{}, domain.Failure("FORBIDDEN", "denied", 403)
	}
	claims.Revision = 4
	_, err = client.Import(ctx, []byte(signed(private, claims)), denied, domain.RequestMetadata{})
	expectCode(t, err, "FORBIDDEN")
	state, _ := client.Status(ctx)
	if state.Revision != 3 || state.Features["ocr"] {
		t.Fatal("failed transaction changed license")
	}
	expectCode(t, client.CheckFeatures(ctx, "ocr"), "LICENSE_FEATURE")
	if err = client.Check(ctx, ReadDocuments); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"activate", "renew", "deactivate"} {
		id := domain.NewID()
		artifact, err := client.OfflineRequest(ctx, action, id, nil, domain.RequestMetadata{})
		if err != nil {
			t.Fatal(err)
		}
		_, payload, err := VerifyOfflineRequest([]byte(artifact.Contents))
		if err != nil || payload.Action != action || payload.Binding != client.binding {
			t.Fatal("offline proof invalid", err)
		}
		again, err := client.OfflineRequest(ctx, action, id, nil, domain.RequestMetadata{})
		if err != nil || again.ID != artifact.ID {
			t.Fatal("request replay duplicated")
		}
		var envelope OfflineEnvelope
		json.Unmarshal([]byte(artifact.Contents), &envelope)
		envelope.Payload = base64.RawURLEncoding.EncodeToString([]byte(`{}`))
		bytes, _ := json.Marshal(envelope)
		if _, _, err = VerifyOfflineRequest(bytes); err == nil {
			t.Fatal("tampered request accepted")
		}
	}
	state, _ = client.Status(ctx)
	if !state.DeactivationPending || !state.WriteAllowed {
		t.Fatal("offline request claimed immediate deactivation")
	}
	claims.Status = "revoked"
	importClaims(t, client, private, claims)
	expectCode(t, client.Check(ctx, WriteDocuments), "LICENSE_READ_ONLY")
	if err = client.Check(ctx, ReadDocuments); err != nil {
		t.Fatal(err)
	}
}
func TestMissingKeyRecoveryRetainsEvidenceAndRequiresNewActivation(t *testing.T) {
	ctx := context.Background()
	client, private := testClient(t)
	claims := testClaims(client)
	importClaims(t, client, private, claims)
	before := client.binding
	if err := os.Remove(identityPath(client.stateDirectory)); err != nil {
		t.Fatal(err)
	}
	lost, err := New(client.Database, client.stateDirectory, client.Options)
	if err != nil {
		t.Fatal(err)
	}
	status, _ := lost.Status(ctx)
	if status.State != "invalid" || status.Diagnostic != "LICENSE_IDENTITY_LOST" {
		t.Fatal("missing key silently replaced", status)
	}
	if err = RecoverIdentity(ctx, client.Database, client.stateDirectory, "Equipo reemplazado en prueba"); err != nil {
		t.Fatal(err)
	}
	recovered, err := New(client.Database, client.stateDirectory, client.Options)
	if err != nil {
		t.Fatal(err)
	}
	status, _ = recovered.Status(ctx)
	if status.State != "unactivated" || status.InstallationID == before.InstallationID {
		t.Fatal("recovery granted license", status)
	}
	_, err = recovered.Import(ctx, []byte(signed(private, claims)), nil, domain.RequestMetadata{})
	expectCode(t, err, "LICENSE_BINDING_MISMATCH")
	var evidence int
	client.Database.Reader.QueryRow("SELECT count(*) FROM license_artifacts").Scan(&evidence)
	if evidence != 1 {
		t.Fatal("recovery deleted evidence")
	}
	if err = client.Database.RollbackEmpty(ctx); err == nil {
		t.Fatal("used license database rolled back")
	}
}

func TestProofExactBytesAndLicenseHTTPSConfiguration(t *testing.T) {
	const installation = "33333333-3333-4333-8333-333333333333"
	const want = "LIC-V1\nactivate\nchallenge\nnonce\ngestor_documental\n33333333-3333-4333-8333-333333333333\n-"
	if string(ProofMessage("activate", "challenge", "nonce", installation, "")) != want {
		t.Fatal("proof byte contract changed")
	}
	for _, url := range []string{"http://127.0.0.1:9443", "https://user:password@example.test", "https://example.test/path", "https://example.test?key=secret"} {
		if err := (Options{ServerURL: url}).Validate(); err == nil {
			t.Fatal("unsafe server origin accepted")
		}
	}
	if !DevelopmentEnabled() {
		contents, err := os.ReadFile("../../testdata/license/v1/perpetual.lic")
		if err != nil {
			t.Fatal(err)
		}
		keyJSON, _ := os.ReadFile("../../testdata/license/v1/trust-key.json")
		var key TrustKey
		json.Unmarshal(keyJSON, &key)
		key.Environment = "production"
		var claims Claims
		payload, _ := os.ReadFile("../../testdata/license/v1/perpetual.payload.json")
		json.Unmarshal(payload, &claims)
		_, err = Verify(strings.TrimSpace(string(contents)), []TrustKey{key}, claims.Binding)
		expectCode(t, err, "LICENSE_UNKNOWN_KEY")
	}
}
