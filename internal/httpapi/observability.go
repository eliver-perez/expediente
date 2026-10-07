package httpapi

import (
	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"net/http"
)

func (s *Server) observabilityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/system/dashboard", s.protected("system.configure", true, func(w http.ResponseWriter, r *http.Request, p domain.Principal) {
		result, err := s.libraries.Dashboard(r.Context(), p)
		s.libraryResult(w, r, 200, result, err)
	}))
	mux.HandleFunc("GET /api/v1/system/dashboard/libraries", s.protected("system.configure", true, func(w http.ResponseWriter, r *http.Request, p domain.Principal) {
		result, err := s.libraries.DashboardLibraries(r.Context(), p, r.URL.Query().Get("cursor"))
		s.libraryResult(w, r, 200, result, err)
	}))
	mux.HandleFunc("GET /api/v1/system/errors", s.protected("system.configure", true, func(w http.ResponseWriter, r *http.Request, p domain.Principal) {
		q := r.URL.Query()
		result, err := s.libraries.DiagnosticEvents(r.Context(), p, libraries.DiagnosticFilter{From: q.Get("from"), To: q.Get("to"), Module: q.Get("module"), Severity: q.Get("severity"), Status: q.Get("status")}, q.Get("cursor"))
		s.libraryResult(w, r, 200, result, err)
	}))
	mux.HandleFunc("PATCH /api/v1/system/errors/{id}", s.protected("system.configure", true, func(w http.ResponseWriter, r *http.Request, p domain.Principal) {
		var input struct {
			Status   string `json:"status"`
			Revision int64  `json:"revision"`
		}
		if err := decodeJSON(w, r, &input); err != nil {
			s.fail(w, r, err)
			return
		}
		err := s.libraries.ReviewDiagnostic(r.Context(), p, r.PathValue("id"), input.Status, input.Revision, metadata(r))
		s.libraryResult(w, r, 204, nil, err)
	}))
	mux.HandleFunc("GET /api/v1/system/errors/{id}/diagnostic", s.protected("system.configure", true, func(w http.ResponseWriter, r *http.Request, p domain.Principal) {
		result, err := s.libraries.SupportDiagnostic(r.Context(), p, r.PathValue("id"))
		s.libraryResult(w, r, 200, result, err)
	}))
	for _, method := range []string{"GET", "PUT"} {
		mux.HandleFunc(method+" /api/v1/system/error-policy", s.protected("system.configure", true, s.diagnosticPolicy))
	}
}
func (s *Server) diagnosticPolicy(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == "PUT" {
		var policy diagnostics.Policy
		if err := decodeJSON(w, r, &policy); err != nil {
			s.fail(w, r, err)
			return
		}
		if err := s.libraries.ConfigureDiagnostics(r.Context(), p, policy, metadata(r)); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	result, err := s.libraries.DiagnosticPolicy(r.Context(), p)
	s.libraryResult(w, r, 200, result, err)
}
