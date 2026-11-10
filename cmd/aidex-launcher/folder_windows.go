package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"gestor-documental/internal/folderpicker"
	"gestor-documental/internal/windowsfolders"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func authorizationEndpoint(ticket string) (string, error) {
	if !folderTicket.MatchString("aidex-auth://authorize/" + ticket) {
		return "", fmt.Errorf("solicitud no válida")
	}
	data, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return "", authorizationFailure("SHGetKnownFolderPath", "ProgramData", err)
	}
	payload, err := os.ReadFile(filepath.Join(data, "AIDEX", "config.json"))
	if err != nil {
		return "", authorizationFailure("ReadFile(service_config)", filepath.Join(data, "AIDEX", "config.json"), err)
	}
	var config struct {
		Listen string `json:"listen_address"`
	}
	if err = json.Unmarshal(payload, &config); err != nil {
		return "", authorizationFailure("decode_service_config", filepath.Join(data, "AIDEX", "config.json"), err)
	}
	host, port, err := net.SplitHostPort(config.Listen)
	if err != nil || (host != "127.0.0.1" && host != "0.0.0.0") {
		return "", fmt.Errorf("dirección local no válida")
	}
	manager, err := mgr.Connect()
	if err != nil {
		return "", authorizationFailure("OpenSCManagerW", "SCM local", err)
	}
	defer manager.Disconnect()
	service, err := manager.OpenService("AIDEX")
	if err != nil {
		return "", authorizationFailure("OpenServiceW", "AIDEX", err)
	}
	defer service.Close()
	settings, err := service.Config()
	if err != nil {
		return "", authorizationFailure("QueryServiceConfigW", "AIDEX", err)
	}
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	expected := `"` + filepath.Join(filepath.Dir(self), "aidex-server.exe") + `" serve --config "` + filepath.Join(data, "AIDEX", "config.json") + `" --windows-service AIDEX`
	if !strings.EqualFold(settings.ServiceStartName, `NT SERVICE\AIDEX`) || !strings.EqualFold(settings.BinaryPathName, expected) {
		return "", fmt.Errorf("la identidad o instalación del servicio no coincide")
	}
	state, err := service.Query()
	if err != nil {
		return "", authorizationFailure("QueryServiceStatusEx", "AIDEX", err)
	}
	if state.State != svc.Running {
		return "", fmt.Errorf("el servicio AIDEX debe estar iniciado")
	}
	return "http://127.0.0.1:" + port + "/api/v1/local-folder-authorizations/" + ticket, nil
}

func authorizationLog() func() {
	data, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return func() {}
	}
	// The installed data directory already has a private, service/admin-only ACL.
	directory := filepath.Join(data, "AIDEX", "logs")
	if os.MkdirAll(directory, 0700) != nil {
		return func() {}
	}
	file, err := os.OpenFile(filepath.Join(directory, "folder-authorization.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return func() {}
	}
	role := "supervisor"
	if len(os.Args) > 1 && os.Args[1] == "--folder-acl-worker" {
		role = "worker"
	}
	slog.SetDefault(authorizationLogger(file, role))
	return func() { file.Close() }
}

// Both processes open the log for append with Go's FILE_SHARE_READ|WRITE.
// Neither requests DELETE. Do not broaden sharing in response to an ACL error.
func authorizationLogger(file *os.File, role string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(file, nil)).With("pid", os.Getpid(), "role", role)
}
func authorizationFailure(operation, resource string, err error) error {
	if err == nil {
		return nil
	}
	var existing *windowsfolders.OperationError
	if errors.As(err, &existing) {
		return err
	}
	return &windowsfolders.OperationError{Operation: operation, Resource: resource, Err: err}
}
func finishAuthorization(endpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var checked struct {
		Accessible bool   `json:"accessible"`
		Code       string `json:"code"`
	}
	if err := folderpicker.Call(ctx, endpoint+"/finish", struct{}{}, &checked); err != nil {
		return authorizationFailure("service_finish_http", "servicio local AIDEX", err)
	}
	slog.Info("folder authorization service check", "accessible", checked.Accessible, "code", checked.Code)
	if !checked.Accessible {
		return fmt.Errorf("el servicio aún no tiene acceso (%s)", checked.Code)
	}
	return nil
}
func applyFolderAuthorization(ticket string) (err error) {
	endpoint := ""
	claimed := false
	finishAttempted := false
	phase := "verify_elevation"
	defer func() {
		if err != nil {
			attrs := append([]any{"phase", phase}, windowsfolders.DiagnosticAttrs(err)...)
			slog.Error("folder authorization failed", attrs...)
			if claimed && !finishAttempted {
				if checkErr := finishAuthorization(endpoint); checkErr != nil {
					slog.Error("folder authorization failure check", windowsfolders.DiagnosticAttrs(checkErr)...)
				}
			}
		}
	}()
	elevated := windows.GetCurrentProcessToken().IsElevated()
	slog.Info("folder authorization worker", "elevated", elevated)
	if !elevated {
		return authorizationFailure("check_elevated_token", "auxiliar Windows", windows.ERROR_ELEVATION_REQUIRED)
	}
	phase = "verify_installation"
	endpoint, err = authorizationEndpoint(ticket)
	if err != nil {
		return authorizationFailure("verify_installed_service", "servicio AIDEX y configuración instalada", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	phase = "verify_local_listener"
	if err = folderpicker.VerifyLocalService(ctx, endpoint); err != nil {
		return authorizationFailure("verify_local_listener", "listener local AIDEX", err)
	}
	var selection struct {
		Path   string `json:"server_path"`
		Source string `json:"storage_source"`
	}
	phase = "claim_ticket"
	if err = folderpicker.Call(ctx, endpoint+"/claim", struct{}{}, &selection); err != nil {
		return authorizationFailure("claim_ticket_http", "servicio local AIDEX", err)
	}
	claimed = true
	phase = "apply_acl"
	err = windowsfolders.Authorize(selection.Path, selection.Source, func() (bool, error) {
		var checked struct {
			Accessible bool   `json:"accessible"`
			Code       string `json:"code"`
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if e := folderpicker.Call(ctx, endpoint+"/probe", struct{}{}, &checked); e != nil {
			return false, authorizationFailure("service_probe_http", selection.Path, e)
		}
		slog.Info("folder authorization service probe", "operation", "service_effective_access", "resource", selection.Path, "accessible", checked.Accessible, "code", checked.Code)
		if !checked.Accessible && checked.Code != "ROOT_PERMISSION_DENIED" {
			return false, authorizationFailure("service_effective_access", selection.Path, fmt.Errorf("diagnóstico %s", checked.Code))
		}
		return checked.Accessible, nil
	})
	if err != nil {
		return err
	}
	phase = "finish_service_check"
	finishAttempted = true
	return finishAuthorization(endpoint)
}

func superviseFolderAuthorization(ticket string) error {
	closeLog := authorizationLog()
	defer closeLog()
	slog.Info("folder authorization started", "timeout_seconds", 90)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, self, "--folder-acl-worker", ticket)
	command.WaitDelay = 5 * time.Second
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if err = command.Run(); err != nil {
		attrs := append([]any{"operation", "wait_acl_worker", "resource", "AIDEX.exe --folder-acl-worker", "timeout", ctx.Err() != nil}, windowsfolders.DiagnosticAttrs(err)...)
		if command.ProcessState != nil {
			attrs = append(attrs, "exit_code", command.ProcessState.ExitCode(), "worker_pid", command.ProcessState.Pid())
		}
		slog.Error("folder authorization worker failed", attrs...)

		return fmt.Errorf("No se completó la autorización. Comprueba el acceso o consulta el registro de autorización de AIDEX")
	}
	return nil
}

// ShellExecuteEx retains a process handle; UAC elevation starts the installed GUI
// executable, never a shell or a browser-provided command/path.
func elevateAuthorization(ticket string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(self)
	args, _ := windows.UTF16PtrFromString("--folder-authorization " + ticket)
	info := struct {
		Size       uint32
		Mask       uint32
		Window     uintptr
		Verb       *uint16
		File       *uint16
		Parameters *uint16
		Directory  *uint16
		Show       int32
		Instance   uintptr
		IDList     uintptr
		Class      *uint16
		ClassKey   uintptr
		HotKey     uint32
		Icon       uintptr
		Process    windows.Handle
	}{Mask: 0x40, Verb: verb, File: file, Parameters: args, Show: windows.SW_HIDE}
	info.Size = uint32(unsafe.Sizeof(info))
	result, _, callErr := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	if result == 0 {
		return callErr
	}
	if info.Process != 0 {
		windows.CloseHandle(info.Process)
	}
	return nil
}
