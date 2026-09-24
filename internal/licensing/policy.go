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
