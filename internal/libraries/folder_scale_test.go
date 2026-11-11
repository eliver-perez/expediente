//go:build development

package libraries

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gestor-documental/internal/domain"
)

func TestFolderNavigationScale(t *testing.T) {
	if os.Getenv("AIDEX_PERF_FOLDERS") != "1" {
		t.Skip("set AIDEX_PERF_FOLDERS=1")
	}
	s, p, dir := fixture(t)
	library := addLibrary(t, s, p, "Folder scale")
	path := filepath.Join(dir, "originals")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	root := addRoot(t, s, p, library, path)
	stamp := domain.Timestamp(time.Now())
	insertStarted := time.Now()
	if err := s.Database.Write(context.Background(), func(tx *sql.Tx) error {
		file, err := tx.Prepare("INSERT INTO physical_files(id,library_id,storage_source,availability,integrity_status,extraction_freshness,created_at) VALUES(?,?,'linked','available','unknown','none',?)")
		if err != nil {
			return err
		}
		defer file.Close()
		doc, err := tx.Prepare("INSERT INTO documents(id,library_id,physical_file_id,original_filename,filename_search_key,created_at) VALUES(?,?,?,?,?,?)")
		if err != nil {
			return err
		}
		defer doc.Close()
		location, err := tx.Prepare("INSERT INTO physical_file_locations(id,physical_file_id,library_id,storage_source,root_id,relative_path,canonical_path,comparison_key,root_configuration_revision,first_seen_at,last_seen_at) VALUES(?,?,?,'linked',?,?,?,?,1,?,?)")
		if err != nil {
			return err
		}
		defer location.Close()
		primary, err := tx.Prepare("UPDATE physical_files SET primary_location_id=? WHERE id=?")
		if err != nil {
			return err
		}
		defer primary.Close()
		for i := 0; i < 50000; i++ {
			id := fmt.Sprintf("folder-file-%05d", i)
			relative := fmt.Sprintf("folder-%03d/sub-%03d/%s.pdf", i%100, i/100%10, id)
			canonical := filepath.Join(path, filepath.FromSlash(relative))
			if _, err = file.Exec(id, library, stamp); err != nil {
				return err
			}
			if _, err = doc.Exec("doc-"+id, library, id, id+".pdf", id+".pdf", stamp); err != nil {
				return err
			}
			if _, err = location.Exec("loc-"+id, id, library, root, relative, canonical, canonical, stamp, stamp); err != nil {
				return err
			}
			if _, err = primary.Exec("loc-"+id, id); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Logf("insert=%s", time.Since(insertStarted))
	for _, prefix := range []string{"", "folder-042"} {
		started := time.Now()
		folders, _, err := s.Folders(context.Background(), p, library, root, "", prefix, "")
		if err != nil {
			t.Fatal(err)
		}
		if prefix == "" && len(folders) != 50 || prefix != "" && len(folders) != 10 {
			t.Fatalf("prefix=%q count=%d", prefix, len(folders))
		}
		t.Logf("prefix=%q folders=%d time=%s", prefix, len(folders), time.Since(started))
	}
}
