//go:build development && windows

package libraries

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// chmod does not remove access on NTFS. Exercise the same operation through
// a restricted thread token while leaving the test owner able to clean up.
func withDeniedDirectory(t *testing.T, path string, mode os.FileMode, run func()) {
	t.Helper()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	restrictedSID, err := windows.StringToSid("S-1-5-80-800021-800022-800023-800024-800025")
	if err != nil {
		t.Fatal(err)
	}
	rights := "0x1200a9" // read, list and traverse; no write or delete
	if mode == 0 {
		rights = "0x20" // traverse only
	}
	nativeProbeACL(t, path, user.User.Sid.String(), restrictedSID.String(), rights)
	asRestrictedToken(t, nativeRestrictedToken(t, restrictedSID), run)
}
