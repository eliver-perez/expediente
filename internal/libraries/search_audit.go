package libraries

import (
	"context"
	"encoding/json"
	"gestor-documental/internal/domain"
)

type SearchEvent struct {
	ID           string   `json:"id"`
	At           string   `json:"occurred_at"`
	Query        string   `json:"exact_query_text"`
	Type         string   `json:"search_type"`
	Libraries    []string `json:"library_ids"`
	LibraryNames []string `json:"library_names"`
	Filters      Filters  `json:"filters"`
	Category     string   `json:"category_name"`
	DocumentType string   `json:"document_type_name"`
	CaseName     string   `json:"case_name"`
	Root         string   `json:"root_name"`
	View         string   `json:"view_name"`
	Count        int      `json:"result_count"`
	Returned     *int     `json:"returned_count"`
}

func (service *Service) SearchEvents(ctx context.Context, principal domain.Principal, after string) ([]SearchEvent, string, error) {
	result := []SearchEvent{}
	next := ""
	// Permission checks are part of SQL before pagination: every consulted library must be authorized.
	rows, err := service.Database.Reader.QueryContext(ctx, `SELECT s.event_id,a.occurred_at,s.exact_query_text,s.search_type,s.consulted_library_ids_json,s.applied_filters_json,s.result_count,(((a.details_json)::jsonb #>> '{returned_count}')::bigint),
        (SELECT json_agg(coalesce(l.name,'Biblioteca no disponible')) FROM json_array_elements_text(s.consulted_library_ids_json::json) AS scope(value) LEFT JOIN libraries l ON l.id=scope.value),
        coalesce((SELECT name FROM categories WHERE id=((s.applied_filters_json)::jsonb #>> '{category_id}')),''),
        coalesce((SELECT name FROM document_types WHERE id=((s.applied_filters_json)::jsonb #>> '{document_type_id}')),''),
        coalesce((SELECT identifier FROM cases WHERE id=((s.applied_filters_json)::jsonb #>> '{case_id}')),''),
        coalesce((SELECT canonical_path FROM storage_roots WHERE id=((s.applied_filters_json)::jsonb #>> '{root_id}')),''),
        coalesce((SELECT name FROM logical_folder_views WHERE id=((s.applied_filters_json)::jsonb #>> '{view_id}')),'')
        FROM search_audit_details s JOIN audit_events a ON a.id=s.event_id WHERE s.event_id>? AND jsonb_array_length((s.consulted_library_ids_json)::jsonb)>0 AND NOT EXISTS(SELECT 1 FROM json_array_elements_text(s.consulted_library_ids_json::json) AS scope(value) WHERE NOT EXISTS(SELECT 1 FROM library_role_assignments r JOIN role_permissions p USING(role_id) WHERE r.user_id=? AND r.library_id=scope.value AND p.permission_key='audit.search_details') OR NOT EXISTS(SELECT 1 FROM library_role_assignments r JOIN role_permissions p USING(role_id) WHERE r.user_id=? AND r.library_id=scope.value AND p.permission_key='audit.read_library')) ORDER BY s.event_id LIMIT 51`, after, principal.User.ID, principal.User.ID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var event SearchEvent
		var ids, filters, names string
		if err = rows.Scan(&event.ID, &event.At, &event.Query, &event.Type, &ids, &filters, &event.Count, &event.Returned, &names, &event.Category, &event.DocumentType, &event.CaseName, &event.Root, &event.View); err != nil {
			return nil, "", err
		}
		if err = json.Unmarshal([]byte(ids), &event.Libraries); err != nil {
			return nil, "", err
		}
		if err = json.Unmarshal([]byte(names), &event.LibraryNames); err != nil {
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
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT id,display_name,username FROM users WHERE disabled_at IS NULL AND (?='' OR strpos(username_key,lower(?))>0 OR strpos(lower(display_name),lower(?))>0) ORDER BY username_key LIMIT 50", query, query, query)
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
