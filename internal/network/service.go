// Package network owns the HTTP listener and read-only operating system diagnostics.
package network

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
)

type Service struct {
	mu            sync.Mutex
	configuration config.Config
	path          string
	factory       func(config.Config) http.Handler
	listener      net.Listener
	server        *http.Server
	errors        chan error
	closed        bool
	draining      map[*http.Server]bool
}

func New(path string, configuration config.Config) *Service {
	return &Service{path: path, configuration: configuration, errors: make(chan error, 1), draining: make(map[*http.Server]bool)}
}

func (s *Service) Start(factory func(config.Config) http.Handler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil || s.closed {
		return fmt.Errorf("network service already started or closed")
	}
	if err := s.configuration.Validate(); err != nil {
		return err
	}
	if s.configuration.TLSCertificate != "" {
		if _, err := tls.LoadX509KeyPair(s.configuration.TLSCertificate, s.configuration.TLSPrivateKey); err != nil {
			return err
		}
	}
	listener, err := listen(s.configuration)
	if err != nil {
		return err
	}
	s.factory = factory
	s.serve(listener, s.configuration)
	return nil
}

func (s *Service) serve(listener net.Listener, configuration config.Config) {
	server := &http.Server{Handler: s.factory(configuration), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	s.server, s.listener = server, listener
	go func() {
		var err error
		if configuration.TLSCertificate != "" {
			err = server.ServeTLS(listener, configuration.TLSCertificate, configuration.TLSPrivateKey)
		} else {
			err = server.Serve(listener)
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			select {
			case s.errors <- err:
			default:
			}
		}
	}()
}

func (s *Service) Errors() <-chan error { return s.errors }

func listen(configuration config.Config) (net.Listener, error) {
	protocol := "tcp"
	if mode(configuration) != "custom" {
		// tcp may promote a wildcard IPv4 address to a dual-stack IPv6 socket.
		// The local/LAN setting deliberately controls IPv4 only on every OS.
		protocol = "tcp4"
	}
	if mode(configuration) == "lan" {
		if err := checkLANPort(configuration.ListenAddress); err != nil {
			return nil, err
		}
	}
	return net.Listen(protocol, configuration.ListenAddress)
}

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

func mode(configuration config.Config) string {
	if configuration.NetworkMode != "" {
		return configuration.NetworkMode
	}
	host, port, _ := net.SplitHostPort(configuration.ListenAddress)
	if host == "127.0.0.1" && configuration.PublicURL == "http://127.0.0.1:"+port && configuration.TLSCertificate == "" && len(configuration.TrustedProxies) == 0 {
		return "local"
	}
	return "custom"
}

func revision(configuration config.Config) string {
	return domain.Digest(configuration.NetworkMode + "\n" + configuration.ListenAddress + "\n" + configuration.PublicURL)
}

func (s *Service) LocalOnly() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mode(s.configuration) == "local"
}

// Apply changes only the listener. Indexing, sessions and license jobs continue.
// The requested port is deliberately fixed for 1.0, preserving installed shortcuts.
func (s *Service) Apply(selected, expectedRevision string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.listener == nil {
		return domain.Failure("NETWORK_UNAVAILABLE", "El servicio de red no está disponible.", 503)
	}
	if selected != "local" && selected != "lan" {
		return domain.Failure("INVALID_REQUEST", "Selecciona acceso local o desde la red.", 422)
	}
	if mode(s.configuration) == "custom" {
		return domain.Failure("NETWORK_CUSTOM", "Esta instalación utiliza una configuración HTTPS o de red personalizada. Se conserva su configuración actual.", 409)
	}
	if expectedRevision != revision(s.configuration) {
		return domain.Failure("REVISION_CONFLICT", "La configuración cambió. Actualiza el diagnóstico antes de guardar.", 409)
	}
	disk, err := config.Load(s.path)
	if err != nil {
		return domain.Failure("NETWORK_CONFIG_READ", "No se pudo leer la configuración privada. Se conserva el acceso actual.", 500)
	}
	if !reflect.DeepEqual(disk, s.configuration) {
		return domain.Failure("REVISION_CONFLICT", "El archivo de configuración cambió fuera de AIBID. Reinicia el servicio para cargarlo.", 409)
	}
	if mode(s.configuration) == selected {
		return nil
	}
	old := s.configuration
	next := old
	next.NetworkMode = selected
	_, port, _ := net.SplitHostPort(old.ListenAddress)
	host := "127.0.0.1"
	if selected == "lan" {
		host = "0.0.0.0"
	}
	next.ListenAddress = net.JoinHostPort(host, port)
	if err := next.Validate(); err != nil {
		return err
	}

	// Wildcard and loopback listeners cannot own the same port together. Close
	// only acceptance, bind the candidate, then persist. Roll back on either
	// failure; accepted requests drain after their response, including this PUT.
	previousServer := s.server
	_ = s.listener.Close()
	defer s.drain(previousServer)
	listener, err := listen(next)
	if err != nil {
		if restoreErr := s.restore(old); restoreErr != nil {
			return restoreErr
		}
		return domain.Failure("NETWORK_PORT_UNAVAILABLE", "El puerto no está disponible en todas las interfaces o faltan permisos. Se restauró el acceso anterior.", 409)
	}
	if err = config.Save(s.path, next); err != nil {
		_ = listener.Close()
		if restoreErr := s.restore(old); restoreErr != nil {
			return restoreErr
		}
		return domain.Failure("NETWORK_CONFIG_WRITE", "No se pudo guardar la configuración privada. Se restauró el acceso anterior; revisa los permisos del servicio.", 500)
	}
	s.configuration = next
	s.serve(listener, next)
	return nil
}

func (s *Service) restore(configuration config.Config) error {
	listener, err := listen(configuration)
	if err != nil {
		s.listener = nil
		// The disk still contains a valid previous configuration. Let the service
		// supervisor restart rather than leave a process claiming to be available.
		select {
		case s.errors <- err:
		default:
		}
		return domain.Failure("NETWORK_RESTART_REQUIRED", "Otro proceso ocupó el puerto durante el cambio. La configuración anterior se conserva; reinicia AIBID cuando el puerto esté libre.", 503)
	}
	s.serve(listener, configuration)
	return nil
}

func (s *Service) drain(server *http.Server) {
	s.draining[server] = true
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		if server.Shutdown(ctx) != nil {
			_ = server.Close()
		}
		s.mu.Lock()
		delete(s.draining, server)
		s.mu.Unlock()
	}()
}

func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	server := s.server
	servers := make([]*http.Server, 0, len(s.draining)+1)
	if server != nil {
		servers = append(servers, server)
	}
	for previous := range s.draining {
		servers = append(servers, previous)
	}
	s.mu.Unlock()
	var result error
	for _, current := range servers {
		if err := current.Shutdown(ctx); err != nil {
			_ = current.Close()
			result = err
		}
	}
	return result
}
