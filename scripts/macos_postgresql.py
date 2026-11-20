"""Build a relocatable Apple Silicon runtime from the pinned upstream release."""
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess

VERSION = "18.7"
ARCHIVE = f"postgresql-{VERSION}.tar.bz2"
SOURCE = f"https://ftp.postgresql.org/pub/source/v{VERSION}/{ARCHIVE}"
SHA256 = "441dfae4280cae3e0425e5f876f80559fb0731c140f1aae22b3e5a54d0411bfd"
PREFIX = Path("/aidex-postgresql")


def run(*args, **kwargs):
    return subprocess.run([str(arg) for arg in args], check=True, timeout=600, **kwargs)


def dependencies(binary):
    output = run("/usr/bin/otool", "-L", binary, capture_output=True, text=True).stdout
    return [line.strip().split(" (compatibility version", 1)[0]
            for line in output.splitlines()[1:]]


def relocate(runtime, binaries):
    for binary in binaries:
        changes = []
        for dependency in dependencies(binary):
            if dependency.startswith(("/usr/lib/", "/System/Library/")):
                continue
            if dependency.startswith(f"{PREFIX}/"):
                target = runtime / Path(dependency).relative_to(PREFIX)
                if not target.is_file():
                    raise RuntimeError(f"{binary.name} requires missing library {target.name}")
                relative = os.path.relpath(target, binary.parent)
                if target == binary:
                    changes.extend(["-id", f"@loader_path/{relative}"])
                else:
                    changes.extend(["-change", dependency, f"@loader_path/{relative}"])
            elif not dependency.startswith("@loader_path/"):
                raise RuntimeError(f"{binary.name} requires external library {dependency}")
        if changes:
            run("/usr/bin/install_name_tool", *changes, binary)
        for dependency in dependencies(binary):
            if dependency.startswith("@loader_path/"):
                target = (binary.parent / dependency.removeprefix("@loader_path/")).resolve()
                if not target.is_relative_to(runtime.resolve()) or not target.is_file():
                    raise RuntimeError(f"{binary.name} requires unavailable library {dependency}")
        architecture = run("/usr/bin/lipo", "-archs", binary, capture_output=True, text=True).stdout.strip()
        build = run("/usr/bin/vtool", "-show-build", binary, capture_output=True, text=True).stdout
        minimum = re.search(r"\bminos (\d+(?:\.\d+)*)", build)
        if architecture != "arm64" or not minimum or tuple(map(int, minimum[1].split("."))) > (13, 0, 0):
            raise RuntimeError(f"{binary.name} does not target macOS 13 on Apple Silicon")
        run("/usr/bin/codesign", "--force", "--sign", "-", binary, capture_output=True)


def build_postgresql(archive, runtime, docs, work):
    if archive is None:
        archive = work / ARCHIVE
        run("/usr/bin/curl", "--fail", "--location", "--silent", "--show-error",
            "--output", archive, SOURCE)
    if hashlib.sha256(archive.read_bytes()).hexdigest() != SHA256:
        raise RuntimeError(f"Source archive differs from the pinned PostgreSQL {VERSION} SHA-256")
    run("/usr/bin/tar", "-xf", archive, "-C", work)
    source = work / f"postgresql-{VERSION}"
    stage = work / "postgresql-stage"
    installed = stage / PREFIX.relative_to("/")
    # Build against the Apple SDK alone. AIDEX uses libc C, SCRAM and loopback
    # without TLS; default zlib and the system editline remain available.
    environment = {"PATH": "/usr/bin:/bin:/usr/sbin:/sbin", "LC_ALL": "C", "TMPDIR": str(work),
                   "MACOSX_DEPLOYMENT_TARGET": "13.0", "CC": "/usr/bin/clang",
                   "CFLAGS": "-O2 -arch arm64", "LDFLAGS": "-arch arm64 -Wl,-headerpad_max_install_names"}
    with (work / "postgresql-build.log").open("w+") as log:
        jobs = min(os.cpu_count() or 4, 12)
        try:
            for command in [
                ("./configure", f"--prefix={PREFIX}", "--without-icu", "--with-libedit-preferred"),
                ("/usr/bin/make", f"-j{jobs}", "-s", "all"),
                ("/usr/bin/make", "-s", "install", f"DESTDIR={stage}"),
                ("/usr/bin/make", "-s", "-C", "contrib/unaccent", "install", f"DESTDIR={stage}"),
            ]:
                run(*command, cwd=source, env=environment, stdout=log, stderr=subprocess.STDOUT)
        except (subprocess.CalledProcessError, subprocess.TimeoutExpired):
            log.seek(0, os.SEEK_END)
            log.seek(max(0, log.tell() - 7000))
            print(log.read())
            raise
    binaries = []
    (runtime / "bin").mkdir(parents=True)
    for name in ("initdb", "pg_ctl", "pg_isready", "postgres", "psql"):
        binaries.append(Path(shutil.copy2(installed / "bin" / name, runtime / "bin")))
    for file in (installed / "lib").rglob("*"):
        if file.suffix in (".so", ".dylib") and file.is_file() and not file.is_symlink():
            target = runtime / file.relative_to(installed)
            target.parent.mkdir(parents=True, exist_ok=True)
            binaries.append(Path(shutil.copy2(file, target)))
    shutil.copytree(installed / "share", runtime / "share")
    for pattern in ("lib/**/unaccent.dylib", "share/**/unaccent.control", "share/**/unaccent.rules"):
        if not list(runtime.glob(pattern)):
            raise RuntimeError(f"Incomplete PostgreSQL runtime: missing {pattern}")
    relocate(runtime, binaries)
    license_dir = docs / "third-party" / "postgresql"
    license_dir.mkdir(parents=True)
    shutil.copy2(source / "COPYRIGHT", license_dir / "COPYRIGHT.txt")
    (license_dir / "source.txt").write_text(f"{SOURCE}\nSHA-256 {SHA256}\n", encoding="utf-8")
    print(f"PostgreSQL {VERSION} arm64 packaged with system dependencies only")
