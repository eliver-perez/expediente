$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$helper = Join-Path $workspace 'build\aidex-benchmark-user.exe'
$database = 'C:\ProgramData\AIDEX\state'
$operation = '37421a3f-89c8-4986-89bb-eb6a12e036af'
$output = Join-Path $workspace '.tools\real-benchmark\deletion-final-api.json'
$errorOutput = Join-Path $workspace '.tools\real-benchmark\deletion-final-api-error.txt'
$base = 'http://127.0.0.1:18090'
$name = 'deletion_final_' + ([guid]::NewGuid().ToString('N').Substring(0, 12))
$bytes = New-Object byte[] 32
$random = [Security.Cryptography.RandomNumberGenerator]::Create()
$random.GetBytes($bytes)
$random.Dispose()
$password = [Convert]::ToBase64String($bytes)
$created = $false
try {
    $password | & $helper create $database $name | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear la cuenta temporal.' }
    $created = $true
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $login = Invoke-WebRequest -Uri "$base/api/v1/auth/login" -Method Post -Body (@{username=$name;password=$password}|ConvertTo-Json -Compress) -ContentType 'application/json' -Headers @{Origin=$base} -WebSession $session -UseBasicParsing -TimeoutSec 30
    $clock = [Diagnostics.Stopwatch]::StartNew()
    $response = Invoke-WebRequest -Uri "$base/api/v1/deletions/$operation" -WebSession $session -UseBasicParsing -TimeoutSec 30
    $clock.Stop()
    $status = $response.Content | ConvertFrom-Json
    [pscustomobject]@{
        state=$status.state
        diagnostics=$status.cleanup_diagnostics
        pending_cleanup=$status.pending_cleanup
        http_status=[int]$response.StatusCode
        duration_ms=[math]::Round($clock.Elapsed.TotalMilliseconds,1)
        service_hash=(Get-FileHash 'C:\Program Files\AIDEX\aidex-server.exe').Hash
    } | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $output -Encoding UTF8
} catch {
    $_ | Out-String | Set-Content -LiteralPath $errorOutput -Encoding UTF8
    throw
} finally {
    $password = $null
    if ($created) { & $helper disable $database $name | Out-Null }
}
