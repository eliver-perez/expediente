package libraries

import (
	"fmt"
	"os"
	"syscall"
)

func physicalIdentity(file *os.File) (string, bool, error) {
	info, err := file.Stat()
	if err != nil {
		return "", false, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", false, fmt.Errorf("unsupported file identity")
	}
	return fmt.Sprintf("darwin:%d:%d:%d:%d", stat.Dev, stat.Ino, stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec), true, nil
}
