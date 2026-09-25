#!/bin/sh
set -eu
if [ "$(/usr/bin/id -u)" -ne 0 ]; then exec /usr/bin/sudo /bin/sh "$0"; fi
BASE='/Library/Application Support/AIBID-Test'
if /bin/launchctl print system/app.aibid.test >/dev/null 2>&1; then
    /bin/launchctl bootout system/app.aibid.test
fi
# Check exclusive access before removing code; do not touch stored documents.
if [ -f "$BASE/data/config.json" ]; then
    /usr/bin/sudo -u _aibidtest "$BASE/bin/gestor-documental" migrate --config "$BASE/data/config.json"
fi
/bin/rm -f /Library/LaunchDaemons/app.aibid.test.plist
/bin/rm -rf "$BASE/bin" "$BASE/docs" '/Applications/AIBID Pruebas'
/usr/sbin/pkgutil --forget app.aibid.test >/dev/null
echo "Servicio y programa eliminados. Datos, documentos y cuenta preservados en $BASE/data."
