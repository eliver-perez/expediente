"""Isolated installer/profile tests. Never changes host services or real data."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import build_installers as builder


@unittest.skipIf(os.name == 'nt', 'Unix shell profiles; Windows is covered by PowerShell transition tests')
class ProfileTests(unittest.TestCase):
    def test_admin_uses_the_production_account(self):
        for platform in ['ubuntu', 'macos']:
            for legacy in [False]:
                with self.subTest(platform=platform, legacy=legacy), tempfile.TemporaryDirectory(prefix='José Muñoz-') as directory:
                    root = Path(directory).resolve()
                    account = '_aidex' if platform == 'macos' else 'aidex'
                    data = root / ('Library/Application Support/AIDEX/data' if platform == 'macos' else 'var/lib/aidex')
                    data.mkdir(parents=True)
                    config = data / 'config.json' if platform == 'macos' else root / ('etc/aidex/config.json')
                    config.parent.mkdir(parents=True, exist_ok=True)
                    config.write_text('{}')
                    profile_source = builder.ROOT / 'packaging' / platform / 'profile.sh'
                    profile = root / 'profile.sh'
                    profile.write_text(profile_source.read_text().replace('/bin/launchctl', '"$AIDEX_TEST_MOCK" launchctl').replace('systemctl ', '"$AIDEX_TEST_MOCK" systemctl '))
                    source = builder.ROOT / 'packaging' / platform / ('admin.command' if platform == 'macos' else 'aidex-admin')
                    script = source.read_text().replace(". '/Library/Application Support/AIDEX/profile.sh'", '. "$AIDEX_TEST_PROFILE"').replace('. /usr/lib/aidex/profile.sh', '. "$AIDEX_TEST_PROFILE"').replace('aidex_select_profile\n', 'aidex_select_profile "$AIDEX_TEST_ROOT"\n')
                    mock = root / 'mock.py'
                    mock.write_text('''#!/usr/bin/env python3
import json,os,pathlib,sys
args=sys.argv[1:]; command=args.pop(0)
root=pathlib.Path(os.environ['AIDEX_TEST_ROOT'])
with (root/'calls.jsonl').open('a') as log: log.write(json.dumps([command,*args])+"\\n")
if command=='id': print('0'); sys.exit(0)
if command in ('sudo','runuser'):
 if args[0]=='test': sys.exit(0 if pathlib.Path(args[-1]).exists() else 1)
 if args[0]=='-u':
  assert args[1]==os.environ['AIDEX_TEST_ACCOUNT'],args
  args=args[2:]
  if args[0]=='--':args=args[1:]
 if pathlib.Path(args[0]).name=='touch':pathlib.Path(args[1]).touch()
 if 'bootstrap-ready' in args:sys.exit(1)
if command=='launchctl':
 if args[0]=='print':sys.exit(0 if (root/'running').exists() else 1)
 if args[0]=='bootstrap':(root/'running').touch()
sys.exit(0)
''')
                    mock.chmod(0o755)
                    for command in ['id', 'sudo', 'runuser', 'launchctl', 'systemctl']:
                        if platform == 'macos':
                            script = script.replace('/usr/bin/' + command, '"$AIDEX_TEST_MOCK" ' + command).replace('/bin/' + command, '"$AIDEX_TEST_MOCK" ' + command)
                        else:
                            script = script.replace(command + ' ', '"$AIDEX_TEST_MOCK" ' + command + ' ')
                            script = script.replace('$(id -u)', '$("$AIDEX_TEST_MOCK" id -u)')
                    temporary_script = root / 'admin.sh'
                    temporary_script.write_text(script)
                    environment = dict(os.environ, AIDEX_TEST_ROOT=str(root), AIDEX_TEST_PROFILE=str(profile), AIDEX_TEST_MOCK=str(mock), AIDEX_TEST_ACCOUNT=account)
                    environment.pop('SUDO_USER', None)
                    result = subprocess.run(['sh', str(temporary_script)], capture_output=True, text=True, timeout=10, env=environment)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    self.assertTrue((data / 'service-enabled').is_file())
                    calls = (root / 'calls.jsonl').read_text()
                    self.assertIn('bootstrap', calls)
                    self.assertIn('aidex.service' if platform == 'ubuntu' else 'system/app.aidex', calls)

    def profile(self, platform, root):
        variables = '$ACCOUNT\\n$BASE\\n$BINARY' if platform == 'macos' else '$ACCOUNT\\n$DATA\\n$BINARY'
        return subprocess.run(['sh', '-c', '. "$1"; aidex_select_profile "$2" || exit; printf "' + variables + '\\n"',
                               'profile', str(builder.ROOT / 'packaging' / platform / 'profile.sh'), str(root)],
                              capture_output=True, text=True, timeout=5)

    def test_profile_preserves_state_and_rejects_symlink_unicode_paths(self):
        for platform, production in [
            ('ubuntu', 'etc/aidex/config.json'),
            ('macos', 'Library/Application Support/AIDEX/data/config.json'),
        ]:
            with self.subTest(platform=platform), tempfile.TemporaryDirectory(prefix='José María López Muñoz ü 東京-') as directory:
                root = Path(directory).resolve()
                fresh = self.profile(platform, root)
                self.assertEqual(fresh.returncode, 0, fresh.stderr)
                self.assertIn('aidex-server', fresh.stdout)
                self.assertIn(str(root), fresh.stdout)
                current = root / production
                current.parent.mkdir(parents=True)
                current.write_text('{"identity":"conservar ñ"}')
                self.assertEqual(fresh.stdout, self.profile(platform, root).stdout)
                self.assertEqual(current.read_text(), '{"identity":"conservar ñ"}')
                current.unlink()
                current.symlink_to(root / 'outside.json')
                self.assertNotEqual(self.profile(platform, root).returncode, 0)

    def test_unix_scripts_syntax_and_no_source_bom(self):
        for path in (builder.ROOT / 'packaging').glob('**/*'):
            if path.is_file() and path.read_bytes().startswith(b'#!/bin/'):
                self.assertFalse(path.read_bytes().startswith(b'\xef\xbb\xbf'))
                engine = 'bash' if path.read_bytes().startswith(b'#!/bin/bash') else 'sh'
                subprocess.run([engine, '-n', str(path)], check=True, timeout=5)

    def test_names_and_unicode_transport_contract(self):
        self.assertEqual(builder.VERSION, '1.0.0')
        self.assertEqual(builder.DEB_VERSION, '1.0.0')
        source = (builder.ROOT / 'packaging/windows/installer.nsi').read_text()
        self.assertIn('FileReadUTF16LE', source)
        self.assertNotIn('nsExec::ExecToLog', source)
        self.assertNotIn('Uninstall\\AIDEXTest', source)
        self.assertIn('InstallDir "$PROGRAMFILES64\\AIDEX"', source)
        for platform in ['ubuntu', 'macos']:
            source = (builder.ROOT / 'packaging' / platform / ('preinst' if platform == 'ubuntu' else 'preinstall')).read_text()
            self.assertNotIn(' migrate ', '\n'.join(line for line in source.splitlines() if not line.lstrip().startswith('#')))
            self.assertNotIn('2.0.0-rc.1', source)


if __name__ == '__main__':
    unittest.main()
