#!/bin/bash
# Run online on Ubuntu 24.04 amd64; downloads the dependency closure from an empty
# dpkg status, not merely the packages missing from the preparation machine.
set -euo pipefail
if [[ $EUID != 0 || $# != 2 ]]; then echo "Uso: sudo $0 paquete.deb directorio-nuevo" >&2; exit 1; fi
source /etc/os-release
[[ $ID == ubuntu && $VERSION_ID == 24.04 && $(dpkg --print-architecture) == amd64 ]] || exit 1
package=$(realpath "$1")
destination=$(realpath -m "$2")
[[ $destination != *[[:space:]]* && ! -e $destination ]] || { echo 'Usa una carpeta nueva sin espacios.' >&2; exit 1; }
[[ $(dpkg-deb -f "$package" Package) == aibid-test ]] || exit 1
command -v python3 >/dev/null
mkdir -m 755 -p "$destination/repository/partial"
apt-get update
apt-get -o Dir::State::status=/dev/null -o Dir::Cache::archives="$destination/repository" \
    --download-only --yes --no-install-recommends install "$package"
cp "$package" "$destination/repository/"
cp "$(dirname "$0")/install-offline.sh" "$destination/install.sh"
chmod 755 "$destination/install.sh"
python3 - "$destination" <<'PY'
import hashlib, pathlib, subprocess, sys
root = pathlib.Path(sys.argv[1])
records = []
for package in sorted((root / "repository").glob("*.deb")):
    control = subprocess.check_output(["dpkg-deb", "-f", str(package)], text=True).rstrip()
    data = package.read_bytes()
    records.append(control + f"\nFilename: repository/{package.name}\nSize: {len(data)}\nSHA256: {hashlib.sha256(data).hexdigest()}\n\n")
(root / "Packages").write_text("".join(records))
files = [root / "Packages", root / "install.sh", *sorted((root / "repository").glob("*.deb"))]
(root / "SHA256SUMS").write_text("".join(hashlib.sha256(path.read_bytes()).hexdigest()+"  "+path.relative_to(root).as_posix()+"\n" for path in files))
PY
echo "Bundle preparado en $destination. Copia la carpeta completa y ejecuta sudo ./install.sh sin red en la máquina de prueba."
echo 'Guarda también su SHA256SUMS por un canal independiente. La instalación offline aún debe certificarse en una VM limpia.'
