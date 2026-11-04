package maintenance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gestor-documental/internal/config"
	"gestor-documental/internal/storage"
)

func macOSFixture(t *testing.T) (string, config.Config) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "AIDEX", "data", "config.json")
	if err = config.Initialize(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(context.Background(), cfg.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	return path, cfg
}

func writeFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("contenido sintético"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMacOSCompleteEraseRemovesResiduesAndPreservesDocuments(t *testing.T) {
	path, cfg := macOSFixture(t)
	data := filepath.Dir(path)
	metadata := []string{"logs/service.log", "package-version", "service-enabled", "state/license/installation.json",
		"state/license/recovery.json", "state/previews/11111111-1111-4111-8111-111111111111.preview",
		"state/extraction/job-123/source.pdf", "state/preview-work/preview-456/profile/user/settings.xml",
		"state/backups/11111111-1111-4111-8111-111111111111.json", "state/backups/11111111-1111-4111-8111-111111111111.aidexbak"}
	for _, relative := range metadata {
		writeFixture(t, filepath.Join(data, relative))
	}
	originals := []string{"state/uploads/pendiente.pdf", "state/uploads/carga.part", "documentos-administrados/original.docx",
		"state/unknown.txt", "state/license/documento.json", "state/extraction/job-personal/original.pdf"}
	for _, relative := range originals {
		writeFixture(t, filepath.Join(data, relative))
	}
	external := filepath.Join(filepath.Dir(filepath.Dir(data)), "biblioteca vinculada", "original.pdf")
	writeFixture(t, external)
	link := filepath.Join(data, "state", "enlace-externo")
	if err := os.Symlink(filepath.Dir(external), link); err != nil {
		t.Fatal(err)
	}
	archive, err := EraseMacOSInstallation(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Lstat(data); !os.IsNotExist(err) {
		t.Fatal("quedaron residuos bloqueando la reinstalación", err)
	}
	for _, relative := range originals {
		contents, err := os.ReadFile(filepath.Join(archive, relative))
		if err != nil || string(contents) != "contenido sintético" {
			t.Fatal("documento perdido", relative, err)
		}
	}
	for _, relative := range metadata {
		if _, err = os.Lstat(filepath.Join(archive, relative)); !os.IsNotExist(err) {
			t.Fatal("metadatos archivados en lugar de eliminados", relative, err)
		}
	}
	if _, err = os.Readlink(filepath.Join(archive, "state/enlace-externo")); err != nil {
		t.Fatal("no se conservó el enlace", err)
	}
	if contents, err := os.ReadFile(external); err != nil || string(contents) != "contenido sintético" {
		t.Fatal("se alteró el original externo", err)
	}
	// Repetir un ciclo limpio no deja lock, logs ni identidad que impidan init.
	for i := 0; i < 2; i++ {
		if err = config.Initialize(path); err != nil {
			t.Fatal(err)
		}
		db, err := storage.Open(context.Background(), cfg.StateDirectory)
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
		if _, err = EraseMacOSInstallation(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMacOSEraseProtectsRegisteredOriginalEvenWithReservedName(t *testing.T) {
	path, cfg := macOSFixture(t)
	data := filepath.Dir(path)
	root := filepath.Join(data, "logs")
	writeFixture(t, filepath.Join(root, "service.log"))
	db, err := storage.Open(context.Background(), cfg.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Writer.Exec("INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('lib','Sintética','hybrid','2026','2026')")
	if err == nil {
		_, err = db.Writer.Exec(`INSERT INTO storage_roots(id,library_id,storage_source,canonical_path,comparison_key,case_sensitive,status,watch_mode,reconcile_interval_seconds,created_at) VALUES('root','lib','managed',?,?,1,'active','polling',900,'2026')`, root, root)
	}
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	archive, err := EraseMacOSInstallation(path)
	if err != nil {
		t.Fatal(err)
	}
	if contents, err := os.ReadFile(filepath.Join(archive, "logs/service.log")); err != nil || string(contents) != "contenido sintético" {
		t.Fatal("se eliminó un documento registrado", err)
	}
}

func TestMacOSEraseRefusesBusyStateAndCorruptDatabase(t *testing.T) {
	path, cfg := macOSFixture(t)
	unlock, err := storage.LockState(cfg.StateDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = EraseMacOSInstallation(path); err == nil {
		t.Fatal("aceptó el estado en uso")
	}
	unlock()
	if err = os.WriteFile(filepath.Join(cfg.StateDirectory, "documental.db"), []byte("documento ajeno"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = EraseMacOSInstallation(path); err == nil {
		t.Fatal("aceptó una base no verificable")
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal("borró configuración antes de validar", err)
	}
}

func TestMacOSEraseRefusesLinkedState(t *testing.T) {
	path, cfg := macOSFixture(t)
	outside := filepath.Join(filepath.Dir(filepath.Dir(path)), "estado conservado")
	if err := os.Rename(cfg.StateDirectory, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, cfg.StateDirectory); err != nil {
		t.Fatal(err)
	}
	if _, err := EraseMacOSInstallation(path); err == nil {
		t.Fatal("aceptó estado enlazado")
	}
	if _, err := os.Stat(filepath.Join(outside, "documental.db")); err != nil {
		t.Fatal(err)
	}
}

func TestMacOSEraseRecoversPreviousPartialEraseWithoutReinitializing(t *testing.T) {
	path, _ := macOSFixture(t)
	data := filepath.Dir(path)
	writeFixture(t, filepath.Join(data, "logs/service.log"))
	writeFixture(t, filepath.Join(data, "state/uploads/original.pdf"))
	if err := EraseInternal(path); err != nil {
		t.Fatal(err)
	}
	archive, err := EraseMacOSInstallation(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(data); !os.IsNotExist(err) {
		t.Fatal("quedaron residuos antiguos", err)
	}
	if _, err = os.Stat(filepath.Join(archive, "state/uploads/original.pdf")); err != nil {
		t.Fatal(err)
	}
}

func TestMacOSEraseWithoutConfigPreservesRecoverableDatabase(t *testing.T) {
	path, cfg := macOSFixture(t)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := EraseMacOSInstallation(path); err == nil {
		t.Fatal("aceptó una base recuperable sin configuración")
	}
	if _, err := os.Stat(filepath.Join(cfg.StateDirectory, "documental.db")); err != nil {
		t.Fatal(err)
	}
}
