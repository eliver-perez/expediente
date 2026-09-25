package licensing

import (
	"crypto/ed25519"
	"fmt"
	"net/url"
	"strings"
)

type Options struct {
	ServerURL         string     `json:"server_url"`
	TrustedKeys       []TrustKey `json:"trusted_keys"`
	RefreshHours      int        `json:"refresh_hours"`
	DevelopmentBypass bool       `json:"development_bypass,omitempty"`
	DevelopmentCAFile string     `json:"development_ca_file,omitempty"`
}

func DefaultOptions() Options {
	return Options{TrustedKeys: []TrustKey{}, RefreshHours: 24, DevelopmentBypass: developmentEnabled}
}
func (options Options) Validate() error {
	if !developmentEnabled && (options.DevelopmentBypass || options.DevelopmentCAFile != "") {
		return fmt.Errorf("development licensing configuration is forbidden in production")
	}
	if options.RefreshHours < 0 || options.RefreshHours > 720 {
		return fmt.Errorf("license refresh_hours must be 0 (manual only) to 720")
	}
	if options.ServerURL != "" {
		address, err := url.Parse(options.ServerURL)
		if err != nil || address.Scheme != "https" || address.Host == "" || address.User != nil || address.RawQuery != "" || address.Fragment != "" || address.Path != "" && address.Path != "/" {
			return fmt.Errorf("license server_url must be an HTTPS origin")
		}
	}
	seen := map[string]bool{}
	for _, key := range options.TrustedKeys {
		public, err := decodeBase64(key.PublicKey, ed25519.PublicKeySize)
		if err != nil || !identifierPattern.MatchString(key.Kid) || seen[key.Kid] || key.Purpose != "license" || key.Environment != "production" && !(developmentEnabled && key.Environment == "development") {
			return fmt.Errorf("invalid license trust key or purpose/environment")
		}
		if !developmentEnabled && (strings.HasPrefix(strings.ToLower(key.Kid), "dev-") || fmt.Sprintf("%x", public) == developmentPublicKeyHex) {
			return fmt.Errorf("development signing keys are forbidden in production")
		}
		seen[key.Kid] = true
	}
	return nil
}
