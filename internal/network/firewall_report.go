package network

import (
	"encoding/json"
	"strings"
)

func describeWindowsFirewall(result Firewall, output, port, executable, serviceName string) Firewall {
	var report struct {
		Profiles []struct {
			Name                       string
			Enabled, LocalRulesBlocked bool
		}
		Rule *struct {
			Enabled, Allow, Inbound, EdgeBlocked               bool
			Profiles, Protocol, Port, Program, Service, Remote string
		}
		Categories []string
	}
	if json.Unmarshal([]byte(output), &report) != nil {
		return result
	}
	restricted := false
	for _, profile := range report.Profiles {
		if profile.Name != "Domain" && profile.Name != "Private" && profile.Name != "Public" {
			continue
		}
		state := "deshabilitado"
		if profile.Enabled {
			state = "habilitado"
		}
		result.Checks = append(result.Checks, "Windows Defender Firewall · "+profile.Name+": "+state+".")
		restricted = restricted || profile.Enabled && profile.LocalRulesBlocked && profile.Name != "Public"
	}
	rule := report.Rule
	if rule == nil {
		result.State = "missing"
		result.Message = "No se encontró la regla efectiva de AIBID. Vuelve a ejecutar el instalador actualizado para prepararla automáticamente; una política del dominio también puede impedir que se aplique."
		return result
	}
	profiles := strings.ReplaceAll(rule.Profiles, " ", "")
	if !rule.Enabled || !rule.Allow || !rule.Inbound || !rule.EdgeBlocked || (profiles != "Domain,Private" && profiles != "Private,Domain") || (rule.Protocol != "TCP" && rule.Protocol != "6") || rule.Port != port || !strings.EqualFold(rule.Program, executable) || serviceName == "" || !strings.EqualFold(rule.Service, serviceName) || !strings.EqualFold(rule.Remote, "LocalSubnet") {
		result.State = "mismatch"
		result.Message = "La regla de AIBID está deshabilitada o no coincide con esta instalación. Vuelve a ejecutar el instalador actualizado para repararla."
		return result
	}
	result.State = "managed"
	result.Message = "Regla de AIBID preparada para la subred local en redes privadas y de dominio."
	if restricted {
		result.State = "policy_restricted"
		result.Message = "Windows informa de restricciones a reglas locales por política. La regla de AIBID existe, pero la conectividad debe revisarse con el administrador del dominio."
	} else if len(report.Categories) > 0 {
		trusted := false
		for _, category := range report.Categories {
			trusted = trusted || category == "Private" || category == "DomainAuthenticated"
		}
		if !trusted {
			result.State = "public_network"
			result.Message = "La conexión actual está clasificada como red pública. La regla de AIBID solo permite redes privadas o de dominio."
		}
	}
	return result
}
