---
description: "Initialize ida-headless-mcp plugin: detect IDA, activate idalib, install Python deps. Use when the plugin shows 'failed' or MCP tools return errors. NEVER start the server manually via shell commands."
---

Initialize the ida-headless-mcp plugin so the MCP server can start and serve analysis tools.

**NEVER start, stop, or diagnose the MCP server process via shell commands (Start-Process, Get-Process, netstat, Bash, etc.). The server lifecycle is managed by the MCP client. This command is the ONLY correct fix path.**

Steps:

1. Determine `$ROOT`. Prefer `${CLAUDE_PLUGIN_ROOT}`; otherwise locate the directory containing `.claude-plugin/plugin.json` or `.codex-plugin/plugin.json`.
2. Run `python "$ROOT/scripts/launch.py" init --skip-build`. If `python` is unavailable, retry with `python3`.
3. Forward user flags verbatim (e.g. `--ida-path "C:\Program Files\IDA Pro 9.3"`, `--skip-ida`, `--skip-python`). Always keep `--skip-build`.
4. Stream the output. Every step prints a checkmark or actionable hint.
5. After success, tell the user to reconnect the MCP server via `/mcp` or restart Claude Code.

If IDA detection fails, suggest:

- Windows: `--ida-path "C:\Program Files\IDA Pro 9.X"` or `--ida-path "F:\Tool\IDA Professional 9.X"`
- macOS: `--ida-path "/Applications/IDA Pro 9.X.app/Contents/MacOS"`
- Linux: `--ida-path "/opt/idapro-9.X"`

Or set the `IDA_PATH` environment variable and restart.

Do not silently proceed past errors — surface the actionable hint each failed step prints.
