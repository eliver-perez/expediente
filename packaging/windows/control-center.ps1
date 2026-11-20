#Requires -Version 5.1
param([ValidateSet('Start','Stop','Restart','Port','Setup')][string]$Operation,
      [int]$Port = 0)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
Add-Type -Path (Join-Path $PSScriptRoot 'control-appearance.cs') -ReferencedAssemblies @('System.Windows.Forms.dll','System.Drawing.dll')
[Windows.Forms.Application]::EnableVisualStyles()

$Ink = [Drawing.ColorTranslator]::FromHtml('#20373d')
$Muted = [Drawing.ColorTranslator]::FromHtml('#536864')
$Navy = [Drawing.ColorTranslator]::FromHtml('#132e38')
$Accent = [Drawing.ColorTranslator]::FromHtml('#256c5c')
$Background = [Drawing.ColorTranslator]::FromHtml('#f4f6f1')
$Manage = Join-Path $PSScriptRoot 'manage.ps1'
$Server = Join-Path $PSScriptRoot 'aidex-server.exe'
$Launcher = Join-Path $PSScriptRoot 'AIDEX.exe'
$BrandIcon = Join-Path $PSScriptRoot 'aidex.ico'
$Data = Join-Path ([Environment]::GetFolderPath('CommonApplicationData')) 'AIDEX'

function Show-Notice([string]$Message, [Windows.Forms.MessageBoxIcon]$Icon = 'Information') {
    [Windows.Forms.MessageBox]::Show($Message, 'Centro de Control de AIDEX', [Windows.Forms.MessageBoxButtons]::OK, $Icon) | Out-Null
}
function New-Button([string]$Text, [bool]$Primary = $false) {
    $Button = [Aidex.ControlCenter.ActionButton]::new()
    $Button.Text = $Text
    $Button.AutoSize = $true
    $Button.MinimumSize = [Drawing.Size]::new(0, 36)
    $Button.Padding = [Windows.Forms.Padding]::new(8, 0, 8, 0)
    $Button.FlatStyle = 'Flat'
    $Button.FlatAppearance.BorderColor = $Accent
    if ($Primary) { $Button.BackColor = $Accent; $Button.ForeColor = [Drawing.Color]::White }
    else { $Button.BackColor = [Drawing.Color]::White; $Button.ForeColor = $Ink }
    return $Button
}
function New-Label([string]$Text, [bool]$Heading = $false) {
    $Label = [Windows.Forms.Label]::new()
    $Label.Text = $Text
    $Label.AutoSize = $true
    $Label.ForeColor = if ($Heading) { $Ink } else { $Muted }
    if ($Heading) { $Label.Font = [Drawing.Font]::new('Segoe UI', [single]11, [Drawing.FontStyle]::Bold) }
    return $Label
}
function New-Form([string]$Title, [Drawing.Size]$Size) {
    $Window = [Aidex.ControlCenter.BrandedForm]::new($Launcher, $BrandIcon)
    $Window.Text = $Title
    $Window.StartPosition = 'CenterScreen'
    $Window.Font = [Drawing.Font]::new('Segoe UI', [single]10)
    $Window.ClientSize = $Size
    $Window.AutoScaleMode = 'Font'
    $Window.BackColor = $Background
    $Window.MinimizeBox = $false
    $Window.MaximizeBox = $false
    $Window.FormBorderStyle = 'FixedDialog'
    return $Window
}
function New-Column {
    $Panel = [Windows.Forms.FlowLayoutPanel]::new()
    $Panel.FlowDirection = 'TopDown'
    $Panel.WrapContents = $false
    $Panel.AutoSize = $true
    $Panel.AutoSizeMode = 'GrowAndShrink'
    return $Panel
}
function New-Row {
    $Panel = [Windows.Forms.FlowLayoutPanel]::new()
    $Panel.WrapContents = $false
    $Panel.AutoSize = $true
    $Panel.AutoSizeMode = 'GrowAndShrink'
    return $Panel
}

function Invoke-Management([string]$Action, [int]$SelectedPort = 0) {
    $Messages = @(& $Manage -Action $Action -Port $SelectedPort *>&1)
    if ($LASTEXITCODE -ne 0) {
        $Detail = $Messages -join ' '
        if ($Detail -match 'ocupado|address already in use|sólo se permite un uso') {
            throw 'Ese puerto ya está ocupado. Elige otro e inténtalo de nuevo.'
        }
        if ($Detail -match 'necesita un administrador inicial') { throw 'Completa la configuración inicial antes de iniciar AIDEX.' }
        if ($Action -eq 'Port') { throw 'No se pudo cambiar el puerto. Revisa el estado del servicio y del firewall antes de reintentar.' }
        if ($Action -like 'Setup*') { throw 'No se pudo completar la configuración inicial. Consulta Diagnóstico y vuelve a intentarlo.' }
        throw 'Windows no pudo completar la operación del servicio. Consulta Diagnóstico.'
    }
    return $Messages
}
function Submit-FirstAdministrator([string]$User, [string]$Name, [string]$Password, [string]$Confirmation) {
    Add-Type -AssemblyName System.Web.Extensions
    $Settings = [Diagnostics.ProcessStartInfo]::new($Server)
    $Config = Join-Path $Data 'config.json'
    $Settings.Arguments = 'bootstrap-ui --config "' + $Config + '" --windows-service AIDEX'
    $Settings.UseShellExecute = $false
    $Settings.CreateNoWindow = $true
    $Settings.RedirectStandardInput = $true
    $Settings.RedirectStandardError = $true
    $Settings.RedirectStandardOutput = $true
    $Settings.StandardErrorEncoding = [Text.Encoding]::UTF8
    $Child = [Diagnostics.Process]::Start($Settings)
    $Bytes = $null
    try {
        $Errors = $Child.StandardError.ReadToEndAsync()
        $Output = $Child.StandardOutput.ReadToEndAsync()
        $Serializer = [System.Web.Script.Serialization.JavaScriptSerializer]::new()
        $Bytes = [Text.Encoding]::UTF8.GetBytes($Serializer.Serialize(@{
            username = $User; display_name = $Name; password = $Password; confirmation = $Confirmation
        }))
        $InputStream = $Child.StandardInput.BaseStream
        $InputStream.Write($Bytes, 0, $Bytes.Length)
        $InputStream.Close()
        if (!$Child.WaitForExit(120000)) { throw 'AIDEX tardó demasiado en responder. Comprueba si se creó el administrador antes de reintentar.' }
        if ($Child.ExitCode -ne 0) {
            $Message = $Errors.GetAwaiter().GetResult()
            if ($Message -match 'ya fue creado|existen usuarios') { throw 'El administrador inicial ya existe. Puedes iniciar AIDEX desde el Centro de Control.' }
            if ($Message -match 'contraseña|usuario|nombre') { throw 'Revisa el nombre, el usuario y la contraseña. El usuario admite letras inglesas, números, punto, guion, @ y guion bajo.' }
            throw 'No se pudo guardar el administrador. Consulta Diagnóstico antes de reintentar.'
        }
    } finally {
        if ($Bytes) { [Array]::Clear($Bytes, 0, $Bytes.Length) }
        $Child.Dispose()
    }
}
function Invoke-SetupWizard {
    $State = Invoke-Management 'SetupStatus'
    if ($State -contains 'configured') {
        Show-Notice 'El administrador inicial ya existe. Usa Iniciar si el servicio está detenido.'
        return
    }
    if ($State -notcontains 'required') { throw 'No se pudo confirmar que la instalación sea nueva.' }

    $Wizard = New-Form 'AIDEX · Configuración inicial' ([Drawing.Size]::new(490, 460))
    $Fields = New-Column
    $Fields.Dock = 'Fill'
    $Fields.Padding = [Windows.Forms.Padding]::new(22)
    $Fields.Controls.Add((New-Label 'Crea la primera cuenta de administración' $true))
    $Fields.Controls.Add((New-Label 'Usa esta cuenta para entrar a AIDEX desde el navegador.'))
    $Inputs = @{}
    foreach ($Field in @('Usuario', 'Nombre visible', 'Contraseña', 'Repite la contraseña')) {
        $Caption = New-Label $Field
        $Caption.Margin = [Windows.Forms.Padding]::new(3, 13, 3, 2)
        $Fields.Controls.Add($Caption)
        $Box = [Windows.Forms.TextBox]::new()
        $Box.Width = 410
        $Box.AccessibleName = $Field
        $Box.UseSystemPasswordChar = $Field -like '*contraseña*'
        $Box.MaxLength = if ($Field -eq 'Usuario') { 80 } elseif ($Box.UseSystemPasswordChar) { 256 } else { 120 }
        $Fields.Controls.Add($Box)
        $Inputs[$Field] = $Box
    }
    $Fields.Controls.Add((New-Label 'Contraseña de 6 a 128 caracteres.'))
    $Actions = New-Row
    $Create = New-Button 'Crear administrador' $true
    $Cancel = New-Button 'Cancelar'
    $Create.DialogResult = 'OK'
    $Cancel.DialogResult = 'Cancel'
    $Create.Add_Click({
        if (!$Inputs['Usuario'].Text.Trim() -or !$Inputs['Nombre visible'].Text.Trim() -or $Inputs['Contraseña'].Text.Length -lt 6) {
            Show-Notice 'Completa el usuario, el nombre visible y una contraseña de al menos 6 caracteres.'
            $Wizard.DialogResult = 'None'
        } elseif ($Inputs['Contraseña'].Text -cne $Inputs['Repite la contraseña'].Text) {
            Show-Notice 'Las contraseñas no coinciden.'
            $Wizard.DialogResult = 'None'
        }
    })
    $Actions.Controls.AddRange(@($Create, $Cancel))
    $Fields.Controls.Add($Actions)
    $Wizard.Controls.Add($Fields)
    $Wizard.AcceptButton = $Create
    $Wizard.CancelButton = $Cancel
    try {
        if ($Wizard.ShowDialog() -ne 'OK') { exit 2 }
        Submit-FirstAdministrator $Inputs['Usuario'].Text $Inputs['Nombre visible'].Text $Inputs['Contraseña'].Text $Inputs['Repite la contraseña'].Text
    } finally {
        $Inputs['Contraseña'].Clear()
        $Inputs['Repite la contraseña'].Clear()
        $Wizard.Dispose()
    }
    try { Invoke-Management 'SetupComplete' | Out-Null }
    catch { throw 'El administrador se creó. No se pudo iniciar AIDEX. Vuelve al Centro de Control y pulsa Iniciar.' }
    # The parent offers to open the browser, so it runs with the desktop user's rights.
    exit 3
}

if ($Operation) {
    try {
        $Principal = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
        if (!$Principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Windows necesita autorizar esta operación.' }
        if ($Operation -eq 'Setup') { Invoke-SetupWizard }
        else { Invoke-Management $Operation $Port | Out-Null }
        exit 0
    } catch {
        Show-Notice $_.Exception.Message 'Warning'
        exit 1
    }
}

function Get-ControlStatus {
    $Service = Get-Service -Name AIDEX -ErrorAction SilentlyContinue
    $Registry = Get-ItemProperty -LiteralPath 'HKLM:\Software\AIDEX' -ErrorAction SilentlyContinue
    $LocalPort = [int]$Registry.Port
    # The web server can change its endpoint without modifying the registry.
    foreach ($File in @((Join-Path $Data 'connection.json'), (Join-Path $PSScriptRoot 'launcher.json'))) {
        try {
            $Target = Get-Content -LiteralPath $File -Encoding UTF8 -Raw | ConvertFrom-Json
            $Url = [Uri]$Target.public_url
            if ($Target.service_name -eq 'AIDEX' -and $Url.IsAbsoluteUri -and $Url.Scheme -in @('http','https')) { $LocalPort = $Url.Port; break }
        } catch { }
    }
    return [pscustomobject]@{ Service = [string]$Service.Status; Port = $LocalPort; Postgres = [int]$Registry.PostgresPort; SetupComplete = ([int]$Registry.SetupComplete -eq 1) }
}
function Start-DatabaseProbe([int]$Number) {
    $Check = Join-Path $PSScriptRoot 'tools\postgresql\bin\pg_isready.exe'
    if ($Number -lt 1024 -or $Number -gt 65535 -or !(Test-Path -LiteralPath $Check -PathType Leaf)) { return $null }
    $Start = [Diagnostics.ProcessStartInfo]::new($Check)
    $Start.Arguments = "-q -h 127.0.0.1 -p $Number -U aidex_app -d aidex -t 1"
    $Start.UseShellExecute = $false
    $Start.CreateNoWindow = $true
    # Use only the local runtime and endpoint, regardless of the desktop environment.
    foreach ($Key in @($Start.EnvironmentVariables.Keys)) {
        if ($Key -like 'PG*') { $Start.EnvironmentVariables.Remove($Key) }
    }
    $Start.EnvironmentVariables['PGSSLMODE'] = 'disable'
    return [Diagnostics.Process]::Start($Start)
}
function Open-Aidex { Start-Process -FilePath $Launcher | Out-Null }

$Form = New-Form 'Centro de Control de AIDEX' ([Drawing.Size]::new(660, 510))
$Header = [Windows.Forms.Panel]::new()
$Header.Dock = 'Top'
$Header.Height = 104
$Header.BackColor = $Navy
$Title = New-Label 'Centro de Control de AIDEX' $true
$Title.Font = [Drawing.Font]::new('Segoe UI', [single]21, [Drawing.FontStyle]::Bold)
$Title.ForeColor = [Drawing.Color]::White
$Title.Location = [Drawing.Point]::new(24, 16)
$Subtitle = New-Label 'Administración local · Windows'
$Subtitle.ForeColor = [Drawing.ColorTranslator]::FromHtml('#c6ded8')
$Subtitle.Location = [Drawing.Point]::new(27, 65)
$Header.Controls.AddRange(@($Title, $Subtitle))

$Body = New-Column
$Body.AutoSize = $false
$Body.AutoScroll = $true
$Body.Dock = 'Fill'
$Body.Padding = [Windows.Forms.Padding]::new(24, 14, 20, 8)
$Body.Controls.Add((New-Label 'Servicio AIDEX' $true))
$ServiceState = New-Label 'Consultando…'
$ServiceState.AccessibleName = 'Estado del servicio AIDEX'
$Body.Controls.Add($ServiceState)
$ServiceActions = New-Row
$StartButton = New-Button 'Iniciar' $true
$StopButton = New-Button 'Detener'
$RestartButton = New-Button 'Reiniciar'
$ServiceActions.Controls.AddRange(@($StartButton, $StopButton, $RestartButton))
$Body.Controls.Add($ServiceActions)
$Body.Controls.Add((New-Label 'Servidor de base de datos' $true))
$DatabaseState = New-Label 'Consultando…'
$DatabaseState.AccessibleName = 'Estado del servidor de base de datos'
$Body.Controls.Add($DatabaseState)
$PortTitle = New-Label 'Puerto de AIDEX' $true
$PortTitle.Margin = [Windows.Forms.Padding]::new(3, 8, 3, 0)
$Body.Controls.Add($PortTitle)
$PortRow = New-Row
$PortField = [Windows.Forms.NumericUpDown]::new()
$PortField.AccessibleName = 'Puerto de AIDEX'
$PortField.Minimum = 1
$PortField.Maximum = 65535
$PortField.Value = 18090
$PortField.Width = 96
$PortField.Margin = [Windows.Forms.Padding]::new(3, 8, 6, 3)
$PortField.Add_EnabledChanged({
    param($Field, $EventArgs)
    $Field.BackColor = if ($Field.Enabled) { [Drawing.SystemColors]::Window } else { [Drawing.Color]::FromArgb(232, 236, 234) }
    $Field.ForeColor = if ($Field.Enabled) { [Drawing.SystemColors]::WindowText } else { [Drawing.Color]::FromArgb(77, 88, 84) }
})
$SavePort = New-Button 'Cambiar puerto'
$CurrentPort = New-Label 'Consultando puerto…'
$CurrentPort.Margin = [Windows.Forms.Padding]::new(7, 11, 3, 3)
$PortRow.Controls.AddRange(@($PortField, $SavePort, $CurrentPort))
$Body.Controls.Add($PortRow)

$MainActions = New-Row
$MainActions.Margin = [Windows.Forms.Padding]::new(0, 12, 0, 0)
$OpenButton = New-Button 'Abrir AIDEX' $true
$SetupButton = New-Button 'Configuración inicial'
$DiagnosticButton = New-Button 'Diagnóstico'
$MainActions.Controls.AddRange(@($OpenButton, $SetupButton, $DiagnosticButton))
$Body.Controls.Add($MainActions)
$Feedback = New-Label 'Windows pide autorización para los cambios administrativos.'
$Feedback.MaximumSize = [Drawing.Size]::new(570, 0)
$Feedback.AccessibleName = 'Resultado de la operación'
$Body.Controls.Add($Feedback)
$Form.Controls.Add($Body)
$Form.Controls.Add($Header)

$script:Child = $null
$script:Probe = $null
$script:LastPort = 0
$script:NextProbe = [DateTime]::MinValue
$script:Status = $null

function Update-Controls {
    $script:Status = Get-ControlStatus
    $State = $script:Status.Service
    $ServiceState.Text = switch ($State) {
        'Running' { 'En ejecución' }
        'Stopped' { 'Detenido' }
        'StartPending' { 'Iniciando…' }
        'StopPending' { 'Deteniéndose…' }
        '' { 'Servicio no disponible' }
        default { 'Estado temporal. Espera un momento.' }
    }
    $Ready = $null -eq $script:Child
    $StartButton.Enabled = $Ready -and $State -eq 'Stopped'
    $StopButton.Enabled = $Ready -and $State -eq 'Running'
    $RestartButton.Enabled = $StopButton.Enabled
    $OpenButton.Enabled = $Ready -and $State -eq 'Running'
    $SetupButton.Visible = !$script:Status.SetupComplete -and $State -eq 'Stopped'
    $SetupButton.Enabled = $Ready
    $PortField.Enabled = $Ready -and $State -in @('Stopped','Running') -and $script:Status.Port -gt 0
    $SavePort.Enabled = $PortField.Enabled
    if ($script:Status.Port -gt 0) {
        if ($script:LastPort -ne $script:Status.Port) { $PortField.Value = $script:Status.Port; $script:LastPort = $script:Status.Port }
        $CurrentPort.Text = "Puerto actual: $($script:Status.Port)"
    } else { $CurrentPort.Text = 'Puerto no disponible' }
}
function Start-Operation([string]$Action, [int]$SelectedPort = 0) {
    if ($script:Child) { return }
    $Arguments = @('--control-operation', $Action)
    if ($Action -eq 'Port') { $Arguments += [string]$SelectedPort }
    try {
        $script:Child = Start-Process -FilePath $Launcher -Verb RunAs -ArgumentList $Arguments -WindowStyle Hidden -PassThru
        $Feedback.Text = 'Operación en curso. Espera un momento…'
        Update-Controls
    } catch {
        $Feedback.Text = 'Operación cancelada o no autorizada por Windows.'
    }
}

$StartButton.Add_Click({ Start-Operation 'Start' })
$StopButton.Add_Click({ Start-Operation 'Stop' })
$RestartButton.Add_Click({ Start-Operation 'Restart' })
$SetupButton.Add_Click({ Start-Operation 'Setup' })
$OpenButton.Add_Click({ Open-Aidex })
$SavePort.Add_Click({
    if ($PortField.Value -eq $script:Status.Port) { $Feedback.Text = 'El puerto ya está configurado.'; return }
    $Confirm = [Windows.Forms.MessageBox]::Show($Form, 'AIDEX puede interrumpirse unos segundos. Se actualizará la regla de firewall de AIDEX. ¿Cambiar el puerto?', 'Cambiar puerto de AIDEX', [Windows.Forms.MessageBoxButtons]::YesNo, [Windows.Forms.MessageBoxIcon]::Question)
    if ($Confirm -eq 'Yes') { Start-Operation 'Port' ([int]$PortField.Value) }
})
$DiagnosticButton.Add_Click({
    $Lines = @("Servicio: $($ServiceState.Text)", "Servidor de base de datos: $($DatabaseState.Text)", "Puerto de AIDEX: $($script:Status.Port)")
    try {
        $Event = Get-WinEvent -FilterHashtable @{ LogName='Application'; ProviderName='AIDEX'; Level=2 } -MaxEvents 1
        $Cause = switch -Regex ($Event.Message) {
            'PostgreSQL|pg_ctl|database' { 'AIDEX no pudo acceder a su base de datos.'; break }
            'puerto|listen|bind' { 'AIDEX no pudo abrir el puerto configurado.'; break }
            'bootstrap' { 'Falta completar la configuración inicial.'; break }
            'access denied|acceso denegado|permission' { 'Windows denegó acceso a un recurso que usa AIDEX.'; break }
            default { 'Windows registró un error de inicio de AIDEX.' }
        }
        $Lines += '', ('Último error registrado: ' + $Event.TimeCreated.ToString('g')), $Cause
    } catch { $Lines += '', 'Windows no mostró errores recientes de AIDEX.' }
    Show-Notice ($Lines -join [Environment]::NewLine)
})

$Timer = [Windows.Forms.Timer]::new()
$Timer.Interval = 500
$Timer.Add_Tick({
    if ($script:Child -and $script:Child.HasExited) {
        $Code = $script:Child.ExitCode
        $script:Child.Dispose()
        $script:Child = $null
        $Feedback.Text = switch ($Code) {
            0 { 'Operación completada.' }
            2 { 'Configuración inicial cancelada.' }
            3 { 'Administrador creado. Ya puedes iniciar sesión.' }
            default { 'La operación no se completó. Consulta Diagnóstico.' }
        }
        $script:NextProbe = [DateTime]::MinValue
        Update-Controls
        if ($Code -eq 3 -and [Windows.Forms.MessageBox]::Show($Form, 'Se creó el administrador inicial. ¿Abrir AIDEX para iniciar sesión?', 'AIDEX', [Windows.Forms.MessageBoxButtons]::YesNo, [Windows.Forms.MessageBoxIcon]::Information) -eq 'Yes') { Open-Aidex }
    }
    if ($script:Probe -and $script:Probe.HasExited) {
        $DatabaseState.Text = switch ($script:Probe.ExitCode) {
            0 { 'En ejecución' }
            1 { 'Iniciando o deteniéndose' }
            2 { if ($script:Status.Service -eq 'Stopped') { 'Detenido' } else { 'No disponible' } }
            default { 'No disponible' }
        }
        $script:Probe.Dispose()
        $script:Probe = $null
    }
    if ([DateTime]::UtcNow -ge $script:NextProbe) {
        Update-Controls
        if (!$script:Probe) {
            try { $script:Probe = Start-DatabaseProbe $script:Status.Postgres }
            catch { $script:Probe = $null }
            if (!$script:Probe) { $DatabaseState.Text = 'No disponible' }
        }
        $script:NextProbe = [DateTime]::UtcNow.AddSeconds(4)
    }
})
try {
    Update-Controls
    $Timer.Start()
    $Form.Add_FormClosing({
        if ($script:Child -and !$script:Child.HasExited) {
            $Feedback.Text = 'Espera a que termine la operación antes de cerrar esta ventana.'
            $_.Cancel = $true
        }
    })
    [Windows.Forms.Application]::Run($Form)
} finally {
    $Timer.Dispose()
    if ($script:Probe) { $script:Probe.Dispose() }
    if ($script:Child) { $script:Child.Dispose() }
    $Form.Dispose()
}
