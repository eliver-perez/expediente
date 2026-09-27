// Package launcher reads only public browser launch metadata, never the private
// service configuration, database, or license identity.
package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"net/url"
)

func ReadTarget(reader io.Reader) (string, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, 4097))
	if err != nil {
		return "", err
	}
	if len(contents) > 4096 {
		return "", fmt.Errorf("launch configuration too large")
	}
	// Windows PowerShell 5.1 writes a BOM for UTF-8; accept it explicitly.
	contents = bytes.TrimPrefix(contents, []byte{0xef, 0xbb, 0xbf})
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var target struct {
		PublicURL string `json:"public_url"`
	}
	if err := decoder.Decode(&target); err != nil {
		return "", err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", fmt.Errorf("expected one launch configuration")
	}
	address, err := url.Parse(target.PublicURL)
	if err != nil || address.Host == "" || address.User != nil || address.Opaque != "" || address.Path != "" || address.RawQuery != "" || address.ForceQuery || address.Fragment != "" {
		return "", fmt.Errorf("invalid browser origin")
	}
	switch address.Scheme {
	case "https":
	case "http":
		ip, err := netip.ParseAddr(address.Hostname())
		if address.Hostname() != "localhost" && (err != nil || !ip.IsLoopback()) {
			return "", fmt.Errorf("HTTP requires loopback")
		}
	default:
		return "", fmt.Errorf("unsupported browser scheme")
	}
	return address.String(), nil
}
