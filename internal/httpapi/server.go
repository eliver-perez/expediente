package httpapi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/licensing"
)

//go:embed all:assets
var compiledAssets embed.FS

type Server struct {
	libraries      *libraries.Service
	identity       *identity.Service
	configuration  config.Config
	logger         *slog.Logger
	publicHost     string
	secure         bool
	trustedProxies []netip.Prefix
}

func New(service *identity.Service, configuration config.Config, logger *slog.Logger) *Server {
	publicURL, _ := url.Parse(configuration.PublicURL)
	server := &Server{libraries: libraries.New(service), identity: service, configuration: configuration, logger: logger, publicHost: publicURL.Host, secure: publicURL.Scheme == "https"}
	for _, cidr := range configuration.TrustedProxies {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil {
			server.trustedProxies = append(server.trustedProxies, prefix)
		}
	}
	return server
}

type contextKey int

const metadataKey contextKey = 1

func metadata(request *http.Request) domain.RequestMetadata {
	return request.Context().Value(metadataKey).(domain.RequestMetadata)
}

func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	server.libraryRoutes(mux)
	server.licenseRoutes(mux)
	mux.HandleFunc("GET /health/live", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/auth/login", server.login)
	mux.HandleFunc("GET /api/v1/auth/session", server.protected("", false, server.session))
	mux.HandleFunc("POST /api/v1/auth/logout", server.protected("", true, server.logout))
	mux.HandleFunc("GET /api/v1/account", server.protected("", true, server.account))
	mux.HandleFunc("POST /api/v1/account/password", server.protected("", true, server.changePassword))
	mux.HandleFunc("GET /api/v1/account/sessions", server.protected("", true, server.sessions))
	mux.HandleFunc("GET /api/v1/users", server.protected("users.manage", true, server.users))
	mux.HandleFunc("POST /api/v1/users", server.protected("users.manage", true, server.createUser))
	mux.HandleFunc("PATCH /api/v1/users/{id}", server.protected("users.manage", true, server.updateUser))
	mux.HandleFunc("POST /api/v1/users/{id}/password-reset", server.protected("users.reset_password", true, server.resetPassword))
	mux.HandleFunc("POST /api/v1/users/{id}/revoke-sessions", server.protected("sessions.revoke", true, server.revokeSessions))
	mux.HandleFunc("GET /api/v1/roles", server.protected("permissions.manage_global", true, server.roles))
	mux.HandleFunc("PUT /api/v1/users/{id}/global-roles", server.protected("permissions.manage_global", true, server.setRoles))
	mux.HandleFunc("GET /api/v1/sessions", server.protected("sessions.read_all", true, server.sessions))
	mux.HandleFunc("GET /api/v1/authentication-attempts", server.protected("authentication_attempts.read", true, server.attempts))
	mux.HandleFunc("GET /api/v1/audit-events", server.protected("audit.read_global", true, server.events))
	mux.HandleFunc("GET /api/v1/audit-events/{id}", server.protected("audit.read_global", true, server.event))
	mux.HandleFunc("GET /api/v1/system/status", server.protected("system.configure", false, server.status))
	mux.HandleFunc("/api/", func(writer http.ResponseWriter, request *http.Request) {
		server.fail(writer, request, domain.Failure("NOT_FOUND", "No se encontró el endpoint.", 404))
	})
	mux.HandleFunc("/", server.frontend)
	return server.boundary(mux)
}

type authorizedHandler func(http.ResponseWriter, *http.Request, domain.Principal)

func (server *Server) protected(permission string, touch bool, handler authorizedHandler) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(server.cookieName())
		if err != nil {
			server.fail(writer, request, domain.Failure("SESSION_REQUIRED", "Inicia sesión para continuar.", 401))
			return
		}
		principal, err := server.identity.Authenticate(request.Context(), cookie.Value, touch, metadata(request))
		if err != nil {
			server.fail(writer, request, err)
			return
		}
		if permission != "" && !principal.Can(permission) {
			server.fail(writer, request, domain.Failure("FORBIDDEN", "No tienes permiso para esta operación.", 403))
			return
		}
		if request.Method != "GET" && request.Method != "HEAD" && !identity.VerifyCSRF(principal, request.Header.Get("X-CSRF-Token")) {
			server.fail(writer, request, domain.Failure("CSRF_INVALID", "La solicitud no pudo validarse. Recarga la página.", 403))
			return
		}
		if err := server.identity.License.Check(request.Context(), licensing.SecurityAdministration); err != nil {
			server.fail(writer, request, err)
			return
		}
		handler(writer, request, principal)
	}
}

func (server *Server) boundary(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := domain.NewID()
		request = request.WithContext(context.WithValue(request.Context(), metadataKey, domain.RequestMetadata{RequestID: requestID, ObservedIP: server.observedIP(request), UserAgent: request.UserAgent()}))
		writer.Header().Set("X-Request-ID", requestID)
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		writer.Header().Set("Cache-Control", "no-store")
		if server.secure {
			writer.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		defer func() {
			if recover() != nil {
				server.logger.Error("request panic", "request_id", requestID)
				server.fail(writer, request, fmt.Errorf("request panic"))
			}
		}()
		if request.Host != server.publicHost {
			server.fail(writer, request, domain.Failure("INVALID_HOST", "Host no permitido.", 400))
			return
		}
		if server.secure && request.TLS == nil && (!server.isTrusted(remoteAddress(request)) || request.Header.Get("X-Forwarded-Proto") != "https") {
			server.fail(writer, request, domain.Failure("HTTPS_REQUIRED", "Se requiere una conexión HTTPS.", 403))
			return
		}
		if request.Method != "GET" && request.Method != "HEAD" {
			if request.Header.Get("Origin") != server.configuration.PublicURL {
				server.fail(writer, request, domain.Failure("CSRF_INVALID", "Origen no permitido.", 403))
				return
			}
		}
		if request.Header.Get("Sec-Fetch-Site") == "cross-site" {
			server.fail(writer, request, domain.Failure("CSRF_INVALID", "Origen no permitido.", 403))
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func remoteAddress(request *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	address, _ := netip.ParseAddr(host)
	return address.Unmap()
}
func (server *Server) isTrusted(address netip.Addr) bool {
	for _, prefix := range server.trustedProxies {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
func (server *Server) observedIP(request *http.Request) string {
	address := remoteAddress(request)
	if !address.IsValid() {
		return "unknown"
	}
	if !server.isTrusted(address) {
		return address.String()
	}
	forwarded := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	if len(forwarded) > 16 {
		return address.String()
	}
	for index := len(forwarded) - 1; index >= 0; index-- {
		if !server.isTrusted(address) {
			break
		}
		candidate, err := netip.ParseAddr(strings.TrimSpace(forwarded[index]))
		if err != nil {
			return remoteAddress(request).String()
		}
		address = candidate.Unmap()
	}
	return address.String()
}

func (server *Server) cookieName() string {
	if server.secure {
		return "__Host-documental-session"
	}
	return "documental-local-session"
}
func (server *Server) clearCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{Name: server.cookieName(), Value: "", Path: "/", HttpOnly: true, Secure: server.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if status != 204 {
		_ = json.NewEncoder(writer).Encode(payload)
	}
}

func (server *Server) fail(writer http.ResponseWriter, request *http.Request, err error) {
	failure := &domain.Error{Code: "INTERNAL_ERROR", Message: "No se pudo completar la operación.", Status: 500}
	if !errors.As(err, &failure) {
		failure = &domain.Error{Code: "INTERNAL_ERROR", Message: "No se pudo completar la operación.", Status: 500}
		// Deliberately exclude raw SQL, request bodies, paths and secret-bearing errors.
		server.logger.Error("request failed", "request_id", metadata(request).RequestID, "error_type", fmt.Sprintf("%T", err))
	}
	// An older in-flight request must not delete a cookie issued by a newer login.
	// Logout/password changes explicitly clear it; stale sessions are already invalid in DB.
	if failure.Status == 429 {
		writer.Header().Set("Retry-After", "60")
	}
	writeJSON(writer, failure.Status, map[string]any{"error": map[string]any{"code": failure.Code, "message": failure.Message, "request_id": metadata(request).RequestID}})
}

func AssetsReady() bool { _, err := compiledAssets.ReadFile("assets/index.html"); return err == nil }
func (server *Server) frontend(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" && request.Method != "HEAD" {
		server.fail(writer, request, domain.Failure("NOT_FOUND", "No se encontró la página.", 404))
		return
	}
	for _, component := range strings.Split(request.URL.Path, "/") {
		if strings.HasPrefix(component, ".") {
			http.NotFound(writer, request)
			return
		}
	}
	assets, err := fs.Sub(compiledAssets, "assets")
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/assets/") {
		http.FileServer(http.FS(assets)).ServeHTTP(writer, request)
		return
	}
	switch request.URL.Path {
	case "/", "/login", "/account", "/libraries", "/search", "/admin/users", "/admin/access", "/admin/events", "/admin/license":
		contents, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			http.Error(writer, "Aplicación no disponible.", 503)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write(contents)
	default:
		http.NotFound(writer, request)
	}
}
