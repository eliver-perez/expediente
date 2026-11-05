# AIDEX 1.0.0: instalador Windows desde este macOS

Instrucciones verificadas contra `scripts/build_installers.py`, `Makefile`,
`go.mod`, `web/package.json` y la distribución local de NSIS. No generan PKG ni DEB.

## Archivo para editar el texto

`packaging/windows/installer.nsi`:

- Línea 31, `MUI_WELCOMEPAGE_TITLE`: encabezado en negrita.
- Línea 32, `MUI_WELCOMEPAGE_TITLE_3LINES`: reserva espacio para tres líneas.
- Línea 33, `MUI_WELCOMEPAGE_TEXT`: tres párrafos del cuerpo.
- Línea 37, `MUI_FINISHPAGE_TEXT`: pantalla final, donde también se retiró “provisional”.

Conservar UTF-8. En NSIS, `$\r$\n$\r$\n` separa párrafos. Las definiciones de
bienvenida deben estar antes de `!insertmacro MUI_PAGE_WELCOME`.
Estos textos informativos no modifican el contrato ni su aceptación en AIDEX.

## Herramientas disponibles

Se conservaron copias portátiles completas dentro del proyecto, sin instalar
herramientas globales ni modificar macOS:

| Herramienta | Versión comprobada | Ubicación |
| --- | --- | --- |
| Go | 1.27.1, la requerida por `go.mod` | `go` en PATH |
| Node.js | v22.15.0; el proyecto exige >=22.12.0 | `node` y `npm` en PATH |
| Python | 3.9.6 | `python3` en PATH; builder usa biblioteca estándar |
| NSIS | 3.13 | `build/windows-tools/nsis/makensis` |
| PowerShell | 7.6.9 | `build/windows-tools/powershell/pwsh` |
| 7-Zip | 26.05 | `build/windows-tools/7zip/7zz`, para extraer un instalador anterior |

Guardar las carpetas completas de NSIS y PowerShell: el ejecutable solo no basta.
`build/` está ignorado por Git; conservarlo o respaldarlo si se limpia el proyecto.
Estas rutas persistentes evitan depender de las herramientas originales de `/private/tmp`.

También quedó preparado `build/windows-installer-payload`: 1114 archivos comprobados
contra el manifiesto del instalador Windows del 4 de noviembre de 2026, más su
`package-files.json`. Solo contiene archivos reales del programa; se excluyeron
los temporales `$PLUGINSDIR` de NSIS. Incluye binarios, scripts, icono, documentación,
OCR, DLL, idiomas y licencias. No usar como STAGE toda una extracción sin depurar.

## Variables comunes

Ejecutar en Terminal, desde la raíz del proyecto:

```sh
cd /Applications/XAMPP/xamppfiles/htdocs/expediente

AIDEX_NSIS="$PWD/build/windows-tools/nsis/makensis"
AIDEX_PWSH="$PWD/build/windows-tools/powershell/pwsh"
AIDEX_PAYLOAD="$PWD/build/windows-installer-payload"
export NSISDIR="$PWD/build/windows-tools/nsis"
```

Para verificar herramientas y archivos:

```sh
go version
node --version
python3 --version
"$AIDEX_NSIS" -VERSION
"$AIDEX_PWSH" -NoProfile -NonInteractive -Command '$PSVersionTable.PSVersion.ToString()'
test -f "$AIDEX_PAYLOAD/package-files.json"
```

## Opción rápida: cambiar únicamente textos NSIS

Tras editar `installer.nsi`, ejecutar:

```sh
mkdir -p "$PWD/dist/aidex-1.0.0-windows-bienvenida-20261105"

"$AIDEX_NSIS" \
  -DVERSION=1.0.0 \
  "-DSTAGE=$AIDEX_PAYLOAD" \
  "-DOUTPUT=$PWD/dist/aidex-1.0.0-windows-bienvenida-20261105/AIDEX-1.0.0-Windows-amd64.exe" \
  packaging/windows/installer.nsi

shasum -a 256 \
  dist/aidex-1.0.0-windows-bienvenida-20261105/AIDEX-1.0.0-Windows-amd64.exe
```

Esta es la invocación NSIS que utiliza el builder: parámetros `VERSION`, `STAGE` y
`OUTPUT`, con los nombres reales del proyecto. Solo recompone el instalador;
no ejecuta Go, Node, PowerShell ni reconstruye OCR. Para este cambio informativo
es suficiente. No hace falta instalar Go/Node/Python para repetir esta opción
si ya se dispone de NSIS y del payload preparado.

**El payload reutilizado conserva los binarios y scripts del instalador anterior.**
Si cambian Go, interfaz web, migraciones, scripts o recursos incluidos, utilizar
la opción completa. Después conviene preparar de nuevo el payload desde ese
instalador actualizado para futuros cambios de texto.

La llamada directa a NSIS no actualiza el archivo `SHA256SUMS` del builder. Para
actualizar solo la suma del EXE, sin sobrescribir sumas de otros archivos:

```sh
(
  cd dist/aidex-1.0.0-windows-bienvenida-20261105
  shasum -a 256 AIDEX-1.0.0-Windows-amd64.exe > AIDEX-1.0.0-Windows-amd64.exe.sha256
)
```

## Opción completa: incorporar cambios de código, web o scripts

Con las variables comunes definidas:

```sh
# Restaurar dependencias si aún no están disponibles.
go mod download
npm --prefix web ci

# Actualizar los assets que Go incorpora al servidor.
npm --prefix web run build

python3 scripts/build_installers.py \
  --target windows \
  --output dist/aidex-1.0.0-windows \
  --windows-runtime-payload "$AIDEX_PAYLOAD" \
  --makensis "$AIDEX_NSIS" \
  --powershell "$AIDEX_PWSH"
```

Resultado:
`dist/aidex-1.0.0-windows/AIDEX-1.0.0-Windows-amd64.exe` y `SHA256SUMS`.

El script recompila `aidex-server.exe` y `AIDEX.exe` para Windows amd64 con
`CGO_ENABLED=0`; no requiere MinGW ni Visual Studio. Valida los scripts con
PowerShell y comprueba el runtime OCR contra el manifiesto y el lock del proyecto.
Reutiliza solo OCR/documentación de terceros del payload anterior, no sus binarios Go.
No hacen falta Micromamba ni descargar nuevamente OCR utilizando esta opción.

El builder **no compila el frontend automáticamente**, pero sí requiere los
`node_modules` para copiar licencias incluso si los assets ya estaban construidos.
Si dependencias y assets están vigentes, se pueden omitir los tres comandos de
restauración/build anteriores y ejecutar directamente `python3 ... --target windows`.
No ejecutar el build frontend en paralelo con el empaquetado: Go incorpora esos archivos.

No existe un target `make installer-windows` en este Makefile. `make build`
reconstruye el frontend y un servidor del sistema anfitrión, que no se necesita
para este instalador. Para Windows basta el comando Python anterior; no usar
`--target all` para este trabajo.

## Recuperar un payload si se elimina el preparado

Extraer un instalador Windows conocido con 7-Zip y copiar únicamente los archivos
indicados en `package-files.json`, verificando sus hashes. El directorio final debe
contener ese manifiesto, `AIDEX.exe`, `aidex-server.exe`, `aidex.ico`, scripts,
`tools/` y `docs/`, sin `$PLUGINSDIR` ni otros temporales de extracción.

La opción completa también admite `--windows-tools` en lugar de
`--windows-runtime-payload` cuando se dispone de un prefijo conda-forge win-64
restaurado desde `packaging/windows/ocr-explicit.txt`. Ese prefijo no es necesario
para los comandos concretos de este documento: el runtime verificado ya está preparado.
