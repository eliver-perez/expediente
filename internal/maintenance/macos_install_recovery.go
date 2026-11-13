package maintenance

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gestor-documental/internal/storage"
)

// RecoverMacOSInstallerResidues permite iniciar de nuevo únicamente cuando una
// desinstalación antigua dejó registros y marcadores reconocidos, sin config.
// No reutiliza la eliminación del desinstalador: aquí nunca se retiran bases,
// licencias, originales, enlaces ni archivos desconocidos, ni se crean archivos
// de conservación. Se valida el inventario completo antes de retirar residuos.
func RecoverMacOSInstallerResidues(configPath string) ([]string, error) {
	configPath = filepath.Clean(configPath)
	data := filepath.Dir(configPath)
	if !filepath.IsAbs(configPath) || filepath.Base(configPath) != "config.json" || filepath.Base(data) != "data" || filepath.Base(filepath.Dir(data)) != "AIDEX" {
		return nil, fmt.Errorf("la recuperación requiere el área data del paquete AIDEX")
	}
	if err := regularWithoutLinks(configPath); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(configPath); !os.IsNotExist(err) {
		return nil, fmt.Errorf("la configuración existe o no puede comprobarse; no se inició la recuperación")
	}
	root, err := os.OpenRoot(data)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	files, err := macOSInstallerResidues(root)
	if err != nil || len(files) == 0 {
		return nil, err
	}
	// El servicio ya debe estar retirado; además se exige el bloqueo del estado.
	// Ante un inventario rechazado ni siquiera se crea un archivo de bloqueo.
	unlock, err := storage.LockState(filepath.Join(data, "state"))
	if err != nil {
		return nil, fmt.Errorf("no se pudo bloquear el estado; no se retiraron residuos: %w", err)
	}
	defer unlock()
	files, err = macOSInstallerResidues(root)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, path := range files {
		if path == "state/service.lock" {
			continue
		}
		if err = root.Remove(path); err != nil {
			return removed, fmt.Errorf("no se pudo retirar el residuo %q: %w", path, err)
		}
		removed = append(removed, path)
	}
	if err = root.Remove("state/service.lock"); err != nil {
		return removed, fmt.Errorf("no se pudo retirar el bloqueo residual: %w", err)
	}
	removed = append(removed, "state/service.lock")
	// Conservar directorios vacíos: init puede usarlos sin borrar ni seguir rutas.
	return removed, nil
}

func macOSInstallerResidues(root *os.Root) ([]string, error) {
	var files []string
	err := fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("no se pudo inspeccionar %q; se conservan los datos: %w", path, walkErr)
		}
		if entry.IsDir() {
			switch path {
			case ".", "logs", "state", "state/license", "state/uploads", "state/previews", "state/extraction", "state/preview-work", "state/backups", "state/upgrade-backups":
				return nil
			default:
				return fmt.Errorf("falta config.json: directorio no reconocido %q; se conservan todos los datos y no se inicializó la instalación", path)
			}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("falta config.json: enlace o tipo de archivo no permitido en %q; se conservan todos los datos", path)
		}
		allowed := path == "logs/service.log" || rotatedServiceLog.MatchString(path)
		switch path {
		case "state/service.lock", "service-enabled", "state/service-enabled":
			allowed = info.Size() == 0
		case "package-version", "state/package-version":
			file, openErr := root.Open(path)
			if openErr != nil {
				return openErr
			}
			value, readErr := io.ReadAll(io.LimitReader(file, 32))
			file.Close()
			if readErr != nil {
				return readErr
			}
			allowed = info.Size() < 32 && strings.TrimSpace(string(value)) == "1.0.0"
		}
		if !allowed {
			kind := "archivo no reconocido o potencialmente recuperable"
			lower := strings.ToLower(path)
			if strings.HasPrefix(path, "state/license/") {
				kind = "estado de licencia potencialmente recuperable"
			} else if strings.HasPrefix(lower, "state/postgresql/") || strings.HasSuffix(lower, ".db") {
				kind = "estado de base de datos potencialmente recuperable"
			}
			return fmt.Errorf("falta config.json: %s en %q; restaura la configuración anterior. Se conservan todos los datos; no se inicializó ni borró ningún dato", kind, path)
		}
		files = append(files, path)
		return nil
	})
	return files, err
}
