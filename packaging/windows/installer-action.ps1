#Requires -Version 5.1
param([Parameter(Mandatory=$true)][string]$ManageScript,
      [Parameter(Mandatory=$true)][ValidateSet('Preflight','Configure','Remove')][string]$Action,
      [Parameter(Mandatory=$true)][string]$InstallRoot,
      [Parameter(Mandatory=$true)][string]$LogPath,
      [string]$PreflightBinary = '', [switch]$EraseInternalData,
      [ValidateRange(1,900)][int]$TimeoutSeconds = 600)
$ErrorActionPreference = 'Stop'
# NSIS reads this file as UTF-16LE. No ACP/OEM conversion or global codepage change.
function Quote-AidexLiteral([string]$Value) { return "'" + $Value.Replace("'", "''") + "'" }
[IO.File]::WriteAllText($LogPath, '', [Text.Encoding]::Unicode)
$Invocation = '& ' + (Quote-AidexLiteral $ManageScript) + ' -Action ' + (Quote-AidexLiteral $Action) + ' -InstallRoot ' + (Quote-AidexLiteral $InstallRoot)
if ($PreflightBinary) { $Invocation += ' -PreflightBinary ' + (Quote-AidexLiteral $PreflightBinary) }
if ($EraseInternalData) { $Invocation += ' -EraseInternalData' }
$Command = $Invocation + ' *>&1 | ForEach-Object { [IO.File]::AppendAllText(' + (Quote-AidexLiteral $LogPath) + ', $_.ToString() + [Environment]::NewLine, [Text.Encoding]::Unicode) }; exit $LASTEXITCODE'
$Encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($Command))
try {
    $Process = Start-Process -FilePath "$PSHOME\powershell.exe" -ArgumentList ('-NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand ' + $Encoded) -WindowStyle Hidden -PassThru
    if (!$Process.WaitForExit($TimeoutSeconds * 1000)) {
        # Terminate the entire installer subprocess tree, not just PowerShell.
        & "$env:SystemRoot\System32\taskkill.exe" /PID $Process.Id /T /F | Out-Null
        throw "La operación excedió el tiempo límite de $TimeoutSeconds segundos. Revisa el registro antes de reintentar."
    }
    exit $Process.ExitCode
} catch {
    [IO.File]::AppendAllText($LogPath, $_.Exception.Message + [Environment]::NewLine, [Text.Encoding]::Unicode)
    exit 1
}
