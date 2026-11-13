package libraries

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/windowsfolders"
	"golang.org/x/sys/windows"
)

func init() {
	platformInspectDirectory = inspectWindowsDirectory
	platformCheckAuthorizedDirectory = checkWindowsAuthorizedDirectory
}

type windowsAccessIdentity struct {
	SID         string
	TokenSource string
	Restricted  bool
	ExpectedSID string
}

func effectiveWindowsIdentity() (windowsAccessIdentity, error) {
	result := windowsAccessIdentity{TokenSource: "thread"}
	var token windows.Token
	err := windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &token)
	if errors.Is(err, windows.ERROR_NO_TOKEN) {
		result.TokenSource = "process"
		err = windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	}
	if err != nil {
		return result, &windowsfolders.OperationError{Operation: "OpenThreadToken/OpenProcessToken(TOKEN_QUERY)", Resource: "effective_access_token", Err: err}
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return result, &windowsfolders.OperationError{Operation: "GetTokenInformation(TokenUser)", Resource: "effective_access_token", Err: err}
	}
	result.SID = user.User.Sid.String()
	result.Restricted, err = token.IsRestricted()
	if err != nil {
		return result, &windowsfolders.OperationError{Operation: "IsTokenRestricted", Resource: "effective_access_token", Err: err}
	}
	if buildinfo.ServiceName == "AIDEX" {
		expected, _, _, err := windows.LookupSID("", `NT SERVICE\AIDEX`)
		if err != nil {
			return result, &windowsfolders.OperationError{Operation: "LookupAccountNameW(service_SID)", Resource: `NT SERVICE\AIDEX`, Err: err}
		}
		result.ExpectedSID = expected.String()
		if result.SID != result.ExpectedSID {
			return result, domain.Failure("ROOT_SERVICE_IDENTITY_MISMATCH", "La comprobación no se está ejecutando con la cuenta del servicio AIDEX. Revisa el servicio instalado.", 422)
		}
	}
	return result, nil
}
func logWindowsAccess(identity windowsAccessIdentity, source, kind, operation, path string, access uint32, err error) {
	attrs := []any{"role", "service", "pid", os.Getpid(), "phase", "service_effective_access", "operation", operation, "resource", path, "storage_source", source, "access_kind", kind, "desired_access", fmt.Sprintf("0x%08X", access), "effective_sid", identity.SID, "token_source", identity.TokenSource, "token_restricted", identity.Restricted}
	if identity.ExpectedSID != "" {
		attrs = append(attrs, "expected_service_sid", identity.ExpectedSID)
	}
	if err != nil {
		attrs = append(attrs, windowsfolders.DiagnosticAttrs(err)...)
	}
	// Default Windows service logger writes service.log. Mirror failures into the
	// existing private authorization log only in an installed AIDEX service build.
	if err != nil {
		slog.Error("folder authorization service_effective_access failed", attrs...)
		if buildinfo.ServiceName == "AIDEX" {
			if base, e := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0); e == nil {
				if file, e := os.OpenFile(filepath.Join(base, "AIDEX", "logs", "folder-authorization.log"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600); e == nil {
					slog.New(slog.NewJSONHandler(file, nil)).Error("folder authorization service_effective_access failed", attrs...)
					file.Close()
				}
			}
		}
	} else {
		slog.Info("folder authorization service_effective_access", append(attrs, "accessible", true)...)
	}
}
func windowsAccessFailure(identity windowsAccessIdentity, source, kind, operation, path string, access uint32, err error) error {
	logWindowsAccess(identity, source, kind, operation, path, access, err)
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}
	var status windows.NTStatus
	if errors.As(err, &status) {
		err = status.Errno()
	}
	return directoryFailure(path, err)
}

func finalWindowsDirectoryPath(handle windows.Handle) (string, error) {
	buffer := make([]uint16, 512)
	for {
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
		if err != nil {
			if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
				// Some Windows versions query ancestor metadata for the DOS form.
				// The NT form stays available from the opened destination handle.
				ntPath, ntErr := normalizedNTPath(handle)
				if ntErr != nil {
					return "", ntErr
				}
				return localDOSPathFromNT(ntPath)
			}
			return "", err
		}
		if n >= uint32(len(buffer)) {
			if n > 32768 {
				return "", windows.ERROR_FILENAME_EXCED_RANGE
			}
			buffer = make([]uint16, n+1)
			continue
		}
		path := windows.UTF16ToString(buffer[:n])
		if strings.HasPrefix(path, `\\?\UNC\`) {
			return `\\` + path[len(`\\?\UNC\`):], nil
		}
		if strings.HasPrefix(path, `\\?\`) && len(path) >= 7 && path[5] == ':' {
			return path[4:], nil
		}
		return "", fmt.Errorf("el sistema no devolvió una ruta DOS o UNC canónica")
	}
}

func normalizedNTPath(handle windows.Handle) (string, error) {
	buffer := make([]uint16, 512)
	for {
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 2) // VOLUME_NAME_NT
		if err != nil {
			return "", err
		}
		if n >= uint32(len(buffer)) {
			if n > 32768 {
				return "", windows.ERROR_FILENAME_EXCED_RANGE
			}
			buffer = make([]uint16, n+1)
			continue
		}
		return windows.UTF16ToString(buffer[:n]), nil
	}
}

// Capture drive mappings before a caller can impersonate a restricted token.
// QueryDosDevice itself can be denied to that token even when the opened
// directory handle permits the normalized NT path query.
var initialNTDriveMappings = queryNTDriveMappings()

type ntDriveMapping struct{ letter, device string }

func queryNTDriveMappings() []ntDriveMapping {
	drives, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	mappings := make([]ntDriveMapping, 0, 26)
	for index := 0; index < 26; index++ {
		if drives&(1<<index) == 0 {
			continue
		}
		letter := string(rune('A'+index)) + ":"
		name, _ := windows.UTF16PtrFromString(letter)
		buffer := make([]uint16, 32768)
		n, err := windows.QueryDosDevice(name, &buffer[0], uint32(len(buffer)))
		if err != nil {
			continue
		}
		device := windows.UTF16ToString(buffer[:n])
		mappings = append(mappings, ntDriveMapping{letter: letter, device: device})
	}
	return mappings
}

func localDOSPathFromNT(path string) (string, error) {
	for _, mappings := range [][]ntDriveMapping{queryNTDriveMappings(), initialNTDriveMappings} {
		for _, mapping := range mappings {
			if len(path) > len(mapping.device) && strings.EqualFold(path[:len(mapping.device)], mapping.device) && path[len(mapping.device)] == '\\' {
				return mapping.letter + path[len(mapping.device):], nil
			}
		}
	}
	return "", fmt.Errorf("NT path does not map to a local drive")
}

func inspectWindowsDirectory(path string) (directoryInfo, error) {
	// Keep identity capture and native I/O on the same OS thread, including when
	// a test or host supplies an impersonation token. Never impersonate here.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	identity, err := effectiveWindowsIdentity()
	if err != nil {
		return directoryInfo{}, windowsAccessFailure(identity, "inspection", "identity", "effective_service_token", path, windows.TOKEN_QUERY, err)
	}
	return inspectWindowsDirectoryAs(path, identity, "inspection")
}
func inspectWindowsDirectoryAs(path string, identity windowsAccessIdentity, source string) (directoryInfo, error) {
	var result directoryInfo
	if !filepath.IsAbs(path) {
		return result, invalid("La carpeta debe ser una ruta absoluta del servidor.")
	}
	ptr, err := windows.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return result, err
	}
	access := uint32(windows.FILE_GENERIC_READ)
	// Open only the destination; the OS checks ancestor traversal with the real
	// effective token. Do not use EvalSymlinks/Lstat/FindFirstFile on ancestors.
	handle, err := windows.CreateFile(ptr, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return result, windowsAccessFailure(identity, source, "read", "CreateFileW(selected_directory)", path, access, err)
	}
	var info windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(handle, &info); err != nil {
		windows.CloseHandle(handle)
		return result, windowsAccessFailure(identity, source, "metadata", "GetFileInformationByHandle(selected_directory)", path, windows.FILE_READ_ATTRIBUTES, err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		windows.CloseHandle(handle)
		return result, invalid("Selecciona una carpeta legible.")
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(handle)
		return result, invalid("No se admite un punto de reanálisis sin resolver.")
	}
	canonical, err := finalWindowsDirectoryPath(handle)
	if err != nil {
		windows.CloseHandle(handle)
		return result, windowsAccessFailure(identity, source, "metadata", "GetFinalPathNameByHandleW(FILE_NAME_NORMALIZED)", path, windows.FILE_READ_ATTRIBUTES, err)
	}
	file := os.NewFile(uintptr(handle), canonical)
	defer file.Close()
	if _, err = file.ReadDir(128); err != nil && err != io.EOF {
		return result, windowsAccessFailure(identity, source, "enumeration", "GetFileInformationByHandleEx(directory_entries)", canonical, windows.FILE_LIST_DIRECTORY, err)
	}
	result = directoryInfo{Path: canonical, Identity: fmt.Sprintf("windows:%d:%d:%d:%d:%d", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime), CaseSensitive: true}
	var flags uint32
	if windows.GetFileInformationByHandleEx(handle, windows.FileCaseSensitiveInfo, (*byte)(unsafe.Pointer(&flags)), 4) == nil {
		result.CaseSensitive = flags&1 != 0
	}
	return result, nil
}

func checkWindowsAuthorizedDirectory(path, source string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	identity, err := effectiveWindowsIdentity()
	if err != nil {
		return windowsAccessFailure(identity, source, "identity", "effective_service_token", path, windows.TOKEN_QUERY, err)
	}
	inspected, err := inspectWindowsDirectoryAs(path, identity, source)
	if err != nil {
		return err
	}
	if source == "managed" {
		if err = probeWindowsManagedDirectory(inspected.Path, identity); err != nil {
			return err
		}
	}
	kind := "read"
	if source == "managed" {
		kind = "read/write/create/publish/delete_probe"
	}
	logWindowsAccess(identity, source, kind, "service_effective_access_complete", inspected.Path, windows.FILE_GENERIC_READ, nil)
	return nil
}
func probeWindowsManagedDirectory(path string, identity windowsAccessIdentity) error {
	fail := func(operation, resource, kind string, access uint32, err error) error {
		return windowsAccessFailure(identity, "managed", kind, operation, resource, access, err)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return fail("CreateFileW(managed_root)", path, "read", windows.FILE_GENERIC_READ, err)
	}
	defer root.Close()
	name := ".aidex-access-" + domain.NewID()
	directoryPath := filepath.Join(path, name)
	if err = root.Mkdir(name, 0700); err != nil {
		return fail("NtCreateFile(create_probe_directory)", directoryPath, "create_directory", windows.FILE_GENERIC_READ, err)
	}
	defer root.Remove(name)
	child, err := root.OpenRoot(name)
	if err != nil {
		return fail("NtCreateFile(open_probe_directory)", directoryPath, "read", windows.FILE_GENERIC_READ, err)
	}
	defer child.Close()
	probePath := filepath.Join(directoryPath, "probe")
	file, err := child.OpenFile("probe", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fail("NtCreateFile(create_probe_file)", probePath, "create_file/write", windows.FILE_GENERIC_WRITE|windows.FILE_READ_ATTRIBUTES|windows.FILE_READ_EA, err)
	}
	defer child.Remove("probe")
	defer child.Remove("published")
	_, writeErr := file.Write([]byte("AIDEX access probe"))
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil {
		return fail("WriteFile(probe)", probePath, "write", windows.FILE_WRITE_DATA, writeErr)
	}
	if syncErr != nil {
		return fail("FlushFileBuffers(probe)", probePath, "write", windows.FILE_WRITE_DATA, syncErr)
	}
	if closeErr != nil {
		return fail("CloseHandle(probe)", probePath, "close", 0, closeErr)
	}
	reader, err := child.Open("probe")
	if err != nil {
		return fail("NtCreateFile(open_probe_for_read)", probePath, "read", windows.FILE_GENERIC_READ, err)
	}
	read, err := io.ReadAll(reader)
	readCloseErr := reader.Close()
	if err != nil {
		return fail("ReadFile(probe)", probePath, "read", windows.FILE_READ_DATA, err)
	}
	if readCloseErr != nil {
		return fail("CloseHandle(read_probe)", probePath, "close", 0, readCloseErr)
	}
	if string(read) != "AIDEX access probe" {
		return domain.Failure("ROOT_PROBE_FAILED", "No se pudo verificar el contenido de la prueba de escritura.", 422)
	}
	directory, err := child.Open(".")
	if err != nil {
		return fail("NtCreateFile(probe_directory_handle)", directoryPath, "read", windows.FILE_GENERIC_READ, err)
	}
	directoryHandle := windows.Handle(directory.Fd())
	linkSource, err := child.OpenFile("probe", os.O_WRONLY, 0600)
	if err != nil {
		directory.Close()
		return fail("NtCreateFile(open_probe_for_link)", probePath, "metadata_write", windows.FILE_GENERIC_WRITE|windows.FILE_READ_ATTRIBUTES|windows.FILE_READ_EA, err)
	}
	linkErr := linkWindowsProbe(windows.Handle(linkSource.Fd()), directoryHandle, "published")
	linkSource.Close()
	if linkErr != nil {
		directory.Close()
		return fail("NtSetInformationFile(FileLinkInformation)", filepath.Join(directoryPath, "published"), "publish", windows.FILE_WRITE_ATTRIBUTES, linkErr)
	}
	for _, name := range []string{"published", "probe"} {
		if operation, err := deleteWindowsProbe(directoryHandle, name); err != nil {
			directory.Close()
			return fail(operation, filepath.Join(directoryPath, name), "delete_probe", windows.DELETE|windows.SYNCHRONIZE, err)
		}
	}
	directory.Close()
	if err = child.Close(); err != nil {
		return fail("CloseHandle(probe_directory)", directoryPath, "close", 0, err)
	}
	parent, err := root.Open(".")
	if err != nil {
		return fail("NtCreateFile(managed_root_handle)", path, "read", windows.FILE_GENERIC_READ, err)
	}
	operation, err := deleteWindowsProbe(windows.Handle(parent.Fd()), name)
	parent.Close()
	if err != nil {
		return fail(operation, directoryPath, "delete_probe", windows.DELETE|windows.SYNCHRONIZE, err)
	}
	return nil
}

// Both names are generated/constant single components inside an exclusive probe
// directory. Relative handles and OPEN_REPARSE_POINT prevent escaping that scope.
func linkWindowsProbe(file, parent windows.Handle, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `\/:`) {
		return windows.ERROR_INVALID_NAME
	}
	var info struct {
		ReplaceIfExists byte
		RootDirectory   windows.Handle
		FileNameLength  uint32
		FileName        [32]uint16
	}
	encoded, err := windows.UTF16FromString(name)
	if err != nil {
		return err
	}
	if len(encoded) > len(info.FileName) {
		return windows.ERROR_FILENAME_EXCED_RANGE
	}
	info.RootDirectory = parent
	info.FileNameLength = uint32((len(encoded) - 1) * 2)
	copy(info.FileName[:], encoded)
	return windows.NtSetInformationFile(file, &windows.IO_STATUS_BLOCK{}, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), 11)
}
func deleteWindowsProbe(parent windows.Handle, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `\/:`) {
		return "validate_probe_name", windows.ERROR_INVALID_NAME
	}
	encoded, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return "encode_probe_name", err
	}
	attributes := windows.OBJECT_ATTRIBUTES{RootDirectory: parent, ObjectName: encoded, Attributes: 0x40}
	attributes.Length = uint32(unsafe.Sizeof(attributes))
	var handle windows.Handle
	err = windows.NtCreateFile(&handle, windows.DELETE|windows.SYNCHRONIZE, &attributes, &windows.IO_STATUS_BLOCK{}, nil, windows.FILE_ATTRIBUTE_NORMAL, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN, windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if err != nil {
		return "NtCreateFile(open_probe_for_delete)", err
	}
	defer windows.CloseHandle(handle)
	disposition := byte(1)
	err = windows.NtSetInformationFile(handle, &windows.IO_STATUS_BLOCK{}, &disposition, 1, windows.FileDispositionInformation)
	return "NtSetInformationFile(FileDispositionInformation)", err
}
