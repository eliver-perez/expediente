package backups

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gestor-documental/internal/buildinfo"
)

// writeRecoveryPoint captures only the portable logical allowlist. A raw
// SQLite copy would also copy the destination's activation and private keys.
func (s *Service) writeRecoveryPoint(ctx context.Context, id string) error {
	final := filepath.Join(s.directory, id+".pre-restore.aidexbak")
	if _, err := os.Lstat(final); err == nil {
		pkg, err := inspectZip(final)
		if err != nil {
			return fmt.Errorf("existing recovery point is invalid: %w", err)
		}
		defer pkg.Close()
		if pkg.Manifest.Kind != "basic" {
			return fmt.Errorf("existing recovery point is not logical")
		}
		return validateInventory(pkg)
	} else if !os.IsNotExist(err) {
		return err
	}
	plain, err := os.CreateTemp(s.directory, ".recovery-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(plain.Name())
	if err = plain.Chmod(0600); err != nil {
		plain.Close()
		return err
	}
	archive := zip.NewWriter(plain)
	tx, err := s.DB.Reader.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	defer tx.Rollback()
	manifest := Manifest{Product: "AIDEX", Format: 1, Schema: 15, Build: buildinfo.DisplayVersion(), Kind: "basic", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Tables: map[string]int64{}, Entries: []Entry{}}
	for _, table := range tables {
		if err = ctx.Err(); err != nil {
			archive.Close()
			plain.Close()
			return err
		}
		count, entry, e := exportTable(ctx, tx, archive, table)
		if e != nil {
			archive.Close()
			plain.Close()
			return e
		}
		manifest.Tables[table], manifest.Entries = count, append(manifest.Entries, entry)
	}
	manifest.Documents = manifest.Tables["documents"]
	rows, err := tx.QueryContext(ctx, "SELECT id,library_id,storage_source,canonical_path,status FROM storage_roots ORDER BY id")
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	for rows.Next() {
		var root RootSummary
		if err = rows.Scan(&root.ID, &root.LibraryID, &root.Source, &root.Path, &root.Status); err != nil {
			rows.Close()
			archive.Close()
			plain.Close()
			return err
		}
		manifest.Roots = append(manifest.Roots, root)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	encoded, err := encodeManifest(manifest)
	if err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	if _, err = addZipEntry(archive, "manifest.json", encoded); err != nil {
		archive.Close()
		plain.Close()
		return err
	}
	if err = archive.Close(); err != nil {
		plain.Close()
		return err
	}
	if err = plain.Sync(); err != nil {
		plain.Close()
		return err
	}
	if err = plain.Close(); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	sealed := filepath.Join(s.directory, ".recovery-"+id+".aidexbak")
	defer os.Remove(sealed)
	if err = seal(plain.Name(), sealed, ""); err != nil {
		return err
	}
	return os.Rename(sealed, final)
}
