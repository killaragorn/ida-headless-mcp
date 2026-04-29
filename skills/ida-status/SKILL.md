---
name: ida-status
description: Use when checking whether ida-headless-mcp is ready, an IDA MCP tool fails, or setup needs localization across binary, Python, idalib, and MCP registration.
---

# ida-status

Diagnose whether `ida-headless-mcp` is ready for Claude Code or Codex.

## When to use

Run this skill when:
- The user asks "is IDA MCP working?", "status", "check setup", or similar.
- An IDA-related tool call fails and you need to localize the failure (binary missing? idalib missing? MCP disconnected?).
- Before the user starts a long analysis task and you want to confirm everything is healthy.

## Checks

Print a one-line-per-check report using `PASS`, `FAIL`, or `UNKNOWN` for:

1. **Plugin root**: find the directory containing `.codex-plugin/plugin.json` or `.claude-plugin/plugin.json`; print its path.
2. **Launcher and bundled binary**: run `python scripts/launch.py version` from the root (fallback `python3`). This verifies the launcher and selected `bin/ida-mcp-server-<os>-<arch>[.exe]` binary together.
3. **Python**: run `python --version` (fallback `python3 --version`). Need 3.10+.
4. **idalib**: run `python -c "import idapro; v=idapro.get_library_version(); print(v)"` with the same Python command. If it fails, suggest `ida-init` or `--ida-path`.
5. **MCP registration/connection**:
   - Codex: run `codex mcp list` and check whether `ida-headless` appears.
   - Claude Code: ask the user to run `/mcp` if the command cannot be invoked directly.
6. **Active sessions**: if the MCP is connected, call `list_sessions` and note any session idle for over an hour.

End with a one-line summary: "Ready" if all checks pass, otherwise the highest-priority next step.
