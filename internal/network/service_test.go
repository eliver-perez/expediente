package network

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gestor-documental/internal/config"
)

func fixture(t *testing.T) *Service {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reserved.Addr().String()
	_ = reserved.Close()
	configuration := config.Defaults(filepath.Join(directory, "state"))
	configuration.ListenAddress, configuration.PublicURL = address, "http://"+address
	path := filepath.Join(directory, "config.json")
	if err = config.InitializeWith(path, func(value *config.Config) { *value = configuration }); err != nil {
		t.Fatal(err)
	}
	service := New(path, configuration)
	if err = service.Start(func(active config.Config) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requested := r.URL.Query().Get("mode"); requested != "" {
				if err := service.Apply(requested, revision(active)); err != nil {
					http.Error(w, err.Error(), http.StatusConflict)
					return
				}
				_, _ = io.WriteString(w, requested)
				return
			}
			_, _ = io.WriteString(w, mode(active))
		})
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Close(ctx)
	})
	return service
}

// Change modes through an accepted HTTP connection, as the browser does. On
// Windows the old request is still ESTABLISHED while we inspect/rebind the port.
func TestModeChangeInsideLiveHTTPRequest(t *testing.T) {
	s := fixture(t)
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}
	defer client.CloseIdleConnections()
	for _, selected := range []string{"lan", "local", "lan", "local"} {
		response, err := client.Get(s.configuration.PublicURL + "?mode=" + selected)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != http.StatusOK || string(body) != selected {
			t.Fatal("mode-change response lost", response.StatusCode, string(body), err)
		}
		assertServing(t, s, selected)
		if !s.Diagnose(context.Background()).Listening {
			t.Fatal("replacement listener failed post-change diagnosis")
		}
	}
}

func assertServing(t *testing.T, service *Service, expected string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}
	defer client.CloseIdleConnections()
	response, err := client.Get(service.configuration.PublicURL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if string(body) != expected {
		t.Fatalf("listener = %q, want %q", body, expected)
	}
}

func TestLocalLANPersistsAndReturnsWithoutRestartingApplication(t *testing.T) {
	s := fixture(t)
	original := s.configuration
	assertServing(t, s, "local")
	for _, selected := range []string{"lan", "local", "lan"} {
		if err := s.Apply(selected, revision(s.configuration)); err != nil {
			t.Fatal(err)
		}
		assertServing(t, s, selected)
		disk, err := config.Load(s.path)
		if err != nil || mode(disk) != selected || disk.PublicURL != original.PublicURL || disk.StateDirectory != original.StateDirectory {
			t.Fatal("configuration not preserved", err)
		}
		host, _, _ := net.SplitHostPort(disk.ListenAddress)
		if selected == "local" && host != "127.0.0.1" || selected == "lan" && host != "0.0.0.0" {
			t.Fatal("wrong listener", host)
		}
	}
	if err := s.Apply("local", revision(original)); err == nil {
		t.Fatal("stale configuration accepted")
	}
	disk, err := config.Load(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	restarted := New(s.path, disk)
	if err = restarted.Start(s.factory); err != nil {
		t.Fatal(err)
	}
	defer restarted.Close(context.Background())
	assertServing(t, restarted, "lan")
}

func TestBindConflictRestoresLocalConfiguration(t *testing.T) {
	s := fixture(t)
	_, port, _ := net.SplitHostPort(s.configuration.ListenAddress)
	available, err := interfaces(port)
	if err != nil || len(available) == 0 {
		t.Skip("needs a non-loopback interface")
	}
	other, err := net.Listen("tcp4", net.JoinHostPort(available[0].Address, port))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	before, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Apply("lan", revision(s.configuration)); err == nil {
		t.Fatal("occupied port accepted")
	}
	after, _ := os.ReadFile(s.path)
	if string(after) != string(before) {
		t.Fatal("failed bind modified configuration")
	}
	assertServing(t, s, "local")
}

func TestExternalConfigurationIsNeverOverwritten(t *testing.T) {
	s := fixture(t)
	changed := s.configuration
	changed.SessionIdleMinutes++
	if err := config.Save(s.path, changed); err != nil {
		t.Fatal(err)
	}
	if err := s.Apply("lan", revision(s.configuration)); err == nil {
		t.Fatal("external configuration overwritten")
	}
	assertServing(t, s, "local")
	disk, err := config.Load(s.path)
	if err != nil || disk.SessionIdleMinutes != changed.SessionIdleMinutes {
		t.Fatal("external edit lost", err)
	}
}

func TestFailedPersistenceRestoresListenerAndOldFile(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix directory permissions for a non-root service")
	}
	s := fixture(t)
	before, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Dir(s.path)
	if err = os.Chmod(directory, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(directory, 0700)
	if err = s.Apply("lan", revision(s.configuration)); err == nil {
		t.Fatal("unwritable configuration accepted")
	}
	after, _ := os.ReadFile(s.path)
	if string(after) != string(before) {
		t.Fatal("failed persistence modified original")
	}
	assertServing(t, s, "local")
}

func TestHostAllowlistUsesActualIPv4InterfacesAndPort(t *testing.T) {
	for _, host := range []string{"attacker.example:18090", "0.0.0.0:18090", "127.0.0.1:18091", "127.0.0.2:18090", "[::1]:18090", "127.0.0.1", "192.0.2.254:18090", "127.0.0.1:018090"} {
		if AllowsHost(host, "0.0.0.0:18090") {
			t.Fatal("untrusted host", host)
		}
	}
	if !AllowsHost("127.0.0.1:18090", "0.0.0.0:18090") {
		t.Fatal("local host rejected")
	}
	available, _ := interfaces("18090")
	for _, item := range available {
		if !AllowsHost(net.JoinHostPort(item.Address, "18090"), "0.0.0.0:18090") {
			t.Fatal("assigned address rejected")
		}
	}
	if !virtualInterface("vEthernet (Default Switch)") || !virtualInterface("docker0") || !virtualInterface("utun2") || virtualInterface("enp0s1") {
		t.Fatal("VM primary interface incorrectly filtered")
	}
}
