//go:build development

package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

func TestIndependentGlobalAuditorsAndExistingAssignments(t *testing.T) {
	fixture := newHTTPFixture(t)
	adminCookie, _ := fixture.login(t, "admin", testPassword)
	admin, err := fixture.service.Authenticate(context.Background(), adminCookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if response := fixture.request("GET", "/users", "", adminCookie, ""); response.Code != 200 {
		t.Fatal("administrator lost user management", response.Code)
	}
	for _, scenario := range []struct {
		username       string
		roles          []string
		access, events bool
	}{
		{"access-only", []string{"access_auditor"}, true, false},
		{"events-only", []string{"event_auditor"}, false, true},
		{"both-audits", []string{"access_auditor", "event_auditor"}, true, true},
	} {
		user, err := fixture.service.CreateUser(context.Background(), admin, scenario.username, scenario.username, "Auditor-test-2026", domain.RequestMetadata{})
		if err != nil {
			t.Fatal(err)
		}
		updated, err := fixture.service.SetGlobalRoles(context.Background(), admin, user.ID, user.Revision, scenario.roles, domain.RequestMetadata{})
		if err != nil {
			t.Fatal(err)
		}
		if len(updated.RoleIDs) != len(scenario.roles) {
			t.Fatal("role assignment changed", updated.RoleIDs)
		}
		cookie, session := fixture.login(t, scenario.username, "Auditor-test-2026")
		if slices.Contains(session.User.Permissions, "audit.read_global") != scenario.events || slices.Contains(session.User.Permissions, "sessions.read_all") != scenario.access {
			t.Fatal("permissions mixed", scenario.username, session.User.Permissions)
		}
		for _, check := range []struct {
			path    string
			allowed bool
		}{
			{"/sessions", scenario.access}, {"/authentication-attempts", scenario.access},
			{"/audit-events", scenario.events}, {"/audit-options", scenario.events},
		} {
			response := fixture.request("GET", check.path, "", cookie, "")
			if check.allowed && response.Code != 200 || !check.allowed && response.Code != 403 {
				t.Fatal(scenario.username, check.path, response.Code, response.Body.String())
			}
		}
		if response := fixture.request("GET", "/users", "", cookie, ""); response.Code != 403 {
			t.Fatal("auditor gained write administration", response.Code)
		}
	}
}

func TestExactSearchAuditExistingDetailsAndZeroMatches(t *testing.T) {
	fixture := newHTTPFixture(t)
	cookie, session := fixture.login(t, "admin", testPassword)
	principal, err := fixture.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	library, err := fixture.server.libraries.Create(context.Background(), principal, "Biblioteca de auditoría", "linked", principal.User.ID, domain.NewID(), domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	stamp := domain.Timestamp(time.Now())
	err = fixture.service.Database.Write(context.Background(), func(tx *sql.Tx) error {
		for index := 0; index < 3; index++ {
			file := domain.NewID()
			if _, e := tx.Exec("INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,extraction_freshness,created_at) VALUES(?,?,'linked','available','unknown','none',?)", file, library, stamp); e != nil {
				return e
			}
			if _, e := tx.Exec("INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES(?,?,?,?,?,?)", domain.NewID(), library, file, "Auditoría ejemplo.pdf", "auditoria ejemplo.pdf", stamp); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	query := `{"query":"","search_type":"general","library_ids":["` + library + `"],"filters":{"format":"pdf","availability":"available","storage_source":"linked","unassigned":true},"limit":1}`
	response := fixture.request("POST", "/search", query, cookie, session.CSRFToken)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var found struct {
		Count int   `json:"result_count"`
		Items []any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatal(err)
	}
	if found.Count != 3 || len(found.Items) != 1 {
		t.Fatal("pagination count", found.Count, len(found.Items))
	}
	zero := `{"query":"sin-coincidencias-xyz-2026","search_type":"name","library_ids":["` + library + `"],"filters":{"format":"pdf"},"limit":10}`
	response = fixture.request("POST", "/search", zero, cookie, session.CSRFToken)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"result_count":0`) {
		t.Fatal("zero-result search", response.Code, response.Body.String())
	}
	var count int
	if err := fixture.service.Database.Reader.QueryRow("SELECT count(*) FROM search_audit_details WHERE exact_query_text='sin-coincidencias-xyz-2026'").Scan(&count); err != nil || count != 1 {
		t.Fatal("zero search not audited", count, err)
	}
	var eventID string
	if err := fixture.service.Database.Reader.QueryRow("SELECT s.event_id FROM search_audit_details s WHERE s.exact_query_text='' ORDER BY s.rowid DESC LIMIT 1").Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	response = fixture.request("GET", "/audit-events/"+eventID, "", cookie, "")
	if response.Code != 403 {
		t.Fatal("admin without event role unexpectedly accessed global events", response.Code)
	}
	admin, err := fixture.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.SetGlobalRoles(context.Background(), admin, admin.User.ID, admin.User.Revision, []string{"installation_admin", "event_auditor"}, domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	response = fixture.request("GET", "/audit-events/"+eventID, "", cookie, "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"search_details_json"`) || !strings.Contains(response.Body.String(), "Biblioteca de auditoría") {
		t.Fatal("exact audit detail unavailable", response.Code, response.Body.String())
	}
	var detail map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	var criteria map[string]any
	if err := json.Unmarshal([]byte(detail["search_details_json"].(string)), &criteria); err != nil {
		t.Fatal(err)
	}
	if criteria["type"] != "general" || criteria["query"] != "" || detail["details"].(map[string]any)["result_count"] != float64(3) || detail["details"].(map[string]any)["returned_count"] != float64(1) {
		t.Fatal("search audit counts or criteria", detail)
	}
	filters := criteria["filters"].(map[string]any)
	if filters["format"] != "pdf" || filters["unassigned"] != true {
		t.Fatal("effective filters missing", filters)
	}
}
