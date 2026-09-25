package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

func TestH6LicenseRecoveryPermissionsCSRFAndPrivateArtifacts(t *testing.T) {
	fixture := newHTTPFixture(t)
	cookie, session := fixture.login(t, "admin", testPassword)
	response := fixture.request("GET", "/license", "", cookie, "")
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "private_seed") || strings.Contains(response.Body.String(), fixture.service.Config.StateDirectory) {
		t.Fatal("license status disclosed private data")
	}
	input := `{"action":"activate","request_id":"` + domain.NewID() + `"}`
	if result := fixture.request("POST", "/license/offline-requests", input, cookie, ""); result.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	response = fixture.request("POST", "/license/offline-requests", input, cookie, session.CSRFToken)
	if response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	var artifact licensing.Artifact
	json.Unmarshal(response.Body.Bytes(), &artifact)
	download := fixture.request("GET", "/license/artifacts/"+artifact.ID, "", cookie, "")
	if download.Code != 200 || !strings.Contains(download.Header().Get("Content-Disposition"), ".licreq") {
		t.Fatal(download.Code, download.Body.String())
	}
	if _, _, err := licensing.VerifyOfflineRequest(download.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	created := fixture.request("POST", "/users", `{"username":"license-reader","display_name":"Reader","password":"abcdef"}`, cookie, session.CSRFToken)
	if created.Code != 201 {
		t.Fatal(created.Body.String())
	}
	reader, _ := fixture.login(t, "license-reader", "abcdef")
	for _, path := range []string{"/license", "/license/artifacts", "/license/artifacts/" + artifact.ID} {
		if result := fixture.request("GET", path, "", reader, ""); result.Code != 403 {
			t.Fatal("unauthorized license recovery", path, result.Code)
		}
	}
	if response = fixture.request("POST", "/license/deactivate", `{"request_id":"`+domain.NewID()+`"}`, cookie, session.CSRFToken); response.Code != 422 || !strings.Contains(response.Body.String(), "CONFIRMATION_REQUIRED") {
		t.Fatal(response.Code, response.Body.String())
	}
	for _, size := range []int{10, 65537} {
		var buffer bytes.Buffer
		form := multipart.NewWriter(&buffer)
		part, _ := form.CreateFormFile("file", "test.lic")
		part.Write(bytes.Repeat([]byte{'x'}, size))
		form.Close()
		request := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/v1/license/import", &buffer)
		request.RemoteAddr = "192.0.2.4:12345"
		request.Header.Set("Origin", "http://127.0.0.1:8090")
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("X-CSRF-Token", session.CSRFToken)
		request.AddCookie(cookie)
		result := httptest.NewRecorder()
		fixture.handler.ServeHTTP(result, request)
		expected := 422
		if size > 65536 {
			expected = http.StatusRequestEntityTooLarge
		}
		if result.Code != expected {
			t.Fatal(size, result.Code, result.Body.String())
		}
	}
	if strings.Contains(fixture.logs.String(), "private_seed") || strings.Contains(fixture.logs.String(), "signature_b64u") {
		t.Fatal("secret in HTTP log")
	}
}
