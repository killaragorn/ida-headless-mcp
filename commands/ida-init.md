---
description: Run the ida-headless-mcp friendly initializer — detect IDA, install idalib, install Python deps, build the Go binary.
---

Run the `init` subcommand of the bundled `ida-mcp-server` binary so the plugin can serve MCP tools.

Steps:

1. Determine the binary path. Prefer `${CLAUDE_PLUGIN_ROOT}/bin/ida-mcp-server` (or `.exe` on Windows). If the binary is missing, run `go build -o ${CLAUDE_PLUGIN_ROOT}/bin/ida-mcp-server[.exe] ./cmd/ida-mcp-server` from `${CLAUDE_PLUGIN_ROOT}` first (Go 1.21+ required).
2. Run `<binary> init`. Stream the output to the user — every step (Python check, Go check, IDA detection, idalib activation, pip install, go build) prints a checkmark or an actionable hint.
3. If the user passes flags after `/ida-init` (such as `--ida-path "/Applications/IDA Pro 9.3.app/Contents/MacOS"`, `--skip-ida`, `--skip-python`, or `--skip-build`), forward them verbatim.
4. After init succeeds, restart the `ida-headless` MCP server so the newly built binary is picked up. Tell the user to run `/mcp` to verify the connection, or remind them to restart Claude Code if the server doesn't reconnect automatically.

If detection fails (no IDA found), suggest:

- Set `IDA_PATH` to the IDA install directory, or pass `--ida-path "<path>"`.
- For macOS: `/Applications/IDA Pro 9.X.app/Contents/MacOS`
- For Windows: `C:\Program Files\IDA Pro 9.X`
- For Linux: `/opt/idapro-9.X` or `~/idapro-9.X`

Do not silently proceed past errors — surface the actionable hint each failed step prints.
