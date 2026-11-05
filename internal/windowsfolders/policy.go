package windowsfolders

import (
	"fmt"
	"strings"
)

// ValidateLocal rejects namespaces, traversal, ADS, volumes, profiles and OS trees
// before any ACL mutation. Reparse-point checks additionally run by handle.
func ValidateLocal(path, source string) error {
	if source != "linked" && source != "managed" {
		return fmt.Errorf("modalidad no válida")
	}
	if len(path) < 4 || len(path) > 1024 || path[1] != ':' || path[2] != '\\' || !((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) || strings.ContainsAny(path[3:], ":/\x00\r\n") {
		return fmt.Errorf("selecciona una carpeta local; las rutas de red requieren autorización en su servidor")
	}
	parts := strings.Split(strings.TrimSuffix(path[3:], "\\"), "\\")
	for _, part := range parts {
		if strings.EqualFold(part, "AppData") {
			return fmt.Errorf("selecciona una carpeta de documentos; los datos privados de aplicaciones están protegidos")
		}
		if part == "" || part == "." || part == ".." || strings.TrimRight(part, " .") != part {
			return fmt.Errorf("ruta local no válida")
		}
	}
	first := strings.ToLower(parts[0])
	if first == "windows" || first == "program files" || first == "program files (x86)" || first == "programdata" || first == "$recycle.bin" || first == "system volume information" || first == "recovery" || (first == "users" && len(parts) <= 2) {
		return fmt.Errorf("selecciona una carpeta de documentos; esta ubicación está protegida")
	}
	return nil
}
