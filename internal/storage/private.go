package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func PreparePrivateDirectory(directory string) error {
	if !filepath.IsAbs(directory) {
		return fmt.Errorf("state directory must be absolute")
	}
	// Inspect each existing component before creating: no symlinks/junction aliases.
	// /tmp is a macOS system alias; callers canonicalize the parent at configuration time.
	for current := filepath.Clean(directory); ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("private directory cannot traverse symlinks or non-directories")
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	return ProtectPrivatePath(directory, true)
}

func PrepareLocalPrivateDirectory(directory string) error {
	if err := PreparePrivateDirectory(directory); err != nil {
		return err
	}
	return requireLocalFilesystem(directory)
}
