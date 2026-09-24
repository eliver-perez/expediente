package storage

import (
	"fmt"
	"golang.org/x/sys/unix"
)

func requireLocalFilesystem(directory string) error {
	var filesystem unix.Statfs_t
	if err := unix.Statfs(directory, &filesystem); err != nil {
		return err
	}
	if filesystem.Flags&unix.MNT_LOCAL == 0 {
		return fmt.Errorf("SQLite state must be on a local filesystem")
	}
	return nil
}
