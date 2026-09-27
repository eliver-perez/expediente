//go:build development

package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"net/url"
	"testing"
	"time"
)

func TestAuditCombinedFiltersSortingAndScope(t *testing.T) {
	fixture := newHTTPFixture(t)
	cookie, session := fixture.login(t, "admin", testPassword)
	principal, err := fixture.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	library, err := fixture.server.libraries.Create(context.Background(), principal, "Audit visible", "linked", principal.User.ID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	err = fixture.service.Database.Write(context.Background(), func(tx *sql.Tx) error {
		for index := 0; index < 30; index++ {
			if err := audit.Append(context.Background(), tx, time.Now(), audit.Event{ID: domain.NewID(), Type: "document.discovered", LibraryID: library, SystemActor: true, Metadata: domain.RequestMetadata{RequestID: domain.NewID()}, Details: map[string]any{"name": fmt.Sprintf("review-%02d", index)}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	path := "/libraries/" + library + "/audit-events?event_type=document.discovered&q=review-&direction=asc&limit=10"
	response := fixture.request("GET", path, "", cookie, session.CSRFToken)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var first page
	if err = json.Unmarshal(response.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 10 || first.NextCursor == nil {
		t.Fatalf("bad page %+v", first)
	}
	second := fixture.request("GET", path+"&cursor="+url.QueryEscape(*first.NextCursor), "", cookie, "")
	if second.Code != 200 {
		t.Fatal(second.Code, second.Body.String())
	}
	var next page
	_ = json.Unmarshal(second.Body.Bytes(), &next)
	if next.Items[0]["id"] == first.Items[0]["id"] {
		t.Fatal("cursor duplicated records")
	}
	mismatched := fixture.request("GET", path+"&sort=actor&cursor="+url.QueryEscape(*first.NextCursor), "", cookie, "")
	if mismatched.Code != 400 {
		t.Fatal("cursor crossed filter scope")
	}
	global := fixture.request("GET", "/audit-events?q=review-", "", cookie, "")
	var globalPage page
	_ = json.Unmarshal(global.Body.Bytes(), &globalPage)
	if len(globalPage.Items) != 0 {
		t.Fatal("global audit leaked library records")
	}
	denied := fixture.request("GET", "/libraries/unassigned/audit-options", "", cookie, "")
	if denied.Code != 404 {
		t.Fatal("audit options authorization", denied.Code)
	}
	filtered := fixture.request("GET", path+"&actor_user_id="+principal.User.ID, "", cookie, "")
	var filteredPage page
	_ = json.Unmarshal(filtered.Body.Bytes(), &filteredPage)
	if len(filteredPage.Items) != 0 {
		t.Fatal("actor filter ignored")
	}
}
