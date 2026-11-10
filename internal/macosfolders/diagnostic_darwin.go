//go:build darwin && cgo

package macosfolders

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"syscall"
)

// La comprobación se ejecuta en un proceso separado con los grupos y la cuenta
// efectivos del destinatario. Nunca se infieren permisos ignorando las ACL deny.
type TraversalProbe func(context.Context, int, string) (bool, error)

type OperationError struct {
	Operation, Path string
	Err             error
}

func (err *OperationError) Error() string {
	return "No se pudieron aplicar los permisos de la carpeta. Consulta el registro de autorización para soporte."
}
func (err *OperationError) Unwrap() error { return err.Err }

func operationFailure(operation, path string, err error) error {
	var errno syscall.Errno
	_ = errors.As(err, &errno)
	fields := []any{"operation", operation, "path", path, "errno", int(errno), "real_uid", os.Getuid(), "effective_uid", os.Geteuid(), "effective_gid", os.Getegid(), "cause", err.Error()}
	if info, statErr := os.Lstat(path); statErr == nil {
		fields = append(fields, "mode", info.Mode().String())
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			fields = append(fields, "owner_uid", stat.Uid, "owner_gid", stat.Gid, "file_flags", stat.Flags, "inode", stat.Ino, "device", stat.Dev)
		}
	}
	slog.Error("folder authorization failed", fields...)
	return &OperationError{Operation: operation, Path: path, Err: err}
}
