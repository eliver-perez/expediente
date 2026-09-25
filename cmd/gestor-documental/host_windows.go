package main

import (
	"context"
	"fmt"
	"gestor-documental/internal/buildinfo"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"os"
	"os/signal"
	"time"
)

type windowsService struct {
	application func(context.Context, func()) error
}

func runHost(application func(context.Context, func()) error) error {
	inService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !inService {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return application(ctx, func() {})
	}
	if buildinfo.ServiceName == "" || len(os.Args) < 2 || os.Args[1] != "serve" {
		return fmt.Errorf("invalid installed service invocation")
	}
	return svc.Run(buildinfo.ServiceName, &windowsService{application: application})
}

func (service *windowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{}, 1)
	finished := make(chan error, 1)
	go func() { finished <- service.application(ctx, func() { ready <- struct{}{} }) }()
	status := svc.Status{State: svc.StartPending, WaitHint: 10000, CheckPoint: 1}
	changes <- status
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ready:
			if status.State != svc.StopPending {
				status = svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
				changes <- status
			}
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- status
			case svc.Stop, svc.Shutdown:
				status = svc.Status{State: svc.StopPending, WaitHint: 30000, CheckPoint: 1}
				changes <- status
				cancel()
			}
		case <-ticker.C:
			if status.State == svc.StartPending || status.State == svc.StopPending {
				status.CheckPoint++
				changes <- status
			}
		case err := <-finished:
			if err != nil {
				if log, openErr := eventlog.Open(buildinfo.ServiceName); openErr == nil {
					_ = log.Error(1, err.Error())
					_ = log.Close()
				}
				return true, 1
			}
			return false, 0
		}
	}
}
