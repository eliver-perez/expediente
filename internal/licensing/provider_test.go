package licensing

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestDefinitiveProviderTrust(t *testing.T) {
	options := ProviderOptions()
	if options.ServerURL != "https://licensing.aidex.com.mx" || options.RefreshHours != 24 || options.DevelopmentBypass || options.DevelopmentCAFile != "" || len(options.TrustedKeys) != 1 {
		t.Fatal("unexpected production provider configuration")
	}
	key := options.TrustedKeys[0]
	if key.Kid != "lic-prod-2026-a" || key.PublicKey != "X8U4PxM_kr3U56k27D6lVkKIEbLUUTAkJs_-jbXdwJM" || key.Environment != "production" || key.Purpose != "license" {
		t.Fatal("unexpected production signing trust")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(key.PublicKey)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		t.Fatal("invalid production Ed25519 public key")
	}
	if err = options.Validate(); err != nil {
		t.Fatal(err)
	}
	if ProductID != "gestor_documental" || AppVersion != "1.0.0" {
		t.Fatal("licensing contract identity changed")
	}
}
