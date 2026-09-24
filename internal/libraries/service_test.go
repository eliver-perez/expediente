//go:build development

package libraries

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/storage"
)

func fixture(t *testing.T) (*Service, domain.Principal, string) {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(directory, "state")
	database, err := storage.Open(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	account, err := identity.New(database, config.Defaults(state))
	if err != nil {
		t.Fatal(err)
	}
	if err = account.Bootstrap(context.Background(), "admin-h3", "Prueba H3", "abcdef"); err != nil {
		t.Fatal(err)
	}
	login, err := account.Login(context.Background(), "admin-h3", "abcdef", domain.RequestMetadata{RequestID: domain.NewID(), ObservedIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := account.Authenticate(context.Background(), login.SessionToken, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	return New(account), principal, directory
}
func addLibrary(t *testing.T, service *Service, principal domain.Principal, name string) string {
	t.Helper()
	identifier, err := service.Create(context.Background(), principal, name, "linked", principal.User.ID, domain.NewID(), domain.RequestMetadata{RequestID: domain.NewID()})
	if err != nil {
		t.Fatal(err)
	}
	return identifier
}
func addRoot(t *testing.T, service *Service, principal domain.Principal, library, path string) string {
	t.Helper()
	plan, err := service.PlanRoot(context.Background(), principal, library, path, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	identifier, err := service.ConfirmRoot(context.Background(), principal, library, plan.ID, plan.Revision, plan.Relation == "ancestor", domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	return identifier
}
func copyFixture(t *testing.T, path, name string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("../../testdata/documents", name))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, contents, 0600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-5 * time.Second)
	if err = os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
}
func scan(t *testing.T, service *Service, root string) {
	t.Helper()
	if err := service.Scan(context.Background(), root, 256<<20, nil); err != nil {
		t.Fatal(err)
	}
}
func drain(t *testing.T, service *Service) {
	t.Helper()
	for _, program := range []string{"pdfinfo", "pdftotext", "pdftoppm", "tesseract"} {
		if _, err := exec.LookPath(program); err != nil {
			t.Skip("PDF/OCR tools unavailable on this runner")
		}
	}
	for attempt := 0; attempt < 20; attempt++ {
		job, err := service.claim(context.Background())
		if err == sql.ErrNoRows {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if job.Kind == "scan" {
			err = service.Scan(context.Background(), job.Version, 256<<20, nil)
		} else {
			err = service.Extract(context.Background(), job, service.Identity.Config.Indexing)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = service.finish(context.Background(), job, nil); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("queue did not drain")
}
func query(t *testing.T, service *Service, principal domain.Principal, library, text string) SearchResult {
	t.Helper()
	result, err := service.Search(context.Background(), principal, SearchInput{Query: text, Libraries: []string{library}}, domain.RequestMetadata{RequestID: domain.NewID()}, true)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestLinkedRetentionRenameConsolidationAndChange(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Planos")
	parent := filepath.Join(directory, "linked")
	child := filepath.Join(parent, "child")
	source := filepath.Join(child, "Plano-PR-008.pdf")
	copyFixture(t, source, "native.pdf")
	root := addRoot(t, service, principal, library, child)
	drain(t, service)
	result := query(t, service, principal, library, "\"estructura metálica\"")
	if result.Count != 1 || len(result.Items[0].Matches) == 0 {
		t.Fatalf("missing phrase: %+v", result)
	}
	documentID := result.Items[0].ID
	renamed := filepath.Join(child, "Renombrado.pdf")
	if err := os.Rename(source, renamed); err != nil {
		t.Fatal(err)
	}
	scan(t, service, root)
	result = query(t, service, principal, library, "estructura")
	if result.Items[0].ID != documentID || result.Items[0].Filename != "Renombrado.pdf" {
		t.Fatal("rename changed identity")
	}
	view, err := service.AddView(context.Background(), principal, library, root, "sub", "Subcarpeta", domain.RequestMetadata{})
	if err != nil || view == "" {
		t.Fatal(err)
	}
	parentRoot := addRoot(t, service, principal, library, parent)
	drain(t, service)
	result = query(t, service, principal, library, "estructura")
	if result.Items[0].ID != documentID || result.Items[0].RootID != parentRoot {
		t.Fatal("consolidation changed identity")
	}
	if err = os.Remove(renamed); err != nil {
		t.Fatal(err)
	}
	scan(t, service, parentRoot)
	result = query(t, service, principal, library, "estructura")
	if result.Count != 1 || result.Items[0].Availability != "missing" || result.Items[0].Preview {
		t.Fatalf("missing index lost: %+v", result)
	}
	if _, _, err = service.OpenDocument(context.Background(), principal, documentID, false, domain.RequestMetadata{}); err == nil {
		t.Fatal("missing content exposed")
	}
}
func TestDisconnectedRootPreservesAvailabilityAndScope(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Disco")
	path := filepath.Join(directory, "disk")
	copyFixture(t, filepath.Join(path, "native.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	drain(t, service)
	if err := os.Rename(path, path+"-offline"); err != nil {
		t.Fatal(err)
	}
	if err := service.Scan(context.Background(), root, 256<<20, nil); err == nil {
		t.Fatal("missing mount accepted")
	}
	result := query(t, service, principal, library, "PR-008")
	if result.Count != 1 || result.Items[0].Availability != "unknown" {
		t.Fatalf("disconnect inferred deletion: %+v", result)
	}
	if err := os.Rename(path+"-offline", path); err != nil {
		t.Fatal(err)
	}
	scan(t, service, root)
	other := addLibrary(t, service, principal, "Otra")
	if _, err := service.PlanRoot(context.Background(), principal, other, path, domain.RequestMetadata{}); err == nil {
		t.Fatal("cross-library overlap accepted")
	}
	account, err := service.Identity.CreateUser(context.Background(), principal, "reader-h3", "Lector", "abcdef", domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	login, err := service.Identity.Login(context.Background(), account.Username, "abcdef", domain.RequestMetadata{ObservedIP: "127.0.0.2"})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := service.Identity.Authenticate(context.Background(), login.SessionToken, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Search(context.Background(), reader, SearchInput{Query: "PR-008", Libraries: []string{library}}, domain.RequestMetadata{}, true); err == nil {
		t.Fatal("unassigned reader saw library")
	}
	if err = service.SetMember(context.Background(), principal, library, reader.User.ID, []string{"library_reader"}, domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	result = query(t, service, reader, library, "PR-008")
	if result.Count != 1 || result.Items[0].OriginalPath != "" {
		t.Fatal("reader path disclosure or missing result")
	}
	var exact string
	if err = service.Database.Reader.QueryRow("SELECT exact_query_text FROM search_audit_details ORDER BY rowid DESC LIMIT 1").Scan(&exact); err != nil || exact != "PR-008" {
		t.Fatal("exact query audit missing", err)
	}
}
func TestOCRChangesDuplicatesAndFailedReplacement(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "OCR")
	path := filepath.Join(directory, "ocr")
	source := filepath.Join(path, "document.pdf")
	copyFixture(t, source, "native.pdf")
	copyFixture(t, filepath.Join(path, "copy.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	drain(t, service)
	duplicates, err := service.Duplicates(context.Background(), principal, library)
	if err != nil || len(duplicates) != 1 || duplicates[0].Count != 2 {
		t.Fatalf("duplicates: %+v %v", duplicates, err)
	}
	result := query(t, service, principal, library, "PR-008")
	if result.Count != 2 {
		t.Fatal(result)
	}
	copyFixture(t, source, "scanned.pdf")
	scan(t, service, root)
	options := service.Identity.Config.Indexing
	service.Identity.Config.Indexing.PDFInfo = "missing-pdfinfo-for-test"
	job, err := service.claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = service.Extract(context.Background(), job, service.Identity.Config.Indexing)
	if err == nil {
		t.Fatal("missing tool unexpectedly worked")
	}
	if err = service.finish(context.Background(), job, err); err != nil {
		t.Fatal(err)
	}
	result = query(t, service, principal, library, "PR-008")
	if result.Count != 2 {
		t.Fatal("failed extraction discarded old index")
	}
	service.Identity.Config.Indexing = options
	if _, err = service.Retry(context.Background(), principal, job.ID, "Herramienta restaurada", domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	drain(t, service)
	result = query(t, service, principal, library, "ESCANEADO")
	if result.Count != 1 {
		t.Fatalf("OCR missing: %+v", result)
	}
	copyFixture(t, source, "native.pdf")
	scan(t, service, root)
	drain(t, service)
	notices, err := service.Notifications(context.Background(), principal)
	if err != nil || len(notices) != 2 {
		t.Fatalf("A→B→A notices: %+v %v", notices, err)
	}
	scan(t, service, root)
	notices, err = service.Notifications(context.Background(), principal)
	if err != nil || len(notices) != 2 {
		t.Fatal("duplicate change notices")
	}
}
func TestRootPlansSymlinksAndSearchInput(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Rutas")
	path := filepath.Join(directory, "roots")
	copyFixture(t, filepath.Join(path, "sub", "a.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	scan(t, service, root)
	plan, err := service.PlanRoot(context.Background(), principal, library, filepath.Join(path, "sub"), domain.RequestMetadata{})
	if err != nil || plan.Relation != "descendant" {
		t.Fatal(plan, err)
	}
	if _, err = service.ConfirmRoot(context.Background(), principal, library, plan.ID, plan.Revision, false, domain.RequestMetadata{}); err == nil {
		t.Fatal("nested root created")
	}
	external := filepath.Join(directory, "secret.pdf")
	copyFixture(t, external, "native.pdf")
	if err = os.Symlink(external, filepath.Join(path, "escape.pdf")); err != nil {
		t.Skip("symlinks unavailable on this runner", err)
	}
	scan(t, service, root)
	registered, err := service.root(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = openLinked(registered, "escape.pdf"); err == nil {
		t.Fatal("symlink exposed")
	}
	if _, err = openLinked(registered, "../secret.pdf"); err == nil {
		t.Fatal("traversal exposed")
	}
	for _, text := range []string{"\"unclosed", strings.Repeat("a", 2001)} {
		if _, err = compileQueryOnly(text); err == nil {
			t.Fatal("malformed query accepted")
		}
	}
}
func compileQueryOnly(text string) (string, error) {
	compiled, _, err := compileQuery(text)
	return compiled, err
}

func TestHardLinksReappearanceAndRetirement(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Aliases")
	first := filepath.Join(directory, "first")
	second := filepath.Join(directory, "second")
	copyFixture(t, filepath.Join(first, "original.pdf"), "native.pdf")
	if err := os.Mkdir(second, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(first, "original.pdf"), filepath.Join(second, "alias.pdf")); err != nil {
		t.Fatal(err)
	}
	firstRoot := addRoot(t, service, principal, library, first)
	secondRoot := addRoot(t, service, principal, library, second)
	drain(t, service)
	result := query(t, service, principal, library, "PR-008")
	if result.Count != 1 {
		t.Fatal("hard links duplicated identity")
	}
	identifier := result.Items[0].ID
	if err := os.Remove(filepath.Join(first, "original.pdf")); err != nil {
		t.Fatal(err)
	}
	scan(t, service, firstRoot)
	result = query(t, service, principal, library, "PR-008")
	if result.Items[0].Availability != "available" {
		t.Fatal("remaining alias not selected")
	}
	alias := filepath.Join(second, "alias.pdf")
	if err := os.Rename(alias, alias+".hidden"); err != nil {
		t.Fatal(err)
	}
	scan(t, service, secondRoot)
	result = query(t, service, principal, library, "PR-008")
	if result.Items[0].Availability != "missing" {
		t.Fatal("all missing aliases not detected")
	}
	if err := os.Rename(alias+".hidden", alias); err != nil {
		t.Fatal(err)
	}
	scan(t, service, secondRoot)
	result = query(t, service, principal, library, "PR-008")
	if result.Items[0].ID != identifier || result.Items[0].Availability != "available" {
		t.Fatal("reappearance did not preserve identity")
	}
	plan, err := service.RetirementPlan(context.Background(), principal, secondRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Retire(context.Background(), principal, secondRoot, plan["plan_id"].(string), "retain_index", domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	result = query(t, service, principal, library, "PR-008")
	if result.Count != 1 {
		t.Fatal("retirement discarded retained text")
	}
	if _, err = os.Stat(alias); err != nil {
		t.Fatal("retirement touched original")
	}
}

func TestQueueRetryLimitRestartAndPublicationFence(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Cola")
	path := filepath.Join(directory, "queue")
	copyFixture(t, filepath.Join(path, "file.pdf"), "native.pdf")
	root := addRoot(t, service, principal, library, path)
	for attempt := 1; attempt <= 5; attempt++ {
		job, err := service.claim(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if job.Attempts != attempt {
			t.Fatal("unexpected attempt count")
		}
		if err = service.finish(context.Background(), job, scanFailure("ROOT_UNAVAILABLE")); err != nil {
			t.Fatal(err)
		}
		if _, err = service.Database.Writer.Exec("UPDATE jobs SET available_at=? WHERE id=?", now(), job.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.claim(context.Background()); err != sql.ErrNoRows {
		t.Fatal("sixth attempt allowed", err)
	}
	var failedID string
	if err := service.Database.Reader.QueryRow("SELECT id FROM jobs WHERE status='failed'").Scan(&failedID); err != nil {
		t.Fatal(err)
	}
	retry, err := service.Retry(context.Background(), principal, failedID, "Recurso restaurado", domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	same, err := service.Retry(context.Background(), principal, failedID, "Recurso restaurado", domain.RequestMetadata{})
	if err != nil || same != retry {
		t.Fatal("retry command duplicated job", err)
	}
	scan(t, service, root)
	job, err := service.claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if job.Kind == "scan" {
		if err = service.finish(context.Background(), job, nil); err != nil {
			t.Fatal(err)
		}
		job, err = service.claim(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}
	registered, err := service.root(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	stale := job
	stale.Fence++
	if err = service.publish(context.Background(), stale, registered, nil, "spa"); err == nil {
		t.Fatal("stale fencing token published")
	}
	runtime, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	runtime.Close()
	var status string
	if err = service.Database.Reader.QueryRow("SELECT status FROM jobs WHERE id=?", job.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status == "running" {
		t.Fatal("restart failed to recover lease")
	}
}

func TestSearchScopePaginationAndPrivateQueryAudit(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Consulta")
	path := filepath.Join(directory, "query")
	copyFixture(t, filepath.Join(path, "sub", "first.pdf"), "native.pdf")
	copyFixture(t, filepath.Join(path, "sub-other", "second.pdf"), "native.pdf")
	addRoot(t, service, principal, library, path)
	drain(t, service)
	input := SearchInput{Query: "  PR-008  ", Libraries: []string{library}, Limit: 1}
	first, err := service.Search(context.Background(), principal, input, domain.RequestMetadata{}, true)
	if err != nil || first.Count != 2 || first.Cursor == "" {
		t.Fatal(first, err)
	}
	input.Cursor = first.Cursor
	second, err := service.Search(context.Background(), principal, input, domain.RequestMetadata{}, true)
	if err != nil || second.Items[0].ID == first.Items[0].ID || second.Cursor != "" {
		t.Fatal("pagination failed", err)
	}
	input.Query = "different"
	if _, err = service.Search(context.Background(), principal, input, domain.RequestMetadata{}, true); err == nil {
		t.Fatal("cursor reused with different query")
	}
	scoped, err := service.Search(context.Background(), principal, SearchInput{Libraries: []string{library}, Filters: Filters{Prefix: "sub"}}, domain.RequestMetadata{}, false)
	if err != nil || scoped.Count != 1 {
		t.Fatal("component prefix matched sibling", err)
	}
	folders, _, err := service.Folders(context.Background(), principal, library, "", "", "", "")
	if err != nil || len(folders) != 2 {
		t.Fatal(folders, err)
	}
	events, _, err := service.SearchEvents(context.Background(), principal, "")
	if err != nil || len(events) != 0 {
		t.Fatal("manager saw sensitive query details", err)
	}
	if err = service.SetMember(context.Background(), principal, library, principal.User.ID, []string{"library_manager", "library_auditor"}, domain.RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	events, _, err = service.SearchEvents(context.Background(), principal, "")
	if err != nil || len(events) != 2 {
		t.Fatal(events, err)
	}
	for _, event := range events {
		if event.Query != "  PR-008  " {
			t.Fatal("query whitespace was altered")
		}
	}
}

func TestNativeWatcherNewDirectoriesAndOverflowRecovery(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Vigilancia")
	path := filepath.Join(directory, "watched")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	root := addRoot(t, service, principal, library, path)
	runtime, err := service.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if runtime.watcher == nil {
		t.Skip("native watcher unavailable")
	}
	await := func(description string, condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if condition() {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatal("timeout: " + description)
	}
	await("initial scan", func() bool {
		registered, err := service.root(context.Background(), root)
		return err == nil && registered.LastScan != ""
	})
	copyFixture(t, filepath.Join(path, "new", "nested", "created.pdf"), "native.pdf")
	await("recursive native event", func() bool {
		var count int
		err := service.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE library_id=?", library).Scan(&count)
		return err == nil && count == 1
	})
	runtime.eventsLost(context.Background())
	await("overflow switches to polling", func() bool {
		registered, err := service.root(context.Background(), root)
		return err == nil && registered.WatchMode == "polling" && registered.LastError == "WATCH_EVENTS_LOST"
	})
	await("barrier reconciliation", func() bool {
		var count int
		err := service.Database.Reader.QueryRow("SELECT count(*) FROM root_scans WHERE root_id=? AND status='complete'", root).Scan(&count)
		return err == nil && count >= 2
	})
}

func TestInvalidPDFDoesNotExposeContentsOrBlockOtherDiscoveries(t *testing.T) {
	service, principal, directory := fixture(t)
	library := addLibrary(t, service, principal, "Formatos")
	path := filepath.Join(directory, "formats")
	copyFixture(t, filepath.Join(path, "valid.pdf"), "native.pdf")
	invalidPath := filepath.Join(path, "invalid.pdf")
	if err := os.WriteFile(invalidPath, []byte("This is private text, not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-5 * time.Second)
	if err := os.Chtimes(invalidPath, past, past); err != nil {
		t.Fatal(err)
	}
	root := addRoot(t, service, principal, library, path)
	if err := service.Scan(context.Background(), root, 256<<20, nil); err == nil || failureCode(err) != "INVALID_PDF" {
		t.Fatal("invalid PDF accepted", err)
	}
	var count int
	if err := service.Database.Reader.QueryRow("SELECT count(*) FROM documents WHERE library_id=?", library).Scan(&count); err != nil || count != 1 {
		t.Fatal("valid document not discovered", err)
	}
	var confirmed int
	if err := service.Database.Reader.QueryRow("SELECT sum(can_confirm_absence) FROM root_scans WHERE root_id=?", root).Scan(&confirmed); err != nil || confirmed != 0 {
		t.Fatal("partial scan inferred deletion", err)
	}
}
