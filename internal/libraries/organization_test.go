//go:build development

package libraries

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/domain"
)

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func managedLibrary(t *testing.T, service *Service, principal domain.Principal, mode string) string {
	t.Helper()
	id, err := service.Create(context.Background(), principal, "Organización", mode, principal.User.ID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	return id
}
func organizationCatalog(t *testing.T, service *Service, principal domain.Principal, libraryID string, multiple, title bool) (string, string) {
	t.Helper()
	ctx := context.Background()
	category, err := service.SaveCatalog(ctx, principal, libraryID, "categories", "", CatalogEntry{Name: "Planos"}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	kind, err := service.SaveCatalog(ctx, principal, libraryID, "document-types", "", CatalogEntry{Name: "Varios", CategoryID: category, Multiple: multiple, TitleRequired: title}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	return category, kind
}
func member(t *testing.T, service *Service, admin domain.Principal, libraryID, username, role string) domain.Principal {
	t.Helper()
	ctx := context.Background()
	user, err := service.Identity.CreateUser(ctx, admin, username, username, "abcdef", domain.RequestMetadata{})
	requireNoError(t, err)
	requireNoError(t, service.SetMember(ctx, admin, libraryID, user.ID, []string{role}, domain.RequestMetadata{}))
	login, err := service.Identity.Login(ctx, username, "abcdef", domain.RequestMetadata{ObservedIP: "127.0.0.5"})
	requireNoError(t, err)
	principal, err := service.Identity.Authenticate(ctx, login.SessionToken, false, domain.RequestMetadata{})
	requireNoError(t, err)
	return principal
}
func uploadFixture(t *testing.T, service *Service, principal domain.Principal, libraryID string) UploadItem {
	t.Helper()
	ctx := context.Background()
	batch, err := service.CreateBatch(ctx, principal, libraryID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	contents, err := os.ReadFile("../../testdata/documents/native.pdf")
	requireNoError(t, err)
	item, err := service.Upload(ctx, principal, batch, domain.NewID(), "Privado.pdf", bytes.NewReader(contents), nil, domain.RequestMetadata{})
	requireNoError(t, err)
	return item
}
func TestH4TemplateSnapshotsInitializationAndDerivedCounts(t *testing.T) {
	service, principal, _ := fixture(t)
	ctx := context.Background()
	libraryID := managedLibrary(t, service, principal, "managed")
	category, kind := organizationCatalog(t, service, principal, libraryID, true, false)
	version, err := service.SaveTemplate(ctx, principal, libraryID, "", "Ingreso", []RequirementInput{{CategoryID: category, TypeID: kind, Count: 2, Mandatory: true}}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	caseID, err := service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "PR-008", TemplateVersionID: version}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	templates, err := service.Templates(ctx, principal, libraryID)
	requireNoError(t, err)
	if len(templates) != 1 {
		t.Fatal(templates)
	}
	newer, err := service.SaveTemplate(ctx, principal, libraryID, templates[0].ID, "", []RequirementInput{{CategoryID: category, TypeID: kind, Count: 3, Mandatory: false}}, 1, domain.RequestMetadata{})
	requireNoError(t, err)
	requireNoError(t, service.InitializeCase(ctx, principal, caseID, newer, domain.RequestMetadata{}))
	requirements, err := service.Requirements(ctx, principal, caseID)
	requireNoError(t, err)
	if len(requirements.Items) != 1 || requirements.Items[0].Count != 2 || requirements.Progress == nil || *requirements.Progress != 0 {
		t.Fatalf("snapshot changed: %+v", requirements)
	}
	requireNoError(t, service.UpdateRequirement(ctx, principal, caseID, requirements.Items[0].ID, 4, false, requirements.Items[0].Revision, domain.RequestMetadata{}))
	requirements, err = service.Requirements(ctx, principal, caseID)
	requireNoError(t, err)
	if requirements.Progress != nil {
		t.Fatal("no mandatory must be null")
	}
	next, err := service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "PR-009", TemplateVersionID: newer}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	nextRequirements, err := service.Requirements(ctx, principal, next)
	requireNoError(t, err)
	if nextRequirements.Items[0].Count != 3 {
		t.Fatal("new case did not use new version")
	}
	if _, err = service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "pr-008"}, 0, domain.RequestMetadata{}); err == nil {
		t.Fatal("duplicate identifier")
	}
}
func TestH4LinkedConversionAssociationAndReassignmentPreservePhysicalIndex(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	libraryID := addLibrary(t, service, principal, "Convertir")
	path := filepath.Join(directory, "linked")
	copyFixture(t, filepath.Join(path, "native.pdf"), "native.pdf")
	rootID := addRoot(t, service, principal, libraryID, path)
	drain(t, service)
	before := query(t, service, principal, libraryID, "metálica").Items[0]
	libraries, err := service.Libraries(ctx, principal)
	requireNoError(t, err)
	requireNoError(t, service.UpdateConfiguration(ctx, principal, libraryID, "Convertir", "spa", "hybrid", nil, libraries[0].Revision, domain.RequestMetadata{}))
	category, kind := organizationCatalog(t, service, principal, libraryID, false, true)
	first, err := service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "A-001"}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	second, err := service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "B-002"}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	classification := Classification{CaseID: first, CategoryID: category, TypeID: kind}
	if err = service.Classify(ctx, principal, before.ID, "associate", classification, before.Revision, domain.RequestMetadata{}); err == nil {
		t.Fatal("Varios allowed without title")
	}
	classification.Title = "Plano de cubierta"
	requireNoError(t, service.Classify(ctx, principal, before.ID, "associate", classification, before.Revision, domain.RequestMetadata{}))
	after, err := service.Document(ctx, principal, before.ID)
	requireNoError(t, err)
	if before.Hash != after.Hash || before.FileID != after.FileID || before.OriginalPath != after.OriginalPath || before.Pages != after.Pages || after.CaseID != first {
		t.Fatal("association changed physical index")
	}
	if err = service.Classify(ctx, principal, before.ID, "associate", classification, after.Revision, domain.RequestMetadata{}); err == nil {
		t.Fatal("second association accepted")
	}
	classification.CaseID = second
	classification.Reason = "Corrección"
	plan, err := service.RetirementPlan(ctx, principal, rootID)
	requireNoError(t, err)
	if plan["affected_case_count"] != 1 {
		t.Fatal("retirement omitted case impact")
	}
	requireNoError(t, service.Classify(ctx, principal, before.ID, "reassign", classification, after.Revision, domain.RequestMetadata{}))
	if err = service.Retire(ctx, principal, rootID, plan["plan_id"].(string), "remove_index_references", domain.RequestMetadata{}); err == nil {
		t.Fatal("stale plan accepted after reassignment")
	}
	result, err := service.Search(ctx, principal, SearchInput{Query: "B-002", Type: "identifier", Libraries: []string{libraryID}}, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if result.Count != 1 || result.Items[0].ID != before.ID {
		t.Fatal("identifier search lost association")
	}
	var runs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM extraction_runs WHERE physical_file_id=?", before.FileID).Scan(&runs))
	if runs != 1 {
		t.Fatal("association re-extracted")
	}
	copyFixture(t, filepath.Join(directory, "other", "new.pdf"), "native.pdf")
	addRoot(t, service, principal, libraryID, filepath.Join(directory, "other"))
	uploadFixture(t, service, principal, libraryID) // Root scan is queued; uploads remain usable.
}

func TestH4ClassificationScopeMultiplicityAndProgress(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	libraryID := managedLibrary(t, service, principal, "hybrid")
	category, kind := organizationCatalog(t, service, principal, libraryID, true, false)
	version, err := service.SaveTemplate(ctx, principal, libraryID, "", "Requisitos", []RequirementInput{{CategoryID: category, TypeID: kind, Count: 1, Mandatory: true}}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	caseID, err := service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "P-001", TemplateVersionID: version}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	otherLibrary := managedLibrary(t, service, principal, "managed")
	otherCategory, otherKind := organizationCatalog(t, service, principal, otherLibrary, false, false)
	otherCase, err := service.SaveCase(ctx, principal, otherLibrary, "", Case{Identifier: "P-001"}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	item := uploadFixture(t, service, principal, libraryID)
	document, err := service.Document(ctx, principal, item.DocumentID)
	requireNoError(t, err)
	for _, input := range []Classification{{CaseID: otherCase, CategoryID: category, TypeID: kind}, {CaseID: caseID, CategoryID: otherCategory, TypeID: otherKind}} {
		if err = service.Classify(ctx, principal, document.ID, "classification", input, document.Revision, domain.RequestMetadata{}); err == nil {
			t.Fatal("cross-library classification accepted")
		}
	}
	for _, uploaded := range []UploadItem{item, uploadFixture(t, service, principal, libraryID)} {
		requireNoError(t, service.Classify(ctx, principal, uploaded.DocumentID, "classification", Classification{CaseID: caseID, CategoryID: category, TypeID: kind}, 1, domain.RequestMetadata{}))
	}
	checkProgress := func(want float64, valid int) {
		t.Helper()
		result, err := service.Requirements(ctx, principal, caseID)
		requireNoError(t, err)
		if result.Progress == nil || *result.Progress != want || result.Items[0].Counts["valid_approved"] != valid {
			t.Fatalf("unexpected progress: %+v", result)
		}
	}
	checkProgress(0, 0)
	drain(t, service)
	// Simulate H5 states only to verify H4 derived counts; no approval API exists yet.
	_, err = service.Database.Writer.Exec("UPDATE documents SET approval_status='approved' WHERE case_id=?;", caseID)
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec("UPDATE physical_files SET availability='available' WHERE library_id=?", libraryID)
	requireNoError(t, err)
	checkProgress(100, 2) // Capped by required_count, never 200%.
	_, err = service.Database.Writer.Exec("UPDATE physical_files SET integrity_status='changed' WHERE library_id=?", libraryID)
	requireNoError(t, err)
	checkProgress(0, 0)
	_, err = service.Database.Writer.Exec("UPDATE physical_files SET availability='missing',integrity_status='verified' WHERE library_id=?", libraryID)
	requireNoError(t, err)
	checkProgress(0, 0)
	path := filepath.Join(directory, "linked-progress")
	copyFixture(t, filepath.Join(path, "native.pdf"), "native.pdf")
	addRoot(t, service, principal, libraryID, path)
	drain(t, service)
	result, err := service.Search(ctx, principal, SearchInput{Type: "general", Libraries: []string{libraryID}, Filters: Filters{Source: "linked"}}, domain.RequestMetadata{}, false)
	requireNoError(t, err)
	linked := result.Items[0]
	requireNoError(t, service.Classify(ctx, principal, linked.ID, "associate", Classification{CaseID: caseID, CategoryID: category, TypeID: kind}, linked.Revision, domain.RequestMetadata{}))
	_, err = service.Database.Writer.Exec("UPDATE documents SET approval_status='approved' WHERE id=?", linked.ID)
	requireNoError(t, err)
	_, err = service.Database.Writer.Exec("UPDATE physical_files SET availability='missing' WHERE id=?", linked.FileID)
	requireNoError(t, err)
	checkProgress(100, 1) // Linked approval remains valid while its original is missing.
	_, err = service.Database.Writer.Exec("UPDATE document_types SET allows_multiple=0 WHERE id=?", kind)
	requireNoError(t, err)
	uniqueVersion, err := service.SaveTemplate(ctx, principal, libraryID, "", "Único", []RequirementInput{{CategoryID: category, TypeID: kind, Count: 1, Mandatory: true}}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	uniqueCase, err := service.SaveCase(ctx, principal, libraryID, "", Case{Identifier: "U-001", TemplateVersionID: uniqueVersion}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	for index := range 2 {
		uploaded := uploadFixture(t, service, principal, libraryID)
		err = service.Classify(ctx, principal, uploaded.DocumentID, "classification", Classification{CaseID: uniqueCase, CategoryID: category, TypeID: kind}, 1, domain.RequestMetadata{})
		if index == 0 {
			requireNoError(t, err)
		} else if err == nil {
			t.Fatal("unique type accepted a second file")
		}
	}
}
func TestH4PrivateUploadsScopeOCRAndIdempotency(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	libraryID := managedLibrary(t, service, admin, "hybrid")
	owner := member(t, service, admin, libraryID, "author-h4", "library_contributor")
	reader := member(t, service, admin, libraryID, "reader-h4", "library_reader")
	reviewer := member(t, service, admin, libraryID, "reviewer-h4", "library_reviewer")
	batch, err := service.CreateBatch(ctx, owner, libraryID, strings.Repeat("b", 20), domain.RequestMetadata{})
	requireNoError(t, err)
	again, err := service.CreateBatch(ctx, owner, libraryID, strings.Repeat("b", 20), domain.RequestMetadata{})
	requireNoError(t, err)
	if batch != again {
		t.Fatal("batch replay duplicated")
	}
	contents, err := os.ReadFile("../../testdata/documents/native.pdf")
	requireNoError(t, err)
	clientID := domain.NewID()
	item, err := service.Upload(ctx, owner, batch, clientID, "Secreto.pdf", bytes.NewReader(contents), nil, domain.RequestMetadata{})
	requireNoError(t, err)
	replay, err := service.Upload(ctx, owner, batch, clientID, "Secreto.pdf", bytes.NewReader(contents), nil, domain.RequestMetadata{})
	requireNoError(t, err)
	if replay.DocumentID != item.DocumentID {
		t.Fatal("upload replay duplicated")
	}
	if _, err = service.Upload(ctx, owner, batch, clientID, "Secreto.pdf", bytes.NewReader(append(contents, ' ')), nil, domain.RequestMetadata{}); err == nil {
		t.Fatal("conflicting replay accepted")
	}
	drain(t, service)
	for _, forbidden := range []domain.Principal{reader, admin} {
		if _, err = service.Document(ctx, forbidden, item.DocumentID); err == nil {
			t.Fatal("private document metadata leaked")
		}
		if _, err = service.Batch(ctx, forbidden, batch); err == nil {
			t.Fatal("private batch leaked")
		}
		result := query(t, service, forbidden, libraryID, "metálica")
		if result.Count != 0 {
			t.Fatal("private search/total leaked")
		}
		if _, err = service.Page(ctx, forbidden, item.DocumentID, 1); err == nil {
			t.Fatal("private OCR leaked")
		}
	}
	for _, allowed := range []domain.Principal{owner, reviewer} {
		document, err := service.Document(ctx, allowed, item.DocumentID)
		requireNoError(t, err)
		serialized, _ := json.Marshal(document)
		if document.OriginalPath != "" || document.RelativePath != "" || strings.Contains(string(serialized), service.Identity.Config.PrivateUploadDirectory()) || document.Availability != "staged" {
			t.Fatal("temporary path leaked")
		}
		file, _, err := service.OpenDocument(ctx, allowed, item.DocumentID, false, domain.RequestMetadata{})
		requireNoError(t, err)
		file.Close()
		if query(t, service, allowed, libraryID, "metálica").Count != 1 {
			t.Fatal("owner/reviewer lost OCR")
		}
	}
	requireNoError(t, service.SetMember(ctx, admin, libraryID, reviewer.User.ID, []string{"library_reader"}, domain.RequestMetadata{}))
	if _, err = service.Document(ctx, reviewer, item.DocumentID); err == nil {
		t.Fatal("revoked reviewer still sees upload")
	}
}
func TestH4UploadValidationRecoveryAndRootOverlap(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	libraryID := managedLibrary(t, service, principal, "hybrid")
	batch, err := service.CreateBatch(ctx, principal, libraryID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	for _, body := range []string{"not a pdf", "%PDF- fake"} {
		if _, err = service.Upload(ctx, principal, batch, domain.NewID(), "Mal.pdf", strings.NewReader(body), nil, domain.RequestMetadata{}); err == nil {
			t.Fatal("invalid PDF accepted")
		}
	}
	entries, err := os.ReadDir(service.Identity.Config.PrivateUploadDirectory())
	requireNoError(t, err)
	if len(entries) != 0 {
		t.Fatal("failed upload left files")
	}
	locator := domain.NewID() + ".pdf"
	identifier := domain.NewID()
	_, err = service.Database.Writer.Exec("INSERT INTO upload_items(id,batch_id,library_id,client_file_id,original_filename,private_temporary_locator,status,created_at) VALUES(?,?,?,?,?,?,'receiving',?)", identifier, batch, libraryID, domain.NewID(), "Interrumpido.pdf", locator, now())
	requireNoError(t, err)
	requireNoError(t, os.WriteFile(filepath.Join(service.Identity.Config.PrivateUploadDirectory(), locator), []byte("partial"), 0600))
	requireNoError(t, service.recoverUploads(ctx))
	if _, err = os.Stat(filepath.Join(service.Identity.Config.PrivateUploadDirectory(), locator)); !os.IsNotExist(err) {
		t.Fatal("restart left receiving file")
	}
	linked := filepath.Join(directory, "linked")
	requireNoError(t, os.MkdirAll(linked, 0700))
	addRoot(t, service, principal, libraryID, linked)
	requireNoError(t, os.MkdirAll(filepath.Join(linked, "nested"), 0700))
	if _, err = service.PlanStorageRoot(ctx, principal, libraryID, filepath.Join(linked, "nested"), "managed", domain.RequestMetadata{}); err == nil {
		t.Fatal("managed nested under linked")
	}
	managed := filepath.Join(directory, "managed")
	requireNoError(t, os.MkdirAll(managed, 0700))
	plan, err := service.PlanStorageRoot(ctx, principal, libraryID, managed, "managed", domain.RequestMetadata{})
	requireNoError(t, err)
	rootID, err := service.ConfirmRoot(ctx, principal, libraryID, plan.ID, plan.Revision, false, domain.RequestMetadata{})
	requireNoError(t, err)
	var jobs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE job_type='scan' AND target_version=?", rootID).Scan(&jobs))
	if jobs != 0 {
		t.Fatal("managed root incorrectly indexed as linked")
	}
	if _, err = service.PlanRoot(ctx, principal, libraryID, managed, domain.RequestMetadata{}); err == nil {
		t.Fatal("linked overlapped managed")
	}
}

func TestHybridLinkedScanPreservesPrivateClassifiedUploads(t *testing.T) {
	service, owner, directory := fixture(t)
	ctx := context.Background()
	libraryID := managedLibrary(t, service, owner, "hybrid")
	reader := member(t, service, owner, libraryID, "hybrid-reader", "library_reader")
	category, kind := organizationCatalog(t, service, owner, libraryID, true, false)
	caseID, err := service.SaveCase(ctx, owner, libraryID, "", Case{Identifier: "HYBRID-001"}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	item := uploadFixture(t, service, owner, libraryID)
	requireNoError(t, service.Classify(ctx, owner, item.DocumentID, "classification", Classification{CaseID: caseID, CategoryID: category, TypeID: kind, Title: "Carga privada clasificada"}, 1, domain.RequestMetadata{}))
	drain(t, service)
	before, err := service.Document(ctx, owner, item.DocumentID)
	requireNoError(t, err)
	path := filepath.Join(directory, "new-linked-root")
	copyFixture(t, filepath.Join(path, "linked.pdf"), "native.pdf")
	rootID := addRoot(t, service, owner, libraryID, path)
	scan(t, service, rootID)
	after, err := service.Document(ctx, owner, item.DocumentID)
	requireNoError(t, err)
	if after.Availability != "staged" || !after.Preview || after.FileID != before.FileID || after.Hash != before.Hash || after.CaseID != caseID || after.Revision != before.Revision || after.Pages != before.Pages {
		t.Fatalf("linked scan changed private upload: before=%+v after=%+v", before, after)
	}
	file, _, err := service.OpenDocument(ctx, owner, item.DocumentID, false, domain.RequestMetadata{})
	requireNoError(t, err)
	file.Close()
	if _, err := service.Document(ctx, reader, item.DocumentID); err == nil {
		t.Fatal("linked scan exposed private metadata")
	}
	result, err := service.Search(ctx, reader, SearchInput{Query: "metálica", Type: "general", Libraries: []string{libraryID}, Filters: Filters{Source: "managed"}}, domain.RequestMetadata{}, true)
	requireNoError(t, err)
	if result.Count != 0 {
		t.Fatal("linked scan exposed private OCR")
	}
	drain(t, service)
	// A later scan still discovers real linked absence and leaves queued uploads alone.
	queued := uploadFixture(t, service, owner, libraryID)
	requireNoError(t, os.Remove(filepath.Join(path, "linked.pdf")))
	scan(t, service, rootID)
	queuedDocument, err := service.Document(ctx, owner, queued.DocumentID)
	requireNoError(t, err)
	if queuedDocument.Availability != "staged" {
		t.Fatal("scan changed an upload awaiting OCR")
	}
	var missing, falseObservations int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM physical_files WHERE library_id=? AND storage_source='linked' AND availability='missing'", libraryID).Scan(&missing))
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE event_type='document.linked_missing' AND document_id IN (?,?)", item.DocumentID, queued.DocumentID).Scan(&falseObservations))
	if missing != 1 || falseObservations != 0 {
		t.Fatalf("incorrect absence scope: linked=%d upload events=%d", missing, falseObservations)
	}
	drain(t, service)
	queuedDocument, err = service.Document(ctx, owner, queued.DocumentID)
	requireNoError(t, err)
	if queuedDocument.Availability != "staged" || queuedDocument.Freshness != "current" || queuedDocument.Pages == 0 {
		t.Fatal("linked scan prevented pending upload extraction")
	}
}

func TestPrivateUploadPrivacySurvivesAvailabilityDamage(t *testing.T) {
	service, owner, _ := fixture(t)
	ctx := context.Background()
	libraryID := managedLibrary(t, service, owner, "hybrid")
	reader := member(t, service, owner, libraryID, "private-reader", "library_reader")
	reviewer := member(t, service, owner, libraryID, "private-reviewer", "library_reviewer")
	item := uploadFixture(t, service, owner, libraryID)
	drain(t, service)
	for _, availability := range []string{"missing", "unknown", "available"} {
		_, err := service.Database.Writer.Exec("UPDATE physical_files SET availability=? WHERE library_id=?", availability, libraryID)
		requireNoError(t, err)
		if _, err = service.Document(ctx, reader, item.DocumentID); err == nil {
			t.Fatal("inconsistent temporary exposed metadata", availability)
		}
		if _, err = service.Page(ctx, reader, item.DocumentID, 1); err == nil {
			t.Fatal("inconsistent temporary exposed page text", availability)
		}
		if query(t, service, reader, libraryID, "metálica").Count != 0 {
			t.Fatal("inconsistent temporary exposed search results", availability)
		}
		for _, permitted := range []domain.Principal{owner, reviewer} {
			if query(t, service, permitted, libraryID, "metálica").Count != 1 {
				t.Fatal("author/reviewer lost access to retained text", availability)
			}
		}
	}
}
