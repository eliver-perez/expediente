package config

import (
	"encoding/base64"
	"encoding/json"
	"gestor-documental/internal/licensing"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderProvisionPreservesConfigurationAndTrust(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.json")
	c := Defaults(filepath.Join(directory, "state"))
	c.PublicURL = "http://127.0.0.1:18090"
	c.ListenAddress = "127.0.0.1:18090"
	c.License = licensing.Options{RefreshHours: 24, DevelopmentBypass: true, TrustedKeys: []licensing.TrustKey{{Kid: "previous-prod", PublicKey: base64.RawURLEncoding.EncodeToString(make([]byte, 32)), Environment: "production", Purpose: "license"}}}
	data, _ := json.Marshal(c)
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = ProvisionProvider(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.License.ServerURL != licensing.ProviderURL || loaded.License.DevelopmentBypass || len(loaded.License.TrustedKeys) != 2 || loaded.StateDirectory != c.StateDirectory || loaded.ListenAddress != c.ListenAddress || loaded.Indexing.PDFText != c.Indexing.PDFText {
		t.Fatal("provisioning replaced unrelated configuration")
	}
	if err = ProvisionProvider(path); err != nil {
		t.Fatal(err)
	}
	loaded, _ = Load(path)
	if len(loaded.License.TrustedKeys) != 2 {
		t.Fatal("duplicate provider trust key")
	}
	snapshots, _ := filepath.Glob(path + ".before-license-*")
	if len(snapshots) != 1 {
		t.Fatal("expected one pre-provision backup", snapshots)
	}
	contents, _ := os.ReadFile(snapshots[0])
	if string(contents) != string(data) {
		t.Fatal("backup changed original configuration")
	}
}
