package network

import (
	"context"
	"encoding/json"
)

func inspectFirewall(ctx context.Context, port string) Firewall {
	result := unknownFirewall()
	path := commandPath("powershell.exe")
	if path != "" {
		output, err := probe(ctx, path, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference='Stop'; @(Get-NetFirewallProfile | Select-Object Name,@{Name='Active';Expression={$_.Enabled -eq 'True'}}) | ConvertTo-Json -Compress")
		var profiles []struct {
			Name   string
			Active bool
		}
		if err == nil && json.Unmarshal([]byte(output), &profiles) == nil {
			for _, profile := range profiles {
				state := "deshabilitado"
				if profile.Active {
					state = "habilitado"
				}
				result.Checks = append(result.Checks, "Windows Defender Firewall · "+profile.Name+": "+state+".")
			}
		}
	}
	result.Instructions = "En Seguridad de Windows > Firewall y protección de red > Configuración avanzada, pide al administrador revisar las reglas de entrada del ejecutable de AIBID y TCP " + port + " para el perfil de dominio o privado que corresponda, limitado a tu subred. Las reglas de bloqueo y las políticas del dominio pueden prevalecer. AIBID no ha creado ni modificado reglas."
	return result
}
