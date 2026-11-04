//go:build darwin && cgo

package macosfolders

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func aclText(t *testing.T, path string) string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output, err := exec.CommandContext(ctx, "/bin/ls", "-led", path).CombinedOutput()
	if err != nil {
		t.Fatal(string(output), err)
	}
	return string(output)
}

func TestNativeACLInheritanceAndIdempotenceWithoutPublicPermissions(t *testing.T) {
	// Solo se modifican archivos propios de esta fixture. Nunca se llama a la
	// autorización de producción ni se alteran /Users, /Library o sus superiores.
	directory := t.TempDir()
	root, err := os.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	nobody, err := user.Lookup("nobody")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := strconv.Atoi(nobody.Uid)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err = allow(root, uid, 2, 2); err != nil {
			t.Fatal(err)
		}
		if err = allow(root, uid, 1, 7); err != nil {
			t.Fatal(err)
		}
	}
	if entries := strings.Count(aclText(t, directory), " allow "); entries != 2 {
		t.Fatal("ACL acumuladas", entries, aclText(t, directory))
	}
	sub := filepath.Join(directory, "subcarpeta")
	if err = os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "Documento á 東京.txt")
	if err = os.WriteFile(file, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	acl := aclText(t, file)
	if !strings.Contains(acl, "inherited allow read") || strings.Contains(acl, "execute") {
		t.Fatal("herencia incorrecta en archivo", acl)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatal("se abrieron permisos públicos", info.Mode())
	}
	contents, err := os.ReadFile(file)
	if err != nil || string(contents) != "original" {
		t.Fatal("original alterado", err)
	}
}

func TestNativeAuthorizationRejectsLinksBeforeChangingPermissions(t *testing.T) {
	for _, hard := range []bool{false, true} {
		t.Run(map[bool]string{false: "simbolico", true: "duro"}[hard], func(t *testing.T) {
			directory, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			external := filepath.Join(t.TempDir(), "original.txt")
			if err = os.WriteFile(external, []byte("externo"), 0600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(directory, "enlace")
			if hard {
				err = os.Link(external, link)
			} else {
				err = os.Symlink(external, link)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := aclText(t, external)
			if count, err := authorizeTree(context.Background(), Selection{Path: directory, Source: "managed"}, os.Getuid(), os.Getuid(), nil); err == nil || count != 0 {
				t.Fatal("se aceptó un enlace", count, err)
			}
			if after := aclText(t, external); after != before {
				t.Fatal("cambió el destino externo")
			}
		})
	}
}

func TestNativeManagedACLRetainsExistingPermissionsAndInheritance(t *testing.T) {
	// La fixture es del usuario actual. No se cambian usuarios ni rutas del sistema.
	path := t.TempDir()
	root, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	account, err := user.Lookup("nobody")
	if err != nil {
		t.Fatal(err)
	}
	serviceUID, err := strconv.Atoi(account.Uid)
	if err != nil {
		t.Fatal(err)
	}
	// Una autorización explícita anterior debe sobrevivir sin ser reemplazada.
	if err = allow(root, os.Getuid(), 2, 0); err != nil {
		t.Fatal(err)
	}
	before := strings.Split(strings.TrimSpace(aclText(t, path)), "\n")
	if len(before) < 2 {
		t.Fatal("no se creó el permiso previo")
	}
	for i := 0; i < 3; i++ {
		if err = allow(root, serviceUID, 4, 2); err != nil {
			t.Fatal(err)
		}
		if err = allow(root, serviceUID, 3, 7); err != nil {
			t.Fatal(err)
		}
	}
	after := aclText(t, path)
	if !strings.Contains(after, before[1]) || strings.Count(after, " allow ") != 3 {
		t.Fatal("se alteraron o duplicaron permisos previos", after)
	}
	sub := filepath.Join(path, "Nueva subcarpeta á")
	if err = os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "documento.txt")
	if err = os.WriteFile(file, []byte("prueba nativa"), 0600); err != nil {
		t.Fatal(err)
	}
	permissions := aclText(t, file)
	if !strings.Contains(permissions, "user:nobody inherited allow read,write") || !strings.Contains(permissions, "append") || strings.Contains(permissions, "execute") {
		t.Fatal("herencia administrada insuficiente o excesiva", permissions)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatal("permisos públicos inesperados", info.Mode())
	}
	if bytes, err := os.ReadFile(file); err != nil || string(bytes) != "prueba nativa" {
		t.Fatal("contenido alterado", err)
	}
	if err = os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(sub); err != nil {
		t.Fatal(err)
	}
}
