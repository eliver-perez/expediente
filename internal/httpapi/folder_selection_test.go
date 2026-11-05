package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
)

func TestFolderSelectionLocalTicketSessionAndCancellation(t *testing.T) {
	f := newHTTPFixture(t)
	cookie, session := f.login(t, "admin", testPassword)
	p, err := f.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	library, err := f.server.libraries.Create(context.Background(), p, "Selector", "hybrid", p.User.ID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, peer string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8090/api/v1"+path, strings.NewReader(body))
		r.RemoteAddr = peer
		r.Header.Set("Content-Type", "application/json")
		if authenticated {
			r.AddCookie(cookie)
			r.Header.Set("Origin", "http://127.0.0.1:8090")
			r.Header.Set("X-CSRF-Token", session.CSRFToken)
		}
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	body := `{"library_id":"` + library + `"}`
	remote := request("POST", "/folder-selections", body, "192.0.2.1:1234", true)
	if remote.Code != 200 || !strings.Contains(remote.Body.String(), `"mode":"server"`) || strings.Contains(remote.Body.String(), "launch_url") {
		t.Fatal(remote.Code, remote.Body.String())
	}
	start := request("POST", "/folder-selections", body, "127.0.0.1:1234", true)
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	if start.Code != 201 || json.Unmarshal(start.Body.Bytes(), &ticket) != nil || len(ticket.Ticket) != 64 {
		t.Fatal(start.Code, start.Body.String())
	}
	base := "/folder-selections/" + ticket.Ticket
	if r := request("POST", base+"/claim", "{}", "192.0.2.1:1234", false); r.Code != 403 {
		t.Fatal("remote claim", r.Code)
	}
	if r := request("POST", base+"/finish", `{"code":"cancelled"}`, "127.0.0.1:1234", false); r.Code != 404 {
		t.Fatal("finish without claim", r.Code)
	}
	if r := request("POST", base+"/claim", "{}", "127.0.0.1:1234", false); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := request("POST", base+"/claim", "{}", "127.0.0.1:1234", false); r.Code != 404 {
		t.Fatal("claim replay", r.Code)
	}
	if r := request("POST", base+"/finish", `{"server_path":"relative/folder"}`, "127.0.0.1:1234", false); r.Code != 422 {
		t.Fatal("relative path", r.Code)
	}
	if r := request("POST", base+"/finish", `{"code":"cancelled"}`, "127.0.0.1:1234", false); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	status := request("GET", base, "", "127.0.0.1:1234", true)
	if status.Code != 200 || !strings.Contains(status.Body.String(), `"code":"cancelled"`) || !strings.Contains(status.Body.String(), `"done":true`) {
		t.Fatal(status.Code, status.Body.String())
	}
	if r := request("POST", base+"/finish", `{"code":"cancelled"}`, "127.0.0.1:1234", false); r.Code != 404 {
		t.Fatal("finish replay", r.Code)
	}
	f.server.folderSelections[ticket.Ticket].authorization.expires = time.Now().Add(-time.Second)
	if r := request("GET", base, "", "127.0.0.1:1234", true); r.Code != 404 {
		t.Fatal("expiry", r.Code)
	}
	start = request("POST", "/folder-selections", body, "127.0.0.1:1234", true)
	_ = json.Unmarshal(start.Body.Bytes(), &ticket)
	if err = f.service.Logout(context.Background(), p, domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	if r := request("POST", "/folder-selections/"+ticket.Ticket+"/claim", "{}", "127.0.0.1:1234", false); r.Code == 200 {
		t.Fatal("revoked session")
	}
}

func TestFolderBrowserConfinesLibraryAndRejectsLinksPrivateStateAndTraversal(t *testing.T) {
	f := newHTTPFixture(t)
	cookie, session := f.login(t, "admin", testPassword)
	p, err := f.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	library, err := f.server.libraries.Create(context.Background(), p, "Explorador", "linked", p.User.ID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(directory, "Permitida"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(f.server.configuration.StateDirectory, filepath.Join(directory, "Privada")); err != nil {
		t.Fatal(err)
	}
	f.server.configuration.FolderBrowseRoots = []config.FolderBrowseRoot{{LibraryID: library, Path: directory}, {LibraryID: "another-library", Path: filepath.Dir(directory)}, {LibraryID: library, Path: f.server.configuration.StateDirectory}}
	browse := func(body string) *httptest.ResponseRecorder {
		return f.request("POST", "/libraries/"+library+"/folder-browser", body, cookie, session.CSRFToken)
	}
	roots := browse("{}")
	if roots.Code != 200 || strings.Contains(roots.Body.String(), "configured-1") || strings.Contains(roots.Body.String(), "configured-2") || !strings.Contains(roots.Body.String(), "configured-0") {
		t.Fatal(roots.Code, roots.Body.String())
	}
	listed := browse(`{"root_id":"configured-0"}`)
	if listed.Code != 200 || !strings.Contains(listed.Body.String(), "Permitida") || strings.Contains(listed.Body.String(), "Privada") {
		t.Fatal(listed.Code, listed.Body.String())
	}
	for _, body := range []string{`{"root_id":"configured-0","relative_path":"../"}`, `{"root_id":"configured-0","relative_path":"Privada"}`, `{"root_id":"configured-1"}`, `{"root_id":"configured-0","relative_path":"/"}`} {
		if r := browse(body); r.Code != 403 {
			t.Fatal("escaped root", body, r.Code, r.Body.String())
		}
	}
	if r := browse(`{"root_id":"configured-0","relative_path":"Permitida"}`); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if _, err = openBrowseRoot(filepath.Join(directory, "Privada")); err == nil {
		t.Fatal("opened symlink boundary")
	}
}
