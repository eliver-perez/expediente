package httpapi

import (
	"gestor-documental/internal/domain"
	"gestor-documental/internal/extraction"
	"gestor-documental/internal/libraries"
	"net/http"
)

func (s *Server) processingConfiguration(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == http.MethodPut {
		var input struct {
			Configuration extraction.Concurrency `json:"configuration"`
			Revision      int64                  `json:"revision"`
		}
		if err := decodeJSON(w, r, &input); err != nil {
			s.fail(w, r, err)
			return
		}
		if err := s.libraries.ConfigureProcessing(r.Context(), p, input.Configuration, input.Revision, metadata(r)); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	result, err := s.libraries.ProcessingConfiguration(r.Context())
	s.libraryResult(w, r, 200, result, err)
}

func (s *Server) processingPolicy(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == http.MethodPut {
		var input libraries.ProcessingPolicy
		if err := decodeJSON(w, r, &input); err != nil {
			s.fail(w, r, err)
			return
		}
		if err := s.libraries.ConfigureProcessingPolicy(r.Context(), p, input, metadata(r)); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	result, err := s.libraries.ProcessingPolicy(r.Context())
	s.libraryResult(w, r, 200, result, err)
}
