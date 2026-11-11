package libraries

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"gestor-documental/internal/audit"
	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensing"
)

type Filters struct {
	Classification string `json:"classification_status"`
	Format         string `json:"format"`
	DirectChildren bool   `json:"direct_children"`
	RootID         string `json:"root_id"`
	ViewID         string `json:"view_id"`
	Prefix         string `json:"prefix"`
	Availability   string `json:"availability"`
	Approval       string `json:"approval_status"`
	CaseID         string `json:"case_id"`
	CategoryID     string `json:"category_id"`
	TypeID         string `json:"document_type_id"`
	Exercise       string `json:"exercise"`
	Source         string `json:"storage_source"`
	Unassigned     bool   `json:"unassigned"`
}
type SearchInput struct {
	SummaryOnly bool     `json:"summary_only,omitempty"`
	SkipCount   bool     `json:"-"`
	Query       string   `json:"query"`
	Type        string   `json:"search_type"`
	Libraries   []string `json:"library_ids"`
	Filters     Filters  `json:"filters"`
	Limit       int      `json:"limit"`
	Cursor      string   `json:"cursor"`
}
type Segment struct {
	Text        string `json:"text"`
	Highlighted bool   `json:"highlighted"`
}
type Match struct {
	Page     int       `json:"page_number"`
	Kind     string    `json:"unit_kind"`
	Label    string    `json:"context_label"`
	Segments []Segment `json:"segments"`
}
type Document struct {
	PreviousClassification *Classification `json:"previous_classification,omitempty"`
	CanRequestCorrection   bool            `json:"can_request_correction"`
	CanReplace             bool            `json:"can_replace"`
	Classification         string          `json:"classification_status"`
	documentformat.Detection
	IndexReason    string            `json:"index_block_reason"`
	Size           *int64            `json:"size_bytes"`
	Created        string            `json:"created_at"`
	Modified       string            `json:"modified_at"`
	LibraryName    string            `json:"library_name,omitempty"`
	Integrity      string            `json:"integrity_status"`
	CategoryName   string            `json:"category_name"`
	TypeName       string            `json:"document_type_name"`
	CanCancel      bool              `json:"can_cancel"`
	CanReindex     bool              `json:"can_reindex"`
	Source         string            `json:"storage_source"`
	CaseID         string            `json:"case_id"`
	CaseIdentifier string            `json:"case_identifier"`
	CategoryID     string            `json:"category_id"`
	TypeID         string            `json:"document_type_id"`
	CreatedBy      string            `json:"created_by"`
	Metadata       map[string]string `json:"metadata"`
	MetadataJSON   string            `json:"-"`
	CanClassify    bool              `json:"can_classify"`
	CanAssociate   bool              `json:"can_associate"`
	CanReassign    bool              `json:"can_reassign"`
	ID             string            `json:"id"`
	LibraryID      string            `json:"library_id"`
	Title          string            `json:"title"`
	Filename       string            `json:"original_filename"`
	Availability   string            `json:"availability"`
	Approval       string            `json:"approval_status"`
	Freshness      string            `json:"extraction_freshness"`
	Revision       int64             `json:"revision"`
	RootID         string            `json:"root_id"`
	RelativePath   string            `json:"relative_path"`
	OriginalPath   string            `json:"original_path,omitempty"`
	FileID         string            `json:"-"`
	Identity       string            `json:"-"`
	Hash           string            `json:"sha256"`
	Pages          int               `json:"page_count"`
	Units          int               `json:"unit_count"`
	FirstTextUnit  int               `json:"first_text_unit,omitempty"`
	Processing     *ProcessingRun    `json:"processing,omitempty"`
	Preview        bool              `json:"can_preview_original"`
	Download       bool              `json:"can_download"`
	Matches        []Match           `json:"matches"`
}
type SearchGroup struct {
	LibraryID string `json:"library_id"`
	Name      string `json:"library_name"`
	Count     int    `json:"result_count"`
}
type SearchResult struct {
	Groups    []SearchGroup `json:"groups"`
	Items     []Document    `json:"items"`
	Count     int           `json:"result_count"`
	Cursor    string        `json:"next_cursor,omitempty"`
	Libraries []string      `json:"consulted_library_ids"`
}

const incompleteClassificationSQL = "(d.category_id IS NULL OR d.document_type_id IS NULL OR (coalesce(json_extract((SELECT settings_json FROM libraries WHERE id=d.library_id),'$.cases_enabled'),0)=1 AND d.case_id IS NULL))"
const classificationStateSQL = "CASE WHEN EXISTS(SELECT 1 FROM document_replacements historical WHERE historical.previous_document_id=d.id AND historical.status='completed') THEN 'historical' WHEN f.availability='staged' THEN 'temporary' WHEN " + incompleteClassificationSQL + " THEN 'unclassified' ELSE 'classified' END"
const documentColumns = classificationStateSQL + ",d.id,d.library_id,d.title,d.original_filename,CASE WHEN r.status='inaccessible' THEN 'unknown' ELSE f.availability END,d.approval_status,f.extraction_freshness,d.revision,coalesce(l.root_id,''),coalesce(l.relative_path,''),coalesce(l.canonical_path,''),f.id,coalesce(f.os_identity_key,''),coalesce(v.sha256,''),coalesce(e.page_count,0),f.storage_source,coalesce(d.case_id,''),coalesce(c.identifier,''),coalesce(d.category_id,''),coalesce(d.document_type_id,''),coalesce(d.created_by,''),d.metadata_json,coalesce(cat.name,''),coalesce(dt.name,''),f.integrity_status,v.size_bytes,d.created_at,coalesce(v.os_modified_at,''),coalesce(v.document_format,'pdf'),coalesce(v.detected_mime,'application/pdf'),coalesce(v.original_extension,'.pdf'),coalesce(v.extension_mismatch,0),coalesce(v.index_block_reason,''),coalesce(nullif(e.unit_count,0),e.page_count,0)"
const documentJoins = " FROM documents d JOIN physical_files f ON f.id=d.physical_file_id LEFT JOIN physical_file_locations l ON l.id=f.primary_location_id LEFT JOIN storage_roots r ON r.id=l.root_id LEFT JOIN content_versions v ON v.id=f.current_content_version_id LEFT JOIN extraction_runs e ON e.id=f.indexed_extraction_id LEFT JOIN cases c ON c.id=d.case_id LEFT JOIN categories cat ON cat.id=d.category_id LEFT JOIN document_types dt ON dt.id=d.document_type_id "

var explorerRootPlanLogged atomic.Bool
var explorerNestedPlanLogged atomic.Bool

func logExplorerPlan(ctx context.Context, tx *sql.Tx, kind, query string, args ...any) {
	rows, err := tx.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		slog.Warn("explorer query plan unavailable", "kind", kind, "error_type", "sql")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			return
		}
		slog.Info("explorer query plan", "kind", kind, "detail", detail)
	}
}

type rowScanner interface{ Scan(...any) error }

func scanDocument(row rowScanner) (Document, error) {
	var document Document
	err := row.Scan(&document.Classification, &document.ID, &document.LibraryID, &document.Title, &document.Filename, &document.Availability, &document.Approval, &document.Freshness, &document.Revision, &document.RootID, &document.RelativePath, &document.OriginalPath, &document.FileID, &document.Identity, &document.Hash, &document.Pages, &document.Source, &document.CaseID, &document.CaseIdentifier, &document.CategoryID, &document.TypeID, &document.CreatedBy, &document.MetadataJSON, &document.CategoryName, &document.TypeName, &document.Integrity, &document.Size, &document.Created, &document.Modified, &document.Format, &document.MIME, &document.Extension, &document.Mismatch, &document.IndexReason, &document.Units)
	document.Matches = []Match{}
	if err == nil {
		err = json.Unmarshal([]byte(document.MetadataJSON), &document.Metadata)
	}
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
	result := SearchResult{Items: []Document{}, Libraries: []string{}, Groups: []SearchGroup{}}
	started := time.Now()
	var countMS, pageMS int64
	defer func() {
		if input.Query == "" && time.Since(started) >= time.Second {
			slog.Warn("explorer timing", "total_ms", time.Since(started).Milliseconds(), "count_ms", countMS, "page_ms", pageMS,
				"root_filter", input.Filters.RootID != "", "prefix_filter", input.Filters.Prefix != "")
		}
	}()
	switch input.Filters.Format {
	case "", "pdf", "docx", "xlsx", "txt", "csv":
	default:
		return result, invalid("Selecciona un formato de archivo válido.")
	}
	if input.Type == "" {
		input.Type = "general"
	}
	if input.Type != "general" && input.Type != "ocr" && input.Type != "name" && input.Type != "identifier" {
		return result, invalid("Tipo de búsqueda inválido.")
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
	if input.Filters.Availability != "" && input.Filters.Availability != "available" && input.Filters.Availability != "missing" && input.Filters.Availability != "unknown" && input.Filters.Availability != "staged" {
		return result, invalid("Disponibilidad inválida.")
	}
	expression, tokens, err := compileQuery(input.Query)
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// License.Status uses its own read transaction. Do not nest it after acquiring
	// a reader: concurrent searches could otherwise exhaust the four-reader pool.
	if err := service.Identity.License.Check(ctx, licensing.ReadDocuments); err != nil {
		return result, err
	}
	transaction, err := service.Database.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, err
	}
	defer transaction.Rollback()
	current, err := service.Identity.LoadPrincipal(ctx, transaction, principal.TokenDigest)
	if err != nil {
		return result, err
	}
	err = func() error {
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
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			authorized[id] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(input.Libraries) > 0 {
			for _, id := range input.Libraries {
				if !authorized[id] {
					return domain.Failure("FORBIDDEN", "No tienes permiso sobre el ámbito solicitado.", 403)
				}
				result.Libraries = append(result.Libraries, id)
			}
		} else {
			for id := range authorized {
				result.Libraries = append(result.Libraries, id)
			}
			sort.Strings(result.Libraries)
		}
		conditions := []string{"d.deleted_at IS NULL", visibleDocumentSQL}
		arguments := []any{current.User.ID, current.User.ID}
		if len(result.Libraries) == 0 {
			conditions = append(conditions, "0=1")
		} else {
			placeholders := []string{}
			for _, id := range result.Libraries {
				placeholders = append(placeholders, "?")
				arguments = append(arguments, id)
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
			if input.Filters.Prefix == "" {
				input.Filters.Prefix = prefix
			} else if prefix != "" && input.Filters.Prefix != prefix && !strings.HasPrefix(input.Filters.Prefix, prefix+"/") {
				return invalid("La carpeta debe pertenecer a la vista seleccionada.")
			}
		}
		if input.Filters.RootID != "" {
			conditions = append(conditions, "l.root_id=?")
			arguments = append(arguments, input.Filters.RootID)
		}
		if input.Filters.Prefix != "" {
			conditions = append(conditions, "(l.relative_path=? OR substr(l.relative_path,1,length(?)+1)=? || '/')")
			arguments = append(arguments, input.Filters.Prefix, input.Filters.Prefix, input.Filters.Prefix)
		}
		if input.Filters.DirectChildren {
			start := 1
			if input.Filters.Prefix != "" {
				start = len([]rune(input.Filters.Prefix)) + 2
			}
			conditions = append(conditions, "instr(substr(coalesce(l.relative_path,d.original_filename),?),'/')=0")
			arguments = append(arguments, start)
		}
		if input.Filters.Availability != "" {
			conditions = append(conditions, "(CASE WHEN r.status='inaccessible' THEN 'unknown' ELSE f.availability END)=?")
			arguments = append(arguments, input.Filters.Availability)
		}
		if input.Filters.Format != "" {
			conditions = append(conditions, "coalesce(v.document_format,'pdf')=?")
			arguments = append(arguments, input.Filters.Format)
		}
		for _, filter := range []struct{ column, value string }{{"d.case_id", input.Filters.CaseID}, {"d.category_id", input.Filters.CategoryID}, {"d.document_type_id", input.Filters.TypeID}, {"c.exercise", input.Filters.Exercise}, {"f.storage_source", input.Filters.Source}, {"d.approval_status", input.Filters.Approval}} {
			if filter.value != "" {
				conditions = append(conditions, filter.column+"=?")
				arguments = append(arguments, filter.value)
			}
		}
		if input.Filters.Classification != "" {
			switch input.Filters.Classification {
			case "temporary", "unclassified", "classified", "historical":
			default:
				return invalid("Filtro de clasificación inválido.")
			}
			conditions = append(conditions, "("+classificationStateSQL+")=?")
			arguments = append(arguments, input.Filters.Classification)
		}
		if input.Filters.Classification != "historical" {
			conditions = append(conditions, "NOT EXISTS(SELECT 1 FROM document_replacements historical WHERE historical.previous_document_id=d.id AND historical.status='completed')")
		}
		if input.Filters.Unassigned {
			conditions = append(conditions, "d.case_id IS NULL")
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
			identifierCondition := "instr(coalesce(c.identifier_key,''),?)>0"
			switch input.Type {
			case "identifier":
				conditions = append(conditions, identifierCondition)
				arguments = append(arguments, searchKey(strings.TrimSpace(input.Query)))
			case "ocr":
				conditions = append(conditions, textCondition)
				arguments = append(arguments, expression)
			case "name":
				conditions = append(conditions, nameCondition)
				arguments = append(arguments, nameArgs...)
			default:
				conditions = append(conditions, "("+textCondition+" OR "+nameCondition+" OR "+identifierCondition+")")
				arguments = append(arguments, expression)
				arguments = append(arguments, nameArgs...)
				arguments = append(arguments, searchKey(strings.TrimSpace(input.Query)))
			}
		}
		scope := domain.Digest(encode([]any{current.User.ID, input.Query, input.Type, result.Libraries, input.Filters}))
		after := ""
		if input.Cursor != "" {
			var cursor struct{ After, Scope string }
			contents, err := base64.RawURLEncoding.DecodeString(input.Cursor)
			if err != nil || json.Unmarshal(contents, &cursor) != nil || cursor.Scope != scope || len(cursor.After) != 36 {
				return invalid("El cursor no corresponde a esta consulta.")
			}
			after = cursor.After
		}
		// The count for a folder used to join every document to its content,
		// extraction, case, category and type before counting. Add only tables
		// referenced by the active filters; the page query below still fetches
		// full details for its at most 50 documents.
		countJoins := " FROM documents d JOIN physical_files f ON f.id=d.physical_file_id"
		nestedRange := input.Query == "" && input.Filters.RootID != "" && input.Filters.Prefix != ""
		if nestedRange {
			// Start with the selected path range, then visit only its primary
			// files and documents. CROSS JOIN preserves the selective loop order.
			countJoins = " FROM physical_file_locations l INDEXED BY physical_locations_explorer CROSS JOIN physical_files f CROSS JOIN documents d"
			conditions = append(conditions, "f.primary_location_id=l.id", "d.physical_file_id=f.id", "l.relative_path>=?", "l.relative_path<?")
			arguments = append(arguments, input.Filters.Prefix, input.Filters.Prefix+"0")
		}
		needLocation := input.Filters.RootID != "" || input.Filters.Prefix != "" || input.Filters.DirectChildren || input.Filters.Availability != ""
		if needLocation && !nestedRange {
			countJoins += " LEFT JOIN physical_file_locations l ON l.id=f.primary_location_id"
		}
		if input.Filters.Availability != "" {
			countJoins += " LEFT JOIN storage_roots r ON r.id=l.root_id"
		}
		if input.Filters.Format != "" {
			countJoins += " LEFT JOIN content_versions v ON v.id=f.current_content_version_id"
		}
		if input.Filters.Exercise != "" || (expression != "" && (input.Type == "identifier" || input.Type == "general")) {
			countJoins += " LEFT JOIN cases c ON c.id=d.case_id"
		}
		where := " WHERE " + strings.Join(conditions, " AND ")
		countStarted := time.Now()
		countQuery := "SELECT d.library_id,(SELECT name FROM libraries WHERE id=d.library_id),count(*)" + countJoins + where + " GROUP BY d.library_id ORDER BY 2,d.library_id"
		pageJoins := documentJoins
		if input.Query == "" && input.Filters.RootID != "" {
			pageJoins = strings.Replace(pageJoins, "documents d JOIN", "documents d INDEXED BY documents_explorer_page JOIN", 1)
		}
		pageQuery := "SELECT " + documentColumns + pageJoins + where + " AND d.id>? ORDER BY d.id LIMIT ?"
		if input.Query == "" && input.Filters.RootID != "" {
			logged := &explorerRootPlanLogged
			kind := "root"
			if input.Filters.Prefix != "" {
				logged, kind = &explorerNestedPlanLogged, "nested"
			}
			if logged.CompareAndSwap(false, true) {
				logExplorerPlan(ctx, transaction, kind+"-count", countQuery, arguments...)
				pageArgs := append(append([]any{}, arguments...), after, input.Limit+1)
				logExplorerPlan(ctx, transaction, kind+"-page", pageQuery, pageArgs...)
			}
		}
		if input.SkipCount && !input.SummaryOnly {
			result.Count = -1 // Exact total was intentionally omitted for page navigation.
		} else {
			groups, err := transaction.QueryContext(ctx, countQuery, arguments...)
			if err != nil {
				return err
			}
			for groups.Next() {
				var group SearchGroup
				if err = groups.Scan(&group.LibraryID, &group.Name, &group.Count); err != nil {
					groups.Close()
					return err
				}
				result.Groups = append(result.Groups, group)
				result.Count += group.Count
			}
			err = groups.Err()
			groups.Close()
			if err != nil {
				return err
			}
			countMS = time.Since(countStarted).Milliseconds()
		}
		if input.SummaryOnly {
			return nil
		}
		pageArgs := append(append([]any{}, arguments...), after, input.Limit+1)
		pageStarted := time.Now()
		rows, err = transaction.QueryContext(ctx, pageQuery, pageArgs...)
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
		pageMS = time.Since(pageStarted).Milliseconds()
		if err != nil {
			return err
		}
		if len(result.Items) > input.Limit {
			result.Items = result.Items[:input.Limit]
			result.Cursor = base64.RawURLEncoding.EncodeToString([]byte(encode(map[string]string{"After": result.Items[len(result.Items)-1].ID, "Scope": scope})))
		}
		permissionsByLibrary := map[string]map[string]bool{}
		for index := range result.Items {
			document := &result.Items[index]
			allowed := permissionsByLibrary[document.LibraryID]
			if allowed == nil {
				permissions, err := service.permissions(ctx, transaction, current.User.ID, document.LibraryID)
				if err != nil {
					return err
				}
				allowed = map[string]bool{}
				for _, permission := range permissions {
					allowed[permission] = true
				}
				permissionsByLibrary[document.LibraryID] = allowed
			}
			available := (document.Availability == "available" || document.Availability == "staged") && document.IndexReason != "content_rejected"
			document.Preview = available && document.Format == "pdf" && allowed["documents.read"]
			document.CanCancel = document.Availability == "staged" && document.CreatedBy == current.User.ID && allowed["documents.cancel_own"] && (document.Approval == "draft" || document.Approval == "rejected")
			document.CanClassify = allowed["documents.classify"]
			document.CanAssociate = allowed["documents.associate"] && document.Source == "linked"
			document.CanReassign = allowed["documents.reassign"]
			document.Download = available && allowed["documents.download"]
			if document.Availability == "staged" || !allowed["storage.view_paths"] {
				document.OriginalPath = ""
			}
			if expression != "" && input.Type != "name" && input.Type != "identifier" {
				matches, err := transaction.QueryContext(ctx, "SELECT p.page_number,snippet(pages_fts,0,char(1),char(2),' … ',24),u.unit_kind,u.context_label FROM pages_fts JOIN indexed_pages p ON p.id=pages_fts.rowid JOIN extraction_pages u ON u.extraction_id=p.extraction_id AND u.page_number=p.page_number WHERE pages_fts MATCH ? AND p.physical_file_id=? ORDER BY p.page_number LIMIT 5", expression, document.FileID)
				if err != nil {
					return err
				}
				for matches.Next() {
					var number int
					var snippet, kind, label string
					if err = matches.Scan(&number, &snippet, &kind, &label); err != nil {
						matches.Close()
						return err
					}
					document.Matches = append(document.Matches, Match{Page: number, Kind: kind, Label: label, Segments: segments(snippet)})
				}
				err = matches.Err()
				matches.Close()
				if err != nil {
					return err
				}
			}
		}
		return nil
	}()
	if err != nil {
		return result, err
	}
	if err = transaction.Commit(); err != nil {
		return result, err
	}
	if auditSearch {
		err = service.Identity.AuthorizedWrite(ctx, principal, "", func(transaction *sql.Tx, current domain.Principal) error {
			for _, libraryID := range result.Libraries {
				if err := service.require(ctx, transaction, current, libraryID, "search.execute"); err != nil {
					return err
				}
			}
			eventID := domain.NewID()
			if err := audit.Append(ctx, transaction, time.Now(), audit.Event{ID: eventID, Type: "search.executed", ActorUserID: current.User.ID, SessionID: current.SessionID, Metadata: metadata, Details: map[string]any{"result_count": result.Count, "returned_count": len(result.Items)}}); err != nil {
				return err
			}
			_, err := transaction.ExecContext(ctx, "INSERT INTO search_audit_details(event_id,exact_query_text,search_type,consulted_library_ids_json,applied_filters_json,result_count) VALUES(?,?,?,?,?,?)", eventID, input.Query, input.Type, encode(result.Libraries), encode(input.Filters), result.Count)
			return err
		})
	}

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
