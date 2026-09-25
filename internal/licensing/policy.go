// Package licensing implements only the local V1.0 license client.
package licensing

import (
	"context"
	"gestor-documental/internal/domain"
)

type Operation string

const (
	SecurityAdministration Operation = "security_administration"
	ReadDocuments          Operation = "read_documents"
	WriteDocuments         Operation = "write_documents"
	VerifyIntegrity        Operation = "verify_integrity"
	BackupExport           Operation = "backup_export"
)

func DevelopmentEnabled() bool { return developmentEnabled }
func (client *Client) Check(ctx context.Context, operation Operation) error {
	if operation == SecurityAdministration || operation == BackupExport {
		return nil
	}
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	return checkStatus(status, operation)
}
func checkStatus(status Status, operation Operation) error {
	if (operation == ReadDocuments || operation == VerifyIntegrity) && status.ReadAllowed {
		return nil
	}
	if operation == WriteDocuments && status.WriteAllowed {
		return nil
	}
	if status.State == "expired" || status.State == "revoked" {
		return domain.Failure("LICENSE_READ_ONLY", "La licencia permite consulta y descarga; las modificaciones están suspendidas.", 403)
	}
	return domain.Failure("LICENSE_RECOVERY_REQUIRED", "Se requiere activar o recuperar la licencia. La administración sigue disponible.", 403)
}
func (client *Client) CheckFeatures(ctx context.Context, features ...string) error {
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}
	if err = checkStatus(status, WriteDocuments); err != nil {
		return err
	}
	for _, feature := range features {
		known := false
		for _, name := range FeatureNames {
			known = known || feature == name
		}
		if !known || !status.Features[feature] {
			return domain.Failure("LICENSE_FEATURE", "La licencia no habilita el módulo necesario para esta operación.", 403)
		}
	}
	return nil
}
