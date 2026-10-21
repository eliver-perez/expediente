#Requires -Version 5.1
# Isolated mocks/transport only. Does not install or change real services.
$ErrorActionPreference = 'Stop'
function Assert-True($Condition, $Message) { if (!$Condition) { throw $Message } }
$Source = Join-Path $PSScriptRoot '../packaging/windows/manage.ps1'
$Tokens = $null; $Errors = $null
$Ast = [Management.Automation.Language.Parser]::ParseFile($Source, [ref]$Tokens, [ref]$Errors)
Assert-True (!$Errors.Count) 'Invalid manage script'
foreach ($Name in @('Get-AidexServiceCommand','Assert-AidexServiceOwnership','ConvertTo-AidexNativeArgument','Write-LauncherTarget')) {
    $Nodes = @($Ast.FindAll({param($Node) $Node -is [Management.Automation.Language.FunctionDefinitionAst] -and $Node.Name -eq $Name}, $true))
    Assert-True ($Nodes.Count -eq 1) "Missing $Name"
    . ([scriptblock]::Create($Nodes[0].Extent.Text))
}
$Root = Join-Path ([IO.Path]::GetTempPath()) ('AIDEX José María López Muñoz ü 東京 ' + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $Root | Out-Null
$CommonData = $Root
$InstallRoot = Join-Path $Root 'Programa con espacios'
$Binary = Join-Path $InstallRoot 'aidex-server.exe'
$script:Services = @{}
function Get-Service { param($Name, $ErrorAction) return $script:Services[$Name] }
function Get-CimInstance { param($ClassName,$Filter,$ErrorAction) return $script:Installed }
function Reset-Profile {
    $script:ServiceName = 'AIDEX'
    $script:DataRoot = Join-Path $CommonData 'AIDEX'
    $script:Config = Join-Path $DataRoot 'config.json'
}
try {
    $Argument = 'C:\María López\Muñoz\'
    Assert-True ((ConvertTo-AidexNativeArgument $Argument) -eq ('"' + $Argument + '\"')) 'Trailing backslash escaped incorrectly'
    Assert-True ((ConvertTo-AidexNativeArgument '"José"') -eq '"\"José\""') 'Embedded service quotes lost'
    Reset-Profile
    $script:Installed = $null
    Assert-AidexServiceOwnership
    $script:Installed = [pscustomobject]@{PathName=(Get-AidexServiceCommand);StartName='NT SERVICE\AIDEX'}
    Assert-AidexServiceOwnership
    $script:Installed.PathName = Get-AidexServiceCommand
    Assert-AidexServiceOwnership
    $script:Installed.PathName = 'unrelated.exe'
    $Rejected = $false
    try { Assert-AidexServiceOwnership } catch { $Rejected = $true }
    Assert-True $Rejected 'Foreign service overwritten'
    $script:Installed.PathName = Get-AidexServiceCommand
    $script:Installed.StartName = 'LocalSystem'
    $Rejected = $false
    try { Assert-AidexServiceOwnership } catch { $Rejected = $true }
    Assert-True $Rejected 'Foreign account accepted'

    # Go writes BOM-less UTF-8 JSON; Windows PowerShell 5.1 must decode it explicitly.
    New-Item -ItemType Directory -Path $InstallRoot | Out-Null
    $Config = Join-Path $Root 'Configuración ñ.json'
    $URL = 'https://biblioteca-ñ.example'
    $JSON = @{ public_url = $URL; license = @{ secret = 'TEST_ONLY_DO_NOT_EXPORT' } } | ConvertTo-Json
    [IO.File]::WriteAllText($Config, $JSON, [Text.UTF8Encoding]::new($false))
    Write-LauncherTarget
    $Saved = [IO.File]::ReadAllText((Join-Path $InstallRoot 'launcher.json'), [Text.Encoding]::UTF8) | ConvertFrom-Json
    Assert-True ($Saved.public_url -eq $URL -and $Saved.service_name -eq 'AIDEX') 'UTF-8 configuration corrupted'
    Assert-True ($Saved.PSObject.Properties.Name.Count -eq 2) 'Private configuration published'

    # On macOS replace only process launch/WindowStyle with pwsh. The actual
    # encoded command, PowerShell streams and UTF-16LE writer remain unchanged.
    if ($PSVersionTable.Platform -eq 'Unix') {
        function Start-Process {
            param($FilePath,$ArgumentList,$WindowStyle,[switch]$PassThru)
            Microsoft.PowerShell.Management\Start-Process -FilePath (Join-Path $PSHOME 'pwsh') -ArgumentList $ArgumentList -PassThru
        }
    }
    $Manage = Join-Path $Root 'Simulación Ñ.ps1'
    $Log = Join-Path $Root 'Registro áéíóú.log'
    $Message = 'ÁÉÍÓÚ áéíóú Ññ ü ¿Instalar? ¡Atención! 東京'
    $Fake = 'param($Action,$InstallRoot,$PreflightBinary,[switch]$EraseInternalData)' + [Environment]::NewLine +
            'Write-Host ' + "'" + $Message + "'" + [Environment]::NewLine + 'Write-Output $InstallRoot' + [Environment]::NewLine + 'exit 7'
    [IO.File]::WriteAllText($Manage, $Fake, [Text.UTF8Encoding]::new($true))
    & (Join-Path $PSScriptRoot '../packaging/windows/installer-action.ps1') -ManageScript $Manage -Action Configure -InstallRoot $Root -LogPath $Log -TimeoutSeconds 30
    Assert-True ($LASTEXITCODE -eq 7) "Lost subprocess exit status: $LASTEXITCODE"
    $Bytes = [IO.File]::ReadAllBytes($Log)
    Assert-True ($Bytes[0] -eq 0xff -and $Bytes[1] -eq 0xfe) 'Missing UTF-16LE BOM'
    $Text = [IO.File]::ReadAllText($Log, [Text.Encoding]::Unicode)
    Assert-True ($Text.Contains($Message) -and $Text.Contains($Root)) 'Unicode message or path corrupted'
    Write-Host 'PASS: production service identity, foreign path/account rejection, Unicode paths/messages and exit status.'
} finally {
    Remove-Item -LiteralPath $Root -Recurse -Force
}
