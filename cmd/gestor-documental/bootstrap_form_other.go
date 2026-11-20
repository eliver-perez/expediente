//go:build !windows && !darwin

package main

import "fmt"

func authorizeBootstrapForm(bool) error {
	return fmt.Errorf("el asistente gráfico requiere Windows o macOS")
}
