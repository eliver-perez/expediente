#Requires -Version 5.1
param([Parameter(Mandatory=$true)][string]$Path)
$ErrorActionPreference = 'Stop'

# Parse the actual generated payload without executing installer actions.
$Bytes = [IO.File]::ReadAllBytes($Path)
if ($Bytes.Length -lt 3 -or $Bytes[0] -ne 0xef -or $Bytes[1] -ne 0xbb -or $Bytes[2] -ne 0xbf) {
    throw 'Windows payload must have one UTF-8 BOM.'
}
$Text = [Text.UTF8Encoding]::new($false, $true).GetString($Bytes, 3, $Bytes.Length - 3)
if ($Text.StartsWith([string][char]0xfeff, [StringComparison]::Ordinal)) { throw 'Duplicate UTF-8 BOM in Windows payload.' }
if ($Text.Contains('@VERSION@')) { throw 'Unresolved package version in Windows payload.' }
$Tokens = $null
$Issues = $null
$Ast = [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$Tokens, [ref]$Issues)
if ($Issues.Count -gt 0) {
    foreach ($Issue in $Issues) {
        Write-Host ("Line {0}: {1}: {2}" -f $Issue.Extent.StartLineNumber, $Issue.ErrorId, $Issue.Message)
    }
    throw 'Invalid PowerShell installer payload.'
}
if ($null -eq $Ast.ParamBlock) { throw 'Installer parameter block was not recognized.' }
$Parameters = @($Ast.ParamBlock.Parameters | ForEach-Object { $_.Name.VariablePath.UserPath })
foreach ($Required in @('Action', 'InstallRoot', 'PreflightBinary', 'EraseInternalData')) {
    if ($Required -notin $Parameters) { throw "Missing installer parameter: $Required" }
}
Write-Host "Windows PowerShell payload: encoding, syntax and parameters OK (parser $($PSVersionTable.PSVersion))"
