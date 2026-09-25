package httpapi

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

func (server *Server) licenseRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/license", server.protected("license.manage", true, server.licenseStatus))
	for _, action := range []string{"activate", "refresh", "deactivate"} {
		mux.HandleFunc("POST /api/v1/license/"+action, server.protected("license.manage", true, server.licenseOnline(action)))
	}
	mux.HandleFunc("POST /api/v1/license/offline-requests", server.protected("license.manage", true, server.licenseOffline))
	mux.HandleFunc("POST /api/v1/license/import", server.protected("license.manage", true, server.licenseImport))
	mux.HandleFunc("GET /api/v1/license/artifacts", server.protected("license.manage", true, server.licenseArtifacts))
	mux.HandleFunc("GET /api/v1/license/artifacts/{id}", server.protected("license.manage", true, server.licenseArtifact))
}
func (server *Server) licenseAuthorization(principal domain.Principal) licensing.Authorization {
	return func(ctx context.Context, transaction *sql.Tx) (domain.Principal, error) {
		current, err := server.identity.LoadPrincipal(ctx, transaction, principal.TokenDigest)
		if err != nil {
			return current, err
		}
		if !current.Can("license.manage") {
			return current, domain.Failure("FORBIDDEN", "No tienes permiso para administrar la licencia.", 403)
		}
		return current, nil
	}
}
func (server *Server) licenseStatus(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.identity.License.Status(request.Context())
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	pending, err := server.identity.License.PendingOperations(request.Context())
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 200, struct {
		licensing.Status
		Pending []licensing.PendingOperation `json:"pending_operations"`
	}{result, pending})
}
func (server *Server) licenseOnline(action string) authorizedHandler {
	return func(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
		var input struct {
			RequestID  string `json:"request_id"`
			LicenseKey string `json:"license_key"`
			Confirm    bool   `json:"confirm"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			server.fail(writer, request, err)
			return
		}
		if action == "deactivate" && !input.Confirm {
			server.fail(writer, request, domain.Failure("CONFIRMATION_REQUIRED", "Confirma la desactivación de esta instalación.", 422))
			return
		}
		result, err := server.identity.License.Online(request.Context(), action, input.RequestID, input.LicenseKey, server.licenseAuthorization(principal), metadata(request))
		if err != nil {
			server.fail(writer, request, err)
			return
		}
		writeJSON(writer, 200, result)
	}
}
func (server *Server) licenseOffline(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Action    string `json:"action"`
		RequestID string `json:"request_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	result, err := server.identity.License.OfflineRequest(request.Context(), input.Action, input.RequestID, server.licenseAuthorization(principal), metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 201, result)
}
func (server *Server) licenseImport(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	request.Body = http.MaxBytesReader(writer, request.Body, licensing.MaximumArtifactBytes+8192)
	reader, err := request.MultipartReader()
	if err != nil {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Selecciona un archivo .lic.", 400))
		return
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Selecciona un único archivo .lic.", 400))
		return
	}
	contents, err := io.ReadAll(io.LimitReader(part, licensing.MaximumArtifactBytes+1))
	part.Close()
	if err != nil || len(contents) > licensing.MaximumArtifactBytes {
		server.fail(writer, request, domain.Failure("REQUEST_TOO_LARGE", "El archivo no debe exceder 64 KiB.", 413))
		return
	}
	if _, err = reader.NextPart(); err != io.EOF {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Se admite un único archivo por importación.", 400))
		return
	}
	result, err := server.identity.License.Import(request.Context(), contents, server.licenseAuthorization(principal), metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 200, result)
}
func (server *Server) licenseArtifacts(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.identity.License.Artifacts(request.Context())
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 200, map[string]any{"items": result})
}
func (server *Server) licenseArtifact(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.identity.License.Artifact(request.Context(), request.PathValue("id"))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	extension := "lic"
	if result.Direction == "request" {
		extension = "licreq"
	} else if result.Action == "deactivate" {
		extension = "json"
	}
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="AIBID-%s.%s"`, result.ID, extension))
	writer.WriteHeader(200)
	_, _ = io.WriteString(writer, result.Contents)
}
