#!/usr/bin/env python3
"""
Source-tree launcher for ida-mcp-server.

Execs the local development binary under ``src/bin/``. The repository root is
the plugin package and has its own launcher plus prebuilt binaries.
"""

from __future__ import annotations

import os
import sys
from pathlib import Path


def repo_root() -> Path:
    return Path(__file__).resolve().parent.parent


def binary_path(root: Path) -> Path:
    suffix = ".exe" if os.name == "nt" else ""
    return root / "bin" / f"ida-mcp-server{suffix}"


def main() -> int:
    root = repo_root()
    binary = binary_path(root)

    if not binary.is_file():
        print(
            f"[ida-headless-mcp] source build binary not found: {binary}\n"
            f"  Build it first:\n"
            f"    cd \"{root}\" && go build -o \"{binary}\" ./cmd/ida-mcp-server\n"
            f"  For plugin prebuilt binaries, use:\n"
            f"    python \"{root.parent / 'scripts' / 'launch.py'}\"",
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
