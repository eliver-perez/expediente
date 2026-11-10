package folderpicker

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

func Choose(ctx context.Context) (string, error) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return "", os.ErrNotExist
	}
	program, args := "zenity", []string{"--file-selection", "--directory", "--title=Selecciona una carpeta para AIDEX"}
	if _, err := exec.LookPath(program); err != nil {
		program = "kdialog"
		args = []string{"--getexistingdirectory", os.Getenv("HOME"), "--title", "AIDEX"}
	}
	output, err := exec.CommandContext(ctx, program, args...).Output()
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return "", nil
	}
	return strings.TrimSuffix(string(output), "\n"), err
}
