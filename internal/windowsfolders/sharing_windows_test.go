package windowsfolders

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNativeMaximumAllowedSharingRegression(t *testing.T) {
	// Reproducer for the old ancestor open: an ordinary Go directory reader
	// shares READ|WRITE, while MAXIMUM_ALLOWED also asks for DELETE.
	path := t.TempDir()
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	legacy, err := openDirectory(path, windows.MAXIMUM_ALLOWED)
	if err == nil {
		windows.CloseHandle(legacy)
		t.Fatal("legacy MAXIMUM_ALLOWED unexpectedly shared a non-delete-shared directory")
	}
	if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("expected Win32 32, got %v", err)
	}
	var detail *OperationError
	if !errors.As(err, &detail) || detail.Operation != "CreateFileW(directory)" || detail.Resource != path || detail.Access != windows.MAXIMUM_ALLOWED {
		t.Fatal("missing exact operation/resource/access", err)
	}
	for _, access := range []uint32{windows.READ_CONTROL, windows.READ_CONTROL | windows.WRITE_DAC} {
		handle, e := openDirectory(path, access)
		if e != nil {
			t.Fatal("minimal security handle conflicts with the same reader", e)
		}
		windows.CloseHandle(handle)
	}
}

func TestNativePinsKeepDeleteSharingProtection(t *testing.T) {
	path := t.TempDir()
	pin, err := openDirectory(path, windows.READ_CONTROL)
	if err != nil {
		t.Fatal(err)
	}
	ptr, _ := windows.UTF16PtrFromString(path)
	deleting, err := windows.CreateFile(ptr, windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	windows.CloseHandle(pin)
	if err == nil {
		windows.CloseHandle(deleting)
		t.Fatal("pin allowed a competing delete/rename handle")
	}
	if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatal(err)
	}
	deleting, err = windows.CreateFile(ptr, windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		t.Fatal("fixture lacks delete permission", err)
	}
	defer windows.CloseHandle(deleting)
	if handle, err := openDirectory(path, windows.READ_CONTROL); err == nil {
		windows.CloseHandle(handle)
		t.Fatal("real conflicting delete handle was silently accepted")
	} else if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatal(err)
	}
}

func TestNativeACLWithReadersEmptyAndExistingDocuments(t *testing.T) {
	sid, err := windows.StringToSid("S-1-5-80-700001-700002-700003-700004-700005")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"linked", "managed"} {
		for _, withDocument := range []bool{false, true} {
			name := source + "/empty"
			if withDocument {
				name = source + "/documents"
			}
			t.Run(name, func(t *testing.T) {
				parent := t.TempDir()
				path := filepath.Join(parent, "Biblioteca José 東京")
				sibling := filepath.Join(parent, "Carpeta ajena")
				for _, dir := range []string{path, sibling} {
					if e := os.Mkdir(dir, 0700); e != nil {
						t.Fatal(e)
					}
				}
				original := filepath.Join(path, "original.txt")
				contents := []byte("documento original intacto")
				if withDocument {
					if e := os.WriteFile(original, contents, 0600); e != nil {
						t.Fatal(e)
					}
				}
				parentReader, e := os.Open(parent)
				if e != nil {
					t.Fatal(e)
				}
				defer parentReader.Close()
				pin, e := openDirectory(parent, windows.READ_CONTROL)
				if e != nil {
					t.Fatal(e)
				}
				defer windows.CloseHandle(pin)
				handle, e := openDirectory(path, windows.READ_CONTROL|windows.WRITE_DAC)
				if e != nil {
					t.Fatal(e)
				}
				defer windows.CloseHandle(handle)
				before, e := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
				if e != nil {
					t.Fatal(e)
				}
				oldOwner, _, e := before.Owner()
				if e != nil {
					t.Fatal(e)
				}
				oldACL, _, e := before.DACL()
				if e != nil {
					t.Fatal(e)
				}
				rootReader, e := os.Open(path)
				if e != nil {
					t.Fatal(e)
				}
				defer rootReader.Close()
				mask := windows.ACCESS_MASK(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_EXECUTE)
				if source == "managed" {
					mask |= windows.FILE_GENERIC_WRITE | windows.DELETE
				}
				var count uint16
				for attempt := 0; attempt < 3; attempt++ {
					if e = grantDirectory(handle, path, sid, mask, windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE, windows.SET_ACCESS, false); e != nil {
						t.Fatal(e)
					}
					descriptor, e := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
					if e != nil {
						t.Fatal(e)
					}
					owner, _, e := descriptor.Owner()
					if e != nil || !oldOwner.Equals(owner) {
						t.Fatal("owner changed", e)
					}
					acl, _, e := descriptor.DACL()
					if e != nil {
						t.Fatal(e)
					}
					assertACEPreserved(t, oldACL, acl)
					if attempt == 0 {
						count = acl.AceCount
						if count != oldACL.AceCount+1 {
							t.Fatal("unexpected ACE count", count, oldACL.AceCount)
						}
					} else if acl.AceCount != count {
						t.Fatal("repeated authorization duplicates ACEs")
					}
				}
				sub := filepath.Join(path, "Nueva subcarpeta")
				if e = os.Mkdir(sub, 0700); e != nil {
					t.Fatal(e)
				}
				child := filepath.Join(sub, "nuevo.txt")
				if e = os.WriteFile(child, contents, 0600); e != nil {
					t.Fatal(e)
				}
				if data, e := os.ReadFile(child); e != nil || string(data) != string(contents) {
					t.Fatal("original user lost access", e)
				}
				for _, item := range []string{sub, child} {
					descriptor, e := windows.GetNamedSecurityInfo(item, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
					if e != nil {
						t.Fatal(e)
					}
					acl, _, e := descriptor.DACL()
					if e != nil {
						t.Fatal(e)
					}
					found := false
					for index := uint32(0); index < uint32(acl.AceCount); index++ {
						var ace *windows.ACCESS_ALLOWED_ACE
						if e = windows.GetAce(acl, index, &ace); e != nil {
							t.Fatal(e)
						}
						if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && aceSid(ace).Equals(sid) {
							found = true
							if ace.Mask&mask != mask || ace.Header.AceFlags&windows.INHERITED_ACE == 0 || ace.Mask&(windows.WRITE_DAC|windows.WRITE_OWNER) != 0 {
								t.Fatal("incorrect service inheritance")
							}
							if source == "linked" && ace.Mask&(windows.FILE_WRITE_DATA|windows.FILE_APPEND_DATA|windows.DELETE) != 0 {
								t.Fatal("linked scope can write")
							}
						}
					}
					if !found {
						t.Fatal("service ACE not inherited", item)
					}
				}
				siblingBefore, e := windows.GetNamedSecurityInfo(sibling, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
				if e != nil {
					t.Fatal(e)
				}
				mutation, e := openDirectory(parent, windows.READ_CONTROL|windows.WRITE_DAC)
				if e != nil {
					t.Fatal(e)
				}
				defer windows.CloseHandle(mutation)
				parentBefore, e := windows.GetSecurityInfo(mutation, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
				if e != nil {
					t.Fatal(e)
				}
				controlBefore, _, e := parentBefore.Control()
				if e != nil {
					t.Fatal(e)
				}
				if e = grantDirectory(mutation, parent, sid, windows.FILE_TRAVERSE, 0, windows.GRANT_ACCESS, true); e != nil {
					t.Fatal(e)
				}
				siblingAfter, e := windows.GetNamedSecurityInfo(sibling, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
				if e != nil {
					t.Fatal(e)
				}
				if siblingBefore.String() != siblingAfter.String() {
					t.Fatal("ancestor change propagated to unrelated folder")
				}
				parentAfter, e := windows.GetSecurityInfo(mutation, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
				if e != nil {
					t.Fatal(e)
				}
				controlAfter, _, e := parentAfter.Control()
				if e != nil {
					t.Fatal(e)
				}
				if controlBefore&windows.SE_DACL_PROTECTED != controlAfter&windows.SE_DACL_PROTECTED {
					t.Fatal("ancestor inheritance protection changed")
				}
				if withDocument {
					data, e := os.ReadFile(original)
					if e != nil || string(data) != string(contents) {
						t.Fatal("original changed", e)
					}
				}
			})
		}
	}
}

func TestDiagnosticOperationResourceAndErrorCode(t *testing.T) {
	err := &OperationError{Operation: "CreateFileW(directory)", Resource: `C:\Archivo`, Access: windows.READ_CONTROL, Share: 3, Err: windows.ERROR_SHARING_VIOLATION}
	attrs := DiagnosticAttrs(err)
	data := map[string]any{}
	for i := 0; i < len(attrs); i += 2 {
		data[attrs[i].(string)] = attrs[i+1]
	}
	if data["operation"] != err.Operation || data["resource"] != err.Resource || data["win32_error"] != uint32(32) || !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatal(attrs)
	}
	token := strings.Repeat("a", 64)
	httpError := &url.Error{Op: "Post", URL: "http://127.0.0.1:18090/" + token, Err: windows.ERROR_ACCESS_DENIED}
	for _, attr := range DiagnosticAttrs(operationError("service_probe_http", "servicio local AIDEX", httpError)) {
		if value, ok := attr.(string); ok && strings.Contains(value, token) {
			t.Fatal("ticket leaked into diagnostics")
		}
	}
}

func aceSid(ace *windows.ACCESS_ALLOWED_ACE) *windows.SID {
	return (*windows.SID)(unsafe.Pointer(&ace.SidStart))
}

func assertACEPreserved(t *testing.T, before, after *windows.ACL) {
	t.Helper()
	counts := func(acl *windows.ACL) map[string]int {
		result := map[string]int{}
		for index := uint32(0); index < uint32(acl.AceCount); index++ {
			var ace *windows.ACCESS_ALLOWED_ACE
			if err := windows.GetAce(acl, index, &ace); err != nil {
				t.Fatal(err)
			}
			raw := unsafe.Slice((*byte)(unsafe.Pointer(ace)), int(ace.Header.AceSize))
			result[string(raw)]++
		}
		return result
	}
	original, current := counts(before), counts(after)
	for ace, count := range original {
		if current[ace] < count {
			t.Fatal("an existing user's ACE was removed or changed")
		}
	}
}
