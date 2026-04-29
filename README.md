# IDA Headless MCP Server

Headless IDA Pro binary analysis via Model Context Protocol. Go orchestrates multi-session concurrency while Python workers handle IDA operations.

## Quick start

The easiest path is to install the plugin, let the MCP client start it over stdio, then run the one-time `idalib` setup. This repository root is the plugin root: plugin manifests, skills, commands, launcher, bundled Python runtime, and prebuilt Go binaries live in top-level plugin subdirectories. Go/Python source code and development-only files live under `src/`. Go is only required if you build from source.

### Install as a Claude Code plugin

```text
/plugin marketplace add killaragorn/ida-headless-mcp
/plugin install ida-headless-mcp@ida-headless-mcp
```

Initialize IDA's Python library once per machine:

```text
/ida-init
/ida-status
/mcp
```

`/ida-init` detects IDA, installs the `idapro` Python package, and installs Python worker dependencies. It uses the bundled binary and does not rebuild it by default. Requirements: Python 3.10+ and IDA Pro 9.0+ or IDA Essential 9.2+.

### Install as a Codex plugin (marketplace)

```bash
codex plugin marketplace add killaragorn/ida-headless-mcp
```

Restart or refresh Codex, then enable/install `ida-headless-mcp` from the marketplace entry if Codex prompts for it. The marketplace entry points at the current repository root, whose manifest declares an `ida-headless` stdio MCP server that runs `python ./scripts/launch.py --stdio`.

Finish the one-time `idalib` setup from the plugin directory, or ask Codex to use the bundled `ida-init` skill:

```bash
cd ida-headless-mcp
python scripts/launch.py init --skip-build
```

Check setup with the bundled status skill or manually:

```bash
python scripts/launch.py version
codex mcp list
```

### Install for Codex CLI without the marketplace

If you'd rather skip the marketplace and register the server directly:

```bash
git clone https://github.com/killaragorn/ida-headless-mcp.git
cd ida-headless-mcp
python scripts/launch.py init --skip-build
python scripts/launch.py print-config codex-add
```

Run the printed `codex mcp add ...` command. It points Codex at the plugin package's platform-specific prebuilt binary, for example `bin/ida-mcp-server-windows-amd64.exe` on Windows.

Print client snippets at any time with:

```bash
python scripts/launch.py print-config claude-desktop
python scripts/launch.py print-config claude-code
python scripts/launch.py print-config codex
python scripts/launch.py print-config codex-add
```

### Use as a standalone HTTP server

Run `python scripts/launch.py` with no flags to start the plugin's prebuilt HTTP server. It listens on `http://localhost:17300/` (Streamable HTTP) and `http://localhost:17300/sse` (SSE). See [Standalone HTTP server](#standalone-http-server) below for client config.

## Architecture

```
┌─────────────────┐
│  MCP Client     │  Claude Desktop, Claude Code, Codex CLI
│  (stdio/HTTP)   │
└────────┬────────┘
         │ stdio or http://localhost:17300/
         ▼
┌─────────────────┐
│   Go Server     │  Session registry, worker manager, watchdog
│   (MCP Tools)   │
└────────┬────────┘
         │ Connect RPC over TCP loopback
         ▼
┌─────────────────┐
│ Python Worker   │  IDA + idalib (one per session)
│ (per session)   │
└─────────────────┘
```

**Key features:**
- Multi-session concurrency via process isolation
- 52 MCP tools for binary analysis
- Automatic session timeouts (4 hours default, configurable)
- Paginated results with configurable limit (default 1000)
- [Il2CppDumper](https://github.com/Perfare/Il2CppDumper) metadata import for Unity games
- [unflutter](https://github.com/zboralski/unflutter) metadata import for Flutter/Dart apps

## Prerequisites

For plugin users:

1. **IDA Pro 9.0+ or IDA Essential 9.2+**
2. **Python 3.10+** available as `python` or `python3`
3. Run `ida-init` once to install/activate `idalib` and Python worker dependencies.

For source builds and development, also install **Go 1.21+**. Protobuf tools are only needed when regenerating protobuf code:

   ```bash
   cd src
   make install-tools
   ```

Optional metadata import helpers:

- [Il2CppDumper](https://github.com/Perfare/Il2CppDumper) for Unity game analysis
- [unflutter](https://github.com/zboralski/unflutter) for Flutter/Dart app analysis

```bash
git clone https://github.com/zboralski/unflutter.git
cd unflutter && make install
```

## Source build

```bash
git clone <repo-url>
cd ida-headless-mcp
cd src
make setup
```

This runs idalib setup, installs Python dependencies, and builds a source-tree `src/bin/ida-mcp-server`.

For manual setup or troubleshooting:

```bash
cd src
./scripts/setup_idalib.sh   # Setup idalib (requires IDA Pro/Essential 9.x)
make install-python         # Install Python dependencies
make build                  # Build Go server
```

## Standalone HTTP server

```bash
python scripts/launch.py
```

This starts the plugin package's platform-specific prebuilt binary. Source builds can use `cd src && make build` followed by `python src/scripts/launch.py`. The server listens on port 17300 (configurable via `config.json`, env, or `--port`):

- Streamable HTTP (recommended): `http://localhost:17300/`
- SSE compatibility endpoint: `http://localhost:17300/sse`

### Configure Claude Desktop (HTTP mode)

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) / `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "ida-headless": {
      "url": "http://127.0.0.1:17300/",
      "type": "http"
    }
  }
}
```

Restart Claude Desktop after editing. For stdio mode, run `python scripts/launch.py print-config claude-desktop`.

### Configure Claude Code (manual, without `/plugin install`)

Prefer the plugin install in Quick start. For manual stdio configuration, run:

```bash
python scripts/launch.py print-config claude-code
```

Copy `.claude/settings.json` to `~/.claude/settings.json` only if you want Claude Code to pre-allow all IDA MCP tools.

### Basic Workflow

```
1. open_binary(path="/path/to/binary.so")
   → {"session_id": "abc123", "has_decompiler": true}

2. run_auto_analysis(session_id="abc123")
   → {"completed": true}

3. get_entry_point(session_id="abc123")
   → {"address": 4198400}

4. get_decompiled_func(session_id="abc123", address=4198400)
   → {pseudocode...}

5. get_functions(session_id="abc123")
   → {"functions": [...], "count": 1523}

6. close_binary(session_id="abc123")
   → {"success": true}
```

### Flutter/Dart Import

```
1. Run unflutter on the target: unflutter meta libapp.so
2. open_binary(path="libapp.so")
3. import_flutter(session_id="...", meta_json_path="flutter_meta.json")
   → {"functions_created": 9926, "structs_created": 2090,
      "signatures_applied": 9926, "comments_set": 34172}
4. run_auto_analysis(session_id="...")
```

The `import_flutter` tool reads structured JSON metadata from unflutter. It creates Dart class structs, function definitions with typed signatures, and annotates THR/PP/string reference comments in a single pass.

Use `tools/list` via MCP to see all available tools.

## Configuration

Configuration is optional. The server starts with safe defaults when `config.json` is absent. For plugin runtime overrides, copy the plugin example file and edit the root-local copy:

```bash
cp config/config.example.json config.json
```

`config.json` is intentionally ignored by Git. Precedence is:

1. Built-in defaults
2. `config.json` or `--config <path>`
3. `IDA_MCP_*` environment variables
4. CLI flags

For source-tree local overrides, copy the source example inside `src/`:

```bash
cp src/config.example.json src/config.json
```

Command line flags:

```bash
python scripts/launch.py \
  --port 17300 \
  --max-sessions 10 \
  --session-timeout 4h \
  --worker python/worker/server.py \
  --debug
```

Environment variables (overridden by CLI flags):

```bash
IDA_MCP_PORT=17300
IDA_MCP_SESSION_TIMEOUT_MIN=240
IDA_MCP_MAX_SESSIONS=10
IDA_MCP_WORKER=/custom/worker.py
IDA_MCP_DEBUG=1
```

## Development

### Build

```bash
cd src
make build          # Build Go server under src/bin/
make plugin-sync    # Copy source Python worker runtime into the repository-root plugin package
make prebuilt       # Sync runtime and rebuild root bin/ plugin binaries for all supported platforms
make proto          # Regenerate protobuf
make test           # Run tests + consistency checks
make restart        # Kill, rebuild, restart server
make clean          # Clean build artifacts
```

Run tests:
```bash
cd src
make test           # All tests
go test ./internal/... ./ida/...
```

### Interactive Testing

Use MCP Inspector:
```bash
cd src
make run            # Start server
make inspector      # Launch inspector at http://localhost:5173
```

### Project Structure

```
ida-headless-mcp/
├── .codex-plugin/        # Codex plugin manifest and MCP config
├── .claude-plugin/       # Claude Code plugin manifest
├── .claude/              # Optional Claude Code allow-list settings
├── bin/                  # Committed plugin prebuilt binaries
├── commands/             # Claude slash-command docs
├── config/               # Example plugin runtime config
├── python/worker/        # Bundled plugin Python worker runtime
├── scripts/              # Self-contained plugin launcher
├── skills/               # Codex setup/status skills
└── src/
    ├── cmd/ida-mcp-server/   # Go MCP server entry point
    ├── internal/             # MCP handlers, session store, worker manager
    ├── proto/                # Protobuf definitions
    ├── python/worker/        # Source Python worker
    ├── python/tests/         # Python worker tests
    ├── contrib/il2cpp/       # Il2CppDumper helpers (MIT)
    └── scripts/              # Source-tree utilities and source launcher
```

### Adding New Tools

1. Add RPC to `src/proto/ida/worker/v1/service.proto`
2. Regenerate from `src/`: `make proto`
3. Implement in `src/python/worker/ida_wrapper.py`
4. Add handler in `src/python/worker/connect_server.py`
5. Register MCP tool in `src/internal/server/server.go`

## Session Lifecycle

1. Client calls `open_binary(path)`
2. Go creates a short session ID in the registry
3. Go allocates a loopback TCP port and spawns a Python worker subprocess
4. Worker listens on `127.0.0.1:<dynamic-port>` and writes a ready marker
5. Worker opens IDA database with idalib
6. Go creates Connect RPC clients over the loopback address
7. Subsequent tool calls proxy to worker via Connect
8. Watchdog monitors idle time (default: 4 hours)
9. On timeout or `close_binary`: save database, kill worker, cleanup
10. Session metadata persists under `<database_directory>/sessions` for automatic restoration after server restart

## Troubleshooting

**Worker fails to start:**
```bash
python -c "import idapro; print('OK')"
```
If this fails, run `/ida-init` in Claude Code or `python scripts/launch.py init --skip-build` from the repository root.

**Worker connection timeout:**
Check Python worker logs. The worker may have crashed during IDA startup, failed to import `idapro`, or been blocked from binding a loopback port.

**Port already in use:**
```bash
lsof -ti:17300 | xargs kill
# or use a different port
python scripts/launch.py --port 17301
```

**Session not found:**
Session may have timed out. Use `list_sessions` to check active sessions.

## License

MIT

## Related Projects

**MCP Servers:**
- [LaurieWired/GhidraMCP](https://github.com/LaurieWired/GhidraMCP)
- [mrexodia/ida-pro-mcp](https://github.com/mrexodia/ida-pro-mcp)
- [cnitlrt/headless-ida-mcp-server](https://github.com/cnitlrt/headless-ida-mcp-server)

**Metadata Dumpers:**
- [Perfare/Il2CppDumper](https://github.com/Perfare/Il2CppDumper) (used by `import_il2cpp`)
- [zboralski/unflutter](https://github.com/zboralski/unflutter) (used by `import_flutter`)

## References

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Connect RPC](https://connectrpc.com/)
- [IDA Pro idalib](https://hex-rays.com/products/ida/support/idapython_docs/)
