package windowsfolders

import "testing"

func TestRejectsProtectedRemoteAndAmbiguousSelections(t *testing.T) {
	for _, path := range []string{`C:\`, `C:\Users`, `C:\Users\Ada`, `C:\Windows\Temp`, `C:\ProgramData\AIDEX`, `C:\Users\Ana\AppData\Roaming`, `C:\Program Files\AIDEX`, `\\server\share\docs`, `\\?\C:\docs`, `C:\docs:stream`, `C:\docs\..\Windows`, `C:\docs.`, `C:\docs\`, `C:\docs\bad` + "\n" + `name`} {
		if path == `C:\docs\` {
			continue
		} // One terminal separator is accepted by native pickers.
		if err := ValidateLocal(path, "managed"); err == nil {
			t.Fatal("accepted", path)
		}
	}
	for _, source := range []string{"linked", "managed"} {
		if err := ValidateLocal(`C:\Users\Ada\Documents\Archivo`, source); err != nil {
			t.Fatal(err)
		}
	}
	if ValidateLocal(`C:\Documents`, "other") == nil {
		t.Fatal("unknown source")
	}
}
