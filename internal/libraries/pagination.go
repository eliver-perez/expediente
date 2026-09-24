package libraries

import (
	"encoding/base64"
	"encoding/json"

	"gestor-documental/internal/domain"
)

type Page[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next_cursor,omitempty"`
}
type listCursor struct {
	Time  string
	ID    string
	Scope string
}

func parseListCursor(raw, scope string) (listCursor, error) {
	if raw == "" {
		return listCursor{}, nil
	}
	var cursor listCursor
	contents, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(contents, &cursor) != nil || cursor.Scope != domain.Digest(scope) || len(cursor.ID) > 100 || len(cursor.Time) > 100 {
		return cursor, invalid("El cursor no corresponde a este listado.")
	}
	return cursor, nil
}
func nextListCursor(scope, time, identifier string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(encode(listCursor{Time: time, ID: identifier, Scope: domain.Digest(scope)})))
}
