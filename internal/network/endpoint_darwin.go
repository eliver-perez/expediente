package network

import (
	"net/url"

	"gestor-documental/internal/macoscontrol"
)

func publishEndpoint(configPath, origin string) error {
	if endpoint, err := url.Parse(origin); err == nil {
		macoscontrol.Endpoint(configPath, endpoint.Host, origin)
	}
	// Publishing the local control view is best effort and does not change
	// whether an existing network reconfiguration can succeed on macOS.
	return nil
}
