//go:build darwin && cgo

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNativeProcessExecutableAndAccount(t *testing.T) {
	pid := strconv.Itoa(os.Getpid())
	info, err := nativeProcessIdentity(pid)
	if err != nil {
		t.Fatal(err)
	}
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if !matchesServiceProcess(info, path, uint32(os.Geteuid())) {
		t.Fatal("identidad propia rechazada", info, path)
	}
	if matchesServiceProcess(info, path+"-ajeno", info.UID) || matchesServiceProcess(info, path, info.UID+1) {
		t.Fatal("se aceptó ejecutable o cuenta ajenos")
	}
	if _, err = nativeProcessIdentity("2147483647"); err == nil {
		t.Fatal("proceso inexistente aceptado")
	}
}

func TestNativeInstalledExecutableProtection(t *testing.T) {
	const path = "/Library/Application Support/AIDEX/bin/aidex-server"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("no hay instalación nativa; no se crea ninguna")
	}
	if err := trustedExecutable(path); err != nil {
		t.Fatal(err)
	}
	// Solo se consulta la instalación real. El enlace es temporal y apunta fuera.
	link := filepath.Join(t.TempDir(), "aidex-server")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := trustedExecutable(link); err == nil {
		t.Fatal("se confió en un enlace sustituible")
	}
}

func nativeListeners(t *testing.T, ctx context.Context, port, format string) string {
	t.Helper()
	out, err := exec.CommandContext(ctx, "/usr/sbin/lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN", format).CombinedOutput()
	if err != nil {
		t.Fatal("lsof", err, string(out))
	}
	return string(out)
}

func TestNativeLsofFalseRejectionAndRealPortConflict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	pid := strconv.Itoa(os.Getpid())
	legacy := nativeListeners(t, ctx, port, "-Fp")
	legacyRejected := false
	for _, line := range strings.Split(strings.TrimSpace(legacy), "\n") {
		if line != "p"+pid {
			legacyRejected = true
		}
	}
	if !legacyRejected || !strings.Contains(legacy, "\nf") {
		t.Fatal("no se reprodujo el descriptor adicional real de macOS", legacy)
	}
	fixed := nativeListeners(t, ctx, port, "-t")
	if !listenerMatches(fixed, pid) {
		t.Fatal("se rechazó el listener nativo válido", fixed)
	}
	t.Logf("Reproducción nativa: -Fp=%q; -t=%q", legacy, fixed)
	// Otro proceso usa el mismo puerto TCP en IPv6. No se cambia ninguna interfaz
	// ni se filtra lsof por PID: el conflicto auténtico debe seguir rechazándose.
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeListenerChild$")
	child.Env = append(os.Environ(), "AIDEX_TEST_LISTENER_PORT="+port)
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			ready <- scanner.Text()
		} else {
			ready <- ""
		}
	}()
	select {
	case line := <-ready:
		if line != "ready" {
			t.Fatal("no se pudo crear el listener IPv6 temporal", line)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	conflict := nativeListeners(t, ctx, port, "-t")
	if listenerMatches(conflict, pid) {
		t.Fatal("se aceptó un listener de otro proceso", conflict)
	}
	if !strings.Contains(conflict, strconv.Itoa(child.Process.Pid)) {
		t.Fatal("lsof no observó el proceso ajeno", conflict)
	}
	t.Logf("Conflicto nativo rechazado: %q", conflict)
}

func TestNativeListenerChild(t *testing.T) {
	port := os.Getenv("AIDEX_TEST_LISTENER_PORT")
	if port == "" {
		return
	}
	listener, err := net.Listen("tcp6", "[::1]:"+port)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("ready")
	// El padre lo termina; este límite impide dejar un proceso indefinido.
	select {
	case <-time.After(12 * time.Second):
	}
}
