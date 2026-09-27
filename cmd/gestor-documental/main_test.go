package main

import (
	"context"
	"gestor-documental/internal/config"
	"gestor-documental/internal/storage"
	"os"
	"path/filepath"
	"testing"
)

func TestInstalledCommandDoesNotRequireUserProfile(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, variable := range []string{"HOME", "APPDATA", "XDG_CONFIG_HOME"} {
		t.Setenv(variable, "")
	}
	if _, err := config.DefaultPath(); err == nil {
		t.Fatal("fixture still has a default user configuration")
	}
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	path := filepath.Join(directory, "config.json")
	os.Args = []string{"gestor-documental", "init", "--config", path, "--listen", "127.0.0.1:18090"}
	if err := run(context.Background(), func() {}); err != nil {
		t.Fatal(err)
	}
	if configuration, err := config.Load(path); err != nil || configuration.ListenAddress != "127.0.0.1:18090" {
		t.Fatalf("explicit configuration: %+v, %v", configuration, err)
	}
}

func TestInstallerCheckStateDoesNotOpenDatabaseAndRequiresExclusiveLock(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.json")
	if err := config.Initialize(path); err != nil {
		t.Fatal(err)
	}
	configuration, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configuration.StateDirectory, 0700); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(configuration.StateDirectory, "documental.db")
	// A deliberately unreadable SQLite fixture proves preflight never migrates,
	// resets or needs to open the DB before replacing the faulty old executable.
	contents := []byte("preexisting state must remain byte-for-byte unchanged")
	if err := os.WriteFile(databasePath, contents, 0600); err != nil {
		t.Fatal(err)
	}
	previous := os.Args
	t.Cleanup(func() { os.Args = previous })
	os.Args = []string{"gestor-documental", "check-state", "--config", path}
	if err := run(context.Background(), func() {}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(databasePath)
	if err != nil || string(after) != string(contents) {
		t.Fatalf("changed database: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configuration.StateDirectory, "license")); !os.IsNotExist(err) {
		t.Fatal("check-state created a license identity")
	}
	unlock, err := storage.LockState(configuration.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if err := run(context.Background(), func() {}); err == nil {
		t.Fatal("preflight accepted an in-use state")
	}
}
