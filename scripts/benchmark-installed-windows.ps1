# Run once in an elevated Windows PowerShell console. Results contain no credentials.
param([string]$OutputDirectory = (Join-Path $PSScriptRoot '..\.tools\real-benchmark'), [switch]$ResumeAfterBaseline)
$ErrorActionPreference = 'Stop'
$principal = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Ejecuta PowerShell como administrador.' }
$configPath = 'C:\ProgramData\AIDEX\config.json'
$config = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
$dbPath = Join-Path $config.state_directory 'documental.db'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$helper = Join-Path $root 'build\aidex-benchmark-user.exe'
$replacement = Join-Path $root 'build\aidex-server.exe'
$installed = 'C:\Program Files\AIDEX\aidex-server.exe'
$base = 'http://127.0.0.1:18090'
if (-not (Test-Path -LiteralPath $dbPath)) { throw "No se encontró la base real: $dbPath" }
New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
$target = (& $helper target $dbPath ignored | ConvertFrom-Json)
if ($LASTEXITCODE -ne 0) { throw 'No se pudo elegir la biblioteca real.' }
$name = 'benchmark_' + ([guid]::NewGuid().ToString('N').Substring(0,12))
$random = [Security.Cryptography.RandomNumberGenerator]::Create()
$secretBytes = New-Object byte[] 32
$random.GetBytes($secretBytes)
$random.Dispose()
$password = [Convert]::ToBase64String($secretBytes)
$password | & $helper create $dbPath $name | Out-Null
if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear el usuario temporal.' }

function Login {
    $script:session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $body = @{ username=$name; password=$password } | ConvertTo-Json -Compress
    $response = Invoke-WebRequest -Uri "$base/api/v1/auth/login" -Method Post -Body $body -ContentType 'application/json' -Headers @{Origin=$base} -WebSession $script:session -UseBasicParsing -TimeoutSec 50
    if ($response.StatusCode -ne 200) { throw "Falló el inicio de sesión: $($response.StatusCode)" }
    $script:csrf = ($response.Content | ConvertFrom-Json).csrf_token
    if (-not $script:csrf) { throw 'La API no devolvió el token CSRF.' }
}
function Request([string]$phase,[string]$endpoint,[string]$path) {
    $sw = [Diagnostics.Stopwatch]::StartNew()
    $status = 0; $body = $null; $failure = ''
    try {
        $response = Invoke-WebRequest -Uri "$base/api/v1$path" -WebSession $script:session -UseBasicParsing -TimeoutSec 50
        $status = [int]$response.StatusCode
        $body = $response.Content | ConvertFrom-Json
    } catch {
        if ($_.Exception.Response) { $status = [int]$_.Exception.Response.StatusCode }
        $failure = $_.Exception.GetType().Name
    }
    $sw.Stop()
    $entry = [ordered]@{phase=$phase;endpoint=$endpoint;ms=[math]::Round($sw.Elapsed.TotalMilliseconds,1);status=$status;error=$failure;time=[DateTime]::UtcNow.ToString('o')}
    if ($body) {
        if ($endpoint -eq 'dashboard') { $entry.total = $body.total }
        if ($endpoint -eq 'processing') {
            $entry.stats_available = $body.stats_available
            $entry.total = $body.stats.total
            $entry.scans = @($body.scans).Count
            $entry.active_jobs = @($body.active_jobs).Count
            $entry.scan_status = @($body.scans | ForEach-Object { $_.status }) -join ','
            $entry.files_seen = [int](($body.scans | Measure-Object -Property files -Sum).Sum)
        }
        if ($endpoint -eq 'folders_root' -or $endpoint -eq 'folders_nested') { $entry.items = @($body.items).Count }
        if ($endpoint -like 'explorer_*') { $entry.items = @($body.items).Count; $entry.result_count = $body.result_count }
    }
    $script:measurements.Add([pscustomobject]$entry)
    Write-Host "$phase $endpoint $($entry.ms) ms HTTP $status"
    return $body
}
function Verify([string]$phase) {
    $body = @{root_id=$target.root_id} | ConvertTo-Json -Compress
    try {
        $response = Invoke-WebRequest -Uri "$base/api/v1/libraries/$($target.library_id)/verify" -Method Post -Body $body -ContentType 'application/json' -Headers @{Origin=$base;'X-CSRF-Token'=$script:csrf} -WebSession $script:session -UseBasicParsing -TimeoutSec 50
        Write-Host "$phase verification HTTP $($response.StatusCode)"
    } catch { Write-Host "$phase verification: $($_.Exception.Message)" }
}
function Phase([string]$phase) {
    Login
    Verify $phase
    $sampler = Start-Job -ArgumentList $OutputDirectory,$phase -ScriptBlock {
        param($dir,$label)
        $csv = Join-Path $dir "resources-$label.csv"
        for ($i=0; $i -lt 300; $i++) {
            $p = Get-Process -Name aidex-server -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($p) { [pscustomobject]@{time=[DateTime]::UtcNow.ToString('o');pid=$p.Id;cpu_seconds=$p.CPU;working_set_bytes=$p.WorkingSet64;private_bytes=$p.PrivateMemorySize64;handles=$p.HandleCount;threads=$p.Threads.Count} | Export-Csv -LiteralPath $csv -NoTypeInformation -Append -Encoding UTF8 }
            Start-Sleep -Seconds 1
        }
    }
    try {
        for ($round=1; $round -le 3; $round++) {
            $tag = "${phase}_$round"
            Request $tag 'session' '/auth/session' | Out-Null
            Request $tag 'dashboard' '/system/dashboard' | Out-Null
            Request $tag 'dashboard_libraries' '/system/dashboard/libraries' | Out-Null
            Request $tag 'libraries' '/libraries' | Out-Null
            Request $tag 'processing' "/libraries/$($target.library_id)/processing" | Out-Null
            $folders = Request $tag 'folders_root' "/libraries/$($target.library_id)/folders?root_id=$($target.root_id)"
            Request $tag 'explorer_root' "/libraries/$($target.library_id)/explorer?root_id=$($target.root_id)" | Out-Null
            Request $tag 'explorer_root_fast' "/libraries/$($target.library_id)/explorer?root_id=$($target.root_id)&include_total=false" | Out-Null
            $prefix = @($folders.items | Select-Object -First 1).relative_prefix
            if ($prefix) {
                $encoded = [uri]::EscapeDataString($prefix)
                Request $tag 'folders_nested' "/libraries/$($target.library_id)/folders?root_id=$($target.root_id)&prefix=$encoded" | Out-Null
                Request $tag 'explorer_nested' "/libraries/$($target.library_id)/explorer?root_id=$($target.root_id)&prefix=$encoded" | Out-Null
                Request $tag 'explorer_nested_fast' "/libraries/$($target.library_id)/explorer?root_id=$($target.root_id)&prefix=$encoded&include_total=false" | Out-Null
            }
            $script:measurements | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $OutputDirectory 'measurements.json') -Encoding UTF8
        }
    } finally {
        Stop-Job $sampler -ErrorAction SilentlyContinue
        Receive-Job $sampler -ErrorAction SilentlyContinue | Out-Null
        Remove-Job $sampler -ErrorAction SilentlyContinue
    }
}

$measurements = New-Object 'System.Collections.Generic.List[object]'
if ($ResumeAfterBaseline) {
    $prior = Get-Content -LiteralPath (Join-Path $OutputDirectory 'measurements.json') -Raw | ConvertFrom-Json
    foreach ($entry in $prior) { if ($entry.phase -like 'before_*') { $measurements.Add($entry) } }
}
try {
    if ($ResumeAfterBaseline) {
        $metadata = Get-Content -LiteralPath (Join-Path $OutputDirectory 'metadata.json') -Raw | ConvertFrom-Json
        if ($metadata.library_id -ne $target.library_id -or $metadata.root_id -ne $target.root_id) { throw 'La biblioteca real cambió desde la línea base.' }
        $currentHash = (Get-FileHash -LiteralPath $installed -Algorithm SHA256).Hash
        if ($currentHash -ne $metadata.installed_sha256 -and $currentHash -ne $metadata.replacement_sha256) { throw 'El binario actual no coincide con la línea base ni con el intento anterior.' }
        $metadata.replacement_sha256 = (Get-FileHash -LiteralPath $replacement -Algorithm SHA256).Hash
    } else {
        $metadata = [ordered]@{started=[DateTime]::UtcNow.ToString('o');library_id=$target.library_id;root_id=$target.root_id;documents_before=$target.documents;installed_sha256=(Get-FileHash -LiteralPath $installed -Algorithm SHA256).Hash;replacement_sha256=(Get-FileHash -LiteralPath $replacement -Algorithm SHA256).Hash}
        $metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $OutputDirectory 'metadata.json') -Encoding UTF8
        Phase 'before'
    }
    & (Join-Path $PSScriptRoot 'apply-performance-windows.ps1') -Source $replacement -Target $installed
    if ($LASTEXITCODE -ne 0) { throw 'Falló la actualización del servicio.' }
    Start-Sleep -Seconds 5
    Phase 'after'
    if ($metadata -is [System.Collections.IDictionary]) { $metadata.completed = [DateTime]::UtcNow.ToString('o') } else { $metadata | Add-Member -NotePropertyName completed -NotePropertyValue ([DateTime]::UtcNow.ToString('o')) -Force }
    $metadata | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $OutputDirectory 'metadata.json') -Encoding UTF8
} catch {
    ($_ | Out-String) | Set-Content -LiteralPath (Join-Path $OutputDirectory 'error.txt') -Encoding UTF8
    throw
} finally {
    & $helper disable $dbPath $name | Out-Null
    $password = $null
    $measurements | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $OutputDirectory 'measurements.json') -Encoding UTF8
}
