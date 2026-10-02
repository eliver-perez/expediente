package network

import (
	"context"
	"os"
	"strings"
)

func inspectFirewall(ctx context.Context, port string) Firewall {
	result := unknownFirewall()
	const firewall = "/usr/libexec/ApplicationFirewall/socketfilterfw"
	output, err := probe(ctx, firewall, "--getglobalstate")
	if err == nil {
		if strings.Contains(output, "State = 0") {
			result.Checks = append(result.Checks, "Application Firewall: deshabilitado. Otros filtros pueden seguir activos.")
		}
		if strings.Contains(output, "State = 1") {
			result.Checks = append(result.Checks, "Application Firewall: habilitado.")
			binary, _ := os.Executable()
			blocked, checkErr := probe(ctx, firewall, "--getappblocked", binary)
			if checkErr == nil && strings.Contains(blocked, "Block incoming connections") {
				result.State = "blocked"
				result.Message = "Application Firewall informa que AIBID tiene bloqueadas las conexiones entrantes."
			}
			if checkErr == nil && strings.Contains(blocked, "Allow incoming connections") {
				result.Checks = append(result.Checks, "Application Firewall permite el ejecutable. No confirma la conectividad desde otro equipo.")
			}
		}
	}
	result.Instructions = "En Ajustes del Sistema > Red > Firewall > Opciones, revisa que el ejecutable gestor-documental de AIBID pueda recibir conexiones entrantes al puerto " + port + ". Puede requerir autorización de administrador. AIBID no ha creado ni modificado reglas."
	return result
}
