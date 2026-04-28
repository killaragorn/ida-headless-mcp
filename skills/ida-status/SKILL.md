---
name: ida-status
description: Report ida-headless-mcp setup status - binary, Python, idalib, MCP connection, active sessions.
---

# ida-status

Diagnose whether the `ida-headless-mcp` plugin is ready.

## When to use

Run this skill when:
- The user asks "is IDA MCP working?", "status", "check setup", or similar.
- An IDA-related tool call fails and you need to localize the failure (binary missing? idalib missing? MCP disconnected?).
- Before the user starts a long analysis task and you want to confirm everything is healthy.

## Checks

Print a one-line-per-check report (✓ / ✗ / ?) for:

1. **Plugin root**: confirm the directory containing `.codex-plugin/plugin.json` exists; print its path.
2. **Binary**: check `bin/ida-mcp-server` (or `.exe` on Windows). Report path, size, mtime. If missing, suggest running the `ida-init` skill.
3. **Binary self-test**: run `<binary> version` and report the output line.
4. **Python**: run `python --version` (fall back to `python3 --version`). Need 3.10+.
5. **idalib**: run `python -c "import idapro; v=idapro.get_library_version(); print(v)"`. If it fails, suggest `ida-init` or an `--ida-path` override.
6. **MCP server registration**: run `codex mcp list` and check whether `ida-headless` appears.
7. **Active analysis sessions**: if the MCP is connected, call the `list_sessions` tool and report the result. Note any session that has been idle for over an hour.

End with a one-line summary: "Ready" if all checks pass, otherwise the highest-priority next step.
