# IDA Headless MCP Server

Headless IDA Pro binary analysis via Model Context Protocol. Go orchestrates multi-session concurrency while Python workers handle IDA operations.

## Quick start

### Install as a Claude Code plugin

```
/plugin install killaragorn/ida-headless-mcp
/ida-init
```

Then `/mcp` to confirm `ida-headless` is connected. `/ida-init` is a bundled slash command that drives the friendly initializer (detect IDA, install idalib, install Python deps, build the Go binary). On a fresh machine you'll need Go 1.21+, Python 3.10+, and IDA Pro 9.0+ / Essential 9.2+ available locally.

### Install as a Codex plugin (marketplace)

```
codex plugin marketplace add killaragorn/ida-headless-mcp
```

This adds the project as a plugin marketplace. After install, finish the one-time setup:

```bash
git clone https://github.com/killaragorn/ida-headless-mcp.git
cd ida-headless-mcp
go run ./cmd/ida-mcp-server init      # detects IDA, installs idalib + Python deps, builds bin/
```

The plugin manifest declares an `ida-headless` stdio MCP server pointing at `./scripts/launch.py`. Codex resolves the relative `cwd` to the cloned plugin directory, so the launcher finds the bundled binary automatically.

### Install for Codex CLI without the marketplace

If you'd rather skip the marketplace and register the server directly:

```bash
git clone https://github.com/killaragorn/ida-headless-mcp.git
cd ida-headless-mcp
go run ./cmd/ida-mcp-server init
codex mcp add ida-headless -- "$(pwd)/bin/ida-mcp-server" --stdio
```

`codex mcp list` should now show `ida-headless`. Print platform-specific snippets at any time with:

```bash
ida-mcp-server print-config claude-desktop
ida-mcp-server print-config claude-code
ida-mcp-server print-config codex
ida-mcp-server print-config codex-add
```

### Use as a standalone HTTP server

`bin/ida-mcp-server` (no flags) listens on `http://localhost:17300/` (Streamable HTTP) and `http://localhost:17300/sse` (SSE). See [Standalone HTTP server](#standalone-http-server) below for client config.

## Architecture

```
┌─────────────────┐
│  MCP Client     │  Claude Desktop, Claude Code, CLI
│  (HTTP/SSE)     │
└────────┬────────┘
         │ http://localhost:17300/
         ▼
┌─────────────────┐
│   Go Server     │  Session registry, worker manager, watchdog
│   (MCP Tools)   │
└────────┬────────┘
         │ Connect RPC over Unix socket
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

1. **IDA Pro 9.0+ or IDA Essential 9.2+**

2. **idalib**: install and activate:

   ```bash
   ./scripts/setup_idalib.sh
   ```

   See [IDA as a Library documentation](https://docs.hex-rays.com/user-guide/idalib).

3. **Go 1.21+** with protoc tools:
   ```bash
   make install-tools
   ```

4. **Python 3.10+** with dependencies:
   ```bash
   pip3 install -r python/requirements.txt
   ```

5. **Optional: [Il2CppDumper](https://github.com/Perfare/Il2CppDumper)** for Unity game analysis

6. **Optional: [unflutter](https://github.com/zboralski/unflutter)** for Flutter/Dart app analysis
   ```bash
   # Install unflutter (provides flutter_meta.json for import_flutter)
   git clone https://github.com/zboralski/unflutter.git
   cd unflutter && make install
   ```

## Installation

```bash
git clone <repo-url>
cd ida-headless-mcp
make setup
```

This runs idalib setup, installs Python dependencies, and builds the server.

For manual setup or troubleshooting:

```bash
./scripts/setup_idalib.sh   # Setup idalib (requires IDA Pro/Essential 9.x)
make install-python         # Install Python dependencies
make build                  # Build Go server
```

## Standalone HTTP server

```bash
./bin/ida-mcp-server
```

Listens on port 17300 (configurable via `config.json`, env, or `--port`):

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

Restart Claude Desktop after editing. For stdio mode, see `ida-mcp-server print-config claude-desktop`.

### Configure Claude Code (manual, without `/plugin install`)

Copy `.claude/settings.json` to `~/.claude/settings.json` to grant access to all IDA MCP tools.

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

Command line flags:

```bash
./bin/ida-mcp-server \
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
make build          # Build Go server
make proto          # Regenerate protobuf
make test           # Run tests + consistency checks
make restart        # Kill, rebuild, restart server
make clean          # Clean build artifacts
```

### Testing

Install test dependencies:
```bash
pip3 install -r requirements-test.txt
```

Run tests:
```bash
make test           # All tests
pytest tests/ -v    # Python tests only
go test ./...       # Go tests only
```

### Interactive Testing

Use MCP Inspector:
```bash
make run            # Start server
make inspector      # Launch inspector at http://localhost:5173
```

### Project Structure

```
ida-headless-mcp/
├── cmd/ida-mcp-server/   # Go MCP server entry point
├── internal/
│   ├── server/           # MCP tool handlers
│   ├── session/          # Session registry
│   └── worker/           # Worker process manager
├── proto/                # Protobuf definitions
├── python/worker/        # Python worker (idalib wrapper)
├── contrib/il2cpp/       # Il2CppDumper helpers (MIT)
└── tests/                # Test suites
```

### Adding New Tools

1. Add RPC to `proto/ida/worker/v1/ida_service.proto`
2. Regenerate: `make proto`
3. Implement in `python/worker/ida_wrapper.py`
4. Add handler in `python/worker/connect_server.py`
5. Register MCP tool in `internal/server/server.go`

## Session Lifecycle

1. Client calls `open_binary(path)`
2. Go creates session in registry (UUID)
3. Go spawns Python worker subprocess
4. Worker creates Unix socket at `/tmp/ida-worker-{id}.sock`
5. Worker opens IDA database with idalib
6. Go creates Connect RPC clients over socket
7. Subsequent tool calls proxy to worker via Connect
8. Watchdog monitors idle time (default: 4 hours)
9. On timeout or `close_binary`: save database, kill worker, cleanup
10. Session metadata persists under `<database_directory>/sessions` for automatic restoration after server restart

## Troubleshooting

**Worker fails to start:**
```bash
python3 -c "import idapro; print('OK')"
```
If this fails, run `./scripts/setup_idalib.sh`

**Socket timeout:**
Check Python worker logs. Worker may have crashed during init.

**Port already in use:**
```bash
lsof -ti:17300 | xargs kill
# or use a different port
./bin/ida-mcp-server --port 17301
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
