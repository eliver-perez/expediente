//go:build darwin || linux

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func runHost(application func(context.Context, func()) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return application(ctx, func() {})
}
