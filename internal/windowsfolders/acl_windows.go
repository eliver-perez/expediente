package windowsfolders

import (
	"fmt"
	"golang.org/x/sys/windows"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

func openDirectory(path string, access uint32) (windows.Handle, error) {
	// Metadata-only rights do not establish delete-sharing protection for
	// directory handles on Windows. FILE_TRAVERSE does, and remains available
	// on ancestors whose contents cannot be listed.
	access |= windows.FILE_TRAVERSE
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, operationError("UTF16PtrFromString", path, err)
	}
	handle, err := windows.CreateFile(ptr, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return 0, &OperationError{Operation: "CreateFileW(directory)", Resource: path, Access: access, Share: windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE, Err: err}
	}
	var info windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(handle, &info); err != nil {
		windows.CloseHandle(handle)
		return 0, operationError("GetFileInformationByHandle", path, err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		windows.CloseHandle(handle)
		return 0, operationError("validate_directory", path, fmt.Errorf("enlace o directorio no válido"))
	}
	return handle, nil
}
func grant(handle windows.Handle, sid *windows.SID, mask windows.ACCESS_MASK, inherit uint32, mode windows.ACCESS_MODE) error {
	return grantDirectory(handle, "selected_directory", sid, mask, inherit, mode, false)
}
func grantDirectory(handle windows.Handle, path string, sid *windows.SID, mask windows.ACCESS_MASK, inherit uint32, mode windows.ACCESS_MODE, ancestor bool) error {
	descriptor, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return operationError("GetSecurityInfo(DACL)", path, err)
	}
	acl, _, err := descriptor.DACL()
	if err != nil {
		return operationError("GetSecurityDescriptorDacl", path, err)
	}
	// A null DACL already grants everyone access. Never turn it into a service-only
	// DACL, which would silently remove the effective access of existing users.
	if acl == nil {
		return nil
	}
	updated, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{AccessPermissions: mask, AccessMode: mode, Inheritance: inherit, Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_USER, TrusteeValue: windows.TrusteeValueFromSID(sid)}}}, acl)
	if err != nil {
		return operationError("SetEntriesInAclW", path, err)
	}
	if ancestor {
		// Only this already-pinned ancestor is changed. NtSetSecurityObject applies
		// a DACL by handle without the Win32 tree-propagation performed by
		// SetSecurityInfo(SE_FILE_OBJECT). No MAXIMUM_ALLOWED/DELETE handle is needed.
		err = setAncestorDACL(handle, path, updated, descriptor)
	} else {
		err = operationError("SetSecurityInfo(selected_root_DACL)", path, windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, updated, nil))
	}
	runtime.KeepAlive(updated)
	runtime.KeepAlive(sid)
	runtime.KeepAlive(descriptor)
	return err
}

var ntSetSecurityObject = windows.NewLazySystemDLL("ntdll.dll").NewProc("NtSetSecurityObject")

func setAncestorDACL(handle windows.Handle, path string, acl *windows.ACL, original *windows.SECURITY_DESCRIPTOR) error {
	descriptor, err := original.ToAbsolute()
	if err != nil {
		return operationError("MakeAbsoluteSD(ancestor)", path, err)
	}
	if err = descriptor.SetDACL(acl, true, false); err != nil {
		return operationError("SetSecurityDescriptorDacl(ancestor)", path, err)
	}
	if err = ntSetSecurityObject.Find(); err != nil {
		return operationError("resolve_NtSetSecurityObject", path, err)
	}
	status, _, _ := ntSetSecurityObject.Call(uintptr(handle), uintptr(windows.DACL_SECURITY_INFORMATION), uintptr(unsafe.Pointer(descriptor)))
	runtime.KeepAlive(descriptor)
	runtime.KeepAlive(acl)
	if int32(status) < 0 {
		return operationError("NtSetSecurityObject(ancestor_DACL)", path, windows.NTStatus(status))
	}
	return nil
}

func Authorize(path, source string, probe func() (bool, error)) error {
	if err := ValidateLocal(path, source); err != nil {
		return err
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return operationError("EvalSymlinks", path, err)
	}
	if !strings.EqualFold(canonical, filepath.Clean(path)) {
		return fmt.Errorf("selecciona la ruta completa de la carpeta, sin enlaces ni nombres abreviados")
	}
	drive, err := windows.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if err != nil {
		return err
	}
	kind := windows.GetDriveType(drive)
	if kind != windows.DRIVE_FIXED && kind != windows.DRIVE_REMOVABLE && kind != windows.DRIVE_RAMDISK {
		return fmt.Errorf("las rutas de red requieren permisos en su servidor")
	}
	// Block nonstandard locations of Windows/program directories too.
	for _, id := range []*windows.KNOWNFOLDERID{windows.FOLDERID_Windows, windows.FOLDERID_ProgramFiles, windows.FOLDERID_ProgramFilesX86, windows.FOLDERID_ProgramData} {
		protected, err := windows.KnownFolderPath(id, 0)
		if err != nil {
			return err
		}
		if protected != "" && (strings.EqualFold(path, protected) || strings.HasPrefix(strings.ToLower(path), strings.ToLower(protected)+`\`)) {
			return fmt.Errorf("ubicación protegida")
		}
	}
	sid, _, _, err := windows.LookupSID("", `NT SERVICE\AIDEX`)
	if err != nil {
		return operationError("LookupAccountNameW(service_SID)", `NT SERVICE\AIDEX`, err)
	}
	// Hold all path components without FILE_SHARE_DELETE. An ancestor/junction
	// cannot be swapped while changing the selected object's ACL.
	handles := []windows.Handle{}
	defer func() {
		for _, h := range handles {
			windows.CloseHandle(h)
		}
	}()
	paths := []string{}
	for current := filepath.Clean(path); len(current) > 3; current = filepath.Dir(current) {
		paths = append(paths, current)
	}
	for index := len(paths) - 1; index >= 0; index-- {
		// Ancestors are pinned with security-read access only. MAXIMUM_ALLOWED
		// also requests data/delete access and conflicts with ordinary Go directory
		// readers that do not share deletion. Reopen for WRITE_DAC only if needed.
		access := uint32(windows.READ_CONTROL)
		if index == 0 {
			access = windows.READ_CONTROL | windows.WRITE_DAC
		}
		h, err := openDirectory(paths[index], access)
		if err != nil {
			return err
		}
		handles = append(handles, h)
	}
	// Other principals, deny entries, owners and protection flags are preserved.
	// Only the service receives RX or Modify (never WRITE_DAC/WRITE_OWNER).
	mask := windows.ACCESS_MASK(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_EXECUTE)
	if source == "managed" {
		mask |= windows.FILE_GENERIC_WRITE | windows.DELETE
	}
	slog.Info("folder authorization ACL", "operation", "grant_selected_root", "path", path, "source", source)
	if err = grantDirectory(handles[len(handles)-1], path, sid, mask, windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE, windows.SET_ACCESS, false); err != nil {
		return err
	}
	if probe != nil {
		accessible, err := probe()
		if err != nil {
			return err
		}
		if accessible {
			slog.Info("folder authorization ancestors unchanged", "reason", "service_access_verified")
			return nil
		}
	}
	for index := 0; index < len(handles)-1; index++ {
		slog.Info("folder authorization ACL", "operation", "grant_ancestor_traverse", "ancestor", paths[len(paths)-1-index])
		parent := paths[len(paths)-1-index]
		change, openErr := openDirectory(parent, windows.READ_CONTROL|windows.WRITE_DAC)
		if openErr != nil {
			return openErr
		}
		// Compare identities as well as holding the non-delete-shared pins.
		var pinned, current windows.ByHandleFileInformation
		err = windows.GetFileInformationByHandle(handles[index], &pinned)
		if err == nil {
			err = windows.GetFileInformationByHandle(change, &current)
		}
		if err != nil {
			windows.CloseHandle(change)
			return operationError("GetFileInformationByHandle(ancestor_identity)", parent, err)
		}
		if pinned.VolumeSerialNumber != current.VolumeSerialNumber || pinned.FileIndexHigh != current.FileIndexHigh || pinned.FileIndexLow != current.FileIndexLow {
			windows.CloseHandle(change)
			return operationError("verify_ancestor_identity", parent, fmt.Errorf("cambió la identidad del directorio"))
		}
		err = grantDirectory(change, parent, sid, windows.FILE_TRAVERSE, 0, windows.GRANT_ACCESS, true)
		windows.CloseHandle(change)
		if err != nil {
			return err
		}
	}
	slog.Info("folder authorization ACL completed", "directories", len(handles), "explicit_recursion", false)
	return nil
}
