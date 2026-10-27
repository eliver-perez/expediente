package httpapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/backups"
	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

const testPassword = "HTTP-fixture-password-2026"

func TestSuppliedContractIsEmbeddedAndOldDraftDoesNotAuthorizeIt(t *testing.T) {
	if licensing.DevelopmentEnabled() {
		t.Skip("development mode bypasses EULA; production enforcement is tested without the development tag")
	}
	source, err := os.ReadFile("../../AIDEX_CLUF_v1.0_BORRADOR.md")
	if err != nil {
		t.Fatal(err)
	}
	marker := "\n---\n\n**NOTA DE ELABORACIÓN (NO INTEGRAR AL TEXTO CONTRACTUAL MOSTRADO AL USUARIO):**"
	parts := strings.Split(string(source), marker)
	if len(parts) != 2 || strings.TrimSpace(parts[0])+"\n" != eulaText {
		t.Fatal("embedded contract differs from the supplied 18-clause text")
	}
	fixture := newHTTPFixtureWithLicense(t, false)
	cookie, session := fixture.login(t, "admin", testPassword)
	state, err := fixture.service.License.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.Database.Writer.Exec("INSERT INTO eula_acceptances(id,installation_id,user_id,version,text_sha256,accepted_at) VALUES(?,?,?,?,?,?)", domain.NewID(), state.InstallationID, session.User.ID, "2026-10-22-draft", "old-draft-hash", domain.Timestamp(time.Now().Add(-time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	response := fixture.request("GET", "/eula", "", cookie, "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"accepted":false`) || !strings.Contains(response.Body.String(), `"accepted_version":"2026-10-22-draft"`) {
		t.Fatal("old draft silently accepted", response.Code, response.Body.String()[:min(response.Body.Len(), 200)])
	}
	if about := fixture.request("GET", "/about", "", cookie, ""); about.Code != 200 || !strings.Contains(about.Body.String(), `"version":"1.0.0"`) {
		t.Fatal("about unavailable without commercial license", about.Code, about.Body.String())
	}
	accepted := fixture.request("POST", "/eula/accept", `{"version":"`+eulaVersion+`","sha256":"`+eulaHash()+`","accept":true}`, cookie, session.CSRFToken)
	if accepted.Code != 200 {
		t.Fatal(accepted.Code, accepted.Body.String())
	}
	var count int
	if err := fixture.service.Database.Reader.QueryRow("SELECT count(*) FROM eula_acceptances WHERE installation_id=?", state.InstallationID).Scan(&count); err != nil || count != 2 {
		t.Fatal("acceptance history overwritten", count, err)
	}
}

type httpFixture struct {
	server  *Server
	handler http.Handler
	service *identity.Service
	logs    *bytes.Buffer
}

func newHTTPFixture(t *testing.T) httpFixture {
	return newHTTPFixtureWithLicense(t, true)
}

func newHTTPFixtureWithLicense(t *testing.T, licensed bool) httpFixture {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	configuration := config.Defaults(directory)
	if !licensed {
		configuration.License.DevelopmentBypass = false
	}
	var signer ed25519.PrivateKey
	if licensed {
		public, private, keyErr := ed25519.GenerateKey(rand.Reader)
		if keyErr != nil {
			t.Fatal(keyErr)
		}
		signer = private
		configuration.License.TrustedKeys = []licensing.TrustKey{{Kid: "http-test-random", PublicKey: base64.RawURLEncoding.EncodeToString(public), Environment: "production", Purpose: "license"}}
	}
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
	if licensed {
		binding, statusErr := service.License.Status(context.Background())
		if statusErr != nil {
			t.Fatal(statusErr)
		}
		one := int64(1)
		features := map[string]bool{}
		for _, name := range licensing.FeatureNames {
			features[name] = true
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		claims := licensing.Claims{SchemaVersion: "1.0", ProductID: licensing.ProductID, LicenseID: domain.NewID(), ActivationID: domain.NewID(), Binding: binding.Binding, Type: "perpetual", Status: "active", Revision: 1, IssuedAt: now, EntitledReleaseUntil: now, Features: features, Limits: map[string]*int64{"max_installations": &one, "max_users": nil, "max_libraries": nil, "max_documents": nil}}
		header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": "http-test-random", "typ": "lic+jws"})
		payload, _ := json.Marshal(claims)
		body := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
		jws := body + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(signer, []byte(body)))
		if _, err := service.License.Import(context.Background(), []byte(jws), nil, domain.RequestMetadata{}); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Writer.Exec("INSERT INTO eula_acceptances VALUES(?,?,?,?,?,?)", domain.NewID(), binding.InstallationID, func() string {
			var id string
			_ = database.Reader.QueryRow("SELECT id FROM users WHERE username='admin'").Scan(&id)
			return id
		}(), eulaVersion, eulaHash(), domain.Timestamp(time.Now())); err != nil {
			t.Fatal(err)
		}
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

func TestEulaAcceptanceAndDeactivatedSessionCannotReadDocuments(t *testing.T) {
	if licensing.DevelopmentEnabled() {
		t.Skip("development mode bypasses EULA; production enforcement is tested without the development tag")
	}
	fixture := newHTTPFixtureWithLicense(t, false)
	cookie, session := fixture.login(t, "admin", testPassword)
	contract := fixture.request("GET", "/eula", "", cookie, "")
	if contract.Code != 200 || !strings.Contains(contract.Body.String(), "Borrador sujeto a revisión jurídica") || !strings.Contains(contract.Body.String(), "## 18. Integridad, comunicaciones y consulta") {
		t.Fatal(contract.Code, contract.Body.String())
	}
	if response := fixture.request("GET", "/libraries", "", cookie, ""); response.Code != 403 || !strings.Contains(response.Body.String(), "EULA_ACCEPTANCE_REQUIRED") {
		t.Fatal("documentary API available before accepting EULA", response.Code, response.Body.String())
	}
	if response := fixture.request("GET", "/license", "", cookie, ""); response.Code != 200 {
		t.Fatal("license management blocked before EULA acceptance", response.Code, response.Body.String())
	}
	if response := fixture.request("POST", "/eula/accept", `{"version":"bad","sha256":"bad","accept":true}`, cookie, session.CSRFToken); response.Code != 422 {
		t.Fatal(response.Code)
	}
	accepted := fixture.request("POST", "/eula/accept", `{"version":"`+eulaVersion+`","sha256":"`+eulaHash()+`","accept":true}`, cookie, session.CSRFToken)
	if accepted.Code != 200 {
		t.Fatal(accepted.Code, accepted.Body.String())
	}
	var count int
	if err := fixture.service.Database.Reader.QueryRow("SELECT count(*) FROM eula_acceptances WHERE version=? AND text_sha256=?", eulaVersion, eulaHash()).Scan(&count); err != nil || count != 1 {
		t.Fatal("acceptance not recorded", err)
	}
	for _, path := range []string{"/libraries", "/search", "/backups", "/documents/unknown/download"} {
		method := "GET"
		if path == "/search" {
			method = "POST"
		}
		response := fixture.request(method, path, `{}`, cookie, session.CSRFToken)
		if response.Code != 403 || !strings.Contains(response.Body.String(), "LICENSE_RECOVERY_REQUIRED") {
			t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
		}
	}
	active := newHTTPFixtureWithLicense(t, true)
	cookie, session = active.login(t, "admin", testPassword)
	state, err := active.service.License.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = active.service.Database.Writer.Exec("UPDATE license_state SET deactivated=1 WHERE singleton=1"); err != nil {
		t.Fatal(err)
	}
	if _, err = active.service.Database.Writer.Exec("UPDATE license_revision_floors SET deactivated=1 WHERE activation_id=?", state.ActivationID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/libraries", "/search", "/backups", "/documents/unknown/download"} {
		method := "GET"
		if path == "/search" {
			method = "POST"
		}
		response := active.request(method, path, `{}`, cookie, session.CSRFToken)
		if response.Code != 403 || !strings.Contains(response.Body.String(), "ACTIVATION_DEACTIVATED") {
			t.Fatalf("deactivated %s: %d %s", path, response.Code, response.Body.String())
		}
	}
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
	for _, path := range []string{"/users", "/sessions", "/authentication-attempts", "/audit-events", "/system/network", "/system/files", "/backups"} {
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

func TestRestoreAPIRequiresCSRFAndDestinationLicense(t *testing.T) {
	fixture := newHTTPFixtureWithLicense(t, false)
	backupService, err := backups.New(fixture.service.Database, fixture.service.Config)
	if err != nil {
		t.Fatal(err)
	}
	fixture.handler = fixture.server.WithBackups(backupService).Handler()
	cookie, login := fixture.login(t, "admin", testPassword)
	fixture.service.License.Options.DevelopmentBypass = false
	if response := fixture.request("POST", "/restores/example/confirm", `{"confirmation":"RESTAURAR"}`, cookie, ""); response.Code != 403 || !strings.Contains(response.Body.String(), "CSRF_INVALID") {
		t.Fatal(response.Code, response.Body.String())
	}
	response := fixture.request("POST", "/restores/example/confirm", `{"confirmation":"RESTAURAR"}`, cookie, login.CSRFToken)
	expected := "EULA_ACCEPTANCE_REQUIRED"
	if licensing.DevelopmentEnabled() {
		expected = "LICENSE_RECOVERY_REQUIRED"
	}
	if response.Code != 403 || !strings.Contains(response.Body.String(), expected) {
		t.Fatal(response.Code, response.Body.String())
	}
}

func TestRestoreOnlyAdministratorCanListWithoutGeneratingBackups(t *testing.T) {
	fixture := newHTTPFixture(t)
	backupService, err := backups.New(fixture.service.Database, fixture.service.Config)
	if err != nil {
		t.Fatal(err)
	}
	fixture.handler = fixture.server.WithBackups(backupService).Handler()
	adminCookie, adminSession := fixture.login(t, "admin", testPassword)
	created := fixture.request("POST", "/users", `{"username":"restoreoperator","display_name":"Restore Operator","password":"Restore-fixture-password"}`, adminCookie, adminSession.CSRFToken)
	if created.Code != 201 {
		t.Fatal(created.Code, created.Body.String())
	}
	if _, err = fixture.service.Database.Writer.Exec("INSERT INTO roles(id,name,scope_kind) VALUES('restore_operator','Restore operator','global')"); err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.Database.Writer.Exec("INSERT INTO role_permissions(role_id,permission_key) VALUES('restore_operator','restore.manage')"); err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.service.Database.Writer.Exec("INSERT INTO global_role_assignments(user_id,role_id) SELECT id,'restore_operator' FROM users WHERE username='restoreoperator'"); err != nil {
		t.Fatal(err)
	}
	cookie, session := fixture.login(t, "restoreoperator", "Restore-fixture-password")
	if response := fixture.request("GET", "/backups", "", cookie, ""); response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	if response := fixture.request("POST", "/backups", `{"kind":"basic"}`, cookie, session.CSRFToken); response.Code != 403 {
		t.Fatal(response.Code, response.Body.String())
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
