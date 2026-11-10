//go:build development

package libraries

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gestor-documental/internal/domain"
)

func TestClassificationSettingPreservesExistingRules(t *testing.T) {
	var settings Settings
	requireNoError(t, json.Unmarshal([]byte(`{"cases_enabled":true}`), &settings))
	if !settings.RequireClassification {
		t.Fatal("se relajó una biblioteca existente")
	}
	requireNoError(t, json.Unmarshal([]byte(`{"require_classification":false}`), &settings))
	if settings.RequireClassification {
		t.Fatal("no se permite desactivar explícitamente")
	}
}

func TestFolderAuthorizationScopeProtectsLibrariesAndPrivateState(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	path := filepath.Join(f.directory, "definitivos")
	requireNoError(t, s.ValidateFolderAuthorizationScope(ctx, f.author, f.library, path, "managed"))
	other := managedLibrary(t, s, f.author, "hybrid")
	if err := s.ValidateFolderAuthorizationScope(ctx, f.author, other, path, "managed"); failureCode(err) != "ROOT_AUTHORIZATION_CONFLICT" {
		t.Fatal("se autorizó otra biblioteca", err)
	}
	if err := s.ValidateFolderAuthorizationScope(ctx, f.author, f.library, path, "linked"); failureCode(err) != "ROOT_AUTHORIZATION_CONFLICT" {
		t.Fatal("se autorizó otro origen", err)
	}
	if err := s.ValidateFolderAuthorizationScope(ctx, f.author, f.library, s.Identity.Config.StateDirectory, "managed"); err == nil {
		t.Fatal("se autorizó el estado privado")
	}
}

func TestUnclassifiedIncorporationAndLaterClassification(t *testing.T) {
	for _, review := range []bool{false, true} {
		t.Run(map[bool]string{false: "directa", true: "revision"}[review], func(t *testing.T) {
			ctx := context.Background()
			f := workflowSetup(t, review)
			s := f.service
			contributor := member(t, s, f.author, f.library, "unclassified-author", "library_contributor")
			other := member(t, s, f.author, f.library, "unclassified-other", "library_contributor")
			reviewer := member(t, s, f.author, f.library, "unclassified-reviewer", "library_reviewer")
			item := uploadFixture(t, s, contributor, f.library)
			d, err := s.Document(ctx, contributor, item.DocumentID)
			requireNoError(t, err)
			if _, err = s.Document(ctx, other, d.ID); err == nil {
				t.Fatal("se reveló un borrador ajeno")
			}
			if _, err = s.FinalizeDocument(ctx, f.author, d.ID, d.Revision, domain.RequestMetadata{}); err == nil {
				t.Fatal("incorporación sin clasificación obligatoria")
			}
			settings, _, err := settingsFor(ctx, s.Database.Reader, f.library)
			requireNoError(t, err)
			settings.RequireClassification = false
			updateWorkflowSettings(t, s, f.author, f.library, settings)
			if !review {
				if _, err = s.Document(ctx, f.author, d.ID); err == nil {
					t.Fatal("desactivar clasificación reveló el borrador al gestor")
				}
				requireNoError(t, s.RequestIncorporation(ctx, contributor, d.ID, d.Revision, domain.RequestMetadata{}))
				d, err = s.Document(ctx, contributor, d.ID)
				requireNoError(t, err)
				if _, err = s.Document(ctx, other, d.ID); err == nil {
					t.Fatal("se compartió el borrador con otro colaborador")
				}
			}
			if review {
				id, err := s.SubmitReview(ctx, contributor, d.ID, "", d.Revision, domain.RequestMetadata{})
				requireNoError(t, err)
				d, err = s.Document(ctx, contributor, d.ID)
				requireNoError(t, err)
				_, err = s.DecideReview(ctx, reviewer, id, d.Revision, true, "", domain.RequestMetadata{})
				requireNoError(t, err)
			} else {
				_, err = s.FinalizeDocument(ctx, f.author, d.ID, d.Revision, domain.RequestMetadata{})
				requireNoError(t, err)
			}
			drain(t, s)
			d, err = s.Document(ctx, other, d.ID)
			requireNoError(t, err)
			if d.Approval != "approved" || d.Availability != "available" || d.Classification != "unclassified" {
				t.Fatalf("estado incorrecto: %+v", d)
			}
			if err = s.Classify(ctx, contributor, d.ID, "classification", Classification{Title: "Cambio no autorizado", Reason: "Solicitud no es permiso"}, d.Revision, domain.RequestMetadata{}); err == nil {
				t.Fatal("el colaborador modificó un aprobado")
			}
			requireNoError(t, s.RequestCorrection(ctx, contributor, d.ID, "Corregir descripción", d.Revision, domain.RequestMetadata{}))
			if err = s.RequestCorrection(ctx, other, d.ID, "Carga ajena", d.Revision, domain.RequestMetadata{}); err == nil {
				t.Fatal("se solicitó corrección de una carga ajena")
			}
			second, err := s.SaveCase(ctx, f.author, f.library, "", Case{Identifier: "SIN-CLASIFICAR"}, 0, domain.RequestMetadata{})
			requireNoError(t, err)
			requireNoError(t, s.Classify(ctx, f.author, d.ID, "classification", Classification{CaseID: second, CategoryID: f.document.CategoryID, TypeID: f.document.TypeID, Title: "Clasificado posteriormente", Reason: "Completar metadatos"}, d.Revision, domain.RequestMetadata{}))
			d, err = s.Document(ctx, f.author, d.ID)
			requireNoError(t, err)
			if d.Classification != "classified" || d.Approval != "approved" {
				t.Fatal("la clasificación posterior alteró la aprobación", d)
			}
		})
	}
}

func TestApprovedMetadataCorrectionsPreserveContentAndReview(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	d := workflowDocumentFor(t, f)
	original, err := os.ReadFile(filepath.Join(f.directory, "definitivos", filepath.FromSlash(d.RelativePath)))
	requireNoError(t, err)
	settings, _, err := settingsFor(ctx, s.Database.Reader, f.library)
	requireNoError(t, err)
	settings.ReviewClassification = true
	updateWorkflowSettings(t, s, f.author, f.library, settings)
	input := classificationSnapshot(d)
	input.Title = "Título corregido"
	input.Reason = "Corrección de captura"
	requireNoError(t, s.Classify(ctx, f.author, d.ID, "classification", input, d.Revision, domain.RequestMetadata{}))
	unchanged := workflowDocumentFor(t, f)
	if unchanged.Title != d.Title || unchanged.Revision != d.Revision || unchanged.Approval != "approved" {
		t.Fatal("se aplicó sin revisión", unchanged)
	}
	items, err := s.Corrections(ctx, f.author, d.ID)
	requireNoError(t, err)
	if len(items) != 1 {
		t.Fatal(items)
	}
	if err = s.DecideCorrection(ctx, f.author, items[0].ID, true, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("el gestor se autorizó sin perfil revisor")
	}
	reviewer := member(t, s, f.author, f.library, "metadata-reviewer", "library_reviewer")
	requireNoError(t, s.DecideCorrection(ctx, reviewer, items[0].ID, true, "Revisado", domain.RequestMetadata{}))
	after := workflowDocumentFor(t, f)
	if after.Title != input.Title || after.Approval != "approved" || after.Hash != d.Hash || after.RelativePath != d.RelativePath {
		t.Fatal("corrección afectó contenido", after)
	}
	actual, err := os.ReadFile(filepath.Join(f.directory, "definitivos", filepath.FromSlash(after.RelativePath)))
	requireNoError(t, err)
	if string(actual) != string(original) {
		t.Fatal("se alteró el original")
	}
}

func TestReplacementKeepsPreviousVersionAndDoesNotDoubleCaseCount(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	old := workflowDocumentFor(t, f)
	path := filepath.Join(f.directory, "definitivos", filepath.FromSlash(old.RelativePath))
	bytes, err := os.ReadFile(path)
	requireNoError(t, err)
	item := uploadFixture(t, s, f.author, f.library)
	// Ejercitar también el tipo único: la excepción solo admite la versión
	// anterior enlazada, nunca otros documentos que ya ocupen el requisito.
	_, err = s.Database.Writer.Exec("UPDATE document_types SET allows_multiple=0 WHERE id=?", old.TypeID)
	requireNoError(t, err)
	_, err = s.Database.Writer.Exec("UPDATE case_requirements SET allows_multiple=0 WHERE case_id=? AND document_type_id=?", old.CaseID, old.TypeID)
	requireNoError(t, err)
	requireNoError(t, s.RegisterReplacement(ctx, f.author, old.ID, item.DocumentID, "Archivo correcto", old.Revision, domain.RequestMetadata{}))
	stillCurrent := workflowDocumentFor(t, f)
	if stillCurrent.CaseID != old.CaseID || stillCurrent.Approval != "approved" {
		t.Fatal("se sustituyó antes de aprobar")
	}
	next, err := s.Document(ctx, f.author, item.DocumentID)
	requireNoError(t, err)
	_, err = s.FinalizeDocument(ctx, f.author, next.ID, next.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	previous := workflowDocumentFor(t, f)
	next, err = s.Document(ctx, f.author, next.ID)
	requireNoError(t, err)
	if previous.Classification != "historical" || previous.CaseID != "" || previous.Approval != "approved" || next.Approval != "approved" || next.CaseID != old.CaseID {
		t.Fatal("versiones incorrectas", previous, next)
	}
	actual, err := os.ReadFile(path)
	requireNoError(t, err)
	if string(actual) != string(bytes) {
		t.Fatal("se sobrescribió la versión anterior")
	}
	versions, err := s.DocumentVersions(ctx, f.author, next.ID)
	requireNoError(t, err)
	if len(versions) != 2 {
		t.Fatal(versions)
	}
	var approved int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE case_id=? AND approval_status='approved'", old.CaseID).Scan(&approved))
	if approved != 1 {
		t.Fatal("se duplicó el cumplimiento", approved)
	}
}

func TestReplacementReviewCancellationAndPermissions(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, true)
	s := f.service
	reviewer := member(t, s, f.author, f.library, "replace-reviewer", "library_reviewer")
	contributor := member(t, s, f.author, f.library, "replace-contributor", "library_contributor")
	id, err := s.SubmitReview(ctx, f.author, f.document.ID, "", f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	d := workflowDocumentFor(t, f)
	_, err = s.DecideReview(ctx, reviewer, id, d.Revision, true, "", domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	old := workflowDocumentFor(t, f)
	unauthorized := uploadFixture(t, s, contributor, f.library)
	if err = s.RegisterReplacement(ctx, contributor, old.ID, unauthorized.DocumentID, "No autorizado", old.Revision, domain.RequestMetadata{}); err == nil {
		t.Fatal("el colaborador obtuvo permisos de sustitución")
	}
	cancelled := uploadFixture(t, s, f.author, f.library)
	requireNoError(t, s.RegisterReplacement(ctx, f.author, old.ID, cancelled.DocumentID, "Primera propuesta", old.Revision, domain.RequestMetadata{}))
	d, err = s.Document(ctx, f.author, cancelled.DocumentID)
	requireNoError(t, err)
	requireNoError(t, s.CancelUpload(ctx, f.author, d.ID, "Se cancela solo la propuesta", d.Revision, domain.RequestMetadata{}))
	if previous := workflowDocumentFor(t, f); previous.CaseID != old.CaseID || !previous.CanReplace {
		t.Fatal("la cancelación afectó el original", previous)
	}
	item := uploadFixture(t, s, f.author, f.library)
	requireNoError(t, s.RegisterReplacement(ctx, f.author, old.ID, item.DocumentID, "Propuesta correcta", old.Revision, domain.RequestMetadata{}))
	d, err = s.Document(ctx, f.author, item.DocumentID)
	requireNoError(t, err)
	if _, err = s.FinalizeDocument(ctx, f.author, d.ID, d.Revision, domain.RequestMetadata{}); failureCode(err) != "REVIEW_REQUIRED" {
		t.Fatal("la versión omitió su revisión", err)
	}
	id, err = s.SubmitReview(ctx, f.author, d.ID, "", d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	d, err = s.Document(ctx, reviewer, d.ID)
	requireNoError(t, err)
	_, err = s.DecideReview(ctx, reviewer, id, d.Revision, true, "Revisión de la nueva versión", domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	if previous := workflowDocumentFor(t, f); previous.Classification != "historical" {
		t.Fatal("no se completó la sustitución revisada", previous)
	}
}
