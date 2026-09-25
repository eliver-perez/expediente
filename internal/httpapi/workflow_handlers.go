package httpapi

import (
	"gestor-documental/internal/domain"
	"net/http"
	"strings"
)

func (server *Server) workflowRoutes(mux *http.ServeMux) {
	routes := map[string]authorizedHandler{
		"GET /api/v1/documents/{document}/workflow":       server.documentWorkflow,
		"POST /api/v1/documents/{document}/submit":        server.submitReview,
		"POST /api/v1/documents/{document}/finalize":      server.finalizeDocument,
		"GET /api/v1/libraries/{library}/reviews":         server.pendingReviews,
		"GET /api/v1/libraries/{library}/reviewers":       server.libraryReviewers,
		"GET /api/v1/reviews/pending":                     server.pendingReviews,
		"POST /api/v1/reviews/{review}/approve":           server.decideReview,
		"POST /api/v1/reviews/{review}/reject":            server.decideReview,
		"GET /api/v1/materializations/{operation}":        server.materialization,
		"POST /api/v1/materializations/{operation}/retry": server.retryMaterialization,
	}
	for route, handler := range routes {
		mux.HandleFunc(route, server.protected("", true, handler))
	}
}
func (server *Server) documentWorkflow(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Workflow(request.Context(), principal, request.PathValue("document"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) submitReview(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		ReviewerID string `json:"reviewer_id"`
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
	identifier, err := server.libraries.SubmitReview(request.Context(), principal, request.PathValue("document"), input.ReviewerID, revision, metadata(request))
	server.libraryResult(writer, request, 201, map[string]string{"review_id": identifier}, err)
}
func (server *Server) finalizeDocument(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	revision, err := expectedRevision(request)
	if err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.FinalizeDocument(request.Context(), principal, request.PathValue("document"), revision, metadata(request))
	status := 200
	if identifier != "" {
		status = 202
	}
	server.libraryResult(writer, request, status, map[string]string{"operation_id": identifier}, err)
}
func (server *Server) pendingReviews(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	libraryID := request.PathValue("library")
	if libraryID == "" {
		libraryID = request.URL.Query().Get("library_id")
	}
	result, err := server.libraries.PendingReviews(request.Context(), principal, libraryID, request.URL.Query().Get("cursor"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) libraryReviewers(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	items, err := server.libraries.Reviewers(request.Context(), principal, request.PathValue("library"))
	server.libraryResult(writer, request, 200, map[string]any{"items": items}, err)
}
func (server *Server) decideReview(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Revision int64  `json:"expected_document_revision"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	approve := strings.HasSuffix(request.URL.Path, "/approve")
	identifier, err := server.libraries.DecideReview(request.Context(), principal, request.PathValue("review"), input.Revision, approve, input.Reason, metadata(request))
	status := 200
	if identifier != "" {
		status = 202
	}
	server.libraryResult(writer, request, status, map[string]string{"operation_id": identifier}, err)
}
func (server *Server) materialization(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	result, err := server.libraries.Materialization(request.Context(), principal, request.PathValue("operation"))
	server.libraryResult(writer, request, 200, result, err)
}
func (server *Server) retryMaterialization(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(writer, request, &input); err != nil {
		server.fail(writer, request, err)
		return
	}
	identifier, err := server.libraries.RetryMaterialization(request.Context(), principal, request.PathValue("operation"), input.Reason, metadata(request))
	server.libraryResult(writer, request, 202, map[string]string{"job_id": identifier}, err)
}
