#!/bin/bash
set -euo pipefail
[[ $EUID == 0 ]] || { echo 'Ejecuta sudo ./install.sh' >&2; exit 1; }
source /etc/os-release
architecture=$(dpkg --print-architecture)
[[ $ID == ubuntu && $VERSION_ID == 24.04 && ( $architecture == amd64 || $architecture == arm64 ) ]] || exit 1
cd -- "$(dirname -- "$(realpath "$0")")"
sha256sum --strict -c SHA256SUMS
shopt -s nullglob
packages=(repository/aidex_*.deb)
[[ ${#packages[@]} == 1 && $(dpkg-deb -f "${packages[0]}" Architecture) == "$architecture" ]] || { echo "El bundle debe contener un paquete AIDEX para $architecture." >&2; exit 1; }
package_version=$(dpkg-deb -f "${packages[0]}" Version)
temporary=$(mktemp -d /var/tmp/aidex-offline.XXXXXXXX)
trap 'rm -rf -- "$temporary"' EXIT
chmod 755 "$temporary"
mkdir -m 755 "$temporary/lists"
ln -s -- "$PWD" "$temporary/repository"
printf 'deb [trusted=yes] file:%s ./\n' "$temporary/repository" > "$temporary/sources.list"
options=(-o "Dir::Etc::sourcelist=$temporary/sources.list" -o Dir::Etc::sourceparts=- -o "Dir::State::lists=$temporary/lists" -o APT::Get::List-Cleanup=0)
# APT reads only this local repository for this invocation; no permanent source is added.
apt-get "${options[@]}" update
apt-get "${options[@]}" --yes --no-install-recommends install "aidex=$package_version"
