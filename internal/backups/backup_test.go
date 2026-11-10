//go:build development

package backups

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/licensefixture"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

type testInstallation struct {
	db       *storage.Database
	cfg      config.Config
	identity *identity.Service
	svc      *Service
	admin    string
	home     string
}

func installation(t *testing.T, name string) *testInstallation {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, name)
	db, err := storage.Open(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	cfg := config.Defaults(state)
	account, err := identity.New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = account.Bootstrap(context.Background(), "admin-"+name, "Administrator "+name, "abcdef"); err != nil {
		t.Fatal(err)
	}
	var admin string
	if err = db.Reader.QueryRow("SELECT first_admin_id FROM bootstrap_state").Scan(&admin); err != nil {
		t.Fatal(err)
	}
	svc, err := New(db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &testInstallation{db, cfg, account, svc, admin, home}
}
func syntheticDocument(t *testing.T, inst *testInstallation) (string, string) {
	t.Helper()
	directory := filepath.Join(inst.home, "documents")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(directory, "original.pdf")
	content := []byte("synthetic managed document")
	if err := os.WriteFile(original, content, 0600); err != nil {
		t.Fatal(err)
	}
	canonical, dirID, sensitive, err := libraries.InspectRestoreDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	fileID := domain.NewID()
	rootID := domain.NewID()
	locationID := domain.NewID()
	versionID := domain.NewID()
	docID := domain.NewID()
	libID := domain.NewID()
	digest := sha256.Sum256(content)
	checksum := hex.EncodeToString(digest[:])
	now := domain.Timestamp(time.Now())
	err = inst.db.Write(context.Background(), func(tx *sql.Tx) error {
		statements := []struct {
			query string
			args  []any
		}{
			{"INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES(?,?,'managed',?,?)", []any{libID, "Synthetic", now, now}},
			{"INSERT INTO storage_roots(id,library_id,storage_source,canonical_path,comparison_key,case_sensitive,status,watch_mode,reconcile_interval_seconds,created_at,directory_identity) VALUES(?,?,'managed',?,?,?,'active','polling',900,?,?)", []any{rootID, libID, canonical, canonical, sensitive, now, dirID}},
			{"INSERT INTO storage_root_configuration_versions(root_id,revision,canonical_path,comparison_key,configuration_json,created_at,created_by) VALUES(?,1,?,?,'{}',?,?)", []any{rootID, canonical, canonical, now, inst.admin}},
			{"INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,primary_location_id,current_content_version_id,created_at) VALUES(?,?,'managed','available','verified',?,?,?)", []any{fileID, libID, locationID, versionID, now}},
			{"INSERT INTO physical_file_locations(id,physical_file_id,library_id,storage_source,root_id,relative_path,canonical_path,comparison_key,root_configuration_revision,first_seen_at,last_seen_at) VALUES(?,?,?,'managed',?,'original.pdf',?,?,1,?,?)", []any{locationID, fileID, libID, rootID, original, original, now, now}},
			{"INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,sha256,observed_change_key,observed_at,change_origin) VALUES(?,?,1,?,?,?,?,?,'upload')", []any{versionID, fileID, len(content), now, checksum, domain.NewID(), now}},
			{"INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES(?,?,?,'original.pdf','original.pdf',?)", []any{docID, libID, fileID, now}},
		}
		for _, st := range statements {
			if _, err := tx.Exec(st.query, st.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return original, checksum
}
func syntheticPrivateUpload(t *testing.T, inst *testInstallation) {
	t.Helper()
	var library string
	if err := inst.db.Reader.QueryRow("SELECT id FROM libraries LIMIT 1").Scan(&library); err != nil {
		t.Fatal(err)
	}
	directory := inst.cfg.PrivateUploadDirectory()
	if err := storage.PrepareLocalPrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	item, batch, file, doc := domain.NewID(), domain.NewID(), domain.NewID(), domain.NewID()
	locator := item + ".document"
	payload := []byte("classified private original")
	if err := os.WriteFile(filepath.Join(directory, locator), payload, 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	now := domain.Timestamp(time.Now())
	err := inst.db.Write(context.Background(), func(tx *sql.Tx) error {
		stmts := []struct {
			q string
			a []any
		}{
			{"INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,created_at) VALUES(?,?,'managed','staged','verified',?)", []any{file, library, now}},
			{"INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,title,created_at) VALUES(?,?,?,'classified.docx','classified.docx','Classified',?)", []any{doc, library, file, now}},
			{"INSERT INTO upload_batches(id,library_id,created_by,client_batch_id,created_at) VALUES(?,?,?,?,?)", []any{batch, library, inst.admin, domain.NewID(), now}},
			{"INSERT INTO upload_items(id,batch_id,library_id,client_file_id,original_filename,document_id,private_temporary_locator,status,size_bytes,sha256,created_at) VALUES(?,?,?,?,'classified.docx',?,?,'retained',?,?,?)", []any{item, batch, library, domain.NewID(), doc, locator, len(payload), hex.EncodeToString(digest[:]), now}},
		}
		for _, st := range stmts {
			if _, e := tx.Exec(st.q, st.a...); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func syntheticPendingWork(t *testing.T, inst *testInstallation) {
	t.Helper()
	var library, root, file string
	if err := inst.db.Reader.QueryRow("SELECT library_id,id FROM storage_roots LIMIT 1").Scan(&library, &root); err != nil {
		t.Fatal(err)
	}
	if err := inst.db.Reader.QueryRow("SELECT id FROM physical_files WHERE primary_location_id IS NOT NULL LIMIT 1").Scan(&file); err != nil {
		t.Fatal(err)
	}
	now := domain.Timestamp(time.Now())
	if _, err := inst.db.Writer.Exec("INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract','1',?,'{}','queued',?,?)", domain.NewID(), library, file, domain.NewID(), now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := inst.db.Writer.Exec("INSERT INTO root_scans(id,root_id,configuration_revision,status,checkpoint_json,started_at) VALUES(?,?,1,'running','{}',?)", domain.NewID(), root, now); err != nil {
		t.Fatal(err)
	}
}

func offlinePerpetual(t *testing.T, inst *testInstallation) string {
	t.Helper()
	client := inst.identity.License
	status, err := client.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	client.Options.DevelopmentBypass = false
	client.Options.ServerURL = ""
	client.Options.TrustedKeys = []licensing.TrustKey{licensefixture.TrustKey()}
	claims := licensefixture.Claims(status.Binding, time.Now())
	jws := licensefixture.Sign(claims)
	if _, err = inst.db.Writer.Exec("UPDATE license_state SET current_jws=? WHERE singleton=1", jws); err != nil {
		t.Fatal(err)
	}
	if _, err = inst.db.Writer.Exec("INSERT INTO license_revision_floors(activation_id,license_id,highest_revision,jws_sha256) VALUES(?,?,?,?)", claims.ActivationID, claims.LicenseID, claims.Revision, domain.Digest(jws)); err != nil {
		t.Fatal(err)
	}
	status, err = client.Status(context.Background())
	if err != nil || status.State != "active" {
		t.Fatalf("offline perpetual license invalid: %+v %v", status, err)
	}
	return jws
}

func syntheticLifecycleEvidence(t *testing.T, inst *testInstallation, checksum string) {
	t.Helper()
	var library, root, document string
	if err := inst.db.Reader.QueryRow("SELECT library_id,id FROM storage_roots LIMIT 1").Scan(&library, &root); err != nil {
		t.Fatal(err)
	}
	if err := inst.db.Reader.QueryRow("SELECT id FROM documents LIMIT 1").Scan(&document); err != nil {
		t.Fatal(err)
	}
	now := domain.Timestamp(time.Now())
	operation := domain.NewID()
	if _, err := inst.db.Writer.Exec("UPDATE libraries SET disabled_at=? WHERE id=?", now, library); err != nil {
		t.Fatal(err)
	}
	statements := []struct {
		q    string
		args []any
	}{
		{"INSERT INTO root_retired_documents(root_id,document_id,removed_revision) VALUES(?,?,1)", []any{root, document}},
		{"INSERT INTO entity_deletions(id,kind,entity_id,library_id,actor_user_id,name,library_name,fingerprint,summary_json,state,created_at) VALUES(?,'root',?,?,?,?,?,'test','{}','cleanup_pending',?)", []any{operation, root, library, inst.admin, "Retired root", "Synthetic", now}},
		{"INSERT INTO deletion_documents(operation_id,document_id,filename,sha256,locations_json) VALUES(?,?,'original.pdf',?,'[]')", []any{operation, document, checksum}},
		{"INSERT INTO deletion_cleanup(id,operation_id,kind,locator,container_path,container_identity,file_identity,sha256,size_bytes,quarantine,status) VALUES(?,?,'upload','orphan.document',?,'source-container','source-file',?,1,'source-quarantine','pending')", []any{domain.NewID(), operation, inst.cfg.PrivateUploadDirectory(), checksum}},
	}
	for _, st := range statements {
		if _, err := inst.db.Writer.Exec(st.q, st.args...); err != nil {
			t.Fatal(err)
		}
	}
}
func waitComplete(t *testing.T, svc *Service, id string) Operation {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		op, err := svc.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if op.State == "complete" {
			return op
		}
		if op.State == "failed" {
			t.Fatal(op.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("backup timeout")
	return Operation{}
}

func TestBasicAndFullTransferNeverMoveLicenseOrOriginals(t *testing.T) {
	source := installation(t, "source")
	original, checksum := syntheticDocument(t, source)
	syntheticPrivateUpload(t, source)
	syntheticPendingWork(t, source)
	syntheticLifecycleEvidence(t, source, checksum)
	_, err := source.db.Writer.Exec("UPDATE license_state SET current_jws='secret-source-activation' WHERE singleton=1")
	if err != nil {
		t.Fatal(err)
	}
	dest := installation(t, "target")
	targetJWS := offlinePerpetual(t, dest)
	var sourceLicense, targetLicense string
	if err := source.db.Reader.QueryRow("SELECT installation_id FROM license_installation").Scan(&sourceLicense); err != nil {
		t.Fatal(err)
	}
	if err := dest.db.Reader.QueryRow("SELECT installation_id FROM license_installation").Scan(&targetLicense); err != nil {
		t.Fatal(err)
	}
	if sourceLicense == targetLicense {
		t.Fatal("test identities equal")
	}
	for _, kind := range []string{"basic", "full"} {
		t.Run(kind, func(t *testing.T) {
			started, err := source.svc.Start(kind, "password", source.admin)
			if err != nil {
				t.Fatal(err)
			}
			waitComplete(t, source.svc, started.ID)
			pathname, err := source.svc.Path(started.ID)
			if err != nil {
				t.Fatal(err)
			}
			packageBytes, err := os.ReadFile(pathname)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(packageBytes), sourceLicense) {
				t.Fatal("source installation ID leaked")
			}
			if _, err = openPackage(pathname, filepath.Join(source.home, "wrong.plain"), "wrong"); err == nil {
				t.Fatal("wrong password accepted")
			}
			file, err := os.Open(pathname)
			if err != nil {
				t.Fatal(err)
			}
			inspected, err := dest.svc.Inspect(context.Background(), file, "password", dest.admin)
			file.Close()
			if err != nil {
				t.Fatal(err)
			}
			if inspected.Manifest.Kind != kind {
				t.Fatal("wrong kind")
			}
			stage, err := dest.svc.stagedPackage(inspected.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stage.Files["tables/license_state.jsonl"] != nil || stage.Files["tables/sessions.jsonl"] != nil {
				t.Fatal("security state exported")
			}
			if kind == "basic" && stage.Manifest.PhysicalFiles != 0 {
				t.Fatal("basic has physical file")
			}
			if kind == "full" && stage.Manifest.PhysicalFiles != 2 {
				t.Fatal("full missing document")
			}
			stage.Close()
			plainPath := filepath.Join(dest.cfg.StateDirectory, "backups", inspected.ID+".plain")
			plainBytes, err := os.ReadFile(plainPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(plainBytes), "secret-source-activation") {
				t.Fatal("license secret leaked into logical archive")
			}
			base := filepath.Join(dest.home, "recovery")
			if err = os.Mkdir(base, 0700); err != nil && !os.IsExist(err) {
				t.Fatal(err)
			}
			sentinel := filepath.Join(base, "original.pdf")
			if err = os.WriteFile(sentinel, []byte("existing original"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = dest.svc.ScheduleRestore(context.Background(), inspected.ID, base, dest.admin, dest.identity.License); err != nil {
				t.Fatal(err)
			}
			if err = dest.svc.ApplyPending(context.Background(), dest.identity.License); err != nil {
				t.Fatal(err)
			}
			result, err := dest.svc.Get(inspected.ID)
			if err != nil {
				t.Fatal(err)
			}
			if result.State != "complete" {
				t.Fatalf("restore failed: %+v", result)
			}
			if err := dest.db.Reader.QueryRow("SELECT id FROM users WHERE username=?", result.RecoveryUser).Scan(&dest.admin); err != nil {
				t.Fatal(err)
			}
			if data, err := os.ReadFile(sentinel); err != nil || string(data) != "existing original" {
				t.Fatal("existing original changed", err)
			}
			if data, err := os.ReadFile(original); err != nil || string(data) != "synthetic managed document" {
				t.Fatal("source original changed", err)
			}
			var installationID string
			if err = dest.db.Reader.QueryRow("SELECT installation_id FROM license_installation").Scan(&installationID); err != nil || installationID != targetLicense {
				t.Fatal("destination license moved", err)
			}
			var keptJWS string
			if err = dest.db.Reader.QueryRow("SELECT current_jws FROM license_state WHERE singleton=1").Scan(&keptJWS); err != nil || keptJWS != targetJWS {
				t.Fatal("destination activation changed", err)
			}
			recovery, err := inspectZip(filepath.Join(dest.cfg.StateDirectory, "backups", inspected.ID+".pre-restore.aidexbak"))
			if err != nil {
				t.Fatal("private recovery point invalid", err)
			}
			if recovery.Files["tables/license_state.jsonl"] != nil || recovery.Manifest.Kind != "basic" {
				recovery.Close()
				t.Fatal("license state copied into recovery point")
			}
			for name, entry := range recovery.Files {
				stream, e := entry.Open()
				if e != nil {
					recovery.Close()
					t.Fatal(e)
				}
				payload, e := io.ReadAll(stream)
				stream.Close()
				if e != nil {
					recovery.Close()
					t.Fatal(e)
				}
				if strings.Contains(string(payload), targetJWS) {
					recovery.Close()
					t.Fatal("destination activation leaked into recovery component", name)
				}
			}
			recovery.Close()
			var availability string
			if err = dest.db.Reader.QueryRow("SELECT availability FROM physical_files WHERE primary_location_id IS NOT NULL LIMIT 1").Scan(&availability); err != nil {
				t.Fatal(err)
			}
			if kind == "basic" && availability != "available" {
				t.Fatal("basic did not revalidate existing original")
			}
			if kind == "full" && availability != "available" {
				t.Fatal("full unavailable")
			}
			if kind == "full" {
				var path string
				if err = dest.db.Reader.QueryRow("SELECT canonical_path FROM physical_file_locations WHERE retired_at IS NULL LIMIT 1").Scan(&path); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(data)
				if hex.EncodeToString(sum[:]) != checksum {
					t.Fatal("restored content mismatch")
				}
				var locator string
				if err = dest.db.Reader.QueryRow("SELECT private_temporary_locator FROM upload_items WHERE status='retained'").Scan(&locator); err != nil {
					t.Fatal(err)
				}
				payload, readErr := os.ReadFile(filepath.Join(dest.cfg.PrivateUploadDirectory(), locator))
				if readErr != nil || string(payload) != "classified private original" {
					t.Fatal("private classified file was not restored", readErr)
				}
			}
			var sessions int
			if err = dest.db.Reader.QueryRow("SELECT count(*) FROM sessions").Scan(&sessions); err != nil || sessions != 0 {
				t.Fatal("sessions restored", err)
			}
			var jobStatus, scanStatus string
			if err = dest.db.Reader.QueryRow("SELECT status FROM jobs LIMIT 1").Scan(&jobStatus); err != nil || jobStatus != "cancelled" {
				t.Fatal("source job resumed", err, jobStatus)
			}
			if err = dest.db.Reader.QueryRow("SELECT status FROM root_scans LIMIT 1").Scan(&scanStatus); err != nil || scanStatus != "failed" {
				t.Fatal("source scan resumed", err, scanStatus)
			}
			var archived, retiredCount int
			if err = dest.db.Reader.QueryRow("SELECT count(*) FROM libraries WHERE disabled_at IS NOT NULL").Scan(&archived); err != nil || archived != 1 {
				t.Fatal("archived library reactivated", err, archived)
			}
			if err = dest.db.Reader.QueryRow("SELECT count(*) FROM root_retired_documents").Scan(&retiredCount); err != nil || retiredCount != 1 {
				t.Fatal("retirement evidence missing", err, retiredCount)
			}
			var cleanupStatus, cleanupCode, cleanupPath, cleanupIdentity string
			if err = dest.db.Reader.QueryRow("SELECT status,error_code,container_path,file_identity FROM deletion_cleanup LIMIT 1").Scan(&cleanupStatus, &cleanupCode, &cleanupPath, &cleanupIdentity); err != nil || cleanupStatus != "attention" || cleanupCode != "CLEANUP_RESTORE_REVIEW" || cleanupPath != "" || cleanupIdentity != "" {
				t.Fatal("inherited cleanup remained executable", err, cleanupStatus, cleanupCode)
			}
		})
	}
}

func TestBadPackageAndInactiveLicenseAreRejected(t *testing.T) {
	source := installation(t, "source")
	syntheticDocument(t, source)
	dest := installation(t, "target")
	started, err := source.svc.Start("basic", "", source.admin)
	if err != nil {
		t.Fatal(err)
	}
	waitComplete(t, source.svc, started.ID)
	pathname, err := source.svc.Path(started.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(pathname)
	if err != nil {
		t.Fatal(err)
	}
	corrupt := append([]byte{}, raw...)
	if len(corrupt) < 40 || string(corrupt[:8]) != magic || string(corrupt[9:13]) != "PK\x03\x04" {
		t.Fatal("unexpected synthetic archive")
	}
	firstBody := 9 + 30 + int(binary.LittleEndian.Uint16(corrupt[9+26:9+28])) + int(binary.LittleEndian.Uint16(corrupt[9+28:9+30]))
	if firstBody >= len(corrupt) {
		t.Fatal("invalid synthetic ZIP")
	}
	corrupt[firstBody] ^= 0x7f
	for _, payload := range [][]byte{raw[:len(raw)/2], append(append([]byte{}, raw...), 1), corrupt} {
		if _, err = dest.svc.Inspect(context.Background(), strings.NewReader(string(payload)), "", dest.admin); err == nil {
			t.Fatal("damaged package accepted")
		}
	}
	file, err := os.Open(pathname)
	if err != nil {
		t.Fatal(err)
	}
	op, err := dest.svc.Inspect(context.Background(), file, "", dest.admin)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	dest.identity.License.Options.DevelopmentBypass = false
	if _, err = dest.svc.ScheduleRestore(context.Background(), op.ID, "", dest.admin, dest.identity.License); err == nil {
		t.Fatal("inactive license accepted")
	}
	// A standalone ZIP cannot silently bypass the AIDEX wrapper.
	bad := filepath.Join(dest.home, "bad.zip")
	out, err := os.Create(bad)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(out)
	entry, _ := writer.Create("../escape")
	_, _ = io.WriteString(entry, "x")
	writer.Close()
	out.Close()
	file, err = os.Open(bad)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dest.svc.Inspect(context.Background(), file, "", dest.admin)
	file.Close()
	if err == nil {
		t.Fatal("malicious ZIP accepted")
	}
	versionPath := filepath.Join(dest.home, "unsupported.aidexbak")
	out, err = os.Create(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = out.Write(append([]byte(magic), 0)); err != nil {
		t.Fatal(err)
	}
	writer = zip.NewWriter(out)
	header := &zip.FileHeader{Name: "manifest.json", Method: zip.Deflate}
	header.SetMode(0600)
	entry, err = writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.WriteString(entry, `{"product":"AIDEX","format":1,"schema":999,"kind":"basic","entries":[]}`); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dest.svc.Inspect(context.Background(), input, "", dest.admin)
	input.Close()
	if err == nil {
		t.Fatal("incompatible schema accepted")
	}
}

func TestBasicRestoreMapsAndVerifiesPhysicalReferences(t *testing.T) {
	source := installation(t, "source")
	original, _ := syntheticDocument(t, source)
	var rootID string
	if err := source.db.Reader.QueryRow("SELECT id FROM storage_roots LIMIT 1").Scan(&rootID); err != nil {
		t.Fatal(err)
	}
	started, err := source.svc.Start("basic", "", source.admin)
	if err != nil {
		t.Fatal(err)
	}
	waitComplete(t, source.svc, started.ID)
	archivePath, err := source.svc.Path(started.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, content, want string }{
		{"matching", "synthetic managed document", "available"},
		{"changed", "different content", "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := installation(t, "target")
			mapped := filepath.Join(dest.home, "mapped")
			if err := os.Mkdir(mapped, 0700); err != nil {
				t.Fatal(err)
			}
			mappedFile := filepath.Join(mapped, "original.pdf")
			if err := os.WriteFile(mappedFile, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			input, err := os.Open(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			op, err := dest.svc.Inspect(context.Background(), input, "", dest.admin)
			input.Close()
			if err != nil {
				t.Fatal(err)
			}
			if _, err = dest.svc.ScheduleRestoreWithMappings(context.Background(), op.ID, "", map[string]string{rootID: mapped}, dest.admin, dest.identity.License); err != nil {
				t.Fatal(err)
			}
			if err = dest.svc.ApplyPending(context.Background(), dest.identity.License); err != nil {
				t.Fatal(err)
			}
			result, err := dest.svc.Get(op.ID)
			if err != nil || result.State != "complete" {
				t.Fatalf("restore failed: %+v %v", result, err)
			}
			var availability, path string
			if err = dest.db.Reader.QueryRow("SELECT f.availability,l.canonical_path FROM physical_files f JOIN physical_file_locations l ON l.id=f.primary_location_id").Scan(&availability, &path); err != nil {
				t.Fatal(err)
			}
			if availability != tc.want || path != mappedFile {
				t.Fatalf("mapping result: %s %s", availability, path)
			}
			var rootPath, rootIdentity, rootStatus string
			if err = dest.db.Reader.QueryRow("SELECT canonical_path,directory_identity,status FROM storage_roots WHERE id=?", rootID).Scan(&rootPath, &rootIdentity, &rootStatus); err != nil {
				t.Fatal(err)
			}
			if rootPath != mapped || rootIdentity == "" || rootStatus != "active" {
				t.Fatalf("mapped root unusable: %s %s", rootPath, rootStatus)
			}
			actual, err := os.ReadFile(mappedFile)
			if err != nil || string(actual) != tc.content {
				t.Fatal("physical original changed", err)
			}
			if actual, err = os.ReadFile(original); err != nil || string(actual) != "synthetic managed document" {
				t.Fatal("source original changed", err)
			}
		})
	}
}

func TestRestoreCollisionLeavesOriginalAndLogicalStateUntouchedUntilRetry(t *testing.T) {
	source := installation(t, "source")
	syntheticDocument(t, source)
	dest := installation(t, "target")
	started, err := source.svc.Start("full", "", source.admin)
	if err != nil {
		t.Fatal(err)
	}
	waitComplete(t, source.svc, started.ID)
	archivePath, err := source.svc.Path(started.ID)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	op, err := dest.svc.Inspect(context.Background(), input, "", dest.admin)
	input.Close()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dest.home, "recovery")
	if err = os.Mkdir(base, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = dest.svc.ScheduleRestore(context.Background(), op.ID, base, dest.admin, dest.identity.License); err != nil {
		t.Fatal(err)
	}
	var rootID, fileID string
	if err = source.db.Reader.QueryRow("SELECT id FROM storage_roots LIMIT 1").Scan(&rootID); err != nil {
		t.Fatal(err)
	}
	if err = source.db.Reader.QueryRow("SELECT id FROM physical_files LIMIT 1").Scan(&fileID); err != nil {
		t.Fatal(err)
	}
	rootDir := filepath.Join(base, "aidex-restore-"+op.ID, rootID)
	if err = os.MkdirAll(rootDir, 0700); err != nil {
		t.Fatal(err)
	}
	collision := filepath.Join(rootDir, fileID+".pdf")
	if err = os.WriteFile(collision, []byte("pre-existing original"), 0600); err != nil {
		t.Fatal(err)
	}
	var oldUsers int
	if err = dest.db.Reader.QueryRow("SELECT count(*) FROM users").Scan(&oldUsers); err != nil {
		t.Fatal(err)
	}
	if err = dest.svc.ApplyPending(context.Background(), dest.identity.License); err != nil {
		t.Fatal(err)
	}
	result, err := dest.svc.Get(op.ID)
	if err != nil || result.State != "attention" {
		t.Fatalf("collision not reported: %+v %v", result, err)
	}
	actual, err := os.ReadFile(collision)
	if err != nil || string(actual) != "pre-existing original" {
		t.Fatal("collision overwritten", err)
	}
	var users int
	if err = dest.db.Reader.QueryRow("SELECT count(*) FROM users").Scan(&users); err != nil || users != oldUsers {
		t.Fatal("logical state changed before safe publication", err)
	}
	if err = os.Remove(collision); err != nil {
		t.Fatal(err)
	}
	if _, err = dest.svc.ScheduleRestore(context.Background(), op.ID, base, dest.admin, dest.identity.License); err != nil {
		t.Fatal(err)
	}
	if err = dest.svc.ApplyPending(context.Background(), dest.identity.License); err != nil {
		t.Fatal(err)
	}
	result, err = dest.svc.Get(op.ID)
	if err != nil || result.State != "complete" {
		t.Fatalf("retry failed: %+v %v", result, err)
	}
}
