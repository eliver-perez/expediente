package folderpicker

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

func verifyLocal(ctx context.Context, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	program, err := exec.CommandContext(ctx, "/usr/bin/plutil", "-extract", "ProgramArguments.0", "raw", "-o", "-", "/Library/LaunchDaemons/app.aidex.plist").Output()
	if err != nil || strings.TrimSpace(string(program)) != "/Library/Application Support/AIDEX/bin/aidex-server" {
		return fmt.Errorf("no coincide la instalación local")
	}
	job, err := exec.CommandContext(ctx, "/bin/launchctl", "print", "system/app.aidex").Output()
	if err != nil {
		return err
	}
	pid := ""
	for _, line := range strings.Split(string(job), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "pid" && fields[1] == "=" {
			number, e := strconv.Atoi(fields[2])
			if e == nil && number > 1 {
				pid = fields[2]
			}
		}
	}
	if pid == "" {
		return fmt.Errorf("el servicio local no está iniciado")
	}
	table, err := exec.CommandContext(ctx, "/usr/sbin/netstat", "-anv", "-p", "tcp").Output()
	if err != nil {
		return err
	}
	if !ownsDarwinListener(string(table), u.Port(), pid) {
		return fmt.Errorf("la conexión no pertenece al servicio local")
	}
	return nil
}
