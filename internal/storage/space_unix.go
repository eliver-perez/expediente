//go:build darwin || linux

package storage

import "golang.org/x/sys/unix"

func AvailableBytes(directory string) (uint64, error) {
	var stats unix.Statfs_t
	if err := unix.Statfs(directory, &stats); err != nil {
		return 0, err
	}
	return uint64(stats.Bavail) * uint64(stats.Bsize), nil
}
