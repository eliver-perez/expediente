package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func authorizeBootstrapForm(requirePipe bool) error {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return fmt.Errorf("el asistente requiere autorización de un administrador de Windows")
	}
	if requirePipe {
		kind, err := windows.GetFileType(windows.Handle(os.Stdin.Fd()))
		if err != nil || kind != windows.FILE_TYPE_PIPE {
			return fmt.Errorf("el asistente requiere una entrada privada redirigida")
		}
	}
	return nil
}
