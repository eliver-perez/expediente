// GUI launcher; native ACL work runs in a separate elevated, supervised process.
package main

import (
	"gestor-documental/internal/folderpicker"
	"gestor-documental/internal/launcher"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--control-center" {
		if err := runControlCenter(nil); err != nil {
			showMessage("No se pudo abrir el Centro de Control. Vuelve a ejecutar el instalador de AIDEX.")
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "--control-operation" {
		arguments, ok := controlOperation(os.Args[2:])
		if !ok {
			os.Exit(1)
		}
		if err := runControlCenter(arguments); err != nil {
			if process, ok := err.(*exec.ExitError); ok {
				os.Exit(process.ExitCode())
			}
			showMessage("No se pudo abrir la operación administrativa de AIDEX.")
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 3 && (os.Args[1] == "--folder-authorization" || os.Args[1] == "--folder-acl-worker") {
		if !folderTicket.MatchString("aidex-auth://authorize/" + os.Args[2]) {
			os.Exit(2)
		}
		var err error
		if os.Args[1] == "--folder-acl-worker" {
			closeLog := authorizationLog()
			defer closeLog()
			err = applyFolderAuthorization(os.Args[2])
		} else {
			err = superviseFolderAuthorization(os.Args[2])
			if err != nil {
				showMessage(err.Error())
			}
		}
		if err != nil {
			os.Exit(1)
		}
		return
	}
	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], "aidex-pick://") {
		if err := folderpicker.Run(os.Args[1]); err != nil {
			showMessage("No se pudo seleccionar la carpeta. Vuelve a AIDEX o introduce la ruta manualmente.")
		}
		return
	}

	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], "aidex-auth://") {
		if message := openFolderAuthorization(os.Args[1]); message != "" {
			showMessage(message)
			os.Exit(1)
		}
		return
	}
	if message := openApplication(); message != "" {
		showMessage(message)
		os.Exit(1)
	}
}

func controlOperation(input []string) ([]string, bool) {
	if len(input) == 1 {
		switch input[0] {
		case "Start", "Stop", "Restart", "Setup":
			return []string{"-Operation", input[0]}, true
		}
	}
	if len(input) == 2 && input[0] == "Port" {
		if port, err := strconv.Atoi(input[1]); err == nil && port >= 1 && port <= 65535 {
			return []string{"-Operation", "Port", "-Port", strconv.Itoa(port)}, true
		}
	}
	return nil, false
}

func runControlCenter(operation []string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return err
	}
	args := []string{"-NoProfile", "-STA", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(filepath.Dir(executable), "control-center.ps1")}
	cmd := exec.Command(filepath.Join(system, "WindowsPowerShell", "v1.0", "powershell.exe"), append(args, operation...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	if len(operation) == 0 {
		// The resident form doesn't need a launcher process holding AIDEX.exe open.
		if err := cmd.Start(); err != nil {
			return err
		}
		return cmd.Process.Release()
	}
	return cmd.Run()
}

func showMessage(message string) {
	text, _ := windows.UTF16PtrFromString(message)
	caption, _ := windows.UTF16PtrFromString("AIDEX")
	_, _ = windows.MessageBox(0, text, caption, windows.MB_OK|windows.MB_ICONINFORMATION)
}

var folderTicket = regexp.MustCompile(`^aidex-auth://authorize/([a-f0-9]{64})/?$`)

func openFolderAuthorization(value string) string {
	parts := folderTicket.FindStringSubmatch(value)
	if parts == nil {
		return "La solicitud de autorización de carpeta no es válida."
	}
	if err := elevateAuthorization(parts[1]); err != nil {
		return "La autorización se canceló o Windows no pudo abrirla. Vuelve a AIDEX y comprueba el acceso."
	}
	return ""
}

func openApplication() string {
	executable, err := os.Executable()
	if err != nil {
		return "No se pudo localizar AIDEX. Vuelve a ejecutar el instalador."
	}
	publicPath := filepath.Join(filepath.Dir(executable), "launcher.json")
	if commonData := os.Getenv("ProgramData"); commonData != "" {
		candidate := filepath.Join(commonData, "AIDEX", "connection.json")
		if _, statErr := os.Lstat(candidate); statErr == nil {
			publicPath = candidate
		}
	}
	file, err := os.Open(publicPath)
	if err != nil {
		return "Falta la dirección de AIDEX. Ejecuta «Configurar AIDEX» desde Inicio."
	}
	target, serviceName, err := launcher.ReadInstallation(file)
	file.Close()
	if err != nil {
		return "La dirección de AIDEX no es válida. Ejecuta «Configurar AIDEX» desde Inicio."
	}
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return "No se pudo consultar el servicio de AIDEX. Solicita ayuda al administrador de este equipo."
	}
	defer windows.CloseServiceHandle(manager)
	name, _ := windows.UTF16PtrFromString(serviceName)
	service, err := windows.OpenService(manager, name, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return "El servicio de AIDEX no está disponible. Ejecuta «Configurar AIDEX» desde Inicio."
	}
	defer windows.CloseServiceHandle(service)
	deadline := time.Now().Add(15 * time.Second)
	for {
		var status windows.SERVICE_STATUS
		if err := windows.QueryServiceStatus(service, &status); err != nil {
			return "No se pudo consultar el servicio de AIDEX. Solicita ayuda al administrador de este equipo."
		}
		if status.CurrentState == windows.SERVICE_RUNNING {
			break
		}
		if status.CurrentState != windows.SERVICE_START_PENDING || time.Now().After(deadline) {
			return "AIDEX todavía no está iniciado.\n\nSi acabas de encender la PC, espera un momento y vuelve a abrirlo.\n\nSi es la primera vez o el problema continúa, ejecuta «Configurar AIDEX» desde Inicio o solicita ayuda al administrador."
		}
		time.Sleep(250 * time.Millisecond)
	}
	verb, _ := windows.UTF16PtrFromString("open")
	address, _ := windows.UTF16PtrFromString(target)
	if err := windows.ShellExecute(0, verb, address, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return "No se pudo abrir el navegador. Abre tu navegador y visita:\n" + target
	}
	return ""
}
