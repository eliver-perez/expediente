package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLANRequiresHTTPSAndTrustedProxy(t *testing.T) {
	configuration := Defaults(filepath.Join(t.TempDir(), "state"))
	configuration.ListenAddress = "0.0.0.0:8090"
	if err := configuration.Validate(); err == nil {
		t.Fatal("insecure LAN accepted")
	}
	configuration.PublicURL = "https://documental.example"
	if err := configuration.Validate(); err == nil {
		t.Fatal("missing TLS/proxy accepted")
	}
	configuration.TrustedProxies = []string{"127.0.0.1/32"}
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerInitializationPreservesExistingConfiguration(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "configuration", "config.json")
	state := filepath.Join(directory, "separate-state")
	if err := InitializeWith(path, func(configuration *Config) {
		configuration.StateDirectory = state
		configuration.ListenAddress = "127.0.0.1:18090"
		configuration.PublicURL = "http://127.0.0.1:18090"
		configuration.Indexing.TessdataDirectory = filepath.Join(directory, "Spanish languages")
	}); err != nil {
		t.Fatal(err)
	}
	configuration, err := Load(path)
	if err != nil || configuration.StateDirectory != state || configuration.ListenAddress != "127.0.0.1:18090" {
		t.Fatalf("installed configuration: %+v, %v", configuration, err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := InitializeWith(path, func(configuration *Config) { configuration.StateDirectory = filepath.Join(directory, "replacement") }); err == nil {
		t.Fatal("reinstall replaced existing configuration")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("configuration modified on failed reinitialization")
	}
	if err := InitializeWith(filepath.Join(directory, "unsafe.json"), func(configuration *Config) { configuration.ListenAddress = "0.0.0.0:18090" }); err == nil {
		t.Fatal("installer bypassed HTTPS validation")
	}
}

func TestConfigurationNeverOverwritesExistingFileOrUsesPublicRoot(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.json")
	if err := Initialize(path); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(path); err == nil {
		t.Fatal("overwritten configuration")
	}
	configuration, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	configuration.StateDirectory = filepath.Join(directory, "htdocs", "state")
	if err := configuration.Validate(); err == nil {
		t.Fatal("public state accepted")
	}
}
