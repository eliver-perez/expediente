$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$config = Get-Content -LiteralPath 'C:\ProgramData\AIDEX\config.json' -Raw | ConvertFrom-Json
$database = $config.state_directory
$output = Join-Path $root '.tools\real-benchmark\migrations.json'
$entries = & (Join-Path $root 'build\aidex-benchmark-user.exe') migrations $database ignored | ConvertFrom-Json
$result = foreach ($entry in $entries) {
    $path = Join-Path $root ('db\migrations\' + $entry.name)
    $current = if (Test-Path -LiteralPath $path) { (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant() } else { '' }
    [pscustomobject]@{name=$entry.name;installed=$entry.checksum;source=$current;match=($entry.checksum -eq $current)}
}
$result | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath $output -Encoding UTF8
Write-Host "Revisión de migraciones: $output"
