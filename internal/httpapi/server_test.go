package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/storage"
)

const testPassword = "HTTP-fixture-password-2026"

type httpFixture struct {
	server  *Server
	handler http.Handler
	service *identity.Service
	logs    *bytes.Buffer
}

func newHTTPFixture(t *testing.T) httpFixture {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	configuration := config.Defaults(directory)
	database, err := storage.Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	service, err := identity.New(database, configuration)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Bootstrap(context.Background(), "admin", "HTTP Administrator", testPassword); err != nil {
		t.Fatal(err)
	}
	logs := new(bytes.Buffer)
	server := New(service, configuration, slog.New(slog.NewJSONHandler(logs, nil)))
	return httpFixture{server: server, handler: server.Handler(), service: service, logs: logs}
}

func (fixture httpFixture) request(method, path, body string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "http://127.0.0.1:8090/api/v1"+path, strings.NewReader(body))
	request.RemoteAddr = "192.0.2.4:12345"
	request.Header.Set("Origin", "http://127.0.0.1:8090")
	request.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	writer := httptest.NewRecorder()
	fixture.handler.ServeHTTP(writer, request)
	return writer
}

func (fixture httpFixture) login(t *testing.T, username, password string) (*http.Cookie, identity.LoginResult) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	response := fixture.request("POST", "/auth/login", string(body), nil, "")
	if response.Code != 200 {
		t.Fatalf("login: %d %s", response.Code, response.Body.String())
	}
	var result identity.LoginResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing session cookie")
	}
	return cookies[0], result
}

func TestHTTPReplacementCookiesCSRFAndPasswordChange(t *testing.T) {
	fixture := newHTTPFixture(t)
	firstCookie, _ := fixture.login(t, "admin", testPassword)
	secondCookie, session := fixture.login(t, "admin", testPassword)
	if !secondCookie.HttpOnly || secondCookie.SameSite != http.SameSiteLaxMode || secondCookie.Path != "/" {
		t.Fatal("unsafe cookie attributes")
	}
	response := fixture.request("GET", "/auth/session", "", firstCookie, "")
	if response.Code != 401 || !strings.Contains(response.Body.String(), "SESSION_REPLACED") || !strings.Contains(response.Body.String(), identity.SessionReplacedMessage) {
		t.Fatal(response.Body.String())
	}
	if len(response.Result().Cookies()) != 0 {
		t.Fatal("old request could clear a newer session cookie")
	}
	if response := fixture.request("POST", "/auth/logout", "", secondCookie, ""); response.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	response = fixture.request("POST", "/account/password", `{"current_password":"`+testPassword+`","new_password":"Changed-HTTP-password-2026"}`, secondCookie, session.CSRFToken)
	if response.Code != 204 {
		t.Fatal(response.Code, response.Body.String())
	}
	if response := fixture.request("GET", "/account", "", secondCookie, ""); response.Code != 401 {
		t.Fatal("old session survived password change")
	}
}

func TestHTTPAuthorizationFailedAttemptsAndNoSecretDisclosure(t *testing.T) {
	fixture := newHTTPFixture(t)
	adminCookie, session := fixture.login(t, "admin", testPassword)
	created := fixture.request("POST", "/users", `{"username":"reader","display_name":"Reader","password":"Reader-fixture-password"}`, adminCookie, session.CSRFToken)
	if created.Code != 201 {
		t.Fatal(created.Code, created.Body.String())
	}
	readerCookie, _ := fixture.login(t, "reader", "Reader-fixture-password")
	for _, path := range []string{"/users", "/sessions", "/authentication-attempts", "/audit-events"} {
		if response := fixture.request("GET", path, "", readerCookie, ""); response.Code != 403 {
			t.Fatal("unprivileged access", path, response.Code)
		}
	}
	response := fixture.request("POST", "/auth/login", `{"username":"no-such-user","password":"NEVER-LOG-THIS-SECRET"}`, nil, "")
	if response.Code != 401 {
		t.Fatal(response.Code)
	}
	var count int
	if err := fixture.service.Database.Reader.QueryRow("SELECT count(*) FROM authentication_attempts WHERE attempted_identifier='no-such-user' AND user_id IS NULL AND outcome='failed'").Scan(&count); err != nil || count != 1 {
		t.Fatal("missing anonymous attempt", err)
	}
	if strings.Contains(fixture.logs.String(), "NEVER-LOG") || strings.Contains(response.Body.String(), "password_hash") || strings.Contains(created.Body.String(), "$argon2id$") {
		t.Fatal("secret disclosure")
	}
}

func TestStrictJSONOriginHostAndProxyBoundaries(t *testing.T) {
	fixture := newHTTPFixture(t)
	for _, body := range []string{`{"username":"admin","username":"reader","password":"x"}`, `{"username":"admin","password":"x","unexpected":true}`, `{} {}`, `null`} {
		if response := fixture.request("POST", "/auth/login", body, nil, ""); response.Code != 400 {
			t.Fatal("invalid JSON accepted", body, response.Code)
		}
	}
	request := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/v1/auth/login", strings.NewReader(`{}`))
	request.Header.Set("Origin", "https://attacker.example")
	writer := httptest.NewRecorder()
	fixture.handler.ServeHTTP(writer, request)
	if writer.Code != 403 {
		t.Fatal("cross-origin accepted")
	}
	request = httptest.NewRequest("GET", "http://attacker.example/api/v1/account", nil)
	writer = httptest.NewRecorder()
	fixture.handler.ServeHTTP(writer, request)
	if writer.Code != 400 {
		t.Fatal("arbitrary Host accepted")
	}
	request = httptest.NewRequest("GET", "http://127.0.0.1:8090/", nil)
	request.RemoteAddr = "192.0.2.30:1000"
	request.Header.Set("X-Forwarded-For", "1.2.3.4")
	if fixture.server.observedIP(request) != "192.0.2.30" {
		t.Fatal("spoofed proxy IP trusted")
	}
	configuration := fixture.server.configuration
	configuration.TrustedProxies = []string{"127.0.0.1/32"}
	proxyServer := New(fixture.service, configuration, fixture.server.logger)
	request.RemoteAddr = "127.0.0.1:1000"
	request.Header.Set("X-Forwarded-For", "1.2.3.4, 192.0.2.30")
	if proxyServer.observedIP(request) != "192.0.2.30" {
		t.Fatal("proxy trust chain incorrect")
	}
}

func TestAuditRolePaginationAndImmutableEvents(t *testing.T) {
	fixture := newHTTPFixture(t)
	cookie, result := fixture.login(t, "admin", testPassword)
	principal, err := fixture.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.SetGlobalRoles(context.Background(), principal, principal.User.ID, principal.User.Revision, []string{"installation_admin", "access_auditor"}, domain.RequestMetadata{RequestID: "fixture-role-change"})
	if err != nil {
		t.Fatal(err)
	}
	response := fixture.request("GET", "/audit-events?limit=1", "", cookie, "")
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var firstPage page
	if err := json.Unmarshal(response.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if firstPage.NextCursor == nil {
		t.Fatal("missing next cursor")
	}
	second := fixture.request("GET", "/audit-events?limit=1&cursor="+*firstPage.NextCursor, "", cookie, "")
	if second.Code != 200 {
		t.Fatal(second.Body.String())
	}
	var secondPage page
	json.Unmarshal(second.Body.Bytes(), &secondPage)
	if firstPage.Items[0]["id"] == secondPage.Items[0]["id"] {
		t.Fatal("repeated row")
	}
	wrongScope := fixture.request("GET", "/audit-events?event_type=user.created&cursor="+*firstPage.NextCursor, "", cookie, "")
	if wrongScope.Code != 400 {
		t.Fatal("cursor reused under different filters")
	}
	if response := fixture.request("POST", "/auth/logout", "", cookie, result.CSRFToken); response.Code != 204 {
		t.Fatal(response.Body.String())
	}
}

func TestSecureCookieAndTransportBehindExplicitProxy(t *testing.T) {
	fixture := newHTTPFixture(t)
	configuration := fixture.server.configuration
	configuration.PublicURL = "https://documental.example"
	configuration.TrustedProxies = []string{"127.0.0.1/32"}
	server := New(fixture.service, configuration, fixture.server.logger)
	request := httptest.NewRequest("POST", "https://documental.example/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"`+testPassword+`"}`))
	request.TLS = nil
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Origin", configuration.PublicURL)
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	server.Handler().ServeHTTP(writer, request)
	if writer.Code != 200 {
		t.Fatal(writer.Code, writer.Body.String())
	}
	cookie := writer.Result().Cookies()[0]
	if !cookie.Secure || cookie.Name != "__Host-documental-session" || cookie.Domain != "" {
		t.Fatal("invalid secure cookie")
	}
}
