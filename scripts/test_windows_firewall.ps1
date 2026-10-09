#Requires -Version 5.1
# Test only the firewall provisioning functions, with in-memory NetSecurity mocks.
# No service, firewall policy, administrator token or Windows machine is needed.
$ErrorActionPreference = 'Stop'
$Source = Join-Path $PSScriptRoot '../packaging/windows/manage.ps1'
$Tokens = $null; $Issues = $null
$Ast = [Management.Automation.Language.Parser]::ParseFile($Source, [ref]$Tokens, [ref]$Issues)
if ($Issues.Count) { throw 'manage.ps1 does not parse' }
foreach ($Name in @('Set-AibidFirewall', 'Remove-AibidFirewall')) {
    $Functions = @($Ast.FindAll({ param($Node) $Node -is [Management.Automation.Language.FunctionDefinitionAst] -and $Node.Name -eq $Name }, $true))
    if ($Functions.Count -ne 1) { throw "Missing function $Name" }
    . ([scriptblock]::Create($Functions[0].Extent.Text))
}
$script:Rule = $null; $script:Creates = 0; $script:Updates = 0; $script:Removes = 0
$FirewallRuleName = 'AIBID-LAN-TCP'; $ServiceName = 'AIBIDTest'; $Binary = 'C:\Program Files\AIBID-Test\gestor-documental.exe'
$Config = [IO.Path]::GetTempFileName()
function Assert-True($Value, [string]$Message) { if (!$Value) { throw $Message } }
function Get-NetFirewallRule {
    param($PolicyStore, $Name, $ErrorAction)
    Assert-True ($PolicyStore -eq 'PersistentStore' -and $Name -eq $FirewallRuleName) 'Broad firewall query'
    return $script:Rule
}
function New-NetFirewallRule {
    [CmdletBinding()]
    param($Name,$Group,$PolicyStore,$DisplayName,$Description,$Enabled,$Direction,$Action,$Profile,$Program,$Service,$Protocol,$LocalPort,$RemoteAddress,$EdgeTraversalPolicy)
    Assert-True ($Name -eq $FirewallRuleName -and $Group -eq 'AIBID') 'Rule identity changed'
    Assert-True ($DisplayName -eq 'AIBID - Red local') 'Rule display name missing'
    Assert-True ($PolicyStore -eq 'PersistentStore' -and $Direction -eq 'Inbound' -and $Action -eq 'Allow') 'Invalid rule action'
    Assert-True ($Profile.Count -eq 2 -and 'Domain' -in $Profile -and 'Private' -in $Profile) 'Public profile exposed'
    Assert-True ($Program -eq $Binary -and $Service -eq $ServiceName) 'Rule not scoped to service/program'
    Assert-True ($Protocol -eq 'TCP' -and $LocalPort -eq 18090 -and $RemoteAddress -eq 'LocalSubnet' -and $EdgeTraversalPolicy -eq 'Block') 'Rule network scope broadened'
    $script:Rule = [pscustomobject]@{Name=$Name;Group=$Group;Enabled=$Enabled;Direction=$Direction;Action=$Action}
    $script:Creates++
}
function Set-NetFirewallRule {
    # Match the real NetSecurity parameter sets: DisplayName selects rules and
    # cannot accompany Name. Renaming uses NewDisplayName in the ByName set.
    [CmdletBinding(DefaultParameterSetName='ByName')]
    param(
        [Parameter(Mandatory=$true,ParameterSetName='ByName')]$Name,
        [Parameter(Mandatory=$true,ParameterSetName='ByDisplayName')]$DisplayName,
        $NewDisplayName,$PolicyStore,$Description,$Enabled,$Direction,$Action,$Profile,$Program,$Service,$Protocol,$LocalPort,$RemoteAddress,$EdgeTraversalPolicy)
    Assert-True ($PSCmdlet.ParameterSetName -eq 'ByName') 'Rule updated by an unstable display name'
    $Before = $script:Creates
    $Creation = @{}
    foreach ($Key in $PSBoundParameters.Keys) { if ($Key -ne 'NewDisplayName') { $Creation[$Key] = $PSBoundParameters[$Key] } }
    New-NetFirewallRule -Group 'AIBID' -DisplayName $NewDisplayName @Creation
    $script:Creates = $Before; $script:Updates++
}
function Remove-NetFirewallRule {
    param($PolicyStore,$Name,$ErrorAction)
    Assert-True ($PolicyStore -eq 'PersistentStore' -and $Name -eq $FirewallRuleName) 'Broad firewall deletion'
    $script:Rule = $null; $script:Removes++
}
try {
    $Rejected = $false
    try { Set-NetFirewallRule -Name $FirewallRuleName -DisplayName 'AIBID - Red local' } catch [Management.Automation.ParameterBindingException] { $Rejected = $true }
    Assert-True $Rejected 'Mock accepts incompatible NetSecurity parameter sets'
    '{"network_mode":"local","listen_address":"127.0.0.1:18090"}' | Set-Content -LiteralPath $Config
    Set-AibidFirewall
    Set-AibidFirewall
    Assert-True ($script:Creates -eq 1 -and $script:Updates -eq 1) 'Repeated upgrade duplicated firewall rules'
    $script:Rule.Group = 'Other application'
    $Rejected = $false
    try { Set-AibidFirewall } catch { $Rejected = $true }
    Assert-True $Rejected 'Installer overwrote an unrelated rule'
    $Rejected = $false
    try { Remove-AibidFirewall } catch { $Rejected = $true }
    Assert-True $Rejected 'Uninstaller removed an unrelated rule'
    $script:Rule.Group = 'AIBID'
    Remove-AibidFirewall
    Remove-AibidFirewall
    Assert-True ($script:Removes -eq 1) 'Uninstall is not idempotent'
    '{"network_mode":"local","listen_address":"127.0.0.1:99999"}' | Set-Content -LiteralPath $Config
    $Rejected = $false
    try { Set-AibidFirewall } catch { $Rejected = $true }
    Assert-True $Rejected 'Invalid port accepted'
    '{"network_mode":"custom","listen_address":"0.0.0.0:443"}' | Set-Content -LiteralPath $Config
    Set-AibidFirewall
    Assert-True ($null -eq $script:Rule) 'Custom network configuration changed'
    Write-Host 'PASS: Windows firewall provisioning, scope, upgrade, ownership, uninstall and invalid input.'
} finally {
    Remove-Item -LiteralPath $Config
}
