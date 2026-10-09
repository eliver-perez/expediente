package network

import (
	"context"
	"os"
	"time"

	"gestor-documental/internal/buildinfo"
)

// Query the effective store, not just the saved rule: domain policy may prevent
// local rules from applying. No raw command output or paths enter the response.
const windowsFirewallQuery = `$ErrorActionPreference='Stop'
$profiles = @(Get-NetFirewallProfile | ForEach-Object { [pscustomobject]@{Name=[string]$_.Name; Enabled=($_.Enabled -eq 'True'); LocalRulesBlocked=($_.AllowLocalFirewallRules -eq 'False')} })
$rule = Get-NetFirewallRule -PolicyStore ActiveStore -Name 'AIBID-LAN-TCP' -ErrorAction SilentlyContinue
$detail = $null
if ($rule) {
    $port = $rule | Get-NetFirewallPortFilter
    $app = $rule | Get-NetFirewallApplicationFilter
    $service = $rule | Get-NetFirewallServiceFilter
    $address = $rule | Get-NetFirewallAddressFilter
    $detail = [pscustomobject]@{
        Enabled=($rule.Enabled -eq 'True'); Allow=($rule.Action -eq 'Allow'); Inbound=($rule.Direction -eq 'Inbound')
        Profiles=[string]$rule.Profile; Protocol=[string]$port.Protocol; Port=([string]($port.LocalPort -join ','))
        Program=[Environment]::ExpandEnvironmentVariables([string]$app.Program); Service=[string]$service.Service
        Remote=([string]($address.RemoteAddress -join ',')); EdgeBlocked=($rule.EdgeTraversalPolicy -eq 'Block')
    }
}
$categories = @(Get-NetConnectionProfile -ErrorAction SilentlyContinue | ForEach-Object { [string]$_.NetworkCategory })
[pscustomobject]@{Profiles=$profiles; Rule=$detail; Categories=$categories} | ConvertTo-Json -Depth 4 -Compress`

func inspectFirewall(ctx context.Context, port string) Firewall {
	result := unknownFirewall()
	result.Instructions = "El instalador prepara una regla de AIBID para su puerto TCP, la subred local y perfiles privados o de dominio. En modo local solo se escucha en 127.0.0.1 aunque la regla permanezca preparada. Las políticas del dominio y reglas de bloqueo pueden prevalecer."
	path := commandPath("powershell.exe")
	if path == "" {
		return result
	}
	output, err := probeWithin(ctx, 10*time.Second, path, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", windowsFirewallQuery)
	if err != nil {
		return result
	}
	executable, err := os.Executable()
	if err != nil {
		return result
	}
	return describeWindowsFirewall(result, output, port, executable, buildinfo.ServiceName)
}
