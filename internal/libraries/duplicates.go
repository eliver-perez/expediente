package libraries

import (
	"context"
	"encoding/hex"
	"gestor-documental/internal/domain"
	"strings"
)

type DuplicatePage struct {
	Page[Duplicate]
	Groups int `json:"groups"`
	Files  int `json:"files"`
}

func (service *Service) DuplicateGroups(ctx context.Context, p domain.Principal, libraryID, cursor string) (DuplicatePage, error) {
	result := DuplicatePage{Page: Page[Duplicate]{Items: []Duplicate{}}}
	if err := service.Read(ctx, p, libraryID, "documents.read"); err != nil {
		return result, err
	}
	scope := p.User.ID + ":duplicates:" + libraryID
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	selection := `SELECT v.sha256 AS hash,count(*) AS copies FROM physical_files f JOIN content_versions v ON v.id=f.current_content_version_id JOIN documents d ON d.physical_file_id=f.id WHERE EXISTS(SELECT 1 FROM library_role_assignments a JOIN role_permissions p USING(role_id) WHERE a.library_id=f.library_id AND a.user_id=? AND p.permission_key='documents.read') AND d.deleted_at IS NULL AND ` + visibleDocumentSQL + ` GROUP BY v.sha256 HAVING count(*)>1 AND sum(f.library_id=?)>0`
	args := []any{p.User.ID, p.User.ID, p.User.ID, libraryID}
	if err = service.Database.Reader.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(copies),0) FROM ("+selection+")", args...).Scan(&result.Groups, &result.Files); err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, "SELECT hash,copies FROM ("+selection+") WHERE hash>? ORDER BY hash LIMIT 26", append(args, after.ID)...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var group Duplicate
		if err = rows.Scan(&group.Hash, &group.Count); err != nil {
			return result, err
		}
		result.Items = append(result.Items, group)
	}
	if len(result.Items) > 25 {
		result.Items = result.Items[:25]
		result.Next = nextListCursor(scope, "", result.Items[24].Hash)
	}
	return result, rows.Err()
}
func (service *Service) DuplicateDocuments(ctx context.Context, p domain.Principal, libraryID, hash, cursor string) (Page[Document], error) {
	result := Page[Document]{Items: []Document{}}
	if err := service.Read(ctx, p, libraryID, "documents.read"); err != nil {
		return result, err
	}
	if len(hash) != 64 {
		return result, invalid("Grupo no válido.")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return result, invalid("Grupo no válido.")
	}
	scope := p.User.ID + ":duplicate-files:" + libraryID + ":" + hash
	after, err := parseListCursor(cursor, scope)
	if err != nil {
		return result, err
	}
	rows, err := service.Database.Reader.QueryContext(ctx, `SELECT d.id FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN content_versions v ON v.id=f.current_content_version_id WHERE EXISTS(SELECT 1 FROM library_role_assignments a JOIN role_permissions p USING(role_id) WHERE a.library_id=f.library_id AND a.user_id=? AND p.permission_key='documents.read') AND v.sha256=? AND d.deleted_at IS NULL AND d.id>? AND `+visibleDocumentSQL+` ORDER BY d.id LIMIT 26`, p.User.ID, strings.ToLower(hash), after.ID, p.User.ID, p.User.ID)
	if err != nil {
		return result, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return result, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(ids) > 25 {
		ids = ids[:25]
		result.Next = nextListCursor(scope, "", ids[24])
	}
	for _, id := range ids {
		document, err := service.Document(ctx, p, id)
		if err != nil {
			return result, err
		}
		if err = service.Database.Reader.QueryRowContext(ctx, "SELECT name FROM libraries WHERE id=?", document.LibraryID).Scan(&document.LibraryName); err != nil {
			return result, err
		}
		result.Items = append(result.Items, document)
	}
	return result, nil
}
