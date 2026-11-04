package maintenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestor-documental/internal/storage"
)

func recoveryFixture(t *testing.T) (string, func(string, string)) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(base, "AIDEX", "data")
	write := func(name, value string) {
		t.Helper()
		path := filepath.Join(data, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("logs/service.log", "registro sintético anterior")
	write("state/service.lock", "")
	return filepath.Join(data, "config.json"), write
}

func TestMacOSInstallerRecoveryLegacyResidues(t *testing.T) {
	path, write := recoveryFixture(t)
	write("logs/service.log.1.gz", "registro rotado")
	write("package-version", "1.0.0\n")
	write("state/service-enabled", "")
	removed, err := RecoverMacOSInstallerResidues(path)
	if err != nil || len(removed) != 5 {
		t.Fatalf("residuos=%v error=%v", removed, err)
	}
	for _, name := range removed {
		if _, err := os.Lstat(filepath.Join(filepath.Dir(path), name)); !os.IsNotExist(err) {
			t.Fatalf("residuo conservado: %s", name)
		}
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("la recuperación no debe crear configuración")
	}
	// Idempotencia: los directorios vacíos son utilizables por init.
	if removed, err = RecoverMacOSInstallerResidues(path); err != nil || len(removed) != 0 {
		t.Fatalf("segunda recuperación: %v %v", removed, err)
	}
}

func TestMacOSInstallerRecoveryProtectsRecoverableData(t *testing.T) {
	for _, name := range []string{"state/documental.db", "state/documental.db-wal", "state/documental.db-shm", "respaldo.sqlite3", "state/license/installation.json", "state/license/recovery.json", "config.json.before-license-20261029", "state/uploads/original.pdf", "logs/desconocido.txt", "state/service.lock", "package-version"} {
		t.Run(name, func(t *testing.T) {
			path, write := recoveryFixture(t)
			write(name, "dato recuperable o no reconocido")
			removed, err := RecoverMacOSInstallerResidues(path)
			if err == nil || len(removed) != 0 {
				t.Fatalf("se permitió un inventario inseguro: %v %v", removed, err)
			}
			for file, want := range map[string]string{name: "dato recuperable o no reconocido", "logs/service.log": "registro sintético anterior"} {
				got, readErr := os.ReadFile(filepath.Join(filepath.Dir(path), file))
				if readErr != nil || string(got) != want {
					t.Fatalf("se modificó %s: %v", file, readErr)
				}
			}
		})
	}
}

func TestMacOSInstallerRecoveryRefusesLinks(t *testing.T) {
	for _, name := range []string{"state", "logs/service.log", "state/license/externo"} {
		t.Run(name, func(t *testing.T) {
			path, _ := recoveryFixture(t)
			external := t.TempDir()
			original := filepath.Join(external, "original.pdf")
			if err := os.WriteFile(original, []byte("original intacto"), 0600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(filepath.Dir(path), name)
			if name == "state" {
				os.Remove(filepath.Join(link, "service.lock"))
				os.Remove(link)
			} else if name == "logs/service.log" {
				os.Remove(link)
			} else {
				os.MkdirAll(filepath.Dir(link), 0700)
			}
			if err := os.Symlink(external, link); err != nil {
				t.Fatal(err)
			}
			if _, err := RecoverMacOSInstallerResidues(path); err == nil {
				t.Fatal("se aceptó un enlace")
			}
			if got, err := os.ReadFile(original); err != nil || string(got) != "original intacto" {
				t.Fatal("se modificó el destino externo")
			}
		})
	}
}

func TestMacOSInstallerRecoveryRequiresExclusiveState(t *testing.T) {
	path, _ := recoveryFixture(t)
	unlock, err := storage.LockState(filepath.Join(filepath.Dir(path), "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err = RecoverMacOSInstallerResidues(path); err == nil || !strings.Contains(err.Error(), "bloquear") {
		t.Fatalf("se permitió recuperar un estado ocupado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "logs/service.log")); err != nil {
		t.Fatal("se retiró un residuo antes de conseguir el bloqueo")
	}
}

func TestMacOSInstallerRecoveryRefusesExistingConfiguration(t *testing.T) {
	path, write := recoveryFixture(t)
	write("config.json", "configuración a conservar")
	if _, err := RecoverMacOSInstallerResidues(path); err == nil {
		t.Fatal("se aceptó una configuración existente")
	}
	if value, _ := os.ReadFile(path); string(value) != "configuración a conservar" {
		t.Fatal("se modificó la configuración")
	}
}
