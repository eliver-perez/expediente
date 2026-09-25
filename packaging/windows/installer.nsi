Unicode true
!include "MUI2.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"
!include "WinVer.nsh"
Name "AIBID Pruebas ${VERSION}"
OutFile "${OUTPUT}"
InstallDir "$PROGRAMFILES64\AIBID-Test"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "AIBID — Tu biblioteca digital, ordenada y al alcance."
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TEXT "Instalación de pruebas independiente. Usa http://127.0.0.1:18090 y conserva los datos al desinstalar.$\r$\n$\r$\nDespués abre Configurar AIBID Pruebas para crear tu administrador (mínimo 6 caracteres). No necesita servidor de licencias.$\r$\n$\r$\nEsta compilación no es una versión comercial."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TEXT "Abre Configurar AIBID Pruebas desde Inicio para crear el administrador e iniciar el servicio. Herramientas PDF/OCR e idiomas español e inglés incluidos."
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "Spanish"

Function .onInit
    ${IfNot} ${RunningX64}
        MessageBox MB_ICONSTOP "Se requiere Windows de 64 bits."
        Abort
    ${EndIf}
    ${IfNot} ${AtLeastWin10}
        MessageBox MB_ICONSTOP "Se requiere Windows 10 o posterior de 64 bits."
        Abort
    ${EndIf}
    SetRegView 64
    SetShellVarContext all
    ${DisableX64FSRedirection}
FunctionEnd

Section "Instalar"
    InitPluginsDir
    SetOutPath "$PLUGINSDIR"
    File "${STAGE}/manage.ps1"
    nsExec::ExecToLog '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$PLUGINSDIR\manage.ps1" -Action Preflight -InstallRoot "$INSTDIR"'
    Pop $0
    ${If} $0 != 0
        Abort "No se pudo preparar la instalación; consulta el detalle."
    ${EndIf}
    SetOutPath "$INSTDIR"
    File /r "${STAGE}/*"
    nsExec::ExecToLog '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\manage.ps1" -Action Configure -InstallRoot "$INSTDIR"'
    Pop $0
    ${If} $0 != 0
        Abort "La configuración falló. No se inició el servicio. Consulta el detalle."
    ${EndIf}
    WriteUninstaller "$INSTDIR\Desinstalar.exe"
    CreateDirectory "$SMPROGRAMS\AIBID Pruebas"
    CreateShortCut "$SMPROGRAMS\AIBID Pruebas\Configurar AIBID Pruebas.lnk" "$INSTDIR\admin.cmd"
    CreateShortCut "$SMPROGRAMS\AIBID Pruebas\Desinstalar.lnk" "$INSTDIR\Desinstalar.exe"
    WriteINIStr "$SMPROGRAMS\AIBID Pruebas\Abrir AIBID.url" "InternetShortcut" "URL" "http://127.0.0.1:18090"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "DisplayName" "AIBID Pruebas"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "DisplayVersion" "${VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "UninstallString" '$\"$INSTDIR\Desinstalar.exe$\"'
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "NoRepair" 1
SectionEnd

Section "Uninstall"
    SetRegView 64
    SetShellVarContext all
    ${DisableX64FSRedirection}
    nsExec::ExecToLog '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\manage.ps1" -Action Remove -InstallRoot "$INSTDIR"'
    Pop $0
    ${If} $0 != 0
        Abort "No se pudo detener el servicio. Se conserva el programa."
    ${EndIf}
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest"
    RMDir /r "$SMPROGRAMS\AIBID Pruebas"
    ; Only code-owned subdirectories; ProgramData and library roots are never removed.
    RMDir /r "$INSTDIR\tools"
    RMDir /r "$INSTDIR\docs"
    Delete "$INSTDIR\gestor-documental.exe"
    Delete "$INSTDIR\manage.ps1"
    Delete "$INSTDIR\admin.cmd"
    Delete "$INSTDIR\Desinstalar.exe"
    RMDir "$INSTDIR"
SectionEnd
