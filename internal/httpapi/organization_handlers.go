package httpapi

import (
	"database/sql"
	"io"
	"net/http"
	"strings"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
)

func (server *Server) organizationRoutes(mux *http.ServeMux) {
	routes := map[string]authorizedHandler{
		"GET /api/v1/libraries/{library}/categories": server.catalogList, "POST /api/v1/libraries/{library}/categories": server.catalogSave, "PATCH /api/v1/categories/{entry}": server.catalogSave,
		"GET /api/v1/libraries/{library}/document-types": server.catalogList, "POST /api/v1/libraries/{library}/document-types": server.catalogSave, "PATCH /api/v1/document-types/{entry}": server.catalogSave,
		"GET /api/v1/libraries/{library}/templates": server.templateList, "POST /api/v1/libraries/{library}/templates": server.templateSave, "POST /api/v1/templates/{template}/versions": server.templateSave,
		"GET /api/v1/libraries/{library}/cases": server.caseList, "POST /api/v1/libraries/{library}/cases": server.caseSave, "GET /api/v1/cases/{case}": server.getCase, "PATCH /api/v1/cases/{case}": server.caseSave,
		"GET /api/v1/cases/{case}/requirements": server.caseRequirements, "POST /api/v1/cases/{case}/initialize-requirements": server.initializeCase, "PATCH /api/v1/cases/{case}/requirements/{requirement}": server.updateRequirement,
		"PATCH /api/v1/documents/{document}/classification": server.classifyDocument, "POST /api/v1/documents/{document}/associate": server.classifyDocument, "POST /api/v1/documents/{document}/reassign": server.classifyDocument, "POST /api/v1/documents/{document}/cancel": server.cancelUpload,
		"POST /api/v1/libraries/{library}/naming-preview": server.namingPreview,
		"GET /api/v1/libraries/{library}/upload-batches":  server.uploadBatches, "POST /api/v1/libraries/{library}/upload-batches": server.createBatch, "GET /api/v1/upload-batches/{batch}": server.getBatch, "POST /api/v1/upload-batches/{batch}/files": server.uploadFile,
	}
	for route, handler := range routes {
		mux.HandleFunc(route, server.protected("", true, handler))
	}
}
func catalogKind(request *http.Request) string {
	if strings.Contains(request.URL.Path, "/document-types") {
		return "document-types"
	}
	return "categories"
}
func (server *Server) resourceLibrary(request *http.Request, table, identifier string) (string, error) {
	var libraryID string
	// Table identifiers are handler constants, never request values.
	switch table {
	case "categories", "document_types", "templates", "cases":
	default:
		return "", domain.Failure("NOT_FOUND", "No se encontró el recurso.", 404)
	}
	err := server.libraries.Database.Reader.QueryRowContext(request.Context(), "SELECT library_id FROM "+table+" WHERE id=?", identifier).Scan(&libraryID)
	if err == sql.ErrNoRows {
		return "", domain.Failure("NOT_FOUND", "No se encontró el recurso.", 404)
	}
	return libraryID, err
}
func (server *Server) catalogList(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Catalog(request.Context(), principal, request.PathValue("library"), catalogKind(request))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) catalogSave(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input libraries.CatalogEntry
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	libraryID, identifier, kind := request.PathValue("library"), request.PathValue("entry"), catalogKind(request)
	var revision int64
	var err error
	if identifier != "" {
		table := "categories"
		if kind == "document-types" {
			table = "document_types"
		}
		libraryID, err = server.resourceLibrary(request, table, identifier)
		if err == nil {
			revision, err = expectedRevision(request)
		}
		if err != nil {
			server.fail(writer, request, err)
			return
		}
	}
	result, err := server.libraries.SaveCatalog(request.Context(), principal, libraryID, kind, identifier, input, revision, metadata(request))
	status := 201
	if identifier != "" {
		status = 200
	}
	server.libraryResult(writer, request, status, map[string]string{"id": result}, err)
}
func (server *Server) templateList(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Templates(request.Context(), principal, request.PathValue("library"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) templateSave(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Name         string                       `json:"name"`
		Requirements []libraries.RequirementInput `json:"requirements"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	libraryID, identifier := request.PathValue("library"), request.PathValue("template")
	var revision int64
	var err error
	if identifier != "" {
		libraryID, err = server.resourceLibrary(request, "templates", identifier)
		if err == nil {
			revision, err = expectedRevision(request)
		}
		if err != nil {
			server.fail(writer, request, err)
			return
		}
	}
	result, err := server.libraries.SaveTemplate(request.Context(), principal, libraryID, identifier, input.Name, input.Requirements, revision, metadata(request))
	server.libraryResult(writer, request, 201, map[string]string{"version_id": result}, err)
}
func (server *Server) caseList(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Cases(request.Context(), principal, request.PathValue("library"), request.URL.Query().Get("q"), request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) getCase(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Case(request.Context(), principal, request.PathValue("case"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) caseSave(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Identifier        string `json:"identifier"`
		Exercise          string `json:"exercise"`
		TemplateVersionID string `json:"template_version_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	libraryID, identifier := request.PathValue("library"), request.PathValue("case")
	var revision int64
	var err error
	if identifier != "" {
		libraryID, err = server.resourceLibrary(request, "cases", identifier)
		if err == nil {
			revision, err = expectedRevision(request)
		}
		if err != nil {
			server.fail(writer, request, err)
			return
		}
	}
	result, err := server.libraries.SaveCase(request.Context(), principal, libraryID, identifier, libraries.Case{Identifier: input.Identifier, Exercise: input.Exercise, TemplateVersionID: input.TemplateVersionID}, revision, metadata(request))
	status := 201
	if identifier != "" {
		status = 200
	}
	server.libraryResult(writer, request, status, map[string]string{"id": result}, err)
}
func (server *Server) caseRequirements(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Requirements(request.Context(), principal, request.PathValue("case"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) initializeCase(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Version string `json:"template_version_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	err := server.libraries.InitializeCase(request.Context(), principal, request.PathValue("case"), input.Version, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) updateRequirement(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Count     int  `json:"required_count"`
		Mandatory bool `json:"mandatory"`
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
	err = server.libraries.UpdateRequirement(request.Context(), principal, request.PathValue("case"), request.PathValue("requirement"), input.Count, input.Mandatory, revision, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) classifyDocument(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input libraries.Classification
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	parts := strings.Split(request.URL.Path, "/")
	err = server.libraries.Classify(request.Context(), principal, request.PathValue("document"), parts[len(parts)-1], input, revision, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) cancelUpload(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Reason string `json:"reason"`
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
	err = server.libraries.CancelUpload(request.Context(), principal, request.PathValue("document"), input.Reason, revision, metadata(request))
	server.libraryResult(writer, request, 204, nil, err)
}
func (server *Server) namingPreview(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input libraries.NamingInput
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	result, err := server.libraries.NamingPreview(request.Context(), principal, request.PathValue("library"), input)
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) uploadBatches(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Batches(request.Context(), principal, request.PathValue("library"), request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) createBatch(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		ClientID string `json:"client_batch_id"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.CreateBatch(request.Context(), principal, request.PathValue("library"), input.ClientID, metadata(request))
	server.libraryResult(writer, request, 201, map[string]string{"id": identifier}, err)
}
func (server *Server) getBatch(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Batch(request.Context(), principal, request.PathValue("batch"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) uploadFile(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	controller := http.NewResponseController(writer)
	_ = controller.SetReadDeadline(time.Now().Add(5 * time.Minute))
	defer controller.SetReadDeadline(time.Time{})
	// The server's ordinary 30 s response deadline also includes body receipt.
	// Allow the bounded upload plus PDF validation before sending its result.
	_ = controller.SetWriteDeadline(time.Now().Add(5*time.Minute + time.Duration(server.libraries.Identity.Config.Indexing.PageTimeoutSeconds)*time.Second + 30*time.Second))
	defer controller.SetWriteDeadline(time.Time{})
	bad := func() {
		server.fail(writer, request, domain.Failure("INVALID_UPLOAD", "Envía client_file_id seguido de un único archivo PDF en multipart.", 400))
	}
	request.Body = http.MaxBytesReader(writer, request.Body, (int64(server.libraries.Identity.Config.Indexing.MaximumFileMB)<<20)+65536)
	multipart, err := request.MultipartReader()
	if err != nil {
		bad()
		return
	}
	key, err := multipart.NextPart()
	if err != nil || key.FormName() != "client_file_id" || key.FileName() != "" {
		bad()
		return
	}
	keyBytes, err := io.ReadAll(io.LimitReader(key, 101))
	if err != nil || len(keyBytes) > 100 {
		bad()
		return
	}
	key.Close()
	file, err := multipart.NextPart()
	if err != nil || file.FormName() != "file" || file.FileName() == "" {
		bad()
		return
	}
	defer file.Close()
	finish := func() error {
		_, err := multipart.NextPart()
		if err != io.EOF {
			return domain.Failure("INVALID_UPLOAD", "Cada solicitud debe contener un solo archivo PDF.", 400)
		}
		return nil
	}
	result, err := server.libraries.Upload(request.Context(), principal, request.PathValue("batch"), string(keyBytes), file.FileName(), file, finish, metadata(request))
	server.libraryResult(writer, request, 201, result, err)
}
