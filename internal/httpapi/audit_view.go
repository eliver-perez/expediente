package httpapi

import (
	"encoding/json"
	"gestor-documental/internal/domain"
	"net/http"
	"strings"
)

const auditTable = `(SELECT a.*,coalesce(u.display_name,CASE a.actor_kind WHEN 'system' THEN 'Sistema' WHEN 'anonymous' THEN 'Sin sesión' ELSE 'Usuario retirado' END) AS actor_name,coalesce(b.name,'') AS library_name,coalesce(nullif(d.title,''),d.original_filename,'') AS document_name FROM audit_events a LEFT JOIN users u ON u.id=a.actor_user_id LEFT JOIN libraries b ON b.id=a.library_id LEFT JOIN documents d ON d.id=a.document_id)`

func (server *Server) auditScope(r *http.Request, p domain.Principal) (string, []any, error) {
	library := r.PathValue("library")
	if library != "" {
		if err := server.libraries.Read(r.Context(), p, library, "audit.read_library"); err != nil {
			return "", nil, err
		}
		return "library_id=?", []any{library}, nil
	}
	return "library_id IS NULL", []any{}, nil
}
func (server *Server) auditList(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	where, args, err := server.auditScope(r, p)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	for _, field := range []string{"event_type", "actor_user_id", "actor_kind"} {
		if value := r.URL.Query().Get(field); value != "" {
			where += " AND " + field + "=?"
			args = append(args, value)
		}
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 200 {
		server.fail(w, r, domain.Failure("INVALID_REQUEST", "La búsqueda admite hasta 200 caracteres.", 400))
		return
	}
	if query != "" {
		where += " AND (instr(lower(event_type),lower(?))>0 OR instr(lower(actor_name),lower(?))>0 OR instr(lower(details_json),lower(?))>0 OR instr(lower(coalesce(document_id,'')),lower(?))>0)"
		args = append(args, query, query, query, query)
	}
	dates, values, err := dateFilters(r, "occurred_at")
	if err != nil {
		server.fail(w, r, err)
		return
	}
	where += dates
	args = append(args, values...)
	column := "occurred_at"
	switch r.URL.Query().Get("sort") {
	case "actor":
		column = "actor_name"
	case "event":
		column = "event_type"
	case "", "date":
	default:
		server.fail(w, r, domain.Failure("INVALID_REQUEST", "Orden no válido.", 400))
		return
	}
	result, err := server.readPage(r, p, "id,occurred_at,actor_kind,actor_user_id,actor_name,session_id,observed_ip,request_id,event_type,library_id,library_name,document_id,document_name,details_json", auditTable, column, where, args)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	writeJSON(w, 200, result)
}
func (server *Server) auditOptions(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	where, args, err := server.auditScope(r, p)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	rows, err := server.identity.Database.Reader.QueryContext(r.Context(), "SELECT DISTINCT event_type FROM audit_events WHERE "+where+" ORDER BY event_type", args...)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	types, err := readMaps(rows)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	rows, err = server.identity.Database.Reader.QueryContext(r.Context(), "SELECT DISTINCT actor_user_id AS id,actor_name AS name FROM "+auditTable+" WHERE "+where+" AND actor_user_id IS NOT NULL ORDER BY actor_name", args...)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	users, err := readMaps(rows)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"types": types, "users": users})
}
func (server *Server) libraryAuditDetail(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	where, args, err := server.auditScope(r, p)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	args = append(args, r.PathValue("event"))
	rows, err := server.identity.Database.Reader.QueryContext(r.Context(), "SELECT id,occurred_at,actor_kind,actor_user_id,actor_name,event_type,library_id,library_name,document_id,document_name,details_json FROM "+auditTable+" WHERE "+where+" AND id=?", args...)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	items, err := readMaps(rows)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	if len(items) == 0 {
		server.fail(w, r, domain.Failure("NOT_FOUND", "Evento no encontrado.", 404))
		return
	}
	var details any
	err = json.Unmarshal([]byte(items[0]["details_json"].(string)), &details)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	items[0]["details"] = details
	writeJSON(w, 200, items[0])
}
