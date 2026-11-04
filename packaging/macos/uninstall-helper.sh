#!/bin/sh
# El paquete instala este auxiliar como root:wheel dentro de la app firmada.
# Solo acepta una de dos operaciones explícitas; nunca elimina árboles de datos.
set -eu
umask 077

CODE='/Library/Application Support/AIDEX'
APP='/Applications/AIDEX/Desinstalar AIDEX.app'
APPS='/Applications/AIDEX'
PLIST='/Library/LaunchDaemons/app.aidex.plist'
RECEIPT='app.aidex'
ACCOUNT='_aidex'
BIN="$CODE/bin/aidex-server"
CONFIG="$CODE/data/config.json"
MANIFEST="$CODE/package-files.tsv"

[ "$(/usr/bin/id -u)" -eq 0 ] || { echo 'Se requiere autorización de administrador.' >&2; exit 1; }
[ "$#" -eq 1 ] || { echo 'Operación inválida.' >&2; exit 1; }
case "$1" in conservar|eliminar) ACTION=$1;; *) echo 'Operación inválida.' >&2; exit 1;; esac

no_links() {
    current=$1
    while [ "$current" != / ]; do
        [ ! -L "$current" ] || { echo "Enlace no permitido: $current" >&2; return 1; }
        current=$(/usr/bin/dirname "$current")
    done
}

allowed_file() {
    case "$1" in
        "$CODE/bin/"*|"$CODE/docs/"*|"$CODE/profile.sh"|"$MANIFEST"|"$APPS/Configurar AIDEX.command"|"$APPS/Configurar puerto.command"|"$APPS/Desinstalar AIDEX.command"|"$APPS/Abrir AIDEX.webloc"|"$APP/Contents/"*|"$APPS/Autorizar carpeta AIDEX.app/Contents/"*|"$PLIST") ;;
        *) echo "Archivo ajeno al paquete: $1" >&2; return 1;;
    esac
    case "$1" in */../*|*/./*|*/..|*/.|*'//'*) echo 'Ruta insegura en inventario.' >&2; return 1;; esac
    no_links "$1"
}

for path in "$CODE" "$CONFIG" "$MANIFEST" "$PLIST" "$APPS" "$APP"; do no_links "$path"; done

# El inventario con SHA-256 es el principal. En una instalación parcial puede
# faltar; el recibo de pkgutil es la alternativa para archivos de este paquete.
LIST=$(/usr/bin/mktemp -t aidex-uninstall.XXXXXX)
trap '/bin/rm -f "$LIST"' EXIT HUP INT TERM
if [ -f "$MANIFEST" ]; then
    /bin/cat "$MANIFEST" > "$LIST"
elif /usr/sbin/pkgutil --pkg-info "$RECEIPT" >/dev/null 2>&1; then
    /usr/sbin/pkgutil --files "$RECEIPT" | while IFS= read -r relative; do
        [ -f "/$relative" ] || continue
        /usr/bin/printf -- '-\t/%s\n' "$relative"
    done > "$LIST"
else
    echo 'Falta el inventario y el recibo del instalador; se conservaron los archivos y datos para una revisión manual.' >&2
    exit 1
fi

TAB=$(/usr/bin/printf '\t')
while IFS="$TAB" read -r HASH FILE; do
    [ -n "$HASH" ] && [ -n "$FILE" ] || { echo 'Inventario incompleto.' >&2; exit 1; }
    allowed_file "$FILE"
    if [ "$HASH" != '-' ]; then
        [ "${#HASH}" -eq 64 ] && /usr/bin/printf '%s' "$HASH" | /usr/bin/grep -Eq '^[0-9a-f]{64}$' || { echo 'Huella inválida en inventario.' >&2; exit 1; }
    fi
done < "$LIST"

if DETAILS=$(/bin/launchctl print "system/$RECEIPT" 2>/dev/null); then
    OLD_PID=$(/usr/bin/printf '%s\n' "$DETAILS" | /usr/bin/awk '/^[[:space:]]*pid = [0-9]+/ {print $3; exit}')
    /bin/launchctl bootout "system/$RECEIPT" || { echo 'No se pudo detener el servicio AIDEX.' >&2; exit 1; }
    ATTEMPT=0
    while /bin/launchctl print "system/$RECEIPT" >/dev/null 2>&1 || { [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; }; do
        ATTEMPT=$((ATTEMPT + 1))
        [ "$ATTEMPT" -lt 60 ] || { echo 'El servicio no terminó en 60 segundos. Se conservaron los datos y las herramientas.' >&2; exit 1; }
        /bin/sleep 1
    done
fi

# Nunca borrar metadatos si el motor no puede verificar acceso exclusivo. Si
# falta la configuración, una instalación parcial puede conservar otros datos.
ERASED=no
if [ "$ACTION" = eliminar ]; then
    [ -f "$BIN" ] || { echo 'Falta el ejecutable para verificar la eliminación completa. Los datos y herramientas se conservan para recuperación.' >&2; exit 1; }
    # La limpieza completa mueve los originales internos fuera de data y opera
    # como administrador. El modo anterior borraba metadatos, pero dejaba logs
    # y service.lock bloqueando la siguiente instalación.
    "$BIN" erase-macos-installation --config "$CONFIG" --confirm-erase-internal-data
    [ ! -e "$CODE/data" ] && [ ! -L "$CODE/data" ] || { echo 'La eliminación dejó residuos en data; no se completó la desinstalación.' >&2; exit 1; }
    ERASED=yes
elif [ -f "$CONFIG" ] && [ -f "$BIN" ]; then
    /usr/bin/sudo -u "$ACCOUNT" "$BIN" check-state --config "$CONFIG" >/dev/null
fi

# Los originales archivados deben poder leerse sin depender de _aidex. Esta
# operación privilegiada solo admite la ubicación fija de documentos conservados.
if [ "$ERASED" = yes ] && [ -d "$CODE/Documentos conservados" ]; then
    [ -x "$CODE/bin/aidex-folder-helper" ] || { echo 'Falta el auxiliar para conservar el acceso desde Finder. Se mantienen las herramientas para recuperación.' >&2; exit 1; }
    /usr/bin/sudo "$CODE/bin/aidex-folder-helper" preserve-archive
fi

SKIPPED=0
while IFS="$TAB" read -r HASH FILE; do
    [ -f "$FILE" ] || continue
    if [ "$HASH" = '-' ] || [ "$(/usr/bin/shasum -a 256 "$FILE" | /usr/bin/cut -d ' ' -f 1)" = "$HASH" ]; then
        /bin/rm -f "$FILE"
    else
        SKIPPED=$((SKIPPED + 1))
    fi
done < "$LIST"

# Solo se eliminan directorios vacíos de ubicaciones fijas del paquete.
/bin/rm -f "$MANIFEST"
for tree in "$APPS/Autorizar carpeta AIDEX.app" "$APP" "$CODE/docs"; do
    [ ! -d "$tree" ] || /usr/bin/find "$tree" -depth -type d -exec /bin/rmdir '{}' \; 2>/dev/null || true
done
for directory in "$APPS" "$CODE/bin" "$CODE"; do
    [ ! -d "$directory" ] || /bin/rmdir "$directory" 2>/dev/null || true
done
if /usr/sbin/pkgutil --pkg-info "$RECEIPT" >/dev/null 2>&1; then
    /usr/sbin/pkgutil --forget "$RECEIPT" >/dev/null || { echo 'Los archivos se retiraron, pero no se pudo borrar el recibo de pkgutil.' >&2; exit 1; }
fi

if [ "$SKIPPED" -gt 0 ]; then
    echo 'Desinstalación parcial: los archivos modificados se conservaron para revisión manual.'
elif [ "$ACTION" = conservar ]; then
    echo "AIDEX desinstalado. Datos y configuración conservados en $CODE/data."
elif [ "$ERASED" != yes ]; then
    echo 'AIDEX desinstalado. No se eliminaron datos internos porque no había una configuración verificable; revisa la instalación parcial.'
else
    echo 'AIDEX desinstalado. Datos internos reconocidos eliminados.'
fi
echo 'Los documentos originales, las cargas y los archivos no reconocidos permanecen intactos.'
echo 'La activación del servidor de licencias no se liberó automáticamente.'
if [ "$SKIPPED" -gt 0 ]; then echo "$SKIPPED archivo(s) modificados se conservaron para revisión manual."; fi
