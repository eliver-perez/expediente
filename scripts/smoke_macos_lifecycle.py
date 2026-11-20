#!/usr/bin/env python3
"""Exercise the delivered PKG with real PostgreSQL in a disposable directory.

Package scripts, binaries and HTTP are real. Only the system service, account
and receipt operations are substituted so the installed AIDEX is untouched.
"""
import argparse
import hashlib
import json
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import tempfile
import urllib.request

from test_macos_install_scripts import InstallScriptsTest, MOCK

SYSTEM_ENV = {"PATH": "/usr/bin:/bin:/usr/sbin:/sbin",
              "AIDEX_POSTGRES_BIN": "/nonexistent", "LANG": "en_US.UTF-8"}


def command(*args, success=True, env=SYSTEM_ENV, input=None):
    result = subprocess.run(args, env=env, input=input, capture_output=True, text=True, timeout=90)
    if success is not None and (result.returncode == 0) != success:
        raise RuntimeError(result.stdout + result.stderr)
    return result.stdout + result.stderr


def replace(source, pairs):
    for old, new in pairs:
        assert old in source, old
        source = source.replace(old, new)
    return source


def free_port():
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        return reservation.getsockname()[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("package", type=Path)
    args = parser.parse_args()
    fixture = InstallScriptsTest()
    fixture.setUp()
    binary = fixture.base / "bin/aidex-server"
    control = fixture.base / "control"
    state = fixture.data / "state"
    runtime = binary.parent / "tools/postgresql/bin"
    descriptor = state / "postgresql/installation.json"
    app_path = "Applications/AIDEX/Centro de Control de AIDEX.app"
    uninstaller_path = "Applications/AIDEX/Desinstalar AIDEX.app"
    try:
        with tempfile.TemporaryDirectory(prefix="aidex-package-smoke-", dir="/private/tmp") as temporary:
            expanded = Path(temporary) / "expanded"
            command("/usr/sbin/pkgutil", "--expand-full", args.package.resolve(), expanded)
            packaged_server = next(expanded.rglob("Payload/Library/Application Support/AIDEX/bin/aidex-server"))
            payload = packaged_server.parents[4]
            fixture.prepare_scripts(packaged_server.parents[5] / "Scripts")
            records = [line.split("\t", 1) for line in (packaged_server.parent.parent / "package-files.tsv").read_text().splitlines()]
            for digest, path in records:
                assert hashlib.sha256((payload / path.lstrip("/")).read_bytes()).hexdigest() == digest
            for app in [app_path, uninstaller_path]:
                command("/usr/bin/codesign", "--verify", "--strict", payload / app)
            pg = packaged_server.parent / "tools/postgresql/bin/postgres"
            assert "18.7" in command(pg, "--version")
            assert command("/usr/bin/lipo", "-archs", pg).strip() == "arm64"

            shutil.rmtree(fixture.base)
            fixture.change(users={}, groups={}, loaded=False, admin=False, native_backend=True, receipt=False)
            pkgutil = fixture.mock / "pkgutil"
            pkgutil.write_text("#!" + sys.executable + "\n" + MOCK)
            pkgutil.chmod(0o755)

            def install():
                fixture.run_script("preinstall")
                shutil.copytree(payload, fixture.root, dirs_exist_ok=True)
                fixture.manifest.write_text("".join(f"{digest}\t{fixture.root / path.lstrip('/')}\n" for digest, path in records))
                fixture.run_script("postinstall")
                fixture.change(receipt=True)

            def helper(*args, success=True):
                common = binary.parent / "installer-common.sh"
                common.write_text(replace((payload / common.relative_to(fixture.root)).read_text(),
                                          [(old, new) for old, new in fixture.substitutions.items()
                                           if old in ("/bin/launchctl", "/usr/bin/curl", "/bin/sleep", "/usr/bin/sudo", "/usr/sbin/lsof")]))
                source = (payload / binary.parent.relative_to(fixture.root) / "aidex-manage").read_text()
                source = replace(source, [
                    ("BASE='/Library/Application Support/AIDEX'", f"BASE='{fixture.base}'"),
                    ('. "$BASE/profile.sh"', f'. "{fixture.scripts}/profile.sh"'),
                    ("$(/usr/bin/id -u)", "0"),
                    ("/usr/bin/sudo", fixture.substitutions["/usr/bin/sudo"]),
                ])
                script = fixture.root / "manage-fixture.sh"
                script.write_text(source)
                try:
                    return command("/bin/bash", script, *args, success=success)
                finally:
                    # The installed payload inventory still matches.
                    shutil.copyfile(payload / common.relative_to(fixture.root), common)

            def uninstall(action):
                source = (fixture.root / uninstaller_path / "Contents/Resources/uninstall-helper.sh").read_text()
                source = replace(source, [
                    ("CODE='/Library/Application Support/AIDEX'", f"CODE='{fixture.base}'"),
                    (f"APP='/{uninstaller_path}'", f"APP='{fixture.root / uninstaller_path}'"),
                    ("APPS='/Applications/AIDEX'", f"APPS='{fixture.root}/Applications/AIDEX'"),
                    ("PLIST='/Library/LaunchDaemons/app.aidex.plist'", f"PLIST='{fixture.plist}'"),
                    ("$(/usr/bin/id -u)", "0"),
                    ("/bin/launchctl", fixture.substitutions["/bin/launchctl"]),
                    ("/usr/bin/sudo", fixture.substitutions["/usr/bin/sudo"]),
                    ("/usr/sbin/pkgutil", '"'+str(pkgutil)+'"'),
                ])
                script = fixture.root / "uninstall-fixture.sh"
                script.write_text(source)
                result = command("/bin/sh", script, action)
                assert not fixture.state()["loaded"] and not fixture.state()["receipt"]
                assert not fixture.plist.exists() and not runtime.exists()
                assert not (fixture.root / app_path).exists() and not control.exists()
                return result

            def status(name):
                return json.loads((control / (name + ".json")).read_text())

            def sql(query):
                settings = json.loads(descriptor.read_text())
                return command(runtime / "psql", "-XAt", "-v", "ON_ERROR_STOP=1", "-h", "127.0.0.1",
                               "-p", str(settings["port"]), "-U", "aidex_app", "-d", "aidex", "-c",
                               "SET search_path TO aidex, public;" + query,
                               env=dict(SYSTEM_ENV, PGPASSWORD=settings["password"])).strip()

            def assert_db_running(running):
                port = status("database")["port"]
                probe = subprocess.run([runtime / "pg_isready", "-h", "127.0.0.1", "-p", str(port), "-q"],
                                       env=SYSTEM_ENV, timeout=5)
                assert probe.returncode == (0 if running else 2), probe.returncode

            def login(password, port):
                origin = f"http://127.0.0.1:{port}"
                opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
                request = urllib.request.Request(origin + "/api/v1/auth/login",
                    data=json.dumps({"username": "qa-macos", "password": password}).encode(),
                    headers={"Content-Type": "application/json", "Origin": origin})
                with opener.open(request, timeout=10) as response:
                    assert response.status == 200

            install()
            assert not fixture.state()["loaded"] and status("database")["setup"] == "required"
            assert status("endpoint")["port"] == 18090
            assert set(status("database")) == {"port", "setup"}
            assert (state / "postgresql/data/PG_VERSION").read_text().strip() == "18"
            assert descriptor.stat().st_mode & 0o777 == 0o600
            assert (state / "postgresql/data").stat().st_mode & 0o777 == 0o700
            assert "scram-sha-256" in (state / "postgresql/data/pg_hba.conf").read_text()
            assert_db_running(False)
            print("PASS paquete ARM64, inventario, clúster privado y configuración inicial sin Homebrew en PATH", flush=True)

            port = free_port()
            helper("port", str(port))
            assert status("endpoint")["port"] == port and not fixture.state()["loaded"]
            password = secrets.token_urlsafe(18) + "-á'💡"
            credentials = json.dumps({"username": "qa-macos", "display_name": "Prueba José",
                                      "password": password, "confirmation": password})
            command(binary, "bootstrap-ui", "--config", fixture.config, input=credentials)
            assert status("database")["setup"] == "configured"
            assert "ya fue creado" in command(binary, "bootstrap-ui", "--config", fixture.config,
                                               input=credentials, success=False)
            helper("start")
            assert_db_running(True)
            login(password, port)
            assert sql("SELECT unaccent('áéíóú'), count(*) FROM users;").endswith("aeiou|1")
            helper("restart")
            login(password, port)
            helper("stop")
            assert_db_running(False)
            assert "Configuración: OK" in helper("doctor", success=None)
            helper("start")
            with socket.socket() as occupied:
                occupied.bind(("127.0.0.1", 0))
                occupied.listen()
                assert "ocupado" in helper("port", str(occupied.getsockname()[1]), success=False)
            login(password, port)
            port = free_port()
            helper("port", str(port))
            assert status("endpoint")["port"] == port
            login(password, port)
            print("PASS administrador único, autenticación, start/stop/restart, diagnóstico y cambio de puerto", flush=True)

            config_bytes = fixture.config.read_bytes()
            secret_bytes = descriptor.read_bytes()
            license_bytes = (state / "license/installation.json").read_bytes()
            user = sql("SELECT id, password_hash FROM users WHERE username='qa-macos';")
            external = fixture.root / "biblioteca externa/original.pdf"
            upload = state / "uploads/carga.pdf"
            registered_root = fixture.data / "logs"
            registered = registered_root / "service.log.123"
            for original in (external, upload, registered):
                original.parent.mkdir(parents=True, exist_ok=True)
                original.write_bytes(b"original de prueba")
            literal = str(registered_root).replace("'", "''")
            sql("INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('qa','Sintética','hybrid','2026','2026');"
                "INSERT INTO storage_roots(id,library_id,storage_source,canonical_path,comparison_key,case_sensitive,status,watch_mode,reconcile_interval_seconds,created_at) "
                f"VALUES('root','qa','managed','{literal}','{literal}',1,'active','polling',900,'2026');")
            install()
            login(password, port)
            assert sql("SELECT id,password_hash FROM users WHERE username='qa-macos';") == user
            uninstall("conservar")
            assert fixture.config.read_bytes() == config_bytes
            assert descriptor.read_bytes() == secret_bytes
            assert (state / "license/installation.json").read_bytes() == license_bytes
            install()
            login(password, port)
            assert sql("SELECT count(*) FROM libraries;").endswith("1")
            print("PASS actualización sobre PostgreSQL y reinstalación con configuración, usuario, biblioteca y licencia conservados", flush=True)

            assert registered.is_file() and upload.is_file()
            assert "Eliminación completa de datos internos verificada" in uninstall("eliminar")
            assert not fixture.data.exists() and not descriptor.exists()
            archive = fixture.base / "Documentos conservados"
            for original in (upload, registered):
                archived = list(archive.rglob(original.name))
                assert len(archived) == 1 and archived[0].read_bytes() == b"original de prueba", original.name
            assert not list(archive.rglob("PG_VERSION")) and not list(archive.rglob("installation.json"))
            assert external.read_bytes() == b"original de prueba"
            install()
            assert status("database")["setup"] == "required"
            assert descriptor.read_bytes() != secret_bytes
            uninstall("eliminar")
            print("PASS borrado completo del clúster, protección de originales y reinstalación limpia", flush=True)
    finally:
        if fixture.state().get("native_pid"):
            command(fixture.mock / "launchctl", "bootout", "system/app.aidex")
        fixture.doCleanups()


if __name__ == "__main__":
    main()
