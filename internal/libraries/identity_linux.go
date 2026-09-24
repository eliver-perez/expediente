package libraries

import (
	"fmt"
	"golang.org/x/sys/unix"
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
	var extended unix.Statx_t
	if err := unix.Statx(int(file.Fd()), "", unix.AT_EMPTY_PATH, unix.STATX_BTIME, &extended); err == nil && extended.Mask&unix.STATX_BTIME != 0 {
		return fmt.Sprintf("linux:%d:%d:%d:%d", stat.Dev, stat.Ino, extended.Btime.Sec, extended.Btime.Nsec), true, nil
	}
	return fmt.Sprintf("linux:%d:%d", stat.Dev, stat.Ino), false, nil
}
