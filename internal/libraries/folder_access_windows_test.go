package libraries

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"gestor-documental/internal/buildinfo"
	"golang.org/x/sys/windows"
)

var testAdvapi = windows.NewLazySystemDLL("advapi32.dll")

func nativeRestrictedToken(t *testing.T, sid *windows.SID) windows.Token {
	t.Helper()
	var base, restricted windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &base); err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	restricting := windows.SIDAndAttributes{Sid: sid}
	ok, _, err := testAdvapi.NewProc("CreateRestrictedToken").Call(uintptr(base), 1, 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&restricting)), uintptr(unsafe.Pointer(&restricted)))
	runtime.KeepAlive(sid)
	if ok == 0 {
		t.Fatal("CreateRestrictedToken", err)
	}
	t.Cleanup(func() { restricted.Close() })
	return restricted
}
func asRestrictedToken(t *testing.T, token windows.Token, check func()) {
	t.Helper()
	runtime.LockOSThread()
	ok, _, err := testAdvapi.NewProc("ImpersonateLoggedOnUser").Call(uintptr(token))
	if ok == 0 {
		runtime.UnlockOSThread()
		t.Fatal("ImpersonateLoggedOnUser", err)
	}
	defer func() {
		ok, _, err := testAdvapi.NewProc("RevertToSelf").Call()
		if ok == 0 {
			t.Errorf("RevertToSelf: %v; locked thread must not be reused", err)
			return
		}
		runtime.UnlockOSThread()
	}()
	check()
}
func nativeProbeACL(t *testing.T, path, user, restricted, rights string) {
	t.Helper()
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user + ")(A;OICI;FA;;;SY)(A;OICI;" + rights + ";;;" + restricted + ")")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
}
func TestNativeWindowsTraversalOnlyAncestors(t *testing.T) {
	if buildinfo.ServiceName != "" {
		t.Skip("test fixture must not impersonate the installed service")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	restrictedSID, err := windows.StringToSid("S-1-5-80-800001-800002-800003-800004-800005")
	if err != nil {
		t.Fatal(err)
	}
	token := nativeRestrictedToken(t, restrictedSID)
	for _, mode := range []string{"linked", "managed"} {
		for _, documents := range []bool{false, true} {
			name := mode + "/empty"
			if documents {
				name = mode + "/documents"
			}
			t.Run(name, func(t *testing.T) {
				parent := filepath.Join(t.TempDir(), "solo recorrido")
				// The physical name intentionally does not imply its storage mode.
				selected := filepath.Join(parent, "Administrada José 東京")
				if err := os.MkdirAll(selected, 0700); err != nil {
					t.Fatal(err)
				}
				nativeProbeACL(t, parent, user.User.Sid.String(), restrictedSID.String(), "0x20") // traverse only; no enumeration or attributes
				rights := "0x1200a9"
				if mode == "managed" {
					rights = "0x1301bf"
				}
				nativeProbeACL(t, selected, user.User.Sid.String(), restrictedSID.String(), rights)
				original := filepath.Join(selected, "original.txt")
				contents := []byte("original sin modificaciones")
				if documents {
					if err := os.WriteFile(original, contents, 0600); err != nil {
						t.Fatal(err)
					}
				}
				var canonical string
				asRestrictedToken(t, token, func() {
					identity, err := effectiveWindowsIdentity()
					if err != nil {
						t.Fatal(err)
					}
					if identity.TokenSource != "thread" || !identity.Restricted || identity.SID != user.User.Sid.String() {
						t.Fatal("wrong effective token", identity)
					}
					ptr, _ := windows.UTF16PtrFromString(parent)
					denied, err := windows.CreateFile(ptr, windows.FILE_LIST_DIRECTORY, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
					if err == nil {
						windows.CloseHandle(denied)
						t.Fatal("fixture unexpectedly allows ancestor enumeration")
					}
					if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
						t.Fatal(err)
					}
					if _, err = filepath.EvalSymlinks(selected); err == nil {
						t.Fatal("legacy canonicalizer unexpectedly succeeded; regression not demonstrated")
					}
					inspected, err := inspectDirectory(selected)
					if err != nil {
						t.Fatal("handle inspector failed with traverse-only ancestors", err)
					}
					canonical = inspected.Path
					if err = CheckAuthorizedDirectory(selected, mode); err != nil {
						t.Fatal("effective service-style probe", err)
					}
					if mode == "linked" {
						file, err := os.OpenFile(filepath.Join(selected, "forbidden-write.txt"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
						if err == nil {
							file.Close()
							t.Fatal("linked fixture unexpectedly allows writes")
						}
						if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
							t.Fatal(err)
						}
					}
					if documents {
						read, err := os.ReadFile(original)
						if err != nil || !bytes.Equal(read, contents) {
							t.Fatal("existing document inaccessible or altered", err)
						}
					}
				})
				if canonical == "" || !filepath.IsAbs(canonical) {
					t.Fatal("missing canonical path")
				}
				entries, err := os.ReadDir(selected)
				if err != nil {
					t.Fatal(err)
				}
				expected := 0
				if documents {
					expected = 1
				}
				if len(entries) != expected {
					t.Fatal("probe left files", entries)
				}
				if documents {
					read, err := os.ReadFile(original)
					if err != nil || !bytes.Equal(read, contents) {
						t.Fatal("owner access/content changed", err)
					}
				}
			})
		}
	}
}
func TestNativeWindowsEffectiveFailureRecordsOriginalCall(t *testing.T) {
	if buildinfo.ServiceName != "" {
		t.Skip("fixture only")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid, err := windows.StringToSid("S-1-5-80-800011-800012-800013-800014-800015")
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir()
	nativeProbeACL(t, path, user.User.Sid.String(), sid.String(), "0x20")
	token := nativeRestrictedToken(t, sid)
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(previous)
	asRestrictedToken(t, token, func() {
		if err := CheckAuthorizedDirectory(path, "linked"); failureCode(err) != "ROOT_PERMISSION_DENIED" {
			t.Fatal(err)
		}
	})
	found := false
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if row["operation"] == "CreateFileW(selected_directory)" {
			found = true
			if row["resource"] != path || row["win32_error"] != float64(5) || row["effective_sid"] != user.User.Sid.String() || row["token_source"] != "thread" || row["access_kind"] != "read" || row["desired_access"] != "0x00120089" {
				t.Fatal(row)
			}
		}
	}
	if !found {
		t.Fatal("native failure was reduced to a domain code", output.String())
	}
}
