#!/usr/bin/env python3
"""Check source formatting without modifying files; suitable for all CI hosts."""
from pathlib import Path
import subprocess
import sys

root = Path(__file__).resolve().parent.parent
sources = [str(path) for directory in ("cmd", "db", "internal")
           for path in (root / directory).rglob("*.go")]
result = subprocess.run(["gofmt", "-l", *sources], capture_output=True, text=True, check=True)
if result.stdout.strip():
    print("Run gofmt on:\n" + result.stdout)
    sys.exit(1)
print("Go source formatting: OK")
