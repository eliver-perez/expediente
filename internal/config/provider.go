package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
	"os"
	"path/filepath"
	"time"
)

// PrepareProvider validates a proposed local provisioning change without touching
// SQLite or the file. Only maintenance commands may use this path; serve uses Load.
func PrepareProvider(path string) (Config, []byte, error) {
	var configuration Config
	info, err := os.Lstat(path)
	if err != nil {
		return configuration, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return configuration, nil, fmt.Errorf("configuration must be a regular private file")
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return configuration, nil, err
	}
	var object map[string]json.RawMessage
	if err = json.Unmarshal(original, &object); err != nil {
		return configuration, nil, err
	}
	var options licensing.Options
	if raw, ok := object["license"]; ok {
		if err = json.Unmarshal(raw, &options); err != nil {
			return configuration, nil, err
		}
	}
	if options.DevelopmentCAFile != "" {
		return configuration, nil, fmt.Errorf("remove the development CA before provisioning production licensing")
	}
	key := licensing.ProviderKey()
	found := false
	for _, existing := range options.TrustedKeys {
		if existing.Kid == key.Kid {
			if existing != key {
				return configuration, nil, fmt.Errorf("provider kid already has a different trust key")
			}
			found = true
		}
	}
	if !found {
		options.TrustedKeys = append(options.TrustedKeys, key)
	}
	if options.ServerURL == "" {
		options.ServerURL = licensing.ProviderURL
		options.RefreshHours = 24
	}
	options.DevelopmentBypass = false
	// Preserve other registered keys. Production Validate rejects development trust.
	object["license"], err = json.Marshal(options)
	if err != nil {
		return configuration, nil, err
	}
	contents, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		return configuration, nil, err
	}
	contents = append(contents, '\n')
	configuration = Defaults("")
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&configuration); err != nil {
		return configuration, nil, err
	}
	if err = configuration.Validate(); err != nil {
		return configuration, nil, err
	}
	return configuration, contents, nil
}
func ProvisionProvider(path string) error {
	configuration, contents, err := PrepareProvider(path)
	if err != nil {
		return err
	}
	unlock, err := storage.LockState(configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer unlock()
	// Re-read under the exclusive lock; a running service or maintenance must stop.
	_, contents, err = PrepareProvider(path)
	if err != nil {
		return err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.Equal(original, contents) {
		return nil
	}
	backup := path + ".before-license-" + time.Now().UTC().Format("20060102T150405.000000000")
	if err = os.WriteFile(backup, original, 0600); err != nil {
		return err
	}
	if err = storage.ProtectPrivatePath(backup, false); err != nil {
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
	if _, err = file.Write(contents); err != nil {
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
