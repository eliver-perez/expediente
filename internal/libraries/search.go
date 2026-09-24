package libraries

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

type Filters struct {
	RootID       string `json:"root_id"`
	ViewID       string `json:"view_id"`
	Prefix       string `json:"prefix"`
	Availability string `json:"availability"`
	Approval     string `json:"approval_status"`
}
type SearchInput struct {
	Query     string   `json:"query"`
	Type      string   `json:"search_type"`
	Libraries []string `json:"library_ids"`
	Filters   Filters  `json:"filters"`
	Limit     int      `json:"limit"`
	Cursor    string   `json:"cursor"`
}
type Segment struct {
	Text        string `json:"text"`
	Highlighted bool   `json:"highlighted"`
}
type Match struct {
	Page     int       `json:"page_number"`
	Segments []Segment `json:"segments"`
}
type Document struct {
	ID           string  `json:"id"`
	LibraryID    string  `json:"library_id"`
	Title        string  `json:"title"`
	Filename     string  `json:"original_filename"`
	Availability string  `json:"availability"`
	Approval     string  `json:"approval_status"`
	Freshness    string  `json:"extraction_freshness"`
	Revision     int64   `json:"revision"`
	RootID       string  `json:"root_id"`
	RelativePath string  `json:"relative_path"`
	OriginalPath string  `json:"original_path,omitempty"`
	FileID       string  `json:"-"`
	Identity     string  `json:"-"`
	Hash         string  `json:"sha256"`
	Pages        int     `json:"page_count"`
	Preview      bool    `json:"can_preview_original"`
	Download     bool    `json:"can_download"`
	Matches      []Match `json:"matches"`
}
type SearchResult struct {
	Items     []Document `json:"items"`
	Count     int        `json:"result_count"`
	Cursor    string     `json:"next_cursor,omitempty"`
	Libraries []string   `json:"consulted_library_ids"`
}

const documentColumns = "d.id,d.library_id,d.title,d.original_filename,CASE WHEN r.status='inaccessible' THEN 'unknown' ELSE f.availability END,d.approval_status,f.extraction_freshness,d.revision,l.root_id,l.relative_path,l.canonical_path,f.id,coalesce(f.os_identity_key,''),coalesce(v.sha256,''),coalesce(e.page_count,0)"
const documentJoins = " FROM documents d JOIN physical_files f ON f.id=d.physical_file_id JOIN physical_file_locations l ON l.id=f.primary_location_id JOIN storage_roots r ON r.id=l.root_id LEFT JOIN content_versions v ON v.id=f.current_content_version_id LEFT JOIN extraction_runs e ON e.id=f.indexed_extraction_id "

type rowScanner interface{ Scan(...any) error }

func scanDocument(row rowScanner) (Document, error) {
	var document Document
	err := row.Scan(&document.ID, &document.LibraryID, &document.Title, &document.Filename, &document.Availability, &document.Approval, &document.Freshness, &document.Revision, &document.RootID, &document.RelativePath, &document.OriginalPath, &document.FileID, &document.Identity, &document.Hash, &document.Pages)
	document.Matches = []Match{}
	return document, err
}
func searchKey(value string) string {
	return strings.Map(func(character rune) rune {
		character = unicode.ToLower(character)
		switch character {
		case 'á', 'à', 'ä', 'â':
			return 'a'
		case 'é', 'è', 'ë', 'ê':
			return 'e'
		case 'í', 'ì', 'ï', 'î':
			return 'i'
		case 'ó', 'ò', 'ö', 'ô':
			return 'o'
		case 'ú', 'ù', 'ü', 'û':
			return 'u'
		case 'ñ':
			return 'n'
		}
		return character
	}, value)
}
func compileQuery(query string) (string, []string, error) {
	if utf8.RuneCountInString(query) > 2000 {
		return "", nil, invalid("La consulta admite hasta 2000 caracteres.")
	}
	tokens := []string{}
	var current strings.Builder
	quoted := false
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, character := range query {
		if character == '"' {
			if quoted {
				flush()
			}
			quoted = !quoted
			continue
		}
		if unicode.IsSpace(character) && !quoted {
			flush()
		} else {
			current.WriteRune(character)
		}
	}
	if quoted {
		return "", nil, invalid("Cierra las comillas de la frase buscada.")
	}
	flush()
	if len(tokens) > 32 {
		return "", nil, invalid("La consulta admite hasta 32 términos o frases.")
	}
	expressions := []string{}
	for _, token := range tokens {
		expressions = append(expressions, "\""+strings.ReplaceAll(token, "\"", "\"\"")+"\"")
	}
	return strings.Join(expressions, " AND "), tokens, nil
}
func (service *Service) Search(ctx context.Context, principal domain.Principal, input SearchInput, metadata domain.RequestMetadata, auditSearch bool) (SearchResult, error) {
	result := SearchResult{Items: []Document{}, Libraries: []string{}}
	if input.Type == "" {
		input.Type = "general"
	}
	if input.Type != "general" && input.Type != "ocr" && input.Type != "name" {
		return result, invalid("El tipo de búsqueda no está disponible en bibliotecas vinculadas.")
	}
	if input.Limit == 0 {
		input.Limit = 50
	}
	if input.Limit < 1 || input.Limit > 100 || len(input.Libraries) > 100 {
		return result, invalid("Límite o ámbito de búsqueda inválido.")
	}
	if input.Filters.Prefix != "" && !relativeSafe(input.Filters.Prefix) {
		return result, invalid("Prefijo relativo inválido.")
	}
	if input.Filters.Availability != "" && input.Filters.Availability != "available" && input.Filters.Availability != "missing" && input.Filters.Availability != "unknown" {
		return result, invalid("Disponibilidad inválida.")
	}
	expression, tokens, err := compileQuery(input.Query)
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
		if err := licensing.Check(licensing.ReadDocuments); err != nil {
			return err
		}
		permission := "search.execute"
		if !auditSearch {
			permission = "documents.read"
		}
		rows, err := transaction.QueryContext(ctx, "SELECT DISTINCT library_id FROM library_role_assignments a JOIN role_permissions p USING(role_id) JOIN libraries l ON l.id=a.library_id WHERE user_id=? AND permission_key=? AND l.disabled_at IS NULL ORDER BY library_id", current.User.ID, permission)
		if err != nil {
			return err
		}
		authorized := map[string]bool{}
		for rows.Next() {
			var identifier string
			if err = rows.Scan(&identifier); err != nil {
				rows.Close()
				return err
			}
			authorized[identifier] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(input.Libraries) > 0 {
			for _, identifier := range input.Libraries {
				if !authorized[identifier] {
					return domain.Failure("FORBIDDEN", "No tienes permiso sobre el ámbito solicitado.", 403)
				}
			}
			result.Libraries = append(result.Libraries, input.Libraries...)
		} else {
			for identifier := range authorized {
				result.Libraries = append(result.Libraries, identifier)
			}
			sort.Strings(result.Libraries)
		}
		conditions := []string{"d.deleted_at IS NULL"}
		arguments := []any{}
		if len(result.Libraries) == 0 {
			conditions = append(conditions, "0=1")
		} else {
			placeholders := []string{}
			for _, identifier := range result.Libraries {
				placeholders = append(placeholders, "?")
				arguments = append(arguments, identifier)
			}
			conditions = append(conditions, "d.library_id IN ("+strings.Join(placeholders, ",")+")")
		}
		if input.Filters.ViewID != "" {
			var libraryID, rootID, prefix string
			err := transaction.QueryRowContext(ctx, "SELECT library_id,root_id,relative_prefix FROM logical_folder_views WHERE id=?", input.Filters.ViewID).Scan(&libraryID, &rootID, &prefix)
			if err == sql.ErrNoRows {
				return notFound()
			}
			if err != nil {
				return err
			}
			if !authorized[libraryID] {
				return notFound()
			}
			input.Filters.RootID = rootID
			input.Filters.Prefix = prefix
		}
		if input.Filters.RootID != "" {
			conditions = append(conditions, "l.root_id=?")
			arguments = append(arguments, input.Filters.RootID)
		}
		if input.Filters.Prefix != "" {
			conditions = append(conditions, "(l.relative_path=? OR substr(l.relative_path,1,length(?)+1)=? || '/')")
			arguments = append(arguments, input.Filters.Prefix, input.Filters.Prefix, input.Filters.Prefix)
		}
		if input.Filters.Availability != "" {
			conditions = append(conditions, "(CASE WHEN r.status='inaccessible' THEN 'unknown' ELSE f.availability END)=?")
			arguments = append(arguments, input.Filters.Availability)
		}
		if input.Filters.Approval != "" {
			conditions = append(conditions, "d.approval_status=?")
			arguments = append(arguments, input.Filters.Approval)
		}
		if expression != "" {
			textCondition := "f.id IN (SELECT p.physical_file_id FROM indexed_pages p JOIN pages_fts ON pages_fts.rowid=p.id WHERE pages_fts MATCH ?)"
			names := []string{}
			nameArgs := []any{}
			for _, token := range tokens {
				names = append(names, "instr(d.filename_search_key,?)>0")
				nameArgs = append(nameArgs, searchKey(token))
			}
			nameCondition := "(" + strings.Join(names, " AND ") + ")"
			if input.Type == "ocr" {
				conditions = append(conditions, textCondition)
				arguments = append(arguments, expression)
			} else if input.Type == "name" {
				conditions = append(conditions, nameCondition)
				arguments = append(arguments, nameArgs...)
			} else {
				conditions = append(conditions, "("+textCondition+" OR "+nameCondition+")")
				arguments = append(arguments, expression)
				arguments = append(arguments, nameArgs...)
			}
		}
		scope := domain.Digest(encode([]any{current.User.ID, input.Query, input.Type, result.Libraries, input.Filters}))
		after := ""
		if input.Cursor != "" {
			var cursor struct {
				After string
				Scope string
			}
			contents, err := base64.RawURLEncoding.DecodeString(input.Cursor)
			if err != nil || json.Unmarshal(contents, &cursor) != nil || cursor.Scope != scope || len(cursor.After) != 36 {
				return invalid("El cursor no corresponde a esta consulta.")
			}
			after = cursor.After
		}
		where := " WHERE " + strings.Join(conditions, " AND ")
		if err = transaction.QueryRowContext(ctx, "SELECT count(*)"+documentJoins+where, arguments...).Scan(&result.Count); err != nil {
			return err
		}
		pageArgs := append(append([]any{}, arguments...), after, input.Limit+1)
		rows, err = transaction.QueryContext(ctx, "SELECT "+documentColumns+documentJoins+where+" AND d.id>? ORDER BY d.id LIMIT ?", pageArgs...)
		if err != nil {
			return err
		}
		for rows.Next() {
			document, err := scanDocument(rows)
			if err != nil {
				rows.Close()
				return err
			}
			result.Items = append(result.Items, document)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(result.Items) > input.Limit {
			result.Items = result.Items[:input.Limit]
			result.Cursor = base64.RawURLEncoding.EncodeToString([]byte(encode(map[string]string{"After": result.Items[len(result.Items)-1].ID, "Scope": scope})))
		}
		for index := range result.Items {
			document := &result.Items[index]
			permissions, err := service.permissions(ctx, transaction, current.User.ID, document.LibraryID)
			if err != nil {
				return err
			}
			allowed := map[string]bool{}
			for _, permission := range permissions {
				allowed[permission] = true
			}
			document.Preview = document.Availability == "available" && allowed["documents.read"]
			document.Download = document.Availability == "available" && allowed["documents.download"]
			if !allowed["storage.view_paths"] {
				document.OriginalPath = ""
			}
			if expression != "" && input.Type != "name" {
				matches, err := transaction.QueryContext(ctx, "SELECT p.page_number,snippet(pages_fts,0,char(1),char(2),' … ',24) FROM pages_fts JOIN indexed_pages p ON p.id=pages_fts.rowid WHERE pages_fts MATCH ? AND p.physical_file_id=? ORDER BY p.page_number LIMIT 5", expression, document.FileID)
				if err != nil {
					return err
				}
				for matches.Next() {
					var number int
					var snippet string
					if err = matches.Scan(&number, &snippet); err != nil {
						matches.Close()
						return err
					}
					document.Matches = append(document.Matches, Match{Page: number, Segments: segments(snippet)})
				}
				err = matches.Err()
				matches.Close()
				if err != nil {
					return err
				}
			}
		}
		if !auditSearch {
			return nil
		}
		eventID := domain.NewID()
		if err = audit.Append(ctx, transaction, time.Now(), audit.Event{ID: eventID, Type: "search.executed", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata, Details: map[string]any{"result_count": result.Count, "returned_count": len(result.Items)}}); err != nil {
			return err
		}
		_, err = transaction.ExecContext(ctx, "INSERT INTO search_audit_details(event_id,exact_query_text,search_type,consulted_library_ids_json,applied_filters_json,result_count) VALUES(?,?,?,?,?,?)", eventID, input.Query, input.Type, encode(result.Libraries), encode(input.Filters), result.Count)
		return err
	})
	return result, err
}
func segments(value string) []Segment {
	result := []Segment{}
	highlighted := false
	var text strings.Builder
	for _, character := range value {
		if character == 1 || character == 2 {
			if text.Len() > 0 {
				result = append(result, Segment{Text: text.String(), Highlighted: highlighted})
				text.Reset()
			}
			highlighted = character == 1
		} else {
			text.WriteRune(character)
		}
	}
	if text.Len() > 0 {
		result = append(result, Segment{Text: text.String(), Highlighted: highlighted})
	}
	return result
}
