//go:build !development

package httpapi

import (
	"context"
	"gestor-documental/internal/domain"
	"strings"
	"testing"
)

func TestProductionBuildBlocksDocumentWritesAndKeepsSecurityAvailable(t *testing.T) {
	fixture := newHTTPFixture(t)
	cookie, session := fixture.login(t, "admin", testPassword)
	response := fixture.request("POST", "/search", `{"query":"private"}`, cookie, session.CSRFToken)
	if response.Code != 403 || !strings.Contains(response.Body.String(), "LICENSE_RECOVERY_REQUIRED") {
		t.Fatalf("document gate: %d %s", response.Code, response.Body.String())
	}
	principal, err := fixture.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"linked", "managed", "hybrid"} {
		if _, err = fixture.server.libraries.Create(context.Background(), principal, "Blocked", mode, principal.User.ID, domain.NewID(), domain.RequestMetadata{}); err == nil {
			t.Fatal("production library write accepted", mode)
		}
	}
	runtime, err := fixture.server.libraries.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	runtime.Close()
	response = fixture.request("GET", "/account", "", cookie, "")
	if response.Code != 200 {
		t.Fatal("recovery administration blocked")
	}
}
