package folderpicker

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"strconv"
)

func verifyLocal(ctx context.Context, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		return err
	}
	account, err := user.Lookup("aidex")
	if err != nil {
		return err
	}
	// /proc/net/tcp exposes the listener UID without exposing other users' file
	// descriptors. The dedicated service UID cannot be impersonated by a tunnel
	// running as a desktop user. Packaged AIDEX binds explicitly to IPv4.
	table, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return err
	}
	if !ownsLinuxListener(string(table), port, account.Uid) {
		return fmt.Errorf("la conexión no pertenece al servicio local")
	}
	return ctx.Err()
}
