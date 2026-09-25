//go:build development

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
)

func TestH5WorkflowHTTPAuthorizationPublicationAndRanges(t *testing.T) {
	fixture := newHTTPFixture(t)
	ctx := context.Background()
	cookie, session := fixture.login(t, "admin", testPassword)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	principal, err := fixture.service.Authenticate(ctx, cookie.Value, false, domain.RequestMetadata{})
	must(err)
	service := fixture.server.libraries
	library, err := service.Create(ctx, principal, "Workflow HTTP", "managed", principal.User.ID, domain.NewID(), domain.RequestMetadata{})
	must(err)
	destination, err := filepath.EvalSymlinks(t.TempDir())
	must(err)
	plan, err := service.PlanStorageRoot(ctx, principal, library, destination, "managed", domain.RequestMetadata{})
	must(err)
	root, err := service.ConfirmRoot(ctx, principal, library, plan.ID, plan.Revision, false, domain.RequestMetadata{})
	must(err)
	all, err := service.Libraries(ctx, principal)
	must(err)
	settings := all[0].Settings
	settings.ManagedRootID = root
	settings.CasesEnabled = false
	settings.StructurePattern = "{categoria}"
	must(service.UpdateConfiguration(ctx, principal, library, "Workflow HTTP", "spa", "managed", &settings, all[0].Revision, domain.RequestMetadata{}))
	category, err := service.SaveCatalog(ctx, principal, library, "categories", "", libraries.CatalogEntry{Name: "Planos"}, 0, domain.RequestMetadata{})
	must(err)
	kind, err := service.SaveCatalog(ctx, principal, library, "document-types", "", libraries.CatalogEntry{Name: "Plano", CategoryID: category, Multiple: true}, 0, domain.RequestMetadata{})
	must(err)
	batch, err := service.CreateBatch(ctx, principal, library, domain.NewID(), domain.RequestMetadata{})
	must(err)
	contents, err := os.ReadFile("../../testdata/documents/native.pdf")
	must(err)
	item, err := service.Upload(ctx, principal, batch, domain.NewID(), "Prueba.pdf", bytes.NewReader(contents), nil, domain.RequestMetadata{})
	must(err)
	must(service.Classify(ctx, principal, item.DocumentID, "classification", libraries.Classification{CategoryID: category, TypeID: kind}, 1, domain.RequestMetadata{}))
	document, err := service.Document(ctx, principal, item.DocumentID)
	must(err)
	user, err := fixture.service.CreateUser(ctx, principal, "h5-http-reader", "Reader", "abcdef", domain.RequestMetadata{})
	must(err)
	must(service.SetMember(ctx, principal, library, user.ID, []string{"library_reader"}, domain.RequestMetadata{}))
	readerCookie, readerSession := fixture.login(t, user.Username, "abcdef")
	endpoint := "/documents/" + document.ID
	if response := fixture.request("GET", endpoint+"/workflow", "", readerCookie, ""); response.Code != 404 {
		t.Fatal("reader saw private workflow", response.Code)
	}
	send := func(caller *http.Cookie, csrf string, revision int64) *httptest.ResponseRecorder {
		request := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/v1"+endpoint+"/finalize", strings.NewReader("{}"))
		request.AddCookie(caller)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://127.0.0.1:8090")
		request.Header.Set("X-CSRF-Token", csrf)
		request.Header.Set("If-Match", fmt.Sprintf("\"%d\"", revision))
		response := httptest.NewRecorder()
		fixture.handler.ServeHTTP(response, request)
		return response
	}
	if response := send(cookie, "", document.Revision); response.Code != 403 {
		t.Fatal("finalization bypassed CSRF", response.Code)
	}
	if response := send(readerCookie, readerSession.CSRFToken, document.Revision); response.Code != 404 {
		t.Fatal("reader finalized private upload", response.Code)
	}
	response := send(cookie, session.CSRFToken, document.Revision)
	if response.Code != 202 {
		t.Fatal(response.Code, response.Body.String())
	}
	var result struct {
		ID string `json:"operation_id"`
	}
	must(json.Unmarshal(response.Body.Bytes(), &result))
	if replay := send(cookie, session.CSRFToken, document.Revision); replay.Code != 202 || replay.Body.String() != response.Body.String() {
		t.Fatal("HTTP replay created another operation", replay.Code, replay.Body.String())
	}
	if response := fixture.request("GET", "/materializations/"+result.ID, "", readerCookie, ""); response.Code != 404 {
		t.Fatal("reader saw private journal")
	}
	operation := fixture.request("GET", "/materializations/"+result.ID, "", cookie, "")
	if operation.Code != 200 || strings.Contains(operation.Body.String(), destination) || strings.Contains(operation.Body.String(), "relative_path") || strings.Contains(operation.Body.String(), "temporary_locator") {
		t.Fatal("journal paths disclosed", operation.Body.String())
	}
	runtime, err := service.Start(ctx)
	must(err)
	defer runtime.Close()
	deadline := time.Now().Add(15 * time.Second)
	for {
		document, err = service.Document(ctx, principal, item.DocumentID)
		must(err)
		if document.Approval == "approved" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("materialization did not complete")
		}
		time.Sleep(25 * time.Millisecond)
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/v1"+endpoint+"/content", nil)
	request.AddCookie(readerCookie)
	request.Header.Set("Range", "bytes=0-4")
	response = httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	if response.Code != 206 || response.Body.String() != "%PDF-" {
		t.Fatal("definitive reader range", response.Code, response.Body.String())
	}
	public := fixture.request("GET", endpoint, "", readerCookie, "")
	if public.Code != 200 || strings.Contains(public.Body.String(), destination) {
		t.Fatal("public content permission/path policy", public.Body.String())
	}
	denied := fixture.request("POST", "/materializations/"+result.ID+"/retry", `{"reason":"No autorizado"}`, readerCookie, readerSession.CSRFToken)
	if denied.Code != 404 {
		t.Fatal("reader retried journal", denied.Code)
	}
}
