//go:build development

package licensefixture

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

func GenerateVectors(directory string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	hashes := map[string]string{}
	write := func(name string, contents []byte) error {
		hashes[name] = domain.Digest(string(contents))
		return os.WriteFile(filepath.Join(directory, name), contents, 0644)
	}
	instant, _ := time.Parse(time.RFC3339, "2026-09-22T18:30:00Z")
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, 32))
	binding := licensing.Binding{InstallationID: "33333333-3333-4333-8333-333333333333", PublicKey: base64.RawURLEncoding.EncodeToString(private.Public().(ed25519.PublicKey)), FingerprintVersion: "1", FingerprintHash: "sha256:" + strings.Repeat("a", 64)}
	claims := Claims(binding, instant)
	payload, _ := json.MarshalIndent(claims, "", "  ")
	if err := write("perpetual.payload.json", append(payload, '\n')); err != nil {
		return err
	}
	key, _ := json.MarshalIndent(TrustKey(), "", "  ")
	if err := write("trust-key.json", append(key, '\n')); err != nil {
		return err
	}
	type vector struct {
		File     string `json:"file"`
		Expected string `json:"expected"`
	}
	cases := []vector{}
	add := func(name, compact, expected string) error {
		cases = append(cases, vector{name, expected})
		return write(name, []byte(compact+"\n"))
	}
	if err := add("perpetual.lic", Sign(claims), "active"); err != nil {
		return err
	}
	for _, kind := range []string{"subscription", "revoked", "other-installation", "other-fingerprint", "wrong-product", "lower-revision", "higher-revision", "modules-linked-only"} {
		item := Claims(binding, instant)
		switch kind {
		case "subscription":
			expiry := "2026-10-22T18:30:00Z"
			item.Type = "subscription"
			item.ExpiresAt = &expiry
			item.GraceDays = 15
			item.EntitledReleaseUntil = expiry
		case "revoked":
			item.Status = "revoked"
			item.Revision = 2
		case "other-installation":
			item.InstallationID = "44444444-4444-4444-8444-444444444444"
		case "other-fingerprint":
			item.FingerprintHash = "sha256:" + strings.Repeat("b", 64)
		case "wrong-product":
			item.ProductID = "other_product"
		case "lower-revision":
			item.Revision = 1
		case "higher-revision":
			item.Revision = 3
		case "modules-linked-only":
			item.Revision = 2
			item.Features["managed_libraries"] = false
			item.Features["expedientes"] = false
			item.Features["review_workflow"] = false
		}
		expected := "valid"
		if strings.HasPrefix(kind, "other-") {
			expected = "LICENSE_BINDING_MISMATCH"
		}
		if kind == "wrong-product" {
			expected = "LICENSE_INVALID"
		}
		if kind == "lower-revision" {
			expected = "LICENSE_REVISION_CONFLICT after higher-revision"
		}
		if err := add(kind+".lic", Sign(item), expected); err != nil {
			return err
		}
	}
	corrupt := Sign(claims)
	parts := strings.Split(corrupt, ".")
	signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
	signature[0] ^= 1
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	if err := add("invalid-signature.lic", strings.Join(parts, "."), "LICENSE_INVALID_SIGNATURE"); err != nil {
		return err
	}
	proof := licensing.ProofMessage("activate", "55555555-5555-4555-8555-555555555555", "development-nonce-0123456789", binding.InstallationID, "")
	proofVector, _ := json.MarshalIndent(map[string]string{"action": "activate", "challenge_id": "55555555-5555-4555-8555-555555555555", "nonce": "development-nonce-0123456789", "installation_id": binding.InstallationID, "installation_public_key": binding.PublicKey, "message_utf8": string(proof), "proof": base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, proof))}, "", "  ")
	if err := write("proof.json", append(proofVector, '\n')); err != nil {
		return err
	}
	for _, action := range []string{"activate", "renew", "deactivate"} {
		value := licensing.OfflinePayload{Action: action, RequestID: "66666666-6666-4666-8666-666666666666", CreatedAt: instant.Format(time.RFC3339), ProductID: licensing.ProductID, Binding: binding}
		if action != "activate" {
			value.LicenseID = claims.LicenseID
			value.ActivationID = claims.ActivationID
		}
		bytes, _ := json.Marshal(value)
		encoded := base64.RawURLEncoding.EncodeToString(bytes)
		envelope := licensing.OfflineEnvelope{SchemaVersion: "1.0", Payload: encoded, Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, []byte("LICREQ-V1\n"+encoded)))}
		bytes, _ = json.MarshalIndent(envelope, "", "  ")
		if err := write(action+".licreq", append(bytes, '\n')); err != nil {
			return err
		}
	}
	boundaries := []map[string]string{{"now": "2026-10-22T18:29:59.999999999Z", "expected": "active"}, {"now": "2026-10-22T18:30:00Z", "expected": "grace"}, {"now": "2026-11-06T18:29:59.999999999Z", "expected": "grace"}, {"now": "2026-11-06T18:30:00Z", "expected": "expired"}}
	data, _ := json.MarshalIndent(boundaries, "", "  ")
	if err := write("boundaries.json", append(data, '\n')); err != nil {
		return err
	}
	manifest, _ := json.MarshalIndent(map[string]any{"schema_version": "1.0", "fixture_version": 1, "test_keys_only": true, "production_expected": "reject every development JWS, even renamed kid", "cases": cases, "sha256": hashes}, "", "  ")
	return os.WriteFile(filepath.Join(directory, "manifest.json"), append(manifest, '\n'), 0644)
}
