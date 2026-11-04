//go:build darwin && cgo

package main

/*
#include <syslog.h>
#include <stdlib.h>
static void aidex_log(const char *message) {
    openlog("AIDEX-folder-authorization", LOG_PID, LOG_USER);
    syslog(LOG_NOTICE, "%s", message);
}
*/
import "C"

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type systemLog struct{}

func (systemLog) Write(data []byte) (int, error) {
	message := C.CString(string(data))
	defer C.free(unsafe.Pointer(message))
	C.aidex_log(message)
	return len(data), nil
}

// Sin crear directorios: el desinstalador también utiliza el auxiliar cuando
// ya retiró data. Ante un fallo se registra en el sistema, nunca en el diálogo.
func configureAuthorizationLog() func() {
	previous := slog.Default()
	file, err := openAuthorizationLog("/Library/Application Support/AIDEX/data/logs")
	if err != nil {
		slog.SetDefault(slog.New(slog.NewJSONHandler(systemLog{}, nil)))
		slog.Warn("authorization file log unavailable", "operation", "open_private_log", "cause", err.Error())
		return func() { slog.SetDefault(previous) }
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(file, nil)))
	return func() { _ = file.Sync(); _ = file.Close(); slog.SetDefault(previous) }
}

func openAuthorizationLog(directory string) (*os.File, error) {
	for path := directory; path != "/"; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("directorio de registro no válido")
		}
	}
	selected, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	if !os.SameFile(selected, opened) {
		return nil, fmt.Errorf("el directorio de registro cambió")
	}
	file, err := root.OpenFile("folder-authorization.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || stat.Uid != uint32(os.Geteuid()) || stat.Nlink != 1 || info.Mode().Perm() != 0600 {
		file.Close()
		return nil, fmt.Errorf("archivo de registro no seguro")
	}
	// Registro acotado y privado. No se guardan tickets, sesiones o licencias.
	if info.Size() > 1<<20 {
		if err = file.Truncate(0); err != nil {
			file.Close()
			return nil, err
		}
	}
	return file, nil
}
