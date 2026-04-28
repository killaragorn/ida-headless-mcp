---
description: Report ida-headless-mcp setup status — binary, Python, IDA, plugin manifest, MCP server connection.
---

Quickly diagnose whether the `ida-headless-mcp` plugin is ready to serve MCP tools.

Check and report each:

1. **Plugin root**: confirm `${CLAUDE_PLUGIN_ROOT}` is set; print its value.
2. **Binary**: does `${CLAUDE_PLUGIN_ROOT}/bin/ida-mcp-server` (or `.exe` on Windows) exist? Print its size and modification time. If missing, suggest `/ida-init`.
3. **Binary self-test**: run `<binary> version` and report the output.
4. **Python**: run `python --version` (fall back to `python3 --version`). Confirm 3.10+.
5. **idalib**: run `python -c "import idapro; v=idapro.get_library_version(); print(v)"`. If the import fails, suggest `/ida-init` or `--ida-path` override.
6. **MCP connection**: run `/mcp` (the Claude Code built-in slash command) so the user can see whether the `ida-headless` server is connected. If you cannot run `/mcp` directly, instruct the user to run it.
7. **Active sessions**: if MCP is connected, call the `list_sessions` tool and report the result.

Output a concise status table — one line per check with ✓ / ✗ / ?. Include a short next-step hint after any failed check. Do not run `init` automatically — only suggest it.
