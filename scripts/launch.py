#!/usr/bin/env python3
"""
Self-contained plugin launcher for ida-headless-mcp.

Bootstraps an isolated venv inside the plugin directory on first run,
installs Python worker dependencies and the IDA Pro `idapro` wheel into
it, then exec's the prebuilt Go server with PATH pointing at the venv so
worker subprocesses use the plugin-private Python (and idalib stays out
of the global environment).

idalib bootstrap is best-effort: if no IDA installation is detected, the
launcher still starts the Go server but logs a warning. The user can
either set IDA_PATH and restart, or run src/scripts/setup_idalib.{ps1,sh}
manually.
"""

from __future__ import annotations

import os
import platform
import subprocess
import sys
import venv
from pathlib import Path


VENV_DIR_NAME = ".venv"
REQS_MARKER = ".bootstrapped"
IDALIB_MARKER = ".idalib_ready"


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


def venv_python(root: Path) -> Path:
    if os.name == "nt":
        return root / VENV_DIR_NAME / "Scripts" / "python.exe"
    return root / VENV_DIR_NAME / "bin" / "python"


def venv_bin_dir(root: Path) -> Path:
    return venv_python(root).parent


def ensure_venv(root: Path) -> Path:
    py = venv_python(root)
    venv_dir = root / VENV_DIR_NAME
    marker = venv_dir / REQS_MARKER

    if not py.is_file():
        print(
            f"[ida-headless-mcp] creating venv at {venv_dir}",
            file=sys.stderr,
        )
        venv.EnvBuilder(with_pip=True, clear=False).create(venv_dir)

    if not marker.is_file():
        req = root / "python" / "requirements.txt"
        if req.is_file():
            print(
                "[ida-headless-mcp] installing worker requirements into venv",
                file=sys.stderr,
            )
            subprocess.check_call(
                [
                    str(py),
                    "-m",
                    "pip",
                    "install",
                    "-q",
                    "--disable-pip-version-check",
                    "-r",
                    str(req),
                ]
            )
        marker.touch()

    return py


def find_ida_install() -> Path | None:
    env_path = os.environ.get("IDA_PATH")
    if env_path:
        candidate = Path(env_path)
        if candidate.is_dir():
            return candidate

    candidates: list[Path] = []
    if sys.platform == "darwin":
        candidates = sorted(
            Path("/Applications").glob("IDA*.app/Contents/MacOS"),
            key=lambda p: p.name,
            reverse=True,
        )
    elif sys.platform.startswith("win"):
        seen: set[Path] = set()
        for pattern in ("IDA Pro*", "IDA Essential*", "IDA*"):
            for p in Path("C:/Program Files").glob(pattern):
                if p.is_dir() and p not in seen:
                    candidates.append(p)
                    seen.add(p)
        candidates.sort(key=lambda p: p.name, reverse=True)
    elif sys.platform.startswith("linux"):
        for base in (Path("/opt"), Path("/usr/local"), Path.home()):
            if base.is_dir():
                candidates.extend(
                    sorted(
                        (p for p in base.glob("ida*") if p.is_dir()),
                        key=lambda p: p.name,
                        reverse=True,
                    )
                )

    for c in candidates:
        if (c / "idalib").is_dir():
            return c
    return None


def ensure_idalib(py: Path, root: Path) -> bool:
    venv_dir = root / VENV_DIR_NAME
    marker = venv_dir / IDALIB_MARKER
    if marker.is_file():
        return True

    # Skip if already importable (e.g. user installed manually)
    probe = subprocess.run(
        [str(py), "-c", "import idapro"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    if probe.returncode == 0:
        marker.touch()
        return True

    ida = find_ida_install()
    if ida is None:
        print(
            "[ida-headless-mcp] IDA installation not found; skipping idalib setup.\n"
            "  Set the IDA_PATH env var to your IDA install dir and restart, e.g.:\n"
            "    Windows : C:\\Program Files\\IDA Pro 9.x\n"
            "    macOS   : /Applications/IDA Pro 9.x.app/Contents/MacOS\n"
            "  Server will start but open_binary will fail until idalib is ready.",
            file=sys.stderr,
        )
        return False

    idalib_dir = ida / "idalib"
    py_dir = idalib_dir / "python"
    if not py_dir.is_dir():
        print(
            f"[ida-headless-mcp] {ida} has no idalib/python; need IDA Pro 9.0+ or IDA Essential 9.2+",
            file=sys.stderr,
        )
        return False

    wheels = sorted(py_dir.glob("*.whl"))
    setup_py = py_dir / "setup.py"
    activate_script = py_dir / "py-activate-idalib.py"

    print(f"[ida-headless-mcp] installing idapro from {ida}", file=sys.stderr)
    try:
        if wheels:
            subprocess.check_call(
                [
                    str(py),
                    "-m",
                    "pip",
                    "install",
                    "--force-reinstall",
                    "--disable-pip-version-check",
                    "-q",
                    str(wheels[0]),
                ]
            )
        elif setup_py.is_file():
            subprocess.check_call(
                [
                    str(py),
                    "-m",
                    "pip",
                    "install",
                    "--disable-pip-version-check",
                    "-q",
                    str(py_dir),
                ]
            )
        else:
            print(
                f"[ida-headless-mcp] no wheel or setup.py under {py_dir}",
                file=sys.stderr,
            )
            return False

        if activate_script.is_file():
            subprocess.check_call(
                [str(py), str(activate_script), "-d", str(ida)],
                stdout=subprocess.DEVNULL,
            )

        subprocess.check_call(
            [str(py), "-c", "import idapro; idapro.get_library_version()"],
            stdout=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError as exc:
        print(
            f"[ida-headless-mcp] idalib bootstrap failed: {exc}.\n"
            "  Server will start but open_binary will fail until idalib is ready.",
            file=sys.stderr,
        )
        return False

    marker.touch()
    print("[ida-headless-mcp] idalib ready", file=sys.stderr)
    return True


def inject_venv_path(root: Path) -> None:
    bin_dir = str(venv_bin_dir(root))
    sep = os.pathsep
    existing = os.environ.get("PATH", "")
    parts = [bin_dir] if bin_dir not in existing.split(sep) else []
    if existing:
        parts.append(existing)
    os.environ["PATH"] = sep.join(parts)
    os.environ["VIRTUAL_ENV"] = str(root / VENV_DIR_NAME)
    os.environ.pop("PYTHONHOME", None)


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

    py = ensure_venv(root)
    ensure_idalib(py, root)
    inject_venv_path(root)

    os.chdir(root)

    if os.name == "nt":
        # On Windows, os.execv is implemented as "spawn child + exit parent",
        # so the originally-spawned python.exe PID dies while a different PID
        # runs the Go binary. Claude Code / Codex track the original PID and
        # report the MCP server as failed. Stay alive as a thin shim instead;
        # stdio handles are inherited by the child via CreateProcess.
        try:
            completed = subprocess.run([str(binary), *sys.argv[1:]])
        except KeyboardInterrupt:
            return 130
        return completed.returncode

    os.execv(str(binary), [str(binary), *sys.argv[1:]])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
