#!/bin/sh
set -eu
BASE='/Library/Application Support/AIDEX'
echo 'AIDEX — configuración local. Contraseña de AIDEX: mínimo 6 caracteres.'
echo 'macOS puede solicitar autorización de administrador.'
/usr/bin/sudo "$BASE/bin/aidex-manage" setup-terminal
if url=$(/usr/bin/plutil -extract url raw -o - "$BASE/control/endpoint.json" 2>/dev/null); then
    printf 'Abre %s. El servicio continúa cuando cierres esta consola.\n' "$url"
    /usr/bin/open "$url"
fi
