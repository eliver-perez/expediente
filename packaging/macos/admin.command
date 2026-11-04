#!/bin/sh
set -eu
if [ "$(/usr/bin/id -u)" -ne 0 ]; then exec /usr/bin/sudo /bin/sh "$0"; fi
. '/Library/Application Support/AIDEX/profile.sh'
aidex_select_profile
echo 'AIDEX — configuración local. Contraseña de AIDEX: mínimo 6 caracteres.'
echo 'Puede solicitarse primero la contraseña de administrador de macOS (sudo).'
/usr/bin/sudo -u "$ACCOUNT" "$BINARY" doctor --config "$CONFIG" --sample-directory "$CODE/docs/pdf"
if ! /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap-ready --config "$CONFIG" >/dev/null 2>&1; then
    /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap --config "$CONFIG"
fi
if ! /usr/bin/sudo test -f "$BASE/data/service-enabled"; then
    /usr/bin/sudo -u "$ACCOUNT" /usr/bin/touch "$BASE/data/service-enabled"
fi
ADDRESS=$(/usr/bin/plutil -extract listen_address raw -o - "$CONFIG")
PORT=${ADDRESS##*:}
case "$PORT" in *[!0-9]*|'') echo 'El puerto configurado no es válido.' >&2; exit 1;; esac
if /usr/sbin/lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1 && ! /bin/launchctl print "system/$LABEL" >/dev/null 2>&1; then
    echo "El puerto $PORT está ocupado por otra aplicación. Detén el servicio anterior o cambia el puerto antes de iniciar AIDEX." >&2
    exit 1
fi
if ! /bin/launchctl print system/app.aidex >/dev/null 2>&1; then
    /bin/launchctl enable system/app.aidex
    /usr/bin/sudo /bin/launchctl bootstrap system "$PLIST"
else
    /bin/launchctl kickstart -k "system/$LABEL"
fi
ATTEMPT=0
while [ "$ATTEMPT" -lt 30 ]; do
    if /usr/bin/curl -fsS --max-time 2 "http://127.0.0.1:$PORT/health/live" 2>/dev/null | /usr/bin/grep -q '"product":"AIDEX"'; then break; fi
    ATTEMPT=$((ATTEMPT + 1))
    /bin/sleep 1
done
[ "$ATTEMPT" -lt 30 ] || { echo "AIDEX no respondió en el puerto $PORT. Revisa $BASE/data/logs/service.log." >&2; exit 1; }
echo "Abre http://127.0.0.1:$PORT. El servicio continúa sin mantener esta consola abierta."
if [ -n "${SUDO_USER-}" ]; then /usr/bin/sudo -u "$SUDO_USER" /usr/bin/open "http://127.0.0.1:$PORT"; fi
