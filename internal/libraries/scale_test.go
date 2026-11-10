//go:build development

package libraries

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

// Explicitly opt in: this creates only temporary SQLite rows, never PDFs or
// scheduled production work. It is intentionally excluded from normal tests.
func TestSyntheticScale(t *testing.T) {
	if os.Getenv("AIDEX_PERF_SCALE") != "1" {
		t.Skip("set AIDEX_PERF_SCALE=1")
	}
	s, p, dir := fixture(t)
	ctx := context.Background()
	libs := make([]string, 3)
	rootPaths := make([]string, 3)
	for i := range libs {
		id, err := s.Create(ctx, p, fmt.Sprintf("Escala %d", i), "hybrid", p.User.ID, domain.NewID(), domain.RequestMetadata{})
		if err != nil {
			t.Fatal(err)
		}
		libs[i] = id
		path := filepath.Join(dir, fmt.Sprintf("originales-%d", i))
		rootPaths[i] = path
		if err = os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		root := addRoot(t, s, p, id, path)
		scan(t, s, root)
	}
	originalMarker := filepath.Join(rootPaths[0], "original-intacto.txt")
	if err := os.WriteFile(originalMarker, []byte("AIDEX nunca elimina originales vinculados"), 0600); err != nil {
		t.Fatal(err)
	}
	caseIDs := make([][]string, 3)
	category := make([]string, 3)
	kind := make([]string, 3)
	if err := s.Database.Write(ctx, func(tx *sql.Tx) error {
		for index, library := range libs {
			category[index] = fmt.Sprintf("cat-%d", index)
			kind[index] = fmt.Sprintf("type-%d", index)
			template := fmt.Sprintf("template-%d", index)
			version := template + "-v1"
			for _, stmt := range []struct {
				sql  string
				args []any
			}{
				{"INSERT INTO categories(id,library_id,name) VALUES(?,?,?)", []any{category[index], library, "Planos"}},
				{"INSERT INTO document_types(id,library_id,category_id,name,allows_multiple,requires_descriptive_title) VALUES(?,?,?,?,1,0)", []any{kind[index], library, category[index], "Documento"}},
				{"INSERT INTO templates(id,library_id,name) VALUES(?,?,?)", []any{template, library, "Expediente base"}},
				{"INSERT INTO template_versions(id,template_id,library_id,revision,created_at) VALUES(?,?,?,?,?)", []any{version, template, library, 1, now()}},
				{"INSERT INTO template_requirements(id,template_version_id,library_id,category_id,document_type_id,required_count,mandatory,allows_multiple) VALUES(?,?,?,?,?,1,1,0)", []any{version + "-req", version, library, category[index], kind[index]}},
			} {
				if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
					return err
				}
			}
			for n := 0; n < 100; n++ {
				id := fmt.Sprintf("case-%d-%03d", index, n)
				caseIDs[index] = append(caseIDs[index], id)
				if _, err := tx.ExecContext(ctx, "INSERT INTO cases(id,library_id,identifier,identifier_key,exercise,template_version_id,created_at) VALUES(?,?,?,?,?,?,?)", id, library, id, id, fmt.Sprintf("%d", 2024+n%3), version, now()); err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, "INSERT INTO case_requirements(id,case_id,library_id,category_id,document_type_id,required_count,mandatory,allows_multiple) VALUES(?,?,?,?,?,1,1,0)", id+"-req", id, library, category[index], kind[index]); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	inserted := 0
	for _, target := range []int{10000, 50000, 100000} {
		started := time.Now()
		err := s.Database.Write(ctx, func(tx *sql.Tx) error {
			file, err := tx.PrepareContext(ctx, "INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,extraction_freshness,created_at) VALUES(?,?,'linked','available','unknown','none',?)")
			if err != nil {
				return err
			}
			defer file.Close()
			doc, err := tx.PrepareContext(ctx, "INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,approval_status,created_at,case_id,category_id,document_type_id) VALUES(?,?,?,?,?,?,?,?,?,?)")
			if err != nil {
				return err
			}
			defer doc.Close()
			version, err := tx.PrepareContext(ctx, "INSERT INTO content_versions(id,physical_file_id,generation,size_bytes,os_modified_at,sha256,observed_change_key,observed_at,change_origin) VALUES(?,?,1,?,?,?,? ,?,'scan')")
			if err != nil {
				return err
			}
			defer version.Close()
			for i := inserted; i < target; i++ {
				libraryIndex := 0
				if i%10 == 0 {
					libraryIndex = 1
				} else if i%10 == 1 {
					libraryIndex = 2
				}
				library := libs[libraryIndex]
				fid := fmt.Sprintf("file-%06d", i)
				did := fmt.Sprintf("doc-%06d", i)
				name := fmt.Sprintf("Expediente-%d-plano-%06d.pdf", i%100, i)
				date := domain.Timestamp(time.Now().AddDate(0, 0, -i%32))
				if _, err = file.ExecContext(ctx, fid, library, date); err != nil {
					return err
				}
				status := "approved"
				if i%7 == 0 {
					status = "draft"
				}
				if _, err = doc.ExecContext(ctx, did, library, fid, name, name, status, date, caseIDs[libraryIndex][i%100], category[libraryIndex], kind[libraryIndex]); err != nil {
					return err
				}
				if i%4 == 0 {
					vid := fmt.Sprintf("version-%06d", i)
					if _, err = version.ExecContext(ctx, vid, fid, 1024, date, fmt.Sprintf("%064x", i%2000), vid, date); err != nil {
						return err
					}
					if _, err = tx.ExecContext(ctx, "UPDATE physical_files SET current_content_version_id=? WHERE id=?", vid, fid); err != nil {
						return err
					}
				}
				if i%50 == 0 {
					jid := fmt.Sprintf("job-%06d", i)
					if _, err = tx.ExecContext(ctx, "INSERT INTO jobs(id,library_id,physical_file_id,job_type,target_version,idempotency_key,payload_json,status,available_at,created_at) VALUES(?,?,?,'extract',?,?,'{}','succeeded',?,?)", jid, library, fid, fmt.Sprintf("version-%06d", i), jid, date, date); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		inserted = target
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		info, _ := os.Stat(filepath.Join(s.Identity.Config.StateDirectory, "documental.db"))
		t.Logf("scale=%d insert=%s heap_mb=%.1f db_mb=%.1f", target, time.Since(started), float64(memory.HeapAlloc)/(1<<20), float64(info.Size())/(1<<20))
		measure := func(label string, fn func() error) {
			begin := time.Now()
			if err := fn(); err != nil {
				t.Fatalf("%s: %v", label, err)
			}
			t.Logf("scale=%d %s=%s", target, label, time.Since(begin))
		}
		measure("dashboard", func() error {
			v, e := s.Dashboard(ctx, p)
			if e == nil && v.Total != target {
				e = fmt.Errorf("total=%d", v.Total)
			}
			return e
		})
		measure("libraries", func() error { _, e := s.DashboardLibraries(ctx, p, ""); return e })
		measure("case_statistics", func() error { _, e := s.CaseStatistics(ctx, p, libs[0], ""); return e })
		measure("list", func() error { _, e := s.ListLibraries(ctx, p, false); return e })
		measure("name_search", func() error {
			_, e := s.Search(ctx, p, SearchInput{Query: "Expediente", Type: "name", Libraries: libs, Limit: 25}, domain.RequestMetadata{}, false)
			return e
		})
	}
	start := time.Now()
	if _,err:=s.Database.Writer.Exec("UPDATE jobs SET status='succeeded' WHERE library_id=? AND status IN ('queued','retry_wait','paused')",libs[0]);err!=nil{t.Fatal(err)}
	var activeScans, activeJobs, pendingMaterials int
	_ = s.Database.Reader.QueryRow("SELECT count(*) FROM root_scans WHERE root_id IN (SELECT id FROM storage_roots WHERE library_id=?) AND status IN ('running','paused')", libs[0]).Scan(&activeScans)
	_ = s.Database.Reader.QueryRow("SELECT count(*) FROM jobs WHERE library_id=? AND status IN ('queued','running','retry_wait','paused')", libs[0]).Scan(&activeJobs)
	_ = s.Database.Reader.QueryRow("SELECT count(*) FROM materializations WHERE library_id=? AND state<>'cleaned'", libs[0]).Scan(&pendingMaterials)
	t.Logf("deletion blockers: scans=%d jobs=%d materializations=%d", activeScans, activeJobs, pendingMaterials)
	plan, err := s.PlanDeletion(ctx, p, "library", libs[0], domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("delete_plan=%s documents=%d", time.Since(start), plan.Counts["documents"])
	start = time.Now()
	_, err = s.ConfirmDeletion(ctx, p, plan.ID, plan.Name, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("delete_confirm=%s", time.Since(start))
	result := make(chan error, 1)
	probeResult := make(chan error, 1)
	start = time.Now()
	go func() { result <- s.processPendingDeletion(ctx) }()
	go func() {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if _, e := s.Dashboard(probeCtx, p); e != nil { probeResult <- fmt.Errorf("dashboard during deletion: %w", e); return }
		if _, e := s.DashboardLibraries(probeCtx, p, ""); e != nil { probeResult <- fmt.Errorf("library dashboard during deletion: %w", e); return }
		if _, e := s.CaseStatistics(probeCtx, p, libs[1], ""); e != nil { probeResult <- fmt.Errorf("statistics during deletion: %w", e); return }
		if _, e := s.JobHistory(probeCtx, p, libs[1], TableFilter{Page: 1, Size: 10, Sort: "date", Direction: "desc"}); e != nil { probeResult <- fmt.Errorf("history during deletion: %w", e); return }
		if _, e := s.DeletionStatus(probeCtx, p, plan.ID); e != nil { probeResult <- fmt.Errorf("deletion status during deletion: %w", e); return }
		probeResult <- nil
	}()
	maxLatency := time.Duration(0)
	samples := 0
	for {
		select {
		case err := <-result:
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("delete_execute=%s concurrent_reads=%d max_read=%s", time.Since(start), samples, maxLatency)
			goto done
		default:
		}
		begin := time.Now()
		_, err := s.ListLibraries(ctx, p, false)
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Identity.LoadPrincipal(ctx, s.Database.Reader, p.TokenDigest)
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.Identity.License.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		latency := time.Since(begin)
		if latency > maxLatency {
			maxLatency = latency
		}
		samples++
		time.Sleep(100 * time.Millisecond)
	}
done:
	if err := <-probeResult; err != nil { t.Fatal(err) }
	if contents, err := os.ReadFile(originalMarker); err != nil || string(contents) != "AIDEX nunca elimina originales vinculados" {
		t.Fatalf("original linked file changed or deleted: %v", err)
	}
	for _, library := range libs[1:] {
		var id string
		if err = s.Database.Reader.QueryRowContext(ctx, "SELECT id FROM libraries WHERE id=?", library).Scan(&id); err != nil {
			t.Fatal(err)
		}
	}
}
