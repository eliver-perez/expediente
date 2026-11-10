package httpapi

import (
	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
	"net/http"
)

func (server *Server) fileConfiguration(writer http.ResponseWriter, request *http.Request, principal domain.Principal) {
	libraryID := request.PathValue("library")
	if request.Method == http.MethodPut {
		var input struct {
			Configuration  documentformat.Policy    `json:"configuration"`
			Overrides      documentformat.Overrides `json:"overrides"`
			Revision       int64                    `json:"revision"`
			GlobalRevision int64                    `json:"global_revision"`
		}
		if err := decodeJSON(writer, request, &input); err != nil {
			server.fail(writer, request, err)
			return
		}
		if err := server.libraries.ConfigureFiles(request.Context(), principal, libraryID, input.Configuration, input.Overrides, input.Revision, input.GlobalRevision, metadata(request)); err != nil {
			server.fail(writer, request, err)
			return
		}
	}
	result, err := server.libraries.FileConfiguration(request.Context(), principal, libraryID)
	server.libraryResult(writer, request, 200, result, err)
}
