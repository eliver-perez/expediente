//go:build development

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
)

func TestPreviewHTTPPrivateAccessCSRFAndRanges(t *testing.T) {
	fixture := newHTTPFixture(t)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	cookie, session := fixture.login(t, "admin", testPassword)
	principal, err := fixture.service.Authenticate(ctx, cookie.Value, false, domain.RequestMetadata{})
	must(err)
	service := fixture.server.libraries
	library, err := service.Create(ctx, principal, "Preview HTTP", "managed", principal.User.ID, domain.NewID(), domain.RequestMetadata{})
	must(err)
	batch, err := service.CreateBatch(ctx, principal, library, domain.NewID(), domain.RequestMetadata{})
	must(err)
	contents, err := os.ReadFile("../../testdata/documents/sample.docx")
	must(err)
	item, err := service.Upload(ctx, principal, batch, domain.NewID(), "Prueba.docx", bytes.NewReader(contents), nil, domain.RequestMetadata{})
	must(err)
	endpoint := "/documents/" + item.DocumentID + "/preview"
	if result := fixture.request("POST", endpoint, `{"retry":false}`, cookie, ""); result.Code != 403 {
		t.Fatal("CSRF bypass", result.Code)
	}
	response := fixture.request("POST", endpoint, `{"retry":false}`, cookie, session.CSRFToken)
	if response.Code != 202 {
		t.Fatal(response.Code, response.Body.String())
	}
	var preview libraries.Preview
	must(json.Unmarshal(response.Body.Bytes(), &preview))
	// A ready controlled PDF isolates the delivery contract from converter availability.
	dir := filepath.Join(fixture.service.Config.StateDirectory, "previews")
	must(os.MkdirAll(dir, 0700))
	pdf := []byte("%PDF-1.7\nHTTP preview fixture")
	must(os.WriteFile(filepath.Join(dir, preview.ID+".preview"), pdf, 0600))
	_, err = fixture.service.Database.Writer.Exec("UPDATE preview_cache SET status='ready',size_bytes=?,media_type='application/pdf' WHERE id=?", len(pdf), preview.ID)
	must(err)
	contentPath := endpoint + "/content?id=" + preview.ID
	response = fixture.request("GET", contentPath, "", cookie, "")
	if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), pdf) || response.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(response.Header().Get("Content-Security-Policy"), "sandbox") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(response.Code, response.Header(), response.Body.String())
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/v1"+contentPath, nil)
	request.AddCookie(cookie)
	request.Header.Set("Range", "bytes=0-4")
	rangeResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(rangeResponse, request)
	if rangeResponse.Code != 206 || rangeResponse.Body.String() != "%PDF-" {
		t.Fatal("range", rangeResponse.Code, rangeResponse.Body.String())
	}
	user, err := fixture.service.CreateUser(ctx, principal, "preview-reader", "Reader", "abcdef", domain.RequestMetadata{})
	must(err)
	must(service.SetMember(ctx, principal, library, user.ID, []string{"library_reader"}, domain.RequestMetadata{}))
	readerCookie, readerSession := fixture.login(t, user.Username, "abcdef")
	for _, path := range []string{endpoint, contentPath} {
		if response = fixture.request("GET", path, "", readerCookie, ""); response.Code != 404 {
			t.Fatal("private preview leaked", response.Code)
		}
		if response = fixture.request("GET", path, "", nil, ""); response.Code != 401 {
			t.Fatal("anonymous preview", response.Code)
		}
	}
	if response = fixture.request("GET", "/system/previews", "", readerCookie, ""); response.Code != 403 {
		t.Fatal("reader configuration", response.Code)
	}
	if response = fixture.request("POST", "/system/previews/clear", `{"confirm":true}`, readerCookie, readerSession.CSRFToken); response.Code != 403 {
		t.Fatal("reader cleared cache", response.Code)
	}
	if response = fixture.request("POST", "/system/previews/clear", `{"confirm":false}`, cookie, session.CSRFToken); response.Code != 422 {
		t.Fatal("missing confirmation", response.Code)
	}
	if response = fixture.request("POST", "/system/previews/clear", `{"confirm":true}`, cookie, session.CSRFToken); response.Code != 204 {
		t.Fatal(response.Code, response.Body.String())
	}
	if response = fixture.request("GET", contentPath, "", cookie, ""); response.Code != 409 {
		t.Fatal("cache survived clear", response.Code)
	}
}
