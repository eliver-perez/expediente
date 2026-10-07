Unicode true
!include "MUI2.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"
!include "WinVer.nsh"
!include "nsDialogs.nsh"
Var EraseData
Var EraseCheckbox
Name "AIBID Pruebas ${VERSION}"
!define MUI_ICON "${STAGE}/aibid.ico"
!define MUI_UNICON "${STAGE}/aibid.ico"
OutFile "${OUTPUT}"
InstallDir "$PROGRAMFILES64\AIBID-Test"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "AIBID — Tu biblioteca digital, ordenada y al alcance."
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TEXT "Instalación de pruebas independiente. Usa http://127.0.0.1:18090.$\r$\n$\r$\nDespués abre Configurar AIBID Pruebas para crear tu administrador (mínimo 6 caracteres) y activa tu licencia desde AIBID.$\r$\n$\r$\nPara previsualizar Word y Excel instala LibreOffice en este equipo y revisa Ajustes > Vistas previas y caché. La extracción y búsqueda funcionan sin LibreOffice.$\r$\n$\r$\nAl desinstalar, los datos internos se conservan por defecto."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TEXT "Primera vez: abre Configurar AIBID Pruebas desde Inicio para crear el administrador e iniciar el servicio.$\r$\n$\r$\nDespués usa AIBID Pruebas en el Escritorio o Abrir AIBID en Inicio: abre el navegador sin pedir permisos de administrador."
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
UninstPage custom un.DataOptions un.ConfirmDataOptions
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

Function un.DataOptions
    StrCpy $EraseData ""
    nsDialogs::Create 1018
    Pop $0
    ${NSD_CreateCheckbox} 0 10u 100% 20u "Eliminar los datos de AIBID al desinstalar"
    Pop $EraseCheckbox
    ${NSD_Uncheck} $EraseCheckbox
    ${NSD_CreateLabel} 0 40u 100% 65u "Se eliminarán la base de datos, los índices, el contenido extraído, los usuarios y la configuración. Tus documentos físicos, incluidas las bibliotecas administradas y cargas, permanecerán intactos en sus rutas.$\r$\n$\r$\nPara usar la licencia en una instalación nueva, desactívala primero desde AIBID. Esta limpieza no libera la activación en el servidor."
    Pop $0
    nsDialogs::Show
FunctionEnd

Function un.ConfirmDataOptions
    ${NSD_GetState} $EraseCheckbox $0
    StrCpy $EraseData ""
    ${If} $0 == ${BST_CHECKED}
        MessageBox MB_YESNO|MB_ICONEXCLAMATION|MB_DEFBUTTON2 "¿Eliminar los datos internos de AIBID? Se perderán usuarios, bibliotecas registradas, índices, historial y configuración. Los documentos físicos permanecerán intactos.$\r$\n$\r$\nUna reinstalación comenzará desde cero y requerirá activar la licencia." IDYES confirmed
        Abort
        confirmed:
        StrCpy $EraseData "-EraseInternalData"
    ${EndIf}
FunctionEnd

Section "Instalar"
    InitPluginsDir
    SetOutPath "$PLUGINSDIR"
    File "${STAGE}/manage.ps1"
    File /oname=gestor-documental-repair.exe "${STAGE}/gestor-documental.exe"
    nsExec::ExecToLog '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$PLUGINSDIR\manage.ps1" -Action Preflight -InstallRoot "$INSTDIR" -PreflightBinary "$PLUGINSDIR\gestor-documental-repair.exe"'
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
    Delete "$SMPROGRAMS\AIBID Pruebas\Abrir AIBID.url"
    CreateShortCut "$SMPROGRAMS\AIBID Pruebas\Abrir AIBID.lnk" "$INSTDIR\AIBID.exe"
    CreateShortCut "$DESKTOP\AIBID Pruebas.lnk" "$INSTDIR\AIBID.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "DisplayName" "AIBID Pruebas"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "DisplayVersion" "${VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "UninstallString" '$\"$INSTDIR\Desinstalar.exe$\"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "DisplayIcon" "$INSTDIR\AIBID.exe,0"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest" "NoRepair" 1
SectionEnd

Section "Uninstall"
    SetRegView 64
    SetShellVarContext all
    ${DisableX64FSRedirection}
    nsExec::ExecToLog '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\manage.ps1" -Action Remove -InstallRoot "$INSTDIR" $EraseData'
    Pop $0
    ${If} $0 != 0
        Abort "No se pudo completar la preparación de la desinstalación. Consulta el detalle; se conserva el programa."
    ${EndIf}
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIBIDTest"
    Delete "$SMPROGRAMS\AIBID Pruebas\Configurar AIBID Pruebas.lnk"
    Delete "$SMPROGRAMS\AIBID Pruebas\Desinstalar.lnk"
    Delete "$SMPROGRAMS\AIBID Pruebas\Abrir AIBID.lnk"
    Delete "$SMPROGRAMS\AIBID Pruebas\Abrir AIBID.url"
    RMDir "$SMPROGRAMS\AIBID Pruebas"
    Delete "$DESKTOP\AIBID Pruebas.lnk"
    ; Packaged files were checked by hash. Keep unknown and modified files.
    RMDir "$INSTDIR\tools"
    RMDir "$INSTDIR\docs"
    Delete "$INSTDIR\launcher.json"
    Delete "$INSTDIR\Desinstalar.exe"
    RMDir "$INSTDIR"
SectionEnd
