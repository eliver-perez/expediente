//go:build development

package licensing_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensefixture"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

func developmentClient(t *testing.T, server *httptest.Server) (*licensing.Client, string) {
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
	options := licensing.Options{TrustedKeys: []licensing.TrustKey{licensefixture.TrustKey()}}
	if server != nil {
		certificate := filepath.Join(directory, "mock.crt")
		if err = os.WriteFile(certificate, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
			t.Fatal(err)
		}
		options.ServerURL = server.URL
		options.DevelopmentCAFile = certificate
	}
	client, err := licensing.New(database, directory, options)
	if err != nil {
		t.Fatal(err)
	}
	return client, directory
}
func TestOnlineIdempotentRestartRevocationAndConfirmedDeactivation(t *testing.T) {
	ctx := context.Background()
	mock, err := licensefixture.NewMock("")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	var bodies []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/activations" {
			contents, _ := io.ReadAll(request.Body)
			bodies = append(bodies, string(contents))
			request.Body = io.NopCloser(strings.NewReader(string(contents)))
			if calls.Add(1) == 1 {
				record := httptest.NewRecorder()
				mock.ServeHTTP(record, request)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(503)
				io.WriteString(writer, `{"error":{"code":"TEMPORARY_UNAVAILABLE","message":"Simulated response lost after activation"}}`)
				return
			}
		}
		mock.ServeHTTP(writer, request)
	}))
	defer server.Close()
	client, directory := developmentClient(t, server)
	id := domain.NewID()
	if _, err = client.Online(ctx, "activate", id, "DEMO-PERPETUAL", nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("lost response not detected")
	}
	pending, err := client.PendingOperations(ctx)
	if err != nil || len(pending) != 1 || pending[0].RequestID != id || pending[0].Action != "activate" {
		t.Fatal("pending request cannot be resumed", pending, err)
	}
	state, _ := client.Status(ctx)
	if state.State != "unactivated" {
		t.Fatal("network failure fabricated license")
	}
	restarted, err := licensing.New(client.Database, directory, client.Options)
	if err != nil {
		t.Fatal(err)
	}
	state, err = restarted.Online(ctx, "activate", id, "DEMO-PERPETUAL", nil, domain.RequestMetadata{})
	if err != nil || state.State != "active" {
		t.Fatal(state, err)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] {
		t.Fatal("restart retried different bytes")
	}
	originalRevision := state.Revision
	originalLicenseID, originalActivationID := state.LicenseID, state.ActivationID
	state, err = restarted.Online(ctx, "activate", id, "DEMO-PERPETUAL", nil, domain.RequestMetadata{})
	if err != nil || state.Revision != originalRevision || calls.Load() != 2 {
		t.Fatal("successful replay contacted server")
	}
	if _, err = restarted.Online(ctx, "activate", id, "DEMO-SUBSCRIPTION", nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("different key reused request")
	}
	var leaks int
	err = client.Database.Reader.QueryRow("SELECT count(*) FROM license_operations WHERE instr(request_json,'DEMO-')>0 OR instr(request_digest,'DEMO-')>0").Scan(&leaks)
	if err != nil || leaks != 0 {
		t.Fatal("commercial key persisted", err)
	}
	second, _ := developmentClient(t, server)
	if _, err = second.Online(ctx, "activate", domain.NewID(), "DEMO-PERPETUAL", nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("second installation activated")
	}
	if err = mock.Revise(state.ActivationID, func(claims *licensing.Claims) { claims.Status = "revoked" }); err != nil {
		t.Fatal(err)
	}
	state, err = restarted.Online(ctx, "refresh", domain.NewID(), "", nil, domain.RequestMetadata{})
	if err != nil || state.State != "revoked" || state.WriteAllowed || !state.ReadAllowed {
		t.Fatal("signed revocation not applied", state, err)
	}
	state, err = restarted.Online(ctx, "deactivate", domain.NewID(), "", nil, domain.RequestMetadata{})
	if err != nil || state.Diagnostic != "ACTIVATION_DEACTIVATED" {
		t.Fatal(state, err)
	}
	state, err = second.Online(ctx, "activate", domain.NewID(), "DEMO-PERPETUAL", nil, domain.RequestMetadata{})
	if err != nil || !state.WriteAllowed {
		t.Fatal("confirmed deactivation did not free simulator slot", err)
	}
	if state.LicenseID != originalLicenseID || state.ActivationID == originalActivationID {
		t.Fatal("transfer changed commercial license or reused activation")
	}
	server.Close()
	before := state.Revision
	if _, err = second.Online(ctx, "refresh", domain.NewID(), "", nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("offline server succeeded")
	}
	state, _ = second.Status(ctx)
	if !state.WriteAllowed || state.Revision != before {
		t.Fatal("network outage invalidated perpetual license")
	}
}
func TestOfflineActivationRenewalAndPendingDeactivation(t *testing.T) {
	ctx := context.Background()
	client, _ := developmentClient(t, nil)
	mock, _ := licensefixture.NewMock("")
	request, err := client.OfflineRequest(ctx, "activate", domain.NewID(), nil, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	response, err := mock.Offline([]byte(request.Contents), "DEMO-PERPETUAL")
	if err != nil {
		t.Fatal(err)
	}
	state, err := client.Import(ctx, response, nil, domain.RequestMetadata{})
	if err != nil || state.State != "active" {
		t.Fatal(state, err)
	}
	request, err = client.OfflineRequest(ctx, "renew", domain.NewID(), nil, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	response, err = mock.Offline([]byte(request.Contents), "")
	if err != nil {
		t.Fatal(err)
	}
	state, err = client.Import(ctx, response, nil, domain.RequestMetadata{})
	if err != nil || state.Revision != 2 {
		t.Fatal("offline renewal failed", err)
	}
	replay, err := mock.Offline([]byte(request.Contents), "")
	if err != nil || string(response) != string(replay) {
		t.Fatal("offline response not idempotent")
	}
	request, err = client.OfflineRequest(ctx, "deactivate", domain.NewID(), nil, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	response, err = mock.Offline([]byte(request.Contents), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Import(ctx, response, nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("uncontracted deactivation JSON imported as license")
	}
	state, _ = client.Status(ctx)
	if !state.WriteAllowed || !state.DeactivationPending {
		t.Fatal("remote manual decision claimed knowledge on offline client")
	}
}
func TestDevelopmentVectorsAreReproducibleAndCrossCheckProofs(t *testing.T) {
	directory := t.TempDir()
	if err := licensefixture.GenerateVectors(directory); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir("../../testdata/license/v1")
	if err != nil {
		t.Fatal(err)
	}
	var claims licensing.Claims
	payload, _ := os.ReadFile("../../testdata/license/v1/perpetual.payload.json")
	if err = json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "README.md" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join("../../testdata/license/v1", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		generated, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil || string(contents) != string(generated) {
			t.Fatal("fixture drift", entry.Name(), err)
		}
		if strings.HasSuffix(entry.Name(), ".licreq") {
			if _, _, err = licensing.VerifyOfflineRequest(contents); err != nil {
				t.Fatal(entry.Name(), err)
			}
		}
	}
	for _, name := range []string{"perpetual.lic", "subscription.lic", "revoked.lic", "higher-revision.lic", "modules-linked-only.lic"} {
		contents, _ := os.ReadFile(filepath.Join(directory, name))
		if _, err = licensing.Verify(strings.TrimSpace(string(contents)), []licensing.TrustKey{licensefixture.TrustKey()}, claims.Binding); err != nil {
			t.Fatal(name, err)
		}
	}
	for _, name := range []string{"invalid-signature.lic", "other-installation.lic", "other-fingerprint.lic", "wrong-product.lic"} {
		contents, _ := os.ReadFile(filepath.Join(directory, name))
		if _, err = licensing.Verify(strings.TrimSpace(string(contents)), []licensing.TrustKey{licensefixture.TrustKey()}, claims.Binding); err == nil {
			t.Fatal("negative vector accepted", name)
		}
	}
	var boundaries []struct {
		Now      string `json:"now"`
		Expected string `json:"expected"`
	}
	contents, _ := os.ReadFile(filepath.Join(directory, "boundaries.json"))
	json.Unmarshal(contents, &boundaries)
	for _, point := range boundaries {
		instant, err := time.Parse(time.RFC3339Nano, point.Now)
		if err != nil || instant.IsZero() || point.Expected == "" {
			t.Fatal("bad boundary vector")
		}
	}
}

func TestOnlineInvalidResponsesDoNotReplaceValidLicense(t *testing.T) {
	for _, variant := range []string{"request_id", "server_time", "signature", "binding", "duplicate_json"} {
		t.Run(variant, func(t *testing.T) {
			mock, _ := licensefixture.NewMock("")
			var altered atomic.Bool
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/v1/activations/refresh" || altered.Swap(true) {
					mock.ServeHTTP(writer, request)
					return
				}
				recorder := httptest.NewRecorder()
				mock.ServeHTTP(recorder, request)
				var response map[string]string
				json.Unmarshal(recorder.Body.Bytes(), &response)
				switch variant {
				case "request_id":
					response["request_id"] = domain.NewID()
				case "server_time":
					response["server_time"] = "not-a-date"
				case "signature":
					response["license_jws"] += "x"
				case "binding":
					pieces := strings.Split(response["license_jws"], ".")
					payload, _ := base64.RawURLEncoding.DecodeString(pieces[1])
					var claims licensing.Claims
					json.Unmarshal(payload, &claims)
					claims.InstallationID = domain.NewID()
					response["license_jws"] = licensefixture.Sign(claims)
				}
				writer.Header().Set("Content-Type", "application/json")
				contents, _ := json.Marshal(response)
				if variant == "duplicate_json" {
					contents = []byte(strings.Replace(string(contents), `"request_id":`, `"REQUEST_ID":"ambiguous","request_id":`, 1))
				}
				writer.Write(contents)
			}))
			defer server.Close()
			client, _ := developmentClient(t, server)
			ctx := context.Background()
			before, err := client.Online(ctx, "activate", domain.NewID(), "DEMO-PERPETUAL", nil, domain.RequestMetadata{})
			if err != nil {
				t.Fatal(err)
			}
			id := domain.NewID()
			if _, err = client.Online(ctx, "refresh", id, "", nil, domain.RequestMetadata{}); err == nil {
				t.Fatal("invalid remote response accepted")
			}
			after, _ := client.Status(ctx)
			if after.Revision != before.Revision || !after.WriteAllowed {
				t.Fatal("failed verification replaced license")
			}
			after, err = client.Online(ctx, "refresh", id, "", nil, domain.RequestMetadata{})
			if err != nil || after.Revision != before.Revision+1 {
				t.Fatal("invalid response retry lost idempotence", err)
			}
		})
	}
}
func TestOnlineRequiresTrustedTLSAndNeverFollowsRedirects(t *testing.T) {
	ctx := context.Background()
	mock, _ := licensefixture.NewMock("")
	server := httptest.NewTLSServer(mock)
	defer server.Close()
	client, directory := developmentClient(t, server)
	options := client.Options
	options.DevelopmentCAFile = ""
	untrusted, err := licensing.New(client.Database, directory, options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = untrusted.Online(ctx, "activate", domain.NewID(), "DEMO-PERPETUAL", nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
	var redirected atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { redirected.Add(1); writer.WriteHeader(500) }))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	client, _ = developmentClient(t, redirect)
	if _, err = client.Online(ctx, "activate", domain.NewID(), "DEMO-PERPETUAL", nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("redirect accepted")
	}
	if redirected.Load() != 0 {
		t.Fatal("license identity sent to redirect target")
	}
}
