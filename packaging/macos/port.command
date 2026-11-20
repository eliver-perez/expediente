#!/bin/sh
set -eu
BASE='/Library/Application Support/AIDEX'
if current=$(/usr/bin/plutil -extract port raw -o - "$BASE/control/endpoint.json" 2>/dev/null); then
    printf 'Puerto actual: %s\n' "$current"
fi
new_port=${1-}
if [ -z "$new_port" ]; then
    printf 'Nuevo puerto TCP (Enter para cancelar): '
    IFS= read -r new_port
fi
[ -n "$new_port" ] || exit 0
exec /usr/bin/sudo "$BASE/bin/aidex-manage" port "$new_port"
