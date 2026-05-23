---
name: ida-status
description: "Use when: checking if ida-headless-mcp is ready, any IDA MCP tool call fails or returns empty, the plugin shows 'failed', or user asks about IDA MCP status. Use this for diagnosis INSTEAD OF running shell commands like Get-Process, netstat, or Start-Process."
---

# ida-status

Diagnose whether `ida-headless-mcp` is ready to serve MCP tools.

## CRITICAL RULES

- **NEVER** diagnose the server by running shell commands like `Get-Process`, `netstat`, `Start-Process`, or directly executing the server binary.
- The server lifecycle is managed by the MCP client. Use THIS skill for diagnosis and `/ida-init` for fixing.

## Checks

Print a one-line-per-check report using `PASS`, `FAIL`, or `UNKNOWN`:

1. **Plugin root**: find the directory containing `.claude-plugin/plugin.json` or `.codex-plugin/plugin.json`; print its path as `$ROOT`.
2. **Launcher and bundled binary**: run `python "$ROOT/scripts/launch.py" version` (fallback `python3`). This verifies the launcher and the selected platform binary.
3. **Python**: run `python --version` (fallback `python3 --version`). Need 3.10+.
4. **idalib**: run `python -c "import idapro; v=idapro.get_library_version(); print(v)"`. If it fails, suggest `/ida-init` or `--ida-path`.
5. **MCP connection**: instruct the user to run `/mcp` to check whether `ida-headless` is connected.
6. **Active sessions**: if MCP is connected, call `list_sessions` and report.

## Summary

End with one line:
- All pass → "Ready — you can call open_binary to start analyzing."
- Any fail → the highest-priority next step (usually: "Run `/ida-init` to fix.").
