//go:build development && windows

package libraries

import (
	"context"
	"os"
	"testing"
	"time"
)

// This opt-in test reads an existing linked root and writes only to its isolated
// PostgreSQL test schema. It never starts extraction or modifies source files.
func TestWindowsRealLinkedRootOnPostgres(t *testing.T) {
	path := os.Getenv("AIDEX_REAL_DOCUMENT_ROOT")
	if path == "" {
		t.Skip("set AIDEX_REAL_DOCUMENT_ROOT to a readable linked root")
	}
	service, principal, _ := fixture(t)
	library := addLibrary(t, service, principal, "Corpus real Windows")
	started := time.Now()
	root := addRoot(t, service, principal, library, path)
	planDuration := time.Since(started)
	started = time.Now()
	if err := service.Scan(context.Background(), root, 256<<20, nil); err != nil {
		t.Fatal(err)
	}
	scanDuration := time.Since(started)
	var documents, locations, jobs int
	for _, item := range []struct {
		query string
		arg   string
		out   *int
	}{
		{"SELECT count(*) FROM documents WHERE library_id=?", library, &documents},
		{"SELECT count(*) FROM physical_file_locations WHERE root_id=?", root, &locations},
		{"SELECT count(*) FROM jobs WHERE library_id=?", library, &jobs},
	} {
		if err := service.Database.Reader.QueryRow(item.query, item.arg).Scan(item.out); err != nil {
			t.Fatal(err)
		}
	}
	if documents < 1000 || locations < 1000 {
		t.Fatalf("real scan too small: documents=%d locations=%d jobs=%d", documents, locations, jobs)
	}
	t.Logf("real linked root: documents=%d locations=%d jobs=%d plan=%s scan=%s", documents, locations, jobs, planDuration, scanDuration)
}
