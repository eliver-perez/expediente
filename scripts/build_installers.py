#!/usr/bin/env python3
"""Build isolated H7 preview installers. No host service or account is modified.

Frontend must be built first. Native runtime verification remains a separate step.
Windows requires a conda-forge win-64 prefix and NSIS (see docs/H7.md).
"""
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import plistlib
import shutil
import struct
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
VERSION = "2.0.0-alpha.3"
PACKAGE_REVISION = "f3"
DEB_VERSION = "2.0.0~alpha.3"
EPOCH = 1790294400  # Fixed packaging timestamp, 2026-09-25 UTC.


def run(arguments, **kwargs):
    subprocess.run([str(argument) for argument in arguments], check=True, cwd=ROOT, **kwargs)


def copy(source, target, mode=0o644, template=False):
    target.parent.mkdir(parents=True, exist_ok=True)
    if template:
        target.write_text(Path(source).read_text().replace("@VERSION@", VERSION), encoding="utf-8-sig" if target.suffix == ".ps1" else "utf-8")
    else:
        shutil.copyfile(source, target)
    target.chmod(mode)


def binary(destination, system, architecture):
    destination.parent.mkdir(parents=True, exist_ok=True)
    environment = dict(os.environ, CGO_ENABLED="0", GOOS=system, GOARCH=architecture)
    run(["go", "build", "-trimpath", "-ldflags",
         "-s -w -X gestor-documental/internal/buildinfo.Channel=installer-test"
         + " -X gestor-documental/internal/buildinfo.Revision="+PACKAGE_REVISION
         + (" -X gestor-documental/internal/buildinfo.ServiceName=AIBIDTest" if system == "windows" else ""),
         "-o", destination, "./cmd/gestor-documental"], env=environment)


def documents(destination):
    copy(ROOT / "docs/AIBID-2.0-FASE-3.md", destination / "FASE-3.md")
    copy(ROOT / "docs/AIBID-2.0-FASE-2.md", destination / "AIBID-2.0-FASE-2.md")
    copy(ROOT / "docs/H7.md", destination / "LEEME.md")
    readme = destination / "LEEME.md"
    readme.write_text("# AIBID 2.0 — fase 3 (alpha de pruebas)\n\nConsulta [FASE-3.md](FASE-3.md) para formatos, reglas, migración y límites de esta entrega. Los nombres de versión de la guía H7 siguiente son históricos.\n\n" + readme.read_text().replace("../packaging/TEST-PLAN.md", "PRUEBAS.md").replace("H7-WINDOWS-R2.md", "WINDOWS-R2.md"))
    copy(ROOT / "LICENSE_CONTRACT.md", destination / "LICENSE_CONTRACT.md")
    copy(ROOT / "packaging/TEST-PLAN.md", destination / "PRUEBAS.md")
    copy(ROOT / "packaging/THIRD-PARTY.md", destination / "THIRD-PARTY.md")
    copy(ROOT / "docs/H7-WINDOWS-R2.md", destination / "WINDOWS-R2.md")
    copy(ROOT / "docs/REVISION-PROCESAMIENTO.md", destination / "REVISION-PROCESAMIENTO.md")
    copy(ROOT / "docs/REVISION-R4.md", destination / "REVISION-R4.md")
    copy(ROOT / "docs/REVISION-R6.md", destination / "REVISION-R6.md")
    copy(ROOT / "docs/CIERRE-1.0.md", destination / "CIERRE-1.0.md")
    # A few small fixtures exercise actual PDF and image OCR after native installation.
    for path in (ROOT / "testdata/documents").glob("*.pdf"):
        copy(path, destination / "pdf" / path.name)
    for extension in ["docx", "xlsx", "txt", "csv"]:
        copy(ROOT / "testdata/documents" / ("sample." + extension), destination / "formatos" / ("sample." + extension))
    for name in ["blank-leading.docx", "printer-settings.xlsx"]:
        copy(ROOT / "testdata/documents" / name, destination / "formatos" / name)


def macos(output, work):
    payload = work / "root"
    base = payload / "Library/Application Support/AIBID-Test"
    binary(base / "bin/gestor-documental", "darwin", "arm64")
    # Local ad-hoc Mach-O signature for Apple Silicon execution; not notarization.
    run(["codesign", "--force", "--sign", "-", base / "bin/gestor-documental"])
    documents(base / "docs")
    apps = payload / "Applications/AIBID Pruebas"
    for source, name in [("admin.command", "Configurar AIBID.command"), ("uninstall.command", "Desinstalar AIBID.command")]:
        copy(ROOT / "packaging/macos" / source, apps / name, 0o755)
    (apps / "Abrir AIBID.webloc").write_bytes(plistlib.dumps({"URL": "http://127.0.0.1:18090"}))
    daemon = payload / "Library/LaunchDaemons/app.aibid.test.plist"
    daemon.parent.mkdir(parents=True, exist_ok=True)
    install = "/Library/Application Support/AIBID-Test"
    daemon.write_bytes(plistlib.dumps({
        "Label": "app.aibid.test", "UserName": "_aibidtest", "GroupName": "_aibidtest",
        "ProgramArguments": [install+"/bin/gestor-documental", "serve", "--config", install+"/data/config.json"],
        "WorkingDirectory": install+"/data", "RunAtLoad": True,
        "KeepAlive": {"SuccessfulExit": False}, "ThrottleInterval": 10, "ExitTimeOut": 60,
        "Umask": 0o077, "EnvironmentVariables": {"PATH": "/opt/homebrew/bin:/usr/bin:/bin"},
        "StandardOutPath": install+"/data/logs/service.log", "StandardErrorPath": install+"/data/logs/service.log"
    }))
    manifest = [hashlib.sha256(path.read_bytes()).hexdigest() + "\t/" + path.relative_to(payload).as_posix()
                for path in sorted(payload.rglob("*")) if path.is_file()]
    (base / "package-files.tsv").write_text("\n".join(manifest) + "\n")
    # Installer ownership must never inherit the developer's UID or group.
    scripts = work / "scripts"
    for name in ["preinstall", "postinstall"]:
        copy(ROOT / "packaging/macos" / name, scripts / name, 0o755, template=True)
    component = work / "component.pkg"
    run(["pkgbuild", "--root", payload, "--scripts", scripts, "--identifier", "app.aibid.test",
         "--version", "2.0.0.3", "--install-location", "/", "--ownership", "recommended", component])
    distribution = work / "distribution.xml"
    distribution.write_text('''<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
  <title>AIBID Pruebas ''' + VERSION + '''</title>
  <options customize="never" require-scripts="false" hostArchitectures="arm64"/>
  <domains enable_localSystem="true" enable_currentUserHome="false" enable_anywhere="false"/>
  <allowed-os-versions><os-version min="13.0"/></allowed-os-versions>
  <choices-outline><line choice="app.aibid.test"/></choices-outline>
  <choice id="app.aibid.test" visible="false"><pkg-ref id="app.aibid.test"/></choice>
  <pkg-ref id="app.aibid.test" version="2.0.0.3">component.pkg</pkg-ref>
</installer-gui-script>
''')
    run(["productbuild", "--distribution", distribution, "--package-path", work,
         output / f"AIBID-Pruebas-{VERSION}-macOS-arm64.pkg"])


def tar_bytes(directory):
    buffer = io.BytesIO()
    with gzip.GzipFile(fileobj=buffer, mode="wb", mtime=EPOCH) as compressed:
        with tarfile.open(fileobj=compressed, mode="w", format=tarfile.GNU_FORMAT) as archive:
            for path in sorted(directory.rglob("*")):
                info = archive.gettarinfo(str(path), "./" + path.relative_to(directory).as_posix())
                info.uid = info.gid = 0
                info.uname = info.gname = "root"
                info.mtime = EPOCH
                with path.open("rb") if path.is_file() else io.BytesIO() as contents:
                    archive.addfile(info, contents if path.is_file() else None)
    return buffer.getvalue()


def ar_member(stream, name, data):
    header = f"{name+'/':<16}{EPOCH:<12}{0:<6}{0:<6}{'100644':<8}{len(data):<10}`\n"
    assert len(header) == 60
    stream.write(header.encode("ascii"))
    stream.write(data)
    if len(data) % 2:
        stream.write(b"\n")


def ubuntu(output, work, architecture):
    payload, control = work / "root", work / "control"
    binary(payload / "usr/lib/aibid-test/gestor-documental", "linux", architecture)
    documents(payload / "usr/share/doc/aibid-test")
    for name in ["prepare-offline.sh", "install-offline.sh"]:
        copy(ROOT / "packaging/ubuntu" / name, payload / "usr/share/doc/aibid-test/offline" / name, 0o755)
    copy(ROOT / "packaging/ubuntu/aibid-test.service", payload / "usr/lib/systemd/system/aibid-test.service")
    copy(ROOT / "packaging/ubuntu/aibid-test-admin", payload / "usr/sbin/aibid-test-admin", 0o755)
    copy(ROOT / "packaging/ubuntu/aibid-test-uninstall", payload / "usr/sbin/aibid-test-uninstall", 0o755)
    control.mkdir()
    size = sum(path.stat().st_size for path in payload.rglob("*") if path.is_file()) // 1024
    (control / "control").write_text(f"""Package: aibid-test
Version: {DEB_VERSION}
Section: utils
Priority: optional
Architecture: {architecture}
Maintainer: AIBID local test builds <aibid@example.invalid>
Installed-Size: {size}
Depends: adduser, util-linux, systemd, poppler-utils, tesseract-ocr, tesseract-ocr-spa, tesseract-ocr-eng, fonts-dejavu-core
Description: AIBID - Aplicacion de Indexacion de Bibliotecas Digitales (pruebas)
 Servicio local aislado de prueba. Datos conservados incluso al desinstalar.
 Requiere activar una licencia para habilitar las modificaciones.
""")
    for name in ["preinst", "postinst", "prerm", "postrm"]:
        copy(ROOT / "packaging/ubuntu" / name, control / name, 0o755, template=True)
    with (output / f"aibid-test_{DEB_VERSION}_{architecture}.deb").open("wb") as stream:
        stream.write(b"!<arch>\n")
        ar_member(stream, "debian-binary", b"2.0\n")
        ar_member(stream, "control.tar.gz", tar_bytes(control))
        ar_member(stream, "data.tar.gz", tar_bytes(payload))


def windows_tools(prefix, destination, docs):
    """Select runtime files; preserve upstream licenses, recipes, and package digests."""
    manifest = []
    expected_hashes = {}
    for record_path in sorted((prefix / "conda-meta").glob("*.json")):
        record = json.loads(record_path.read_text())
        for entry in record.get("paths_data", {}).get("paths", []):
            digest = entry.get("sha256_in_prefix") or entry.get("sha256")
            if digest:
                expected_hashes[entry["_path"]] = digest
        manifest.append({key: record.get(key) for key in ("name", "version", "build", "url", "sha256", "license")})
        upstream = Path(record["extracted_package_dir"]) / "info"
        if not upstream.is_dir():
            raise RuntimeError(f"Missing upstream metadata: {upstream}")
        for component in ["licenses", "recipe"]:
            if (upstream / component).is_dir():
                shutil.copytree(upstream / component, docs / "third-party" / record["name"] / component)
        if (upstream / "about.json").exists():
            copy(upstream / "about.json", docs / "third-party" / record["name"] / "about.json")
    if not manifest:
        raise RuntimeError("Windows prefix does not contain package metadata")
    locked = json.loads((ROOT / "packaging/windows/ocr.lock.json").read_text())
    if manifest != locked:
        raise RuntimeError("Windows OCR prefix differs from packaging/windows/ocr.lock.json; use ocr-explicit.txt")
    def copy_verified(source, target):
        expected = expected_hashes.get(source.relative_to(prefix).as_posix())
        if not expected or hashlib.sha256(source.read_bytes()).hexdigest() != expected:
            raise RuntimeError(f"Windows runtime file differs from package metadata: {source}")
        copy(source, target)
    # DLLs beside the four executables are found without global PATH modification.
    for source in sorted(prefix.rglob("*.dll")):
        target = destination / "Library/bin" / source.name
        if target.exists() and target.read_bytes() != source.read_bytes():
            raise RuntimeError(f"Different DLLs share a filename: {source.name}")
        copy_verified(source, target)
    for name in ["pdfinfo", "pdftotext", "pdftoppm", "tesseract"]:
        copy_verified(prefix / "Library/bin" / (name + ".exe"), destination / "Library/bin" / (name + ".exe"))
    for name in ["spa", "eng", "osd"]:
        copy_verified(prefix / "share/tessdata" / (name + ".traineddata"), destination / "tessdata" / (name + ".traineddata"))
    for component in ["configs", "tessconfigs"]:
        shutil.copytree(prefix / "Library/share/tessdata" / component, destination / "tessdata" / component)
    # conda-forge's windows-data.patch removes BOTH bin and Library from the DLL
    # path, then appends share/poppler. Preserve that upstream relative layout.
    shutil.copytree(prefix / "share/poppler", destination / "share/poppler")
    shutil.copytree(prefix / "fonts", destination / "fonts")
    font_config = destination / "fonts.conf"
    font_config.write_text('<?xml version="1.0"?><!DOCTYPE fontconfig SYSTEM "urn:fontconfig:fonts.dtd">'
                           '<fontconfig><dir prefix="relative">fonts</dir>'
                           '<dir>WINDOWSFONTDIR</dir><cachedir prefix="xdg">fontconfig</cachedir></fontconfig>')
    (docs / "windows-ocr.lock.json").write_text(json.dumps(manifest, indent=2) + "\n")
    return manifest


def check_pe_imports(directory):
    """Detect missing non-system DLLs before shipping; native execution is still required."""
    system = set("kernel32 kernelbase ntdll user32 advapi32 gdi32 shell32 ole32 oleaut32 comdlg32 comctl32 "
                 "ws2_32 crypt32 bcrypt secur32 wldap32 winmm version shlwapi normaliz iphlpapi "
                 "userenv usp10 rpcrt4 psapi powrprof netapi32 mpr dhcpcsvc dnsapi mswsock "
                 "msvcrt imm32 setupapi winspool winhttp wininet shcore uxtheme dwrite d2d1 "
                 "dxgi d3d11 d3d12 dbghelp hid authz wintrust ncrypt cfgmgr32 win32u wsock32 "
                 "msasn1 cabinet msimg32 cryptbase bcryptprimitives credui netutils wkscli dsrole d3dcompiler_47".split())
    available = {path.name.lower() for path in directory.glob("*")}
    missing = []
    for path in directory.glob("*"):
        if path.suffix.lower() not in (".exe", ".dll"):
            continue
        data = path.read_bytes()
        pe = struct.unpack_from("<I", data, 0x3c)[0]
        machine, sections = struct.unpack_from("<HH", data, pe + 4)
        if machine != 0x8664:
            raise RuntimeError(f"Expected amd64 runtime PE: {path}")
        optional_size = struct.unpack_from("<H", data, pe + 20)[0]
        optional = pe + 24
        imports = struct.unpack_from("<I", data, optional + 112 + 8)[0]
        def file_offset(address):
            for index in range(sections):
                section = optional + optional_size + index * 40
                virtual_size, virtual, raw_size, raw = struct.unpack_from("<IIII", data, section + 8)
                if virtual <= address < virtual + max(virtual_size, raw_size):
                    return raw + address - virtual
            raise RuntimeError(f"Invalid PE address in {path.name}")
        if not imports:
            continue
        position = file_offset(imports)
        while any(data[position:position + 20]):
            name_address = struct.unpack_from("<I", data, position + 12)[0]
            offset = file_offset(name_address)
            name = data[offset:data.index(b"\0", offset)].decode("ascii").lower()
            if name not in available and name.removesuffix(".dll") not in system and not name.startswith(("api-ms-", "ext-ms-")):
                missing.append(f"{path.name} -> {name}")
            position += 20
    if missing:
        raise RuntimeError("Missing DLLs: " + "; ".join(missing))
    print("Windows runtime PE/amd64 import closure: OK (native test still required)")


def check_windows_icon(executable, gui=False):
    """Verify actual PE resources and all icon sizes before packaging."""
    data = executable.read_bytes()
    pe = struct.unpack_from("<I", data, 0x3c)[0]
    machine, count = struct.unpack_from("<HH", data, pe + 4)
    assert machine == 0x8664, "Expected Windows amd64 executable"
    optional = pe + 24
    section_start = optional + struct.unpack_from("<H", data, pe + 20)[0]
    def offset(address):
        for index in range(count):
            size, virtual, raw_size, raw = struct.unpack_from("<IIII", data, section_start + index * 40 + 8)
            if virtual <= address < virtual + max(size, raw_size):
                return raw + address - virtual
        raise RuntimeError("Invalid resource RVA")
    resource_rva = struct.unpack_from("<I", data, optional + 112 + 16)[0]
    assert resource_rva, "Missing icon resource section"
    base = offset(resource_rva)
    groups = []
    def visit(relative, ancestry):
        directory = base + relative
        named, identifiers = struct.unpack_from("<HH", data, directory + 12)
        for index in range(named + identifiers):
            identifier, target = struct.unpack_from("<II", data, directory + 16 + index * 8)
            path = ancestry + [identifier]
            if target & 0x80000000:
                visit(target & 0x7fffffff, path)
            elif path[0] == 14:  # RT_GROUP_ICON
                address, length = struct.unpack_from("<II", data, base + target)
                groups.append(data[offset(address):offset(address) + length])
    visit(0, [])
    assert groups, "Missing RT_GROUP_ICON"
    sizes = set()
    for group in groups:
        frames = struct.unpack_from("<H", group, 4)[0]
        sizes.update(group[6 + index * 14] or 256 for index in range(frames))
    assert sizes == {16, 24, 32, 40, 48, 64, 128, 256}, sizes
    if gui:
        assert struct.unpack_from("<H", data, optional + 68)[0] == 2, "Launcher must use Windows GUI subsystem"
    print(f"Windows icon resources: {executable.name}: {sorted(sizes)} OK")


def windows(output, work, prefix, makensis):
    if not prefix or not makensis:
        raise RuntimeError("Windows requires --windows-tools and --makensis")
    payload = work / "root"
    # The checked-in .syso resources are reproducible from the existing brand SVG.
    copy(ROOT / "packaging/windows/aibid.ico", payload / "aibid.ico")
    binary(payload / "gestor-documental.exe", "windows", "amd64")
    run(["go", "build", "-trimpath", "-ldflags", "-s -w -H windowsgui", "-o", payload / "AIBID.exe", "./cmd/aibid-launcher"],
        env=dict(os.environ, CGO_ENABLED="0", GOOS="windows", GOARCH="amd64"))
    check_windows_icon(payload / "gestor-documental.exe")
    check_windows_icon(payload / "AIBID.exe", gui=True)
    documents(payload / "docs")
    windows_tools(prefix, payload / "tools", payload / "docs")
    check_pe_imports(payload / "tools/Library/bin")
    copy(ROOT / "packaging/windows/manage.ps1", payload / "manage.ps1", template=True)
    copy(ROOT / "packaging/windows/admin.cmd", payload / "admin.cmd")
    manifest = [{"path": path.relative_to(payload).as_posix(), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}
                for path in sorted(payload.rglob("*")) if path.is_file()]
    (payload / "package-files.json").write_text(json.dumps(manifest), encoding="utf-8")
    define = "/D" if os.name == "nt" else "-D"
    package_version = VERSION+"-"+PACKAGE_REVISION
    run([makensis, f"{define}VERSION={package_version}", f"{define}STAGE={payload}",
         f"{define}OUTPUT={output / ('AIBID-Pruebas-' + package_version + '-Windows-amd64.exe')}", ROOT / "packaging/windows/installer.nsi"])
    copy(payload / "docs/windows-ocr.lock.json", output / "windows-ocr.lock.json")
    copy(ROOT / "packaging/windows/ocr-explicit.txt", output / "windows-ocr-explicit.txt")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=["macos", "windows", "ubuntu", "all"], required=True)
    parser.add_argument("--output", type=Path, default=ROOT / "dist/aibid-2.0-fase-3")
    parser.add_argument("--windows-tools", type=Path)
    parser.add_argument("--makensis")
    parser.add_argument("--ubuntu-arch", choices=["amd64", "arm64", "all"], default="all",
                        help="Ubuntu architectures to build (default: both)")
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    targets = ["macos", "ubuntu", "windows"] if args.target == "all" else [args.target]
    for target in targets:
        with tempfile.TemporaryDirectory(prefix="aibid-package-") as directory:
            work = Path(directory).resolve()
            if target == "windows":
                windows(output, work, args.windows_tools, args.makensis)
            elif target == "ubuntu":
                architectures = ["amd64", "arm64"] if args.ubuntu_arch == "all" else [args.ubuntu_arch]
                for architecture in architectures:
                    ubuntu(output, work / architecture, architecture)
            else:
                macos(output, work)
    documents(output / "LEEME")
    for name in ["prepare-offline.sh", "install-offline.sh"]:
        copy(ROOT / "packaging/ubuntu" / name, output / "ubuntu-offline" / name, 0o755)
    records = []
    for artifact in sorted(output.rglob("*")):
        if artifact.is_file() and artifact.name != "SHA256SUMS" and not any(part.startswith(".") for part in artifact.relative_to(output).parts):
            records.append(hashlib.sha256(artifact.read_bytes()).hexdigest() + "  " + artifact.relative_to(output).as_posix())
    (output / "SHA256SUMS").write_text("\n".join(records) + "\n")
    print(f"Installers: {output}")


if __name__ == "__main__":
    main()
