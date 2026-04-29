#!/usr/bin/env python3
"""
Self-contained plugin launcher for ida-headless-mcp.

The plugin package owns its prebuilt binaries and Python worker runtime. It
does not search for or depend on the source checkout at runtime.
"""

from __future__ import annotations

import os
import platform
import sys
from pathlib import Path


def plugin_root() -> Path:
    return Path(__file__).resolve().parent.parent


def detect_platform() -> tuple[str, str]:
    system = sys.platform
    if system.startswith("win"):
        goos = "windows"
    elif system == "darwin":
        goos = "darwin"
    elif system.startswith("linux"):
        goos = "linux"
    else:
        goos = system

    machine = platform.machine().lower()
    if machine in ("x86_64", "amd64"):
        goarch = "amd64"
    elif machine in ("aarch64", "arm64"):
        goarch = "arm64"
    elif machine in ("i386", "i686", "x86"):
        goarch = "386"
    else:
        goarch = machine

    return goos, goarch


def binary_path(root: Path) -> Path:
    goos, goarch = detect_platform()
    suffix = ".exe" if goos == "windows" else ""
    return root / "bin" / f"ida-mcp-server-{goos}-{goarch}{suffix}"


def main() -> int:
    root = plugin_root()
    binary = binary_path(root)

    if not binary.is_file():
        goos, goarch = detect_platform()
        print(
            f"[ida-headless-mcp] plugin binary not found: {binary}\n"
            f"  Detected platform: {goos}/{goarch}\n"
            f"  Rebuild plugin binaries from the source checkout with:\n"
            f"    cd src && make prebuilt",
            file=sys.stderr,
        )
        return 1

    if os.name != "nt" and not os.access(str(binary), os.X_OK):
        try:
            binary.chmod(binary.stat().st_mode | 0o111)
        except OSError as exc:
            print(
                f"[ida-headless-mcp] cannot make {binary} executable: {exc}",
                file=sys.stderr,
            )
            return 1

    os.chdir(root)
    os.execv(str(binary), [str(binary), *sys.argv[1:]])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
