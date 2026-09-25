package main

import (
	"context"
	"gestor-documental/internal/config"
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
