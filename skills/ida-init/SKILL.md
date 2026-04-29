---
name: ida-init
description: Use when ida-headless-mcp needs first-run setup, IDA or idapro is missing, idalib activation fails, or a Claude Code/Codex plugin install needs initialization.
---

# ida-init

Initialize `ida-headless-mcp` for Claude Code or Codex without assuming a locally built binary.

## When to use

Run this skill when:
- The user just installed the plugin and `ida-headless` MCP server fails to start.
- The user asks "how do I set up IDA MCP", "init", or similar.
- A previous tool call returned an error mentioning a missing binary, missing `idapro` Python module, or missing IDA installation.

## Steps

1. Locate the plugin root. Prefer `CLAUDE_PLUGIN_ROOT` when set; otherwise find the directory containing `.codex-plugin/plugin.json` or `.claude-plugin/plugin.json`.
2. From `$ROOT`, run the cross-platform launcher:
   - `python scripts/launch.py init --skip-build`
   - If `python` is unavailable, retry with `python3`.
3. Stream the initializer output. It reports Python, IDA detection, idalib activation, and Python dependency status.
4. Forward user-supplied flags verbatim. Useful flags:
   - `--ida-path "/path/to/IDA"` to override IDA detection
   - `--skip-ida`, `--skip-python`, `--skip-build` to skip individual steps
5. After success, restart or refresh the MCP connection:
   - Claude Code: run `/mcp` or restart Claude Code if it does not reconnect.
   - Codex: start a new Codex session or run `codex mcp list` to confirm registration.

## Failure handling

If a step fails, surface the actionable hint the initializer prints. Do not retry blindly. Common failures:
- **No IDA found**: pass `--ida-path "<absolute path to IDA install dir>"` or set `IDA_PATH` env var.
- **Prebuilt binary missing**: the installed plugin does not include this platform. Rebuild from the source checkout with `cd src && make prebuilt`, or install a plugin build that ships the platform.
- **idalib import fails**: the IDA install is older than 9.0 (Pro) or 9.2 (Essential), or activation didn't run.

## Reference

- macOS IDA path example: `/Applications/IDA Pro 9.3.app/Contents/MacOS`
- Windows IDA path example: `C:\Program Files\IDA Pro 9.2`
- Linux IDA path example: `/opt/idapro-9.3` or `~/idapro-9.3`
