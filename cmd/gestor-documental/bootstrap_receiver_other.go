//go:build !darwin

package main

import (
	"context"
	"fmt"
)

func receiveMacOSBootstrap(context.Context, string, string, int) error {
	return fmt.Errorf("el asistente requiere macOS")
}
