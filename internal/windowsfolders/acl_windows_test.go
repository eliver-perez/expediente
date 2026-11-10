package windowsfolders

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

func TestNativeGrantPreservesOwnerUserAndInheritsMinimalServiceAccess(t *testing.T) {
	// Runs only on Windows, on disposable directories owned by the test account.
	// No installed service, elevation or document directory is modified.
	for _, managed := range []bool{false, true} {
		directory := t.TempDir()
		before, err := windows.GetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		owner, _, err := before.Owner()
		if err != nil {
			t.Fatal(err)
		}
		sid, err := windows.StringToSid("S-1-5-80-700001-700002-700003-700004-700005")
		if err != nil {
			t.Fatal(err)
		}
		handle, err := openDirectory(directory, windows.READ_CONTROL|windows.WRITE_DAC)
		if err != nil {
			t.Fatal(err)
		}
		mask := windows.ACCESS_MASK(windows.FILE_GENERIC_READ | windows.FILE_GENERIC_EXECUTE)
		if managed {
			mask |= windows.FILE_GENERIC_WRITE | windows.DELETE
		}
		err = grant(handle, sid, mask, windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE, windows.SET_ACCESS)
		windows.CloseHandle(handle)
		if err != nil {
			t.Fatal(err)
		}
		after, err := windows.GetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		newOwner, _, err := after.Owner()
		if err != nil || !owner.Equals(newOwner) {
			t.Fatal("owner changed", err)
		}
		oldACL, _, err := before.DACL()
		if err != nil {
			t.Fatal(err)
		}
		newACL, _, err := after.DACL()
		if err != nil {
			t.Fatal(err)
		}
		if newACL.AceCount < oldACL.AceCount+1 {
			t.Fatal("existing ACL replaced or service ACE missing")
		}
		// Real creation and read with the original account prove it retained access.
		child := filepath.Join(directory, "new-document.txt")
		if err = os.WriteFile(child, []byte("original user retains access"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = os.ReadFile(child); err != nil {
			t.Fatal(err)
		}
		inherited, err := windows.GetNamedSecurityInfo(child, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		acl, _, err := inherited.DACL()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for index := uint32(0); index < uint32(acl.AceCount); index++ {
			var ace *windows.ACCESS_ALLOWED_ACE
			if err = windows.GetAce(acl, index, &ace); err != nil {
				t.Fatal(err)
			}
			aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
			if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && aceSID.Equals(sid) {
				found = true
				if ace.Header.AceFlags&windows.INHERITED_ACE == 0 || ace.Mask&mask != mask || ace.Mask&(windows.WRITE_DAC|windows.WRITE_OWNER) != 0 {
					t.Fatal("incorrect inherited service rights")
				}
				if !managed && ace.Mask&(windows.FILE_WRITE_DATA|windows.FILE_APPEND_DATA|windows.DELETE) != 0 {
					t.Fatal("linked scope granted writes")
				}
			}
		}
		if !found {
			t.Fatal("new document lacks inherited service access")
		}
	}
}
