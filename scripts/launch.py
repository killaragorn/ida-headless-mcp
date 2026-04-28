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
import platform
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


def binary_candidates() -> list[str]:
    """Return ordered list of binary names to look for under bin/."""
    goos, goarch = detect_platform()
    suffix = ".exe" if goos == "windows" else ""
    # Prefer the platform-specific prebuilt binary, then fall back to a
    # generic name (which `make build` / init produces locally).
    return [
        f"ida-mcp-server-{goos}-{goarch}{suffix}",
        f"ida-mcp-server{suffix}",
    ]


def find_binary(root: Path) -> Path | None:
    bin_dir = root / "bin"
    for name in binary_candidates():
        candidate = bin_dir / name
        if candidate.is_file():
            # On Unix, the bundled binary may have lost +x during clone/copy
            if os.name != "nt" and not os.access(str(candidate), os.X_OK):
                try:
                    candidate.chmod(candidate.stat().st_mode | 0o111)
                except OSError:
                    pass
            return candidate
    # Try PATH as a last resort
    on_path = shutil.which("ida-mcp-server")
    if on_path:
        return Path(on_path)
    return None


def attempt_build(root: Path) -> Path | None:
    if shutil.which("go") is None:
        return None
    name = binary_candidates()[-1]  # generic name (no platform suffix)
    out = root / "bin" / name
    out.parent.mkdir(parents=True, exist_ok=True)
    print(
        f"[ida-headless-mcp] prebuilt binary not found; building via `go build` at {root}",
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
