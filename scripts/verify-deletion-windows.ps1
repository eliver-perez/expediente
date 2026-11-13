$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$helper = Join-Path $workspace 'build\aidex-deletion-verify.exe'
$database = 'C:\ProgramData\AIDEX\state\documental.db'
$operation = '37421a3f-89c8-4986-89bb-eb6a12e036af'
$output = Join-Path $workspace '.tools\real-benchmark\deletion-verified.json'
$errorOutput = Join-Path $workspace '.tools\real-benchmark\deletion-verify-error.txt'
try {
    $result = & $helper deletionverify $database $operation
    if ($LASTEXITCODE -ne 0) { throw "La verificación terminó con código $LASTEXITCODE" }
    [System.IO.File]::WriteAllText($output, ($result | Out-String), [System.Text.UTF8Encoding]::new($false))
} catch {
    $_ | Out-String | Set-Content -LiteralPath $errorOutput -Encoding UTF8
    throw
}
