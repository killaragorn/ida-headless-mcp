#!/usr/bin/env python3
"""
Cross-platform launcher for ida-mcp-server.

Resolves the bundled Go binary based on OS, builds it on first run if missing
(when Go is available), and execs it with the supplied arguments. Used by the
Claude Code plugin manifest so a single `command` entry works on every OS.

Search order:
  1. ${CLAUDE_PLUGIN_ROOT}/bin/ida-mcp-server[.exe]  (set by Claude Code)
  2. <repo-root>/bin/ida-mcp-server[.exe]
  3. ida-mcp-server[.exe] on PATH

If none found and Go is on PATH, the launcher attempts `go build` once.
"""

from __future__ import annotations

import os
import shutil
import subprocess
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


def binary_name() -> str:
    return "ida-mcp-server.exe" if os.name == "nt" else "ida-mcp-server"


def find_binary(root: Path) -> Path | None:
    candidate = root / "bin" / binary_name()
    if candidate.is_file():
        return candidate
    # Try PATH as a last resort
    on_path = shutil.which(binary_name())
    if on_path:
        return Path(on_path)
    return None


def attempt_build(root: Path) -> Path | None:
    if shutil.which("go") is None:
        return None
    out = root / "bin" / binary_name()
    out.parent.mkdir(parents=True, exist_ok=True)
    print(
        f"[ida-headless-mcp] binary not found; building via `go build` at {root}",
        file=sys.stderr,
    )
    try:
        subprocess.run(
            ["go", "build", "-o", str(out), "./cmd/ida-mcp-server"],
            cwd=str(root),
            check=True,
        )
    except subprocess.CalledProcessError as exc:
        print(f"[ida-headless-mcp] go build failed: {exc}", file=sys.stderr)
        return None
    return out if out.is_file() else None


def main() -> int:
    root = repo_root()
    binary = find_binary(root)
    if binary is None:
        binary = attempt_build(root)
    if binary is None:
        print(
            "[ida-headless-mcp] Server binary not found. Run `ida-mcp-server init` "
            "or `make build` from the project root to build it.",
            file=sys.stderr,
        )
        return 1

    args = [str(binary), *sys.argv[1:]]
    if os.name == "nt":
        # On Windows, os.execv replaces the process and Claude Code's stdio
        # pipes hand off cleanly; subprocess handoff loses them.
        os.execv(str(binary), args)
    else:
        os.execv(str(binary), args)
    return 0  # unreachable


if __name__ == "__main__":
    sys.exit(main())
