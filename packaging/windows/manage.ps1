#Requires -Version 5.1
param([ValidateSet('Preflight','Configure','Admin','Remove','Doctor')][string]$Action,
      [string]$InstallRoot = $PSScriptRoot,
      [string]$PreflightBinary = '',
      [switch]$EraseInternalData)
$ErrorActionPreference = 'Stop'
# PowerShell 5.1 otherwise decodes Go's UTF-8 output using the console codepage.
$UTF8 = New-Object Text.UTF8Encoding($false)
[Console]::OutputEncoding = $UTF8
$OutputEncoding = $UTF8
$ServiceName = 'AIDEX'
$FirewallRuleName = 'AIDEX-LAN-TCP'
$CommonData = [Environment]::GetFolderPath('CommonApplicationData')
$DataRoot = Join-Path $CommonData 'AIDEX'
$Config = Join-Path $DataRoot 'config.json'
$Binary = Join-Path $InstallRoot 'aidex-server.exe'
$Version = '@VERSION@'

function Get-AidexServiceCommand {
    return '"' + $Binary + '" serve --config "' + $Config + '" --windows-service ' + $ServiceName
}
function Assert-AidexServiceOwnership {
    $Installed = Get-CimInstance Win32_Service -Filter "Name='$ServiceName'" -ErrorAction Stop
    if ($Installed) {
        if ($Installed.PathName -ne (Get-AidexServiceCommand)) { throw 'Existe un servicio AIDEX con otra ruta. Requiere revisión antes de actualizar.' }
        if ($Installed.StartName -ne ('NT SERVICE\' + $ServiceName)) { throw 'La cuenta del servicio existente requiere revisión antes de actualizar.' }
    }
}
function ConvertTo-AidexNativeArgument([string]$Value) {
    # Windows CRT escaping, including embedded quotes in sc.exe binPath=.
    $Result = '"'
    $Slashes = 0
    foreach ($Character in $Value.ToCharArray()) {
        if ($Character -eq '\') { $Slashes++; continue }
        if ($Character -eq '"') { $Result += ('\' * (2 * $Slashes + 1)) + '"' }
        else { $Result += ('\' * $Slashes) + $Character }
        $Slashes = 0
    }
    return $Result + ('\' * (2 * $Slashes)) + '"'
}
function Invoke-Native([string]$Program, [string[]]$Arguments) {
    if ($Program -eq 'sc.exe') {
        # Bypass PS 5.1 native binding, which can strip binPath's embedded quotes.
        $Start = New-Object Diagnostics.ProcessStartInfo
        $Start.FileName = Join-Path $env:SystemRoot 'System32\sc.exe'
        $Start.Arguments = (($Arguments | ForEach-Object { ConvertTo-AidexNativeArgument $_ }) -join ' ')
        $Start.UseShellExecute = $false
        $Start.CreateNoWindow = $true
        $Start.RedirectStandardOutput = $true
        $Start.RedirectStandardError = $true
        $Process = [Diagnostics.Process]::Start($Start)
        try {
            $Out = $Process.StandardOutput.ReadToEndAsync()
            $Err = $Process.StandardError.ReadToEndAsync()
            if (!$Process.WaitForExit(60000)) { $Process.Kill(); throw 'Windows no completó la operación del servicio en 60 segundos.' }
            # Do not transport sc.exe's legacy-encoded output to the installer.
            $Code = $Process.ExitCode
        } finally { $Process.Dispose() }
    } else {
        & $Program @Arguments --windows-service $ServiceName
        $Code = $LASTEXITCODE
    }
    if ($Code -ne 0) { throw "La operación terminó con código $Code." }
}
function Stop-Aidex {
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
    $Configuration = Get-Content -LiteralPath $Config -Raw -Encoding UTF8 | ConvertFrom-Json
    @{ public_url = [string]$Configuration.public_url; service_name = $ServiceName } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $InstallRoot 'launcher.json') -Encoding UTF8
}
# The elevated installer provisions one narrowly scoped rule. The service keeps
# its virtual account and controls exposure by binding loopback or LAN. The rule
# can remain enabled in local mode: no LAN socket is listening in that mode.
function Set-AidexFirewall {
    $Configuration = Get-Content -LiteralPath $Config -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($Configuration.network_mode -and $Configuration.network_mode -notin @('local', 'lan')) { return }
    $Address = [string]$Configuration.listen_address
    if ($Address -notmatch '^(127\.0\.0\.1|0\.0\.0\.0):([0-9]{1,5})$') { throw 'Dirección de AIDEX no válida para preparar la regla LAN.' }
    $Port = [int]$Matches[2]
    if ($Port -lt 1 -or $Port -gt 65535) { throw 'Puerto de AIDEX inválido.' }
    $Existing = Get-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction SilentlyContinue
    if ($Existing -and $Existing.Group -ne 'AIDEX') { throw 'Existe una regla con el identificador de AIDEX que no pertenece a esta instalación.' }
    $Scope = @{
        PolicyStore = 'PersistentStore'
        Description = 'Regla del instalador AIDEX. Solo el servicio, su puerto TCP y subred local en perfiles privados o de dominio. El modo local de AIDEX no escucha en la LAN.'
        Enabled = 'True'; Direction = 'Inbound'; Action = 'Allow'; Profile = @('Domain', 'Private')
        Program = $Binary; Service = $ServiceName; Protocol = 'TCP'; LocalPort = $Port
        RemoteAddress = 'LocalSubnet'; EdgeTraversalPolicy = 'Block'; ErrorAction = 'Stop'
    }
    if ($Existing) { Set-NetFirewallRule -Name $FirewallRuleName -NewDisplayName 'AIDEX - Red local' @Scope | Out-Null }
    else { New-NetFirewallRule -Name $FirewallRuleName -DisplayName 'AIDEX - Red local' -Group 'AIDEX' @Scope | Out-Null }
    $Saved = Get-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction Stop
    if ($Saved.Enabled -ne 'True' -or $Saved.Action -ne 'Allow' -or $Saved.Direction -ne 'Inbound') {
        throw 'Windows no confirmó la regla de acceso LAN de AIDEX.'
    }
    Write-Host "Firewall: regla AIDEX preparada para TCP $Port, subred local, dominio y red privada."
}
function Remove-AidexFirewall {
    $Existing = Get-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction SilentlyContinue
    if ($Existing) {
        if ($Existing.Group -ne 'AIDEX') { throw 'Se conserva una regla ajena con el identificador de AIDEX.' }
        Remove-NetFirewallRule -PolicyStore PersistentStore -Name $FirewallRuleName -ErrorAction Stop
    }
}
function Remove-PackagedFiles {
    $Manifest = Join-Path $InstallRoot 'package-files.json'
    $Entries = Get-Content -LiteralPath $Manifest -Raw -Encoding UTF8 | ConvertFrom-Json
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
        Start-Process powershell.exe -Verb RunAs -ArgumentList ('-NoProfile -NoExit -ExecutionPolicy Bypass -File "' + $PSCommandPath + '" -Action Admin -InstallRoot "' + $InstallRoot + '"')
        exit 0
    }
    Assert-AidexServiceOwnership
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
            # Updates preserve private state and accept only this production version.
            if ((Test-Path $Marker) -and ((Get-Content $Marker -Raw).Trim() -notin @($Version))) {
                throw 'Versión de datos no compatible con este instalador. Se conservan los datos.'
            }
            if ((Test-Path $Config) -and !(Test-Path $Marker)) { throw 'Configuración existente sin versión de paquete. Requiere revisión antes de instalar.' }
            Assert-AidexServiceOwnership
            Stop-Aidex
            if (Test-Path $Config) {
                if (!$PreflightBinary -or !(Test-Path -LiteralPath $PreflightBinary)) { throw 'Falta el comprobador del nuevo instalador.' }
                # Check exclusive access without opening or altering SQLite.
                Invoke-Native $PreflightBinary @('check-state','--config',$Config)
            }
        }
        'Configure' {
            Assert-AidexServiceOwnership
            Stop-Aidex
            $CommandLine = Get-AidexServiceCommand
            if (!(Get-Service $ServiceName -ErrorAction SilentlyContinue)) {
                New-Service -Name $ServiceName -BinaryPathName $CommandLine -StartupType Manual -DisplayName 'AIDEX' | Out-Null
                Invoke-Native sc.exe @('config',$ServiceName,'obj=',('NT SERVICE\' + $ServiceName))
            } else {
                # Verified installation command; retain the same SID, data and ACLs.
                Invoke-Native sc.exe @('config',$ServiceName,'binPath=',$CommandLine)
            }
            Set-Service -Name $ServiceName -DisplayName 'AIDEX'
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
            Set-AidexFirewall
            Set-Content -LiteralPath (Join-Path $DataRoot 'package-version') -Value $Version -Encoding ASCII
            if (Test-Path (Join-Path $DataRoot 'service-enabled')) { Start-Service $ServiceName; (Get-Service $ServiceName).WaitForStatus('Running', [TimeSpan]::FromSeconds(60)) }
        }
        'Admin' {
            Invoke-Native $Binary @('doctor','--config',$Config,'--sample-directory',(Join-Path $InstallRoot 'docs\pdf'))
            if (!(Test-Path (Join-Path $DataRoot 'service-enabled'))) {
                & $Binary bootstrap-ready --config $Config --windows-service $ServiceName
                if ($LASTEXITCODE -ne 0) { Invoke-Native $Binary @('bootstrap','--config',$Config) }
                New-Item -ItemType File -Path (Join-Path $DataRoot 'service-enabled') -Force | Out-Null
            }
            Invoke-Native sc.exe @('config',$ServiceName,'start=','delayed-auto')
            Write-LauncherTarget
            Set-AidexFirewall
            Start-Service $ServiceName
            (Get-Service $ServiceName).WaitForStatus('Running', [TimeSpan]::FromSeconds(60))
            Write-Host 'AIDEX: http://127.0.0.1:18090. Puedes cerrar la consola.'
            $Configuration = Get-Content -LiteralPath $Config -Raw -Encoding UTF8 | ConvertFrom-Json
            Start-Process ([string]$Configuration.public_url)
        }
        'Doctor' { Invoke-Native $Binary @('doctor','--config',$Config,'--sample-directory',(Join-Path $InstallRoot 'docs\pdf')) }
        'Remove' {
            Stop-Aidex
            if (Test-Path $Config) { Invoke-Native $Binary @('check-state','--config',$Config) }
            Remove-AidexFirewall
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
