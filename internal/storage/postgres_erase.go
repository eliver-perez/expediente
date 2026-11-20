package storage

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErasePrivatePostgres removes the stopped cluster owned by this installation.
// The caller holds the outer state lock. It does not start or connect to a server.
func ErasePrivatePostgres(state string) error {
	root, err := os.OpenRoot(state)
	if err != nil {
		return err
	}
	defer root.Close()
	const cluster = "postgresql"
	const descriptor = cluster + "/installation.json"
	const version = cluster + "/data/PG_VERSION"
	const lock = cluster + "/service.lock"
	info, err := root.Lstat(cluster)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("ruta del clúster PostgreSQL no válida; se conservan los datos")
	}
	unlock, err := LockState(filepath.Join(state, cluster))
	if err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			_ = unlock()
		}
	}()
	if _, err := root.Lstat(cluster + "/data/postmaster.pid"); !os.IsNotExist(err) {
		return fmt.Errorf("no se confirmó que el clúster PostgreSQL esté detenido; se conservan los datos")
	}

	// Validate every entry before deleting anything. WalkDir never follows links.
	var files, dirs []string
	err = fs.WalkDir(root.FS(), cluster, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		relative := strings.TrimPrefix(path, cluster+"/")
		if !entry.Type().IsRegular() || !privateClusterFile(relative) {
			return fmt.Errorf("archivo o enlace no reconocido en el clúster PostgreSQL: %s; se conservan los datos", path)
		}
		if path != lock && path != descriptor && path != version {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	versionBytes, err := root.ReadFile(version)
	if err == nil && strings.TrimSpace(string(versionBytes)) != "18" {
		return fmt.Errorf("versión inesperada del clúster PostgreSQL; se conservan los datos")
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	configuration, err := root.ReadFile(descriptor)
	switch {
	case err == nil:
		var owned postgresInstallation
		if json.Unmarshal(configuration, &owned) != nil || owned.Format != 1 || owned.Port < 1024 || owned.Port > 65535 || len(owned.Password) != 64 || len(owned.AdminPassword) != 64 {
			return fmt.Errorf("descriptor del clúster PostgreSQL no válido; se conservan los datos")
		}
	case os.IsNotExist(err) && versionBytes == nil && len(files) == 0:
		// A prior attempt may have already removed the owned files.
	default:
		return fmt.Errorf("clúster PostgreSQL sin descriptor verificable; se conservan los datos: %w", err)
	}
	for _, path := range append(files, version) {
		if err := root.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	// Remove only empty, validated directories and keep the ownership descriptor
	// until their contents have been removed, so interrupted erasure is retryable.
	for i := len(dirs) - 1; i > 0; i-- {
		if err := root.Remove(dirs[i]); err != nil {
			return err
		}
	}
	if err := root.Remove(descriptor); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := unlock(); err != nil {
		return err
	}
	locked = false
	if err := root.Remove(lock); err != nil {
		return err
	}
	return root.Remove(cluster)
}

func privateClusterFile(relative string) bool {
	switch relative {
	case "data/global/config_exec_params", "data/pg_logical/replorigin_checkpoint", "data/pg_stat/pgstat.stat", ".init-password":
		return true
	default:
		return IsPostgresMetadata(relative)
	}
}
