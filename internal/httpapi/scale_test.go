//go:build development

package httpapi

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

// Exercises authentication, routing, JSON serialization and the actual queries.
// Rows live only in the test's temporary database; no real originals are touched.
func TestDashboardHTTPTiming(t *testing.T) {
	if os.Getenv("AIDEX_PERF_SCALE") != "1" {
		t.Skip("set AIDEX_PERF_SCALE=1")
	}
	f := newHTTPFixture(t)
	cookie, _ := f.login(t, "admin", testPassword)
	principal, err := f.service.Authenticate(context.Background(), cookie.Value, false, domain.RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	var ids [2]string
	for i, mode := range []string{"linked", "managed"} {
		ids[i], err = f.server.libraries.Create(context.Background(), principal, fmt.Sprintf("Medición %d", i), mode, principal.User.ID, domain.NewID(), domain.RequestMetadata{})
		if err != nil {
			t.Fatal(err)
		}
	}
	stamp := domain.Timestamp(time.Now())
	err = f.service.Database.Write(context.Background(), func(tx *sql.Tx) error {
		file, e := tx.Prepare("INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,extraction_freshness,created_at) VALUES(?,?,?,'available','unknown','none',?)")
		if e != nil {
			return e
		}
		defer file.Close()
		doc, e := tx.Prepare("INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES(?,?,?,?,?,?)")
		if e != nil {
			return e
		}
		defer doc.Close()
		for i := 0; i < 10000; i++ {
			index := i % 2
			id := fmt.Sprintf("http-scale-%05d", i)
			if _, e = file.Exec(id, ids[index], []string{"linked", "managed"}[index], stamp); e != nil {
				return e
			}
			if _, e = doc.Exec("doc-"+id, ids[index], id, id+".pdf", id+".pdf", stamp); e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"/system/dashboard", "/system/dashboard/libraries", "/libraries?archived=false"} {
		started := time.Now()
		response := f.request("GET", endpoint, "", cookie, "")
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", endpoint, response.Code, response.Body.String())
		}
		t.Logf("endpoint=%s documents=10000 time=%s bytes=%d", endpoint, time.Since(started), response.Body.Len())
	}
}
