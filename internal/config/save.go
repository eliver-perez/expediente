package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gestor-documental/internal/storage"
)

// Save replaces a validated configuration inside its private directory. The
// service holds the installation lock; a failed write leaves the old file intact.
func Save(path string, configuration Config) error {
	if err := configuration.Validate(); err != nil {
		return err
	}
	if _, err := Load(path); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".aibid-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err = storage.ProtectPrivatePath(file.Name(), false); err != nil {
		return err
	}
	if _, err = file.Write(append(contents, '\n')); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
