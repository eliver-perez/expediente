package httpapi

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
)

type cursorPosition struct {
	Scope string `json:"scope"`
	At    string `json:"at"`
	ID    string `json:"id"`
}
type page struct {
	Items      []map[string]any `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

func readMaps(rows *sql.Rows) ([]map[string]any, error) {
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		item := map[string]any{}
		for index, column := range columns {
			item[column] = values[index]
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// The SQL expression/column names are fixed by handlers. Only values originate from HTTP.
func (server *Server) readPage(request *http.Request, principal domain.Principal, selection, table, timeColumn, where string, arguments []any) (page, error) {
	result := page{Items: []map[string]any{}}
	limit := 50
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return result, domain.Failure("INVALID_REQUEST", "El límite debe estar entre 1 y 100.", 400)
		}
		limit = parsed
	}
	parameters := request.URL.Query()
	parameters.Del("cursor")
	parameters.Del("limit")
	scope := domain.Digest(principal.User.ID + "\n" + request.URL.Path + "\n" + parameters.Encode())
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		contents, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || len(contents) > 1024 {
			return result, domain.Failure("INVALID_CURSOR", "Cursor no válido.", 400)
		}
		var position cursorPosition
		if err := json.Unmarshal(contents, &position); err != nil || position.Scope != scope {
			return result, domain.Failure("INVALID_CURSOR", "Cursor no válido para estos filtros.", 400)
		}
		where += " AND (" + timeColumn + " < ? OR (" + timeColumn + " = ? AND id < ?))"
		arguments = append(arguments, position.At, position.At, position.ID)
	}
	arguments = append(arguments, limit+1)
	rows, err := server.identity.Database.Reader.QueryContext(request.Context(), "SELECT "+selection+", "+timeColumn+" AS sort_time FROM "+table+" WHERE "+where+" ORDER BY "+timeColumn+" DESC,id DESC LIMIT ?", arguments...)
	if err != nil {
		return result, err
	}
	items, err := readMaps(rows)
	if err != nil {
		return result, err
	}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		contents, err := json.Marshal(cursorPosition{Scope: scope, At: last["sort_time"].(string), ID: last["id"].(string)})
		if err != nil {
			return result, err
		}
		cursor := base64.RawURLEncoding.EncodeToString(contents)
		result.NextCursor = &cursor
	}
	for _, item := range items {
		delete(item, "sort_time")
	}
	result.Items = items
	return result, nil
}

func dateFilters(request *http.Request, column string) (string, []any, error) {
	where := ""
	arguments := []any{}
	for _, bound := range []struct{ name, operator string }{{"from", ">="}, {"to", "<"}} {
		if raw := request.URL.Query().Get(bound.name); raw != "" {
			instant, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return "", nil, domain.Failure("INVALID_REQUEST", "Las fechas deben usar RFC 3339.", 400)
			}
			where += " AND " + column + " " + bound.operator + " ?"
			arguments = append(arguments, domain.Timestamp(instant))
		}
	}
	return where, arguments, nil
}

func (server *Server) users(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.readPage(request, principal, "id", "users", "created_at", "1=1", nil)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	users := []domain.User{}
	for _, item := range result.Items {
		user, err := identity.UserByID(request.Context(), server.identity.Database.Reader, item["id"].(string))
		if err != nil {
			server.fail(writer, request, err)
			return
		}
		users = append(users, user)
	}
	writeJSON(writer, 200, map[string]any{"items": users, "next_cursor": result.NextCursor})
}

func (server *Server) sessions(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	where := "1=1"
	arguments := []any{}
	if request.URL.Path == "/api/v1/account/sessions" {
		where = "user_id=?"
		arguments = append(arguments, principal.User.ID)
	} else if userID := request.URL.Query().Get("user_id"); userID != "" {
		where += " AND user_id=?"
		arguments = append(arguments, userID)
	}
	if reason := request.URL.Query().Get("close_reason"); reason != "" {
		where += " AND close_reason=?"
		arguments = append(arguments, reason)
	}
	dates, values, err := dateFilters(request, "started_at")
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	where += dates
	arguments = append(arguments, values...)
	result, err := server.readPage(request, principal, "id,user_id,started_at,last_activity_at,expires_at,closed_at,close_reason,observed_ip,user_agent", "sessions", "started_at", where, arguments)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	for _, item := range result.Items {
		item["is_current"] = item["id"] == principal.SessionID
	}
	writeJSON(writer, 200, result)
}

func (server *Server) attempts(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	where := "1=1"
	arguments := []any{}
	if outcome := request.URL.Query().Get("outcome"); outcome != "" {
		where += " AND outcome=?"
		arguments = append(arguments, outcome)
	}
	dates, values, err := dateFilters(request, "occurred_at")
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	where += dates
	arguments = append(arguments, values...)
	result, err := server.readPage(request, principal, "id,attempted_identifier,user_id,occurred_at,observed_ip,user_agent,outcome,request_id", "authentication_attempts", "occurred_at", where, arguments)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 200, result)
}

func (server *Server) events(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	where := "library_id IS NULL"
	arguments := []any{}
	for _, field := range []string{"event_type", "actor_user_id"} {
		if value := request.URL.Query().Get(field); value != "" {
			where += " AND " + field + "=?"
			arguments = append(arguments, value)
		}
	}
	dates, values, err := dateFilters(request, "occurred_at")
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	where += dates
	arguments = append(arguments, values...)
	result, err := server.readPage(request, principal, "id,occurred_at,actor_kind,actor_user_id,session_id,observed_ip,request_id,event_type", "audit_events", "occurred_at", where, arguments)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	writeJSON(writer, 200, result)
}

func (server *Server) event(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	rows, err := server.identity.Database.Reader.QueryContext(request.Context(), "SELECT id,occurred_at,actor_kind,actor_user_id,session_id,observed_ip,request_id,event_type,details_json FROM audit_events WHERE id=? AND library_id IS NULL", request.PathValue("id"))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	items, err := readMaps(rows)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	if len(items) == 0 {
		server.fail(writer, request, domain.Failure("NOT_FOUND", "No se encontró el evento.", 404))
		return
	}
	item := items[0]
	var details any
	if err := json.Unmarshal([]byte(item["details_json"].(string)), &details); err != nil {
		server.fail(writer, request, err)
		return
	}
	delete(item, "details_json")
	item["details"] = details
	writeJSON(writer, 200, item)
}

func (server *Server) roles(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	rows, err := server.identity.Database.Reader.QueryContext(request.Context(), "SELECT id,name,scope_kind FROM roles WHERE scope_kind='global' ORDER BY name")
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	items, err := readMaps(rows)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	for _, item := range items {
		rows, err := server.identity.Database.Reader.QueryContext(request.Context(), "SELECT permission_key FROM role_permissions WHERE role_id=? ORDER BY permission_key", item["id"])
		if err != nil {
			server.fail(writer, request, err)
			return
		}
		permissions, err := readMaps(rows)
		if err != nil {
			server.fail(writer, request, err)
			return
		}
		keys := []string{}
		for _, permission := range permissions {
			keys = append(keys, permission["permission_key"].(string))
		}
		item["permissions"] = keys
	}
	writeJSON(writer, 200, map[string]any{"items": items})
}
