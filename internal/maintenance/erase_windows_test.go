package maintenance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gestor-documental/internal/config"
	"gestor-documental/internal/storage"
)

func TestEraseWindowsInstallationRemovesPrivateCluster(t *testing.T) {
	// Use a fresh, genuinely private cluster, never the shared integration DSN.
	t.Setenv("AIDEX_TEST_POSTGRES_DSN", "")
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(directory, "AIDEX")
	path := filepath.Join(data, "config.json")
	if err = config.InitializeWith(path, func(*config.Config) {}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cluster := filepath.Join(cfg.StateDirectory, "postgresql")
	write := func(name string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("unchanged"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(data, "connection.json"))
	originals := []string{filepath.Join(directory, "linked", "original.pdf"), filepath.Join(directory, "managed", "original.pdf"), filepath.Join(cfg.PrivateUploadDirectory(), "pending.pdf"), filepath.Join(cfg.StateDirectory, "unknown.pdf")}
	for _, name := range originals {
		write(name)
	}
	db, err := storage.Open(context.Background(), cfg.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Writer.Exec("INSERT INTO processing_settings VALUES(1,'manual',1,1,1,1,'2026-09-27')"); err != nil {
		t.Fatal(err)
	}
	if err = EraseWindowsInstallation(path); err == nil {
		t.Fatal("erasure accepted an open cluster")
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(cluster, "data", "original.pdf")
	write(unknown)
	if err = EraseWindowsInstallation(path); err == nil {
		t.Fatal("erasure accepted an unrecognized file in PostgreSQL")
	}
	for _, name := range []string{path, unknown, filepath.Join(cluster, "installation.json"), filepath.Join(cluster, "data", "PG_VERSION")} {
		if _, err = os.Stat(name); err != nil {
			t.Fatalf("failed validation removed %s: %v", name, err)
		}
	}
	if err = os.Remove(unknown); err != nil {
		t.Fatal(err)
	}
	if err = EraseWindowsInstallation(path); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{path, cluster, filepath.Join(data, "connection.json")} {
		if _, err = os.Lstat(name); !os.IsNotExist(err) {
			t.Fatalf("internal path survived: %s: %v", name, err)
		}
	}
	for _, name := range originals {
		contents, err := os.ReadFile(name)
		if err != nil || string(contents) != "unchanged" {
			t.Fatalf("original changed: %s: %v", name, err)
		}
	}
	if err = EraseWindowsInstallation(path); err != nil {
		t.Fatalf("confirmed uninstallation could not be retried: %v", err)
	}
}
