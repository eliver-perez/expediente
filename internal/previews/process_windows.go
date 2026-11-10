package previews

import (
	"context"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// LibreOffice starts a helper process on Windows. Terminate that process tree,
// not unrelated interactive LibreOffice sessions (each job has a private profile).
func configureProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		killer := exec.CommandContext(ctx, "taskkill.exe", "/PID", strconv.Itoa(command.Process.Pid), "/T", "/F")
		killer.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		killer.WaitDelay = time.Second
		if err := killer.Run(); err != nil {
			return command.Process.Kill()
		}
		return nil
	}
}
