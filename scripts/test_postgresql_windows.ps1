param([string[]]$GoArgs = @('test', './...'))

$ErrorActionPreference = 'Stop'
$workspace = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$runtime = Join-Path $workspace '.tools\postgresql-windows\runtime\pgsql\bin'
$goRoot = Join-Path $workspace '.tools\go-sdk\go'
$base = Join-Path $workspace '.tools\postgresql-windows\sessions'
New-Item -ItemType Directory -Force -Path $base | Out-Null
$session = Join-Path $base ('pg cluster with spaces ' + [guid]::NewGuid().ToString('N'))
$session = [IO.Path]::GetFullPath($session)
$base = [IO.Path]::GetFullPath($base)
if (-not $session.StartsWith($base + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'La ruta temporal de PostgreSQL está fuera de .tools.'
}
New-Item -ItemType Directory -Path $session | Out-Null
$data = Join-Path $session 'data'
$log = Join-Path $session 'server.log'
$passwordFile = Join-Path $session 'password.txt'
$listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
$listener.Start()
$port = ([Net.IPEndPoint]$listener.LocalEndpoint).Port
$listener.Stop()
$bytes = New-Object byte[] 32
$random = [Security.Cryptography.RandomNumberGenerator]::Create()
$random.GetBytes($bytes)
$random.Dispose()
$password = [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+','A').Replace('/','B')
[IO.File]::WriteAllText($passwordFile, $password + "`n", [Text.UTF8Encoding]::new($false))

function Invoke-PgCtl([string]$arguments) {
    $start = [Diagnostics.ProcessStartInfo]::new((Join-Path $runtime 'pg_ctl.exe'))
    $start.Arguments = $arguments
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $process = [Diagnostics.Process]::Start($start)
    try {
        if (-not $process.WaitForExit(60000)) { throw 'pg_ctl excedió 60 segundos.' }
        if ($process.ExitCode -ne 0) { throw "pg_ctl terminó con código $($process.ExitCode). Consulta $log" }
    } finally { $process.Dispose() }
}

$started = $false
try {
    $env:PATH = "$runtime;$goRoot\bin;$env:PATH"
    $init = & (Join-Path $runtime 'initdb.exe') -D $data --username=aidex_test "--pwfile=$passwordFile" --auth=scram-sha-256 --encoding=UTF8 --locale=C --data-checksums 2>&1
    if ($LASTEXITCODE -ne 0) { throw "initdb falló: $($init | Out-String)" }
    Remove-Item -LiteralPath $passwordFile
    Invoke-PgCtl ('-D "' + $data + '" -l "' + $log + '" -o "-h 127.0.0.1 -p ' + $port + '" -w -t 30 start')
    $started = $true
    # Fixtures use independent schemas in one database. Install the one global
    # extension first so parallel migrations cannot race on pg_extension.
    $env:PGPASSWORD = $password
    & (Join-Path $runtime 'psql.exe') -X -q -v ON_ERROR_STOP=1 -h 127.0.0.1 -p $port -U aidex_test -d postgres -c 'CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'No se pudo crear unaccent en el clúster temporal.' }
    Remove-Item Env:\PGPASSWORD
    $env:AIDEX_TEST_POSTGRES_DSN = "postgresql://aidex_test:$password@127.0.0.1:$port/postgres?sslmode=disable"
    $env:AIDEX_POSTGRES_BIN = $runtime
    $env:GOROOT = $goRoot
    $env:GOMODCACHE = Join-Path $workspace '.tools\gomodcache'
    $env:GOCACHE = Join-Path $workspace '.tools\gocache'
    $env:TEMP = Join-Path $workspace '.tools\postgresql-windows\testtmp'
    $env:TMP = $env:TEMP
    & (Join-Path $goRoot 'bin\go.exe') @GoArgs
    if ($LASTEXITCODE -ne 0) { throw "go $($GoArgs -join ' ') terminó con código $LASTEXITCODE" }
} finally {
    Remove-Item Env:\AIDEX_TEST_POSTGRES_DSN,Env:\AIDEX_POSTGRES_BIN,Env:\PGPASSWORD -ErrorAction SilentlyContinue
    $password = $null
    if ($started) {
        try { Invoke-PgCtl ('-D "' + $data + '" -m immediate -w -t 30 stop') } catch { Write-Warning $_ }
    }
    if (Test-Path -LiteralPath $session) { Remove-Item -LiteralPath $session -Recurse -Force }
}
