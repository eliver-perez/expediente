package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

func TestLocalFolderAuthorizationRejectsBroadOrRemotePaths(t *testing.T) {
	for _, path := range []string{`\\server\share\docs`, `C:\`, `C:relative`, `C:\docs\bad` + "\n" + `name`, `D:\\`, `/tmp/docs`} {
		if localFolderAuthorizationPath(path) {
			t.Fatalf("accepted unsafe path %q", path)
		}
	}
	if !localFolderAuthorizationPath(`C:\Users\Ada\Documents`) {
		t.Fatal("rejected local directory")
	}
	for _, test := range []struct {
		remote, host string
		allowed      bool
	}{
		{"127.0.0.1:53210", "127.0.0.1:18090", true},
		{"[::1]:53210", "localhost:18090", true},
		{"192.0.2.15:53210", "127.0.0.1:18090", false},
		{"127.0.0.1:53210", "192.0.2.15:18090", false},
	} {
		r := httptest.NewRequest("POST", "http://"+test.host+"/api/v1/local-folder-authorizations", nil)
		r.RemoteAddr = test.remote
		if localOnly(r) != test.allowed {
			t.Fatalf("remote=%s host=%s", test.remote, test.host)
		}
	}
	proxy := httptest.NewRequest("POST", "http://127.0.0.1:18090/api/v1/local-folder-authorizations", nil)
	proxy.RemoteAddr = "127.0.0.1:53210"
	proxy.Header.Set("X-Forwarded-For", "192.0.2.15")
	if localOnly(proxy) {
		t.Fatal("reverse proxy request accepted as local")
	}
}

func TestMacFolderTicketsSingleUseEffectiveProbeAndSession(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("flujo macOS")
	}
	f := newHTTPFixture(t)
	cookie, _ := f.login(t, "admin", testPassword)
	p, err := f.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	library, err := f.server.libraries.Create(context.Background(), p, "Autorización sintética", "hybrid", p.User.ID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, ticket, operation string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8090/api/v1/local-folder-authorizations/"+ticket+"/"+operation, strings.NewReader("{}"))
		r.RemoteAddr = "127.0.0.1:50400"
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	ticket := strings.Repeat("a", 64)
	f.server.folderAuth[ticket] = &folderAuthorization{path: path, source: "managed", libraryID: library, userID: p.User.ID, sessionID: p.SessionID, tokenDigest: p.TokenDigest, expires: time.Now().Add(time.Minute)}
	if result := request(http.MethodGet, ticket, "describe"); result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
	if result := request(http.MethodPost, ticket, "claim"); result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
	if result := request(http.MethodPost, ticket, "claim"); result.Code != 404 {
		t.Fatal("ticket reutilizado", result.Code)
	}
	result := request(http.MethodPost, ticket, "finish")
	if result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
	var checked struct {
		Accessible bool `json:"accessible"`
	}
	if err = json.Unmarshal(result.Body.Bytes(), &checked); err != nil || !checked.Accessible {
		t.Fatal(result.Body.String(), err)
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		t.Fatal("la prueba de escritura dejó residuos", entries, err)
	}
	if result = request(http.MethodPost, ticket, "finish"); result.Code != 404 {
		t.Fatal("finish reutilizado", result.Code)
	}
	// Revocar directamente la sesión sintética no debe permitir usar su ticket.
	ticket = strings.Repeat("b", 64)
	f.server.folderAuth[ticket] = &folderAuthorization{path: path, source: "linked", libraryID: library, userID: p.User.ID, sessionID: p.SessionID, tokenDigest: p.TokenDigest, expires: time.Now().Add(time.Minute)}
	if err = f.service.Logout(context.Background(), p, domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	if result = request(http.MethodPost, ticket, "claim"); result.Code == 200 {
		t.Fatal("se autorizó una sesión revocada")
	}
}

func TestFolderAuthorizationTicketExpiry(t *testing.T) {
	server := &Server{folderAuth: map[string]*folderAuthorization{"expired": {expires: time.Now().Add(-time.Second)}, "pending": {expires: time.Now().Add(time.Minute)}}}
	server.expireFolderTickets(time.Now())
	if server.folderAuth["expired"] != nil || server.folderAuth["pending"] == nil {
		t.Fatal("ticket expiry not enforced")
	}
}
