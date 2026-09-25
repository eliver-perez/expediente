package licensing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/domain"
)

const ProductID = "gestor_documental"
const AppVersion = buildinfo.Version
const MaximumArtifactBytes = 65536
const developmentPublicKeyHex = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var installationPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
var FeatureNames = []string{"linked_libraries", "managed_libraries", "ocr", "expedientes", "review_workflow"}

type Binding struct {
	InstallationID     string `json:"installation_id"`
	PublicKey          string `json:"installation_public_key"`
	FingerprintVersion string `json:"fingerprint_version"`
	FingerprintHash    string `json:"fingerprint_hash"`
}
type Claims struct {
	SchemaVersion string `json:"schema_version"`
	ProductID     string `json:"product_id"`
	LicenseID     string `json:"license_id"`
	ActivationID  string `json:"activation_id"`
	Binding
	Type                 string            `json:"license_type"`
	Status               string            `json:"license_status"`
	Revision             int64             `json:"license_revision"`
	IssuedAt             string            `json:"issued_at"`
	ExpiresAt            *string           `json:"expires_at"`
	GraceDays            int               `json:"grace_days"`
	MaintenanceUntil     *string           `json:"maintenance_until"`
	EntitledReleaseUntil string            `json:"entitled_release_until"`
	Features             map[string]bool   `json:"features"`
	Limits               map[string]*int64 `json:"limits"`
}
type TrustKey struct {
	Kid         string `json:"kid"`
	PublicKey   string `json:"public_key"`
	Environment string `json:"environment"`
	Purpose     string `json:"purpose"`
}

func failure(code, message string) error { return domain.Failure(code, message, 422) }
func contractFailure() error {
	return failure("LICENSE_INVALID", "El archivo no cumple el contrato de licencia V1.0.")
}
func decodeBase64(value string, size int) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != value || size > 0 && len(decoded) != size {
		return nil, contractFailure()
	}
	return decoded, nil
}
func utcInstant(value string) (time.Time, error) {
	instant, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return instant, contractFailure()
	}
	_, offset := instant.Zone()
	if offset != 0 {
		return instant, contractFailure()
	}
	return instant.UTC(), nil
}

// Reject duplicate names (including case aliases used by encoding/json), excessive
// nesting, non-UTF8 and trailing values before interpreting any signed structure.
func strictObject(contents []byte) (map[string]json.RawMessage, error) {
	if len(bytes.TrimSpace(contents)) == 0 || len(contents) > MaximumArtifactBytes || !utf8.Valid(contents) || bytes.TrimSpace(contents)[0] != '{' {
		return nil, contractFailure()
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 24 {
			return contractFailure()
		}
		token, err := decoder.Token()
		if err != nil {
			return contractFailure()
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		if delimiter != '{' && delimiter != '[' {
			return contractFailure()
		}
		seen := map[string]bool{}
		for decoder.More() {
			if delimiter == '{' {
				keyToken, err := decoder.Token()
				if err != nil {
					return contractFailure()
				}
				key, ok := keyToken.(string)
				if !ok {
					return contractFailure()
				}
				key = strings.ToLower(key)
				if seen[key] {
					return contractFailure()
				}
				seen[key] = true
			}
			if err = walk(depth + 1); err != nil {
				return err
			}
		}
		closeToken, err := decoder.Token()
		if err != nil || delimiter == '{' && closeToken != json.Delim('}') || delimiter == '[' && closeToken != json.Delim(']') {
			return contractFailure()
		}
		return nil
	}
	if err := walk(0); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, contractFailure()
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(contents, &object) != nil {
		return nil, contractFailure()
	}
	return object, nil
}
func fieldsPresent(object map[string]json.RawMessage, names ...string) bool {
	for _, name := range names {
		if _, ok := object[name]; !ok {
			return false
		}
	}
	return true
}
func Verify(compact string, keys []TrustKey, binding Binding) (Claims, error) {
	var claims Claims
	if len(compact) > MaximumArtifactBytes {
		return claims, contractFailure()
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return claims, contractFailure()
	}
	header, err := decodeBase64(parts[0], 0)
	if err != nil || len(header) > 1024 {
		return claims, contractFailure()
	}
	object, err := strictObject(header)
	if err != nil || len(object) != 3 || !fieldsPresent(object, "alg", "kid", "typ") {
		return claims, contractFailure()
	}
	var algorithm, kid, kind string
	if json.Unmarshal(object["alg"], &algorithm) != nil || json.Unmarshal(object["kid"], &kid) != nil || json.Unmarshal(object["typ"], &kind) != nil || algorithm != "EdDSA" || kind != "lic+jws" || !identifierPattern.MatchString(kid) {
		return claims, contractFailure()
	}
	var publicKey []byte
	for _, key := range keys {
		if key.Kid == kid && key.Purpose == "license" && (key.Environment == "production" || developmentEnabled && key.Environment == "development") {
			publicKey, err = decodeBase64(key.PublicKey, ed25519.PublicKeySize)
			if err != nil {
				return claims, err
			}
			if !developmentEnabled && (strings.HasPrefix(strings.ToLower(kid), "dev-") || fmt.Sprintf("%x", publicKey) == developmentPublicKeyHex) {
				publicKey = nil
			}
			break
		}
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return claims, failure("LICENSE_UNKNOWN_KEY", "La firma requiere una clave pública autorizada para este entorno.")
	}
	signature, err := decodeBase64(parts[2], ed25519.SignatureSize)
	if err != nil || !ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), signature) {
		return claims, failure("LICENSE_INVALID_SIGNATURE", "La firma de la licencia no es válida.")
	}
	payload, err := decodeBase64(parts[1], 0)
	if err != nil {
		return claims, err
	}
	object, err = strictObject(payload)
	if err != nil || !fieldsPresent(object, "schema_version", "product_id", "license_id", "activation_id", "installation_id", "installation_public_key", "fingerprint_version", "fingerprint_hash", "license_type", "license_status", "license_revision", "issued_at", "expires_at", "grace_days", "maintenance_until", "entitled_release_until", "features", "limits") {
		return claims, contractFailure()
	}
	if err = json.Unmarshal(payload, &claims); err != nil {
		return claims, contractFailure()
	}
	if claims.SchemaVersion != "1.0" {
		return claims, failure("INCOMPATIBLE_SCHEMA", "Esta versión no reconoce el esquema de licencia.")
	}
	if claims.ProductID != ProductID || !uuidPattern.MatchString(claims.LicenseID) || !uuidPattern.MatchString(claims.ActivationID) || claims.Revision < 1 || (claims.Status != "active" && claims.Status != "revoked") {
		return claims, contractFailure()
	}
	if claims.Binding != binding || !installationPattern.MatchString(claims.InstallationID) || claims.FingerprintVersion != "1" || !fingerprintPattern.MatchString(claims.FingerprintHash) {
		return claims, failure("LICENSE_BINDING_MISMATCH", "La licencia no corresponde a esta instalación, clave o equipo.")
	}
	if _, err = decodeBase64(claims.PublicKey, ed25519.PublicKeySize); err != nil {
		return claims, err
	}
	_, err = utcInstant(claims.IssuedAt)
	if err != nil {
		return claims, err
	}
	entitled, err := utcInstant(claims.EntitledReleaseUntil)
	if err != nil || entitled.IsZero() {
		return claims, contractFailure()
	}
	if string(object["grace_days"]) != "0" && string(object["grace_days"]) != "15" {
		return claims, contractFailure()
	}
	if claims.Type == "perpetual" {
		if claims.ExpiresAt != nil || claims.GraceDays != 0 {
			return claims, contractFailure()
		}
	} else if claims.Type == "subscription" {
		if claims.ExpiresAt == nil || claims.GraceDays != 15 || claims.MaintenanceUntil != nil {
			return claims, contractFailure()
		}
		_, err := utcInstant(*claims.ExpiresAt)
		if err != nil {
			return claims, contractFailure()
		}
	} else {
		return claims, contractFailure()
	}
	if claims.MaintenanceUntil != nil {
		if _, err = utcInstant(*claims.MaintenanceUntil); err != nil {
			return claims, err
		}
	}
	features, err := strictObject(object["features"])
	if err != nil || !fieldsPresent(features, FeatureNames...) {
		return claims, contractFailure()
	}
	for _, value := range features {
		if string(value) != "true" && string(value) != "false" {
			return claims, contractFailure()
		}
	}
	if claims.Features["review_workflow"] && !claims.Features["expedientes"] {
		return claims, failure("LICENSE_FEATURE_COMBINATION", "El módulo de revisión requiere expedientes.")
	}
	limits, err := strictObject(object["limits"])
	if err != nil || !fieldsPresent(limits, "max_installations", "max_users", "max_libraries", "max_documents") {
		return claims, contractFailure()
	}
	if claims.Limits["max_installations"] == nil || *claims.Limits["max_installations"] != 1 || claims.Limits["max_users"] != nil || claims.Limits["max_libraries"] != nil || claims.Limits["max_documents"] != nil {
		return claims, contractFailure()
	}
	return claims, nil
}
func ProofMessage(action, challengeID, nonce, installationID, activationID string) []byte {
	if activationID == "" {
		activationID = "-"
	}
	return []byte("LIC-V1\n" + action + "\n" + challengeID + "\n" + nonce + "\n" + ProductID + "\n" + installationID + "\n" + activationID)
}
