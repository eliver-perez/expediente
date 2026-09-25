//go:build development

package libraries

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/storage"
)

type workflowFixture struct {
	service                  *Service
	author                   domain.Principal
	library, root, directory string
	document                 Document
}

func workflowSetup(t *testing.T, review bool) workflowFixture {
	t.Helper()
	ctx := context.Background()
	s, author, directory := fixture(t)
	library := managedLibrary(t, s, author, "hybrid")
	destination := filepath.Join(directory, "definitivos")
	requireNoError(t, os.MkdirAll(destination, 0700))
	plan, err := s.PlanStorageRoot(ctx, author, library, destination, "managed", domain.RequestMetadata{})
	requireNoError(t, err)
	root, err := s.ConfirmRoot(ctx, author, library, plan.ID, plan.Revision, false, domain.RequestMetadata{})
	requireNoError(t, err)
	settings, _, err := settingsFor(ctx, s.Database.Reader, library)
	requireNoError(t, err)
	settings.ManagedRootID = root
	settings.ReviewManaged = review
	settings.ReviewLinked = review
	settings.FilenamePattern = "{tipo_documento}_{consecutivo}"
	updateWorkflowSettings(t, s, author, library, settings)
	category, kind := organizationCatalog(t, s, author, library, true, false)
	version, err := s.SaveTemplate(ctx, author, library, "", "Ingreso", []RequirementInput{{CategoryID: category, TypeID: kind, Count: 1, Mandatory: true}}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	caseID, err := s.SaveCase(ctx, author, library, "", Case{Identifier: "PR-008", TemplateVersionID: version}, 0, domain.RequestMetadata{})
	requireNoError(t, err)
	item := uploadFixture(t, s, author, library)
	requireNoError(t, s.Classify(ctx, author, item.DocumentID, "classification", Classification{CaseID: caseID, CategoryID: category, TypeID: kind}, 1, domain.RequestMetadata{}))
	drain(t, s)
	document, err := s.Document(ctx, author, item.DocumentID)
	requireNoError(t, err)
	return workflowFixture{s, author, library, root, directory, document}
}
func updateWorkflowSettings(t *testing.T, s *Service, p domain.Principal, library string, settings Settings) {
	t.Helper()
	var revision int64
	requireNoError(t, s.Database.Reader.QueryRow("SELECT revision FROM libraries WHERE id=?", library).Scan(&revision))
	requireNoError(t, s.UpdateConfiguration(context.Background(), p, library, "Organización", "spa", "hybrid", &settings, revision, domain.RequestMetadata{}))
}
func workflowDocumentFor(t *testing.T, f workflowFixture) Document {
	t.Helper()
	d, err := f.service.Document(context.Background(), f.author, f.document.ID)
	requireNoError(t, err)
	return d
}
func temporaryPath(t *testing.T, f workflowFixture) string {
	t.Helper()
	var locator string
	requireNoError(t, f.service.Database.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE document_id=?", f.document.ID).Scan(&locator))
	return filepath.Join(f.service.Identity.Config.PrivateUploadDirectory(), locator)
}
func TestH5DirectPublicationAndManagedIntegrity(t *testing.T) {
	f := workflowSetup(t, false)
	s := f.service
	ctx := context.Background()
	reader := member(t, s, f.author, f.library, "lector-h5", "library_reader")
	temporary := temporaryPath(t, f)
	op, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	replay, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	if replay != op {
		t.Fatal("duplicate publication")
	}
	if d := workflowDocumentFor(t, f); d.Approval != "materializing" {
		t.Fatal(d)
	}
	if _, err = s.Document(ctx, reader, f.document.ID); err == nil {
		t.Fatal("temporary exposed")
	}
	if _, err = os.Stat(temporary); err != nil {
		t.Fatal("temporary removed before commit", err)
	}
	drain(t, s)
	d := workflowDocumentFor(t, f)
	if d.Approval != "approved" || d.Availability != "available" || d.Hash != f.document.Hash || d.Pages != f.document.Pages || d.Integrity != "verified" {
		t.Fatalf("bad approval: %+v", d)
	}
	if _, err = os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatal("temporary cleanup missing", err)
	}
	if _, err = s.Document(ctx, reader, d.ID); err != nil {
		t.Fatal("approved document private", err)
	}
	finalBytes, err := os.ReadFile(d.OriginalPath)
	requireNoError(t, err)
	original, err := os.ReadFile("../../testdata/documents/native.pdf")
	requireNoError(t, err)
	if string(finalBytes) != string(original) {
		t.Fatal("published bytes differ")
	}
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	d = workflowDocumentFor(t, f)
	if d.Approval != "approved" {
		t.Fatal("own write considered external")
	}
	var notices int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM document_notifications WHERE document_id=?", d.ID).Scan(&notices))
	if notices != 0 {
		t.Fatal("false change alert")
	}
	progress, err := s.Requirements(ctx, f.author, d.CaseID)
	requireNoError(t, err)
	if progress.Progress == nil || *progress.Progress != 100 {
		t.Fatal(progress)
	}
	copyFixture(t, d.OriginalPath, "native.pdf")
	file, err := os.OpenFile(d.OriginalPath, os.O_APPEND|os.O_WRONLY, 0600)
	requireNoError(t, err)
	_, err = file.WriteString("\n% external change\n")
	requireNoError(t, err)
	requireNoError(t, file.Close())
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	d = workflowDocumentFor(t, f)
	if d.Approval != "needs_review" || d.Integrity != "changed" || d.Freshness != "stale" || d.Pages != f.document.Pages {
		t.Fatalf("change unhandled: %+v", d)
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM document_notifications WHERE document_id=?", d.ID).Scan(&notices))
	if notices != 1 {
		t.Fatal("duplicate change notices", notices)
	}
	progress, err = s.Requirements(ctx, f.author, d.CaseID)
	requireNoError(t, err)
	if *progress.Progress != 0 {
		t.Fatal("changed managed counted valid")
	}
	_, err = s.FinalizeDocument(ctx, f.author, d.ID, d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	d = workflowDocumentFor(t, f)
	if d.Approval != "approved" || d.Integrity != "verified" {
		t.Fatal("new version not approved")
	}
	requireNoError(t, os.Remove(d.OriginalPath))
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	d = workflowDocumentFor(t, f)
	if d.Approval != "approved" || d.Availability != "missing" || d.Pages == 0 {
		t.Fatal("missing erased evidence")
	}
	progress, err = s.Requirements(ctx, f.author, d.CaseID)
	requireNoError(t, err)
	if *progress.Progress != 0 {
		t.Fatal("missing counted valid")
	}
}
func TestH5MaterializationFailureRecovery(t *testing.T) {
	for _, point := range []string{"before_publish", "after_publish", "before_commit", "after_commit", "before_cleanup"} {
		t.Run(point, func(t *testing.T) {
			f := workflowSetup(t, false)
			s := f.service
			ctx := context.Background()
			temporary := temporaryPath(t, f)
			op, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
			requireNoError(t, err)
			job, err := s.claim(ctx)
			requireNoError(t, err)
			if job.Kind != "materialize" {
				t.Fatal(job)
			}
			s.materializationFault = func(at string) error {
				if at == point {
					return errors.New("simulated stop")
				}
				return nil
			}
			if err = s.Materialize(ctx, job); err == nil {
				t.Fatal("failure hook not reached")
			}
			if _, err = os.Stat(temporary); err != nil {
				t.Fatal("temporary removed before cleanup", err)
			}
			d := workflowDocumentFor(t, f)
			want := "materializing"
			if point == "after_commit" || point == "before_cleanup" {
				want = "approved"
			}
			if d.Approval != want {
				t.Fatal(d.Approval, want)
			}
			// Reopen the actual SQLite file and use the same restart recovery as production.
			configuration := s.Identity.Config
			requireNoError(t, s.Database.Close())
			reopened, err := storage.Open(ctx, configuration.StateDirectory)
			requireNoError(t, err)
			t.Cleanup(func() { reopened.Close() })
			account, err := identity.New(reopened, configuration)
			requireNoError(t, err)
			resumed := New(account)
			requireNoError(t, resumed.recoverJobs(ctx))
			next, err := resumed.claim(ctx)
			requireNoError(t, err)
			if err = resumed.Materialize(ctx, job); err == nil {
				t.Fatal("stale worker accepted")
			}
			s = resumed
			f.service = resumed
			requireNoError(t, resumed.Materialize(ctx, next))
			requireNoError(t, resumed.finish(ctx, next, nil))
			requireNoError(t, resumed.finish(ctx, job, errors.New("late stale worker")))
			materialization, err := resumed.Materialization(ctx, f.author, op)
			requireNoError(t, err)
			if materialization.State != "cleaned" || materialization.Error != "" {
				t.Fatal(materialization)
			}
			var locations, events int
			requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM physical_file_locations WHERE physical_file_id=?", d.FileID).Scan(&locations))
			requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE document_id=? AND event_type='document.approved'", d.ID).Scan(&events))
			if locations != 1 || events != 1 {
				t.Fatal("duplicate commit", locations, events)
			}
			if _, err = os.Stat(temporary); !os.IsNotExist(err) {
				t.Fatal("temporary survived successful recovery")
			}
		})
	}
}

func TestH5ReviewAssignmentDecisionsAndRetention(t *testing.T) {
	f := workflowSetup(t, true)
	ctx := context.Background()
	s := f.service
	reviewer := member(t, s, f.author, f.library, "reviewer-h5", "library_reviewer")
	other := member(t, s, f.author, f.library, "other-reviewer-h5", "library_reviewer")
	reader := member(t, s, f.author, f.library, "reader-review-h5", "library_reader")
	requireNoError(t, s.SetMember(ctx, f.author, f.library, f.author.User.ID, []string{"library_manager", "library_reviewer"}, domain.RequestMetadata{}))
	d := f.document
	if _, err := s.FinalizeDocument(ctx, f.author, d.ID, d.Revision, domain.RequestMetadata{}); err == nil {
		t.Fatal("review policy bypassed")
	}
	review, err := s.SubmitReview(ctx, f.author, d.ID, reviewer.User.ID, d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	replay, err := s.SubmitReview(ctx, f.author, d.ID, reviewer.User.ID, d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	if replay != review {
		t.Fatal("duplicate review")
	}
	d = workflowDocumentFor(t, f)
	queue, err := s.PendingReviews(ctx, other, f.library, "")
	requireNoError(t, err)
	if len(queue.Items) != 0 {
		t.Fatal("assigned queue leaked")
	}
	queue, err = s.PendingReviews(ctx, reviewer, f.library, "")
	requireNoError(t, err)
	if len(queue.Items) != 1 {
		t.Fatal("assigned review missing")
	}
	if _, err = s.DecideReview(ctx, other, review, d.Revision, true, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("other reviewer approved assigned review")
	}
	if _, err = s.DecideReview(ctx, reviewer, review, d.Revision, false, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("rejected without reason")
	}
	// A classification change supersedes the exact reviewed revision.
	requireNoError(t, s.Classify(ctx, f.author, d.ID, "classification", Classification{CaseID: d.CaseID, CategoryID: d.CategoryID, TypeID: d.TypeID, Title: "Corregido"}, d.Revision, domain.RequestMetadata{}))
	if _, err = s.DecideReview(ctx, reviewer, review, d.Revision, true, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("approved obsolete classification")
	}
	d = workflowDocumentFor(t, f)
	review, err = s.SubmitReview(ctx, f.author, d.ID, "", d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	d = workflowDocumentFor(t, f)
	if _, err = s.DecideReview(ctx, f.author, review, d.Revision, true, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("self-review accepted")
	}
	if _, err = s.DecideReview(ctx, reader, review, d.Revision, true, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("reader approved")
	}
	_, err = s.DecideReview(ctx, reviewer, review, d.Revision, false, "Falta corregir el título", domain.RequestMetadata{})
	requireNoError(t, err)
	_, err = s.DecideReview(ctx, reviewer, review, d.Revision, false, "Falta corregir el título", domain.RequestMetadata{})
	requireNoError(t, err)
	requireNoError(t, s.cleanExpiredUploads(ctx))
	if _, err = os.Stat(temporaryPath(t, f)); err != nil {
		t.Fatal("default retention deleted temporary")
	}
	settings, _, err := settingsFor(ctx, s.Database.Reader, f.library)
	requireNoError(t, err)
	settings.RetentionDays = 1
	updateWorkflowSettings(t, s, f.author, f.library, settings)
	d = workflowDocumentFor(t, f)
	review, err = s.SubmitReview(ctx, f.author, d.ID, "", d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	d = workflowDocumentFor(t, f)
	_, err = s.DecideReview(ctx, reviewer, review, d.Revision, false, "Retención programada", domain.RequestMetadata{})
	requireNoError(t, err)
	requireNoError(t, s.cleanExpiredUploads(ctx))
	if _, err = os.Stat(temporaryPath(t, f)); err != nil {
		t.Fatal("deleted before deadline")
	}
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE upload_items SET retain_until='2000-01-01' WHERE document_id=?", d.ID)
		return err
	}))
	// Resubmission cancels scheduled retention; approval starts one recoverable publication.
	d = workflowDocumentFor(t, f)
	review, err = s.SubmitReview(ctx, f.author, d.ID, "", d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	requireNoError(t, s.cleanExpiredUploads(ctx))
	d = workflowDocumentFor(t, f)
	op, err := s.DecideReview(ctx, reviewer, review, d.Revision, true, "", domain.RequestMetadata{})
	requireNoError(t, err)
	duplicate, err := s.DecideReview(ctx, reviewer, review, d.Revision, true, "", domain.RequestMetadata{})
	requireNoError(t, err)
	if duplicate != op {
		t.Fatal("duplicate approval")
	}
	drain(t, s)
	duplicate, err = s.DecideReview(ctx, reviewer, review, d.Revision, true, "", domain.RequestMetadata{})
	requireNoError(t, err)
	if duplicate != op {
		t.Fatal("completed approval replay failed")
	}
	if _, err = s.Document(ctx, reader, d.ID); err != nil {
		t.Fatal("reader cannot see approved document", err)
	}
	// Expired rejected uploads retain searchable evidence for author/reviewers only.
	item := uploadFixture(t, s, f.author, f.library)
	drain(t, s)
	requireNoError(t, s.CancelUpload(ctx, f.author, item.DocumentID, "Ya no se requiere", 1, domain.RequestMetadata{}))
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("UPDATE upload_items SET retain_until='2000-01-01' WHERE document_id=?", item.DocumentID)
		return err
	}))
	requireNoError(t, s.cleanExpiredUploads(ctx))
	retained, err := s.Document(ctx, f.author, item.DocumentID)
	requireNoError(t, err)
	if retained.Availability != "missing" || retained.Pages == 0 {
		t.Fatal("retention lost evidence")
	}
	if _, err = s.Document(ctx, reader, item.DocumentID); err == nil {
		t.Fatal("retention exposed cancelled document")
	}
	if _, err = s.Document(ctx, reviewer, item.DocumentID); err != nil {
		t.Fatal("retention hid evidence from reviewer", err)
	}
}
func TestH5PublicationCollisionAndFrozenNaming(t *testing.T) {
	f := workflowSetup(t, false)
	s := f.service
	ctx := context.Background()
	opID, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	op, err := s.Materialization(ctx, f.author, opID)
	requireNoError(t, err)
	root, err := s.root(ctx, f.root)
	requireNoError(t, err)
	path := filepath.Join(root.Path, filepath.FromSlash(op.Relative))
	requireNoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	requireNoError(t, os.WriteFile(path, []byte("existing file"), 0600))
	job, err := s.claim(ctx)
	requireNoError(t, err)
	err = s.Materialize(ctx, job)
	if err == nil {
		t.Fatal("overwrote collision")
	}
	requireNoError(t, s.finish(ctx, job, err))
	contents, err := os.ReadFile(path)
	requireNoError(t, err)
	if string(contents) != "existing file" {
		t.Fatal("changed colliding file")
	}
	if _, err = os.Stat(temporaryPath(t, f)); err != nil {
		t.Fatal("lost temporary")
	}
	settings, _, err := settingsFor(ctx, s.Database.Reader, f.library)
	requireNoError(t, err)
	settings.FilenamePattern = "original"
	settings.StructurePattern = "{categoria}"
	updateWorkflowSettings(t, s, f.author, f.library, settings)
	requireNoError(t, os.Remove(path))
	retry, err := s.RetryMaterialization(ctx, f.author, opID, "Colisión retirada", domain.RequestMetadata{})
	requireNoError(t, err)
	if retry == job.ID {
		t.Fatal("retry lost provenance")
	}
	drain(t, s)
	d := workflowDocumentFor(t, f)
	if d.OriginalPath != path {
		t.Fatal("pending journal changed naming")
	}
	settings.FilenamePrefix = "Nuevo"
	updateWorkflowSettings(t, s, f.author, f.library, settings)
	d = workflowDocumentFor(t, f)
	if d.OriginalPath != path {
		t.Fatal("configuration moved historic file")
	}
}
func TestH5LinkedApprovalKeepsOriginalAndApprovalOnExternalChange(t *testing.T) {
	f := workflowSetup(t, true)
	s := f.service
	ctx := context.Background()
	reviewer := member(t, s, f.author, f.library, "linked-reviewer-h5", "library_reviewer")
	path := filepath.Join(f.directory, "linked", "Original.pdf")
	copyFixture(t, path, "native.pdf")
	root := addRoot(t, s, f.author, f.library, filepath.Dir(path))
	drain(t, s)
	result := query(t, s, f.author, f.library, "Original.pdf")
	if len(result.Items) != 1 {
		t.Fatal(result)
	}
	d := result.Items[0]
	requireNoError(t, s.Classify(ctx, f.author, d.ID, "associate", Classification{CaseID: f.document.CaseID, CategoryID: f.document.CategoryID, TypeID: f.document.TypeID}, d.Revision, domain.RequestMetadata{}))
	d, err := s.Document(ctx, f.author, d.ID)
	requireNoError(t, err)
	review, err := s.SubmitReview(ctx, f.author, d.ID, "", d.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	d, err = s.Document(ctx, f.author, d.ID)
	requireNoError(t, err)
	_, err = s.DecideReview(ctx, reviewer, review, d.Revision, true, "", domain.RequestMetadata{})
	requireNoError(t, err)
	contents, err := os.ReadFile(path)
	requireNoError(t, err)
	original, err := os.ReadFile("../../testdata/documents/native.pdf")
	requireNoError(t, err)
	if string(contents) != string(original) {
		t.Fatal("approval modified linked original")
	}
	copyFixture(t, path, "scanned.pdf")
	scan(t, s, root)
	d, err = s.Document(ctx, f.author, d.ID)
	requireNoError(t, err)
	if d.Approval != "approved" || d.Freshness != "stale" {
		t.Fatal("linked change revoked approval")
	}
}

func TestH5PublicationRejectsMissingDestinationAndRevokedAuthority(t *testing.T) {
	for _, scenario := range []string{"missing_before_commit", "authority_revoked", "changed_before_cleanup"} {
		t.Run(scenario, func(t *testing.T) {
			f := workflowSetup(t, false)
			s := f.service
			ctx := context.Background()
			operationID, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
			requireNoError(t, err)
			operation, err := s.Materialization(ctx, f.author, operationID)
			requireNoError(t, err)
			root, err := s.root(ctx, f.root)
			requireNoError(t, err)
			path := filepath.Join(root.Path, filepath.FromSlash(operation.Relative))
			if err = s.RemoveIndexConfirmed(ctx, f.author, f.document.ID, "No debe retirarse", f.document.CaseID, f.document.Revision+1, domain.RequestMetadata{}); err == nil {
				t.Fatal("removed materializing document")
			}
			plan, err := s.RetirementPlan(ctx, f.author, f.root)
			requireNoError(t, err)
			if err = s.Retire(ctx, f.author, f.root, plan["plan_id"].(string), "retain_index", domain.RequestMetadata{}); err == nil {
				t.Fatal("retired pending destination")
			}
			if scenario == "authority_revoked" {
				member(t, s, f.author, f.library, "replacement-manager", "library_manager")
				requireNoError(t, s.SetMember(ctx, f.author, f.library, f.author.User.ID, []string{"library_reader"}, domain.RequestMetadata{}))
			}
			s.materializationFault = func(point string) error {
				if scenario == "missing_before_commit" && point == "before_commit" {
					return os.Remove(path)
				}
				if scenario == "changed_before_cleanup" && point == "before_cleanup" {
					return os.WriteFile(path, []byte("external alteration"), 0600)
				}
				return nil
			}
			job, err := s.claim(ctx)
			requireNoError(t, err)
			err = s.Materialize(ctx, job)
			if err == nil {
				t.Fatal("unsafe operation accepted")
			}
			requireNoError(t, s.finish(ctx, job, err))
			if _, err = os.Stat(temporaryPath(t, f)); err != nil {
				t.Fatal("lost source evidence", err)
			}
			d := workflowDocumentFor(t, f)
			if scenario != "changed_before_cleanup" && d.Approval != "materializing" {
				t.Fatal("approved without valid destination/authority")
			}
			if scenario == "changed_before_cleanup" {
				requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
				d = workflowDocumentFor(t, f)
				if d.Approval != "needs_review" {
					t.Fatal("external alteration not recognized")
				}
			}
		})
	}
}
func TestH5SelfReviewExceptionIsExplicitAndAudited(t *testing.T) {
	f := workflowSetup(t, true)
	s := f.service
	ctx := context.Background()
	requireNoError(t, s.SetMember(ctx, f.author, f.library, f.author.User.ID, []string{"library_manager", "library_reviewer"}, domain.RequestMetadata{}))
	review, err := s.SubmitReview(ctx, f.author, f.document.ID, "", f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	d := workflowDocumentFor(t, f)
	if _, err = s.DecideReview(ctx, f.author, review, d.Revision, true, "", domain.RequestMetadata{}); err == nil {
		t.Fatal("implicit self approval")
	}
	requireNoError(t, s.Database.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO role_permissions VALUES('library_reviewer','documents.approve_own')")
		return err
	}))
	_, err = s.DecideReview(ctx, f.author, review, d.Revision, true, "", domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	var count int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE document_id=? AND event_type='document.self_review_exception'", d.ID).Scan(&count))
	if count != 1 {
		t.Fatal("exception unaudited", count)
	}
}
