package folderpicker

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"gestor-documental/internal/launcher"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// A loopback HTTP peer alone cannot distinguish a browser on the server from an
// SSH tunnel. Verify the local listener's PID against the installed SCM service
// before asking for a path; a client-side helper must never send client paths.
func verifyLocal(ctx context.Context, endpoint string) error {
	result := make(chan error, 1)
	go func() { result <- verifyWindowsListener(endpoint) }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func verifyWindowsListener(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	data, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if err != nil {
		return err
	}
	file, err := os.Open(filepath.Join(data, "AIDEX", "connection.json"))
	if err != nil {
		return err
	}
	target, _, err := launcher.ReadInstallation(file)
	file.Close()
	if err != nil {
		return err
	}
	public, err := url.Parse(target)
	if err != nil || public.Scheme != "http" || public.Port() != u.Port() {
		return fmt.Errorf("el puerto no pertenece a esta instalación")
	}
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(manager)
	name, _ := windows.UTF16PtrFromString("AIDEX")
	handle, err := windows.OpenService(manager, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(handle)
	service := &mgr.Service{Name: "AIDEX", Handle: handle}
	state, err := service.Query()
	if err != nil || state.State != svc.Running || state.ProcessId == 0 {
		return fmt.Errorf("el servicio local no está iniciado")
	}
	settings, err := service.Config()
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	expected := `"` + filepath.Join(filepath.Dir(self), "aidex-server.exe") + `" serve --config "` + filepath.Join(data, "AIDEX", "config.json") + `" --windows-service AIDEX`
	if !strings.EqualFold(settings.ServiceStartName, `NT SERVICE\AIDEX`) || !strings.EqualFold(settings.BinaryPathName, expected) {
		return fmt.Errorf("no coincide la instalación local")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return err
	}
	table := windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")
	var size uint32
	code, _, _ := table.Call(0, uintptr(unsafe.Pointer(&size)), 0, windows.AF_INET, 3, 0)
	if code != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) || size < 4 || size > 16*1024*1024 {
		return fmt.Errorf("no se pudo comprobar el listener local")
	}
	buffer := make([]byte, size)
	code, _, _ = table.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)), 0, windows.AF_INET, 3, 0)
	if code != 0 || int(size) > len(buffer) || !ownsWindowsListener(buffer[:size], port, state.ProcessId) {
		return fmt.Errorf("la conexión no pertenece al servicio local")
	}
	return nil
}
