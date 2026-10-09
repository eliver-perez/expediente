#!/bin/sh
set -eu
if [ "$(/usr/bin/id -u)" -ne 0 ]; then exec /usr/bin/sudo /bin/sh "$0"; fi
BASE='/Library/Application Support/AIBID-Test'
ERASE=no
echo 'Por defecto se conservarán los datos de AIBID.'
printf '¿Eliminar también base de datos, índices, texto extraído y configuración? [s/N]: '
read -r ANSWER || ANSWER=no
case "$ANSWER" in
    s|S|si|sí)
        echo 'Los documentos físicos, incluidas cargas privadas, permanecerán intactos.'
        echo 'Se perderán usuarios, bibliotecas registradas, historial y activación local.'
        printf 'Escribe ELIMINAR para confirmar: '
        read -r ANSWER || ANSWER=no
        [ "$ANSWER" != ELIMINAR ] || ERASE=yes
        ;;
esac
if /bin/launchctl print system/app.aibid.test >/dev/null 2>&1; then
    /bin/launchctl bootout system/app.aibid.test
fi
# Check exclusive access before removing code; do not touch stored documents.
if [ -f "$BASE/data/config.json" ]; then
    /usr/bin/sudo -u _aibidtest "$BASE/bin/gestor-documental" check-state --config "$BASE/data/config.json"
    if [ "$ERASE" = yes ]; then
        /usr/bin/sudo -u _aibidtest "$BASE/bin/gestor-documental" erase-internal-data --config "$BASE/data/config.json" --confirm-erase-internal-data
    fi
fi
# Only unmodified packaged files. Unknown files and all directories remain.
TAB=$(printf '\t')
while IFS="$TAB" read -r HASH FILE; do
    case "$FILE" in
        "$BASE/bin/"*|"$BASE/docs/"*|'/Applications/AIBID/'*|'/Applications/AIBID Pruebas/'*|'/Library/LaunchDaemons/app.aibid.test.plist') ;;
        *) echo 'Ruta de paquete inválida.' >&2; exit 1 ;;
    esac
    CURRENT="$FILE"
    while [ "$CURRENT" != / ]; do
        if [ -L "$CURRENT" ]; then echo "Enlace no permitido: $CURRENT" >&2; exit 1; fi
        CURRENT=$(/usr/bin/dirname "$CURRENT")
    done
done < "$BASE/package-files.tsv"
while IFS="$TAB" read -r HASH FILE; do
    if [ -f "$FILE" ] && [ "$(/usr/bin/shasum -a 256 "$FILE" | /usr/bin/cut -d ' ' -f 1)" = "$HASH" ]; then /bin/rm -f "$FILE"; fi
done < "$BASE/package-files.tsv"
/bin/rm -f "$BASE/package-files.tsv"
/usr/sbin/pkgutil --forget app.aibid.test >/dev/null
echo 'Programa eliminado. Documentos físicos y cuenta del servicio conservados.'
if [ "$ERASE" = yes ]; then echo 'Datos internos eliminados; la reinstalación comenzará desde cero.'
else echo "Datos conservados en $BASE/data."; fi
