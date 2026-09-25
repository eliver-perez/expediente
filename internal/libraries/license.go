package libraries

import (
	"context"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
	"strings"
)

func licenseBlocked(err error) bool {
	return err != nil && strings.HasPrefix(failureCode(err), "LICENSE_")
}
func (service *Service) jobLicense(ctx context.Context, query storage.Querier, job Job) error {
	if job.Kind == "verify_managed" {
		return service.Identity.License.Check(ctx, licensing.VerifyIntegrity)
	}
	_, mode, err := settingsFor(ctx, query, job.LibraryID)
	if err != nil {
		return err
	}
	features := modeCapabilities(mode)
	if job.Kind == "extract" {
		features = append(features, "ocr")
	}
	if job.Kind == "materialize" {
		features = append(features, "managed_libraries")
		var decision, caseID string
		if err = query.QueryRowContext(ctx, "SELECT m.decision_kind,coalesce(d.case_id,'') FROM materializations m JOIN documents d ON d.id=m.document_id WHERE m.id=?", job.Version).Scan(&decision, &caseID); err != nil {
			return err
		}
		if decision == "review" {
			features = append(features, "review_workflow")
		}
		if caseID != "" {
			features = append(features, "expedientes")
		}
	}
	return service.Identity.License.CheckFeatures(ctx, features...)
}
