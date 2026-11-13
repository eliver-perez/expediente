param(
    [string]$OperationID = '4fc85534-ddc6-4321-aa2c-fbb238e3ebec',
    [string]$LibraryID = '2644f1e4-2d90-4e34-91b6-b941246b46b4',
    [switch]$NewPlan,
    [switch]$ObserveOnly,
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '..\.tools\real-benchmark')
)
$ErrorActionPreference = 'Stop'
$principal = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Ejecuta PowerShell como administrador.' }
$workspace = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$helper = Join-Path $workspace 'build\aidex-benchmark-user.exe'
$config = Get-Content 'C:\ProgramData\AIDEX\config.json' -Raw | ConvertFrom-Json
$database = $config.state_directory
$base = 'http://127.0.0.1:18090'
$trace = Join-Path $OutputDirectory 'deletion-progress.jsonl'
$final = Join-Path $OutputDirectory 'deletion-result.json'
$errorFile = Join-Path $OutputDirectory 'deletion-error.txt'
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
Remove-Item -LiteralPath $trace,$final,$errorFile -ErrorAction SilentlyContinue
$prepared = [bool]$env:AIDEX_DELETION_BENCH_USER
if ($prepared) {
    $name = $env:AIDEX_DELETION_BENCH_USER
    $password = $env:AIDEX_DELETION_BENCH_PASSWORD
    if (-not $password) { throw 'Falta la contraseña de la cuenta temporal preparada.' }
} else {
    $name = 'deletion_bench_' + ([guid]::NewGuid().ToString('N').Substring(0, 12))
    $bytes = New-Object byte[] 32
    $random = [Security.Cryptography.RandomNumberGenerator]::Create()
    $random.GetBytes($bytes)
    $random.Dispose()
    $password = [Convert]::ToBase64String($bytes)
    $password | & $helper create $database $name | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear la cuenta temporal.' }
}
try {
    $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $login = Invoke-WebRequest -Uri "$base/api/v1/auth/login" -Method Post -Body (@{username=$name;password=$password}|ConvertTo-Json -Compress) -ContentType 'application/json' -Headers @{Origin=$base} -WebSession $session -UseBasicParsing -TimeoutSec 50
    $csrf = ($login.Content | ConvertFrom-Json).csrf_token
    if (-not $csrf) { throw 'La API no devolvió el token CSRF.' }
    if ($NewPlan) {
        $planResponse = Invoke-WebRequest -Uri "$base/api/v1/libraries/$LibraryID/deletion-plans" -Method Post -Headers @{Origin=$base;'X-CSRF-Token'=$csrf} -WebSession $session -UseBasicParsing -TimeoutSec 180
        $plan = $planResponse.Content | ConvertFrom-Json
        $OperationID = $plan.id
        [pscustomobject]@{id=$OperationID;name=$plan.name;documents=$plan.counts.documents;created=[DateTime]::UtcNow.ToString('o')} | ConvertTo-Json | Set-Content (Join-Path $OutputDirectory 'deletion-new-plan.json') -Encoding UTF8
        $sampleRows = & $helper deletionsample $database $OperationID | ConvertFrom-Json
        $verified = foreach ($sample in $sampleRows) {
            $paths = $sample.locations_json | ConvertFrom-Json
            $path = @($paths | Where-Object { Test-Path -LiteralPath $_ }) | Select-Object -First 1
            if ($path) { [pscustomobject]@{ path=$path; sha256=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash; manifest_sha256=$sample.sha256 } }
        }
        $verified | ConvertTo-Json -Depth 3 | Set-Content (Join-Path $OutputDirectory 'deletion-active.sample.json') -Encoding UTF8
        $confirm = Invoke-WebRequest -Uri "$base/api/v1/deletions/$OperationID/confirm" -Method Post -Body (@{confirmation_name=$plan.name}|ConvertTo-Json -Compress) -ContentType 'application/json' -Headers @{Origin=$base;'X-CSRF-Token'=$csrf} -WebSession $session -UseBasicParsing -TimeoutSec 180
    } elseif (-not $ObserveOnly) {
        $retry = Invoke-WebRequest -Uri "$base/api/v1/deletions/$OperationID/retry" -Method Post -Headers @{Origin=$base;'X-CSRF-Token'=$csrf} -WebSession $session -UseBasicParsing -TimeoutSec 50
    }
    $started = [DateTime]::UtcNow
    $last = $null
    for ($i = 0; $i -lt 3600; $i++) {
        $watch = [Diagnostics.Stopwatch]::StartNew()
        $response = Invoke-WebRequest -Uri "$base/api/v1/deletions/$OperationID" -WebSession $session -UseBasicParsing -TimeoutSec 50
        $watch.Stop()
        $last = $response.Content | ConvertFrom-Json
        $process = Get-Process -Name aidex-server -ErrorAction SilentlyContinue | Select-Object -First 1
        $sample = [ordered]@{
            time=[DateTime]::UtcNow.ToString('o'); state=$last.state; phase=$last.progress.phase;
            done=$last.progress.done; total=$last.progress.total; unit=$last.progress.unit;
            status_ms=[math]::Round($watch.Elapsed.TotalMilliseconds,1); handles=$process.HandleCount;
            working_set_bytes=$process.WorkingSet64; private_bytes=$process.PrivateMemorySize64
        }
        Add-Content -LiteralPath $trace -Value ($sample|ConvertTo-Json -Compress) -Encoding UTF8
        if ($i % 10 -eq 0) {
            $sessionWatch = [Diagnostics.Stopwatch]::StartNew()
            $sessionResponse = Invoke-WebRequest -Uri "$base/api/v1/auth/session" -WebSession $session -UseBasicParsing -TimeoutSec 20
            $sessionWatch.Stop()
            Add-Content -LiteralPath $trace -Value (([ordered]@{time=[DateTime]::UtcNow.ToString('o');endpoint='session';status=[int]$sessionResponse.StatusCode;ms=[math]::Round($sessionWatch.Elapsed.TotalMilliseconds,1)})|ConvertTo-Json -Compress) -Encoding UTF8
        }
        if ($last.state -in @('failed','complete','cleanup_pending')) { break }
        Start-Sleep -Seconds 2
    }
    $sampleFile = if ($NewPlan) { 'deletion-active.sample.json' } else { 'deletion-before.sample.json' }
    $samples = Get-Content (Join-Path $OutputDirectory $sampleFile) -Raw | ConvertFrom-Json
    $originals = foreach ($sample in $samples) {
        $exists = Test-Path -LiteralPath $sample.path
        $hash = if ($exists) { (Get-FileHash -LiteralPath $sample.path -Algorithm SHA256).Hash } else { '' }
        [pscustomobject]@{ exists=$exists; same_hash=($hash -eq $sample.sha256) }
    }
    [pscustomobject]@{operation_id=$OperationID; elapsed_seconds=([math]::Round(([DateTime]::UtcNow-$started).TotalSeconds,1)); state=$last.state; diagnostics=$last.cleanup_diagnostics; progress=$last.progress; originals_checked=@($originals).Count; originals_intact=@($originals|Where-Object {$_.exists -and $_.same_hash}).Count; service_hash=(Get-FileHash 'C:\Program Files\AIDEX\aidex-server.exe').Hash} | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $final -Encoding UTF8
} catch {
    $details = $_ | Out-String
    if ($_.Exception.Response) {
        $details += "`nHTTP $([int]$_.Exception.Response.StatusCode)"
        $stream = $_.Exception.Response.GetResponseStream()
        if ($stream) { $reader = [System.IO.StreamReader]::new($stream); $details += "`n" + $reader.ReadToEnd(); $reader.Dispose() }
    }
    $details | Set-Content -LiteralPath $errorFile -Encoding UTF8
    throw
} finally {
    if (-not $prepared) { & $helper disable $database $name | Out-Null }
    $password = $null
}
