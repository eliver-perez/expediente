#!/bin/sh
# AIDEX owns these paths only; no previous product state is imported.
# The optional prefix is used only by isolated tests; installed callers pass none.
aidex_select_profile() {
    prefix=${1-}
    ACCOUNT=aidex
    DATA="$prefix/var/lib/aidex"
    CONFIG_DIR="$prefix/etc/aidex"
    CONFIG="$CONFIG_DIR/config.json"
    BINARY="$prefix/usr/lib/aidex/aidex-server"
    UNIT=aidex.service
    for path in "$DATA" "$CONFIG_DIR" "$CONFIG" "$BINARY" "$prefix/etc/systemd/system/aidex.service.d/installation.conf"; do
        current=$path
        while [ "$current" != / ] && [ -n "$current" ]; do
            [ ! -L "$current" ] || { echo "Enlace no permitido: $current" >&2; return 1; }
            current=$(dirname "$current")
        done
    done
}
