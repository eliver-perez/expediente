package libraries

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"gestor-documental/internal/domain"
)

func TestDirectoryFailureClassifiesWithoutDisclosingPath(t *testing.T) {
	for _, item := range []struct {
		path, code string
		err        error
	}{
		{"/private/sensitive/folder", "ROOT_NOT_FOUND", os.ErrNotExist},
		{"/private/sensitive/folder", map[bool]string{true: "ROOT_PRIVACY_DENIED", false: "ROOT_PERMISSION_DENIED"}[runtime.GOOS == "darwin"], syscall.EPERM},
		{`\\server\share\sensitive`, "ROOT_PERMISSION_DENIED", os.ErrPermission},
		{`\\server\share\sensitive`, "ROOT_NETWORK_UNAVAILABLE", fmt.Errorf("network offline")},
	} {
		failure := directoryFailure(item.path, item.err)
		var detail *domain.Error
		if !errors.As(failure, &detail) || failureCode(failure) != item.code || strings.Contains(detail.Message, "sensitive") {
			t.Fatalf("%s: %v", item.code, failure)
		}
		if runtime.GOOS == "darwin" && item.err == syscall.EPERM && !strings.Contains(detail.Message, "Privacidad") {
			t.Fatal("macOS privacy instructions missing", failure)
		}
	}
}
