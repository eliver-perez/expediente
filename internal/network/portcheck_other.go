//go:build !windows

package network

import (
	"context"
	"fmt"
	"net"
	"time"
)

// BSD may allow a wildcard listener beside a listener bound to a specific IP.
// Binding alone therefore cannot establish that every advertised URL is ours.
// Probe only this machine's addresses, on our configured port, before binding.
func checkLANPort(address string) error {
	_, port, _ := net.SplitHostPort(address)
	available, err := interfaces(port)
	if err != nil {
		return err
	}
	available = append(available, Interface{Address: "127.0.0.1"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, device := range available {
		connection, err := (&net.Dialer{Timeout: 250 * time.Millisecond}).DialContext(ctx, "tcp4", net.JoinHostPort(device.Address, port))
		if err == nil {
			_ = connection.Close()
			return fmt.Errorf("configured port already accepts connections on %s", device.Address)
		}
		if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
			return fmt.Errorf("could not verify configured port availability")
		}
	}
	return nil
}
