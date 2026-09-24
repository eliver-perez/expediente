package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"gestor-documental/internal/domain"
)

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return domain.Failure("INVALID_REQUEST", "Se requiere JSON.", 400)
	}
	contents, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 65536))
	if err != nil {
		return domain.Failure("REQUEST_TOO_LARGE", "La solicitud excede el tamaño permitido.", 413)
	}
	if !utf8.Valid(contents) || len(bytes.TrimSpace(contents)) == 0 || bytes.TrimSpace(contents)[0] != '{' {
		return domain.Failure("INVALID_REQUEST", "JSON no válido.", 400)
	}
	probe := json.NewDecoder(bytes.NewReader(contents))
	if err := checkJSONValue(probe, 0); err != nil {
		return err
	}
	if _, err := probe.Token(); err != io.EOF {
		return domain.Failure("INVALID_REQUEST", "Se requiere un solo objeto JSON.", 400)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return domain.Failure("INVALID_REQUEST", "Revisa los campos de la solicitud.", 400)
	}
	return nil
}

func checkJSONValue(decoder *json.Decoder, depth int) error {
	invalid := func() error { return domain.Failure("INVALID_REQUEST", "JSON no válido o claves repetidas.", 400) }
	if depth > 16 {
		return invalid()
	}
	token, err := decoder.Token()
	if err != nil {
		return invalid()
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return invalid()
	}
	keys := map[string]bool{}
	for decoder.More() {
		if delimiter == '{' {
			keyToken, err := decoder.Token()
			if err != nil {
				return invalid()
			}
			key, ok := keyToken.(string)
			key = strings.ToLower(key) // encoding/json matches struct fields case-insensitively.
			if !ok || keys[key] {
				return invalid()
			}
			keys[key] = true
		}
		if err := checkJSONValue(decoder, depth+1); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil || (delimiter == '{' && closing != json.Delim('}')) || (delimiter == '[' && closing != json.Delim(']')) {
		return invalid()
	}
	return nil
}

func expectedRevision(request *http.Request) (int64, error) {
	value := request.Header.Get("If-Match")
	if value == "" {
		return 0, domain.Failure("PRECONDITION_REQUIRED", "Falta If-Match con la revisión del usuario.", 428)
	}
	if !strings.HasPrefix(value, "\"") || !strings.HasSuffix(value, "\"") {
		return 0, domain.Failure("INVALID_REQUEST", "If-Match no válido.", 400)
	}
	revision, err := strconv.ParseInt(strings.Trim(value, "\""), 10, 64)
	if err != nil || revision < 1 {
		return 0, domain.Failure("INVALID_REQUEST", "Revisión no válida.", 400)
	}
	return revision, nil
}

func userJSON(writer http.ResponseWriter, status int, user domain.User) {
	writer.Header().Set("ETag", strconv.Quote(strconv.FormatInt(user.Revision, 10)))
	writeJSON(writer, status, user)
}
