#!/usr/bin/env python3
"""Build AIDEX production installers. No host service or account is modified.

Frontend must be built first. Native runtime verification remains a separate step.
Windows requires the locked OCR runtime, PostgreSQL 18 archive and NSIS.
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
import zipfile

ROOT = Path(__file__).resolve().parents[1]
VERSION = "1.0.0"
PACKAGE_REVISION = ""
DEB_VERSION = VERSION
EPOCH = 1790294400  # Fixed packaging timestamp, 2026-09-25 UTC.


def run(arguments, **kwargs):
    subprocess.run([str(argument) for argument in arguments], check=True, cwd=ROOT, timeout=600, **kwargs)


def copy(source, target, mode=0o644, template=False):
    target.parent.mkdir(parents=True, exist_ok=True)
    if template:
        # Decode a possible source BOM before adding the single BOM required by
        # Windows PowerShell 5.1 for accented UTF-8 text. Two BOMs break param().
        target.write_text(Path(source).read_text(encoding="utf-8-sig").replace("@VERSION@", VERSION).replace("@PROFILE@", (ROOT / "packaging/ubuntu/profile.sh").read_text(encoding="utf-8")), encoding="utf-8-sig" if target.suffix == ".ps1" else "utf-8")
    else:
        shutil.copyfile(source, target)
    target.chmod(mode)


def binary(destination, system, architecture):
    destination.parent.mkdir(parents=True, exist_ok=True)
    environment = dict(os.environ, CGO_ENABLED="0", GOOS=system, GOARCH=architecture)
    run(["go", "build", "-trimpath", "-ldflags",
         "-s -w -X gestor-documental/internal/buildinfo.Channel=production"
         + " -X gestor-documental/internal/buildinfo.Revision="+PACKAGE_REVISION
         + (" -X gestor-documental/internal/buildinfo.ServiceName=AIDEX" if system == "windows" else ""),
         "-o", destination, "./cmd/gestor-documental"], env=environment)


def documents(destination):
    guide = (ROOT / "docs/PRODUCCION.md").read_text(encoding="utf-8").replace("../LICENSE_CONTRACT.md", "LICENSE_CONTRACT.md").replace("TRANSICION-PRODUCCION.md", "TRANSICION.md").replace("../packaging/THIRD-PARTY.md", "THIRD-PARTY.md")
    for name in ["LEEME.md", "PRODUCCION.md"]:
        (destination / name).parent.mkdir(parents=True, exist_ok=True)
        (destination / name).write_text(guide, encoding="utf-8")
    copy(ROOT / "docs/TRANSICION-PRODUCCION.md", destination / "TRANSICION.md")
    copy(ROOT / "LICENSE_CONTRACT.md", destination / "LICENSE_CONTRACT.md")
    copy(ROOT / "packaging/THIRD-PARTY.md", destination / "THIRD-PARTY.md")
    copy(ROOT / "web/node_modules/chart.js/LICENSE.md", destination / "third-party/frontend/chartjs-MIT.txt")
    copy(ROOT / "web/node_modules/@kurkle/color/LICENSE.md", destination / "third-party/frontend/kurkle-color-MIT.txt")
    # A few small fixtures exercise actual PDF and image OCR after native installation.
    for path in (ROOT / "testdata/documents").glob("*.pdf"):
        copy(path, destination / "pdf" / path.name)
    for extension in ["docx", "xlsx", "txt", "csv"]:
        copy(ROOT / "testdata/documents" / ("sample." + extension), destination / "formatos" / ("sample." + extension))
    for name in ["blank-leading.docx", "printer-settings.xlsx"]:
        copy(ROOT / "testdata/documents" / name, destination / "formatos" / name)


def macos_uninstaller(destination, work):
    """Compile the native applet and bundle the root-owned, narrow helper."""
    run(["osacompile", "-o", destination, ROOT / "packaging/macos/uninstall.applescript"])
    resources = destination / "Contents/Resources"
    copy(ROOT / "packaging/macos/uninstall-helper.sh", resources / "uninstall-helper.sh", 0o755)
    # The official icon is already rasterized in the project's Windows asset.
    # sips selects its 256 px frame; no extra graphics dependency is required.
    source = work / "aidex-icon.png"
    run(["sips", "-s", "format", "png", ROOT / "packaging/windows/aidex.ico",
         "--out", source], stdout=subprocess.DEVNULL)
    iconset = work / "aidex.iconset"
    iconset.mkdir()
    for size, name in [(16, "icon_16x16.png"), (32, "icon_16x16@2x.png"),
                       (32, "icon_32x32.png"), (64, "icon_32x32@2x.png"),
                       (128, "icon_128x128.png"), (256, "icon_128x128@2x.png"),
                       (256, "icon_256x256.png"), (512, "icon_256x256@2x.png"),
                       (512, "icon_512x512.png"), (1024, "icon_512x512@2x.png")]:
        run(["sips", "-z", size, size, source, "--out", iconset / name], stdout=subprocess.DEVNULL)
    run(["iconutil", "-c", "icns", iconset, "-o", resources / "aidex.icns"])
    info = destination / "Contents/Info.plist"
    details = plistlib.loads(info.read_bytes())
    details.update(CFBundleIdentifier="mx.com.aidex.uninstaller", CFBundleName="Desinstalar AIDEX",
                   CFBundleDisplayName="Desinstalar AIDEX", CFBundleIconFile="aidex.icns",
                   CFBundleVersion=VERSION, CFBundleShortVersionString=VERSION,
                   LSMinimumSystemVersion="13.0")
    details.pop("CFBundleIconName", None)
    info.write_bytes(plistlib.dumps(details))
    run(["codesign", "--force", "--deep", "--sign", "-", destination])
    run(["codesign", "--verify", "--deep", "--strict", destination])


def macos(output, work):
    payload = work / "root"
    base = payload / "Library/Application Support/AIDEX"
    binary(base / "bin/aidex-server", "darwin", "arm64")
    # Local ad-hoc Mach-O signature for Apple Silicon execution; not notarization.
    run(["codesign", "--force", "--sign", "-", base / "bin/aidex-server"])
    # El auxiliar de ACL utiliza las API nativas por descriptor, sin chmod
    # recursivo ni interpretación de comandos procedentes del navegador.
    helper_environment = dict(os.environ, CGO_ENABLED="1", GOOS="darwin", GOARCH="arm64",
                              MACOSX_DEPLOYMENT_TARGET="13.0")
    run(["go", "build", "-trimpath", "-ldflags", "-s -w", "-o", base / "bin/aidex-folder-helper",
         "./cmd/macos-folder-helper"], env=helper_environment)
    run(["codesign", "--force", "--sign", "-", base / "bin/aidex-folder-helper"])
    documents(base / "docs")
    copy(ROOT / "packaging/macos/profile.sh", base / "profile.sh")
    apps = payload / "Applications/AIDEX"
    for source, name in [("admin.command", "Configurar AIDEX.command"), ("port.command", "Configurar puerto.command")]:
        copy(ROOT / "packaging/macos" / source, apps / name, 0o755)
    macos_uninstaller(apps / "Desinstalar AIDEX.app", work)
    authorization_app = apps / "Autorizar carpeta AIDEX.app"
    run(["osacompile", "-o", authorization_app, ROOT / "packaging/macos/authorize-folder.applescript"])
    info = authorization_app / "Contents/Info.plist"
    details = plistlib.loads(info.read_bytes())
    details.update(CFBundleIdentifier="mx.com.aidex.folder-authorization", CFBundleName="Autorizar carpeta AIDEX",
                   CFBundleDisplayName="Autorizar carpeta AIDEX", CFBundleVersion=VERSION,
                   CFBundleShortVersionString=VERSION, LSMinimumSystemVersion="13.0",
                   CFBundleURLTypes=[{"CFBundleURLName":"Autorización local AIDEX", "CFBundleURLSchemes":["aidex-auth"]}])
    copy(apps / "Desinstalar AIDEX.app/Contents/Resources/aidex.icns", authorization_app / "Contents/Resources/aidex.icns")
    details.update(CFBundleIconFile="aidex.icns")
    details.pop("CFBundleIconName", None)
    info.write_bytes(plistlib.dumps(details))
    run(["codesign", "--force", "--deep", "--sign", "-", authorization_app])
    run(["codesign", "--verify", "--deep", "--strict", authorization_app])
    run(["go", "build", "-trimpath", "-ldflags", "-s -w", "-o", base / "bin/aidex-folder-picker", "./cmd/aidex-folder-picker"],
        env=dict(os.environ, CGO_ENABLED="0", GOOS="darwin", GOARCH="arm64"))
    run(["codesign", "--force", "--sign", "-", base / "bin/aidex-folder-picker"])
    selection_app = apps / "Seleccionar carpeta AIDEX.app"
    run(["osacompile", "-o", selection_app, ROOT / "packaging/macos/select-folder.applescript"])
    selection_info = selection_app / "Contents/Info.plist"
    selection_details = plistlib.loads(selection_info.read_bytes())
    selection_details.update(CFBundleIdentifier="mx.com.aidex.folder-selection", CFBundleName="Seleccionar carpeta AIDEX",
                            CFBundleVersion=VERSION, CFBundleShortVersionString=VERSION, LSMinimumSystemVersion="13.0",
                            CFBundleIconFile="aidex.icns", CFBundleURLTypes=[{"CFBundleURLName":"Selección local AIDEX", "CFBundleURLSchemes":["aidex-pick"]}])
    selection_info.write_bytes(plistlib.dumps(selection_details))
    copy(apps / "Desinstalar AIDEX.app/Contents/Resources/aidex.icns", selection_app / "Contents/Resources/aidex.icns")
    run(["codesign", "--force", "--deep", "--sign", "-", selection_app])
    run(["codesign", "--verify", "--deep", "--strict", selection_app])
    (apps / "Abrir AIDEX.webloc").write_bytes(plistlib.dumps({"URL": "http://127.0.0.1:18090"}))
    daemon = payload / "Library/LaunchDaemons/app.aidex.plist"
    daemon.parent.mkdir(parents=True, exist_ok=True)
    install = "/Library/Application Support/AIDEX"
    daemon.write_bytes(plistlib.dumps({
        "Label": "app.aidex", "UserName": "_aidex", "GroupName": "_aidex",
        "ProgramArguments": [install+"/bin/aidex-server", "serve", "--config", install+"/data/config.json"],
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
    for name in ["preinstall", "postinstall", "profile.sh", "installer-common.sh"]:
        copy(ROOT / "packaging/macos" / name, scripts / name, 0o755, template=True)
    component = work / "component.pkg"
    run(["pkgbuild", "--root", payload, "--scripts", scripts, "--identifier", "app.aidex",
         "--version", VERSION, "--install-location", "/", "--ownership", "recommended", component])
    resources = work / "resources"
    resources.mkdir()
    welcome = resources / "welcome.html"
    welcome.write_text("<html><head><meta charset=\"utf-8\"></head><body><h1>AIDEX " + VERSION + "</h1><p>Dashboard con gráficas locales y configuración reunida en Ajustes.</p><p>PDF/OCR necesita las herramientas indicadas en Configurar AIDEX. Las vistas previas de Word y Excel necesitan LibreOffice instalado en este equipo. Sin LibreOffice continúan la extracción, búsqueda y descarga de esos formatos.</p><p>Instala LibreOffice desde https://www.libreoffice.org/download/ y comprueba su detección en Ajustes → Vistas previas y caché.</p><p>Esta actualización conserva los datos. No desinstales ni borres la base de datos para actualizar.</p></body></html>", encoding="utf-8")
    distribution = work / "distribution.xml"
    distribution.write_text('''<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
  <title>AIDEX ''' + VERSION + '''</title>
  <welcome file="welcome.html"/>
  <options customize="never" require-scripts="false" hostArchitectures="arm64"/>
  <domains enable_localSystem="true" enable_currentUserHome="false" enable_anywhere="false"/>
  <allowed-os-versions><os-version min="13.0"/></allowed-os-versions>
  <choices-outline><line choice="app.aidex"/></choices-outline>
  <choice id="app.aidex" visible="false"><pkg-ref id="app.aidex"/></choice>
  <pkg-ref id="app.aidex" version="1.0.0">component.pkg</pkg-ref>
</installer-gui-script>
''')
    run(["productbuild", "--distribution", distribution, "--resources", resources, "--package-path", work,
         output / f"AIDEX-{VERSION}-macOS-arm64.pkg"])


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
    binary(payload / "usr/lib/aidex/aidex-server", "linux", architecture)
    run(["go", "build", "-trimpath", "-ldflags", "-s -w", "-o", payload / "usr/lib/aidex/aidex-folder-picker", "./cmd/aidex-folder-picker"],
        env=dict(os.environ, CGO_ENABLED="0", GOOS="linux", GOARCH=architecture))
    copy(ROOT / "packaging/ubuntu/aidex-folder-picker.desktop", payload / "usr/share/applications/aidex-folder-picker.desktop")
    documents(payload / "usr/share/doc/aidex")
    for name in ["prepare-offline.sh", "install-offline.sh"]:
        copy(ROOT / "packaging/ubuntu" / name, payload / "usr/share/doc/aidex/offline" / name, 0o755)
    copy(ROOT / "packaging/ubuntu/aidex.service", payload / "usr/lib/systemd/system/aidex.service")
    copy(ROOT / "packaging/ubuntu/profile.sh", payload / "usr/lib/aidex/profile.sh")
    for action in ["admin", "uninstall"]:
        copy(ROOT / "packaging/ubuntu" / ("aidex-" + action), payload / "usr/sbin" / ("aidex-" + action), 0o755)
    control.mkdir()
    size = sum(path.stat().st_size for path in payload.rglob("*") if path.is_file()) // 1024
    (control / "control").write_text(f"""Package: aidex
Version: {DEB_VERSION}
Section: utils
Priority: optional
Architecture: {architecture}
Maintainer: AIDEX
Installed-Size: {size}
Depends: adduser, util-linux, systemd, python3, poppler-utils, tesseract-ocr, tesseract-ocr-spa, tesseract-ocr-eng, fonts-dejavu-core
Recommends: libreoffice-writer, libreoffice-calc, zenity | kdialog, desktop-file-utils
Description: AIDEX - Aplicación de Indexación Digital de Expedientes
 Servicio local de biblioteca digital. Datos conservados incluso al desinstalar.
 Requiere activar una licencia para habilitar las modificaciones.
""")
    for name in ["preinst", "postinst", "prerm", "postrm"]:
        copy(ROOT / "packaging/ubuntu" / name, control / name, 0o755, template=True)
    with (output / f"aidex_{DEB_VERSION}_{architecture}.deb").open("wb") as stream:
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


def reuse_verified_windows_runtime(prior, destination, docs):
    """Reuse only OCR assets from a previously verified installer payload."""
    prior = prior.resolve()
    records = json.loads((prior / "package-files.json").read_text(encoding="utf-8"))
    hashes = {item["path"]: item["sha256"] for item in records}
    if len(hashes) != len(records):
        raise RuntimeError("Duplicate paths in prior Windows package manifest")
    lock_path = prior / "docs/windows-ocr.lock.json"
    if json.loads(lock_path.read_text(encoding="utf-8")) != json.loads(
        (ROOT / "packaging/windows/ocr.lock.json").read_text(encoding="utf-8")
    ):
        raise RuntimeError("Prior Windows OCR lock differs from current lock")
    selected = [path for path in prior.rglob("*") if path.is_file() and (
        (path.relative_to(prior).parts[0] == "tools"
         and path.relative_to(prior).parts[:2] != ("tools", "postgresql"))
        or path.relative_to(prior).parts[:2] == ("docs", "third-party")
        or path == lock_path
    )]
    for source in selected:
        relative = source.relative_to(prior)
        if source.is_symlink() or source.resolve() != prior / relative:
            raise RuntimeError(f"Unsafe prior Windows runtime path: {relative}")
        expected = hashes.get(relative.as_posix())
        if not expected or hashlib.sha256(source.read_bytes()).hexdigest() != expected:
            raise RuntimeError(f"Prior Windows runtime differs from package manifest: {relative}")
        target = destination / relative.relative_to("tools") if relative.parts[0] == "tools" else docs / relative.relative_to("docs")
        copy(source, target)
    for path in ["Library/bin/pdfinfo.exe", "Library/bin/pdftotext.exe", "Library/bin/pdftoppm.exe",
                 "Library/bin/tesseract.exe", "tessdata/spa.traineddata", "tessdata/eng.traineddata",
                 "tessdata/osd.traineddata", "fonts.conf"]:
        if not (destination / path).is_file():
            raise RuntimeError(f"Missing verified Windows OCR runtime component: {path}")
    if not (docs / "windows-ocr.lock.json").is_file() or not (docs / "third-party").is_dir():
        raise RuntimeError("Prior Windows OCR runtime lacks provenance documentation")
    print(f"Verified {len(selected)} Windows OCR runtime files against prior installer manifest")


def windows_postgresql(archive, destination, docs):
    """Copy the complete EDB PostgreSQL server runtime from its pinned archive."""
    lock = json.loads((ROOT / "packaging/windows/postgresql.lock.json").read_text(encoding="utf-8"))
    if archive.name != lock["archive"]:
        raise RuntimeError("Windows PostgreSQL archive name differs from lock")
    with archive.open("rb") as archive_file:
        digest = hashlib.file_digest(archive_file, "sha256").hexdigest()
    if digest != lock["sha256"]:
        raise RuntimeError("Windows PostgreSQL archive differs from lock")
    required = {"bin/initdb.exe", "bin/pg_ctl.exe", "bin/postgres.exe",
                "lib/unaccent.dll", "share/extension/unaccent.control",
                "share/extension/unaccent--1.1.sql"}
    copied = set()
    with zipfile.ZipFile(archive) as source:
        for member in source.infolist():
            parts = Path(member.filename).parts
            if member.is_dir() or len(parts) < 3 or parts[0] != "pgsql":
                continue
            if parts[1] in ("bin", "lib", "share"):
                relative = Path(*parts[1:])
                if any(part in ("", ".", "..") for part in parts) or relative.is_absolute() or relative.as_posix() in copied:
                    raise RuntimeError(f"Unsafe PostgreSQL archive path: {member.filename}")
                target = destination / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                with source.open(member) as input_file, target.open("wb") as output_file:
                    shutil.copyfileobj(input_file, output_file)
                copied.add(relative.as_posix())
        if not required <= copied:
            raise RuntimeError("PostgreSQL archive is missing required server or unaccent files: "
                               + ", ".join(sorted(required - copied)))
        for name in ("server_license.txt", "commandlinetools_3rd_party_licenses.txt"):
            with source.open("pgsql/" + name) as input_file:
                target = docs / "third-party" / "postgresql" / name
                target.parent.mkdir(parents=True, exist_ok=True)
                with target.open("wb") as output_file:
                    shutil.copyfileobj(input_file, output_file)
    (docs / "postgresql-runtime.json").write_text(json.dumps(lock, indent=2) + "\n", encoding="utf-8")
    print(f"PostgreSQL {lock['version']} runtime: {len(copied)} files from pinned archive")


def check_pe_imports(directory):
    """Detect missing non-system DLLs before shipping; native execution is still required."""
    system = set("kernel32 kernelbase ntdll user32 advapi32 gdi32 shell32 ole32 oleaut32 comdlg32 comctl32 "
                 "ws2_32 crypt32 bcrypt secur32 wldap32 winmm version shlwapi normaliz iphlpapi "
                 "userenv usp10 rpcrt4 psapi powrprof netapi32 mpr dhcpcsvc dnsapi mswsock "
                 "msvcrt imm32 setupapi winspool winhttp wininet shcore uxtheme dwrite d2d1 "
                 "dxgi d3d11 d3d12 dbghelp hid authz wintrust ncrypt cfgmgr32 win32u wsock32 "
                 "msasn1 cabinet msimg32 cryptbase bcryptprimitives credui netutils wkscli dsrole d3dcompiler_47 "
                 "winspool.drv oleacc".split())
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


def windows(output, work, prefix, makensis, postgres_archive, powershell=None, prior_runtime=None):
    if (not prefix and not prior_runtime) or not makensis or not postgres_archive:
        raise RuntimeError("Windows requires OCR runtime, --windows-postgres-archive, and --makensis")
    powershell = powershell or shutil.which("powershell.exe") or shutil.which("pwsh")
    if not powershell:
        raise RuntimeError("Windows packaging requires PowerShell to validate the generated script; use --powershell")
    payload = work / "root"
    copy(ROOT / "packaging/windows/manage.ps1", payload / "manage.ps1", template=True)
    copy(ROOT / "packaging/windows/installer-action.ps1", payload / "installer-action.ps1", template=True)
    run([powershell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
         "-File", ROOT / "scripts/check_windows_script.ps1", "-Path", payload / "manage.ps1"])
    run([powershell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
         "-File", ROOT / "scripts/check_windows_script.ps1", "-Kind", "Transport", "-Path", payload / "installer-action.ps1"])
    # The checked-in .syso resources are reproducible from the existing brand SVG.
    copy(ROOT / "packaging/windows/aidex.ico", payload / "aidex.ico")
    binary(payload / "aidex-server.exe", "windows", "amd64")
    run(["go", "build", "-trimpath", "-ldflags", "-s -w -H windowsgui", "-o", payload / "AIDEX.exe", "./cmd/aidex-launcher"],
        env=dict(os.environ, CGO_ENABLED="0", GOOS="windows", GOARCH="amd64"))
    check_windows_icon(payload / "aidex-server.exe")
    check_windows_icon(payload / "AIDEX.exe", gui=True)
    documents(payload / "docs")
    copy(ROOT / "internal/httpapi/eula-draft.txt", payload / "docs/CLUF-BORRADOR-PENDIENTE.txt")
    copy(ROOT / "docs/PRUEBAS-WINDOWS-1.0.0.md", payload / "docs/PRUEBAS-WINDOWS-1.0.0.md")
    if prior_runtime:
        reuse_verified_windows_runtime(prior_runtime, payload / "tools", payload / "docs")
    else:
        windows_tools(prefix, payload / "tools", payload / "docs")
    check_pe_imports(payload / "tools/Library/bin")
    windows_postgresql(postgres_archive, payload / "tools/postgresql", payload / "docs")
    # The locked conda VC runtime is already redistributed for OCR. PostgreSQL's
    # binaries need the same app-local CRT on machines without a global VC++ setup.
    for name in ("vcruntime140.dll", "vcruntime140_1.dll", "msvcp140.dll"):
        copy(payload / "tools/Library/bin" / name, payload / "tools/postgresql/bin" / name)
    check_pe_imports(payload / "tools/postgresql/bin")
    copy(ROOT / "packaging/windows/admin.cmd", payload / "admin.cmd")
    copy(ROOT / "packaging/windows/server-port.cmd", payload / "server-port.cmd")
    manifest = [{"path": path.relative_to(payload).as_posix(), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}
                for path in sorted(payload.rglob("*")) if path.is_file()]
    (payload / "package-files.json").write_text(json.dumps(manifest), encoding="utf-8")
    define = "/D" if os.name == "nt" else "-D"
    package_version = VERSION
    # Relocated NSIS distributions may retain a compiled-in Homebrew prefix.
    # Use their own stubs/includes, without changing the host installation.
    environment = dict(os.environ)
    executable = Path(makensis).resolve()
    for candidate in [executable.parent, executable.parent.parent / "share/nsis"]:
        if (candidate / "Stubs").is_dir():
            environment.setdefault("NSISDIR", str(candidate))
            break
    run([makensis, f"{define}VERSION={package_version}", f"{define}STAGE={payload}{os.sep}",
         f"{define}OUTPUT={output / ('AIDEX-' + package_version + '-Windows-amd64.exe')}", ROOT / "packaging/windows/installer.nsi"], env=environment)
    copy(payload / "docs/windows-ocr.lock.json", output / "windows-ocr.lock.json")
    copy(ROOT / "packaging/windows/ocr-explicit.txt", output / "windows-ocr-explicit.txt")
    copy(ROOT / "docs/PRUEBAS-WINDOWS-1.0.0.md", output / "PRUEBAS-WINDOWS-1.0.0.md")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=["macos", "windows", "ubuntu", "all"], required=True)
    parser.add_argument("--output", type=Path, default=ROOT / "dist/aidex-1.0.0")
    parser.add_argument("--windows-tools", type=Path)
    parser.add_argument("--windows-runtime-payload", type=Path,
                        help="Extracted prior installer payload; verify OCR files against its SHA manifest and current lock")
    parser.add_argument("--makensis")
    parser.add_argument("--windows-postgres-archive", type=Path,
                        help="Pinned EDB PostgreSQL 18 Windows x64 binary archive")
    parser.add_argument("--powershell", help="PowerShell parser for Windows scripts (prefers Windows PowerShell 5.1 on Windows)")
    parser.add_argument("--ubuntu-arch", choices=["amd64", "arm64", "all"], default="all",
                        help="Ubuntu architectures to build (default: both)")
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    targets = ["macos", "ubuntu", "windows"] if args.target == "all" else [args.target]
    for target in targets:
        # NSIS reads the staged files directly; keep its Windows stage on the
        # output volume and remove it when packaging finishes.
        with tempfile.TemporaryDirectory(prefix="aidex-package-", dir=output if target == "windows" else None) as directory:
            work = Path(directory).resolve()
            if target == "windows":
                windows(output, work, args.windows_tools, args.makensis, args.windows_postgres_archive, args.powershell,
                        args.windows_runtime_payload)
            elif target == "ubuntu":
                architectures = ["amd64", "arm64"] if args.ubuntu_arch == "all" else [args.ubuntu_arch]
                for architecture in architectures:
                    ubuntu(output, work / architecture, architecture)
            else:
                macos(output, work)
    documents(output / "LEEME")
    if "ubuntu" in targets:
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
