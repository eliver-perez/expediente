package httpapi

import (
	"net/http"
	"strings"

	"gestor-documental/internal/domain"
)

func (server *Server) requestIncorporation(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	revision, err := expectedRevision(r)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	err = server.libraries.RequestIncorporation(r.Context(), p, r.PathValue("document"), revision, metadata(r))
	server.libraryResult(w, r, 200, map[string]bool{"requested": err == nil}, err)
}

func (server *Server) documentCorrections(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == "GET" {
		items, err := server.libraries.Corrections(r.Context(), p, r.PathValue("document"))
		server.libraryResult(w, r, 200, map[string]any{"items": items}, err)
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	revision, err := expectedRevision(r)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	err = server.libraries.RequestCorrection(r.Context(), p, r.PathValue("document"), input.Reason, revision, metadata(r))
	server.libraryResult(w, r, 201, map[string]bool{"requested": err == nil}, err)
}
func (server *Server) decideCorrection(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	err := server.libraries.DecideCorrection(r.Context(), p, r.PathValue("correction"), strings.HasSuffix(r.URL.Path, "/approve"), input.Reason, metadata(r))
	server.libraryResult(w, r, 200, map[string]bool{"decided": err == nil}, err)
}
func (server *Server) libraryCorrections(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	result, err := server.libraries.PendingCorrections(r.Context(), p, r.PathValue("library"), r.URL.Query().Get("cursor"))
	server.libraryResult(w, r, 200, result, err)
}
func (server *Server) documentVersions(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == "GET" {
		items, err := server.libraries.DocumentVersions(r.Context(), p, r.PathValue("document"))
		server.libraryResult(w, r, 200, map[string]any{"items": items}, err)
		return
	}
	var input struct {
		NewID  string `json:"new_document_id"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	revision, err := expectedRevision(r)
	if err != nil {
		server.fail(w, r, err)
		return
	}
	err = server.libraries.RegisterReplacement(r.Context(), p, r.PathValue("document"), input.NewID, input.Reason, revision, metadata(r))
	server.libraryResult(w, r, 201, map[string]bool{"registered": err == nil}, err)
}
