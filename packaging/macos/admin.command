#!/bin/sh
set -eu
if [ "$(/usr/bin/id -u)" -ne 0 ]; then exec /usr/bin/sudo /bin/sh "$0"; fi
. '/Library/Application Support/AIDEX/profile.sh'
aidex_select_profile
echo 'AIDEX — configuración local. Contraseña de AIDEX: mínimo 6 caracteres.'
echo 'Puede solicitarse primero la contraseña de administrador de macOS (sudo).'
/usr/bin/sudo -u "$ACCOUNT" "$BINARY" doctor --config "$CONFIG" --sample-directory "$CODE/docs/pdf"
if ! /usr/bin/sudo test -f "$BASE/data/service-enabled"; then
    if ! /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap-ready --config "$CONFIG"; then
        /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap --config "$CONFIG"
    fi
    /usr/bin/sudo -u "$ACCOUNT" /usr/bin/touch "$BASE/data/service-enabled"
fi
if ! /bin/launchctl print system/app.aidex >/dev/null 2>&1; then
    /bin/launchctl enable system/app.aidex
    /usr/bin/sudo /bin/launchctl bootstrap system "$PLIST"
fi
echo 'Abre http://127.0.0.1:18090. El servicio continúa sin mantener esta consola abierta.'
if [ -n "${SUDO_USER-}" ]; then /usr/bin/sudo -u "$SUDO_USER" /usr/bin/open http://127.0.0.1:18090; fi
