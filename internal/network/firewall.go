package network

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Firewall struct {
	State        string   `json:"state"`
	Message      string   `json:"message"`
	Checks       []string `json:"checks"`
	Instructions string   `json:"instructions"`
}

func unknownFirewall() Firewall {
	return Firewall{State: "unknown", Message: "No fue posible verificar automáticamente si el firewall permite conexiones a AIBID.", Checks: []string{}}
}

// Diagnostics are strictly read-only. The service never elevates privileges,
// changes global policy or creates firewall rules that an uninstaller must guess.
// Command output stays bounded and is never returned raw to the browser.
type limitedOutput struct{ bytes.Buffer }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 32768 {
		return 0, fmt.Errorf("diagnostic output too large")
	}
	return b.Buffer.Write(p)
}

func commandPath(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	for _, directory := range []string{"/usr/sbin", "/sbin", "/usr/bin", "/bin"} {
		path := filepath.Join(directory, name)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			return path
		}
	}
	return ""
}

func probe(ctx context.Context, path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	command.Env = append(os.Environ(), "LC_ALL=C")
	command.WaitDelay = time.Second
	var output limitedOutput
	command.Stdout = &output
	err := command.Run()
	return output.String(), err
}
