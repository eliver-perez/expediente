"""Inspect AIDEX packages without installing them. Commands have bounded timeouts."""
import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import plistlib
import re
import struct
import subprocess
import tarfile
import tempfile
import xml.etree.ElementTree as ET
import zipfile

import build_installers as builder


def command(args):
    return subprocess.run([str(arg) for arg in args], check=True, capture_output=True, text=True, timeout=90).stdout


def ar(path):
    data = path.read_bytes()
    assert data[:8] == b'!<arch>\n'
    entries, position = {}, 8
    while position < len(data):
        header = data[position:position + 60]
        assert header[58:60] == b'`\n'
        size = int(header[48:58])
        name = header[:16].decode('ascii').strip().rstrip('/')
        assert name not in entries
        entries[name] = data[position + 60:position + 60 + size]
        position += 60 + size + size % 2
    assert position == len(data)
    return entries


def check_docs(directory):
    for name in ['LEEME.md', 'PRODUCCION.md', 'TRANSICION.md', 'THIRD-PARTY.md']:
        assert (directory / name).is_file(), name
    for path in directory.rglob('*.md'):
        if 'third-party' in path.parts:
            continue  # Supplier recipes/licenses retain upstream names and links.
        text = path.read_text(encoding='utf-8')
        assert not re.search(r'AIDEX (?:2\.0|Pruebas|Test|RC)', text)
        assert not re.search(r'a' + r'ibid', text, re.I)
        for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)', text):
            if target.startswith(('https:', 'http:', '#', 'mailto:')):
                continue
            assert (path.parent / target.split('#')[0]).exists(), (path, target)


def check_identity(name, data):
    former = bytes([97, 105, 98, 105, 100])
    assert former not in name.lower().encode(), name
    # Exact pinned supplier data: numeric Unicode conversion tables and Gaelic
    # translations of "alphabet" contain incidental matches, unrelated to branding.
    supplier = {
        'tools/Library/bin/iconv.dll': ('4fe67a10b7455892ac546d79bc0b2ab2e3483135495255ea2c31d2c794c28e38', [536212], []),
        'tools/Library/bin/icudt78.dll': ('18dd8e1881c5d378de868f8f2c2309d95d3f6757ef40868b76657e7a1f061a51', [12216692, 13578872, 13725110], [12407050, 18067836, 18068658, 18068690, 18079510, 18083882, 18085640]),
    }
    if name in supplier:
        digest, positions, wide_positions = supplier[name]
        assert hashlib.sha256(data).hexdigest() == digest, name
        assert [match.start() for match in re.finditer(former, data.lower())] == positions, name
        assert [match.start() for match in re.finditer(former.decode().encode('utf-16le'), data.lower())] == wide_positions, name
    else:
        assert former not in data.lower(), name
        assert former.decode().encode('utf-16le') not in data.lower(), name
    if data.startswith(b'PK') and zipfile.is_zipfile(io.BytesIO(data)):
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            for entry in archive.namelist():
                check_identity(name + ':' + entry, archive.read(entry))


def check_backend_identity(data):
    assert b'https://licensing.aidex.com.mx' in data
    assert b'X8U4PxM_kr3U56k27D6lVkKIEbLUUTAkJs_-jbXdwJM' in data
    assert b'lic-prod-2026-a' in data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('output', type=Path)
    parser.add_argument('--sevenzip', required=True)
    parser.add_argument('--powershell', required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    results = []
    with tempfile.TemporaryDirectory(prefix='AIDEX José María Muñoz 東京-', dir='/private/tmp') as temporary:
        root = Path(temporary)
        for architecture, machine in [('amd64', 62), ('arm64', 183)]:
            entries = ar(output / f'aidex_1.0.0_{architecture}.deb')
            assert entries['debian-binary'] == b'2.0\n'  # Debian format, not product version.
            with tarfile.open(fileobj=io.BytesIO(entries['control.tar.gz'])) as archive:
                control = archive.extractfile('./control').read().decode('utf-8')
                check_identity('control', control.encode())
                assert 'Package: aidex\n' in control and 'Version: 1.0.0\n' in control
                assert f'Architecture: {architecture}\n' in control
                assert 'Replaces:' not in control
                assert 'Conflicts:' not in control
                assert 'poppler-utils' in control and 'tesseract-ocr-spa' in control
                for name in ['preinst', 'postinst', 'prerm', 'postrm']:
                    text = archive.extractfile('./' + name).read().decode('utf-8')
                    check_identity(name, text.encode())
                    assert '@VERSION@' not in text and '@PROFILE@' not in text
                    path = root / name
                    path.write_text(text)
                    command(['sh', '-n', path])
                postinst = archive.extractfile('./postinst').read().decode()
                assert postinst.index(' migrate --config ') < postinst.index('package-version')
            with tarfile.open(fileobj=io.BytesIO(entries['data.tar.gz'])) as archive:
                for member in archive:
                    path = PurePosixPath(member.name)
                    assert not path.is_absolute() and '..' not in path.parts
                    assert member.uid == member.gid == 0 and (member.isfile() or member.isdir())
                    assert path.parts[0] == 'usr' and 'aidex-test' not in member.name
                    assert not member.name.endswith(('.db', 'installation.json', 'config.json'))
                    if member.isfile():
                        check_identity(member.name, archive.extractfile(member).read())
                binary = archive.extractfile('./usr/lib/aidex/aidex-server').read()
                check_backend_identity(binary)
                assert binary[:6] == b'\x7fELF\x02\x01' and struct.unpack_from('<H', binary, 18)[0] == machine
                assert b'1.0.0' in binary and b'0014_observability' in binary
                unit = archive.extractfile('./usr/lib/systemd/system/aidex.service').read().decode()
                assert 'User=aidex\n' in unit and '/usr/lib/aidex/aidex-server serve' in unit
                for name in ['profile.sh', 'aidex-admin', 'aidex-uninstall']:
                    package_path = './usr/lib/aidex/' + name if name == 'profile.sh' else './usr/sbin/' + name
                    source = builder.ROOT / 'packaging/ubuntu' / name
                    assert archive.extractfile(package_path).read() == source.read_bytes()
                docs = root / ('docs-' + architecture)
                for member in archive:
                    prefix = './usr/share/doc/aidex/'
                    if member.isfile() and member.name.startswith(prefix):
                        path = docs / member.name[len(prefix):]
                        path.parent.mkdir(parents=True, exist_ok=True)
                        path.write_bytes(archive.extractfile(member).read())
                check_docs(docs)
            results.append({'target': 'ubuntu-' + architecture, 'result': 'PASS', 'scope': 'control, ELF, ownership, templates, unit, current wrappers and documentation'})

        windows = output / 'AIDEX-1.0.0-Windows-amd64.exe'
        win = root / 'windows'
        command([args.sevenzip, 'x', '-y', '-o' + str(win), windows])
        installer = windows.read_bytes()
        assert 'FileVersion'.encode('utf-16le') in installer and '1.0.0'.encode('utf-16le') in installer
        for name, kind in [('manage.ps1', 'Manage'), ('installer-action.ps1', 'Transport')]:
            scripts = list(win.rglob(name))
            assert len(scripts) == 2, (name, scripts)
            for path in scripts:
                assert path.read_bytes().startswith(b'\xef\xbb\xbf')
                command([args.powershell, '-NoProfile', '-NonInteractive', '-File', builder.ROOT / 'scripts/check_windows_script.ps1', '-Path', path, '-Kind', kind])
                template = (builder.ROOT / 'packaging/windows' / name).read_text(encoding='utf-8-sig').replace('@VERSION@', builder.VERSION)
                assert path.read_text(encoding='utf-8-sig') == template
        for path in win.rglob('*'):
            if path.is_file():
                check_identity(path.relative_to(win).as_posix(), path.read_bytes())
        manifest = next(win.rglob('package-files.json'))
        check_backend_identity((manifest.parent / 'aidex-server.exe').read_bytes())
        for item in json.loads(manifest.read_text()):
            path = manifest.parent / item['path']
            assert path.resolve().is_relative_to(manifest.parent.resolve())
            assert hashlib.sha256(path.read_bytes()).hexdigest() == item['sha256']
        builder.check_windows_icon(manifest.parent / 'aidex-server.exe')
        builder.check_windows_icon(manifest.parent / 'AIDEX.exe', gui=True)
        builder.check_pe_imports(manifest.parent / 'tools/Library/bin')
        assert not (manifest.parent / 'gestor-documental.exe').exists()
        check_docs(manifest.parent / 'docs')
        results.append({'target': 'windows-amd64', 'result': 'PASS', 'scope': 'installer metadata, UTF-8 BOM/parser/template, both scripts, complete SHA manifest, icons, DLL closure, current documentation'})

        mac = root / 'macos'
        command(['pkgutil', '--expand-full', output / 'AIDEX-1.0.0-macOS-arm64.pkg', mac])
        for path in mac.rglob('*'):
            if path.is_file():
                check_identity(path.relative_to(mac).as_posix(), path.read_bytes())
        distribution = ET.parse(mac / 'Distribution').getroot()
        assert distribution.find('pkg-ref').attrib['id'] == 'app.aidex'
        binary = next(mac.rglob('Payload/Library/Application Support/AIDEX/bin/aidex-server'))
        check_backend_identity(binary.read_bytes())
        payload = binary.parents[4]
        base = binary.parent.parent
        for entry in (base / 'package-files.tsv').read_text().splitlines():
            digest, name = entry.split('\t')
            path = payload / name.lstrip('/')
            assert path.resolve().is_relative_to(payload.resolve())
            assert hashlib.sha256(path.read_bytes()).hexdigest() == digest
        plist = next(mac.rglob('app.aidex.plist'))
        info = plistlib.loads(plist.read_bytes())
        assert info['Label'] == 'app.aidex' and info['UserName'] == '_aidex'
        assert info['ProgramArguments'][0].endswith('/AIDEX/bin/aidex-server')
        # Exercise the real plist editor used by postinstall on a synthetic Unicode path.
        cfg = root / 'María López/Documentación Muñoz/配置.json'
        cfg.parent.mkdir(parents=True)
        command(['plutil', '-replace', 'ProgramArguments.3', '-string', cfg, plist])
        assert plistlib.loads(plist.read_bytes())['ProgramArguments'][3] == str(cfg)
        command(['codesign', '--verify', '--strict', binary])
        assert command([binary, 'version']).startswith('AIDEX 1.0.0 (production;')
        check_docs(base / 'docs')
        results.append({'target': 'macos-arm64', 'result': 'PASS', 'scope': 'distribution/plist, ownership, complete SHA manifest, UTF-8 docs, native plutil Unicode, ad-hoc signature, packaged binary version'})
    print(json.dumps({'result': 'PASS', 'checks': results, 'native_installation': 'not performed'}, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
