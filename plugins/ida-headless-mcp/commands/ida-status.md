---
description: "Diagnose ida-headless-mcp setup status. Use when MCP tools fail, the plugin shows 'failed', or before starting analysis. NEVER diagnose via shell commands like Get-Process or netstat."
---

Diagnose whether the `ida-headless-mcp` plugin is ready to serve MCP tools.

**NEVER diagnose by running Get-Process, netstat, Start-Process, or executing the server binary directly. The server is managed by the MCP client. Use this command for diagnosis and /ida-init for fixes.**

Check and report each:

1. **Plugin root**: confirm `${CLAUDE_PLUGIN_ROOT}` is set, or locate the plugin root containing `.claude-plugin/plugin.json`; print the path used as `$ROOT`.
2. **Launcher and bundled binary**: run `python "$ROOT/scripts/launch.py" version` (fallback `python3`). This verifies the selected `bin/ida-mcp-server-<os>-<arch>[.exe]` binary.
3. **Python**: run `python --version` (fallback `python3 --version`). Confirm 3.10+.
4. **idalib**: run `python -c "import idapro; v=idapro.get_library_version(); print(v)"` using the same Python command. If the import fails, suggest `/ida-init` or `--ida-path` override.
5. **MCP connection**: instruct the user to run `/mcp` to check whether `ida-headless` is connected.
6. **Active sessions**: if MCP is connected, call `list_sessions` and report the result.

Output a concise status table: one line per check with `PASS`, `FAIL`, or `UNKNOWN`. Include a short next-step hint after any failed check.

End with: "Ready" if all pass, otherwise "Run `/ida-init` to fix."
