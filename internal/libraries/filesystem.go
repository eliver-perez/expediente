package libraries

import (
	"errors"
	"fmt"
	"gestor-documental/internal/domain"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unicode"
)

type directoryInfo struct {
	Path, Identity string
	CaseSensitive  bool
}

func inspectDirectory(path string) (directoryInfo, error) {
	var result directoryInfo
	if !filepath.IsAbs(path) {
		return result, invalid("La carpeta debe ser una ruta absoluta del servidor.")
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return result, directoryFailure(path, err)
	}
	directory, err := os.Open(canonical)
	if err != nil {
		return result, directoryFailure(path, err)
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil {
		return result, directoryFailure(path, err)
	}
	if !info.IsDir() {
		return result, invalid("Selecciona una carpeta legible.")
	}
	entries, err := directory.ReadDir(128)
	if err != nil && err != io.EOF {
		return result, directoryFailure(path, err)
	}
	identity, _, err := physicalIdentity(directory)
	if err != nil {
		return result, directoryFailure(path, err)
	}
	result = directoryInfo{Path: canonical, Identity: identity, CaseSensitive: true}
	// Probe entries inside the selected directory, not its parent volume. If the
	// directory is empty or ambiguous, preserve spelling; OS identities still detect aliases.
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		alternateName := strings.Map(func(character rune) rune {
			if unicode.IsLower(character) {
				return unicode.ToUpper(character)
			}
			return unicode.ToLower(character)
		}, entry.Name())
		if alternateName == entry.Name() {
			continue
		}
		originalInfo, originalError := os.Lstat(filepath.Join(canonical, entry.Name()))
		alternateInfo, alternateError := os.Lstat(filepath.Join(canonical, alternateName))
		if originalError == nil && alternateError == nil && os.SameFile(originalInfo, alternateInfo) {
			result.CaseSensitive = false
			break
		}
		if originalError == nil && (os.IsNotExist(alternateError) || alternateError == nil) {
			break
		}
	}
	return result, nil
}

// CheckLinkedDirectory repeats the complete read probe in the service process
// after a local administrator changes a folder ACL.
func CheckLinkedDirectory(path string) error {
	_, err := inspectDirectory(path)
	return err
}
func within(parent, child string, caseSensitive bool) bool {
	if !caseSensitive {
		parent = strings.ToLower(parent)
		child = strings.ToLower(child)
	}
	relative, err := filepath.Rel(parent, child)
	return err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)))
}
func comparison(path string, sensitive bool) string {
	if sensitive {
		return path
	}
	return strings.ToLower(path)
}
func relativeSafe(path string) bool {
	return path != "" && path != "." && filepath.IsLocal(filepath.FromSlash(path)) && !strings.Contains(path, "\\") && filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) == path
}

// OpenRoot confines resolution even if a path component is swapped after Lstat.
// Internal symlinks are rejected, including links whose target is within the root.
func openLinked(root Root, relative string) (*os.File, error) {
	if !relativeSafe(relative) {
		return nil, fmt.Errorf("invalid relative path")
	}
	container, err := os.OpenRoot(root.Path)
	if err != nil {
		return nil, err
	}
	defer container.Close()
	directory, err := container.Open(".")
	if err != nil {
		return nil, err
	}
	identity, _, identityError := physicalIdentity(directory)
	directory.Close()
	if identityError != nil || identity != root.Identity {
		return nil, fmt.Errorf("root identity changed")
	}
	prefix := ""
	for _, component := range strings.Split(relative, "/") {
		prefix = filepath.Join(prefix, component)
		info, err := container.Lstat(prefix)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return nil, fmt.Errorf("link or special file skipped")
		}
	}
	before, err := container.Lstat(filepath.FromSlash(relative))
	if err != nil {
		return nil, err
	}
	file, err := container.Open(filepath.FromSlash(relative))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		file.Close()
		return nil, fmt.Errorf("file changed while opening")
	}
	return file, nil
}

func volumeKey(identity string) string {
	parts := strings.Split(identity, ":")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + ":" + parts[1]
}

func directoryFailure(path string, err error) error {
	shared := strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//")
	// Preserve the OS error number for support without logging a private path.
	var errno syscall.Errno
	if errors.As(err, &errno) {
		slog.Warn("directory inspection failed", "os", runtime.GOOS, "errno", uint64(errno), "network_path", shared)
	}
	if errors.Is(err, os.ErrPermission) {
		if runtime.GOOS == "darwin" && errors.Is(err, syscall.EPERM) {
			return domain.Failure("ROOT_PRIVACY_DENIED", "macOS denegó el acceso del servicio AIDEX a esta ubicación. Revisa Privacidad y seguridad para el proceso del servicio; autorizar el navegador no le concede acceso.", 422)
		}
		message := "La cuenta del servicio AIDEX no puede leer esta carpeta. Comprueba que tenga permiso de lectura y recorrido en esta ruta y sus carpetas superiores."
		if runtime.GOOS == "darwin" {
			message = "El servicio AIDEX (_aidex) no puede leer esta carpeta. Comprueba los permisos de la ruta y sus carpetas superiores."
		} else if runtime.GOOS == "windows" {
			message = `El servicio AIDEX (NT SERVICE\AIDEX) no puede leer esta carpeta. Concede lectura y recorrido a esa cuenta en la ruta y sus carpetas superiores.`
			if shared {
				message = "El servicio AIDEX no puede leer este recurso de red. Revisa tanto los permisos del recurso compartido como los de Seguridad para la identidad con la que accede el servicio; la sesión del usuario del navegador no le concede acceso."
			}
		}
		return domain.Failure("ROOT_PERMISSION_DENIED", message, 422)
	}
	if errors.Is(err, os.ErrNotExist) {
		message := "No se encontró la carpeta indicada. Comprueba que la ruta exista en el equipo donde corre AIDEX."
		if shared {
			message = "No se encontró la ruta de red. Comprueba el nombre del equipo, el recurso compartido y las carpetas desde el equipo donde corre AIDEX."
		}
		return domain.Failure("ROOT_NOT_FOUND", message, 422)
	}
	if shared {
		return domain.Failure("ROOT_NETWORK_UNAVAILABLE", "No se pudo acceder a la ruta de red. Comprueba la disponibilidad del equipo y el acceso de la cuenta del servicio.", 422)
	}
	return domain.Failure("ROOT_READ_FAILED", "No se pudo leer la carpeta. Comprueba su disponibilidad y vuelve a intentarlo; si persiste, consulta Registro de errores.", 422)
}
