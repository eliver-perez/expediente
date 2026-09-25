//go:build development

package licensefixture

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

type Activation struct {
	Key         string
	Claims      licensing.Claims
	Deactivated bool
}
type Cached struct {
	Digest   string
	Response json.RawMessage
}
type State struct {
	Activations map[string]Activation
	Responses   map[string]Cached
}
type challenge struct {
	Action, Installation, Activation, Nonce string
	Expires                                 time.Time
}
type Mock struct {
	mutex      sync.Mutex
	State      State
	challenges map[string]challenge
	Now        func() time.Time
	StatePath  string
}

func NewMock(path string) (*Mock, error) {
	mock := &Mock{State: State{Activations: map[string]Activation{}, Responses: map[string]Cached{}}, challenges: map[string]challenge{}, Now: time.Now, StatePath: path}
	if path != "" {
		contents, err := os.ReadFile(path)
		if err == nil {
			if err = json.Unmarshal(contents, &mock.State); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if mock.State.Activations == nil || mock.State.Responses == nil {
		return nil, fmt.Errorf("invalid mock state")
	}
	return mock, nil
}
func (mock *Mock) persist() error {
	if mock.StatePath == "" {
		return nil
	}
	contents, err := json.MarshalIndent(mock.State, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(mock.StatePath), "mock-state-")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if _, err = temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, mock.StatePath)
}
func reply(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
func reject(writer http.ResponseWriter, code, requestID string) {
	status := 422
	if code == "ACTIVATION_LIMIT" {
		status = 409
	} else if code == "LICENSE_NOT_FOUND" {
		status = 404
	} else if code == "TEMPORARY_UNAVAILABLE" {
		status = 503
	}
	reply(writer, status, map[string]any{"error": map[string]string{"code": code, "message": "Development simulator: " + code, "request_id": requestID}})
}
func (mock *Mock) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.TLS == nil || request.Method != "POST" {
		reject(writer, "INVALID_REQUEST", "")
		return
	}
	contents, err := io.ReadAll(io.LimitReader(request.Body, 65537))
	if err != nil || len(contents) > 65536 {
		reject(writer, "INVALID_REQUEST", "")
		return
	}
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	if request.URL.Path == "/v1/activations/challenge" {
		var input struct {
			Action       string  `json:"action"`
			Product      string  `json:"product_id"`
			Installation string  `json:"installation_id"`
			Activation   *string `json:"activation_id"`
		}
		if json.Unmarshal(contents, &input) != nil || input.Product != licensing.ProductID || input.Installation == "" || (input.Action != "activate" && input.Action != "refresh" && input.Action != "deactivate") || (input.Action == "activate") != (input.Activation == nil) {
			reject(writer, "INVALID_REQUEST", "")
			return
		}
		for id, value := range mock.challenges {
			if !mock.Now().Before(value.Expires) {
				delete(mock.challenges, id)
			}
		}
		nonce := make([]byte, 32)
		if _, err = rand.Read(nonce); err != nil {
			reject(writer, "TEMPORARY_UNAVAILABLE", "")
			return
		}
		value := challenge{Action: input.Action, Installation: input.Installation, Nonce: base64.RawURLEncoding.EncodeToString(nonce), Expires: mock.Now().Add(3 * time.Minute)}
		if input.Activation != nil {
			value.Activation = *input.Activation
		}
		id := domain.NewID()
		mock.challenges[id] = value
		reply(writer, 200, map[string]string{"challenge_id": id, "nonce": value.Nonce, "expires_at": value.Expires.UTC().Format(time.RFC3339)})
		return
	}
	action := ""
	switch request.URL.Path {
	case "/v1/activations":
		action = "activate"
	case "/v1/activations/refresh":
		action = "refresh"
	case "/v1/activations/deactivate":
		action = "deactivate"
	default:
		reject(writer, "INVALID_REQUEST", "")
		return
	}
	var input struct {
		RequestID  string `json:"request_id"`
		Product    string `json:"product_id"`
		LicenseKey string `json:"license_key"`
		licensing.Binding
		ActivationID string `json:"activation_id"`
		ChallengeID  string `json:"challenge_id"`
		Proof        string `json:"proof"`
		Version      string `json:"app_version"`
	}
	if json.Unmarshal(contents, &input) != nil || input.Product != licensing.ProductID || input.RequestID == "" || input.Version == "" {
		reject(writer, "INVALID_REQUEST", input.RequestID)
		return
	}
	digest := domain.Digest(action + "\n" + string(contents))
	if previous, ok := mock.State.Responses[input.RequestID]; ok {
		if previous.Digest != digest {
			reject(writer, "INVALID_REQUEST", input.RequestID)
			return
		}
		reply(writer, 200, previous.Response)
		return
	}
	proof, ok := mock.challenges[input.ChallengeID]
	if !ok || !mock.Now().Before(proof.Expires) || proof.Action != action || proof.Installation != input.InstallationID || proof.Activation != input.ActivationID {
		reject(writer, "INVALID_PROOF", input.RequestID)
		return
	}
	delete(mock.challenges, input.ChallengeID)
	var activation Activation
	binding := input.Binding
	if action != "activate" {
		var found bool
		activation, found = mock.State.Activations[input.ActivationID]
		if !found || activation.Deactivated || activation.Claims.InstallationID != input.InstallationID {
			reject(writer, "INVALID_PROOF", input.RequestID)
			return
		}
		binding = activation.Claims.Binding
	}
	public, err := base64.RawURLEncoding.Strict().DecodeString(binding.PublicKey)
	signature, signatureErr := base64.RawURLEncoding.Strict().DecodeString(input.Proof)
	if err != nil || signatureErr != nil || len(public) != 32 || !ed25519.Verify(public, licensing.ProofMessage(action, input.ChallengeID, proof.Nonce, input.InstallationID, input.ActivationID), signature) {
		reject(writer, "INVALID_PROOF", input.RequestID)
		return
	}
	var response any
	if action == "activate" {
		activation, err = mock.activate(binding, input.LicenseKey)
		if err != nil {
			reject(writer, err.Error(), input.RequestID)
			return
		}
	} else if action == "deactivate" {
		activation.Deactivated = true
		mock.State.Activations[input.ActivationID] = activation
		response = map[string]string{"activation_id": input.ActivationID, "deactivated_at": mock.Now().UTC().Format(time.RFC3339), "status": "deactivated"}
	} else {
		activation.Claims.Revision++
		activation.Claims.IssuedAt = mock.Now().UTC().Format(time.RFC3339)
		mock.State.Activations[input.ActivationID] = activation
	}
	if response == nil {
		response = map[string]string{"license_jws": Sign(activation.Claims), "server_time": mock.Now().UTC().Format(time.RFC3339), "request_id": input.RequestID}
	}
	encoded, _ := json.Marshal(response)
	mock.State.Responses[input.RequestID] = Cached{Digest: digest, Response: encoded}
	if err = mock.persist(); err != nil {
		reject(writer, "TEMPORARY_UNAVAILABLE", input.RequestID)
		return
	}
	reply(writer, 200, response)
}
func (mock *Mock) activate(binding licensing.Binding, key string) (Activation, error) {
	kind := strings.TrimPrefix(key, "DEMO-")
	switch kind {
	case "PERPETUAL", "SUBSCRIPTION", "GRACE", "EXPIRED", "REVOKED", "LINKED":
	default:
		return Activation{}, fmt.Errorf("LICENSE_NOT_FOUND")
	}
	commercialID := ""
	for _, previous := range mock.State.Activations {
		if previous.Key == key {
			if !previous.Deactivated {
				return Activation{}, fmt.Errorf("ACTIVATION_LIMIT")
			}
			commercialID = previous.Claims.LicenseID
		}
	}
	claims := Claims(binding, mock.Now())
	claims.LicenseID = commercialID
	if claims.LicenseID == "" {
		claims.LicenseID = domain.NewID()
	}
	claims.ActivationID = domain.NewID()
	if kind == "SUBSCRIPTION" || kind == "GRACE" || kind == "EXPIRED" {
		claims.Type = "subscription"
		claims.GraceDays = 15
		expiry := mock.Now().Add(30 * 24 * time.Hour)
		if kind == "GRACE" {
			expiry = mock.Now().Add(-24 * time.Hour)
		}
		if kind == "EXPIRED" {
			expiry = mock.Now().Add(-16 * 24 * time.Hour)
		}
		value := expiry.UTC().Format(time.RFC3339)
		claims.ExpiresAt = &value
		claims.EntitledReleaseUntil = value
	}
	if kind == "REVOKED" {
		claims.Status = "revoked"
	}
	if kind == "LINKED" {
		claims.Features["managed_libraries"] = false
		claims.Features["expedientes"] = false
		claims.Features["review_workflow"] = false
	}
	// Exercise the same verifier before issuing a development fixture.
	if _, err := licensing.Verify(Sign(claims), []licensing.TrustKey{TrustKey()}, binding); err != nil {
		return Activation{}, fmt.Errorf("INVALID_REQUEST")
	}
	activation := Activation{Key: key, Claims: claims}
	mock.State.Activations[claims.ActivationID] = activation
	return activation, nil
}

// Offline processes the exact V1 envelope. Deactivation yields evidence for the
// provider; there is deliberately no invented .lic acknowledgement format.
func (mock *Mock) Offline(contents []byte, key string) ([]byte, error) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	envelope, payload, err := licensing.VerifyOfflineRequest(contents)
	if err != nil {
		return nil, err
	}
	_ = envelope
	digest := domain.Digest(string(contents) + "\n" + key)
	if previous, ok := mock.State.Responses[payload.RequestID]; ok {
		if previous.Digest != digest {
			return nil, fmt.Errorf("INVALID_REQUEST")
		}
		var output string
		if err = json.Unmarshal(previous.Response, &output); err != nil {
			return nil, err
		}
		return []byte(output), nil
	}
	var activation Activation
	if payload.Action == "activate" {
		activation, err = mock.activate(payload.Binding, key)
	} else {
		var ok bool
		activation, ok = mock.State.Activations[payload.ActivationID]
		if !ok || activation.Deactivated || activation.Claims.LicenseID != payload.LicenseID || activation.Claims.Binding != payload.Binding {
			return nil, fmt.Errorf("INVALID_PROOF")
		}
		if payload.Action == "deactivate" {
			activation.Deactivated = true
		} else {
			activation.Claims.Revision++
			activation.Claims.IssuedAt = mock.Now().UTC().Format(time.RFC3339)
		}
		mock.State.Activations[payload.ActivationID] = activation
	}
	if err != nil {
		return nil, err
	}
	output := Sign(activation.Claims)
	if payload.Action == "deactivate" {
		evidence, _ := json.Marshal(map[string]string{"activation_id": payload.ActivationID, "deactivated_at": mock.Now().UTC().Format(time.RFC3339), "status": "deactivated"})
		output = string(evidence)
	}
	encoded, _ := json.Marshal(output)
	mock.State.Responses[payload.RequestID] = Cached{Digest: digest, Response: encoded}
	if err = mock.persist(); err != nil {
		return nil, err
	}
	return bytes.TrimSpace([]byte(output)), nil
}

// Revise changes a mock activation for contract tests, never production data.
func (mock *Mock) Revise(identifier string, change func(*licensing.Claims)) error {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	item, ok := mock.State.Activations[identifier]
	if !ok {
		return fmt.Errorf("unknown development activation")
	}
	change(&item.Claims)
	mock.State.Activations[identifier] = item
	return mock.persist()
}
