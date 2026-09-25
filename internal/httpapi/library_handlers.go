package httpapi

import (
	"mime"
	"net/http"
	"strconv"
	"strings"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
)

func (server *Server) libraryRoutes(mux *http.ServeMux) {
	server.organizationRoutes(mux)
	server.workflowRoutes(mux)
	routes := map[string]authorizedHandler{
		"GET /api/v1/libraries/{library}": server.getLibrary, "GET /api/v1/jobs/{job}": server.getJob, "POST /api/v1/storage/path-inspections": server.inspectPath,
		"GET /api/v1/libraries/{library}/folders": server.libraryFolders,
		"GET /api/v1/search-events":               server.searchEvents, "GET /api/v1/libraries/{library}/member-candidates": server.memberCandidates,
		"GET /api/v1/libraries": server.listLibraries, "POST /api/v1/libraries": server.createLibrary, "PATCH /api/v1/libraries/{library}": server.updateLibrary,
		"GET /api/v1/libraries/{library}/members": server.libraryMembers, "PUT /api/v1/libraries/{library}/members/{user}": server.setLibraryMember,
		"POST /api/v1/libraries/{library}/root-plans": server.planRoot, "POST /api/v1/libraries/{library}/roots": server.confirmRoot, "POST /api/v1/libraries/{library}/root-consolidations": server.confirmRoot,
		"GET /api/v1/libraries/{library}/roots": server.libraryRoots, "PATCH /api/v1/roots/{root}": server.configureRoot, "POST /api/v1/roots/{root}/retirement-plans": server.retirementPlan, "POST /api/v1/roots/{root}/retire": server.retireRoot,
		"GET /api/v1/libraries/{library}/folder-views": server.libraryViews, "POST /api/v1/libraries/{library}/folder-views": server.addView, "POST /api/v1/libraries/{library}/verify": server.verifyLibrary,
		"GET /api/v1/libraries/{library}/explorer": server.explorer, "POST /api/v1/search": server.search,
		"GET /api/v1/documents/{document}": server.document, "GET /api/v1/documents/{document}/content": server.documentContent, "GET /api/v1/documents/{document}/download": server.documentContent,
		"GET /api/v1/documents/{document}/pages/{page}/text": server.documentText, "GET /api/v1/documents/{document}/history": server.documentHistory, "POST /api/v1/documents/{document}/remove-index": server.removeDocumentIndex,
		"GET /api/v1/libraries/{library}/duplicates": server.duplicates, "GET /api/v1/libraries/{library}/jobs": server.libraryJobs, "POST /api/v1/jobs/{job}/retry": server.retryJob,
		"GET /api/v1/notifications": server.notifications, "POST /api/v1/notifications/{notice}/acknowledge": server.acknowledgeNotice, "GET /api/v1/libraries/{library}/audit-events": server.libraryEvents,
	}
	for route, handler := range routes {
		mux.HandleFunc(route, server.protected("", true, handler))
	}
}
func (server *Server) libraryResult(writer http.ResponseWriter, request *http.Request, status int, result any, err error) {
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	if status == 204 {
		writer.WriteHeader(204)
		return
	}
	writeJSON(writer, status, result)
}
func (server *Server) listLibraries(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Libraries(request.Context(), principal)
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) createLibrary(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Name    string `json:"name"`
		Mode    string `json:"mode"`
		Manager string `json:"manager_user_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.Create(request.Context(), principal, input.Name, input.Mode, input.Manager, request.Header.Get("Idempotency-Key"), metadata(request))
	server.libraryResult(writer, request, 201, map[string]string{"id": identifier}, err)
}
func (server *Server) updateLibrary(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Name      string              `json:"name"`
		Languages string              `json:"ocr_languages"`
		Mode      string              `json:"mode"`
		Settings  *libraries.Settings `json:"settings"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	err = server.libraries.UpdateConfiguration(request.Context(), principal, request.PathValue("library"), input.Name, input.Languages, input.Mode, input.Settings, revision, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) libraryMembers(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Members(request.Context(), principal, request.PathValue("library"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) setLibraryMember(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Roles []string `json:"role_ids"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	err := server.libraries.SetMember(request.Context(), principal, request.PathValue("library"), request.PathValue("user"), input.Roles, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) planRoot(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Path   string `json:"server_path"`
		Source string `json:"storage_source"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	if input.Source != "" && input.Source != "linked" && input.Source != "managed" {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Selecciona un origen válido.", 422))
		return
	}
	if input.Source == "" {
		input.Source = "linked"
	}
	result, err := server.libraries.PlanStorageRoot(request.Context(), principal, request.PathValue("library"), input.Path, input.Source, metadata(request))
	server.libraryResult(writer, request, 201, result, err)
}
func (server *Server) confirmRoot(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Plan     string `json:"plan_id"`
		Revision int64  `json:"expected_configuration_revision"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.ConfirmRoot(request.Context(), principal, request.PathValue("library"), input.Plan, input.Revision, strings.HasSuffix(request.URL.Path, "root-consolidations"), metadata(request))
	server.libraryResult(writer, request, 202, map[string]string{"root_id": identifier}, err)
}
func (server *Server) libraryRoots(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Roots(request.Context(), principal, request.PathValue("library"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) configureRoot(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Enabled  bool `json:"enabled"`
		Interval int  `json:"reconcile_interval_seconds"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	err = server.libraries.ConfigureRoot(request.Context(), principal, request.PathValue("root"), input.Enabled, input.Interval, revision, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) retirementPlan(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.RetirementPlan(request.Context(), principal, request.PathValue("root"))
	server.libraryResult(writer, request, 201, result, err)
}
func (server *Server) retireRoot(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Plan   string `json:"plan_id"`
		Policy string `json:"reference_policy"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	err := server.libraries.Retire(request.Context(), principal, request.PathValue("root"), input.Plan, input.Policy, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) libraryViews(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Views(request.Context(), principal, request.PathValue("library"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) addView(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Root   string `json:"root_id"`
		Prefix string `json:"relative_prefix"`
		Name   string `json:"name"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.AddView(request.Context(), principal, request.PathValue("library"), input.Root, input.Prefix, input.Name, metadata(request))
	server.libraryResult(writer, request, 201, map[string]string{"id": identifier}, err)
}
func (server *Server) verifyLibrary(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Root string `json:"root_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	err := server.libraries.Verify(request.Context(), principal, request.PathValue("library"), input.Root, metadata(request))
	server.libraryResult(writer, request, 202, map[string]string{"status": "queued"}, err)
}
func (server *Server) search(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input libraries.SearchInput
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	result, err := server.libraries.Search(request.Context(), principal, input, metadata(request), true)
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) explorer(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	query := request.URL.Query()
	limit := 50
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Límite inválido.", 422))
			return
		}
		limit = parsed
	}
	input := libraries.SearchInput{Libraries: []string{request.PathValue("library")}, Limit: limit, Cursor: query.Get("cursor"), Filters: libraries.Filters{RootID: query.Get("root_id"), ViewID: query.Get("view_id"), Prefix: query.Get("prefix"), Availability: query.Get("availability"), CaseID: query.Get("case_id"), CategoryID: query.Get("category_id"), TypeID: query.Get("document_type_id"), Exercise: query.Get("exercise"), Source: query.Get("storage_source"), Unassigned: query.Get("unassigned") == "true"}}
	result, err := server.libraries.Search(request.Context(), principal, input, metadata(request), false)
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) document(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Document(request.Context(), principal, request.PathValue("document"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) documentContent(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	download := strings.HasSuffix(request.URL.Path, "/download")
	file, document, err := server.libraries.OpenDocument(request.Context(), principal, request.PathValue("document"), download, metadata(request))
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	writer.Header().Set("Content-Type", "application/pdf")
	writer.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": document.Filename}))
	writer.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; frame-ancestors 'self'")
	writer.Header().Set("X-Frame-Options", "SAMEORIGIN")
	http.ServeContent(writer, request, document.Filename, info.ModTime(), file)
}
func (server *Server) documentText(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	number, err := strconv.Atoi(request.PathValue("page"))
	if err != nil || number < 1 {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Página inválida.", 422))
		return
	}
	result, err := server.libraries.Page(request.Context(), principal, request.PathValue("document"), number)
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) documentHistory(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Events(request.Context(), principal, "", request.PathValue("document"), request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, items, err)
}
func (server *Server) removeDocumentIndex(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Reason string `json:"reason"`
		CaseID string `json:"expected_case_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	err = server.libraries.RemoveIndexConfirmed(request.Context(), principal, request.PathValue("document"), input.Reason, input.CaseID, revision, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) duplicates(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Duplicates(request.Context(), principal, request.PathValue("library"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) libraryJobs(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Jobs(request.Context(), principal, request.PathValue("library"), request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, items, err)
}
func (server *Server) retryJob(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.Retry(request.Context(), principal, request.PathValue("job"), input.Reason, metadata(request))
	server.libraryResult(writer, request, 202, map[string]string{"id": identifier}, err)
}
func (server *Server) notifications(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Notifications(request.Context(), principal)
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) acknowledgeNotice(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	err := server.libraries.Acknowledge(request.Context(), principal, request.PathValue("notice"))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) libraryEvents(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Events(request.Context(), principal, request.PathValue("library"), "", request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, items, err)
}

func (server *Server) searchEvents(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, next, err := server.libraries.SearchEvents(request.Context(), principal, request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items, "next_cursor": next}, err)
}
func (server *Server) memberCandidates(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Candidates(request.Context(), principal, request.PathValue("library"), request.URL.Query().Get("q"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}

func (server *Server) libraryFolders(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	query := request.URL.Query()
	items, next, err := server.libraries.Folders(request.Context(), principal, request.PathValue("library"), query.Get("root_id"), query.Get("view_id"), query.Get("prefix"), query.Get("cursor"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items, "next_cursor": next}, err)
}

func (server *Server) getLibrary(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Libraries(request.Context(), principal)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	for _, library := range items {
		if library.ID == request.PathValue("library") {
			writeJSON(writer, 200, library)
			return
		}
	}
	server.fail(writer, request, domain.Failure("NOT_FOUND", "No se encontró la biblioteca.", 404))
}
func (server *Server) getJob(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	job, err := server.libraries.Job(request.Context(), principal, request.PathValue("job"))
	server.libraryResult(writer, request, 200, job, err)
}
func (server *Server) inspectPath(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Library string `json:"library_id"`
		Path    string `json:"server_path"`
		Source  string `json:"storage_source"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	if input.Source != "linked" && input.Source != "managed" {
		server.fail(writer, request, domain.Failure("INVALID_REQUEST", "Selecciona almacenamiento vinculado o administrado.", 422))
		return
	}
	result, err := server.libraries.PlanStorageRoot(request.Context(), principal, input.Library, input.Path, input.Source, metadata(request))
	server.libraryResult(writer, request, 200, result, err)
}
