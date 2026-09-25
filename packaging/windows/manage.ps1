#Requires -Version 5.1
param([ValidateSet('Preflight','Configure','Admin','Remove','Doctor')][string]$Action,
      [string]$InstallRoot = "$env:ProgramFiles\AIBID-Test")
$ErrorActionPreference = 'Stop'
$ServiceName = 'AIBIDTest'
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
            if ((Test-Path $Marker) -and ((Get-Content $Marker -Raw).Trim() -ne $Version)) {
                throw 'Actualización entre versiones pendiente del respaldo/restauración H7. Se conservan los datos.'
            }
            if ((Test-Path $Config) -and !(Test-Path $Marker)) { throw 'Configuración existente sin versión de paquete. Requiere revisión antes de instalar.' }
            Stop-Aibid
            if ((Test-Path $Binary) -and (Test-Path $Config)) { Invoke-Native $Binary @('migrate','--config',$Config) }
        }
        'Configure' {
            $CommandLine = '"' + $Binary + '" serve --config "' + $Config + '"'
            if (!(Get-Service $ServiceName -ErrorAction SilentlyContinue)) {
                New-Service -Name $ServiceName -BinaryPathName $CommandLine -StartupType Manual -DisplayName 'AIBID Pruebas' | Out-Null
            } else {
                $Installed = Get-CimInstance Win32_Service -Filter "Name='AIBIDTest'"
                if ($Installed.PathName -ne $CommandLine) { throw 'Existe un servicio AIBIDTest con otra ruta. Requiere revisión.' }
            }
            Invoke-Native sc.exe @('config',$ServiceName,'obj=','NT SERVICE\AIBIDTest')
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
            Start-Service $ServiceName
            Write-Host 'AIBID Pruebas: http://127.0.0.1:18090. Puedes cerrar la consola.'
            Start-Process 'http://127.0.0.1:18090'
        }
        'Doctor' { Invoke-Native $Binary @('doctor','--config',$Config,'--sample-directory',(Join-Path $InstallRoot 'docs\pdf')) }
        'Remove' {
            Stop-Aibid
            if (Test-Path $Config) { Invoke-Native $Binary @('migrate','--config',$Config) }
            if (Get-Service $ServiceName -ErrorAction SilentlyContinue) { Invoke-Native sc.exe @('delete',$ServiceName) }
            Write-Host "Datos, configuración y documentos preservados en $DataRoot."
        }
    }
    exit 0
} catch {
    Write-Error $_ -ErrorAction Continue
    exit 1
}
