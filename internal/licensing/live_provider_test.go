package licensing_test

import (
	"context"
	"encoding/json"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Explicit opt-in; the commercial key is read from a private file, never logged.
// Keep a failed activation's state for an identical retry / confirmed deactivation.
func TestLiveProvider(t *testing.T) {
	keyPath := os.Getenv("AIBID_TEST_LICENSE_FILE")
	stateDirectory := os.Getenv("AIBID_TEST_LICENSE_STATE")
	if keyPath == "" || stateDirectory == "" {
		t.Skip("live provider test requires isolated state and an authorized test license")
	}
	contents, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	database, err := storage.Open(ctx, stateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	client, err := licensing.New(database, stateDirectory, licensing.ProviderOptions())
	if err != nil {
		t.Fatal(err)
	}
	requestIDPath := filepath.Join(stateDirectory, "activation-request-id")
	idBytes, _ := os.ReadFile(requestIDPath)
	id := strings.TrimSpace(string(idBytes))
	if id == "" {
		id = domain.NewID()
		if err = os.WriteFile(requestIDPath, []byte(id), 0600); err != nil {
			t.Fatal(err)
		}
	}
	state, err := client.Online(ctx, "activate", id, strings.TrimSpace(string(contents)), nil, domain.RequestMetadata{RequestID: domain.NewID()})
	if err != nil {
		t.Fatal(err)
	}
	if state.State != "active" || state.LicenseType != "perpetual" || !state.WriteAllowed {
		t.Fatal("provider activation did not authorize a perpetual license")
	}
	defer func() {
		if _, err := client.Online(ctx, "deactivate", domain.NewID(), "", nil, domain.RequestMetadata{RequestID: domain.NewID()}); err != nil {
			t.Error("temporary installation still requires confirmed deactivation:", err)
		} else {
			t.Log("temporary activation deactivated; commercial slot released")
		}
	}()
	for _, feature := range []string{"linked_libraries", "managed_libraries", "ocr", "expedientes", "review_workflow"} {
		if !state.Features[feature] {
			t.Error("missing feature:", feature)
		}
	}
	t.Log("HTTPS activation, Ed25519 signature, binding, perpetual policy and all five modules accepted")
	if _, err = client.Online(ctx, "refresh", domain.NewID(), "", nil, domain.RequestMetadata{RequestID: domain.NewID()}); err != nil {
		t.Fatal(err)
	}
	t.Log("challenge/proof refresh accepted")
	// The signed artifact can be imported again offline without creating an activation.
	var jws string
	if err = database.Reader.QueryRow("SELECT current_jws FROM license_state WHERE singleton=1").Scan(&jws); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Import(ctx, []byte(jws), nil, domain.RequestMetadata{RequestID: domain.NewID()}); err != nil {
		t.Fatal(err)
	}
	request, err := client.OfflineRequest(ctx, "renew", domain.NewID(), nil, domain.RequestMetadata{RequestID: domain.NewID()})
	if err != nil {
		t.Fatal(err)
	}
	var envelope licensing.OfflineEnvelope
	if json.Unmarshal([]byte(request.Contents), &envelope) != nil || envelope.Signature == "" {
		t.Fatal("offline request malformed")
	}
	t.Log("signed .lic import and .licreq renewal generation accepted; server-side offline fulfillment remains separate")
}
