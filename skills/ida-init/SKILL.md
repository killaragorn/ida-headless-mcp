---
name: ida-init
description: Run the friendly initializer for ida-headless-mcp - detect IDA, install idalib, install Python deps, build the Go binary.
---

# ida-init

Drive the `ida-mcp-server init` command so the plugin can serve MCP tools.

## When to use

Run this skill when:
- The user just installed the plugin and `ida-headless` MCP server fails to start.
- The user asks "how do I set up IDA MCP", "init", or similar.
- A previous tool call returned an error mentioning a missing binary, missing `idapro` Python module, or missing IDA installation.

## Steps

1. Locate the plugin root (the directory containing `.codex-plugin/plugin.json`). Treat that as `$ROOT`.
2. Resolve the binary path: `$ROOT/bin/ida-mcp-server` on Linux/macOS, `$ROOT/bin/ida-mcp-server.exe` on Windows.
3. If the binary is missing, build it: `cd $ROOT && go build -o bin/ida-mcp-server[.exe] ./cmd/ida-mcp-server` (Go 1.21+ required).
4. Run `<binary> init` from `$ROOT` and stream the output to the user. The initializer prints a 5-step checklist (Python, Go, idalib, Python deps, Go binary) and ends with "Init complete."
5. Forward any user-supplied flags verbatim. Useful flags:
   - `--ida-path "/path/to/IDA"` to override IDA detection
   - `--skip-ida`, `--skip-python`, `--skip-build` to skip individual steps
6. After success, tell the user to restart the MCP connection (in Codex CLI: re-launch the session, or run `codex mcp list` to confirm).

## Failure handling

If a step fails, surface the actionable hint the initializer prints. Do not retry blindly. Common failures:
- **No IDA found**: pass `--ida-path "<absolute path to IDA install dir>"` or set `IDA_PATH` env var.
- **idalib import fails**: the IDA install is older than 9.0 (Pro) or 9.2 (Essential), or activation didn't run.
- **`go: command not found`**: install Go 1.21+, or pass `--skip-build` and build manually elsewhere.

## Reference

- macOS IDA path example: `/Applications/IDA Pro 9.3.app/Contents/MacOS`
- Windows IDA path example: `C:\Program Files\IDA Pro 9.2`
- Linux IDA path example: `/opt/idapro-9.3` or `~/idapro-9.3`
