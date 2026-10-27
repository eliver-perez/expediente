package httpapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"net/http"
	"runtime"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

//go:embed eula-draft.txt
var eulaText string

// This identifier differs from the phase-4 placeholder, so accepting that
// placeholder never grants acceptance of the supplied 18-clause contract.
const eulaVersion = "1.0-borrador-2026-10-22"

func eulaHash() string { sum := sha256.Sum256([]byte(eulaText)); return hex.EncodeToString(sum[:]) }

func (server *Server) about(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	status, err := server.identity.License.Status(r.Context())
	if err != nil {
		server.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"product": "AIDEX", "full_name": "Aplicación de Indexación Digital de Expedientes", "version": buildinfo.DisplayVersion(), "os": runtime.GOOS, "installation_id": status.InstallationID})
}

func (server *Server) eulaAccepted(ctx context.Context) (bool, error) {
	if licensing.DevelopmentEnabled() {
		return true, nil
	}
	status, err := server.identity.License.Status(ctx)
	if err != nil {
		return false, err
	}
	if status.Development {
		return true, nil
	}
	if status.InstallationID == "" {
		return false, nil
	}
	var count int
	err = server.identity.Database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM eula_acceptances WHERE installation_id=? AND version=? AND text_sha256=?", status.InstallationID, eulaVersion, eulaHash()).Scan(&count)
	return count > 0, err
}

func (server *Server) eulaGet(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	accepted, err := server.eulaAccepted(r.Context())
	if err != nil {
		server.fail(w, r, err)
		return
	}
	status, err := server.identity.License.Status(r.Context())
	if err != nil {
		server.fail(w, r, err)
		return
	}
	var acceptedVersion, acceptedAt string
	if status.InstallationID != "" {
		err = server.identity.Database.Reader.QueryRowContext(r.Context(), "SELECT version,accepted_at FROM eula_acceptances WHERE installation_id=? ORDER BY accepted_at DESC,id DESC LIMIT 1", status.InstallationID).Scan(&acceptedVersion, &acceptedAt)
		if err != nil && err != sql.ErrNoRows {
			server.fail(w, r, err)
			return
		}
	}
	writeJSON(w, 200, map[string]any{"version": eulaVersion, "sha256": eulaHash(), "text": eulaText, "draft": true, "accepted": accepted, "accepted_version": acceptedVersion, "accepted_at": acceptedAt, "installation_id": status.InstallationID})
}

func (server *Server) eulaAccept(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	var input struct {
		Version string `json:"version"`
		SHA256  string `json:"sha256"`
		Accept  bool   `json:"accept"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	if !input.Accept || input.Version != eulaVersion || input.SHA256 != eulaHash() {
		server.fail(w, r, domain.Failure("EULA_CONFIRMATION_REQUIRED", "Lee y acepta expresamente la versión actual del contrato.", 422))
		return
	}
	status, err := server.identity.License.Status(r.Context())
	if err != nil {
		server.fail(w, r, err)
		return
	}
	if status.InstallationID == "" {
		server.fail(w, r, domain.Failure("LICENSE_IDENTITY_INVALID", "Se requiere una identidad de instalación válida.", 422))
		return
	}
	err = server.identity.Database.Write(r.Context(), func(tx *sql.Tx) error {
		current, err := server.identity.LoadPrincipal(r.Context(), tx, p.TokenDigest)
		if err != nil {
			return err
		}
		if !current.Can("license.manage") {
			return domain.Failure("FORBIDDEN", "Solo un administrador puede aceptar el contrato.", 403)
		}
		var count int
		if err = tx.QueryRowContext(r.Context(), "SELECT count(*) FROM eula_acceptances WHERE installation_id=? AND version=? AND text_sha256=?", status.InstallationID, eulaVersion, eulaHash()).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		id := domain.NewID()
		if _, err = tx.ExecContext(r.Context(), "INSERT INTO eula_acceptances VALUES(?,?,?,?,?,?)", id, status.InstallationID, current.User.ID, eulaVersion, eulaHash(), domain.Timestamp(time.Now())); err != nil {
			return err
		}
		return audit.Append(r.Context(), tx, time.Now(), audit.Event{Type: "eula.accepted", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata(r), Details: map[string]any{"version": eulaVersion, "sha256": eulaHash(), "installation_id": status.InstallationID}})
	})
	if err != nil {
		server.fail(w, r, err)
		return
	}
	server.eulaGet(w, r, p)
}
