#!/usr/bin/env python3
"""
Cross-platform launcher for ida-mcp-server.

Picks the prebuilt binary under ``bin/`` that matches the current OS and
architecture and execs it with the supplied arguments. The repository ships
with binaries for windows/amd64, linux/amd64, linux/arm64, darwin/amd64, and
darwin/arm64. If the matching binary isn't present, the launcher exits with a
clear error - it never falls back to PATH lookup or attempts to build.
"""

from __future__ import annotations

import os
import platform
import sys
from pathlib import Path


def repo_root() -> Path:
    """Return the plugin/repository root.

    When loaded by Claude Code, ``CLAUDE_PLUGIN_ROOT`` points at the plugin
    directory. Otherwise fall back to the script's parent's parent.
    """
    plugin_root = os.environ.get("CLAUDE_PLUGIN_ROOT")
    if plugin_root:
        return Path(plugin_root)
    return Path(__file__).resolve().parent.parent


def detect_platform() -> tuple[str, str]:
    """Return (goos, goarch) tuple matching Go's GOOS/GOARCH naming."""
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
    root = repo_root()
    binary = binary_path(root)

    if not binary.is_file():
        goos, goarch = detect_platform()
        print(
            f"[ida-headless-mcp] Prebuilt binary not found: {binary}\n"
            f"  Detected platform: {goos}/{goarch}\n"
            f"  Available binaries are committed under {root / 'bin'}.\n"
            f"  If your platform isn't shipped, build from source:\n"
            f"    cd \"{root}\" && go build -o \"{binary}\" ./cmd/ida-mcp-server",
            file=sys.stderr,
        )
        return 1

    # Restore +x if the bundled binary lost its mode during clone or download.
    if os.name != "nt" and not os.access(str(binary), os.X_OK):
        try:
            binary.chmod(binary.stat().st_mode | 0o111)
        except OSError as exc:
            print(
                f"[ida-headless-mcp] cannot make {binary} executable: {exc}",
                file=sys.stderr,
            )
            return 1

    args = [str(binary), *sys.argv[1:]]
    os.execv(str(binary), args)
    return 0  # unreachable


if __name__ == "__main__":
    sys.exit(main())
