// Package licensing provides the enforcement boundary. JWS client implementation is H6.
package licensing

import "gestor-documental/internal/domain"

type Operation string

const (
	SecurityAdministration Operation = "security_administration"
	ReadDocuments          Operation = "read_documents"
	WriteDocuments         Operation = "write_documents"
)

func Check(operation Operation) error {
	if operation == SecurityAdministration {
		return nil
	}
	if developmentEnabled {
		return nil
	}
	return domain.Failure("LICENSE_RECOVERY_REQUIRED", "Se requiere activar o recuperar la licencia.", 403)
}

func DevelopmentEnabled() bool { return developmentEnabled }

// Capability names are the V1 contract names. H6 will evaluate signed features
// here; until then only the explicitly tagged development build can write.
func CheckFeatures(features ...string) error {
	for _, feature := range features {
		switch feature {
		case "linked_libraries", "managed_libraries", "expedientes", "review_workflow", "ocr":
		default:
			return domain.Failure("LICENSE_FEATURE", "Capacidad no reconocida.", 403)
		}
	}
	return Check(WriteDocuments)
}
