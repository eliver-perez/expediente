package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gestor-documental/internal/extraction"
	"gestor-documental/internal/storage"
)

type Config struct {
	Indexing             extraction.Options `json:"indexing"`
	ListenAddress        string             `json:"listen_address"`
	PublicURL            string             `json:"public_url"`
	StateDirectory       string             `json:"state_directory"`
	UploadDirectory      string             `json:"upload_directory,omitempty"`
	TLSCertificate       string             `json:"tls_certificate"`
	TLSPrivateKey        string             `json:"tls_private_key"`
	TrustedProxies       []string           `json:"trusted_proxies"`
	SessionIdleMinutes   int                `json:"session_idle_minutes"`
	SessionAbsoluteHours int                `json:"session_absolute_hours"`
	LoginWindowMinutes   int                `json:"login_window_minutes"`
	LoginIdentifierLimit int                `json:"login_identifier_limit"`
	LoginIPLimit         int                `json:"login_ip_limit"`
}

func DefaultPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "gestor-documental", "config.json"), nil
}

func Defaults(stateDirectory string) Config {
	return Config{Indexing: extraction.Defaults(), ListenAddress: "127.0.0.1:8090", PublicURL: "http://127.0.0.1:8090", StateDirectory: stateDirectory,
		TrustedProxies: []string{}, SessionIdleMinutes: 30, SessionAbsoluteHours: 12,
		LoginWindowMinutes: 15, LoginIdentifierLimit: 5, LoginIPLimit: 30}
}

func Initialize(path string) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := storage.PreparePrivateDirectory(filepath.Dir(absolutePath)); err != nil {
		return err
	}
	configuration := Defaults(filepath.Join(filepath.Dir(absolutePath), "state"))
	if err := configuration.Validate(); err != nil {
		return err
	}
	file, err := os.OpenFile(absolutePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("configuration already exists or is not writable: %w", err)
	}
	defer file.Close()
	if err := storage.ProtectPrivatePath(absolutePath, false); err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(configuration); err != nil {
		return err
	}
	return file.Sync()
}

func Load(path string) (Config, error) {
	configuration := Defaults("")
	info, err := os.Lstat(path)
	if err != nil {
		return configuration, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return configuration, fmt.Errorf("configuration must be a regular private file")
	}
	file, err := os.Open(path)
	if err != nil {
		return configuration, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 65537))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return configuration, fmt.Errorf("invalid configuration: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return configuration, fmt.Errorf("configuration must contain one JSON object")
	}
	return configuration, configuration.Validate()
}

func (configuration Config) Validate() error {
	if err := configuration.Indexing.Validate(); err != nil {
		return err
	}
	host, port, err := net.SplitHostPort(configuration.ListenAddress)
	if err != nil || port == "0" || port == "" {
		return fmt.Errorf("listen_address must contain an explicit IP and port")
	}
	listenIP, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("listen_address must use an IP address")
	}
	publicURL, err := url.Parse(configuration.PublicURL)
	if err != nil || publicURL.Host == "" || publicURL.User != nil || publicURL.Path != "" || publicURL.RawQuery != "" || publicURL.Fragment != "" {
		return fmt.Errorf("public_url must be an origin without path or credentials")
	}
	if publicURL.Scheme != "https" {
		publicIP, parseError := netip.ParseAddr(publicURL.Hostname())
		if publicURL.Scheme != "http" || !listenIP.IsLoopback() || (publicURL.Hostname() != "localhost" && (parseError != nil || !publicIP.IsLoopback())) {
			return fmt.Errorf("LAN access requires HTTPS; HTTP is restricted to loopback")
		}
	}
	if (configuration.TLSCertificate == "") != (configuration.TLSPrivateKey == "") {
		return fmt.Errorf("TLS needs both certificate and private key")
	}
	if configuration.TLSCertificate != "" && publicURL.Scheme != "https" {
		return fmt.Errorf("direct TLS requires an HTTPS public URL")
	}
	if publicURL.Scheme == "https" && configuration.TLSCertificate == "" && len(configuration.TrustedProxies) == 0 {
		return fmt.Errorf("HTTPS requires direct TLS or explicit trusted proxy CIDRs")
	}
	for _, proxy := range configuration.TrustedProxies {
		if _, err := netip.ParsePrefix(proxy); err != nil {
			return fmt.Errorf("invalid trusted proxy CIDR")
		}
	}
	if !filepath.IsAbs(configuration.StateDirectory) || filepath.Clean(configuration.StateDirectory) == string(filepath.Separator) {
		return fmt.Errorf("state_directory must be an absolute private directory")
	}
	if configuration.UploadDirectory != "" && (!filepath.IsAbs(configuration.UploadDirectory) || filepath.Clean(configuration.UploadDirectory) == string(filepath.Separator)) {
		return fmt.Errorf("upload_directory must be an absolute private local directory")
	}
	// Explicitly exclude common web roots; an installer cannot discover every custom server root.
	for _, component := range strings.Split(filepath.ToSlash(configuration.StateDirectory)+"/"+filepath.ToSlash(configuration.UploadDirectory), "/") {
		if strings.EqualFold(component, "htdocs") || strings.EqualFold(component, "wwwroot") {
			return fmt.Errorf("state_directory cannot be inside a public web root")
		}
	}
	if configuration.SessionIdleMinutes < 1 || configuration.SessionIdleMinutes > 1440 || configuration.SessionAbsoluteHours < 1 || configuration.SessionAbsoluteHours > 168 {
		return fmt.Errorf("session timeouts outside allowed range")
	}
	if configuration.LoginWindowMinutes < 1 || configuration.LoginWindowMinutes > 60 || configuration.LoginIdentifierLimit < 1 || configuration.LoginIdentifierLimit > 100 || configuration.LoginIPLimit < 1 || configuration.LoginIPLimit > 1000 {
		return fmt.Errorf("login rate limits outside allowed range")
	}
	return nil
}

func (configuration Config) PrivateUploadDirectory() string {
	if configuration.UploadDirectory != "" {
		return configuration.UploadDirectory
	}
	return filepath.Join(configuration.StateDirectory, "uploads")
}
