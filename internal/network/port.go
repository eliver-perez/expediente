package network

import (
	"fmt"
	"net"
	"strconv"

	"gestor-documental/internal/config"
)

// ConfigureStopped is for an administrator with the exclusive installation
// lock. It changes no library or license state and refuses custom TLS setups.
func ConfigureStopped(path string, port int) (config.Config, error) {
	old, err := config.Load(path)
	if err != nil {
		return old, err
	}
	if port < 1 || port > 65535 {
		return old, fmt.Errorf("el puerto debe estar entre 1 y 65535")
	}
	if mode(old) == "custom" {
		return old, fmt.Errorf("la configuración HTTPS personalizada requiere administración propia")
	}
	_, previous, _ := net.SplitHostPort(old.ListenAddress)
	if previous == strconv.Itoa(port) {
		return old, nil
	}
	next := old
	host, _, _ := net.SplitHostPort(old.ListenAddress)
	next.ListenAddress = net.JoinHostPort(host, strconv.Itoa(port))
	next.PublicURL = "http://127.0.0.1:" + strconv.Itoa(port)
	if err = next.Validate(); err != nil {
		return old, err
	}
	listener, err := listen(next)
	if err != nil {
		return old, fmt.Errorf("el puerto %d está ocupado o no disponible: %w", port, err)
	}
	_ = listener.Close()
	if err = config.Save(path, next); err != nil {
		return old, err
	}
	return next, nil
}
