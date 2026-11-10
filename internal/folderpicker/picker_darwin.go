package folderpicker

import (
	"context"
	"os/exec"
	"strings"
)

func Choose(ctx context.Context) (string, error) {
	// Fixed AppleScript: no browser-provided paths or commands enter the script.
	output, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", `try
 return POSIX path of (choose folder with prompt "Selecciona una carpeta para AIDEX")
 on error number -128
 return ""
 end try`).Output()
	return strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "/"), err
}
