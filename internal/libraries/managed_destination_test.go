//go:build development

package libraries

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
)

func destinationRevision(t *testing.T, s *Service, library string) int64 {
	t.Helper()
	var revision int64
	requireNoError(t, s.Database.Reader.QueryRow("SELECT revision FROM libraries WHERE id=?", library).Scan(&revision))
	return revision
}
func destinationValue(t *testing.T, s *Service, library string) string {
	t.Helper()
	settings, _, err := settingsFor(context.Background(), s.Database.Reader, library)
	requireNoError(t, err)
	return settings.ManagedRootID
}
func registerManagedDestination(t *testing.T, s *Service, p domain.Principal, library, path string) string {
	t.Helper()
	requireNoError(t, os.MkdirAll(path, 0700))
	plan, err := s.PlanStorageRoot(context.Background(), p, library, path, "managed", domain.RequestMetadata{})
	requireNoError(t, err)
	id, err := s.ConfirmRoot(context.Background(), p, library, plan.ID, plan.Revision, false, domain.RequestMetadata{})
	requireNoError(t, err)
	return id
}

func TestManagedDestinationAssignmentValidationAndAudit(t *testing.T) {
	ctx := context.Background()
	s, p, dir := fixture(t)
	lib := managedLibrary(t, s, p, "hybrid")
	first := registerManagedDestination(t, s, p, lib, filepath.Join(dir, "primera"))
	if destinationValue(t, s, lib) != first {
		t.Fatal("primera carpeta sin destino")
	}
	second := registerManagedDestination(t, s, p, lib, filepath.Join(dir, "segunda"))
	if destinationValue(t, s, lib) != first {
		t.Fatal("segunda reemplazó el destino")
	}
	rev := destinationRevision(t, s, lib)
	requireNoError(t, s.SetManagedDestination(ctx, p, lib, second, rev, domain.RequestMetadata{}))
	if destinationValue(t, s, lib) != second {
		t.Fatal("cambio no persistido")
	}
	if err := s.SetManagedDestination(ctx, p, lib, first, rev, domain.RequestMetadata{}); failureCode(err) != "VERSION_CONFLICT" {
		t.Fatal("escritura concurrente obsoleta", err)
	}
	linkedPath := filepath.Join(dir, "vinculada")
	requireNoError(t, os.MkdirAll(linkedPath, 0700))
	linked := addRoot(t, s, p, lib, linkedPath)
	if err := s.SetManagedDestination(ctx, p, lib, linked, destinationRevision(t, s, lib), domain.RequestMetadata{}); err == nil {
		t.Fatal("destino vinculado permitido")
	}
	other := managedLibrary(t, s, p, "managed")
	foreign := registerManagedDestination(t, s, p, other, filepath.Join(dir, "otra biblioteca"))
	if err := s.SetManagedDestination(ctx, p, lib, foreign, destinationRevision(t, s, lib), domain.RequestMetadata{}); err == nil {
		t.Fatal("destino ajeno permitido")
	}
	reader := member(t, s, p, lib, "destination-reader", "library_reader")
	if err := s.SetManagedDestination(ctx, reader, lib, first, destinationRevision(t, s, lib), domain.RequestMetadata{}); err == nil {
		t.Fatal("lector cambió destino")
	}
	// No elegir arbitrariamente en una biblioteca anterior con varios destinos.
	_, err := s.Database.Writer.Exec("UPDATE libraries SET settings_json=jsonb_set((settings_json)::jsonb,'{default_managed_root_id}',to_jsonb(''::text))::text WHERE id=?", lib)
	requireNoError(t, err)
	registerManagedDestination(t, s, p, lib, filepath.Join(dir, "tercera"))
	if destinationValue(t, s, lib) != "" {
		t.Fatal("selección arbitraria de destinos existentes")
	}
	var actor, details string
	requireNoError(t, s.Database.Reader.QueryRow("SELECT actor_user_id,details_json FROM audit_events WHERE library_id=? AND event_type='storage.destination_changed' AND ((details_json)::jsonb #>> '{new_root_id}')=?", lib, second).Scan(&actor, &details))
	var data map[string]string
	requireNoError(t, json.Unmarshal([]byte(details), &data))
	if actor != p.User.ID || data["previous_root_id"] != first {
		t.Fatal("auditoría incompleta", details)
	}
}

func TestManagedDestinationPreservesLocationsAndBlocksDisabled(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	path := filepath.Join(f.directory, "Destino nuevo á 東京")
	second := registerManagedDestination(t, s, f.author, f.library, path)
	requireNoError(t, s.SetManagedDestination(ctx, f.author, f.library, second, destinationRevision(t, s, f.library), domain.RequestMetadata{}))
	// El trabajo ya planificado conserva su destino; el cambio afecta los nuevos.
	drain(t, s)
	old := workflowDocumentFor(t, f)
	firstRoot, err := s.root(ctx, f.root)
	requireNoError(t, err)
	if !strings.HasPrefix(old.OriginalPath, firstRoot.Path+string(os.PathSeparator)) {
		t.Fatal("se redirigió una incorporación ya planificada")
	}
	bytes, err := os.ReadFile(old.OriginalPath)
	requireNoError(t, err)
	settings, _, err := settingsFor(ctx, s.Database.Reader, f.library)
	requireNoError(t, err)
	settings.RequireClassification = false
	updateWorkflowSettings(t, s, f.author, f.library, settings)
	item := uploadFixture(t, s, f.author, f.library)
	drain(t, s)
	doc, err := s.Document(ctx, f.author, item.DocumentID)
	requireNoError(t, err)
	_, err = s.FinalizeDocument(ctx, f.author, doc.ID, doc.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	doc, err = s.Document(ctx, f.author, doc.ID)
	requireNoError(t, err)
	if !strings.HasPrefix(doc.OriginalPath, path+string(os.PathSeparator)) {
		t.Fatal("no usó el nuevo destino", doc.OriginalPath)
	}
	oldAgain := workflowDocumentFor(t, f)
	if oldAgain.OriginalPath != old.OriginalPath {
		t.Fatal("se movió el documento anterior")
	}
	after, err := os.ReadFile(old.OriginalPath)
	requireNoError(t, err)
	if string(after) != string(bytes) {
		t.Fatal("se alteró el original")
	}
	root, err := s.root(ctx, second)
	requireNoError(t, err)
	requireNoError(t, s.ConfigureRoot(ctx, f.author, second, false, 900, root.Revision, domain.RequestMetadata{}))
	if destinationValue(t, s, f.library) != "" {
		t.Fatal("destino desactivado no retirado")
	}
	blocked := uploadFixture(t, s, f.author, f.library)
	drain(t, s)
	doc, err = s.Document(ctx, f.author, blocked.DocumentID)
	requireNoError(t, err)
	if _, err = s.FinalizeDocument(ctx, f.author, doc.ID, doc.Revision, domain.RequestMetadata{}); err == nil {
		t.Fatal("incorporación sin destino")
	}
	if err = s.SetManagedDestination(ctx, f.author, f.library, second, destinationRevision(t, s, f.library), domain.RequestMetadata{}); err == nil {
		t.Fatal("selección de destino inactivo")
	}
	root, err = s.root(ctx, second)
	requireNoError(t, err)
	requireNoError(t, s.ConfigureRoot(ctx, f.author, second, true, 900, root.Revision, domain.RequestMetadata{}))
	if destinationValue(t, s, f.library) != "" {
		t.Fatal("reactivar seleccionó automáticamente")
	}
	requireNoError(t, s.SetManagedDestination(ctx, f.author, f.library, second, destinationRevision(t, s, f.library), domain.RequestMetadata{}))
	plan, err := s.RetirementPlan(ctx, f.author, second)
	requireNoError(t, err)
	requireNoError(t, s.Retire(ctx, f.author, second, plan["plan_id"].(string), "retain_index", domain.RequestMetadata{}))
	if destinationValue(t, s, f.library) != "" {
		t.Fatal("destino retirado conservado")
	}
	if _, err = os.Stat(old.OriginalPath); err != nil {
		t.Fatal("original borrado", err)
	}
}

func TestManagedDestinationPermissionFailureUsesAuthorizationCode(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("la prueba necesita un usuario sin privilegios")
	}
	ctx := context.Background()
	s, p, dir := fixture(t)
	lib := managedLibrary(t, s, p, "hybrid")
	path := filepath.Join(dir, "Solo lectura")
	requireNoError(t, os.MkdirAll(path, 0500))
	defer os.Chmod(path, 0700)
	if _, err := s.PlanStorageRoot(ctx, p, lib, path, "managed", domain.RequestMetadata{}); failureCode(err) != "ROOT_PERMISSION_DENIED" {
		t.Fatal("no ofrece autorización para escritura", err)
	}
	requireNoError(t, os.Chmod(path, 0700))
	id := registerManagedDestination(t, s, p, lib, path)
	requireNoError(t, os.Chmod(path, 0500))
	if err := s.SetManagedDestination(ctx, p, lib, id, destinationRevision(t, s, lib), domain.RequestMetadata{}); failureCode(err) != "ROOT_PERMISSION_DENIED" {
		t.Fatal("no comprobó escritura efectiva", err)
	}
	entries, err := os.ReadDir(path)
	requireNoError(t, err)
	if len(entries) != 0 {
		t.Fatal("residuos de la prueba", entries)
	}
}

func TestManagedDestinationDisabledDuringPublicationCannotCommit(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	job, err := s.claim(ctx)
	requireNoError(t, err)
	s.materializationFault = func(point string) error {
		if point == "before_commit" {
			root, e := s.root(ctx, f.root)
			if e != nil {
				return e
			}
			return s.ConfigureRoot(ctx, f.author, f.root, false, 900, root.Revision, domain.RequestMetadata{})
		}
		return nil
	}
	if err = s.Materialize(ctx, job); err == nil {
		t.Fatal("se confirmó una incorporación sobre destino desactivado")
	}
	if workflowDocumentFor(t, f).Approval == "approved" {
		t.Fatal("aprobación inválida")
	}
	if destinationValue(t, s, f.library) != "" {
		t.Fatal("destino desactivado sigue seleccionado")
	}
}
