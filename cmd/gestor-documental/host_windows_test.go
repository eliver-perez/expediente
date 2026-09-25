package main

import (
	"context"
	"golang.org/x/sys/windows/svc"
	"testing"
	"time"
)

func TestWindowsServiceReadinessAndStop(t *testing.T) {
	requests := make(chan svc.ChangeRequest)
	changes := make(chan svc.Status, 8)
	finished := make(chan uint32, 1)
	allowReady := make(chan struct{})
	service := &windowsService{application: func(ctx context.Context, ready func()) error {
		<-allowReady
		ready()
		<-ctx.Done()
		return nil
	}}
	go func() { _, code := service.Execute(nil, requests, changes); finished <- code }()
	next := func() svc.Status {
		t.Helper()
		select {
		case status := <-changes:
			return status
		case <-time.After(5 * time.Second):
			t.Fatal("SCM status timed out")
			return svc.Status{}
		}
	}
	if status := next(); status.State != svc.StartPending || status.Accepts != 0 {
		t.Fatal(status)
	}
	select {
	case status := <-changes:
		t.Fatalf("reported before application readiness: %+v", status)
	default:
	}
	close(allowReady)
	if status := next(); status.State != svc.Running || status.Accepts&svc.AcceptStop == 0 {
		t.Fatal(status)
	}
	requests <- svc.ChangeRequest{Cmd: svc.Stop}
	if status := next(); status.State != svc.StopPending {
		t.Fatal(status)
	}
	select {
	case code := <-finished:
		if code != 0 {
			t.Fatal(code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("application not cancelled")
	}
}
