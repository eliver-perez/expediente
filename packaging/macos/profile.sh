#!/bin/sh
# Do not move persistent state or change UID/GID during an upgrade.
aidex_select_profile() {
    prefix=${1-}
    CODE="$prefix/Library/Application Support/AIDEX"
    BASE="$CODE"
    ACCOUNT=_aidex
    CONFIG="$BASE/data/config.json"
    BINARY="$CODE/bin/aidex-server"
    LABEL=app.aidex
    PLIST="$prefix/Library/LaunchDaemons/app.aidex.plist"
    for path in "$BASE" "$CODE" "$CONFIG" "$PLIST" "$CODE/package-files.tsv" "$CODE/package-files.tsv.new"; do
        current=$path
        while [ "$current" != / ] && [ -n "$current" ]; do
            [ ! -L "$current" ] || { echo "Enlace no permitido: $current" >&2; return 1; }
            current=$(dirname "$current")
        done
    done
}
