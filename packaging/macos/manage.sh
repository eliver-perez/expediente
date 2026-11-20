#!/bin/bash
set -Eeuo pipefail
umask 077
[ "$(/usr/bin/id -u)" -eq 0 ] || { echo 'Se requiere autorización de administrador de macOS.' >&2; exit 1; }
BASE='/Library/Application Support/AIDEX'
. "$BASE/bin/installer-common.sh"
. "$BASE/profile.sh"
aidex_select_profile
[ -f "$CONFIG" ] || aidex_fail 'La configuración de AIDEX no está disponible.'

aidex_start_configured() {
    if ! aidex_service_registered; then
        /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap-ready --config "$CONFIG"
    fi
    aidex_start_service
}

case "$#:${1-}" in
    1:start) aidex_start_configured;;
    1:stop) aidex_stop_service;;
    1:restart) aidex_stop_service; aidex_start_configured;;
    2:port) aidex_change_port "$2";;
    1:doctor) /usr/bin/sudo -u "$ACCOUNT" "$BINARY" doctor --config "$CONFIG";;
    1:setup-terminal)
        /usr/bin/sudo -u "$ACCOUNT" "$BINARY" doctor --config "$CONFIG" --sample-directory "$CODE/docs/pdf"
        if ! aidex_service_registered && ! /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap-ready --config "$CONFIG" >/dev/null 2>&1; then
            /usr/bin/sudo -u "$ACCOUNT" "$BINARY" bootstrap --config "$CONFIG"
        fi
        aidex_start_configured;;
    3:setup)
        if aidex_service_registered; then aidex_fail 'Detén AIDEX antes de completar la configuración inicial.'; fi
        "$BINARY" macos-receive-bootstrap --config "$CONFIG" --bootstrap-socket "$2" --bootstrap-peer-uid "$3"
        echo 'Administrador inicial creado.'
        # Report account creation separately if launchd or the port needs attention.
        aidex_start_service;;
    *) aidex_fail 'Operación de administración inválida.';;
esac
