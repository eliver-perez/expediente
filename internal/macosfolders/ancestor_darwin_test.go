//go:build darwin && cgo

package macosfolders

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestNativeAuthorizationSkipsAlreadyTraversableAncestors(t *testing.T) {
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(path, "original.txt")
	if err = os.WriteFile(original, []byte("original intacto"), 0600); err != nil {
		t.Fatal(err)
	}
	ancestors := map[string]string{}
	for parent := filepath.Dir(path); parent != "/"; parent = filepath.Dir(parent) {
		ancestors[parent] = aclText(t, parent)
	}
	initial, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	uid := os.Getuid()
	checks := 0
	probe := func(ctx context.Context, targetUID int, parent string) (bool, error) {
		if targetUID != uid {
			t.Fatal("la prueba no puede suplantar cuentas")
		}
		checks++
		// Consulta nativa efectiva de la cuenta de la fixture, no permisos simulados.
		if err := syscall.Access(parent, 1); err != nil {
			return false, err
		}
		return true, nil
	}
	for attempt := 0; attempt < 2; attempt++ {
		count, err := authorizeTree(context.Background(), Selection{Path: path, Source: "managed"}, uid, uid, probe)
		if err != nil || count != 2 {
			t.Fatal(count, err)
		}
	}
	if checks == 0 {
		t.Fatal("no se verificó recorrido")
	}
	for parent, before := range ancestors {
		if aclText(t, parent) != before {
			t.Fatal("se modificó un superior innecesariamente", parent)
		}
	}
	if data, err := os.ReadFile(original); err != nil || string(data) != "original intacto" {
		t.Fatal("original alterado", err)
	}
	before := aclText(t, path)
	if strings.Count(before, " allow ") != 2 {
		t.Fatal("ACL duplicadas", before)
	}
	final, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Mode().Perm() != final.Mode().Perm() || initial.Sys().(*syscall.Stat_t).Uid != final.Sys().(*syscall.Stat_t).Uid {
		t.Fatal("cambió el propietario o modo POSIX")
	}
	child := filepath.Join(path, "Subcarpeta nueva")
	if err = os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	created := filepath.Join(child, "Documento nuevo.txt")
	if err = os.WriteFile(created, []byte("nuevo"), 0600); err != nil {
		t.Fatal(err)
	}
	permissions := aclText(t, created)
	if !strings.Contains(permissions, "inherited allow read,write") || !strings.Contains(permissions, "inherited allow read,") {
		t.Fatal("faltan permisos heredados", permissions)
	}
}

func TestAncestorProbeFailureStopsWithoutChangingPermissions(t *testing.T) {
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := aclText(t, path)
	_, err = authorizeTree(context.Background(), Selection{Path: path, Source: "managed"}, os.Getuid(), os.Getuid(), func(context.Context, int, string) (bool, error) { return false, syscall.EPERM })
	var failure *OperationError
	if !errors.As(err, &failure) || failure.Operation != "check_ancestor_traverse" || !errors.Is(err, syscall.EPERM) {
		t.Fatal("se ocultó un fallo de comprobación", err)
	}
	if aclText(t, path) != before {
		t.Fatal("se modificaron permisos tras fallar la comprobación")
	}
}

func TestNativeFailureRecordsExactOperationPathIdentityAndAttributes(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "diagnostico-")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	file.Close()
	var captured bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&captured, nil)))
	defer slog.SetDefault(previous)
	err = allow(file, os.Getuid(), 3, 0)
	var failure *OperationError
	if !errors.As(err, &failure) || failure.Operation != "acl_get_fd_np" || !errors.Is(err, syscall.EBADF) {
		t.Fatal("operación nativa no identificada", err)
	}
	var event map[string]any
	for _, line := range strings.Split(strings.TrimSpace(captured.String()), "\n") {
		if err = json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err, captured.String())
		}
		if _, exists := event["owner_uid"]; exists {
			break
		}
	}
	for _, field := range []string{"operation", "path", "errno", "real_uid", "effective_uid", "effective_gid", "owner_uid", "owner_gid", "file_flags", "inode", "device"} {
		if _, ok := event[field]; !ok {
			t.Fatal("falta diagnóstico", field, event)
		}
	}
	if event["path"] != path || event["errno"] != float64(syscall.EBADF) {
		t.Fatal(event)
	}
	if strings.Contains(failure.Error(), path) || strings.Contains(failure.Error(), "acl_") {
		t.Fatal("detalle técnico expuesto al usuario", failure.Error())
	}
}

func TestNativeNewDocumentsFolderWithoutManualPermissions(t *testing.T) {
	if os.Getenv("AIDEX_MACOS_DOCUMENTS_QA") != "1" {
		t.Skip("prueba nativa opcional en Documentos; requiere autorización explícita del entorno")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(filepath.Join(home, "Documents"), "AIDEX-QA-permisos-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory) // Exclusivamente esta carpeta sintética recién creada.
	dir, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	account, err := user.Lookup("_aidex")
	if err != nil {
		t.Fatal(err)
	}
	serviceUID, err := strconv.Atoi(account.Uid)
	if err != nil {
		t.Fatal(err)
	}
	before, err := dir.Stat()
	if err != nil {
		t.Fatal(err)
	}
	// Misma API nativa y misma construcción de ACL que producción. La cuenta
	// propietaria aplica permisos, sin chmod previo ni suplantar aquí a _aidex.
	for attempt := 0; attempt < 2; attempt++ {
		for _, grant := range []struct{ uid, mode, flags int }{{serviceUID, 4, 2}, {serviceUID, 3, 7}, {os.Getuid(), 2, 2}, {os.Getuid(), 1, 7}} {
			if err = allow(dir, grant.uid, grant.mode, grant.flags); err != nil {
				t.Fatal(err)
			}
		}
	}
	if strings.Count(aclText(t, directory), " allow ") != 4 {
		t.Fatal("ACL acumuladas")
	}
	sub := filepath.Join(directory, "Subcarpeta nueva")
	if err = os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sub, "Documento nuevo.txt")
	if err = os.WriteFile(path, []byte("documento sintético"), 0600); err != nil {
		t.Fatal(err)
	}
	acl := aclText(t, path)
	if !strings.Contains(acl, "user:_aidex inherited allow read,write") || !strings.Contains(acl, "user:"+os.Getenv("USER")+" inherited allow read,") {
		t.Fatal("herencia de servicio/Finder incorrecta", acl)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "documento sintético" {
		t.Fatal(err)
	}
	after, err := dir.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode().Perm() != after.Mode().Perm() || before.Sys().(*syscall.Stat_t).Uid != after.Sys().(*syscall.Stat_t).Uid {
		t.Fatal("cambió propietario o permisos POSIX")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(sub); err != nil {
		t.Fatal(err)
	}
	t.Log("ACL nativas e herencia verificadas en carpeta nueva de Documentos; falta ejecución efectiva como _aidex y flujo gráfico elevado")
}

func TestNativeLinkedTreeAddsReadOnlyPermissionsAndIsIdempotent(t *testing.T) {
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(path, "original.txt")
	if err = os.WriteFile(original, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	uid := os.Getuid()
	probe := func(_ context.Context, identity int, parent string) (bool, error) {
		if identity != uid {
			t.Fatal("identidad ajena en fixture")
		}
		err := syscall.Access(parent, 1)
		return err == nil, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err = authorizeTree(context.Background(), Selection{Path: path, Source: "linked"}, uid, uid, probe); err != nil {
			t.Fatal(err)
		}
	}
	acl := aclText(t, original)
	if strings.Count(acl, " allow ") != 1 || !strings.Contains(acl, "allow read,") {
		t.Fatal("permisos vinculados incorrectos", acl)
	}
	for _, permission := range []string{"write", "append", "delete", "execute"} {
		if strings.Contains(acl, permission) {
			t.Fatal("se concedió un permiso excesivo", permission, acl)
		}
	}
	if data, err := os.ReadFile(original); err != nil || string(data) != "original" {
		t.Fatal("original alterado", err)
	}
}
