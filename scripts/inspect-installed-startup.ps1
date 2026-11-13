$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$config = Get-Content -LiteralPath 'C:\ProgramData\AIDEX\config.json' -Raw | ConvertFrom-Json
$database = $config.state_directory
$output = Join-Path $root '.tools\real-benchmark\startup.json'
$process = Get-Process -Name aidex-server -ErrorAction SilentlyContinue | Select-Object -First 1
$files = foreach ($suffix in @('', '-wal', '-shm')) {
    $path = $database + $suffix
    if (Test-Path -LiteralPath $path) { $item = Get-Item -LiteralPath $path; [pscustomobject]@{name=$item.Name;bytes=$item.Length;updated=$item.LastWriteTimeUtc.ToString('o')} }
}
$stats = & (Join-Path $root 'build\aidex-benchmark-user.exe') stats $database ignored | ConvertFrom-Json
$eula = & (Join-Path $root 'build\aidex-benchmark-user.exe') eula $database ignored | ConvertFrom-Json
$ocr = & (Join-Path $root 'build\aidex-benchmark-user.exe') ocrcheck $database ignored | ConvertFrom-Json
$result = [pscustomobject]@{time=[DateTime]::UtcNow.ToString('o');pid=$process.Id;cpu_seconds=$process.CPU;working_set_bytes=$process.WorkingSet64;private_bytes=$process.PrivateMemorySize64;handles=$process.HandleCount;files=$files;tables=$stats;eula=$eula;ocr=$ocr}
$result | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $output -Encoding UTF8
