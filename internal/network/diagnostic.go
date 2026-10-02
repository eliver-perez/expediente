package network

import (
	"context"
	"net"
	"runtime"
	"strconv"
	"time"
)

type Diagnostic struct {
	Mode            string      `json:"mode"`
	ListenAddress   string      `json:"listen_address"`
	Port            int         `json:"port"`
	Revision        string      `json:"revision"`
	Editable        bool        `json:"editable"`
	Listening       bool        `json:"listening"`
	LocalURL        string      `json:"local_url"`
	Interfaces      []Interface `json:"interfaces"`
	InterfacesError bool        `json:"interfaces_error"`
	Firewall        Firewall    `json:"firewall"`
	Platform        string      `json:"platform"`
}

func (s *Service) Diagnose(ctx context.Context) Diagnostic {
	s.mu.Lock()
	configuration := s.configuration
	running := s.listener != nil && !s.closed
	s.mu.Unlock()
	_, port, _ := net.SplitHostPort(configuration.ListenAddress)
	number, _ := strconv.Atoi(port)
	available, err := interfaces(port)
	result := Diagnostic{Mode: mode(configuration), ListenAddress: configuration.ListenAddress, Port: number,
		Revision: revision(configuration), Editable: mode(configuration) != "custom", LocalURL: configuration.PublicURL,
		Interfaces: available, InterfacesError: err != nil, Platform: runtime.GOOS}
	if running {
		target := configuration.ListenAddress
		if result.Editable {
			target = net.JoinHostPort("127.0.0.1", port)
		}
		connection, dialErr := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", target)
		if dialErr == nil {
			result.Listening = true
			_ = connection.Close()
		}
	}
	result.Firewall = inspectFirewall(ctx, port)
	return result
}
