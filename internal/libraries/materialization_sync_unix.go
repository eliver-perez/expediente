//go:build darwin || linux

package libraries

import "os"

func syncManagedDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return scanFailure("STORAGE_UNAVAILABLE")
	}
	defer directory.Close()
	if err = directory.Sync(); err != nil {
		return scanFailure("STORAGE_SYNC_UNSUPPORTED")
	}
	return nil
}
