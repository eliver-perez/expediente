package licensing

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"strings"
)

// V1 hashes one stable OS installation/platform identifier and its OS namespace.
// Raw machine identifiers never enter SQLite, logs, artifacts or network requests.
func MachineFingerprint() (string, error) {
	identifier, err := machineIdentifier()
	if err != nil {
		return "", failure("LICENSE_FINGERPRINT_UNAVAILABLE", "No se pudo verificar la identidad del equipo. Revisa el diagnóstico local.")
	}
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	compact := strings.ReplaceAll(identifier, "-", "")
	if len(compact) != 32 || strings.Trim(compact, "0") == "" {
		return "", failure("LICENSE_FINGERPRINT_UNAVAILABLE", "El identificador del equipo no es válido.")
	}
	for _, character := range compact {
		if !(character >= 'a' && character <= 'f' || character >= '0' && character <= '9') {
			return "", failure("LICENSE_FINGERPRINT_UNAVAILABLE", "El identificador del equipo no es válido.")
		}
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("AIBID-FINGERPRINT-V1\n"+runtime.GOOS+"\n"+compact))), nil
}
