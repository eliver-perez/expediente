package httpapi

import (
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"net/http"
)

func (server *Server) advancedConfiguration(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	library := r.PathValue("library")
	if r.Method == http.MethodPut {
		var input libraries.AdvancedConfiguration
		if err := decodeJSON(w, r, &input); err != nil {
			server.fail(w, r, err)
			return
		}
		if err := server.libraries.ConfigureAdvanced(r.Context(), p, library, input, metadata(r)); err != nil {
			server.fail(w, r, err)
			return
		}
	}
	result, err := server.libraries.AdvancedConfiguration(r.Context(), p, library)
	server.libraryResult(w, r, 200, result, err)
}
