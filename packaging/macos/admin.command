#!/bin/sh
set -eu
BASE='/Library/Application Support/AIBID-Test'
CONFIG="$BASE/data/config.json"
echo 'AIBID Pruebas — configuración local. Contraseña de AIBID: mínimo 6 caracteres.'
echo 'Puede solicitarse primero la contraseña de administrador de macOS (sudo).'
/usr/bin/sudo -u _aibidtest "$BASE/bin/gestor-documental" doctor --config "$CONFIG" --sample-directory "$BASE/docs/pdf"
if ! /usr/bin/sudo test -f "$BASE/data/service-enabled"; then
    if ! /usr/bin/sudo -u _aibidtest "$BASE/bin/gestor-documental" bootstrap-ready --config "$CONFIG"; then
        /usr/bin/sudo -u _aibidtest "$BASE/bin/gestor-documental" bootstrap --config "$CONFIG"
    fi
    /usr/bin/sudo -u _aibidtest /usr/bin/touch "$BASE/data/service-enabled"
fi
if ! /bin/launchctl print system/app.aibid.test >/dev/null 2>&1; then
    /usr/bin/sudo /bin/launchctl bootstrap system /Library/LaunchDaemons/app.aibid.test.plist
fi
echo 'Abre http://127.0.0.1:18090. El servicio continúa sin mantener esta consola abierta.'
/usr/bin/open http://127.0.0.1:18090
