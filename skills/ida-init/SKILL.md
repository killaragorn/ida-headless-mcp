---
name: ida-init
description: "Use when: ida-headless MCP server shows 'failed' status, any ida-headless MCP tool call fails or returns empty/error, open_binary fails, idalib or idapro is missing, user asks to set up or initialize IDA MCP plugin. This is the ONLY correct fix for plugin startup failures — NEVER start/stop/diagnose the server via shell commands (Start-Process, Get-Process, netstat, etc.)."
---

# ida-init

Initialize the `ida-headless-mcp` plugin so the MCP server can start successfully.

## CRITICAL RULES

- **NEVER** run shell commands to start, stop, or diagnose the MCP server process. No `Start-Process`, `Get-Process`, `netstat`, `Get-NetTCPConnection`, or direct binary execution. The server lifecycle is managed entirely by the MCP client (Claude Code / Codex).
- **NEVER** guess or construct paths to `ida-mcp-server*.exe` or `launch.py` — always use `$ROOT/scripts/launch.py`.
- When the plugin shows `× failed` or any MCP tool returns an error, run THIS skill — not shell commands.

## Steps

1. **Locate `$ROOT`**: Use `${CLAUDE_PLUGIN_ROOT}` if set. Otherwise search for the nearest directory containing `.claude-plugin/plugin.json` or `.codex-plugin/plugin.json`.

2. **Run the initializer**:
   ```
   python "$ROOT/scripts/launch.py" init --skip-build
   ```
   If `python` is unavailable, retry with `python3`.

3. **Forward user flags** verbatim if provided:
   - `--ida-path "/path/to/IDA"` — override IDA auto-detection
   - `--skip-ida` / `--skip-python` / `--skip-build` — skip individual steps
   Always keep `--skip-build` for plugin installs (prebuilt binaries are already shipped).

4. **Stream the output** to the user. Every step prints a checkmark or an actionable hint.

5. **After success**, tell the user to reconnect the MCP server:
   - Claude Code: run `/mcp` and click Reconnect, or restart Claude Code.
   - Codex: start a new session or run `codex mcp list` to confirm.

## Common failures and fixes

| Failure | Fix |
|---|---|
| `IDA installation not found` | Pass `--ida-path "<absolute path>"` or set `IDA_PATH` env var |
| `idalib import fails` | IDA version is too old: need IDA Pro 9.0+ or IDA Essential 9.2+ |
| `Prebuilt binary missing` | This platform is not shipped. Rebuild from source: `cd src && make prebuilt` |
| `pip install fails` | Check network connectivity and Python version (need 3.10+) |

## IDA path examples

- Windows: `C:\Program Files\IDA Pro 9.3` or `F:\Tool\IDA Professional 9.3`
- macOS: `/Applications/IDA Pro 9.3.app/Contents/MacOS`
- Linux: `/opt/idapro-9.3` or `~/idapro-9.3`

Do not silently proceed past errors — surface the actionable hint each failed step prints.
