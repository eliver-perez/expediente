package maintenance

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/storage"
)

var generatedID = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var generatedWork = regexp.MustCompile(`^(job|preview)-[0-9]+$`)
var rotatedServiceLog = regexp.MustCompile(`^logs/service\.log\.[0-9]+(\.gz)?$`)
var temporaryConfig = regexp.MustCompile(`^\.aidex-config-[0-9]+$`)

// EraseMacOSInstallation limpia el área fija del paquete. Los documentos y
// archivos no identificados se trasladan fuera de data antes de borrar metadatos.
// El instalador nunca llama esta operación: requiere confirmación del usuario.
func EraseMacOSInstallation(configPath string) (string, error) {
	configPath = filepath.Clean(configPath)
	data := filepath.Dir(configPath)
	code := filepath.Dir(data)
	if !filepath.IsAbs(configPath) || filepath.Base(configPath) != "config.json" || filepath.Base(data) != "data" || filepath.Base(code) != "AIDEX" {
		return "", fmt.Errorf("la eliminación completa requiere el área data del paquete AIDEX")
	}
	if err := regularWithoutLinks(configPath); err != nil {
		return "", err
	}
	cfg, err := config.Load(configPath)
	configExists := err == nil
	if os.IsNotExist(err) {
		// Recuperar un borrado antiguo que ya retiró config y SQLite, pero dejó
		// logs y service.lock. Solo el desinstalador confirmado permite esto.
		cfg = config.Defaults(filepath.Join(data, "state"))
		if err = storage.PreparePrivateDirectory(cfg.StateDirectory); err != nil {
			return "", err
		}
		for _, name := range []string{"documental.db", "documental.db-wal", "documental.db-shm"} {
			if _, statErr := os.Lstat(filepath.Join(cfg.StateDirectory, name)); !os.IsNotExist(statErr) {
				return "", fmt.Errorf("existe una base potencialmente recuperable sin configuración; se conservan los datos")
			}
		}
	} else if err != nil {
		return "", fmt.Errorf("no se pudo validar la configuración; se conservan los datos: %w", err)
	}
	// No recorrer un estado configurable externo, compartido o enlazado.
	if filepath.Clean(cfg.StateDirectory) != filepath.Join(data, "state") {
		return "", fmt.Errorf("el estado está fuera del área interna del paquete; requiere revisión sin borrar datos")
	}
	unlock, err := storage.LockState(cfg.StateDirectory)
	if err != nil {
		return "", err
	}
	defer unlock()
	protected, err := macOSDocumentRoots(cfg)
	if err != nil {
		return "", err
	}
	for _, path := range protected {
		if containsPath(path, configPath) || containsPath(path, cfg.StateDirectory) {
			return "", fmt.Errorf("una ruta documental se superpone con los metadatos; se conservan los datos")
		}
	}
	root, err := os.OpenRoot(code)
	if err != nil {
		return "", err
	}
	defer root.Close()
	var remove, preserve, directories []string
	err = fs.WalkDir(root.FS(), "data", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			directories = append(directories, path)
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("tipo de archivo no reconocido en el área interna; no se inició la eliminación")
		}
		relative := strings.TrimPrefix(path, "data/")
		internal := info.Mode().IsRegular() && macOSInternalFile(relative, configPath)
		for _, documentRoot := range protected {
			if containsPath(documentRoot, filepath.Join(code, filepath.FromSlash(path))) {
				internal = false
			}
		}
		if internal {
			remove = append(remove, path)
		} else {
			// WalkDir no sigue enlaces. Se conserva el enlace mismo, nunca su destino.
			preserve = append(preserve, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	archive := ""
	if len(preserve) > 0 {
		archive = filepath.Join("Documentos conservados", domain.NewID())
		if err = storage.PreparePrivateDirectory(filepath.Join(code, archive)); err != nil {
			return "", err
		}
		// Rename dentro del mismo volumen conserva bytes y evita requerir espacio
		// para una segunda copia. Ninguna ruta documental externa se modifica.
		for _, path := range preserve {
			destination := filepath.Join(archive, strings.TrimPrefix(path, "data/"))
			if err = root.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				return filepath.Join(code, archive), err
			}
			if err = root.Rename(path, destination); err != nil {
				return filepath.Join(code, archive), err
			}
		}
	}
	// Mantener configuración y bloqueo hasta haber borrado el resto. Un fallo
	// conserva las herramientas y no se comunica como desinstalación completada.
	for _, path := range remove {
		if path == "data/config.json" || path == "data/state/service.lock" {
			continue
		}
		if err = root.Remove(path); err != nil {
			return archivePath(code, archive), err
		}
	}
	if configExists {
		if err = root.Remove("data/config.json"); err != nil {
			return archivePath(code, archive), err
		}
	}
	if err = root.Remove("data/state/service.lock"); err != nil && !os.IsNotExist(err) {
		return archivePath(code, archive), err
	}
	// Solo rmdir sobre directorios ya vacíos, en orden inverso. No RemoveAll.
	for index := len(directories) - 1; index >= 0; index-- {
		if err = root.Remove(directories[index]); err != nil {
			return archivePath(code, archive), err
		}
	}
	if _, err = root.Lstat("data"); !os.IsNotExist(err) {
		return archivePath(code, archive), fmt.Errorf("no se pudo verificar la eliminación completa de data")
	}
	return archivePath(code, archive), nil
}

func archivePath(code, archive string) string {
	if archive == "" {
		return ""
	}
	return filepath.Join(code, archive)
}

func containsPath(parent, child string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func macOSDocumentRoots(cfg config.Config) ([]string, error) {
	roots := []string{cfg.PrivateUploadDirectory()}
	path := filepath.Join(cfg.StateDirectory, "documental.db")
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return roots, nil
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := regularWithoutLinks(path + suffix); err != nil {
			return nil, err
		}
	}
	address := url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", address.String())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, "SELECT canonical_path FROM storage_roots")
	if err != nil {
		return nil, fmt.Errorf("no se pudieron verificar las rutas documentales; se conservan los datos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var path string
		if err = rows.Scan(&path); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("ruta documental inválida; se conservan los datos")
		}
		roots = append(roots, filepath.Clean(path))
	}
	return roots, rows.Err()
}

// Solo nombres reservados o formatos generados por AIDEX. Lo desconocido se
// conserva como posible documento, aunque esté dentro de una carpeta interna.
func macOSInternalFile(path, configPath string) bool {
	if rotatedServiceLog.MatchString(path) || temporaryConfig.MatchString(path) {
		return true
	}
	switch path {
	case "config.json", "package-version", "service-enabled", "logs/service.log",
		"state/documental.db", "state/documental.db-wal", "state/documental.db-shm", "state/service.lock",
		"state/package-version", "state/service-enabled", "state/license/installation.json", "state/license/recovery.json":
		return true
	}
	if !strings.Contains(path, "/") && recognizedArchive(filepath.Join(filepath.Dir(configPath), path), configPath) {
		return true
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "state" {
		return false
	}
	name := parts[2]
	switch parts[1] {
	case "license":
		return len(parts) == 3 && installationArchive.MatchString(name)
	case "upgrade-backups":
		return len(parts) == 3 && strings.HasPrefix(name, "before-processing-") && strings.HasSuffix(name, ".db") && recognizedArchive(name, configPath)
	case "previews":
		return len(parts) == 3 && strings.HasSuffix(name, ".preview") && generatedID.MatchString(strings.TrimSuffix(name, ".preview"))
	case "extraction", "preview-work":
		return len(parts) > 3 && generatedWork.MatchString(name)
	case "backups":
		if len(parts) > 3 {
			return strings.HasPrefix(name, ".inspect-db-")
		}
		for _, extension := range []string{".json", ".aidexbak", ".plain", ".pre-restore.aidexbak"} {
			if strings.HasSuffix(name, extension) && generatedID.MatchString(strings.TrimSuffix(name, extension)) {
				return true
			}
		}
		return strings.HasPrefix(name, ".backup-") || strings.HasPrefix(name, ".recovery-") || strings.HasPrefix(name, ".operation-")
	}
	return false
}
