package config

import (
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
