//go:build darwin || linux

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func ProtectPrivatePath(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("private path is a symlink")
	}
	permissions := os.FileMode(0600)
	if directory {
		permissions = 0700
	}
	return os.Chmod(path, permissions)
}

// Hold the same kernel lock for serve, bootstrap and migration commands.
func LockState(directory string) (func() error, error) {
	if err := PreparePrivateDirectory(directory); err != nil {
		return nil, err
	}
	fileDescriptor, err := unix.Open(filepath.Join(directory, "service.lock"), unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	lockFile := os.NewFile(uintptr(fileDescriptor), "service.lock")
	if err := unix.Flock(fileDescriptor, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("another service or maintenance command is using this state")
	}
	return lockFile.Close, nil
}
