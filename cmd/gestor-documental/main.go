package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/httpapi"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/storage"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		var failure *domain.Error
		if errors.As(err, &failure) {
			fmt.Fprintln(os.Stderr, "Error:", failure.Message)
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("uso: gestor-documental init|bootstrap|recover-admin|migrate|rollback-empty|serve [--config RUTA]")
	}
	command := os.Args[1]
	defaultPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	configurationPath := flags.String("config", defaultPath, "Archivo privado de configuración")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("argumentos no reconocidos")
	}
	if command == "init" {
		if err := config.Initialize(*configurationPath); err != nil {
			return err
		}
		fmt.Println("Configuración privada creada. Ejecuta bootstrap para crear el primer administrador.")
		return nil
	}
	if command != "serve" && command != "bootstrap" && command != "recover-admin" && command != "migrate" && command != "rollback-empty" {
		return fmt.Errorf("comando no reconocido")
	}
	configuration, err := config.Load(*configurationPath)
	if err != nil {
		return err
	}
	unlock, err := storage.LockState(configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer unlock()
	ctx := context.Background()
	database, err := storage.Open(ctx, configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer database.Close()
	if command == "rollback-empty" {
		return database.RollbackEmpty(ctx)
	}
	if command == "migrate" {
		fmt.Println("Migraciones verificadas y aplicadas.")
		return nil
	}
	service, err := identity.New(database, configuration)
	if err != nil {
		return err
	}
	if command == "bootstrap" || command == "recover-admin" {
		username, displayName, password, err := readCredentials(command == "bootstrap")
		if err != nil {
			return err
		}
		if command == "bootstrap" {
			err = service.Bootstrap(ctx, username, displayName, password)
		} else {
			err = service.RecoverAdministrator(ctx, username, password)
		}
		if err != nil {
			return err
		}
		fmt.Println("Operación completada y registrada. Inicia sesión con la nueva contraseña.")
		return nil
	}
	if !httpapi.AssetsReady() {
		return fmt.Errorf("compila primero el frontend con npm --prefix web run build y vuelve a compilar Go")
	}
	var administratorReady int
	if err := database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM bootstrap_state").Scan(&administratorReady); err != nil {
		return err
	}
	if administratorReady == 0 {
		return fmt.Errorf("ejecuta bootstrap antes de iniciar el servicio")
	}
	background, err := libraries.New(service).Start(ctx)
	if err != nil {
		return err
	}
	defer background.Close()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	application := httpapi.New(service, configuration, logger)
	server := &http.Server{Addr: configuration.ListenAddress, Handler: application.Handler(), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	termination, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	go func() {
		if configuration.TLSCertificate != "" {
			serverError <- server.ListenAndServeTLS(configuration.TLSCertificate, configuration.TLSPrivateKey)
		} else {
			serverError <- server.ListenAndServe()
		}
	}()
	logger.Info("service starting", "address", configuration.ListenAddress, "stage", "H4")
	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-termination.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return err
		}
	}
	return nil
}

func readCredentials(includeName bool) (string, string, string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", "", "", fmt.Errorf("se requiere una terminal local interactiva; no se aceptan contraseñas por argumento o pipe")
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Usuario: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", "", err
	}
	displayName := ""
	if includeName {
		fmt.Print("Nombre visible: ")
		displayName, err = reader.ReadString('\n')
		if err != nil {
			return "", "", "", err
		}
	}
	fmt.Print("Nueva contraseña: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", "", "", err
	}
	fmt.Print("Repite la contraseña: ")
	confirmation, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", "", "", err
	}
	defer clear(password)
	defer clear(confirmation)
	if string(password) != string(confirmation) {
		return "", "", "", fmt.Errorf("las contraseñas no coinciden")
	}
	return strings.TrimSpace(username), strings.TrimSpace(displayName), string(password), nil
}
