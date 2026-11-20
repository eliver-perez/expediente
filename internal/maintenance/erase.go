// Package maintenance implements explicit local maintenance while the service
// is stopped. It never traverses library roots.
package maintenance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gestor-documental/internal/config"
	"gestor-documental/internal/storage"
)

var backupTimestamp = regexp.MustCompile(`^\d{8}T\d{6}\.\d{9}$`)
var installationArchive = regexp.MustCompile(`^installation\.previous-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\.json$`)

func recognizedArchive(path, configPath string) bool {
	name := filepath.Base(path)
	if strings.HasPrefix(path, configPath+".before-license-") {
		return backupTimestamp.MatchString(strings.TrimPrefix(path, configPath+".before-license-"))
	}
	return installationArchive.MatchString(name)
}

// EraseInternal removes only reserved metadata files. Uploads, managed originals,
// snapshots, and unknown files are deliberately left at their existing paths.
// The caller must obtain explicit confirmation before invoking this operation.
func EraseInternal(configPath string) error {
	return eraseInternal(configPath, false)
}

// EraseWindowsInstallation also removes the stopped, private PostgreSQL cluster
// owned by the Windows package. The caller has obtained explicit confirmation.
func EraseWindowsInstallation(configPath string) error {
	configPath = filepath.Clean(configPath)
	if !filepath.IsAbs(configPath) || !strings.EqualFold(filepath.Base(configPath), "config.json") || !strings.EqualFold(filepath.Base(filepath.Dir(configPath)), "AIDEX") {
		return fmt.Errorf("la eliminación completa requiere el área interna del paquete AIDEX")
	}
	return eraseInternal(configPath, true)
}

func eraseInternal(configPath string, windowsPackage bool) error {
	if windowsPackage {
		if err := regularWithoutLinks(configPath); err != nil {
			return err
		}
	}
	configuration, err := config.Load(configPath)
	if err != nil {
		if windowsPackage && os.IsNotExist(err) {
			// A rerun after successful erasure is harmless. Without configuration,
			// any remaining cluster could instead belong to a recoverable setup.
			if _, statErr := os.Lstat(filepath.Join(filepath.Dir(configPath), "state", "postgresql")); os.IsNotExist(statErr) {
				return nil
			}
			return fmt.Errorf("existe un clúster sin configuración verificable; se conservan los datos")
		}
		return err
	}
	state := configuration.StateDirectory
	if windowsPackage {
		if !strings.EqualFold(filepath.Clean(state), filepath.Join(filepath.Dir(configPath), "state")) {
			return fmt.Errorf("el estado está fuera del área interna del paquete; se conservan los datos")
		}
		cluster := filepath.Join(state, "postgresql")
		uploads := configuration.PrivateUploadDirectory()
		if containsPath(cluster, uploads) || containsPath(uploads, cluster) {
			return fmt.Errorf("las cargas se superponen con el clúster PostgreSQL; se conservan los datos")
		}
	}
	unlock, err := storage.LockState(configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer unlock()
	paths := []string{filepath.Join(state, "license", "installation.json"), filepath.Join(state, "license", "recovery.json")}
	if windowsPackage {
		paths = append(paths, filepath.Join(filepath.Dir(configPath), "connection.json"))
	}
	for _, pattern := range []string{filepath.Join(state, "license", "installation.previous-*.json"), configPath + ".before-license-*"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}
		for _, match := range matches {
			if recognizedArchive(match, configPath) {
				paths = append(paths, match)
			}
		}
	}
	// Include package markers at their known locations, not documents under uploads.
	for _, directory := range []string{state, filepath.Dir(configPath)} {
		for _, name := range []string{"service-enabled", "package-version"} {
			paths = append(paths, filepath.Join(directory, name))
		}
	}
	// Remove configuration last so a failed cleanup remains retryable.
	paths = append(paths, configPath)
	unique := map[string]bool{}
	for _, path := range paths {
		if unique[path] {
			continue
		}
		unique[path] = true
		if err := regularWithoutLinks(path); err != nil {
			return err
		}
	}
	if windowsPackage {
		if err := storage.ErasePrivatePostgres(state); err != nil {
			return err
		}
	} else if storage.PostgresPresent(state) {
		if err := storage.ResetPostgres(context.Background(), state); err != nil {
			return err
		}
	}
	for path := range unique {
		if path == configPath {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Remove(configPath)
}

func regularWithoutLinks(path string) error {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || (current == path && !info.Mode().IsRegular()) || (current != path && !info.IsDir())) {
			return fmt.Errorf("refusing unsafe metadata path: %s", current)
		}
		if filepath.Dir(current) == current {
			return nil
		}
	}
}
