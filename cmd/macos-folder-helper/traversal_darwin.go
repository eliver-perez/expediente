//go:build darwin && cgo

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

var errTraversalDenied = errors.New("recorrido no disponible")

func probeTraversal(ctx context.Context, uid int, path string) (bool, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	self, err := os.Executable()
	if err != nil {
		return false, err
	}
	probe := exec.CommandContext(probeCtx, self, "check-traverse", path)
	if os.Geteuid() == 0 {
		account, err := user.LookupId(strconv.Itoa(uid))
		if err != nil {
			return false, err
		}
		gid, err := strconv.ParseUint(account.Gid, 10, 32)
		if err != nil {
			return false, err
		}
		ids, err := account.GroupIds()
		if err != nil {
			return false, err
		}
		groups := make([]uint32, 0, len(ids))
		for _, id := range ids {
			number, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				return false, err
			}
			groups = append(groups, uint32(number))
		}
		probe.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: groups}}
	} else if uid != os.Getuid() || uid != os.Geteuid() {
		return false, fmt.Errorf("no se pudo comprobar la identidad de recorrido")
	}
	if err = probe.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 3 {
			return false, nil
		}
		return false, fmt.Errorf("falló la comprobación de recorrido: %w", err)
	}
	return true, nil
}
