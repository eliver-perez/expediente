package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
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
)

// Tickets live only in the service process, expire quickly and contain no shell
// arguments. The HTTP interface never changes an ACL: only the installed,
// elevated local administrator utility can do that after a second confirmation.
type folderAuthorization struct {
	path, libraryID, userID, sessionID string
	expires                            time.Time
	claimed                            bool
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
		return audit.Append(r.Context(), tx, time.Now(), audit.Event{Type: event, ActorUserID: ticket.userID, SessionID: ticket.sessionID, LibraryID: ticket.libraryID, Metadata: metadata(r), Details: map[string]any{"path_sha256": hex.EncodeToString(hash[:]), "scope": "linked-read-traverse", "outcome": outcome}})
	})
}

func (server *Server) startFolderAuthorization(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if runtime.GOOS != "windows" || !localOnly(r) {
		server.fail(w, r, domain.Failure("LOCAL_AUTHORIZATION_REQUIRED", "Abre AIDEX en 127.0.0.1 desde el equipo Windows donde está instalado el servicio.", 403))
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
	if input.Source != "linked" || !localFolderAuthorizationPath(input.Path) {
		server.fail(w, r, domain.Failure("FOLDER_AUTHORIZATION_SCOPE", "La autorización automática solo admite una carpeta local vinculada. Las rutas UNC y destinos administrados requieren revisión de sus permisos en el equipo correspondiente.", 422))
		return
	}
	if _, err := server.libraries.PlanStorageRoot(r.Context(), p, input.LibraryID, input.Path, input.Source, metadata(r)); err == nil {
		server.fail(w, r, domain.Failure("FOLDER_ALREADY_ACCESSIBLE", "El servicio ya puede leer la carpeta. Vuelve a inspeccionarla.", 409))
		return
	} else {
		var failure *domain.Error
		if !errors.As(err, &failure) || failure.Code != "ROOT_PERMISSION_DENIED" {
			server.fail(w, r, err)
			return
		}
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		server.fail(w, r, err)
		return
	}
	key := hex.EncodeToString(nonce[:])
	ticket := &folderAuthorization{path: filepath.Clean(input.Path), libraryID: input.LibraryID, userID: p.User.ID, sessionID: p.SessionID, expires: time.Now().Add(5 * time.Minute)}
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
	writeJSON(w, 201, map[string]any{"launch_url": "aidex-auth://authorize/" + key, "expires_in_seconds": 300})
}

func (server *Server) claimFolderAuthorization(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" || !localOnly(r) || !authorizationToken.MatchString(r.PathValue("ticket")) {
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
	ticket.claimed = true
	path := ticket.path
	server.folderAuthMu.Unlock()
	writeJSON(w, 200, map[string]string{"server_path": path, "scope": "linked-read-traverse"})
}

func (server *Server) finishFolderAuthorization(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" || !localOnly(r) || !authorizationToken.MatchString(r.PathValue("ticket")) {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud de autorización no disponible.", 404))
		return
	}
	server.folderAuthMu.Lock()
	ticket := server.folderAuth[r.PathValue("ticket")]
	if ticket == nil || !ticket.claimed || !time.Now().Before(ticket.expires) {
		server.folderAuthMu.Unlock()
		server.fail(w, r, domain.Failure("NOT_FOUND", "Solicitud de autorización vencida o no utilizada.", 404))
		return
	}
	delete(server.folderAuth, r.PathValue("ticket"))
	copy := *ticket
	server.folderAuthMu.Unlock()
	// This check runs in the service process (NT SERVICE\AIDEX), not in the
	// elevated helper. It is the only proof that the new ACL is effective.
	err := libraries.CheckLinkedDirectory(copy.path)
	outcome := "accessible"
	if err != nil {
		outcome = "unavailable"
	}
	if auditErr := server.folderAuthorizationAudit(r, copy, "folder.authorization_checked", outcome); auditErr != nil {
		server.fail(w, r, auditErr)
		return
	}
	writeJSON(w, 200, map[string]any{"accessible": err == nil})
}
