//go:build development

// Package licensefixture contains ONLY public development fixtures and a mock.
// It is excluded from every production build.
package licensefixture

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	"gestor-documental/internal/licensing"
)

const Kid = "dev-license-v1"

// RFC 8032 public test seed. Never use this published key for a real license.
const TestSeedHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

func PrivateKey() ed25519.PrivateKey {
	seed, _ := hex.DecodeString(TestSeedHex)
	return ed25519.NewKeyFromSeed(seed)
}
func TrustKey() licensing.TrustKey {
	return licensing.TrustKey{Kid: Kid, PublicKey: base64.RawURLEncoding.EncodeToString(PrivateKey().Public().(ed25519.PublicKey)), Environment: "development", Purpose: "license"}
}
func Sign(claims licensing.Claims) string {
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": Kid, "typ": "lic+jws"})
	payload, _ := json.Marshal(claims)
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(PrivateKey(), []byte(input)))
}
func Claims(binding licensing.Binding, instant time.Time) licensing.Claims {
	one := int64(1)
	features := map[string]bool{}
	for _, name := range licensing.FeatureNames {
		features[name] = true
	}
	return licensing.Claims{SchemaVersion: "1.0", ProductID: licensing.ProductID, LicenseID: "11111111-1111-4111-8111-111111111111", ActivationID: "22222222-2222-4222-8222-222222222222", Binding: binding, Type: "perpetual", Status: "active", Revision: 1, IssuedAt: instant.UTC().Format(time.RFC3339), EntitledReleaseUntil: instant.UTC().Format(time.RFC3339), Features: features, Limits: map[string]*int64{"max_installations": &one, "max_users": nil, "max_libraries": nil, "max_documents": nil}}
}
