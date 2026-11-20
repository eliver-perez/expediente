package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

func authorizeBootstrapForm(requirePipe bool) error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("el asistente debe utilizar la cuenta de servicio de AIDEX")
	}
	if requirePipe {
		info, err := os.Stdin.Stat()
		if err != nil || info.Mode()&os.ModeNamedPipe == 0 {
			return fmt.Errorf("el asistente requiere una entrada privada redirigida")
		}
	}
	return nil
}

// Only the authorized local helper can connect. The GUI verifies its effective
// UID in turn, before it sends the one-shot form to the service account.
func receiveMacOSBootstrap(ctx context.Context, configPath, socket string, uid int) error {
	if os.Geteuid() != 0 || uid <= 0 || uint64(uid) >= 1<<32 {
		return fmt.Errorf("se requiere autorización de administrador de macOS")
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	dialer := net.Dialer{Timeout: 5 * time.Second}
	connection, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		return fmt.Errorf("no se pudo abrir la conexión privada del asistente: %w", err)
	}
	defer connection.Close()
	raw, err := connection.(*net.UnixConn).SyscallConn()
	if err != nil {
		return err
	}
	var authorized bool
	if err = raw.Control(func(fd uintptr) {
		peer, err := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		authorized = err == nil && peer.Uid == uint32(uid)
	}); err != nil || !authorized {
		return fmt.Errorf("no se pudo verificar la identidad local del asistente")
	}
	if err := connection.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, "/usr/bin/sudo", "-n", "-u", "_aidex",
		executable, "bootstrap-ui", "--config", configPath)
	command.Stdin, command.Stdout, command.Stderr = connection, os.Stdout, os.Stderr
	return command.Run()
}
