---
description: Report ida-headless-mcp setup status - binary, Python, IDA, plugin manifest, MCP server connection.
---

Quickly diagnose whether the `ida-headless-mcp` plugin is ready to serve MCP tools.

Check and report each:

1. **Plugin root**: confirm `${CLAUDE_PLUGIN_ROOT}` is set, or locate the plugin root containing `.claude-plugin/plugin.json`; print the path used as `$ROOT`.
2. **Launcher and bundled binary**: run `python "$ROOT/scripts/launch.py" version` (fallback `python3`). This verifies the selected `bin/ida-mcp-server-<os>-<arch>[.exe]` binary.
3. **Python**: run `python --version` (fallback `python3 --version`). Confirm 3.10+.
4. **idalib**: run `python -c "import idapro; v=idapro.get_library_version(); print(v)"` using the same Python command that worked above. If the import fails, suggest `/ida-init` or `--ida-path` override.
5. **MCP connection**: run `/mcp` so the user can see whether the `ida-headless` server is connected. If you cannot run `/mcp` directly, instruct the user to run it.
6. **Active sessions**: if MCP is connected, call `list_sessions` and report the result.

Output a concise status table: one line per check with `PASS`, `FAIL`, or `UNKNOWN`. Include a short next-step hint after any failed check. Do not run `init` automatically - only suggest it.
