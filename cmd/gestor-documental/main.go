package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gestor-documental/internal/buildinfo"
	"gestor-documental/internal/config"
	"gestor-documental/internal/domain"
	"gestor-documental/internal/httpapi"
	"gestor-documental/internal/identity"
	"gestor-documental/internal/libraries"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
	"golang.org/x/term"
)

func main() {
	if err := runHost(run); err != nil {
		var failure *domain.Error
		if errors.As(err, &failure) {
			fmt.Fprintln(os.Stderr, "Error:", failure.Message)
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, ready func()) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("uso: gestor-documental version|init|doctor|bootstrap|recover-admin|recover-license|migrate|rollback-empty|serve [--config RUTA]")
	}
	command := os.Args[1]
	if command == "version" {
		fmt.Printf("AIBID %s (%s; %s/%s)\n", buildinfo.Version, buildinfo.Channel, runtime.GOOS, runtime.GOARCH)
		return nil
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	recoveryReason := flags.String("reason", "", "Motivo de recuperación de identidad de licencia")
	confirmRecovery := flags.Bool("confirm-license-recovery", false, "Confirmar nueva identidad; requiere nueva activación del proveedor")
	configurationPath := flags.String("config", "", "Archivo privado de configuración (por defecto, el del usuario)")
	initialState := flags.String("state", "", "Directorio de datos (solo init)")
	initialListen := flags.String("listen", "", "IP:puerto inicial (solo init; HTTP local)")
	initialTools := flags.String("tools", "", "Directorio absoluto de herramientas PDF/OCR (solo init)")
	initialTessdata := flags.String("tessdata", "", "Directorio absoluto de idiomas OCR (solo init)")
	initialFontconfig := flags.String("fontconfig", "", "Archivo absoluto de fuentes (solo init)")
	samples := flags.String("sample-directory", "", "Directorio de PDF de diagnóstico (solo doctor)")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("argumentos no reconocidos")
	}
	// System accounts may have no HOME/APPDATA. An explicit installer path must
	// work without consulting a user profile that the service does not possess.
	if *configurationPath == "" {
		defaultPath, err := config.DefaultPath()
		if err != nil {
			return err
		}
		*configurationPath = defaultPath
	}
	if *samples != "" && command != "doctor" {
		return fmt.Errorf("sample-directory only applies to doctor")
	}
	if command == "init" {
		if *initialTools != "" && !filepath.IsAbs(*initialTools) {
			return fmt.Errorf("tools must be absolute")
		}
		if err := config.InitializeWith(*configurationPath, func(initial *config.Config) {
			if *initialState != "" {
				initial.StateDirectory = *initialState
			}
			if *initialListen != "" {
				initial.ListenAddress = *initialListen
				initial.PublicURL = "http://" + *initialListen
			}
			if *initialTools != "" {
				extension := ""
				if runtime.GOOS == "windows" {
					extension = ".exe"
				}
				initial.Indexing.PDFInfo = filepath.Join(*initialTools, "pdfinfo"+extension)
				initial.Indexing.PDFText = filepath.Join(*initialTools, "pdftotext"+extension)
				initial.Indexing.PDFRender = filepath.Join(*initialTools, "pdftoppm"+extension)
				initial.Indexing.Tesseract = filepath.Join(*initialTools, "tesseract"+extension)
			}
			initial.Indexing.TessdataDirectory = *initialTessdata
			initial.Indexing.FontconfigFile = *initialFontconfig
		}); err != nil {
			return err
		}
		fmt.Println("Configuración privada creada. Ejecuta bootstrap para crear el primer administrador.")
		return nil
	}
	if *initialState != "" || *initialListen != "" || *initialTools != "" || *initialTessdata != "" || *initialFontconfig != "" {
		return fmt.Errorf("state/listen/tools/tessdata only apply to init")
	}
	if command != "serve" && command != "doctor" && command != "bootstrap-ready" && command != "bootstrap" && command != "recover-admin" && command != "migrate" && command != "rollback-empty" && command != "recover-license" {
		return fmt.Errorf("comando no reconocido")
	}
	configuration, err := config.Load(*configurationPath)
	if err != nil {
		return err
	}
	if command == "doctor" {
		return doctor(ctx, configuration, *samples)
	}
	unlock, err := storage.LockState(configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer unlock()
	database, err := storage.Open(ctx, configuration.StateDirectory)
	if err != nil {
		return err
	}
	defer database.Close()
	if command == "bootstrap-ready" {
		var count int
		if err := database.Reader.QueryRowContext(ctx, "SELECT count(*) FROM bootstrap_state").Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("falta crear el primer administrador")
		}
		return nil
	}
	if command == "rollback-empty" {
		return database.RollbackEmpty(ctx)
	}
	if command == "migrate" {
		fmt.Println("Migraciones verificadas y aplicadas.")
		return nil
	}
	if command == "recover-license" {
		if !*confirmRecovery {
			return fmt.Errorf("recover-license requires --confirm-license-recovery and --reason; it does not release the commercial activation")
		}
		if err = licensing.RecoverIdentity(ctx, database, configuration.StateDirectory, *recoveryReason); err != nil {
			return err
		}
		fmt.Println("Identidad recuperada y auditada. Solicita al proveedor la transferencia y una nueva activación.")
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
	termination, stop := context.WithCancel(ctx)
	defer stop()
	licenseDone := make(chan struct{})
	go func() { defer close(licenseDone); service.License.Run(termination) }()
	defer func() { stop(); <-licenseDone }()
	serverError := make(chan error, 1)
	listener, err := net.Listen("tcp", configuration.ListenAddress)
	if err != nil {
		return err
	}
	defer listener.Close()
	go func() {
		if configuration.TLSCertificate != "" {
			serverError <- server.ServeTLS(listener, configuration.TLSCertificate, configuration.TLSPrivateKey)
		} else {
			serverError <- server.Serve(listener)
		}
	}()
	logger.Info("service starting", "address", configuration.ListenAddress, "version", buildinfo.Version, "channel", buildinfo.Channel)
	ready()
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
