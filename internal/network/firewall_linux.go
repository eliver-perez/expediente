package network

import (
	"context"
	"strings"
)

func inspectFirewall(ctx context.Context, port string) Firewall {
	result := unknownFirewall()
	for _, tool := range []string{"ufw", "firewall-cmd", "nft", "iptables"} {
		path := commandPath(tool)
		if path == "" {
			continue
		}
		check := tool + ": disponible; no se pudo consultar con los permisos del servicio."
		switch tool {
		case "ufw":
			output, err := probe(ctx, path, "status")
			if err == nil {
				if strings.Contains(output, "Status: active") {
					check = "UFW: activo. Es necesario revisar sus reglas de entrada."
				}
				if strings.Contains(output, "Status: inactive") {
					check = "UFW: inactivo. Otros filtros pueden seguir activos."
				}
			}
		case "firewall-cmd":
			output, err := probe(ctx, path, "--state")
			if err == nil && strings.TrimSpace(output) == "running" {
				check = "firewalld: activo. Es necesario revisar la zona de la interfaz de red."
			}
		case "nft", "iptables":
			check = tool + ": disponible. Sus reglas requieren revisión por un administrador."
		}
		result.Checks = append(result.Checks, check)
	}
	result.Instructions = "Pide al administrador que revise el firewall detectado y permita TCP al puerto " + port + " únicamente desde tu red local. No basta con que la herramienta esté instalada: deben revisarse sus reglas y la interfaz o zona activa. AIBID no ha creado ni modificado reglas."
	return result
}
