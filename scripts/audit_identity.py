#!/usr/bin/env python3
"""Audit owned identity, frozen research, embedded Office fixtures and resource names."""
import argparse
import hashlib
import io
import json
from pathlib import Path
import re
import zipfile

ROOT = Path(__file__).resolve().parents[1]
EXCLUDED = {'.git', 'node_modules', 'dist', 'build', '.cache', '__pycache__', 'test-results', 'playwright-report'}
FORMER = bytes([97, 105, 98, 105, 100])
PATTERN = re.compile(FORMER.decode(), re.I)


def classification(path):
    if path.parts[0] == 'documentacion' or path.name == 'documentacion.zip':
        return 'inventarios_preservados'
    if path.parts[0] == 'docs' and path.name not in {'PRODUCCION.md', 'TRANSICION-PRODUCCION.md', 'AIDEX-PRODUCCION.md'}:
        return 'historia'
    if path.as_posix().startswith('internal/httpapi/assets/'):
        return 'generado'
    return 'activo'


def audit():
    items, names = [], []
    for path in sorted(ROOT.rglob('*')):
        relative = path.relative_to(ROOT)
        if any(part in EXCLUDED for part in relative.parts):
            continue
        category = classification(relative)
        if PATTERN.search(str(relative)):
            names.append({'path': str(relative), 'category': category})
        if not path.is_file() or path.is_symlink() or path.stat().st_size > 4_000_000:
            continue
        data = path.read_bytes()
        if zipfile.is_zipfile(io.BytesIO(data)):
            with zipfile.ZipFile(io.BytesIO(data)) as archive:
                for name in archive.namelist():
                    matches = list(re.finditer(FORMER, archive.read(name), re.I))
                    if matches:
                        items.append({'path': str(relative), 'entry': name, 'category': category, 'occurrences': len(matches)})
        else:
            try:
                text = data.decode('utf-8-sig')
            except UnicodeError:
                if category in {'activo', 'generado'}:
                    assert FORMER not in data.lower() and FORMER.decode().encode('utf-16le') not in data.lower(), relative
                continue
            matches = list(PATTERN.finditer(text))
            if matches:
                items.append({'path': str(relative), 'category': category, 'occurrences': len(matches)})
    return {'items': items, 'names': names, 'content_occurrences': sum(i['occurrences'] for i in items), 'matched_files': len({i['path'] for i in items})}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    result = audit()
    result['scope'] = 'Owned source/docs/resources, including synthetic Office ZIP entries; dependency caches and historical build copies excluded.'
    result['active_remaining'] = sum(i['occurrences'] for i in result['items'] if i['category'] in {'activo', 'generado'})
    if args.baseline:
        baseline = json.loads(args.baseline.read_text())
        result['initial_occurrences'] = baseline['content_occurrences']
        result['initial_matched_files'] = baseline['matched_files']
        if 'protected_archive_sha256' in baseline:
            assert hashlib.sha256((ROOT / 'documentacion.zip').read_bytes()).hexdigest() == baseline['protected_archive_sha256']
            result['frozen_inventory_archive_unchanged'] = True
        result['corrected_initial_occurrences'] = sum(i['occurrences'] for i in baseline['items'] + baseline.get('archive_items', []) if i['category'] in {'activo', 'generado'})
        result['initial_paths'] = len(baseline['names'])
        result['corrected_paths'] = sum(i['category'] == 'activo' for i in baseline['names'])
        for name, digest in baseline['protected'].items():
            assert hashlib.sha256((ROOT / name).read_bytes()).hexdigest() == digest, name
        result['frozen_inventory_files_unchanged'] = len(baseline['protected'])
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding='utf-8')
    assert result['active_remaining'] == 0, 'Obsolete active identity remains'
    assert all(i['category'] not in {'activo', 'generado'} for i in result['names']), 'Obsolete active path remains'
    print(json.dumps({k: v for k, v in result.items() if k not in {'items', 'names'}}, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
