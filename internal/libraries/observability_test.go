//go:build development

package libraries

import (
	"context"
	"encoding/json"
	"fmt"
	"gestor-documental/internal/diagnostics"
	"gestor-documental/internal/domain"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestObservabilityDashboardCountsDatesAndPublishedContent(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	// A fixed local Tuesday tests day/week/month boundaries independently of UTC.
	instant := time.Date(2026, 10, 6, 0, 30, 0, 0, time.FixedZone("CST", -6*3600))
	managed := managedLibrary(t, s, p, "managed")
	allowContentFormats(t, s, p)
	dates := []time.Time{instant, instant.Add(-time.Hour), instant.AddDate(0, 0, -2), instant.AddDate(0, 0, -10), instant}
	ids := []string{}
	for n, date := range dates {
		item := uploadFormat(t, s, p, managed, fmt.Sprintf("private-%d.txt", n), []byte("Contenido idéntico de prueba"))
		ids = append(ids, item.DocumentID)
		_, err := s.Database.Writer.Exec("UPDATE documents SET created_at=? WHERE id=?", domain.Timestamp(date), item.DocumentID)
		requireNoError(t, err)
	}
	for n := 0; n < 2; n++ {
		job, err := s.claim(ctx)
		requireNoError(t, err)
		requireNoError(t, s.Extract(ctx, job, s.Identity.Config.Indexing))
		requireNoError(t, s.finish(ctx, job, nil))
	}
	job, err := s.claim(ctx)
	requireNoError(t, err)
	requireNoError(t, s.finish(ctx, job, scanFailure("EXTRACTION_FAILED")))
	_, err = s.Database.Writer.Exec("UPDATE job_attempts SET started_at=?,finished_at=? WHERE error_code=''", domain.Timestamp(instant.Add(-30*time.Second)), domain.Timestamp(instant.Add(-10*time.Second)))
	requireNoError(t, err)
	linked := addLibrary(t, s, p, "Carpeta")
	copyFixture(t, filepath.Join(dir, "root", "doc.pdf"), "native.pdf")
	root := addRoot(t, s, p, linked, filepath.Join(dir, "root"))
	scan(t, s, root)
	_, err = s.Database.Writer.Exec("UPDATE documents SET created_at=? WHERE library_id=?", domain.Timestamp(instant), linked)
	requireNoError(t, err)
	s.Identity.Now = func() time.Time { return instant }
	report, err := s.Dashboard(ctx, p)
	requireNoError(t, err)
	if report.Total != 6 || report.Today != 3 || report.Week != 4 || report.Month != 5 || report.Linked != 1 || report.Managed != 5 || report.Indexed != 2 {
		t.Fatalf("counts: %+v", report)
	}
	if len(report.Daily) != 30 || report.Daily[29].Date != "2026-10-06" || report.Daily[29].Count != 3 || report.Daily[28].Count != 1 || report.Daily[27].Count != 1 || report.Daily[19].Count != 1 {
		t.Fatal("daily chart disagrees with local dates", report.Daily)
	}
	if report.DuplicateGroups != 1 || report.DuplicateFiles != 5 || report.ExtraCopies != 4 {
		t.Fatal("duplicate counts", report)
	}
	if report.TimingSamples != 2 || report.AverageSeconds == nil || *report.AverageSeconds < 19.9 || *report.AverageSeconds > 20.1 {
		t.Fatal("timings", report)
	}
	totals := map[string]int{}
	for _, g := range report.States {
		totals[g.Key] = g.Count
	}
	if totals["completed"] != 2 || totals["error"] != 1 || totals["pending"] != 3 {
		t.Fatal(totals)
	}
	libraries, err := s.DashboardLibraries(ctx, p, "")
	requireNoError(t, err)
	if len(libraries.Items) != 2 {
		t.Fatal(libraries)
	}
	raw, _ := json.Marshal(report)
	if strings.Contains(string(raw), "private-") || strings.Contains(string(raw), "Contenido idéntico") {
		t.Fatal("document content leaked")
	}
	if _, err = s.Dashboard(ctx, domain.Principal{}); err == nil {
		t.Fatal("anonymous metrics")
	}
	_, err = s.Database.Writer.Exec("UPDATE documents SET deleted_at=? WHERE id=?", now(), ids[4])
	requireNoError(t, err)
	report, err = s.Dashboard(ctx, p)
	requireNoError(t, err)
	if report.Total != 5 || report.ExtraCopies != 3 {
		t.Fatal("retired counted", report)
	}
	if report.Daily[29].Count != 2 {
		t.Fatal("retired counted in daily chart", report.Daily)
	}
}

func TestDashboardDailyCalendarAcrossDST(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	zone, err := time.LoadLocation("America/New_York")
	requireNoError(t, err)
	instant := time.Date(2026, 11, 2, 0, 15, 0, 0, zone)
	library := managedLibrary(t, s, p, "managed")
	allowContentFormats(t, s, p)
	// Both 01:30 instants on the 25-hour day belong to November 1.
	dates := []string{"2026-11-01T01:30:00-04:00", "2026-11-01T01:30:00-05:00", "2026-11-02T00:10:00-05:00", "2026-11-02T01:00:00-05:00"}
	for _, date := range dates {
		item := uploadFormat(t, s, p, library, "calendar.txt", []byte("Calendar fixture"))
		at, err := time.Parse(time.RFC3339, date)
		requireNoError(t, err)
		_, err = s.Database.Writer.Exec("UPDATE documents SET created_at=? WHERE id=?", domain.Timestamp(at), item.DocumentID)
		requireNoError(t, err)
	}
	s.Identity.Now = func() time.Time { return instant }
	report, err := s.Dashboard(ctx, p)
	requireNoError(t, err)
	if len(report.Daily) != 30 || report.Daily[28].Count != 2 || report.Daily[29].Count != 1 || report.Today != 1 || report.Week != 1 {
		t.Fatal("DST/future timestamp bucketing", report)
	}
}

func TestObservabilityErrorLifecyclePrivacyPaginationAndRetention(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	jobID := domain.NewID()
	secret := "password=SECRET token=PRIVATE C:/private/document.txt"
	requireNoError(t, diagnostics.Record(ctx, s.Database, secret, secret, diagnostics.Context{JobID: jobID, RootID: secret, Operation: secret, RequestID: secret}))
	page, err := s.DiagnosticEvents(ctx, p, DiagnosticFilter{}, "")
	requireNoError(t, err)
	if len(page.Items) != 1 || page.Items[0].Code != "INTERNAL_ERROR" {
		t.Fatal(page)
	}
	event := page.Items[0]
	support, err := s.SupportDiagnostic(ctx, p, event.ID)
	requireNoError(t, err)
	raw, _ := json.Marshal(support)
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "PRIVATE") || strings.Contains(string(raw), "C:/") || !strings.Contains(string(raw), jobID) {
		t.Fatal(string(raw))
	}
	var stored string
	requireNoError(t, s.Database.Reader.QueryRow("SELECT context_json||code||message||module FROM diagnostic_events WHERE id=?", event.ID).Scan(&stored))
	if strings.Contains(stored, "SECRET") {
		t.Fatal("stored secret")
	}
	requireNoError(t, s.ReviewDiagnostic(ctx, p, event.ID, "reviewed", event.Revision, domain.RequestMetadata{}))
	if s.ReviewDiagnostic(ctx, p, event.ID, "open", event.Revision, domain.RequestMetadata{}) == nil {
		t.Fatal("obsolete revision accepted")
	}
	requireNoError(t, diagnostics.Record(ctx, s.Database, secret, secret, diagnostics.Context{JobID: jobID}))
	page, err = s.DiagnosticEvents(ctx, p, DiagnosticFilter{Status: "open"}, "")
	requireNoError(t, err)
	if len(page.Items) != 1 || page.Items[0].Occurrences != 2 {
		t.Fatal("not reopened", page)
	}
	for n := 0; n < 105; n++ {
		requireNoError(t, diagnostics.Record(ctx, s.Database, "processing", "OCR_FAILED", diagnostics.Context{JobID: domain.NewID()}))
	}
	page, err = s.DiagnosticEvents(ctx, p, DiagnosticFilter{Module: "processing"}, "")
	requireNoError(t, err)
	if len(page.Items) != 50 || page.Next == "" {
		t.Fatal(page)
	}
	next, err := s.DiagnosticEvents(ctx, p, DiagnosticFilter{Module: "processing"}, page.Next)
	requireNoError(t, err)
	if len(next.Items) != 50 || next.Items[0].ID == page.Items[0].ID {
		t.Fatal(next)
	}
	if _, err = s.DiagnosticEvents(ctx, p, DiagnosticFilter{Severity: "critical"}, page.Next); err == nil {
		t.Fatal("cursor crossed filters")
	}
	if _, err = s.DiagnosticEvents(ctx, p, DiagnosticFilter{From: "bad date"}, ""); err == nil {
		t.Fatal("invalid date")
	}
	if _, err = s.DiagnosticEvents(ctx, domain.Principal{}, DiagnosticFilter{}, ""); err == nil {
		t.Fatal("anonymous log")
	}
	if s.ConfigureDiagnostics(ctx, domain.Principal{}, diagnostics.Policy{Days: 30, Maximum: 100}, domain.RequestMetadata{}) == nil {
		t.Fatal("anonymous retention")
	}
	policy, err := s.DiagnosticPolicy(ctx, p)
	requireNoError(t, err)
	policy.Maximum = 100
	policy.Days = 2
	var audits int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM audit_events").Scan(&audits))
	requireNoError(t, s.ConfigureDiagnostics(ctx, p, policy, domain.RequestMetadata{}))
	if s.ConfigureDiagnostics(ctx, p, policy, domain.RequestMetadata{}) == nil {
		t.Fatal("stale retention accepted")
	}
	var count int
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM diagnostic_events").Scan(&count))
	if count != 100 {
		t.Fatal(count)
	}
	_, err = s.Database.Writer.Exec("UPDATE diagnostic_events SET last_occurred_at=?", domain.Timestamp(time.Now().Add(-72*time.Hour)))
	requireNoError(t, err)
	requireNoError(t, diagnostics.Prune(ctx, s.Database))
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM diagnostic_events").Scan(&count))
	if count != 0 {
		t.Fatal("retention ignored", count)
	}
	requireNoError(t, s.Database.Reader.QueryRow("SELECT count(*) FROM audit_events").Scan(&count))
	if count != audits+1 {
		t.Fatal("audit changed by cleanup", count, audits)
	}
	policy, err = New(s.Identity).DiagnosticPolicy(ctx, p)
	requireNoError(t, err)
	if policy.Days != 2 || policy.Maximum != 100 {
		t.Fatal("policy lost")
	}
}

func TestObservabilityEmptyAndLibraryPagination(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	empty, err := s.Dashboard(ctx, p)
	requireNoError(t, err)
	if empty.Total != 0 || empty.AverageSeconds != nil || empty.TimingSamples != 0 {
		t.Fatal(empty)
	}
	for n := 0; n < 26; n++ {
		addLibrary(t, s, p, fmt.Sprintf("Biblioteca %02d", n))
	}
	page, err := s.DashboardLibraries(ctx, p, "")
	requireNoError(t, err)
	if len(page.Items) != 25 || page.Next == "" {
		t.Fatal(page)
	}
	next, err := s.DashboardLibraries(ctx, p, page.Next)
	requireNoError(t, err)
	if len(next.Items) != 1 || next.Items[0].Total != 0 {
		t.Fatal(next)
	}
	if _, err = s.DiagnosticEvents(ctx, p, DiagnosticFilter{}, page.Next); err == nil {
		t.Fatal("cross scope cursor")
	}
}

func TestObservabilityConcurrentOccurrences(t *testing.T) {
	s, p, _ := fixture(t)
	ctx := context.Background()
	id := domain.NewID()
	var workers sync.WaitGroup
	errors := make(chan error, 40)
	for n := 0; n < 40; n++ {
		workers.Go(func() {
			errors <- diagnostics.Record(ctx, s.Database, "processing", "OCR_FAILED", diagnostics.Context{JobID: id})
		})
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		requireNoError(t, err)
	}
	page, err := s.DiagnosticEvents(ctx, p, DiagnosticFilter{}, "")
	requireNoError(t, err)
	if len(page.Items) != 1 || page.Items[0].Occurrences != 40 || page.Items[0].Revision != 40 {
		t.Fatalf("concurrent occurrences lost: %+v", page)
	}
}

func TestObservabilityOCRAndTimingExclusions(t *testing.T) {
	s, p, dir := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, s, p, "OCR")
	copyFixture(t, filepath.Join(dir, "root", "scanned.pdf"), "scanned.pdf")
	root := addRoot(t, s, p, library, filepath.Join(dir, "root"))
	scan(t, s, root)
	drain(t, s)
	report, err := s.Dashboard(ctx, p)
	requireNoError(t, err)
	if report.OCR != 1 || report.Indexed != 1 || report.TimingSamples != 1 {
		t.Fatalf("OCR was not counted: %+v", report)
	}
	_, err = s.Database.Writer.Exec("UPDATE job_attempts SET finished_at=?", domain.Timestamp(time.Now().Add(-48*time.Hour)))
	requireNoError(t, err)
	report, err = s.Dashboard(ctx, p)
	requireNoError(t, err)
	if report.AverageSeconds != nil || report.TimingSamples != 0 {
		t.Fatal("invalid/old duration was counted", report)
	}
	_, err = s.Database.Writer.Exec("UPDATE physical_files SET extraction_freshness='stale'")
	requireNoError(t, err)
	report, err = s.Dashboard(ctx, p)
	requireNoError(t, err)
	if report.OCR != 0 || report.Indexed != 0 {
		t.Fatal("old OCR counted as a current index", report)
	}
}
