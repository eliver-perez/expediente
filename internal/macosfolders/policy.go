// Package macosfolders limita la autorización nativa a carpetas documentales.
package macosfolders

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

type Selection struct {
	Path   string `json:"server_path"`
	Source string `json:"storage_source"`
}

func ValidateSelection(selection Selection) error {
	path := selection.Path
	if selection.Source != "managed" && selection.Source != "linked" {
		return fmt.Errorf("origen no permitido")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 1024 || strings.ContainsAny(path, "\\\x00") || strings.ContainsFunc(path, func(r rune) bool { return unicode.IsControl(r) || unicode.In(r, unicode.Cf) }) {
		return fmt.Errorf("ruta absoluta no válida")
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	// No autorizar raíces de volumen, /Users, un home completo, Library personal,
	// componentes de AIDEX ni directorios del sistema. Solo carpetas seleccionadas
	// dentro de homes/Shared o de un volumen montado; el recorrido no se hereda.
	if len(parts) < 3 || (parts[0] != "Users" && parts[0] != "Volumes") || parts[1] == "" || strings.HasPrefix(parts[1], "_") || parts[0] == "Users" && parts[2] == "Library" {
		return fmt.Errorf("selecciona una carpeta documental concreta dentro de Usuarios o de un volumen; no una raíz, home ni Library")
	}
	return nil
}

func Fingerprint(selection Selection, uid int) string {
	data, _ := json.Marshal(struct {
		Selection Selection
		FinderUID int
	}{selection, uid})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
