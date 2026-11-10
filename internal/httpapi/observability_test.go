//go:build development

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObservabilityHTTPAuthorizationCSRFAndSanitizedFailure(t *testing.T) {
	f := newHTTPFixture(t)
	ctx := context.Background()
	cookie, session := f.login(t, "admin", testPassword)
	secret := "password=DO_NOT_LOG token=HIDDEN_KEY file=C:/private/budget.txt"
	request := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/v1/example?token=HIDDEN_KEY", nil)
	response := httptest.NewRecorder()
	f.server.boundary(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.server.fail(w, r, errors.New(secret)) })).ServeHTTP(response, request)
	if response.Code != 500 || strings.Contains(f.logs.String(), "DO_NOT_LOG") {
		t.Fatal("unsafe HTTP failure", response.Code, f.logs.String())
	}
	response = f.request("GET", "/system/errors?module=http", "", cookie, "")
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var page libraries.Page[libraries.DiagnosticEvent]
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Code != "INTERNAL_ERROR" {
		t.Fatal(page)
	}
	event := page.Items[0]
	endpoint := "/system/errors/" + event.ID
	response = f.request("GET", endpoint+"/diagnostic", "", cookie, "")
	for _, forbidden := range []string{"DO_NOT_LOG", "HIDDEN_KEY", "budget.txt", "password="} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatal(response.Body.String())
		}
	}
	if response.Code != 200 || !strings.Contains(response.Body.String(), "operating_system") {
		t.Fatal(response.Code, response.Body.String())
	}
	body := fmt.Sprintf(`{"status":"reviewed","revision":%d}`, event.Revision)
	if got := f.request("PATCH", endpoint, body, cookie, ""); got.Code != 403 {
		t.Fatal("CSRF", got.Code)
	}
	if got := f.request("PATCH", endpoint, body, cookie, session.CSRFToken); got.Code != 204 {
		t.Fatal(got.Code, got.Body.String())
	}
	principal, err := f.service.Authenticate(ctx, cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.service.CreateUser(ctx, principal, "diagnostic-reader", "Reader", "abcdef", domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	reader, _ := f.login(t, user.Username, "abcdef")
	for _, endpoint := range []string{"/system/dashboard", "/system/dashboard/libraries", "/system/errors", "/system/error-policy", endpoint + "/diagnostic"} {
		if got := f.request("GET", endpoint, "", reader, ""); got.Code != 403 {
			t.Fatal("reader bypass", endpoint, got.Code)
		}
		if got := f.request("GET", endpoint, "", nil, ""); got.Code != 401 {
			t.Fatal("anonymous bypass", endpoint, got.Code)
		}
	}
}
