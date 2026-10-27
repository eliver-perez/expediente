package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"gestor-documental/internal/domain"
)

func (s *Server) deletionList(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	result, err := s.libraries.Deletions(r.Context(), p, r.URL.Query().Get("cursor"))
	s.libraryResult(w, r, 200, result, err)
}

func (s *Server) libraryLifecycle(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	revision, err := expectedRevision(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	err = s.libraries.ArchiveLibrary(r.Context(), p, r.PathValue("library"), revision, strings.HasSuffix(r.URL.Path, "/restore"), metadata(r))
	s.libraryResult(w, r, 204, nil, err)
}
func (s *Server) resumeRoot(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	revision, err := expectedRevision(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	err = s.libraries.ResumeRoot(r.Context(), p, r.PathValue("root"), revision, metadata(r))
	s.libraryResult(w, r, 202, map[string]string{"status": "queued"}, err)
}
func (s *Server) planEntityDeletion(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	kind, id := "library", r.PathValue("library")
	if id == "" {
		kind, id = "root", r.PathValue("root")
	}
	result, err := s.libraries.PlanDeletion(r.Context(), p, kind, id, metadata(r))
	s.libraryResult(w, r, 201, result, err)
}
func (s *Server) entityDeletion(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	id := r.PathValue("operation")
	if r.Method == "GET" {
		result, err := s.libraries.DeletionStatus(r.Context(), p, id)
		s.libraryResult(w, r, 200, result, err)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/retry") {
		result, err := s.libraries.RetryDeletionCleanup(r.Context(), p, id)
		s.libraryResult(w, r, 200, result, err)
		return
	}
	var input struct {
		Name string `json:"confirmation_name"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		s.fail(w, r, err)
		return
	}
	result, err := s.libraries.ConfirmDeletion(r.Context(), p, id, input.Name, metadata(r))
	status := 200
	if result.State == "pending" || result.State == "running" || result.State == "cleanup_pending" {
		status = 202
	}
	s.libraryResult(w, r, status, result, err)
}
func (s *Server) deletionManifest(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if _, err := s.libraries.DeletionStatus(r.Context(), p, r.PathValue("operation")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="documentos-conservados.jsonl"`)
	if err := s.libraries.DeletionManifest(r.Context(), p, r.PathValue("operation"), func(line string) error { _, err := fmt.Fprint(w, line); return err }); err != nil {
		return
	}
}
