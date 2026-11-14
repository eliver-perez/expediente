//go:build development && !windows

package libraries

import (
	"os"
	"testing"
)

func withDeniedDirectory(t *testing.T, path string, mode os.FileMode, run func()) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(path, info.Mode().Perm()); err != nil {
			t.Error(err)
		}
	}()
	run()
}
