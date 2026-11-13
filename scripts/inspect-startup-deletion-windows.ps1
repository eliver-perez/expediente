$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$output = Join-Path $workspace '.tools\real-benchmark\deletion-startup.json'
$service = Get-CimInstance Win32_Service -Filter "Name='AIDEX'"
$process = if ($service.ProcessId) { Get-Process -Id $service.ProcessId -ErrorAction SilentlyContinue }
$files = Get-Item 'C:\ProgramData\AIDEX\state\postgresql\data\PG_VERSION' -ErrorAction SilentlyContinue |
    Select-Object Name,Length,LastWriteTime
$result = [pscustomobject]@{
    time=[DateTime]::UtcNow.ToString('o'); service_state=$service.State; pid=$service.ProcessId;
    cpu_seconds=$process.CPU; handles=$process.HandleCount; working_set_bytes=$process.WorkingSet64;
    files=$files; log_tail=Get-Content 'C:\ProgramData\AIDEX\logs\service.log' -Tail 6
}
$result | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $output -Encoding UTF8
