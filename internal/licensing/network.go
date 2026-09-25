package licensing

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gestor-documental/internal/domain"
)

func licenseTransport(options Options) (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if options.DevelopmentCAFile != "" {
		if !developmentEnabled {
			return nil, fmt.Errorf("development TLS CA forbidden")
		}
		contents, err := os.ReadFile(options.DevelopmentCAFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read development license CA")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(contents) {
			return nil, fmt.Errorf("invalid development license CA")
		}
		transport.TLSClientConfig.RootCAs = pool
	}
	return transport, nil
}

var remoteCodes = map[string]bool{"INVALID_REQUEST": true, "INVALID_PROOF": true, "LICENSE_NOT_FOUND": true, "ACTIVATION_LIMIT": true, "REVOKED": true, "INCOMPATIBLE_SCHEMA": true, "RATE_LIMITED": true, "TEMPORARY_UNAVAILABLE": true}

func remoteFailure(code string) error {
	messages := map[string]string{"INVALID_REQUEST": "El servidor rechazó la solicitud. Genera una nueva solicitud.", "INVALID_PROOF": "El servidor no pudo verificar la identidad de la instalación.", "LICENSE_NOT_FOUND": "No se encontró la licencia comercial.", "ACTIVATION_LIMIT": "La licencia ya tiene una instalación activa. Gestiona su transferencia.", "REVOKED": "El servidor informó una revocación. Renueva para obtener el estado firmado.", "INCOMPATIBLE_SCHEMA": "El servidor utiliza una versión de contrato incompatible.", "RATE_LIMITED": "El servidor limitó temporalmente las solicitudes. Reintenta más tarde.", "TEMPORARY_UNAVAILABLE": "No fue posible contactar de forma segura al servidor de licencias. Puedes reintentar la misma solicitud."}
	if !remoteCodes[code] {
		code = "TEMPORARY_UNAVAILABLE"
	}
	return domain.Failure(code, messages[code], 502)
}
func (client *Client) post(ctx context.Context, path string, body any, result any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.Options.ServerURL, "/")+path, bytes.NewReader(encoded))
	if err != nil {
		return remoteFailure("TEMPORARY_UNAVAILABLE")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return remoteFailure("TEMPORARY_UNAVAILABLE")
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(io.LimitReader(response.Body, MaximumArtifactBytes+1))
	if err != nil {
		return remoteFailure("TEMPORARY_UNAVAILABLE")
	}
	if _, err = strictObject(contents); err != nil {
		return failure("LICENSE_SERVER_RESPONSE", "El servidor devolvió una respuesta de licencia no válida.")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failureEnvelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(contents, &failureEnvelope) != nil {
			return remoteFailure("TEMPORARY_UNAVAILABLE")
		}
		return remoteFailure(failureEnvelope.Error.Code)
	}
	if json.Unmarshal(contents, result) != nil {
		return failure("LICENSE_SERVER_RESPONSE", "El servidor devolvió una respuesta de licencia no válida.")
	}
	return nil
}

type onlineRequest struct {
	RequestID          string `json:"request_id"`
	ProductID          string `json:"product_id"`
	LicenseKey         string `json:"license_key,omitempty"`
	InstallationID     string `json:"installation_id"`
	PublicKey          string `json:"installation_public_key,omitempty"`
	FingerprintVersion string `json:"fingerprint_version,omitempty"`
	FingerprintHash    string `json:"fingerprint_hash,omitempty"`
	ActivationID       string `json:"activation_id,omitempty"`
	ChallengeID        string `json:"challenge_id"`
	Proof              string `json:"proof"`
	AppVersion         string `json:"app_version"`
}

func (client *Client) requestDigest(action, key string) string {
	mac := hmac.New(sha256.New, client.privateKey.Seed())
	mac.Write([]byte(action + "\n" + client.binding.InstallationID + "\n" + key))
	return hex.EncodeToString(mac.Sum(nil))
}

// The journal stores the exact signed request, excluding the commercial key.
// Reentering the same key recreates identical activation bytes after a restart.
func (client *Client) Online(ctx context.Context, action, requestID, commercialKey string, authorization Authorization, metadata domain.RequestMetadata) (Status, error) {
	if action != "activate" && action != "refresh" && action != "deactivate" || !uuidPattern.MatchString(requestID) {
		return Status{}, failure("INVALID_REQUEST", "Acción o identificador de solicitud no válido.")
	}
	if client.Options.ServerURL == "" {
		return Status{}, failure("LICENSE_SERVER_NOT_CONFIGURED", "Configura el servidor HTTPS de licencias antes de continuar.")
	}
	if client.identityError != "" || len(client.privateKey) != ed25519.PrivateKeySize {
		return Status{}, failure("LICENSE_IDENTITY_INVALID", "Recupera primero la identidad de esta instalación.")
	}
	if action == "activate" && (len(commercialKey) < 1 || len(commercialKey) > 1024) || action != "activate" && commercialKey != "" {
		return Status{}, failure("INVALID_REQUEST", "La clave comercial solo se utiliza al activar.")
	}
	client.operationMutex.Lock()
	defer client.operationMutex.Unlock()
	digest := client.requestDigest(action, commercialKey)
	var request onlineRequest
	var operationStatus, storedDigest, storedAction, storedJSON, storedError string
	err := client.Database.Write(ctx, func(tx *sql.Tx) error {
		if _, err := authorize(ctx, tx, authorization); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx, "SELECT action,request_digest,request_json,status,error_code FROM license_operations WHERE request_id=?", requestID).Scan(&storedAction, &storedDigest, &storedJSON, &operationStatus, &storedError)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		if storedAction != action || !hmac.Equal([]byte(storedDigest), []byte(digest)) {
			return failure("IDEMPOTENCY_CONFLICT", "La solicitud ya corresponde a otra acción o clave.")
		}
		if json.Unmarshal([]byte(storedJSON), &request) != nil {
			return contractFailure()
		}
		return nil
	})
	if err != nil {
		return Status{}, err
	}
	if operationStatus == "succeeded" {
		return client.Status(ctx)
	}
	if operationStatus == "failed" {
		return Status{}, remoteFailure(storedError)
	}
	if operationStatus == "" {
		state, err := readLocal(ctx, client.Database.Reader)
		if err != nil {
			return Status{}, err
		}
		var activationID *string
		if action != "activate" {
			claims, err := client.priorClaims(state.JWS)
			if err != nil || state.Deactivated {
				return Status{}, failure("LICENSE_ACTIVATION_REQUIRED", "Se requiere una activación reconocida para esta acción.")
			}
			activationID = &claims.ActivationID
		} else if state.JWS != "" && !state.Deactivated {
			claims, err := client.priorClaims(state.JWS)
			if err == nil && claims.Status != "revoked" {
				return Status{}, failure("LICENSE_ALREADY_ACTIVE", "La instalación ya está activada. Utiliza Renovar.")
			}
		}
		challengeBody := struct {
			Action         string  `json:"action"`
			ProductID      string  `json:"product_id"`
			InstallationID string  `json:"installation_id"`
			ActivationID   *string `json:"activation_id"`
		}{action, ProductID, client.binding.InstallationID, activationID}
		var challenge struct {
			ID      string `json:"challenge_id"`
			Nonce   string `json:"nonce"`
			Expires string `json:"expires_at"`
		}
		if err = client.post(ctx, "/v1/activations/challenge", challengeBody, &challenge); err != nil {
			client.recordContact(ctx, err)
			return Status{}, err
		}
		expiry, err := utcInstant(challenge.Expires)
		if err != nil || challenge.ID == "" || len(challenge.ID) > 256 || len(challenge.Nonce) < 16 || len(challenge.Nonce) > 1024 || strings.ContainsAny(challenge.ID+challenge.Nonce, "\r\n") || !expiry.After(client.Now().UTC()) || expiry.After(client.Now().UTC().Add(15*time.Minute)) {
			return Status{}, failure("LICENSE_CHALLENGE_INVALID", "El desafío del servidor no es válido o el reloj requiere corrección.")
		}
		request = onlineRequest{RequestID: requestID, ProductID: ProductID, InstallationID: client.binding.InstallationID, ChallengeID: challenge.ID, AppVersion: AppVersion}
		if activationID != nil {
			request.ActivationID = *activationID
		} else {
			request.PublicKey = client.binding.PublicKey
			request.FingerprintVersion = client.binding.FingerprintVersion
			request.FingerprintHash = client.binding.FingerprintHash
		}
		request.Proof = base64.RawURLEncoding.EncodeToString(ed25519.Sign(client.privateKey, ProofMessage(action, challenge.ID, challenge.Nonce, client.binding.InstallationID, request.ActivationID)))
		encoded, _ := json.Marshal(request)
		err = client.Database.Write(ctx, func(tx *sql.Tx) error {
			principal, err := authorize(ctx, tx, authorization)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO license_operations(request_id,action,request_digest,request_json,status,created_at,updated_at) VALUES(?,?,?,?,'prepared',?,?)", requestID, action, digest, string(encoded), domain.Timestamp(client.Now()), domain.Timestamp(client.Now())); err != nil {
				return err
			}
			return licenseEvent(ctx, tx, principal, "license.online_requested", map[string]any{"action": action, "request_id": requestID}, metadata)
		})
		if err != nil {
			return Status{}, err
		}
	}
	request.LicenseKey = commercialKey
	path := "/v1/activations"
	if action != "activate" {
		path += "/" + action
	}
	var response struct {
		JWS           string `json:"license_jws"`
		ServerTime    string `json:"server_time"`
		RequestID     string `json:"request_id"`
		ActivationID  string `json:"activation_id"`
		DeactivatedAt string `json:"deactivated_at"`
		Status        string `json:"status"`
	}
	err = client.post(ctx, path, request, &response)
	if err == nil {
		var claims Claims
		if action == "deactivate" {
			if response.ActivationID != request.ActivationID || response.Status != "deactivated" {
				err = contractFailure()
			}
			if _, timeErr := utcInstant(response.DeactivatedAt); timeErr != nil {
				err = contractFailure()
			}
		} else {
			if response.RequestID != requestID {
				err = contractFailure()
			} else if _, timeErr := utcInstant(response.ServerTime); timeErr != nil {
				err = contractFailure()
			} else {
				claims, err = Verify(response.JWS, client.Options.TrustedKeys, client.binding)
			}
			if err == nil && action == "refresh" && claims.ActivationID != request.ActivationID {
				err = contractFailure()
			}
		}
		if err == nil {
			err = client.Database.Write(ctx, func(tx *sql.Tx) error {
				principal, err := authorize(ctx, tx, authorization)
				if err != nil {
					return err
				}
				if action == "deactivate" {
					state, err := readLocal(ctx, tx)
					if err != nil {
						return err
					}
					previous, err := client.priorClaims(state.JWS)
					if err != nil || previous.ActivationID != request.ActivationID {
						return failure("LICENSE_ACTIVATION_MISMATCH", "La activación cambió mientras se procesaba la solicitud.")
					}
					if _, err = tx.ExecContext(ctx, "UPDATE license_revision_floors SET deactivated=1 WHERE activation_id=?", request.ActivationID); err != nil {
						return err
					}
					if _, err = tx.ExecContext(ctx, "UPDATE license_state SET deactivated=1,offline_deactivation_pending=0 WHERE singleton=1"); err != nil {
						return err
					}
					evidence, _ := json.Marshal(map[string]string{"activation_id": response.ActivationID, "deactivated_at": response.DeactivatedAt, "status": response.Status})
					if _, err = saveArtifact(ctx, tx, principal, "response", "deactivate", requestID, string(evidence)); err != nil {
						return err
					}
					if err = licenseEvent(ctx, tx, principal, "license.deactivated", map[string]any{"activation_id": request.ActivationID, "request_id": requestID}, metadata); err != nil {
						return err
					}
				} else if err = client.acceptLicense(ctx, tx, response.JWS, claims, response.ServerTime, principal, requestID, action, metadata); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, "UPDATE license_operations SET status='succeeded',error_code='',updated_at=? WHERE request_id=?", domain.Timestamp(client.Now()), requestID); err != nil {
					return err
				}
				_, err = tx.ExecContext(ctx, "UPDATE license_state SET last_contact_at=?,last_error_code='' WHERE singleton=1", domain.Timestamp(client.Now()))
				return err
			})
		}
	}
	if err != nil {
		client.recordContact(ctx, err)
		code := errorCode(err)
		if remoteCodes[code] && code != "TEMPORARY_UNAVAILABLE" && code != "RATE_LIMITED" {
			_ = client.Database.Write(ctx, func(tx *sql.Tx) error {
				_, dbErr := tx.ExecContext(ctx, "UPDATE license_operations SET status='failed',error_code=?,updated_at=? WHERE request_id=?", code, domain.Timestamp(client.Now()), requestID)
				return dbErr
			})
		}
		return Status{}, err
	}
	return client.Status(ctx)
}
func (client *Client) recordContact(ctx context.Context, cause error) {
	code := errorCode(cause)
	_ = client.Database.Write(ctx, func(tx *sql.Tx) error {
		state, err := readLocal(ctx, tx)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE license_state SET last_error_code=? WHERE singleton=1", code); err != nil {
			return err
		}
		if state.LastError == code {
			return nil
		}
		return licenseEvent(ctx, tx, domain.Principal{}, "license.contact_failed", map[string]any{"error_code": code}, domain.RequestMetadata{})
	})
}

// Run maintains the clock high-water mark even without a server. A failed contact
// never changes the signed state. All refresh retries reuse a persisted request ID.
func (client *Client) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	nextAttempt := time.Time{}
	for {
		client.ObserveClock(ctx)
		if client.Options.ServerURL != "" && client.Options.RefreshHours > 0 && !client.Now().Before(nextAttempt) {
			state, err := readLocal(ctx, client.Database.Reader)
			if err == nil && state.JWS != "" && !state.Deactivated {
				last, _ := utcInstant(state.LastContact)
				if last.IsZero() || client.Now().Sub(last) >= time.Duration(client.Options.RefreshHours)*time.Hour {
					var requestID string
					claims, _ := client.priorClaims(state.JWS)
					_ = client.Database.Reader.QueryRowContext(ctx, "SELECT request_id FROM license_operations WHERE action='refresh' AND status='prepared' AND json_extract(request_json,'$.activation_id')=? AND json_extract(request_json,'$.installation_id')=? ORDER BY created_at LIMIT 1", claims.ActivationID, client.binding.InstallationID).Scan(&requestID)
					if requestID == "" {
						requestID = domain.NewID()
					}
					_, _ = client.Online(ctx, "refresh", requestID, "", nil, domain.RequestMetadata{RequestID: requestID})
					nextAttempt = client.Now().Add(15 * time.Minute)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (client *Client) ObserveClock(ctx context.Context) {
	_ = client.Database.Write(ctx, func(tx *sql.Tx) error {
		state, err := readLocal(ctx, tx)
		if err != nil {
			return err
		}
		effective, warning := client.effectiveTime(state)
		if _, err = tx.ExecContext(ctx, "UPDATE license_state SET last_observed_at=?,clock_warning=? WHERE singleton=1", effective.UTC().Format(time.RFC3339Nano), warning); err != nil {
			return err
		}
		if warning && !state.ClockWarning {
			return licenseEvent(ctx, tx, domain.Principal{}, "license.clock_warning", map[string]any{}, domain.RequestMetadata{})
		}
		return nil
	})
}

// PendingOperations exposes only resumable identifiers to an authorized admin.
// Proofs, request bodies and the commercial-key comparison digest stay private.
type PendingOperation struct {
	RequestID string `json:"request_id"`
	Action    string `json:"action"`
	CreatedAt string `json:"created_at"`
}

func (client *Client) PendingOperations(ctx context.Context) ([]PendingOperation, error) {
	result := []PendingOperation{}
	rows, err := client.Database.Reader.QueryContext(ctx, "SELECT request_id,action,created_at FROM license_operations WHERE status='prepared' AND json_extract(request_json,'$.installation_id')=? ORDER BY created_at DESC,request_id LIMIT 100", client.binding.InstallationID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item PendingOperation
		if err = rows.Scan(&item.RequestID, &item.Action, &item.CreatedAt); err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
