package httpapi

import (
	"net/http"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/licensing"
)

func (server *Server) login(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	result, err := server.identity.Login(request.Context(), input.Username, input.Password, metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	expires, _ := time.Parse(domain.TimeLayout, result.ExpiresAt)
	http.SetCookie(writer, &http.Cookie{Name: server.cookieName(), Value: result.SessionToken, Path: "/", HttpOnly: true, Secure: server.secure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: server.configuration.SessionAbsoluteHours * 3600})
	writeJSON(writer, 200, result)
}

func (server *Server) session(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	cookie, _ := request.Cookie(server.cookieName())
	writeJSON(writer, 200, map[string]any{"user": principal.User, "session_expires_at": principal.ExpiresAt, "csrf_token": identity.CSRFToken(cookie.Value)})
}
func (server *Server) account(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	userJSON(writer, 200, principal.User)
}
func (server *Server) logout(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	if err := server.identity.Logout(request.Context(), principal, metadata(request)); err != nil {
		server.fail(writer, request, err)
		return
	}
	server.clearCookie(writer)
	writer.WriteHeader(204)
}
func (server *Server) changePassword(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	if err := server.identity.ChangePassword(request.Context(), principal, input.CurrentPassword, input.NewPassword, metadata(request)); err != nil {
		server.fail(writer, request, err)
		return
	}
	server.clearCookie(writer)
	writer.WriteHeader(204)
}
func (server *Server) createUser(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	user, err := server.identity.CreateUser(request.Context(), principal, input.Username, input.DisplayName, input.Password, metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writer.Header().Set("Location", "/api/v1/users/"+user.ID)
	userJSON(writer, 201, user)
}
func (server *Server) updateUser(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		DisplayName *string `json:"display_name"`
		Disabled    *bool   `json:"disabled"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	if input.DisplayName == nil && input.Disabled == nil {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "No hay cambios.", 400))
		return
	}
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	user, err := server.identity.UpdateUser(request.Context(), principal, request.PathValue("id"), revision, input.DisplayName, input.Disabled, metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	userJSON(writer, 200, user)
}
func (server *Server) resetPassword(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	if err := server.identity.ResetPassword(request.Context(), principal, request.PathValue("id"), input.NewPassword, metadata(request)); err != nil {
		server.fail(writer, request, err)
		return
	}
	writer.WriteHeader(204)
}
func (server *Server) revokeSessions(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	if err := server.identity.RevokeSessions(request.Context(), principal, request.PathValue("id"), input.Reason, metadata(request)); err != nil {
		server.fail(writer, request, err)
		return
	}
	writer.WriteHeader(204)
}
func (server *Server) setRoles(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		RoleIDs []string `json:"role_ids"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	user, err := server.identity.SetGlobalRoles(request.Context(), principal, request.PathValue("id"), revision, input.RoleIDs, metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	userJSON(writer, 200, user)
}
func (server *Server) status(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var version string
	if err := server.identity.Database.Reader.QueryRowContext(request.Context(), "SELECT sqlite_version()").Scan(&version); err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 200, map[string]any{"stage": "H4", "product_name": "AIBID", "extraction_tools": server.configuration.Indexing.Diagnostics(), "sqlite_version": version, "development_license": licensing.DevelopmentEnabled(), "license_state": "unactivated"})
}
