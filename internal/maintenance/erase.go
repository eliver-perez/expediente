// Package maintenance implements explicit local maintenance while the service
// is stopped. It never traverses library roots or removes a directory tree.
package maintenance

import (
	"bytes"
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
	if strings.HasPrefix(name, "before-processing-") {
		return backupTimestamp.MatchString(strings.TrimSuffix(strings.TrimPrefix(name, "before-processing-"), ".db"))
	}
	return installationArchive.MatchString(name)
}

// EraseInternal removes only reserved metadata files. Uploads, managed originals,
// snapshots, and unknown files are deliberately left at their existing paths.
// The caller must obtain explicit confirmation before invoking this operation.
func EraseInternal(configPath string) error {
	configuration, err := config.Load(configPath)
	if err != nil {
		return err
	}
	unlock, err := storage.LockState(configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer unlock()
	state := configuration.StateDirectory
	paths := []string{filepath.Join(state, "documental.db-wal"), filepath.Join(state, "documental.db-shm"), filepath.Join(state, "documental.db"), filepath.Join(state, "license", "installation.json"), filepath.Join(state, "license", "recovery.json")}
	for _, pattern := range []string{filepath.Join(state, "upgrade-backups", "before-processing-*.db"), filepath.Join(state, "license", "installation.previous-*.json"), configPath + ".before-license-*"} {
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
		if strings.HasSuffix(path, ".db") {
			file, err := os.Open(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			header := make([]byte, 16)
			n, readErr := file.Read(header)
			file.Close()
			if readErr != nil || n != 16 || !bytes.Equal(header, []byte("SQLite format 3\x00")) {
				return fmt.Errorf("refusing to remove an unrecognized database: %s", path)
			}
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
