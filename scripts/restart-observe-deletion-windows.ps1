$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$helper = Join-Path $workspace 'build\aidex-benchmark-user.exe'
$config = Get-Content 'C:\ProgramData\AIDEX\config.json' -Raw | ConvertFrom-Json
$database = Join-Path $config.state_directory 'documental.db'
$output = Join-Path $workspace '.tools\real-benchmark\deletion-observe-wrapper-error.txt'
$name = 'deletion_bench_' + ([guid]::NewGuid().ToString('N').Substring(0, 12))
$bytes = New-Object byte[] 32
$random = [Security.Cryptography.RandomNumberGenerator]::Create()
$random.GetBytes($bytes)
$random.Dispose()
$password = [Convert]::ToBase64String($bytes)
$created = $false
try {
    Stop-Service AIDEX
    (Get-Service AIDEX).WaitForStatus('Stopped', [TimeSpan]::FromMinutes(5))
    $password | & $helper create $database $name | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear la cuenta temporal con el servicio detenido.' }
    $created = $true
    Start-Service AIDEX
    (Get-Service AIDEX).WaitForStatus('Running', [TimeSpan]::FromMinutes(5))
    $env:AIDEX_DELETION_BENCH_USER = $name
    $env:AIDEX_DELETION_BENCH_PASSWORD = $password
    & (Join-Path $PSScriptRoot 'benchmark-deletion-windows.ps1') -OperationID '37421a3f-89c8-4986-89bb-eb6a12e036af' -ObserveOnly
} catch {
    $_ | Out-String | Set-Content -LiteralPath $output -Encoding UTF8
    throw
} finally {
    Remove-Item Env:\AIDEX_DELETION_BENCH_USER,Env:\AIDEX_DELETION_BENCH_PASSWORD -ErrorAction SilentlyContinue
    $password = $null
    if ((Get-Service AIDEX).Status -eq 'Stopped') { Start-Service AIDEX }
    if ($created) { & $helper disable $database $name | Out-Null }
}
