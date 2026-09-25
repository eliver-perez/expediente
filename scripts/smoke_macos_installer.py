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
    args = parser.parse_args()
    package = args.package.resolve()
    with tempfile.TemporaryDirectory(prefix="aibid-native-smoke-", dir="/private/tmp") as temporary:
        root = Path(temporary)
        command(["pkgutil", "--expand-full", package, root / "expanded"])
        matches = list((root / "expanded").rglob("Payload/Library/Application Support/AIBID-Test/bin/gestor-documental"))
        if len(matches) != 1:
            raise RuntimeError("Unexpected package payload")
        binary = matches[0]
        print(command([binary, "version"]))
        command(["codesign", "--verify", "--strict", binary])
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            port = reservation.getsockname()[1]
        configuration = root / "private/config.json"
        command([binary, "init", "--config", configuration, "--listen", f"127.0.0.1:{port}", "--tools", "/opt/homebrew/bin"])
        previous = configuration.read_bytes()
        command([binary, "init", "--config", configuration], success=False)
        assert previous == configuration.read_bytes()
        print(command([binary, "doctor", "--config", configuration, "--sample-directory", binary.parent.parent / "docs/pdf"]))
        command([binary, "serve", "--config", configuration], success=False)
        bootstrap(binary, configuration)
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
                    with urllib.request.urlopen(f"http://127.0.0.1:{port}/", timeout=5) as response:
                        assert b"AIBID" in response.read()
                finally:
                    service.terminate()
                    service.wait(timeout=30)
                assert service.returncode == 0
        assert previous == configuration.read_bytes()
        assert configuration.stat().st_mode & 0o777 == 0o600
        state = configuration.parent / "state"
        assert state.stat().st_mode & 0o777 == 0o700
        with sqlite3.connect(str(state / "documental.db")) as database:
            assert database.execute("PRAGMA integrity_check").fetchone()[0] == "ok"
            assert not database.execute("PRAGMA foreign_key_check").fetchall()
            assert database.execute("SELECT count(*) FROM schema_migrations").fetchone()[0] == 5
            assert database.execute("SELECT count(*) FROM bootstrap_state").fetchone()[0] == 1
        print(json.dumps({"package_sha256": hashlib.sha256(package.read_bytes()).hexdigest(),
                          "result": "PASS", "checks": "PDF/OCR, PTY bootstrap with 6 characters, HTTP/login, lock, stop/restart, SQLite integrity, private permissions",
                          "not_tested": "system installer, service account, launchd, native Windows/Ubuntu"}, indent=2))


if __name__ == "__main__":
    main()
