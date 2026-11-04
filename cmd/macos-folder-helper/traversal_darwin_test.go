//go:build darwin && cgo

package main

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Los subprocesos de comprobación ejecutan run() real, no el runner de Go.
func TestMain(m *testing.M) {
	if len(os.Args) == 3 && (os.Args[1] == "check-traverse" || os.Args[1] == "check-read") {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestNativeTraversalProbeUsesEffectiveIdentityAndDenial(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("fixture de cuenta no privilegiada")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := probeTraversal(ctx, os.Getuid(), path)
	if err != nil || !allowed {
		t.Fatal("recorrido propio rechazado", allowed, err)
	}
	if err = os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0700)
	allowed, err = probeTraversal(ctx, os.Getuid(), path)
	if err != nil || allowed {
		t.Fatal("no se detectó la denegación real", allowed, err)
	}
	if _, err = probeTraversal(ctx, os.Getuid()+1, path); err == nil {
		t.Fatal("se infirieron permisos de una cuenta que no se pudo comprobar")
	}
}

func TestPrivateAuthorizationLogRejectsLinksAndIsBounded(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file, err := openAuthorizationLog(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("registro sintético\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()
	path := filepath.Join(directory, "folder-authorization.log")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 || info.Sys().(*syscall.Stat_t).Uid != uint32(os.Geteuid()) {
		t.Fatal("registro sin privacidad", info)
	}
	if err = os.Truncate(path, (1<<20)+1); err != nil {
		t.Fatal(err)
	}
	file, err = openAuthorizationLog(directory)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	info, err = os.Stat(path)
	if err != nil || info.Size() != 0 {
		t.Fatal("registro sin límite", info, err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "original.txt")
	if err = os.WriteFile(external, []byte("intacto"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(external, path); err != nil {
		t.Fatal(err)
	}
	if file, err = openAuthorizationLog(directory); err == nil {
		file.Close()
		t.Fatal("se siguió un enlace")
	}
	if data, err := os.ReadFile(external); err != nil || string(data) != "intacto" {
		t.Fatal("se alteró un original", err)
	}
}
