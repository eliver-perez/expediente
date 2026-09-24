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
	switch uint64(filesystem.Type) {
	case 0xef53, 0x58465342, 0x9123683e, 0x01021994, 0x794c7630, 0x2fc12fc1:
		return nil // ext, XFS, Btrfs, tmpfs, overlayfs (CI), ZFS.
	default:
		return fmt.Errorf("SQLite state requires a verified local filesystem (type %x)", filesystem.Type)
	}
}
