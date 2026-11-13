param(
    [string]$Database = 'C:\ProgramData\AIDEX\state\documental.db',
    [string]$Output = ''
)
$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$helper = Join-Path $workspace 'build\aidex-benchmark-user.exe'
if (-not $Output) { $Output = Join-Path $workspace '.tools\real-benchmark\deletion-before.json' }
$result = & $helper deletiondiag $Database unused
if ($LASTEXITCODE -ne 0) { throw 'Falló la lectura del historial de eliminaciones.' }
[System.IO.File]::WriteAllText($Output, ($result | Out-String), [System.Text.UTF8Encoding]::new($false))
$items = $result | ConvertFrom-Json
$failed = $items | Where-Object { $_.kind -eq 'library' -and $_.execution -eq 'failed' } | Select-Object -First 1
if ($failed) {
    $state = & $helper deletionstate $Database $failed.library_id
    [System.IO.File]::WriteAllText([System.IO.Path]::ChangeExtension($Output, '.state.json'), ($state | Out-String), [System.Text.UTF8Encoding]::new($false))
    $samples = & $helper deletionsample $Database $failed.id | ConvertFrom-Json | Select-Object -First 5
    $verified = foreach ($sample in $samples) {
        $paths = $sample.locations_json | ConvertFrom-Json
        $path = @($paths | Where-Object { Test-Path -LiteralPath $_ }) | Select-Object -First 1
        if ($path) {
            $file = Get-Item -LiteralPath $path
            [pscustomobject]@{ path=$path; size=$file.Length; sha256=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash; manifest_sha256=$sample.sha256 }
        }
    }
    $verified | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath ([System.IO.Path]::ChangeExtension($Output, '.sample.json')) -Encoding UTF8
}
$logOutput = [System.IO.Path]::ChangeExtension($Output, '.logs.txt')
$lines = [System.Collections.Generic.List[string]]::new()
Get-ChildItem 'C:\ProgramData\AIDEX' -Recurse -File -Include '*.log','*.jsonl' -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Descending | Select-Object -First 8 | ForEach-Object {
        $lines.Add("LOG $($_.FullName)")
        Select-String -LiteralPath $_.FullName -Pattern 'deletion execution failed','slow deletion SQL step','deletion failure state' -ErrorAction SilentlyContinue |
            Select-Object -Last 30 | ForEach-Object { $lines.Add($_.Line) }
    }
[System.IO.File]::WriteAllLines($logOutput, $lines, [System.Text.UTF8Encoding]::new($false))
