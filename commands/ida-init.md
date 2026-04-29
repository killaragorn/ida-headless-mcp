---
description: Run the ida-headless-mcp friendly initializer - detect IDA, activate idalib, and install Python deps.
---

Run the bundled launcher so the plugin can initialize IDA/idapro dependencies without requiring a local Go build.

Steps:

1. Determine `$ROOT`. Prefer `${CLAUDE_PLUGIN_ROOT}`; otherwise use the current plugin root if it contains `.claude-plugin/plugin.json` or `.codex-plugin/plugin.json`.
2. Run `python "$ROOT/scripts/launch.py" init --skip-build` for normal plugin installs. If `python` is unavailable, retry with `python3`.
3. If the user passes flags after `/ida-init` (such as `--ida-path "/Applications/IDA Pro 9.3.app/Contents/MacOS"`, `--skip-ida`, `--skip-python`, or `--skip-build`), forward them verbatim. Keep `--skip-build`; plugin packages are runtime artifacts, not source build roots.
4. Stream the output to the user. Every step prints a checkmark or an actionable hint.
5. After init succeeds, restart the `ida-headless` MCP server so the initialized environment is picked up. Tell the user to run `/mcp`, or restart Claude Code if the server does not reconnect automatically.

If detection fails (no IDA found), suggest:

- Set `IDA_PATH` to the IDA install directory, or pass `--ida-path "<path>"`.
- For macOS: `/Applications/IDA Pro 9.X.app/Contents/MacOS`
- For Windows: `C:\Program Files\IDA Pro 9.X`
- For Linux: `/opt/idapro-9.X` or `~/idapro-9.X`

Do not silently proceed past errors - surface the actionable hint each failed step prints.
