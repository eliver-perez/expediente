package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gestor-documental/internal/domain"
)

type folderSelection struct {
	authorization folderAuthorization
	claimed, done bool
	path, code    string
}

func (server *Server) startFolderSelection(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	var input struct {
		LibraryID string `json:"library_id"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	if err := server.libraries.Read(r.Context(), p, input.LibraryID, "storage.manage_roots"); err != nil {
		server.fail(w, r, err)
		return
	}
	if err := server.libraries.Read(r.Context(), p, input.LibraryID, "storage.view_paths"); err != nil {
		server.fail(w, r, err)
		return
	}
	// Never launch a client-side native picker for a remote browser.
	if !localOnly(r) {
		writeJSON(w, 200, map[string]any{"mode": "server"})
		return
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		server.fail(w, r, err)
		return
	}
	key := hex.EncodeToString(nonce[:])
	server.folderAuthMu.Lock()
	defer server.folderAuthMu.Unlock()
	if server.folderSelections == nil {
		server.folderSelections = make(map[string]*folderSelection)
	}
	for k, v := range server.folderSelections {
		if !time.Now().Before(v.authorization.expires) {
			delete(server.folderSelections, k)
		}
	}
	if len(server.folderSelections) >= 64 {
		server.fail(w, r, domain.Failure("FOLDER_SELECTION_BUSY", "Hay demasiadas selecciones pendientes. Vuelve a intentarlo en unos minutos.", 429))
		return
	}
	server.folderSelections[key] = &folderSelection{authorization: folderAuthorization{libraryID: input.LibraryID, userID: p.User.ID, sessionID: p.SessionID, tokenDigest: p.TokenDigest, expires: time.Now().Add(3 * time.Minute)}}
	writeJSON(w, 201, map[string]any{"mode": "native", "ticket": key, "launch_url": fmt.Sprintf("aidex-pick://select/%s?port=%s", key, serverLocalPort(server.configuration.ListenAddress))})
}

func (server *Server) selection(r *http.Request) *folderSelection {
	if !authorizationToken.MatchString(r.PathValue("ticket")) {
		return nil
	}
	selection := server.folderSelections[r.PathValue("ticket")]
	if selection == nil || !time.Now().Before(selection.authorization.expires) {
		return nil
	}
	return selection
}
func (server *Server) folderSelectionStatus(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	server.folderAuthMu.Lock()
	defer server.folderAuthMu.Unlock()
	selection := server.selection(r)
	if selection == nil || selection.authorization.userID != p.User.ID || selection.authorization.sessionID != p.SessionID {
		server.fail(w, r, domain.Failure("NOT_FOUND", "La selección venció. Puedes seleccionar de nuevo o introducir la ruta.", 404))
		return
	}
	if err := server.validateFolderTicket(r, selection.authorization); err != nil {
		server.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"done": selection.done, "server_path": selection.path, "code": selection.code})
}
func (server *Server) claimFolderSelection(w http.ResponseWriter, r *http.Request) {
	server.folderAuthMu.Lock()
	defer server.folderAuthMu.Unlock()
	selection := server.selection(r)
	if !localOnly(r) || selection == nil || selection.claimed {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Selección no disponible.", 404))
		return
	}
	if err := server.validateFolderTicket(r, selection.authorization); err != nil {
		server.fail(w, r, err)
		return
	}
	selection.claimed = true
	writeJSON(w, 200, map[string]bool{"available": true})
}
func (server *Server) finishFolderSelection(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"server_path"`
		Code string `json:"code"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	server.folderAuthMu.Lock()
	defer server.folderAuthMu.Unlock()
	selection := server.selection(r)
	if !localOnly(r) || selection == nil || !selection.claimed || selection.done {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Selección no disponible.", 404))
		return
	}
	if err := server.validateFolderTicket(r, selection.authorization); err != nil {
		server.fail(w, r, err)
		return
	}
	if input.Code != "" && input.Code != "cancelled" && input.Code != "unavailable" {
		server.fail(w, r, domain.Failure("INVALID_INPUT", "Resultado no válido.", 422))
		return
	}
	if input.Code == "" && (!filepath.IsAbs(input.Path) || len(input.Path) > 1024 || strings.ContainsAny(input.Path, "\x00\r\n")) {
		server.fail(w, r, domain.Failure("INVALID_INPUT", "La selección no contiene una ruta del servidor.", 422))
		return
	}
	selection.done = true
	selection.code = input.Code
	if input.Code == "" {
		selection.path = filepath.Clean(input.Path)
	}
	// Selection grants no filesystem access. Existing root inspection and scope
	// validation run with the service identity before registration/authorization.
	writeJSON(w, 200, map[string]bool{"received": true})
}

func pathContains(parent, path string) bool {
	relative, err := filepath.Rel(strings.ToLower(parent), strings.ToLower(path))
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}
func (server *Server) browseFolders(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	library := r.PathValue("library")
	for _, permission := range []string{"storage.manage_roots", "storage.view_paths"} {
		if err := server.libraries.Read(r.Context(), p, library, permission); err != nil {
			server.fail(w, r, err)
			return
		}
	}
	var input struct {
		Root     string `json:"root_id"`
		Relative string `json:"relative_path"`
		Cursor   int    `json:"cursor"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	roots, err := server.libraries.Roots(r.Context(), p, library)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	allowed := map[string]string{}
	for _, root := range roots {
		if root.Path != "" && root.Status != "superseded" && root.Status != "retired" {
			allowed[root.ID] = root.Path
		}
	}
	for index, root := range server.configuration.FolderBrowseRoots {
		if root.LibraryID == library {
			allowed["configured-"+strconv.Itoa(index)] = root.Path
		}
	}
	// Do not expose AIDEX private state, even through administrator configuration.
	for key, path := range allowed {
		canonical, e := filepath.EvalSymlinks(path)
		unsafe := e != nil || canonical != filepath.Clean(path)
		for _, private := range []string{server.configuration.StateDirectory, server.configuration.PrivateUploadDirectory()} {
			resolved, e := filepath.EvalSymlinks(private)
			if e != nil {
				resolved = filepath.Clean(private)
			}
			unsafe = unsafe || pathContains(canonical, resolved) || pathContains(resolved, canonical)
		}
		if unsafe {
			delete(allowed, key)
		}
	}
	if input.Root == "" {
		items := []map[string]string{}
		for key, path := range allowed {
			items = append(items, map[string]string{"id": key, "name": filepath.Base(path)})
		}
		sort.Slice(items, func(i, j int) bool { return items[i]["name"] < items[j]["name"] })
		writeJSON(w, 200, map[string]any{"roots": items})
		return
	}
	base, ok := allowed[input.Root]
	relative := input.Relative
	if relative == "" {
		relative = "."
	}
	if !ok || (relative != "." && (!filepath.IsLocal(filepath.FromSlash(relative)) || strings.Contains(relative, `\`) || filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative))) != relative)) || input.Cursor < 0 || input.Cursor > 100000 {
		server.fail(w, r, domain.Failure("FOLDER_BROWSE_FORBIDDEN", "Esta ubicación no está habilitada para explorar.", 403))
		return
	}
	relative = filepath.FromSlash(relative)
	root, err := openBrowseRoot(base)
	if err != nil {
		server.fail(w, r, domain.Failure("ROOT_PERMISSION_DENIED", "El servicio no puede abrir esta carpeta. Introduce su ruta para solicitar autorización.", 422))
		return
	}
	defer root.Close()
	prefix := ""
	if relative != "." {
		for _, part := range strings.Split(relative, string(filepath.Separator)) {
			prefix = filepath.Join(prefix, part)
			info, e := root.Lstat(prefix)
			if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				server.fail(w, r, domain.Failure("FOLDER_BROWSE_FORBIDDEN", "No se admiten enlaces ni ubicaciones fuera de la carpeta habilitada.", 403))
				return
			}
		}
	}
	directory, err := root.Open(relative)
	if err != nil {
		server.fail(w, r, domain.Failure("ROOT_PERMISSION_DENIED", "No se pudo leer la carpeta con los permisos del servicio.", 422))
		return
	}
	defer directory.Close()
	if input.Cursor > 0 {
		if _, err = directory.ReadDir(input.Cursor); err != nil && err != io.EOF {
			server.fail(w, r, err)
			return
		}
	}
	entries, err := directory.ReadDir(256)
	if err != nil && err != io.EOF {
		server.fail(w, r, err)
		return
	}
	items := []map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 && !strings.HasPrefix(entry.Name(), ".") {
			items = append(items, map[string]string{"name": entry.Name(), "relative_path": filepath.ToSlash(filepath.Join(relative, entry.Name()))})
		}
	}
	next := 0
	if len(entries) == 256 {
		next = input.Cursor + 256
	}
	writeJSON(w, 200, map[string]any{"directories": items, "server_path": filepath.Join(base, relative), "next_cursor": next})
}

// Pin every ancestor before opening the next component. A replaced symlink
// cannot redirect the initial browse boundary into private service state.
func openBrowseRoot(path string) (*os.Root, error) {
	base := filepath.VolumeName(path) + string(filepath.Separator)
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}
	for _, part := range strings.Split(strings.TrimPrefix(path, base), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		before, e := root.Lstat(part)
		if e != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, fmt.Errorf("browse boundary contains a link or inaccessible directory")
		}
		next, e := root.OpenRoot(part)
		if e != nil {
			root.Close()
			return nil, e
		}
		after, e := next.Stat(".")
		root.Close()
		if e != nil || !os.SameFile(before, after) {
			next.Close()
			return nil, fmt.Errorf("browse boundary changed")
		}
		root = next
	}
	return root, nil
}
