//go:build development

package libraries

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"gestor-documental/internal/domain"
	"gestor-documental/internal/licensefixture"
	"gestor-documental/internal/licensing"
)

func attachLicense(t *testing.T, service *Service) licensing.Claims {
	t.Helper()
	client, err := licensing.New(service.Database, service.Identity.Config.StateDirectory, licensing.Options{TrustedKeys: []licensing.TrustKey{licensefixture.TrustKey()}})
	requireNoError(t, err)
	service.Identity.License = client
	state, err := client.Status(context.Background())
	requireNoError(t, err)
	claims := licensefixture.Claims(state.Binding, time.Now())
	applyLicense(t, service, claims)
	return claims
}
func applyLicense(t *testing.T, service *Service, claims licensing.Claims) {
	t.Helper()
	_, err := service.Identity.License.Import(context.Background(), []byte(licensefixture.Sign(claims)), nil, domain.RequestMetadata{})
	requireNoError(t, err)
}
func TestH6ReadOnlyPreservesDownloadsAndManagedIntegrity(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	drain(t, s)
	original := workflowDocumentFor(t, f)
	claims := attachLicense(t, s)
	claims.Revision++
	claims.Type = "subscription"
	claims.GraceDays = 15
	expired := time.Now().Add(-16 * 24 * time.Hour).UTC().Format(time.RFC3339)
	claims.ExpiresAt = &expired
	applyLicense(t, s, claims)
	document, err := s.Document(ctx, f.author, f.document.ID)
	requireNoError(t, err)
	if document.Pages != original.Pages || document.Hash != original.Hash {
		t.Fatal("read-only lost document data")
	}
	file, _, err := s.OpenDocument(ctx, f.author, document.ID, true, domain.RequestMetadata{})
	requireNoError(t, err)
	file.Close()
	if _, err = s.CreateBatch(ctx, f.author, f.library, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "LICENSE_READ_ONLY" {
		t.Fatal("expired license allowed upload", err)
	}
	if err = s.Classify(ctx, f.author, document.ID, "classification", Classification{CaseID: document.CaseID, CategoryID: document.CategoryID, TypeID: document.TypeID}, document.Revision, domain.RequestMetadata{}); failureCode(err) != "LICENSE_READ_ONLY" {
		t.Fatal("expired license allowed classification", err)
	}
	bytes, err := os.ReadFile(document.OriginalPath)
	requireNoError(t, err)
	requireNoError(t, os.WriteFile(document.OriginalPath, append(bytes, []byte("\n% changed after expiry\n")...), 0600))
	past := time.Now().Add(-5 * time.Second)
	requireNoError(t, os.Chtimes(document.OriginalPath, past, past))
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	document = workflowDocumentFor(t, f)
	if document.Approval != "needs_review" {
		t.Fatal("read-only disabled integrity observation", document.Approval)
	}
	claims.Revision++
	claims.Type = "perpetual"
	claims.ExpiresAt = nil
	claims.GraceDays = 0
	claims.Features["managed_libraries"] = false
	claims.Features["expedientes"] = false
	claims.Features["review_workflow"] = false
	applyLicense(t, s, claims)
	if _, err = s.CreateBatch(ctx, f.author, f.library, domain.NewID(), domain.RequestMetadata{}); failureCode(err) != "LICENSE_FEATURE" {
		t.Fatal("hybrid wrote with missing managed module", err)
	}
	requireNoError(t, s.VerifyManagedRoot(ctx, f.root))
	_, err = s.Document(ctx, f.author, document.ID)
	requireNoError(t, err)
}
func TestH6WorkerPausesWithoutBurningAttemptsAndResumesAfterModuleRenewal(t *testing.T) {
	ctx := context.Background()
	s, principal, _ := fixture(t)
	library := managedLibrary(t, s, principal, "managed")
	item := uploadFixture(t, s, principal, library)
	claims := attachLicense(t, s)
	claims.Revision++
	claims.Features["ocr"] = false
	applyLicense(t, s, claims)
	for i := 0; i < 7; i++ {
		_, err := s.claim(ctx)
		if err != sql.ErrNoRows {
			t.Fatal("unlicensed extraction claimed", err)
		}
	}
	var state string
	var attempts int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT status,attempt_count FROM jobs WHERE physical_file_id=(SELECT physical_file_id FROM documents WHERE id=?) AND job_type='extract'", item.DocumentID).Scan(&state, &attempts))
	if state != "paused" || attempts != 0 {
		t.Fatal("license pause consumed retries", state, attempts)
	}
	claims.Revision++
	claims.Features["ocr"] = true
	applyLicense(t, s, claims)
	_, errReset := s.Database.Writer.Exec("UPDATE jobs SET available_at=? WHERE status='paused'", now())
	requireNoError(t, errReset)
	job, err := s.claim(ctx)
	requireNoError(t, err)
	if job.Attempts != 1 || job.Kind != "extract" {
		t.Fatal(job)
	}
	// A license change during execution also pauses publication and preserves the
	// previous attempt as evidence without consuming the operational retry budget.
	claims.Revision++
	claims.Features["ocr"] = false
	applyLicense(t, s, claims)
	err = s.Extract(ctx, job, s.Identity.Config.Indexing)
	if failureCode(err) != "LICENSE_FEATURE" {
		t.Fatal(err)
	}
	requireNoError(t, s.finish(ctx, job, err))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT status,attempt_count FROM jobs WHERE retry_of_job_id=?", job.ID).Scan(&state, &attempts))
	if state != "paused" || attempts != 0 {
		t.Fatal("in-flight license pause consumed retry budget")
	}
	claims.Revision++
	claims.Features["ocr"] = true
	applyLicense(t, s, claims)
	drain(t, s)
	document, err := s.Document(ctx, principal, item.DocumentID)
	requireNoError(t, err)
	if document.Pages == 0 {
		t.Fatal("renewal did not resume extraction")
	}
}
func TestH6MaterializationPausesBeforePublishingOnLostModule(t *testing.T) {
	ctx := context.Background()
	f := workflowSetup(t, false)
	s := f.service
	claims := attachLicense(t, s)
	_, err := s.FinalizeDocument(ctx, f.author, f.document.ID, f.document.Revision, domain.RequestMetadata{})
	requireNoError(t, err)
	var job Job
	for i := 0; i < 10; i++ {
		job, err = s.claim(ctx)
		requireNoError(t, err)
		if job.Kind == "materialize" {
			break
		}
		requireNoError(t, s.finish(ctx, job, nil))
	}
	if job.Kind != "materialize" {
		t.Fatal("missing materialization")
	}
	claims.Revision++
	claims.Features["expedientes"] = false
	claims.Features["review_workflow"] = false
	applyLicense(t, s, claims)
	err = s.Materialize(ctx, job)
	if failureCode(err) != "LICENSE_FEATURE" {
		t.Fatal("materialized a case after losing module", err)
	}
	requireNoError(t, s.finish(ctx, job, err))
	document := workflowDocumentFor(t, f)
	if document.Approval != "materializing" || document.Availability != "staged" {
		t.Fatal("license failure changed document", document)
	}
	if _, err = os.Stat(temporaryPath(t, f)); err != nil {
		t.Fatal("license loss removed upload", err)
	}
	claims.Revision++
	claims.Features["expedientes"] = true
	claims.Features["review_workflow"] = true
	applyLicense(t, s, claims)
	drain(t, s)
	document = workflowDocumentFor(t, f)
	if document.Approval != "approved" || document.Availability != "available" {
		t.Fatal("materialization did not resume", document)
	}
}
