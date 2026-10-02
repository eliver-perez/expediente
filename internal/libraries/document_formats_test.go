//go:build development

package libraries

import (
	"bytes"
	"context"
	"gestor-documental/internal/documentformat"
	"gestor-documental/internal/domain"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func uploadFormat(t *testing.T, service *Service, principal domain.Principal, libraryID, name string, contents []byte) UploadItem {
	t.Helper()
	ctx := context.Background()
	batch, err := service.CreateBatch(ctx, principal, libraryID, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	item, err := service.Upload(ctx, principal, batch, domain.NewID(), name, bytes.NewReader(contents), nil, domain.RequestMetadata{})
	requireNoError(t, err)
	return item
}

func TestMultiformatManagedRoundTripAndCasePublication(t *testing.T) {
	f := workflowSetup(t, false)
	service := f.service
	ctx := context.Background()
	for _, format := range []string{"docx", "xlsx", "txt", "csv"} {
		t.Run(format, func(t *testing.T) {
			contents, err := os.ReadFile("../../testdata/documents/sample." + format)
			requireNoError(t, err)
			item := uploadFormat(t, service, f.author, f.library, "Original."+format, contents)
			document, err := service.Document(ctx, f.author, item.DocumentID)
			requireNoError(t, err)
			if document.Format != format || document.Mismatch || document.Preview || !document.Download || document.IndexReason != "format_not_indexed" {
				t.Fatalf("incorrect metadata: %+v", document)
			}
			var jobs int
			requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE physical_file_id=? AND job_type='extract'", document.FileID).Scan(&jobs))
			if jobs != 0 {
				t.Fatal("non-PDF sent to PDF worker")
			}
			if file, _, err := service.OpenDocument(ctx, f.author, document.ID, false, domain.RequestMetadata{}); err == nil {
				file.Close()
				t.Fatal("active inline content allowed")
			}
			file, _, err := service.OpenDocument(ctx, f.author, document.ID, true, domain.RequestMetadata{})
			requireNoError(t, err)
			downloaded, err := io.ReadAll(file)
			file.Close()
			requireNoError(t, err)
			if !bytes.Equal(downloaded, contents) {
				t.Fatal("download changed bytes")
			}
			requireNoError(t, service.Classify(ctx, f.author, document.ID, "classification", Classification{CaseID: f.document.CaseID, CategoryID: f.document.CategoryID, TypeID: f.document.TypeID}, document.Revision, domain.RequestMetadata{}))
			document, err = service.Document(ctx, f.author, document.ID)
			requireNoError(t, err)
			operation, err := service.FinalizeDocument(ctx, f.author, document.ID, document.Revision, domain.RequestMetadata{})
			requireNoError(t, err)
			drain(t, service)
			result, err := service.Materialization(ctx, f.author, operation)
			requireNoError(t, err)
			if result.State != "cleaned" {
				t.Fatal(result)
			}
			published, err := service.Document(ctx, f.author, document.ID)
			requireNoError(t, err)
			if !strings.HasSuffix(published.OriginalPath, "."+format) || published.Hash != document.Hash || published.CaseID != f.document.CaseID || published.Approval != "approved" {
				t.Fatalf("publication lost original format/identity: %+v", published)
			}
			stored, err := os.ReadFile(published.OriginalPath)
			requireNoError(t, err)
			if !bytes.Equal(stored, contents) {
				t.Fatal("publication rewrote original")
			}
			requireNoError(t, service.VerifyManagedRoot(ctx, f.root))
		})
	}
	report, err := service.Processing(ctx, f.author, f.library)
	requireNoError(t, err)
	if report.Stats.StoredOnly != 4 || report.Stats.Pending != 0 {
		t.Fatalf("stored-only files masquerade as pending: %+v", report.Stats)
	}
}

func TestMultiformatLinkedPolicySkipsAndCache(t *testing.T) {
	service, principal, directory := fixture(t)
	ctx := context.Background()
	library := addLibrary(t, service, principal, "Formatos vinculados")
	path := filepath.Join(directory, "originals")
	for _, format := range []string{"docx", "xlsx", "txt", "csv"} {
		copyFixture(t, filepath.Join(path, "Original."+format), "sample."+format)
	}
	for name, data := range map[string]string{"script.txt": "#!/bin/sh\nexit 0", "program.exe": "MZ test", "data.blob": "Texto desconocido"} {
		target := filepath.Join(path, name)
		requireNoError(t, os.WriteFile(target, []byte(data), 0600))
		past := time.Now().Add(-5 * time.Second)
		requireNoError(t, os.Chtimes(target, past, past))
	}
	root := addRoot(t, service, principal, library, path)
	scan(t, service, root)
	report, err := service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if report.Stats.Total != 4 || report.Stats.StoredOnly != 4 || len(report.Skips) != 3 || len(report.Errors) != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	var jobs int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE library_id=? AND job_type='extract'", library).Scan(&jobs))
	if jobs != 0 {
		t.Fatal("created unsupported extraction jobs")
	}
	scan(t, service, root)
	report, err = service.Processing(ctx, principal, library)
	requireNoError(t, err)
	if report.Scans[0].Unchanged != 4 {
		t.Fatal("multiformat cache lost", report.Scans)
	}
	configuration, err := service.FileConfiguration(ctx, principal, library)
	requireNoError(t, err)
	store := []string{"pdf"}
	override := documentformat.Overrides{Store: &store}
	requireNoError(t, service.ConfigureFiles(ctx, principal, library, documentformat.Policy{}, override, configuration.Revision, configuration.GlobalRevision, domain.RequestMetadata{}))
	scan(t, service, root)
	var missing int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM physical_files WHERE library_id=? AND availability='missing'", library).Scan(&missing))
	if missing != 0 {
		t.Fatal("policy exclusion falsely reported deletion")
	}
	for _, format := range []string{"docx", "xlsx", "txt", "csv"} {
		contents, err := os.ReadFile(filepath.Join(path, "Original."+format))
		requireNoError(t, err)
		original, err := os.ReadFile("../../testdata/documents/sample." + format)
		requireNoError(t, err)
		if !bytes.Equal(contents, original) {
			t.Fatal("linked original changed")
		}
	}
	if _, err = os.Stat(filepath.Join(path, "program.exe")); err != nil {
		t.Fatal("prohibited original removed")
	}
}

func TestFileRulesAuthorizationVersioningAndWorkerGate(t *testing.T) {
	service, admin, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, admin, "hybrid")
	reader := member(t, service, admin, library, "format-reader", "library_reader")
	global, err := service.FileConfiguration(ctx, admin, "")
	requireNoError(t, err)
	if _, err = service.FileConfiguration(ctx, reader, ""); err == nil {
		t.Fatal("global rules exposed to nonadministrator")
	}
	if err = service.ConfigureFiles(ctx, reader, library, global.Effective, documentformat.Overrides{}, 0, 0, domain.RequestMetadata{}); err == nil {
		t.Fatal("reader changed file rules")
	}
	if err = service.ConfigureFiles(ctx, reader, "", global.Effective, documentformat.Overrides{}, 0, 0, domain.RequestMetadata{}); err == nil {
		t.Fatal("reader changed global rules")
	}
	bad := global.Effective
	bad.Store = []string{"pdf"}
	bad.Index = []string{"txt"}
	if err = service.ConfigureFiles(ctx, admin, "", bad, documentformat.Overrides{}, 0, 0, domain.RequestMetadata{}); err == nil {
		t.Fatal("index without storage admitted")
	}
	item := uploadFixture(t, service, admin, library)
	job, err := service.claim(ctx)
	requireNoError(t, err)
	none := []string{}
	override := documentformat.Overrides{Index: &none}
	requireNoError(t, service.ConfigureFiles(ctx, admin, library, documentformat.Policy{}, override, 0, 0, domain.RequestMetadata{}))
	if err = service.Extract(ctx, job, service.Identity.Config.Indexing); failureCode(err) != "FILE_INDEX_DISABLED" {
		t.Fatalf("worker ignored changed policy: %v", err)
	}
	requireNoError(t, service.finish(ctx, job, err))
	var state string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT status FROM jobs WHERE id=?", job.ID).Scan(&state))
	if state != "cancelled" {
		t.Fatal("policy denial retried", state)
	}
	document, err := service.Document(ctx, admin, item.DocumentID)
	requireNoError(t, err)
	if document.IndexReason != "format_not_indexed" {
		t.Fatal("missing policy reason")
	}
	if err = service.ConfigureFiles(ctx, admin, library, documentformat.Policy{}, override, 0, 0, domain.RequestMetadata{}); failureCode(err) != "REVISION_CONFLICT" {
		t.Fatal("stale revision accepted", err)
	}
	var count int
	requireNoError(t, service.Database.Reader.QueryRow("SELECT count(*) FROM audit_events WHERE event_type='configuration.files_changed'").Scan(&count))
	if count != 1 {
		t.Fatal("missing rule audit", count)
	}
}

func TestRejectedUploadRecordsDetectionAndCleansTemporary(t *testing.T) {
	service, principal, _ := fixture(t)
	ctx := context.Background()
	library := managedLibrary(t, service, principal, "managed")
	batch, err := service.CreateBatch(ctx, principal, library, domain.NewID(), domain.RequestMetadata{})
	requireNoError(t, err)
	_, err = service.Upload(ctx, principal, batch, domain.NewID(), "falso.pdf", strings.NewReader("Texto normal, no es un PDF."), nil, domain.RequestMetadata{})
	if failureCode(err) != "INVALID_PDF" {
		t.Fatal(err)
	}
	var status, code, raw, locator string
	requireNoError(t, service.Database.Reader.QueryRow("SELECT status,error_code,detection_json,private_temporary_locator FROM upload_items WHERE batch_id=?", batch).Scan(&status, &code, &raw, &locator))
	if status != "failed" || code != "INVALID_PDF" || !strings.Contains(raw, `"extension_mismatch":true`) || !strings.Contains(raw, `"format":"txt"`) {
		t.Fatalf("rejection not recorded: %s %s %s", status, code, raw)
	}
	if _, err = os.Stat(filepath.Join(service.Identity.Config.PrivateUploadDirectory(), locator)); !os.IsNotExist(err) {
		t.Fatal("rejected content retained", err)
	}
	if _, err = os.Stat(filepath.Join(service.Identity.Config.PrivateUploadDirectory(), locator+".part")); !os.IsNotExist(err) {
		t.Fatal("partial content retained", err)
	}
}
