$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$folder=Join-Path $root '.tools\real-benchmark'
try {
    $config=Get-Content 'C:\ProgramData\AIDEX\config.json' -Raw|ConvertFrom-Json
    $database=$config.state_directory
    $output=Join-Path $folder 'job-query.json'
    $result=& (Join-Path $root 'build\aidex-benchmark-user.exe') jobbench $database ignored 2>&1
    if($LASTEXITCODE -ne 0){throw ($result|Out-String)}
    $result | Set-Content -LiteralPath $output -Encoding UTF8
} catch {
    ($_|Out-String)|Set-Content -LiteralPath (Join-Path $folder 'job-query-error.txt') -Encoding UTF8
    throw
}
