package httpapi

import (
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/previews"
	"net/http"
)

func (server *Server) previewConfiguration(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == http.MethodPut {
		var input libraries.PreviewSettings
		if err := decodeJSON(w, r, &input); err != nil {
			server.fail(w, r, err)
			return
		}
		if err := server.libraries.ConfigurePreviews(r.Context(), p, input, metadata(r)); err != nil {
			server.fail(w, r, err)
			return
		}
	}
	settings, err := server.libraries.PreviewSettings(r.Context())
	server.libraryResult(w, r, 200, struct {
		libraries.PreviewSettings
		ConverterAvailable bool   `json:"converter_available"`
		Generator          string `json:"generator_version"`
	}{settings, previews.Available(settings.ConverterPath), previews.Generator(settings.ConverterPath)}, err)
}
func (server *Server) clearPreviews(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	var input struct {
		Confirm bool `json:"confirm"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		server.fail(w, r, err)
		return
	}
	if !input.Confirm {
		server.fail(w, r, domain.Failure("INVALID_REQUEST", "Confirma que deseas vaciar la caché de vistas previas.", 422))
		return
	}
	err := server.libraries.ClearPreviews(r.Context(), p, metadata(r))
	server.libraryResult(w, r, 204, nil, err)
}
func (server *Server) documentPreview(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	if r.Method == http.MethodPost {
		var input struct {
			Retry bool `json:"retry"`
		}
		if err := decodeJSON(w, r, &input); err != nil {
			server.fail(w, r, err)
			return
		}
		result, err := server.libraries.RequestPreview(r.Context(), p, r.PathValue("document"), input.Retry, metadata(r))
		server.libraryResult(w, r, 202, result, err)
		return
	}
	result, err := server.libraries.Preview(r.Context(), p, r.PathValue("document"))
	server.libraryResult(w, r, 200, result, err)
}
func (server *Server) previewContent(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	file, media, err := server.libraries.OpenPreview(r.Context(), p, r.PathValue("document"), r.URL.Query().Get("id"), metadata(r))
	if err != nil {
		server.fail(w, r, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		server.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", media)
	w.Header().Set("Content-Disposition", `inline; filename="vista-previa.pdf"`)
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; frame-ancestors 'self'")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "vista-previa.pdf", info.ModTime(), file)
}
