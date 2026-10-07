//go:build development

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReindexHTTPAuthorizationIdempotencyAndExplorerFormat(t *testing.T) {
	f := newHTTPFixture(t)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	cookie, session := f.login(t, "admin", testPassword)
	p, err := f.service.Authenticate(ctx, cookie.Value, false, domain.RequestMetadata{})
	must(err)
	s := f.server.libraries
	library, err := s.Create(ctx, p, "Manual HTTP", "managed", p.User.ID, domain.NewID(), domain.RequestMetadata{})
	must(err)
	settings, err := s.FileConfiguration(ctx, p, "")
	must(err)
	settings.Effective.Index = []string{"pdf", "txt"}
	must(s.ConfigureFiles(ctx, p, "", settings.Effective, documentformat.Overrides{}, settings.Revision, settings.GlobalRevision, domain.RequestMetadata{}))
	batch, err := s.CreateBatch(ctx, p, library, domain.NewID(), domain.RequestMetadata{})
	must(err)
	item, err := s.Upload(ctx, p, batch, domain.NewID(), "Texto.txt", bytes.NewBufferString("Texto para reindexación"), nil, domain.RequestMetadata{})
	must(err)
	endpoint := "/documents/" + item.DocumentID + "/reindex"
	if response := f.request("POST", endpoint, "{}", cookie, ""); response.Code != 403 {
		t.Fatal("CSRF", response.Code)
	}
	if response := f.request("POST", endpoint, "{}", nil, ""); response.Code != 401 {
		t.Fatal("anonymous", response.Code)
	}
	if response := f.request("POST", endpoint, "{}", cookie, session.CSRFToken); response.Code != 422 {
		t.Fatal("missing key", response.Code)
	}
	key := domain.NewID()
	var previous string
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/v1"+endpoint, strings.NewReader("{}"))
		request.AddCookie(cookie)
		request.Header.Set("Origin", "http://127.0.0.1:8090")
		request.Header.Set("X-CSRF-Token", session.CSRFToken)
		request.Header.Set("Idempotency-Key", key)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		f.handler.ServeHTTP(response, request)
		if response.Code != 202 {
			t.Fatal(response.Code, response.Body.String())
		}
		var result struct {
			ID string `json:"id"`
		}
		must(json.Unmarshal(response.Body.Bytes(), &result))
		if result.ID == "" || previous != "" && previous != result.ID {
			t.Fatal("duplicate request", result)
		}
		previous = result.ID
	}
	for _, format := range []string{"txt", "pdf", "exe"} {
		response := f.request("GET", "/libraries/"+library+"/explorer?format="+format, "", cookie, "")
		if format == "exe" {
			if response.Code != 422 {
				t.Fatal(response.Code)
			}
			continue
		}
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		var result struct {
			Count int `json:"result_count"`
		}
		must(json.Unmarshal(response.Body.Bytes(), &result))
		expected := 0
		if format == "txt" {
			expected = 1
		}
		if result.Count != expected {
			t.Fatal(format, result)
		}
	}
}
