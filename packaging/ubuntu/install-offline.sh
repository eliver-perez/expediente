#!/bin/bash
set -euo pipefail
[[ $EUID == 0 ]] || { echo 'Ejecuta sudo ./install.sh' >&2; exit 1; }
source /etc/os-release
architecture=$(dpkg --print-architecture)
[[ $ID == ubuntu && $VERSION_ID == 24.04 && ( $architecture == amd64 || $architecture == arm64 ) ]] || exit 1
cd -- "$(dirname -- "$(realpath "$0")")"
[[ $PWD != *[[:space:]]* ]] || { echo 'Usa una ruta sin espacios.' >&2; exit 1; }
sha256sum --strict -c SHA256SUMS
shopt -s nullglob
packages=(repository/aibid-test_*.deb)
[[ ${#packages[@]} == 1 && $(dpkg-deb -f "${packages[0]}" Architecture) == "$architecture" ]] || { echo "El bundle debe contener un paquete AIBID para $architecture." >&2; exit 1; }
temporary=$(mktemp -d /var/tmp/aibid-offline.XXXXXXXX)
trap 'rm -rf -- "$temporary"' EXIT
chmod 755 "$temporary"
mkdir -m 755 "$temporary/lists"
printf 'deb [trusted=yes] file:%s ./\n' "$PWD" > "$temporary/sources.list"
options=(-o "Dir::Etc::sourcelist=$temporary/sources.list" -o Dir::Etc::sourceparts=- -o "Dir::State::lists=$temporary/lists" -o APT::Get::List-Cleanup=0)
# APT reads only this local repository for this invocation; no permanent source is added.
apt-get "${options[@]}" update
apt-get "${options[@]}" --yes --no-install-recommends install aibid-test=0.7.0~test.1
