package maintenance

import (
	"context"
	"gestor-documental/internal/config"
	"gestor-documental/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestErasePreservesPhysicalDocumentsAndAllowsFreshInstall(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.json")
	if err = config.InitializeWith(path, func(*config.Config) {}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	database, err := storage.Open(context.Background(), c.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = database.Writer.Exec("INSERT INTO processing_settings VALUES(1,'manual',1,1,1,1,'2026-09-27')"); err != nil {
		t.Fatal(err)
	}
	database.Close()
	originals := []string{filepath.Join(directory, "linked", "original.pdf"), filepath.Join(directory, "managed", "original.pdf"), filepath.Join(c.StateDirectory, "uploads", "pending.pdf"), filepath.Join(c.StateDirectory, "uploads", "pending.part"), filepath.Join(c.StateDirectory, "extraction", "job-1", "source.pdf"), filepath.Join(c.StateDirectory, "upgrade-backups", "user.pdf"), filepath.Join(c.StateDirectory, "unknown.txt")}
	originals = append(originals, path+".before-license-original.pdf", filepath.Join(c.StateDirectory, "license", "installation.previous-document.json"))
	for _, file := range originals {
		if err = os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(file, []byte("original unchanged"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	configBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	databaseBytes, err := os.ReadFile(filepath.Join(c.StateDirectory, "documental.db"))
	if err != nil {
		t.Fatal(err)
	}
	metadata := map[string][]byte{
		path + ".before-license-20260927T170000.000000000":                                                            configBytes,
		filepath.Join(c.StateDirectory, "upgrade-backups", "before-processing-20260927T170000.000000000.db"):          databaseBytes,
		filepath.Join(c.StateDirectory, "license", "installation.previous-11111111-1111-4111-8111-111111111111.json"): []byte(`{"installation_id":"11111111-1111-4111-8111-111111111111"}`),
		filepath.Join(c.StateDirectory, "license", "installation.json"):                                               []byte(`{"installation_id":"current"}`),
	}
	for file, contents := range metadata {
		if err = os.WriteFile(file, contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = EraseInternal(path); err != nil {
		t.Fatal(err)
	}
	for file := range metadata {
		if _, err = os.Stat(file); !os.IsNotExist(err) {
			t.Fatal("metadata survived", file, err)
		}
	}
	for _, file := range originals {
		data, err := os.ReadFile(file)
		if err != nil || string(data) != "original unchanged" {
			t.Fatalf("original changed: %s: %v", file, err)
		}
	}
	if _, err = os.Stat(filepath.Join(c.StateDirectory, "documental.db")); !os.IsNotExist(err) {
		t.Fatal("database survived", err)
	}
	if err = config.InitializeWith(path, func(*config.Config) {}); err != nil {
		t.Fatal(err)
	}
	database, err = storage.Open(context.Background(), c.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var count int
	if err = database.Reader.QueryRow("SELECT count(*) FROM processing_settings").Scan(&count); err != nil || count != 0 {
		t.Fatal("not a fresh database", count, err)
	}
}

func TestEraseRefusesLinksBeforeDeletingAnything(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.json")
	if err = config.InitializeWith(path, func(*config.Config) {}); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load(path)
	database, err := storage.Open(context.Background(), c.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	database.Close()
	outside := filepath.Join(directory, "originals")
	os.Mkdir(outside, 0700)
	if err = os.Symlink(outside, filepath.Join(c.StateDirectory, "license")); err != nil {
		t.Skip(err)
	}
	if err = EraseInternal(path); err == nil {
		t.Fatal("accepted linked metadata directory")
	}
	for _, file := range []string{path, filepath.Join(c.StateDirectory, "documental.db")} {
		if _, err = os.Stat(file); err != nil {
			t.Fatal("deleted before validation", err)
		}
	}
}
