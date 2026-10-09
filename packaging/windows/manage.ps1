#Requires -Version 5.1
param([ValidateSet('Preflight','Configure','Admin','Remove','Doctor')][string]$Action,
      [string]$InstallRoot = "$env:ProgramFiles\AIBID-Test",
      [string]$PreflightBinary = '',
      [switch]$EraseInternalData)
$ErrorActionPreference = 'Stop'
# Keep the service SID and data paths during the production-name transition.
$ServiceName = 'AIBIDTest'
$FirewallRuleName = 'AIBID-LAN-TCP'
$DataRoot = Join-Path ([Environment]::GetFolderPath('CommonApplicationData')) 'AIBID-Test'
$Config = Join-Path $DataRoot 'config.json'
$Binary = Join-Path $InstallRoot 'gestor-documental.exe'
$Version = '@VERSION@'

function Invoke-Native([string]$Program, [string[]]$Arguments) {
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Program terminó con código $LASTEXITCODE" }
}
function Stop-Aibid {
    $Service = Get-Service $ServiceName -ErrorAction SilentlyContinue
    if ($Service -and $Service.Status -ne 'Stopped') {
        Stop-Service $ServiceName -ErrorAction Stop
        $Service.WaitForStatus('Stopped', [TimeSpan]::FromSeconds(60))
    }
}
function Assert-PrivatePath([string]$Path) {
    $Current = $Path
    while ($Current) {
        if (Test-Path -LiteralPath $Current) {
            if ((Get-Item -LiteralPath $Current -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "No se admiten enlaces: $Current" }
        }
        $Current = Split-Path -Parent $Current
    }
}
function Write-LauncherTarget {
    # Publish only the browser origin in Program Files. Private configuration,
    # license keys and database remain inaccessible to ordinary desktop users.
    $Configuration = Get-Content -LiteralPath $Config -Raw | ConvertFrom-Json
    @{ public_url = [string]$Configuration.public_url } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $InstallRoot 'launcher.json') -Encoding UTF8
}
# The elevated installer provisions one narrowly scoped rule. The service keeps
# its virtual account and controls exposure by binding loopback or LAN. The rule
# can remain enabled in local mode: no LAN socket is listening in that mode.
function Set-AibidFirewall {
    $Configuration = Get-Content -LiteralPath $Config -Raw | ConvertFrom-Json
    if ($Configuration.network_mode -and $Configuration.network_mode -notin @('local', 'lan')) { return }
    $Address = [string]$Configuration.listen_address
    if ($Address -notmatch '^(127\.0\.0\.1|0\.0\.0\.0):([0-9]{1,5})$') { throw 'Dirección de AIBID no válida para preparar la regla LAN.' }
    $Port = [int]$Matches[2]
    if ($Port -lt 1 -or $Port -gt 65535) { throw 'Puerto de AIBID inválido.' }
    $Existing = Get-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction SilentlyContinue
    if ($Existing -and $Existing.Group -ne 'AIBID') { throw 'Existe una regla con el identificador de AIBID que no pertenece a esta instalación.' }
    $Scope = @{
        PolicyStore = 'PersistentStore'
        Description = 'Regla del instalador AIBID. Solo el servicio, su puerto TCP y subred local en perfiles privados o de dominio. El modo local de AIBID no escucha en la LAN.'
        Enabled = 'True'; Direction = 'Inbound'; Action = 'Allow'; Profile = @('Domain', 'Private')
        Program = $Binary; Service = $ServiceName; Protocol = 'TCP'; LocalPort = $Port
        RemoteAddress = 'LocalSubnet'; EdgeTraversalPolicy = 'Block'; ErrorAction = 'Stop'
    }
    if ($Existing) { Set-NetFirewallRule -Name $FirewallRuleName -NewDisplayName 'AIBID - Red local' @Scope | Out-Null }
    else { New-NetFirewallRule -Name $FirewallRuleName -DisplayName 'AIBID - Red local' -Group 'AIBID' @Scope | Out-Null }
    $Saved = Get-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction Stop
    if ($Saved.Enabled -ne 'True' -or $Saved.Action -ne 'Allow' -or $Saved.Direction -ne 'Inbound') {
        throw 'Windows no confirmó la regla de acceso LAN de AIBID.'
    }
    Write-Host "Firewall: regla AIBID preparada para TCP $Port, subred local, dominio y red privada."
}
function Remove-AibidFirewall {
    $Existing = Get-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction SilentlyContinue
    if ($Existing) {
        if ($Existing.Group -ne 'AIBID') { throw 'Se conserva una regla ajena con el identificador de AIBID.' }
        Remove-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction Stop
    }
}
function Remove-PackagedFiles {
    $Manifest = Join-Path $InstallRoot 'package-files.json'
    $Entries = Get-Content -LiteralPath $Manifest -Raw | ConvertFrom-Json
    $RootPrefix = [IO.Path]::GetFullPath($InstallRoot).TrimEnd('\') + '\'
    # Validate the complete manifest before removing anything. Unknown or changed
    # files are preserved, even if someone placed documents in the program folder.
    foreach ($Entry in $Entries) {
        $Path = [IO.Path]::GetFullPath((Join-Path $InstallRoot $Entry.path))
        if (!$Path.StartsWith($RootPrefix, [StringComparison]::OrdinalIgnoreCase) -or $Entry.sha256 -notmatch '^[a-f0-9]{64}$') { throw 'Manifiesto de programa inválido.' }
        Assert-PrivatePath $Path
    }
    foreach ($Entry in $Entries) {
        $Path = Join-Path $InstallRoot $Entry.path
        if (Test-Path -LiteralPath $Path -PathType Leaf) {
            if ((Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash -eq $Entry.sha256) {
                Remove-Item -LiteralPath $Path -Force
            } else { Write-Host "Se conserva el archivo modificado: $Path" }
        }
    }
    Remove-Item -LiteralPath $Manifest
    # Leave directories in place; the installer only removes empty top-level folders.
}
try {
    $Administrator = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (!$Administrator) {
        if ($Action -ne 'Admin') { throw 'Esta operación requiere una consola de administrador.' }
        # Only the interactive launcher self-elevates; installer failures remain visible.
        Start-Process powershell.exe -Verb RunAs -ArgumentList ('-NoProfile -NoExit -ExecutionPolicy Bypass -File "' + $PSCommandPath + '" -Action Admin')
        exit 0
    }
    Assert-PrivatePath $DataRoot
    Assert-PrivatePath $InstallRoot
    Assert-PrivatePath $Config
    switch ($Action) {
        'Preflight' {
            if (Test-Path $DataRoot) {
                $Owner = (Get-Acl -LiteralPath $DataRoot).GetOwner([Security.Principal.SecurityIdentifier]).Value
                $CurrentUser = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
                if ($Owner -notin @('S-1-5-18','S-1-5-32-544',$CurrentUser)) { throw 'La carpeta de datos existente pertenece a otra cuenta. Requiere revisión antes de instalar.' }
            }
            $Marker = Join-Path $DataRoot 'package-version'
            # Only this phase and the existing preview have a tested additive
            # migration. Configure snapshots/migrates before updating the marker.
            if ((Test-Path $Marker) -and ((Get-Content $Marker -Raw).Trim() -notin @('0.7.0-test.1', '2.0.0-alpha.1', '2.0.0-alpha.2', '2.0.0-alpha.3', '2.0.0-alpha.4', '2.0.0-alpha.5', '2.0.0-alpha.6', '2.0.0-alpha.7', '2.0.0-alpha.8', $Version))) {
                throw 'Versión de datos no compatible con esta fase. Se conservan los datos.'
            }
            if ((Test-Path $Config) -and !(Test-Path $Marker)) { throw 'Configuración existente sin versión de paquete. Requiere revisión antes de instalar.' }
            Stop-Aibid
            if (Test-Path $Config) {
                if (!$PreflightBinary -or !(Test-Path -LiteralPath $PreflightBinary)) { throw 'Falta el comprobador del nuevo instalador.' }
                # The old executable has the Windows URI bug. Never call it here;
                # the new checker only holds the state lock, without opening SQLite.
                Invoke-Native $PreflightBinary @('check-state','--config',$Config)
            }
        }
        'Configure' {
            $CommandLine = '"' + $Binary + '" serve --config "' + $Config + '"'
            if (!(Get-Service $ServiceName -ErrorAction SilentlyContinue)) {
                New-Service -Name $ServiceName -BinaryPathName $CommandLine -StartupType Manual -DisplayName 'AIBID' | Out-Null
                Invoke-Native sc.exe @('config',$ServiceName,'obj=','NT SERVICE\AIBIDTest')
            } else {
                $Installed = Get-CimInstance Win32_Service -Filter "Name='AIBIDTest'"
                if ($Installed.PathName -ne $CommandLine) { throw 'Existe un servicio AIBIDTest con otra ruta. Requiere revisión.' }
            }
            Set-Service -Name $ServiceName -DisplayName 'AIBID'
            Invoke-Native sc.exe @('sidtype',$ServiceName,'unrestricted')
            if (![Diagnostics.EventLog]::SourceExists($ServiceName)) { New-EventLog -LogName Application -Source $ServiceName }
            if (!(Test-Path $DataRoot)) { New-Item -ItemType Directory -Path $DataRoot | Out-Null }
            $ServiceSID = (New-Object Security.Principal.NTAccount('NT SERVICE', $ServiceName)).Translate([Security.Principal.SecurityIdentifier]).Value
            $Acl = New-Object Security.AccessControl.DirectorySecurity
            $Acl.SetSecurityDescriptorSddlForm("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;$ServiceSID)")
            $Acl.SetOwner((New-Object Security.Principal.SecurityIdentifier('S-1-5-32-544')))
            Set-Acl -LiteralPath $DataRoot -AclObject $Acl
            if (!(Test-Path $Config)) {
                Invoke-Native $Binary @('init','--config',$Config,'--listen','127.0.0.1:18090','--tools',(Join-Path $InstallRoot 'tools\Library\bin'),'--tessdata',(Join-Path $InstallRoot 'tools\tessdata'),'--fontconfig',(Join-Path $InstallRoot 'tools\fonts.conf'))
            }
            Invoke-Native $Binary @('configure-license','--config',$Config)
            Invoke-Native $Binary @('migrate','--config',$Config)
            Write-LauncherTarget
            Set-AibidFirewall
            Set-Content -LiteralPath (Join-Path $DataRoot 'package-version') -Value $Version -Encoding ASCII
            if (Test-Path (Join-Path $DataRoot 'service-enabled')) { Start-Service $ServiceName }
        }
        'Admin' {
            Invoke-Native $Binary @('doctor','--config',$Config,'--sample-directory',(Join-Path $InstallRoot 'docs\pdf'))
            if (!(Test-Path (Join-Path $DataRoot 'service-enabled'))) {
                & $Binary bootstrap-ready --config $Config
                if ($LASTEXITCODE -ne 0) { Invoke-Native $Binary @('bootstrap','--config',$Config) }
                New-Item -ItemType File -Path (Join-Path $DataRoot 'service-enabled') -Force | Out-Null
            }
            Invoke-Native sc.exe @('config',$ServiceName,'start=','delayed-auto')
            Write-LauncherTarget
            Set-AibidFirewall
            Start-Service $ServiceName
            Write-Host 'AIBID: http://127.0.0.1:18090. Puedes cerrar la consola.'
            $Configuration = Get-Content -LiteralPath $Config -Raw | ConvertFrom-Json
            Start-Process ([string]$Configuration.public_url)
        }
        'Doctor' { Invoke-Native $Binary @('doctor','--config',$Config,'--sample-directory',(Join-Path $InstallRoot 'docs\pdf')) }
        'Remove' {
            Stop-Aibid
            if (Test-Path $Config) { Invoke-Native $Binary @('check-state','--config',$Config) }
            Remove-AibidFirewall
            if ($EraseInternalData -and (Test-Path $Config)) {
                Invoke-Native $Binary @('erase-internal-data','--config',$Config,'--confirm-erase-internal-data')
            }
            if (Get-Service $ServiceName -ErrorAction SilentlyContinue) { Invoke-Native sc.exe @('delete',$ServiceName) }
            Remove-PackagedFiles
            if ($EraseInternalData) { Write-Host "Datos internos eliminados. Documentos físicos conservados; las cargas privadas permanecen en $DataRoot\state\uploads." }
            else { Write-Host "Datos, configuración y documentos preservados en $DataRoot." }
        }
    }
    exit 0
} catch {
    Write-Error $_ -ErrorAction Continue
    exit 1
}
