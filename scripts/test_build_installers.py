import codecs
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

import build_installers as builder


class WindowsTemplateTests(unittest.TestCase):
    def test_template_accepts_bom_and_bomless_sources_without_duplicating_bom(self):
        text = '#Requires -Version 5.1\nparam([string]$Action)\n# Configuración @VERSION@\n'
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for encoding in ('utf-8', 'utf-8-sig'):
                with self.subTest(encoding=encoding):
                    source, target = root / 'source.ps1', root / 'payload/manage.ps1'
                    source.write_text(text, encoding=encoding)
                    builder.copy(source, target, template=True)
                    expected = codecs.BOM_UTF8 + text.replace('@VERSION@', builder.VERSION).encode('utf-8')
                    self.assertEqual(target.read_bytes(), expected)

    def test_actual_manage_script_preserves_header_and_spanish_text(self):
        with tempfile.TemporaryDirectory() as directory:
            source = builder.ROOT / 'packaging/windows/manage.ps1'
            target = Path(directory) / 'manage.ps1'
            builder.copy(source, target, template=True)
            expected = source.read_text(encoding='utf-8-sig').replace('@VERSION@', builder.VERSION)
            self.assertEqual(target.read_bytes(), codecs.BOM_UTF8 + expected.encode('utf-8'))
            self.assertTrue(target.read_text(encoding='utf-8-sig').startswith('#Requires -Version 5.1\nparam('))
            self.assertIn('Configuración', target.read_text(encoding='utf-8-sig'))

    def test_non_powershell_templates_remain_bomless(self):
        with tempfile.TemporaryDirectory() as directory:
            source, target = Path(directory) / 'source', Path(directory) / 'preinst'
            source.write_text('#!/bin/sh\n# @VERSION@\n', encoding='utf-8')
            builder.copy(source, target, mode=0o755, template=True)
            self.assertTrue(target.read_bytes().startswith(b'#!/bin/sh\n'))
            self.assertEqual(target.stat().st_mode & 0o777, 0o755)


class WindowsParserTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.engine = os.environ.get('AIBID_TEST_POWERSHELL') or shutil.which('powershell.exe') or shutil.which('pwsh')
        if not cls.engine:
            raise unittest.SkipTest('PowerShell is required for parser tests')

    def check_payload(self, transform=lambda content: content):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / 'manage.ps1'
            builder.copy(builder.ROOT / 'packaging/windows/manage.ps1', target, template=True)
            target.write_bytes(transform(target.read_bytes()))
            return subprocess.run([self.engine, '-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass',
                                   '-File', str(builder.ROOT / 'scripts/check_windows_script.ps1'), '-Path', str(target)],
                                  capture_output=True, text=True)

    def test_actual_payload_parses_and_has_installer_parameters(self):
        result = self.check_payload()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('syntax and parameters OK', result.stdout)

    def test_rejects_original_double_bom_regression(self):
        result = self.check_payload(lambda content: codecs.BOM_UTF8 + content)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Duplicate UTF-8 BOM', result.stdout + result.stderr)

    def test_rejects_syntax_errors(self):
        result = self.check_payload(lambda content: content + b'\nif (\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Invalid PowerShell installer payload', result.stdout + result.stderr)

    def test_rejects_missing_parameter_contract(self):
        result = self.check_payload(lambda content: content.replace(b'$PreflightBinary', b'$DifferentParameter'))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Missing installer parameter', result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
