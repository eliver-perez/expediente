package licensing

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"gestor-documental/internal/domain"
)

type OfflinePayload struct {
	Action    string `json:"action"`
	RequestID string `json:"request_id"`
	CreatedAt string `json:"created_at"`
	ProductID string `json:"product_id"`
	Binding
	LicenseID    string `json:"license_id,omitempty"`
	ActivationID string `json:"activation_id,omitempty"`
}
type OfflineEnvelope struct {
	SchemaVersion string `json:"schema_version"`
	Payload       string `json:"payload_b64u"`
	Signature     string `json:"signature_b64u"`
}
type Artifact struct {
	ID        string `json:"id"`
	Direction string `json:"direction"`
	Action    string `json:"action"`
	RequestID string `json:"request_id"`
	SHA256    string `json:"sha256"`
	CreatedAt string `json:"created_at"`
	Contents  string `json:"-"`
}

func saveArtifact(ctx context.Context, transaction *sql.Tx, principal domain.Principal, direction, action, requestID, contents string) (Artifact, error) {
	result := Artifact{ID: domain.NewID(), Direction: direction, Action: action, RequestID: requestID, SHA256: domain.Digest(contents), CreatedAt: domain.Timestamp(time.Now()), Contents: contents}
	err := transaction.QueryRowContext(ctx, "SELECT id,created_at FROM license_artifacts WHERE direction=? AND sha256=?", direction, result.SHA256).Scan(&result.ID, &result.CreatedAt)
	if err == nil {
		return result, nil
	}
	if err != sql.ErrNoRows {
		return result, err
	}
	_, err = transaction.ExecContext(ctx, "INSERT INTO license_artifacts VALUES(?,?,?,?,?,?,?,?)", result.ID, direction, action, requestID, contents, result.SHA256, nullableUser(principal.User.ID), result.CreatedAt)
	return result, err
}
func nullableUser(identifier string) any {
	if identifier == "" {
		return nil
	}
	return identifier
}
func (client *Client) OfflineRequest(ctx context.Context, action, requestID string, authorization Authorization, metadata domain.RequestMetadata) (Artifact, error) {
	result := Artifact{}
	if action != "activate" && action != "renew" && action != "deactivate" || !uuidPattern.MatchString(requestID) {
		return result, failure("INVALID_REQUEST", "Selecciona una acción y un identificador de solicitud válidos.")
	}
	if client.identityError != "" || len(client.privateKey) != ed25519.PrivateKeySize {
		return result, failure("LICENSE_IDENTITY_INVALID", "Recupera la identidad local antes de generar la solicitud.")
	}
	err := client.Database.Write(ctx, func(transaction *sql.Tx) error {
		principal, err := authorize(ctx, transaction, authorization)
		if err != nil {
			return err
		}
		err = transaction.QueryRowContext(ctx, "SELECT id,direction,action,request_id,sha256,created_at,contents FROM license_artifacts WHERE direction='request' AND request_id=?", requestID).Scan(&result.ID, &result.Direction, &result.Action, &result.RequestID, &result.SHA256, &result.CreatedAt, &result.Contents)
		if err == nil {
			if result.Action != action {
				return failure("IDEMPOTENCY_CONFLICT", "La solicitud ya se utilizó para otra acción.")
			}
			return nil
		}
		if err != sql.ErrNoRows {
			return err
		}
		state, err := readLocal(ctx, transaction)
		if err != nil {
			return err
		}
		payload := OfflinePayload{Action: action, RequestID: requestID, CreatedAt: client.Now().UTC().Format(time.RFC3339Nano), ProductID: ProductID, Binding: client.binding}
		if action != "activate" {
			claims, err := client.priorClaims(state.JWS)
			if err != nil {
				return failure("LICENSE_ACTIVATION_REQUIRED", "Se necesita una licencia reconocida para renovar o desactivar.")
			}
			payload.LicenseID = claims.LicenseID
			payload.ActivationID = claims.ActivationID
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		encoded := base64.RawURLEncoding.EncodeToString(bytes)
		envelope := OfflineEnvelope{SchemaVersion: "1.0", Payload: encoded, Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(client.privateKey, []byte("LICREQ-V1\n"+encoded)))}
		bytes, err = json.MarshalIndent(envelope, "", "  ")
		if err != nil {
			return err
		}
		result, err = saveArtifact(ctx, transaction, principal, "request", action, requestID, string(bytes)+"\n")
		if err != nil {
			return err
		}
		if action == "deactivate" {
			if _, err = transaction.ExecContext(ctx, "UPDATE license_state SET offline_deactivation_pending=1 WHERE singleton=1"); err != nil {
				return err
			}
		}
		return licenseEvent(ctx, transaction, principal, "license.offline_request_created", map[string]any{"action": action, "request_id": requestID, "artifact_id": result.ID}, metadata)
	})
	return result, err
}
func (client *Client) acceptLicense(ctx context.Context, transaction *sql.Tx, compact string, claims Claims, serverTime string, principal domain.Principal, requestID, action string, metadata domain.RequestMetadata) error {
	state, err := readLocal(ctx, transaction)
	if err != nil {
		return err
	}
	// Renewal cannot silently replace a still active installation/license binding.
	if state.JWS != "" && !state.Deactivated {
		previous, verifyErr := client.priorClaims(state.JWS)
		if verifyErr == nil && previous.Status != "revoked" && (claims.LicenseID != previous.LicenseID || claims.ActivationID != previous.ActivationID) {
			return failure("LICENSE_ACTIVATION_MISMATCH", "La renovación debe corresponder a la misma licencia y activación. Desactiva o utiliza la recuperación administrativa.")
		}
	}
	var revision int64
	var digest, licenseID string
	var deactivated bool
	err = transaction.QueryRowContext(ctx, "SELECT highest_revision,jws_sha256,license_id,deactivated FROM license_revision_floors WHERE activation_id=?", claims.ActivationID).Scan(&revision, &digest, &licenseID, &deactivated)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		if deactivated {
			return failure("ACTIVATION_DEACTIVATED", "Esta activación ya fue desactivada; solicita una nueva autorización.")
		}
		if licenseID != claims.LicenseID || claims.Revision < revision || claims.Revision == revision && digest != domain.Digest(compact) {
			return failure("LICENSE_REVISION_CONFLICT", "La licencia retrocede o contradice una revisión ya aceptada.")
		}
	}
	trusted, err := utcInstant(claims.IssuedAt)
	if err != nil {
		return err
	}
	if serverTime != "" {
		server, err := utcInstant(serverTime)
		if err != nil || trusted.After(server.Add(5*time.Minute)) {
			return contractFailure()
		}
		if server.After(trusted) {
			trusted = server
		}
	}
	if existing, err := utcInstant(state.Trusted); err == nil && existing.After(trusted) {
		trusted = existing
	}
	effective, _ := client.effectiveTime(state)
	if trusted.After(effective) {
		effective = trusted
	}
	if _, err = transaction.ExecContext(ctx, "INSERT INTO license_revision_floors VALUES(?,?,?,?,0) ON CONFLICT(activation_id) DO UPDATE SET highest_revision=excluded.highest_revision,jws_sha256=excluded.jws_sha256", claims.ActivationID, claims.LicenseID, claims.Revision, domain.Digest(compact)); err != nil {
		return err
	}
	if _, err = transaction.ExecContext(ctx, "UPDATE license_state SET current_jws=?,deactivated=0,last_trusted_at=?,last_observed_at=?,last_error_code='' WHERE singleton=1", compact, trusted.UTC().Format(time.RFC3339Nano), effective.UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	artifact, err := saveArtifact(ctx, transaction, principal, "response", action, requestID, compact)
	if err != nil {
		return err
	}
	if state.JWS == compact {
		return nil
	}
	return licenseEvent(ctx, transaction, principal, "license.accepted", map[string]any{"license_id": claims.LicenseID, "activation_id": claims.ActivationID, "license_revision": claims.Revision, "license_status": claims.Status, "artifact_id": artifact.ID, "source": action}, metadata)
}
func (client *Client) Import(ctx context.Context, contents []byte, authorization Authorization, metadata domain.RequestMetadata) (Status, error) {
	if len(contents) > MaximumArtifactBytes {
		return Status{}, failure("REQUEST_TOO_LARGE", "El archivo de licencia excede el límite de 64 KiB.")
	}
	compact := strings.TrimSpace(string(contents))
	claims, err := Verify(compact, client.Options.TrustedKeys, client.binding)
	if err != nil {
		return Status{}, err
	}
	if client.identityError != "" {
		return Status{}, failure("LICENSE_IDENTITY_INVALID", "Recupera primero la identidad de esta instalación.")
	}
	err = client.Database.Write(ctx, func(transaction *sql.Tx) error {
		principal, err := authorize(ctx, transaction, authorization)
		if err != nil {
			return err
		}
		return client.acceptLicense(ctx, transaction, compact, claims, "", principal, metadata.RequestID, "offline_import", metadata)
	})
	if err != nil {
		return Status{}, err
	}
	return client.Status(ctx)
}
func (client *Client) Artifacts(ctx context.Context) ([]Artifact, error) {
	result := []Artifact{}
	rows, err := client.Database.Reader.QueryContext(ctx, "SELECT id,direction,action,request_id,sha256,created_at FROM license_artifacts ORDER BY created_at DESC,id DESC LIMIT 100")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Artifact
		if err = rows.Scan(&item.ID, &item.Direction, &item.Action, &item.RequestID, &item.SHA256, &item.CreatedAt); err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
func (client *Client) Artifact(ctx context.Context, identifier string) (Artifact, error) {
	var item Artifact
	err := client.Database.Reader.QueryRowContext(ctx, "SELECT id,direction,action,request_id,sha256,created_at,contents FROM license_artifacts WHERE id=?", identifier).Scan(&item.ID, &item.Direction, &item.Action, &item.RequestID, &item.SHA256, &item.CreatedAt, &item.Contents)
	if err == sql.ErrNoRows {
		err = domain.Failure("NOT_FOUND", "No se encontró el archivo.", 404)
	}
	return item, err
}

// VerifyOfflineRequest is shared by the development contract simulator and tests.
// It verifies proof of possession, never grants a commercial activation.
func VerifyOfflineRequest(contents []byte) (OfflineEnvelope, OfflinePayload, error) {
	var envelope OfflineEnvelope
	var payload OfflinePayload
	object, err := strictObject(contents)
	if err != nil || len(object) != 3 || !fieldsPresent(object, "schema_version", "payload_b64u", "signature_b64u") || json.Unmarshal(contents, &envelope) != nil || envelope.SchemaVersion != "1.0" {
		return envelope, payload, contractFailure()
	}
	encoded, err := decodeBase64(envelope.Payload, 0)
	if err != nil {
		return envelope, payload, err
	}
	object, err = strictObject(encoded)
	if err != nil || !fieldsPresent(object, "action", "request_id", "created_at", "product_id", "installation_id", "installation_public_key", "fingerprint_version", "fingerprint_hash") || json.Unmarshal(encoded, &payload) != nil {
		return envelope, payload, contractFailure()
	}
	if payload.ProductID != ProductID || !uuidPattern.MatchString(payload.RequestID) || !installationPattern.MatchString(payload.InstallationID) || payload.FingerprintVersion != "1" || !fingerprintPattern.MatchString(payload.FingerprintHash) {
		return envelope, payload, contractFailure()
	}
	if _, err = utcInstant(payload.CreatedAt); err != nil {
		return envelope, payload, err
	}
	if payload.Action != "activate" && payload.Action != "renew" && payload.Action != "deactivate" || payload.Action != "activate" && (!uuidPattern.MatchString(payload.LicenseID) || !uuidPattern.MatchString(payload.ActivationID)) {
		return envelope, payload, contractFailure()
	}
	public, err := decodeBase64(payload.PublicKey, ed25519.PublicKeySize)
	if err != nil {
		return envelope, payload, err
	}
	signature, err := decodeBase64(envelope.Signature, ed25519.SignatureSize)
	if err != nil || !ed25519.Verify(public, []byte("LICREQ-V1\n"+envelope.Payload), signature) {
		return envelope, payload, failure("INVALID_PROOF", "La firma de la solicitud no es válida.")
	}
	return envelope, payload, nil
}
