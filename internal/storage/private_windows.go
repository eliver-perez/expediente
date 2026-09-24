package storage

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func ProtectPrivatePath(path string, directory bool) error {
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	inheritance := ""
	if directory {
		inheritance = "OICI"
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;" + inheritance + ";FA;;;SY)(A;" + inheritance + ";FA;;;BA)(A;" + inheritance + ";FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	accessList, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, accessList, nil)
}

func requireLocalFilesystem(directory string) error {
	path, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return err
	}
	var volume [windows.MAX_PATH + 1]uint16
	if err := windows.GetVolumePathName(path, &volume[0], uint32(len(volume))); err != nil {
		return err
	}
	driveType := windows.GetDriveType(&volume[0])
	if driveType != windows.DRIVE_FIXED && driveType != windows.DRIVE_RAMDISK {
		return fmt.Errorf("SQLite state must be on a local drive")
	}
	return nil
}

func LockState(directory string) (func() error, error) {
	if err := PreparePrivateDirectory(directory); err != nil {
		return nil, err
	}
	path, err := windows.UTF16PtrFromString(filepath.Join(directory, "service.lock"))
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(path, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, fmt.Errorf("another service or maintenance command may be using this state: %w", err)
	}
	var attributes windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &attributes); err != nil || attributes.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(handle)
		return nil, fmt.Errorf("invalid state lock file")
	}
	return func() error { return windows.CloseHandle(handle) }, nil
}
