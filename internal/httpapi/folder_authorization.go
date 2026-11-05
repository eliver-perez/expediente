package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/macosfolders"
	"gestor-documental/internal/windowsfolders"
)

// Tickets live only in the service process, expire quickly and contain no shell
// arguments. The HTTP interface never changes an ACL: only the installed,
// elevated local administrator utility can do that after a second confirmation.
type folderAuthorization struct {
	path, libraryID, userID, sessionID string
	source, tokenDigest                string
	expires                            time.Time
	claimed                            bool
	completed                          bool
	finishing                          bool
	resultCode                         string
	accessible                         bool
}

var localWindowsDirectory = regexp.MustCompile(`(?i)^[a-z]:\\[^\\].*`)
var authorizationToken = regexp.MustCompile(`^[a-f0-9]{64}$`)

func localFolderAuthorizationPath(path string) bool {
	if len(path) > 1024 || strings.ContainsAny(path, "\x00\r\n") || !localWindowsDirectory.MatchString(path) {
		return false
	}
	clean := filepath.Clean(path)
	return len(clean) > 3 && !strings.HasPrefix(clean, `\\`) && !strings.HasPrefix(clean, `//`)
}

func localOnly(r *http.Request) bool {
	// A reverse proxy on loopback must not turn a remote browser into a local
	// administrator request. Normal AIDEX desktop access sends none of these.
	if r.Header.Get("Forwarded") != "" || r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" {
		return false
	}
	peer := remoteAddress(r)
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	localHost := strings.EqualFold(host, "localhost")
	if address, err := netip.ParseAddr(host); err == nil {
		localHost = address.IsLoopback()
	}
	return peer.IsValid() && peer.IsLoopback() && localHost
}

func (server *Server) expireFolderTickets(now time.Time) {
	for key, ticket := range server.folderAuth {
		if !now.Before(ticket.expires) {
			delete(server.folderAuth, key)
		}
	}
}

func (server *Server) folderAuthorizationAudit(r *http.Request, ticket folderAuthorization, event, outcome string) error {
	hash := sha256.Sum256([]byte(strings.ToLower(ticket.path)))
	return server.identity.Database.Write(r.Context(), func(tx *sql.Tx) error {
		return audit.Append(r.Context(), tx, time.Now(), audit.Event{Type: event, ActorUserID: ticket.userID, SessionID: ticket.sessionID, LibraryID: ticket.libraryID, Metadata: metadata(r), Details: map[string]any{"path_sha256": hex.EncodeToString(hash[:]), "scope": ticket.source, "outcome": outcome}})
	})
}

func (server *Server) startFolderAuthorization(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if (runtime.GOOS != "windows" && runtime.GOOS != "darwin") || !localOnly(r) {
		server.fail(w, r, domain.Failure("LOCAL_AUTHORIZATION_REQUIRED", "Abre AIDEX en 127.0.0.1 desde el mismo equipo donde está instalado el servicio.", 403))
		return
	}
	if runtime.GOOS == "windows" && !p.Can("system.configure") {
		server.fail(w, r, domain.Failure("FORBIDDEN", "Se requiere administración de esta instalación.", 403))
		return
	}
	var input struct {
		LibraryID string `json:"library_id"`
		Path      string `json:"server_path"`
		Source    string `json:"storage_source"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	if runtime.GOOS == "windows" {
		if err := windowsfolders.ValidateLocal(input.Path, input.Source); err != nil {
			server.fail(w, r, domain.Failure("FOLDER_AUTHORIZATION_SCOPE", err.Error(), 422))
			return
		}
	}
	if err := server.libraries.ValidateFolderAuthorizationScope(r.Context(), p, input.LibraryID, input.Path, input.Source); err != nil {
		server.fail(w, r, err)
		return
	}
	if runtime.GOOS == "darwin" {
		if err := macosfolders.ValidateSelection(macosfolders.Selection{Path: input.Path, Source: input.Source}); err != nil {
			server.fail(w, r, domain.Failure("FOLDER_AUTHORIZATION_SCOPE", err.Error(), 422))
			return
		}
	}
	_, inspectionErr := server.libraries.PlanStorageRoot(r.Context(), p, input.LibraryID, input.Path, input.Source, metadata(r))
	if inspectionErr == nil && runtime.GOOS == "windows" {
		inspectionErr = libraries.CheckAuthorizedDirectory(input.Path, input.Source)
		if inspectionErr == nil {
			server.fail(w, r, domain.Failure("FOLDER_ALREADY_ACCESSIBLE", "El servicio ya puede acceder a la carpeta. Vuelve a inspeccionarla.", 409))
			return
		}
	}
	if inspectionErr != nil {
		var failure *domain.Error
		if !errors.As(inspectionErr, &failure) || (failure.Code != "ROOT_PERMISSION_DENIED" && !(runtime.GOOS == "darwin" && failure.Code == "ROOT_PRIVACY_DENIED")) {
			server.fail(w, r, inspectionErr)
			return
		}
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		server.fail(w, r, err)
		return
	}
	key := hex.EncodeToString(nonce[:])
	ticket := &folderAuthorization{source: input.Source, tokenDigest: p.TokenDigest, path: filepath.Clean(input.Path), libraryID: input.LibraryID, userID: p.User.ID, sessionID: p.SessionID, expires: time.Now().Add(5 * time.Minute)}
	if err := server.folderAuthorizationAudit(r, *ticket, "folder.authorization_requested", "pending"); err != nil {
		server.fail(w, r, err)
		return
	}
	server.folderAuthMu.Lock()
	server.expireFolderTickets(time.Now())
	if len(server.folderAuth) >= 64 {
		server.folderAuthMu.Unlock()
		server.fail(w, r, domain.Failure("FOLDER_AUTHORIZATION_BUSY", "Hay demasiadas solicitudes pendientes. Espera unos minutos y vuelve a intentarlo.", 429))
		return
	}
	server.folderAuth[key] = ticket
	server.folderAuthMu.Unlock()
	launchURL := "aidex-auth://authorize/" + key
	if runtime.GOOS == "darwin" {
		launchURL += fmt.Sprintf("?port=%s", serverLocalPort(server.configuration.ListenAddress))
	}
	writeJSON(w, 201, map[string]any{"launch_url": launchURL, "expires_in_seconds": 300})
}

func (server *Server) claimFolderAuthorization(w http.ResponseWriter, r *http.Request) {
	if (runtime.GOOS != "windows" && runtime.GOOS != "darwin") || !localOnly(r) || !authorizationToken.MatchString(r.PathValue("ticket")) {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud de autorización no disponible.", 404))
		return
	}
	server.folderAuthMu.Lock()
	server.expireFolderTickets(time.Now())
	ticket := server.folderAuth[r.PathValue("ticket")]
	if ticket == nil || ticket.claimed {
		server.folderAuthMu.Unlock()
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud de autorización vencida o utilizada.", 404))
		return
	}
	if err := server.validateFolderTicket(r, *ticket); err != nil {
		server.folderAuthMu.Unlock()
		server.fail(w, r, err)
		return
	}
	ticket.claimed = true
	source := ticket.source
	path := ticket.path
	server.folderAuthMu.Unlock()
	scope := source + "-access"
	if runtime.GOOS == "darwin" {
		scope = "linked-read-traverse"
	}
	writeJSON(w, 200, map[string]string{"server_path": path, "storage_source": source, "scope": scope})
}

func (server *Server) finishFolderAuthorization(w http.ResponseWriter, r *http.Request) {
	if (runtime.GOOS != "windows" && runtime.GOOS != "darwin") || !localOnly(r) || !authorizationToken.MatchString(r.PathValue("ticket")) {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud de autorización no disponible.", 404))
		return
	}
	server.folderAuthMu.Lock()
	ticket := server.folderAuth[r.PathValue("ticket")]
	if ticket == nil || !ticket.claimed || ticket.completed || ticket.finishing || !time.Now().Before(ticket.expires) {
		server.folderAuthMu.Unlock()
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud de autorización vencida o no utilizada.", 404))
		return
	}
	if runtime.GOOS == "darwin" {
		delete(server.folderAuth, r.PathValue("ticket"))
	} else {
		ticket.finishing = true
	}
	copy := *ticket
	server.folderAuthMu.Unlock()
	// This check runs in the service process (NT SERVICE\AIDEX), not in the
	// elevated helper. It is the only proof that the new ACL is effective.
	err := server.validateFolderTicket(r, copy)
	if err == nil {
		err = libraries.CheckAuthorizedDirectory(copy.path, copy.source)
	}
	outcome := "accessible"
	code := ""
	if err != nil {
		outcome = "unavailable"
		var failure *domain.Error
		if errors.As(err, &failure) {
			code = failure.Code
		} else {
			code = "ROOT_READ_FAILED"
		}
	}
	if auditErr := server.folderAuthorizationAudit(r, copy, "folder.authorization_checked", outcome); auditErr != nil {
		server.fail(w, r, auditErr)
		return
	}
	server.folderAuthMu.Lock()
	ticket.accessible = err == nil
	ticket.resultCode = code
	ticket.completed = true
	server.folderAuthMu.Unlock()
	writeJSON(w, 200, map[string]any{"accessible": err == nil, "code": code, "checked_scope": "root"})
}

func serverLocalPort(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return "18090"
	}
	return port
}

func (server *Server) validateFolderTicket(r *http.Request, ticket folderAuthorization) error {
	principal, err := server.identity.LoadPrincipal(r.Context(), server.identity.Database.Reader, ticket.tokenDigest)
	if err != nil {
		return err
	}
	if principal.User.ID != ticket.userID || principal.SessionID != ticket.sessionID {
		return domain.Failure("SESSION_REQUIRED", "La sesión que solicitó la autorización ya no está vigente.", 401)
	}
	if err = server.identity.License.Check(r.Context(), licensing.WriteDocuments); err != nil {
		return err
	}
	return server.libraries.Read(r.Context(), principal, ticket.libraryID, "storage.manage_roots")
}

func (server *Server) describeFolderAuthorization(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "darwin" || !localOnly(r) || !authorizationToken.MatchString(r.PathValue("ticket")) {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud no disponible.", 404))
		return
	}
	server.folderAuthMu.Lock()
	defer server.folderAuthMu.Unlock()
	server.expireFolderTickets(time.Now())
	ticket := server.folderAuth[r.PathValue("ticket")]
	if ticket == nil || ticket.claimed {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud vencida o utilizada.", 404))
		return
	}
	if err := server.validateFolderTicket(r, *ticket); err != nil {
		server.fail(w, r, err)
		return
	}
	writeJSON(w, 200, macosfolders.Selection{Path: ticket.path, Source: ticket.source})
}

func (server *Server) folderAuthorizationStatus(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	server.folderAuthMu.Lock()
	defer server.folderAuthMu.Unlock()
	ticket := server.folderAuth[r.PathValue("ticket")]
	if ticket == nil || !time.Now().Before(ticket.expires) || ticket.userID != p.User.ID || ticket.sessionID != p.SessionID {
		server.fail(w, r, domain.Failure("NOT_FOUND", "La autorización venció. Comprueba el acceso para continuar.", 404))
		return
	}
	if err := server.validateFolderTicket(r, *ticket); err != nil {
		server.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"done": ticket.completed, "accessible": ticket.accessible, "code": ticket.resultCode})
}

// Non-consuming probe lets the Windows helper leave ancestor ACLs untouched
// whenever granting access to the selected root is sufficient.
func (server *Server) probeFolderAuthorization(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" || !localOnly(r) || !authorizationToken.MatchString(r.PathValue("ticket")) {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud no disponible.", 404))
		return
	}
	server.folderAuthMu.Lock()
	ticket := server.folderAuth[r.PathValue("ticket")]
	if ticket == nil || !ticket.claimed || ticket.finishing || !time.Now().Before(ticket.expires) {
		server.folderAuthMu.Unlock()
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud vencida o utilizada.", 404))
		return
	}
	copy := *ticket
	server.folderAuthMu.Unlock()
	err := server.validateFolderTicket(r, copy)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	err = libraries.CheckAuthorizedDirectory(copy.path, copy.source)
	code := ""
	if err != nil {
		var failure *domain.Error
		if errors.As(err, &failure) {
			code = failure.Code
		} else {
			code = "ROOT_READ_FAILED"
		}
	}
	writeJSON(w, 200, map[string]any{"accessible": err == nil, "code": code})
}
