$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$helper = Join-Path $workspace 'build\aidex-benchmark-user.exe'
$output = Join-Path $workspace '.tools\real-benchmark\deletion-query-plan.json'
$result = & $helper deletionplanquery 'C:\ProgramData\AIDEX\state\documental.db' '2644f1e4-2d90-4e34-91b6-b941246b46b4'
if ($LASTEXITCODE -ne 0) { throw 'No se pudo explicar la consulta de páginas.' }
$result | Set-Content -LiteralPath $output -Encoding UTF8
