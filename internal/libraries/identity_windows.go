package libraries

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
)

func physicalIdentity(file *os.File) (string, bool, error) {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return "", false, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return "", false, fmt.Errorf("reparse point skipped")
	}
	return fmt.Sprintf("windows:%d:%d:%d:%d:%d", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime), true, nil
}
