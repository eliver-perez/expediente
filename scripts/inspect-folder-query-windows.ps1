$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$folder=Join-Path $root '.tools\real-benchmark'
try {
    $config=Get-Content 'C:\ProgramData\AIDEX\config.json' -Raw|ConvertFrom-Json
    $database=Join-Path $config.state_directory 'documental.db'
    $target=Get-Content (Join-Path $folder 'metadata.json') -Raw|ConvertFrom-Json
    $result=& (Join-Path $root 'build\aidex-benchmark-user.exe') folderbench $database $target.root_id 2>&1
    if($LASTEXITCODE -ne 0){throw ($result|Out-String)}
    $result|Set-Content -LiteralPath (Join-Path $folder 'folder-query.json') -Encoding UTF8
} catch {
    ($_|Out-String)|Set-Content -LiteralPath (Join-Path $folder 'folder-query-error.txt') -Encoding UTF8
    throw
}
