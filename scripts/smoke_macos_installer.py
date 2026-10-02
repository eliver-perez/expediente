#!/usr/bin/env python3
"""Execute the binary extracted from a .pkg using disposable data, not launchd.

Does not install a service, edit /Library, or use the existing 8090 installation.
Requires macOS arm64, PDF/OCR dependencies, a PTY and a loopback socket.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import pty
import select
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.request


def command(arguments, success=True):
    result = subprocess.run([str(value) for value in arguments], capture_output=True, text=True, timeout=60)
    if success and result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    if not success and not result.returncode:
        raise RuntimeError("Unexpected success: " + str(arguments))
    return result.stdout.strip()


def bootstrap(binary, configuration):
    process, terminal = pty.fork()
    if process == 0:
        os.execl(str(binary), str(binary), "bootstrap", "--config", str(configuration))
    try:
        pending = b""
        deadline = time.monotonic() + 30
        for prompt, answer in [("Usuario:", "installer-test"), ("Nombre visible:", "Instalador de prueba"),
                               ("Nueva contraseña:", "abcdef"), ("Repite la contraseña:", "abcdef")]:
            marker = prompt.encode()
            while marker not in pending:
                if time.monotonic() > deadline:
                    raise RuntimeError("Bootstrap timed out at " + prompt)
                if select.select([terminal], [], [], .2)[0]:
                    pending += os.read(terminal, 4096)
            pending = pending.split(marker, 1)[1]
            os.write(terminal, (answer + "\n").encode())
        while time.monotonic() < deadline:
            completed, status = os.waitpid(process, os.WNOHANG)
            if completed:
                process = None
                if os.waitstatus_to_exitcode(status):
                    raise RuntimeError("Bootstrap failed")
                return
            time.sleep(.05)
        raise RuntimeError("Bootstrap did not finish")
    finally:
        os.close(terminal)
        if process:
            os.kill(process, signal.SIGTERM)
            os.waitpid(process, 0)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("package", type=Path)
    parser.add_argument("--previous-package", type=Path,
                        help="Also migrate disposable administrator data created by the schema-8 package")
    args = parser.parse_args()
    package = args.package.resolve()
    with tempfile.TemporaryDirectory(prefix="aibid-native-smoke-", dir="/private/tmp") as temporary:
        root = Path(temporary)
        command(["pkgutil", "--expand-full", package, root / "expanded"])
        matches = list((root / "expanded").rglob("Payload/Library/Application Support/AIBID-Test/bin/gestor-documental"))
        if len(matches) != 1:
            raise RuntimeError("Unexpected package payload")
        binary = matches[0]
        initialization_binary = binary
        if args.previous_package:
            command(["pkgutil", "--expand-full", args.previous_package.resolve(), root / "previous"])
            previous_matches = list((root / "previous").rglob("Payload/Library/Application Support/AIBID-Test/bin/gestor-documental"))
            if len(previous_matches) != 1:
                raise RuntimeError("Unexpected previous package payload")
            initialization_binary = previous_matches[0]
        payload = binary.parents[4]
        manifest = binary.parent.parent / "package-files.tsv"
        for entry in manifest.read_text().splitlines():
            digest, installed_path = entry.split("\t", 1)
            packaged_path = payload / installed_path.lstrip("/")
            assert packaged_path.resolve().is_relative_to(payload.resolve())
            assert hashlib.sha256(packaged_path.read_bytes()).hexdigest() == digest
        print(command([binary, "version"]))
        command(["codesign", "--verify", "--strict", binary])
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            port = reservation.getsockname()[1]
        configuration = root / "private/config.json"
        command([initialization_binary, "init", "--config", configuration, "--listen", f"127.0.0.1:{port}", "--tools", "/opt/homebrew/bin"])
        previous = configuration.read_bytes()
        command([binary, "init", "--config", configuration], success=False)
        assert previous == configuration.read_bytes()
        print(command([binary, "doctor", "--config", configuration, "--sample-directory", binary.parent.parent / "docs/pdf"]))
        command([initialization_binary, "serve", "--config", configuration], success=False)
        bootstrap(initialization_binary, configuration)
        if args.previous_package:
            state_directory = configuration.parent / "state"
            with sqlite3.connect(str(state_directory / "documental.db")) as database:
                assert database.execute("SELECT count(*) FROM schema_migrations").fetchone()[0] == 8
                original_users = database.execute("SELECT * FROM users ORDER BY id").fetchall()
                original_audit = database.execute("SELECT * FROM audit_events ORDER BY id").fetchall()
            database.close()
            command([binary, "migrate", "--config", configuration])
            assert configuration.read_bytes() == previous
            with sqlite3.connect(str(state_directory / "documental.db")) as database:
                assert database.execute("SELECT count(*) FROM schema_migrations").fetchone()[0] == 9
                assert database.execute("SELECT * FROM users ORDER BY id").fetchall() == original_users
                assert database.execute("SELECT * FROM audit_events ORDER BY id").fetchall() == original_audit
            database.close()
            snapshots = list((state_directory / "upgrade-backups").glob("*.db"))
            assert len(snapshots) == 1
            with sqlite3.connect(str(snapshots[0])) as database:
                assert database.execute("SELECT count(*) FROM schema_migrations").fetchone()[0] == 8
                assert database.execute("SELECT * FROM users ORDER BY id").fetchall() == original_users
                assert database.execute("PRAGMA integrity_check").fetchone()[0] == "ok"
            database.close()
            command([binary, "migrate", "--config", configuration])
            assert list((state_directory / "upgrade-backups").glob("*.db")) == snapshots
        command([binary, "bootstrap-ready", "--config", configuration])
        for iteration in range(2):
            with (root / "service.log").open("a") as log:
                service_environment = dict(os.environ, HOME="", APPDATA="", XDG_CONFIG_HOME="")
                service = subprocess.Popen([str(binary), "serve", "--config", str(configuration)], stdout=log, stderr=log, env=service_environment)
                try:
                    for attempt in range(100):
                        if service.poll() is not None:
                            raise RuntimeError((root / "service.log").read_text())
                        try:
                            with urllib.request.urlopen(f"http://127.0.0.1:{port}/health/live", timeout=1) as response:
                                assert response.status == 200
                            break
                        except OSError:
                            time.sleep(.1)
                    else:
                        raise RuntimeError("Service failed to become ready")
                    command([binary, "migrate", "--config", configuration], success=False)
                    request = urllib.request.Request(f"http://127.0.0.1:{port}/api/v1/auth/login",
                        data=json.dumps({"username": "installer-test", "password": "abcdef"}).encode(),
                        headers={"Content-Type": "application/json", "Origin": f"http://127.0.0.1:{port}"})
                    with urllib.request.urlopen(request, timeout=15) as response:
                        assert response.status == 200
                        session = json.load(response)
                        cookie = response.headers['Set-Cookie'].split(';', 1)[0]
                    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
                    def network_request(body=None):
                        request = urllib.request.Request(f"http://127.0.0.1:{port}/api/v1/system/network",
                            data=json.dumps(body).encode() if body is not None else None,
                            method="PUT" if body is not None else "GET",
                            headers={"Content-Type": "application/json", "Origin": f"http://127.0.0.1:{port}",
                                     "Cookie": cookie, "X-CSRF-Token": session['csrf_token']})
                        with opener.open(request, timeout=15) as response:
                            return json.load(response)
                    diagnostic = network_request()
                    assert diagnostic['mode'] == ('local' if iteration == 0 else 'lan')
                    assert diagnostic['listening']
                    if iteration == 0:
                        diagnostic = network_request({'mode': 'lan', 'revision': diagnostic['revision']})
                    # Exercise the packaged production binary by its actual IPv4,
                    # then restart with the persisted LAN setting before reverting.
                    addresses = [item['url'] for item in diagnostic['interfaces'] if item['suggested']]
                    if not addresses:
                        raise RuntimeError('A connected IPv4 interface is required for this smoke test')
                    with opener.open(addresses[0] + '/health/live', timeout=5) as response:
                        assert response.status == 200
                    if iteration == 1:
                        diagnostic = network_request({'mode': 'local', 'revision': diagnostic['revision']})
                        assert diagnostic['mode'] == 'local'
                        try:
                            opener.open(addresses[0] + '/health/live', timeout=2)
                        except OSError:
                            pass
                        else:
                            raise RuntimeError('LAN still accessible after reverting to local')
                    with urllib.request.urlopen(f"http://127.0.0.1:{port}/", timeout=5) as response:
                        assert b"AIBID" in response.read()
                finally:
                    service.terminate()
                    service.wait(timeout=30)
                assert service.returncode == 0
        current = json.loads(configuration.read_bytes())
        assert current.pop('network_mode') == 'local'
        assert current == json.loads(previous)
        assert configuration.stat().st_mode & 0o777 == 0o600
        state = configuration.parent / "state"
        assert state.stat().st_mode & 0o777 == 0o700
        with sqlite3.connect(str(state / "documental.db")) as database:
            assert database.execute("PRAGMA integrity_check").fetchone()[0] == "ok"
            assert not database.execute("PRAGMA foreign_key_check").fetchall()
            assert database.execute("SELECT count(*) FROM schema_migrations").fetchone()[0] == 9
            assert database.execute("SELECT count(*) FROM bootstrap_state").fetchone()[0] == 1
        database.close()
        original = state / "uploads/keep.pdf"
        original.parent.mkdir(parents=True, exist_ok=True)
        contents = (binary.parent.parent / "docs/pdf/native.pdf").read_bytes()
        original.write_bytes(contents)
        command([binary, "erase-internal-data", "--config", configuration], success=False)
        assert configuration.exists()
        command([binary, "erase-internal-data", "--config", configuration, "--confirm-erase-internal-data"])
        assert original.read_bytes() == contents
        assert not configuration.exists() and not (state / "documental.db").exists()
        command([binary, "init", "--config", configuration, "--listen", f"127.0.0.1:{port}", "--tools", "/opt/homebrew/bin"])
        command([binary, "migrate", "--config", configuration])
        with sqlite3.connect(str(state / "documental.db")) as database:
            assert database.execute("SELECT count(*) FROM bootstrap_state").fetchone()[0] == 0
        database.close()
        assert original.read_bytes() == contents
        print(json.dumps({"package_sha256": hashlib.sha256(package.read_bytes()).hexdigest(),
                          "previous_package_upgrade": "PASS" if args.previous_package else "not requested",
                          "result": "PASS", "checks": "package manifest, PDF/OCR, PTY bootstrap with 6 characters, HTTP/login, local/LAN IPv4 with persisted restart and local-only restoration, lock, stop/restart, SQLite integrity, private permissions, confirmed metadata cleanup preserves PDF, fresh database",
                          "not_tested": "system installer, service account, launchd, native Windows/Ubuntu"}, indent=2))


if __name__ == "__main__":
    main()
