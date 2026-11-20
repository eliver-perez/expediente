//go:build !windows

package main

import "fmt"

func authorizeWindowsBootstrap(bool) error {
	return fmt.Errorf("el asistente gráfico requiere Windows")
}
