package network

import (
	"errors"
	"net"
	"testing"
)

// Runs on the Windows CI runner without admin privileges. No firewall changes.
func TestWindowsNativePortInspection(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	address := net.JoinHostPort("0.0.0.0", port)
	if err := checkLANPort(address); !errors.Is(err, errPortOccupied) {
		t.Fatal("occupied local port not detected", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := checkLANPort(address); err != nil {
		t.Fatal("closed port incorrectly rejected", err)
	}
}
