#!/usr/bin/env python3
"""Pruebas aisladas del flujo de desinstalación; nunca acceden a /Library real."""
import hashlib
from pathlib import Path
import subprocess
import tempfile
import unittest

SOURCE = Path(__file__).resolve().parents[1] / "packaging/macos/uninstall-helper.sh"


class UninstallerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="aidex-uninstall-fixture-", dir="/private/tmp")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.code = self.root / "Library/Application Support/AIDEX"
        self.apps = self.root / "Applications/AIDEX"
        self.app = self.apps / "Desinstalar AIDEX.app"
        self.plist = self.root / "Library/LaunchDaemons/app.aidex.plist"
        self.config = self.code / "data/config.json"
        self.database = self.code / "data/documental.db"
        self.original = self.root / "bibliotecas/administrada/original.pdf"
        self.upload = self.code / "data/uploads/original.pdf"
        self.other = self.root / "Library/LaunchDaemons/app.aibid.plist"
        self.receipt = self.root / "receipt"
        self.running = self.root / "running"
        for path, contents in [(self.config, "configuración"), (self.database, "base de datos"),
                               (self.original, "documento original"), (self.upload, "carga"),
                               (self.other, "servicio AIBID"), (self.plist, "servicio AIDEX"),
                               (self.apps / "Abrir AIDEX.webloc", "acceso"),
                               (self.app / "Contents/Resources/uninstall-helper.sh", "auxiliar")]:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents)
        self.binary = self.code / "bin/aidex-server"
        self.binary.parent.mkdir(parents=True)
        self.binary.write_text('#!/bin/sh\ncase "$1" in check-state) test -f "$3";; erase-macos-installation) base=$(dirname "$3"); [ -f "$3" ] || exit 1; mv "$base/uploads" "$base/../Documentos conservados"; rm -f "$3" "$base/documental.db"; rmdir "$base";; esac\n')
        self.binary.chmod(0o755)
        self.folder_helper=self.code/'bin/aidex-folder-helper'
        self.folder_helper.write_text('#!/bin/sh\n[ "$#" -eq 1 ] && [ "$1" = preserve-archive ] || exit 1\nprintf "autorizado\\n" > "'+str(self.root/'finder-access')+'"\n')
        self.folder_helper.chmod(0o755)
        self.picker=self.code/'bin/aidex-folder-picker'
        self.picker.write_text('selector de prueba')
        self.selection=self.apps/'Seleccionar carpeta AIDEX.app/Contents/MacOS/applet'
        self.selection.parent.mkdir(parents=True)
        self.selection.write_text('aplicación selectora de prueba')
        self.receipt.write_text("installed")
        self.running.write_text("running")
        self.fake = self.root / "fake"
        self.fake.mkdir()
        receipt_files = [self.binary, self.folder_helper, self.picker, self.selection, self.plist, self.apps / "Abrir AIDEX.webloc",
                         self.app / "Contents/Resources/uninstall-helper.sh", self.code / "package-files.tsv"]
        receipt_lines = " ".join('"' + str(path).lstrip("/") + '"' for path in receipt_files)
        for name, body in {
            "launchctl": 'case "$1" in print) test -f "' + str(self.running) + '";; bootout) rm -f "' + str(self.running) + '";; esac',
            "pkgutil": 'case "$1" in --pkg-info) test -f "' + str(self.receipt) + '";; --files) printf "%s\\n" ' + receipt_lines + ';; --forget) rm -f "' + str(self.receipt) + '";; esac',
            "sudo": 'if [ "$1" = -u ]; then shift 2; fi; exec "$@"',
        }.items():
            path = self.fake / name
            path.write_text("#!/bin/sh\n" + body + "\n")
            path.chmod(0o755)
        self.manifest = self.code / "package-files.tsv"
        self.write_manifest()
        source = SOURCE.read_text()
        for old, new in [("CODE='/Library/Application Support/AIDEX'", f"CODE='{self.code}'"),
                         ("APP='/Applications/AIDEX/Desinstalar AIDEX.app'", f"APP='{self.app}'"),
                         ("APPS='/Applications/AIDEX'", f"APPS='{self.apps}'"),
                         ("PLIST='/Library/LaunchDaemons/app.aidex.plist'", f"PLIST='{self.plist}'"),
                         ('$(/usr/bin/id -u)', '0'),
                         ('/bin/launchctl', str(self.fake / 'launchctl')),
                         ('/usr/sbin/pkgutil', str(self.fake / 'pkgutil')),
                         ('/usr/bin/sudo', str(self.fake / 'sudo'))]:
            assert old in source, old
            source = source.replace(old, new)
        self.script = self.root / "uninstall.sh"
        self.script.write_text(source)
        self.script.chmod(0o755)

    def write_manifest(self):
        files = [self.binary, self.folder_helper, self.picker, self.selection, self.plist, self.apps / "Abrir AIDEX.webloc",
                 self.app / "Contents/Resources/uninstall-helper.sh"]
        self.manifest.write_text("".join(hashlib.sha256(path.read_bytes()).hexdigest() + "\t" + str(path) + "\n" for path in files))

    def run_uninstall(self, action):
        return subprocess.run([str(self.script), action], text=True, capture_output=True, timeout=15)

    def assert_originals(self):
        self.assertEqual(self.original.read_text(), "documento original")
        upload = self.upload if self.upload.exists() else self.code / 'Documentos conservados/original.pdf'
        self.assertEqual(upload.read_text(), "carga")
        self.assertEqual(self.other.read_text(), "servicio AIBID")

    def test_keep_and_reinstall(self):
        result = self.run_uninstall("conservar")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(self.config.exists())
        self.assertTrue(self.database.exists())
        self.assertFalse(self.binary.exists())
        self.assertFalse(self.picker.exists())
        self.assertFalse(self.selection.exists())
        self.assertFalse(self.plist.exists())
        self.assertFalse(self.running.exists())
        self.assertFalse(self.receipt.exists())
        self.assert_originals()
        self.binary.parent.mkdir(parents=True, exist_ok=True)
        self.binary.write_text("reinstalled")
        self.assertEqual(self.config.read_text(), "configuración")
        self.assertEqual(self.database.read_text(), "base de datos")

    def test_erase_only_internal(self):
        result = self.run_uninstall("eliminar")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(self.config.exists())
        self.assertFalse(self.database.exists())
        self.assertFalse((self.code / 'data').exists())
        self.assertTrue((self.root/'finder-access').exists())
        self.assert_originals()

    def test_erase_cannot_report_success_with_residues(self):
        self.binary.write_text('#!/bin/sh\necho "éxito simulado sin eliminar nada"\n')
        result = self.run_uninstall('eliminar')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('residuos', result.stderr)
        self.assertTrue(self.config.exists())
        self.assertTrue(self.binary.exists())
        self.assertTrue(self.receipt.exists())
        self.assert_originals()

    def test_erase_without_configuration_preserves_recoverable_data(self):
        self.config.unlink()
        result = self.run_uninstall('eliminar')
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.database.exists())
        self.assertTrue(self.binary.exists())
        self.assertTrue(self.receipt.exists())
        self.assert_originals()

    def test_stopped_service(self):
        self.running.unlink()
        result = self.run_uninstall("conservar")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_originals()

    def test_partial_without_manifest_uses_receipt(self):
        self.manifest.unlink()
        result = self.run_uninstall("conservar")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(self.binary.exists())
        self.assertFalse(self.picker.exists())
        self.assertFalse(self.selection.exists())
        self.assertFalse(self.plist.exists())
        self.assertTrue(self.config.exists())
        self.assert_originals()

    def test_partial_with_missing_binary_refuses_erase(self):
        self.binary.unlink()
        result = self.run_uninstall("eliminar")
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.config.exists())
        self.assertTrue(self.plist.exists())
        self.assert_originals()

    def test_symlink_manifest_refused(self):
        self.manifest.unlink()
        self.manifest.symlink_to(self.config)
        result = self.run_uninstall("conservar")
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.plist.exists())
        self.assert_originals()

    def test_invalid_action_refused(self):
        result = self.run_uninstall("everything")
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.plist.exists())

    def test_modified_component_retained(self):
        self.plist.write_text("archivo modificado")
        result = self.run_uninstall("conservar")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Desinstalación parcial", result.stdout)
        self.assertEqual(self.plist.read_text(), "archivo modificado")
        self.assert_originals()

    def test_foreign_manifest_path_refused_before_stopping_service(self):
        with self.manifest.open("a") as output:
            output.write(hashlib.sha256(self.other.read_bytes()).hexdigest() + "\t" + str(self.other) + "\n")
        result = self.run_uninstall("conservar")
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.running.exists())
        self.assertTrue(self.binary.exists())
        self.assert_originals()


if __name__ == "__main__":
    unittest.main()
