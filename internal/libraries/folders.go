package libraries

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"gestor-documental/internal/domain"
)

type Folder struct {
	Name   string `json:"name"`
	Prefix string `json:"relative_prefix"`
}

func (service *Service) Folders(ctx context.Context, principal domain.Principal, libraryID, rootID, viewID, prefix, cursor string) ([]Folder, string, error) {
	folders := []Folder{}
	if err := service.Read(ctx, principal, libraryID, "documents.read"); err != nil {
		return nil, "", err
	}
	if prefix != "" && !relativeSafe(prefix) {
		return nil, "", invalid("Prefijo relativo inválido.")
	}
	if viewID != "" {
		err := service.Database.Reader.QueryRowContext(ctx, "SELECT root_id,relative_prefix FROM logical_folder_views WHERE id=? AND library_id=?", viewID, libraryID).Scan(&rootID, &prefix)
		if err == sql.ErrNoRows {
			return nil, "", notFound()
		}
		if err != nil {
			return nil, "", err
		}
	}
	scope := domain.Digest(encode([]string{principal.User.ID, libraryID, rootID, viewID, prefix}))
	after := ""
	if cursor != "" {
		var value struct {
			After string
			Scope string
		}
		contents, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || json.Unmarshal(contents, &value) != nil || value.Scope != scope || len(value.After) > 1024 {
			return nil, "", invalid("Cursor de carpetas inválido.")
		}
		after = value.After
	}
	start := 1
	if prefix != "" {
		start = len([]rune(prefix)) + 2
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT DISTINCT substr(l.relative_path,?,instr(substr(l.relative_path,?),'/')-1) AS folder FROM physical_file_locations l JOIN physical_files f ON f.primary_location_id=l.id JOIN documents d ON d.physical_file_id=f.id WHERE d.library_id=? AND d.deleted_at IS NULL AND (?='' OR l.root_id=?) AND (?='' OR substr(l.relative_path,1,length(?)+1)=? || '/') AND instr(substr(l.relative_path,?),'/')>0 AND substr(l.relative_path,?,instr(substr(l.relative_path,?),'/')-1)>? ORDER BY folder LIMIT 51", start, start, libraryID, rootID, rootID, prefix, prefix, prefix, start, start, start, after)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, "", err
		}
		relative := name
		if prefix != "" {
			relative = prefix + "/" + name
		}
		folders = append(folders, Folder{Name: name, Prefix: relative})
	}
	next := ""
	if len(folders) > 50 {
		folders = folders[:50]
		next = base64.RawURLEncoding.EncodeToString([]byte(encode(map[string]string{"After": folders[49].Name, "Scope": scope})))
	}
	return folders, next, rows.Err()
}
