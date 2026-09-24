package libraries

import (
	"context"
	"encoding/json"
	"gestor-documental/internal/domain"
)

type SearchEvent struct {
	ID        string   `json:"id"`
	At        string   `json:"occurred_at"`
	Query     string   `json:"exact_query_text"`
	Type      string   `json:"search_type"`
	Libraries []string `json:"library_ids"`
	Filters   Filters  `json:"filters"`
	Count     int      `json:"result_count"`
}

func (service *Service) SearchEvents(ctx context.Context, principal domain.Principal, after string) ([]SearchEvent, string, error) {
	result := []SearchEvent{}
	next := ""
	// Permission checks are part of SQL before pagination: every consulted library must be authorized.
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT s.event_id,a.occurred_at,s.exact_query_text,s.search_type,s.consulted_library_ids_json,s.applied_filters_json,s.result_count FROM search_audit_details s JOIN audit_events a ON a.id=s.event_id WHERE s.event_id>? AND json_array_length(s.consulted_library_ids_json)>0 AND NOT EXISTS(SELECT 1 FROM json_each(s.consulted_library_ids_json) scope WHERE NOT EXISTS(SELECT 1 FROM library_role_assignments r JOIN role_permissions p USING(role_id) WHERE r.user_id=? AND r.library_id=scope.value AND p.permission_key='audit.search_details') OR NOT EXISTS(SELECT 1 FROM library_role_assignments r JOIN role_permissions p USING(role_id) WHERE r.user_id=? AND r.library_id=scope.value AND p.permission_key='audit.read_library')) ORDER BY s.event_id LIMIT 51", after, principal.User.ID, principal.User.ID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var event SearchEvent
		var ids, filters string
		if err = rows.Scan(&event.ID, &event.At, &event.Query, &event.Type, &ids, &filters, &event.Count); err != nil {
			return nil, "", err
		}
		if err = json.Unmarshal([]byte(ids), &event.Libraries); err != nil {
			return nil, "", err
		}
		if err = json.Unmarshal([]byte(filters), &event.Filters); err != nil {
			return nil, "", err
		}
		result = append(result, event)
	}
	if len(result) > 50 {
		result = result[:50]
		next = result[49].ID
	}
	return result, next, rows.Err()
}

type Candidate struct {
	ID       string `json:"id"`
	Name     string `json:"display_name"`
	Username string `json:"username"`
}

func (service *Service) Candidates(ctx context.Context, principal domain.Principal, libraryID, query string) ([]Candidate, error) {
	if !principal.Can("permissions.manage_global") {
		if err := service.Read(ctx, principal, libraryID, "permissions.manage_library"); err != nil {
			return nil, err
		}
	}
	if len(query) > 120 {
		return nil, invalid("El filtro de usuarios es demasiado largo.")
	}
	var exists int
	if err := service.Database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM libraries WHERE id=?", libraryID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, notFound()
	}
	result := []Candidate{}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,display_name,username FROM users WHERE disabled_at IS NULL AND (?='' OR instr(username_key,lower(?))>0 OR instr(lower(display_name),lower(?))>0) ORDER BY username_key LIMIT 50", query, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate Candidate
		if err = rows.Scan(&candidate.ID, &candidate.Name, &candidate.Username); err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, rows.Err()
}
