# -*- coding: utf-8 -*-
Unicode true
!include "MUI2.nsh"
!include "x64.nsh"
!include "LogicLib.nsh"
!include "WinVer.nsh"
!include "nsDialogs.nsh"
!include "FileFunc.nsh"
Var LogPath
VIProductVersion "1.0.0.0"
VIAddVersionKey /LANG=1034 "ProductName" "AIDEX"
VIAddVersionKey /LANG=1034 "FileDescription" "Instalador de AIDEX"
VIAddVersionKey /LANG=1034 "FileVersion" "1.0.0"
VIAddVersionKey /LANG=1034 "ProductVersion" "1.0.0"
VIAddVersionKey /LANG=1034 "LegalCopyright" "AIDEX"
Var EraseData
Var EraseCheckbox
Name "AIDEX"
!define MUI_ICON "${STAGE}/aidex.ico"
!define MUI_UNICON "${STAGE}/aidex.ico"
OutFile "${OUTPUT}"
InstallDir "$PROGRAMFILES64\AIDEX"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "AIDEX — Tu biblioteca digital, ordenada y al alcance."
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TEXT "Instalación de AIDEX. Usa http://127.0.0.1:18090.$\r$\n$\r$\nDespués abre Configurar AIDEX para crear tu administrador (mínimo 6 caracteres) y activa tu licencia desde AIDEX.$\r$\n$\r$\nPara previsualizar Word y Excel instala LibreOffice en este equipo y revisa Ajustes > Vistas previas y caché. La extracción y búsqueda funcionan sin LibreOffice.$\r$\n$\r$\nAl desinstalar, los datos internos se conservan por defecto."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TEXT "Primera vez: abre Configurar AIDEX desde Inicio para crear el administrador e iniciar el servicio.$\r$\n$\r$\nDespués usa AIDEX en el Escritorio o Abrir AIDEX en Inicio: abre el navegador sin pedir permisos de administrador."
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
    ReadRegStr $0 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "InstallLocation"
    ${If} $0 != ""
        StrCpy $INSTDIR $0
    ${EndIf}
FunctionEnd

!macro ReadAidexLog Prefix
Function ${Prefix}ReadAidexLog
    ClearErrors
    FileOpen $1 $LogPath r
    IfErrors done
    read:
    FileReadUTF16LE $1 $2
    IfErrors close
    DetailPrint $2
    Goto read
    close:
    FileClose $1
    done:
    DetailPrint "Registro de instalación: $LogPath"
FunctionEnd
!macroend
!insertmacro ReadAidexLog ""
!insertmacro ReadAidexLog "un."

Function un.DataOptions
    StrCpy $EraseData ""
    nsDialogs::Create 1018
    Pop $0
    ${NSD_CreateCheckbox} 0 10u 100% 20u "Eliminar los datos de AIDEX al desinstalar"
    Pop $EraseCheckbox
    ${NSD_Uncheck} $EraseCheckbox
    ${NSD_CreateLabel} 0 40u 100% 65u "Se eliminarán la base de datos, los índices, el contenido extraído, los usuarios y la configuración. Tus documentos físicos, incluidas las bibliotecas administradas y cargas, permanecerán intactos en sus rutas.$\r$\n$\r$\nPara usar la licencia en una instalación nueva, desactívala primero desde AIDEX. Esta limpieza no libera la activación en el servidor."
    Pop $0
    nsDialogs::Show
FunctionEnd

Function un.ConfirmDataOptions
    ${NSD_GetState} $EraseCheckbox $0
    StrCpy $EraseData ""
    ${If} $0 == ${BST_CHECKED}
        MessageBox MB_YESNO|MB_ICONEXCLAMATION|MB_DEFBUTTON2 "¿Eliminar los datos internos de AIDEX? Se perderán usuarios, bibliotecas registradas, índices, historial y configuración. Los documentos físicos permanecerán intactos.$\r$\n$\r$\nUna reinstalación comenzará desde cero y requerirá activar la licencia." IDYES confirmed
        Abort
        confirmed:
        StrCpy $EraseData "-EraseInternalData"
    ${EndIf}
FunctionEnd

Section "Instalar"
    InitPluginsDir
    SetOutPath "$PLUGINSDIR"
    File "${STAGE}/manage.ps1"
    File "${STAGE}/installer-action.ps1"
    File /oname=aidex-repair.exe "${STAGE}/aidex-server.exe"
    GetTempFileName $LogPath
    ExecWait '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$PLUGINSDIR\installer-action.ps1" -ManageScript "$PLUGINSDIR\manage.ps1" -LogPath "$LogPath" -Action Preflight -InstallRoot "$INSTDIR" -PreflightBinary "$PLUGINSDIR\aidex-repair.exe"' $0
    Call ReadAidexLog
    ${If} $0 != 0
        Abort "No se pudo preparar la instalación; consulta el detalle."
    ${EndIf}
    SetOutPath "$INSTDIR"
    File /r "${STAGE}/*"
    GetTempFileName $LogPath
    ExecWait '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\installer-action.ps1" -ManageScript "$INSTDIR\manage.ps1" -LogPath "$LogPath" -Action Configure -InstallRoot "$INSTDIR"' $0
    Call ReadAidexLog
    ${If} $0 != 0
        Abort "La configuración falló. No se inició el servicio. Consulta el detalle."
    ${EndIf}
    WriteUninstaller "$INSTDIR\Desinstalar.exe"
    CreateDirectory "$SMPROGRAMS\AIDEX"
    CreateShortCut "$SMPROGRAMS\AIDEX\Configurar AIDEX.lnk" "$INSTDIR\admin.cmd"
    CreateShortCut "$SMPROGRAMS\AIDEX\Desinstalar.lnk" "$INSTDIR\Desinstalar.exe"
    Delete "$SMPROGRAMS\AIDEX\Abrir AIDEX.url"
    CreateShortCut "$SMPROGRAMS\AIDEX\Abrir AIDEX.lnk" "$INSTDIR\AIDEX.exe"
    CreateShortCut "$DESKTOP\AIDEX.lnk" "$INSTDIR\AIDEX.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "DisplayName" "AIDEX"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "DisplayVersion" "${VERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "UninstallString" '$\"$INSTDIR\Desinstalar.exe$\"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "DisplayIcon" "$INSTDIR\AIDEX.exe,0"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX" "NoRepair" 1
SectionEnd

Section "Uninstall"
    SetRegView 64
    SetShellVarContext all
    ${DisableX64FSRedirection}
    GetTempFileName $LogPath
    ExecWait '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\installer-action.ps1" -ManageScript "$INSTDIR\manage.ps1" -LogPath "$LogPath" -Action Remove -InstallRoot "$INSTDIR" $EraseData' $0
    Call un.ReadAidexLog
    ${If} $0 != 0
        Abort "No se pudo completar la preparación de la desinstalación. Consulta el detalle; se conserva el programa."
    ${EndIf}
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AIDEX"
    Delete "$SMPROGRAMS\AIDEX\Configurar AIDEX.lnk"
    Delete "$SMPROGRAMS\AIDEX\Desinstalar.lnk"
    Delete "$SMPROGRAMS\AIDEX\Abrir AIDEX.lnk"
    Delete "$SMPROGRAMS\AIDEX\Abrir AIDEX.url"
    RMDir "$SMPROGRAMS\AIDEX"
    Delete "$DESKTOP\AIDEX.lnk"
    ; Packaged files were checked by hash. Keep unknown and modified files.
    RMDir "$INSTDIR\tools"
    RMDir "$INSTDIR\docs"
    Delete "$INSTDIR\launcher.json"
    Delete "$INSTDIR\Desinstalar.exe"
    RMDir "$INSTDIR"
SectionEnd
