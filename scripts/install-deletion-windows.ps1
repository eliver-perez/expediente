$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$output = Join-Path $workspace '.tools\real-benchmark\deletion-install.txt'
try {
    $result = & (Join-Path $PSScriptRoot 'apply-performance-windows.ps1') 2>&1
    $result | Out-String | Set-Content -LiteralPath $output -Encoding UTF8
    if ($LASTEXITCODE -ne 0) { throw 'Falló la sustitución del servicio.' }
} catch {
    $_ | Out-String | Set-Content -LiteralPath $output -Encoding UTF8
    throw
}
