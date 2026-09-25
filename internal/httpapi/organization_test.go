//go:build development

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
)

func TestH4MultipartAuthorizationRangesAndNoTemporaryDisclosure(t *testing.T) {
	fixture := newHTTPFixture(t)
	cookie, session := fixture.login(t, "admin", testPassword)
	ctx := context.Background()
	principal, err := fixture.service.Authenticate(ctx, cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	libraryID, err := fixture.server.libraries.Create(ctx, principal, "Carga HTTP", "managed", principal.User.ID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	batchID, err := fixture.server.libraries.CreateBatch(ctx, principal, libraryID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile("../../testdata/documents/native.pdf")
	if err != nil {
		t.Fatal(err)
	}
	send := func(csrf string, extra bool) *httptest.ResponseRecorder {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		if err := form.WriteField("client_file_id", domain.NewID()); err != nil {
			t.Fatal(err)
		}
		part, err := form.CreateFormFile("file", "Plano.pdf")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = part.Write(contents); err != nil {
			t.Fatal(err)
		}
		if extra {
			if err = form.WriteField("unexpected", "no"); err != nil {
				t.Fatal(err)
			}
		}
		if err = form.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/v1/upload-batches/"+batchID+"/files", &body)
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("Origin", "http://127.0.0.1:8090")
		request.Header.Set("X-CSRF-Token", csrf)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		fixture.handler.ServeHTTP(response, request)
		return response
	}
	if response := send("", false); response.Code != 403 {
		t.Fatalf("multipart bypassed CSRF: %d", response.Code)
	}
	if response := send(session.CSRFToken, true); response.Code != 400 {
		t.Fatalf("extra multipart accepted: %d %s", response.Code, response.Body.String())
	}
	response := send(session.CSRFToken, false)
	if response.Code != 201 {
		t.Fatalf("upload: %d %s", response.Code, response.Body.String())
	}
	var item libraries.UploadItem
	if err = json.Unmarshal(response.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"/documents/" + item.DocumentID, "/upload-batches/" + batchID, "/libraries/" + libraryID + "/explorer"} {
		result := fixture.request("GET", endpoint, "", cookie, "")
		if result.Code != 200 || strings.Contains(result.Body.String(), fixture.service.Config.PrivateUploadDirectory()) || strings.Contains(result.Body.String(), "private_temporary_locator") {
			t.Fatalf("private locator in response or failure: %s", result.Body.String())
		}
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/v1/documents/"+item.DocumentID+"/content", nil)
	request.AddCookie(cookie)
	request.Header.Set("Range", "bytes=0-4")
	rangeResponse := httptest.NewRecorder()
	fixture.handler.ServeHTTP(rangeResponse, request)
	if rangeResponse.Code != http.StatusPartialContent || rangeResponse.Body.String() != "%PDF-" {
		t.Fatalf("private preview range: %d %s", rangeResponse.Code, rangeResponse.Body.String())
	}
	user, err := fixture.service.CreateUser(ctx, principal, "reader-h4", "Reader", "abcdef", domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if err = fixture.server.libraries.SetMember(ctx, principal, libraryID, user.ID, []string{"library_reader"}, domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	readerCookie, _ := fixture.login(t, user.Username, "abcdef")
	for _, endpoint := range []string{"/documents/" + item.DocumentID, "/documents/" + item.DocumentID + "/content", "/documents/" + item.DocumentID + "/download", "/documents/" + item.DocumentID + "/pages/1/text", "/documents/" + item.DocumentID + "/history", "/upload-batches/" + batchID} {
		result := fixture.request("GET", endpoint, "", readerCookie, "")
		if result.Code != 404 {
			t.Fatalf("reader private endpoint %s: %d", endpoint, result.Code)
		}
	}
	if strings.Contains(fixture.logs.String(), fixture.service.Config.PrivateUploadDirectory()) {
		t.Fatal("temporary path leaked to logs")
	}
}
