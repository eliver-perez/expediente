//go:build e2e && development

// This executable is only for browser tests. It cannot enter a normal product build.
package main

import (
	"context"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/httpapi"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/licensefixture"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	directory, err := os.MkdirTemp("", "documental-browser-tests-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	directory, err = filepath.EvalSymlinks(directory)
	if err != nil {
		return err
	}
	mock, err := licensefixture.NewMock("")
	if err != nil {
		return err
	}
	var lostActivationResponse atomic.Bool
	licenseServer := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/activations" && !lostActivationResponse.Swap(true) {
			record := httptest.NewRecorder()
			mock.ServeHTTP(record, request)
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(503)
			_, _ = io.WriteString(writer, `{"error":{"code":"TEMPORARY_UNAVAILABLE","message":"E2E response loss after commit"}}`)
			return
		}
		mock.ServeHTTP(writer, request)
	}))
	defer licenseServer.Close()
	certificate := filepath.Join(directory, "mock.crt")
	if err = os.WriteFile(certificate, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: licenseServer.Certificate().Raw}), 0600); err != nil {
		return err
	}
	configuration := config.Defaults(directory)
	configuration.License.ServerURL = licenseServer.URL
	configuration.License.DevelopmentCAFile = certificate
	configuration.License.TrustedKeys = []licensing.TrustKey{licensefixture.TrustKey()}
	configuration.License.RefreshHours = 0
	configuration.ListenAddress = "127.0.0.1:8099"
	configuration.PublicURL = "http://127.0.0.1:8099"
	database, err := storage.Open(context.Background(), directory)
	if err != nil {
		return err
	}
	defer database.Close()
	service, err := identity.New(database, configuration)
	if err != nil {
		return err
	}
	if err := service.Bootstrap(context.Background(), "admin-e2e", "Administradora de pruebas", "Browser-fixture-password-2026"); err != nil {
		return err
	}
	login, err := service.Login(context.Background(), "admin-e2e", "Browser-fixture-password-2026", domain.RequestMetadata{RequestID: domain.NewID(), ObservedIP: "127.0.0.1"})
	if err != nil {
		return err
	}
	principal, err := service.Authenticate(context.Background(), login.SessionToken, false, domain.RequestMetadata{})
	if err != nil {
		return err
	}
	if _, err := service.SetGlobalRoles(context.Background(), principal, principal.User.ID, principal.User.Revision, []string{"installation_admin", "access_auditor"}, domain.RequestMetadata{RequestID: domain.NewID()}); err != nil {
		return err
	}
	runtime, err := libraries.New(service).Start(context.Background())
	if err != nil {
		return err
	}
	defer runtime.Close()
	server := &http.Server{Addr: configuration.ListenAddress, Handler: httpapi.New(service, configuration, slog.Default()).Handler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); server.Close() }()
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
