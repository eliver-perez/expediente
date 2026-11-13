# Run from an elevated PowerShell session on the AIDEX test computer.
# Replaces only the service executable; the service's state and source files stay put.
param(
    [string]$Source = (Join-Path $PSScriptRoot '..\build\aidex-server.exe'),
    [string]$Target = 'C:\Program Files\AIDEX\aidex-server.exe'
)

$ErrorActionPreference = 'Stop'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [Security.Principal.WindowsPrincipal]::new($identity)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Ejecuta este script en PowerShell como administrador.'
}

$sourcePath = (Resolve-Path -LiteralPath $Source).Path
$targetPath = (Resolve-Path -LiteralPath $Target).Path
$version = (& $sourcePath version | Select-Object -First 1)
if ($LASTEXITCODE -ne 0 -or $version -notmatch '^AIDEX 1\.0\.0 \(production; windows/amd64\)$') {
    throw 'El binario preparado no es AIDEX 1.0.0 de produccion para Windows amd64.'
}

$stagedPath = "$targetPath.pending"
Copy-Item -LiteralPath $sourcePath -Destination $stagedPath -Force
if ((Get-FileHash -LiteralPath $sourcePath -Algorithm SHA256).Hash -ne
    (Get-FileHash -LiteralPath $stagedPath -Algorithm SHA256).Hash) {
    throw 'El binario copiado no coincide con el compilado.'
}

$service = Get-Service -Name AIDEX
$previousPath = "$targetPath.previous-$([guid]::NewGuid().ToString('N'))"
$oldProcess = (Get-CimInstance Win32_Service -Filter "Name='AIDEX'").ProcessId
try {
    if ($service.Status -ne 'Stopped') {
        Stop-Service -Name AIDEX
        $service.WaitForStatus('Stopped', [TimeSpan]::FromMinutes(5))
    }
    if ($oldProcess -gt 0) {
        $running = Get-Process -Id $oldProcess -ErrorAction SilentlyContinue
        if ($running) { $running.WaitForExit(30000) | Out-Null }
    }
    Move-Item -LiteralPath $targetPath -Destination $previousPath
    Move-Item -LiteralPath $stagedPath -Destination $targetPath
} finally {
    if (-not (Test-Path -LiteralPath $targetPath) -and (Test-Path -LiteralPath $previousPath)) {
        Move-Item -LiteralPath $previousPath -Destination $targetPath
    }
    if ((Get-Service -Name AIDEX).Status -ne 'Running') {
        Start-Service -Name AIDEX
        (Get-Service -Name AIDEX).WaitForStatus('Running', [TimeSpan]::FromMinutes(5))
    }
}
if ((Get-FileHash -LiteralPath $targetPath -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $sourcePath -Algorithm SHA256).Hash) {
    throw 'El servicio reinició con un binario distinto del preparado.'
}
Remove-Item -LiteralPath $previousPath -Force
Write-Output "AIDEX actualizado: $version"
