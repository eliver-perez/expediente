#!/bin/sh
set -eu
if [ "$(/usr/bin/id -u)" -ne 0 ]; then exec /usr/bin/sudo /bin/sh "$0" "$@"; fi
. '/Library/Application Support/AIDEX/profile.sh'
aidex_select_profile
if [ ! -f "$CONFIG" ]; then echo 'Primero configura AIDEX.' >&2; exit 1; fi
old_address=$(/usr/bin/plutil -extract listen_address raw -o - "$CONFIG")
old_port=${old_address##*:}
printf 'Puerto actual de AIDEX: %s\n' "$old_port"
new_port=${1-}
if [ -z "$new_port" ]; then printf 'Nuevo puerto TCP (Enter para cancelar): '; IFS= read -r new_port; fi
[ -n "$new_port" ] || exit 0
case $new_port in *[!0-9]*|'') echo 'Introduce un puerto entre 1 y 65535.' >&2; exit 1;; esac
[ "$new_port" -ge 1 ] && [ "$new_port" -le 65535 ] || { echo 'Puerto fuera de intervalo.' >&2; exit 1; }
[ "$new_port" != "$old_port" ] || { echo 'El puerto ya está configurado.'; exit 0; }
was_loaded=0
if /bin/launchctl print "system/$LABEL" >/dev/null 2>&1; then
    was_loaded=1
    /bin/launchctl bootout "system/$LABEL"
fi
if ! /usr/bin/sudo -u "$ACCOUNT" "$BINARY" configure-port --config "$CONFIG" --port "$new_port"; then
    [ "$was_loaded" -eq 0 ] || /bin/launchctl bootstrap system "$PLIST"
    exit 1
fi
if [ "$was_loaded" -eq 1 ]; then
    ready=0
    if /bin/launchctl bootstrap system "$PLIST"; then
        attempt=0
        while [ "$attempt" -lt 30 ]; do
            if /usr/bin/curl -fsS --max-time 2 "http://127.0.0.1:$new_port/health/live" >/dev/null 2>&1; then ready=1; break; fi
            attempt=$((attempt + 1))
            /bin/sleep 1
        done
    fi
    if [ "$ready" -ne 1 ]; then
        echo 'El servicio no respondió. Restaurando el puerto anterior.' >&2
        /bin/launchctl bootout "system/$LABEL" >/dev/null 2>&1 || true
        /usr/bin/sudo -u "$ACCOUNT" "$BINARY" configure-port --config "$CONFIG" --port "$old_port"
        /bin/launchctl bootstrap system "$PLIST"
        exit 1
    fi
fi
printf 'AIDEX configurado en http://127.0.0.1:%s\n' "$new_port"
