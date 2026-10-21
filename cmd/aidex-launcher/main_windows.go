// A GUI-subsystem Windows executable: no console, elevation, credentials or writes.
package main

import (
	"gestor-documental/internal/launcher"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if message := openApplication(); message != "" {
		text, _ := windows.UTF16PtrFromString(message)
		caption, _ := windows.UTF16PtrFromString("AIDEX")
		_, _ = windows.MessageBox(0, text, caption, windows.MB_OK|windows.MB_ICONINFORMATION)
		os.Exit(1)
	}
}

func openApplication() string {
	executable, err := os.Executable()
	if err != nil {
		return "No se pudo localizar AIDEX. Vuelve a ejecutar el instalador."
	}
	file, err := os.Open(filepath.Join(filepath.Dir(executable), "launcher.json"))
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
