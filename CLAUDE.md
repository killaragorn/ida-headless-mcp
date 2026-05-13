# CLAUDE.md

项目级指引文件,用于在新会话中快速建立对本仓库的理解。

## 项目身份

**ida-headless-mcp** —— 通过 Model Context Protocol(MCP)对外暴露 IDA Pro 的无头(headless)二进制分析能力。
Go 语言负责对外的 MCP 服务、多会话编排、Worker 生命周期管理与持久化;Python 进程承载真正的 IDA(`idalib`)调用。每个分析会话独立进程,彼此隔离。

- Go module:`github.com/zboralski/ida-headless-mcp`(go 1.25)
- 主要依赖:`connectrpc.com/connect`、`github.com/modelcontextprotocol/go-sdk`、`google.golang.org/protobuf`、`github.com/google/uuid`
- 默认监听端口:`17300`(`/` 为 Streamable HTTP,`/sse` 为 SSE 兼容端点)
- License:MIT
- 当前仓库根目录就是插件根;源码、测试和开发配置集中在 `src/`。

## 架构

```
┌─────────────────────────────┐
│   MCP Client (HTTP/SSE)     │  Claude Desktop / Claude Code / CLI
└──────────────┬──────────────┘
               │  http://localhost:17300/
┌──────────────▼──────────────┐
│   Go Server (src/cmd + src/internal)│
│  - MCP tool handlers         │
│  - Session registry/store    │
│  - Worker manager            │
│  - Watchdog (idle timeout)   │
└──────────────┬──────────────┘
               │  Connect RPC over TCP loopback (127.0.0.1:动态端口)
┌──────────────▼──────────────┐
│   Python Worker (idalib)    │  每个 session 一个进程
│  - server.py (HTTP/socket)   │
│  - connect_server.py (路由)  │
│  - ida_wrapper.py (IDA 封装) │
└─────────────────────────────┘
```

工作流程(单次会话)
1. 客户端调用 `open_binary(path)`。
2. Go 在 `session.Registry` 中创建会话(8 字符 UUID),分配 loopback 空闲端口作为 `SocketPath`。
3. Go 通过 `worker.Manager.Start` 拉起 `python python/worker/server.py --socket host:port --binary <path> --session-id <id>`,使用独立的 `context.Background()` 派生的 `workerCtx`(worker 寿命独立于 HTTP 请求)。
4. Worker 启动监听后写出 ready 文件,Go 端通过 `waitForSocket` 拨号确认。
5. Go 用 Connect 客户端(`SessionControl`/`AnalysisTools`/`Healthcheck`)代理后续工具调用到 Worker。
6. `Watchdog` 监控空闲超时(默认 4 小时),`close_binary` 或超时触发 `SaveDatabase` 后 Kill worker 并清理。
7. 会话元数据通过 `session.Store` 落盘到 `<DatabaseDirectory>/sessions/<id>.json`,服务重启时由 `RestoreSessions` 恢复。

## 目录结构

```
ida-headless-mcp/
├── .codex-plugin/                 # Codex plugin manifest + MCP 配置
├── .claude-plugin/                # Claude Code plugin manifest
├── .claude/settings.json          # Claude Code 权限白名单示例
├── bin/                           # 插件预编译二进制
├── commands/                      # Claude slash command 文档
├── config/config.example.json     # 插件运行时 config.json 示例
├── python/worker/                 # 插件运行时 Python worker 副本
├── skills/                        # Codex setup/status skills
├── scripts/launch.py              # 插件自包含 launcher
├── src/
│   ├── cmd/ida-mcp-server/main.go # 入口:解析 flags、加载 config、启动 HTTP 服务、注册信号处理
│   ├── internal/                  # MCP 工具、session、worker 管理
│   ├── proto/ida/worker/v1/       # service.proto + generate.go
│   ├── ida/worker/v1/             # protoc 生成的 Go 代码
│   ├── python/worker/             # 源码版 Python worker
│   ├── python/tests/              # Python worker 测试
│   ├── contrib/il2cpp/            # Il2CppDumper 配套脚本(MIT,第三方代码)
│   ├── samples/                   # ls_arm64e 示例二进制 + .i64
│   ├── scripts/                   # setup_idalib / consistency / inspector / source launcher
│   ├── Makefile                   # 默认目标 = setup
│   ├── consistency.yaml           # consistency.sh 使用的规则
│   ├── config.example.json        # src/config.json 示例
│   └── go.mod / go.sum
├── CLAUDE.md
└── README.md
```

> 注意:Python 测试位于 `src/python/tests/`;Go 的黄金文件位于 `src/internal/server/testdata/`。

## 配置

`server.Config` 字段(`src/internal/server/server.go`):

| 字段 | JSON key | 默认值 | env override | flag override |
|---|---|---|---|---|
| `Port` | `port` | 17300 | `IDA_MCP_PORT` | `--port` |
| `SessionTimeoutMin` | `session_timeout_minutes` | 240 | `IDA_MCP_SESSION_TIMEOUT_MIN` | `--session-timeout`(Duration) |
| `AutoSaveIntervalMin` | `auto_save_interval_minutes` | 5 | — | — |
| `MaxConcurrentSession` | `max_concurrent_sessions` | 0(=无限) | `IDA_MCP_MAX_SESSIONS` | `--max-sessions` |
| `DatabaseDirectory` | `database_directory` | XDG / `~/.local/share/ida-mcp/sessions` / `os.TempDir()/ida_sessions` | — | — |
| `PythonWorkerPath` | `python_worker_path` | `python/worker/server.py` | `IDA_MCP_WORKER` | `--worker` |
| `Debug` | `debug` | false | `IDA_MCP_DEBUG` | `--debug` |

启动时 `validateConfig` 会:`MaxConcurrentSession >= 0`、`PythonWorkerPath` 存在且为文件、Unix 上检查可执行位(Windows 跳过)。

## MCP 工具(共 52 个)

会话级别:`open_binary`, `close_binary`, `list_sessions`, `save_database`, `get_session_progress`, `run_auto_analysis`, `watch_auto_analysis`。

读取分析:`get_bytes`, `get_disasm`, `get_function_disasm`, `get_decompiled_func`, `get_functions`, `get_imports`, `get_exports`, `get_strings`, `get_xrefs_to`, `get_xrefs_from`, `get_data_refs`, `get_string_xrefs`, `get_segments`, `get_function_name`, `get_function_info`, `get_entry_point`, `get_dword_at`, `get_qword_at`, `get_instruction_length`, `get_type_at`, `data_read_string`, `data_read_byte`, `get_globals`, `list_structs`, `get_struct`, `list_enums`, `get_enum`, `get_name`, `get_comment`, `get_func_comment`。

写入修改:`set_comment`, `set_func_comment`, `set_decompiler_comment`, `set_lvar_type`, `rename_lvar`, `set_global_type`, `rename_global`, `set_name`, `set_function_type`, `delete_name`, `make_function`。

搜索:`find_binary`, `find_text`。

元数据导入:`import_il2cpp`(Unity / Il2CppDumper)、`import_flutter`(Flutter / unflutter 输出 `flutter_meta.json`,最近从 Blutter 切换而来)。

分页约定:`offset >= 0`、`limit` 默认 1000、上限 10000(`server.normalizePagination`)。

## Proto 与代码生成

- 权威来源:`src/proto/ida/worker/v1/service.proto`(三个服务 `SessionControl` / `AnalysisTools` / `Healthcheck`)。
- Go 生成产物:`src/ida/worker/v1/*.pb.go` + `workerconnect/`(已提交到仓库)。
- Python 生成产物:`src/python/worker/gen/ida/worker/v1/*_pb2.py`(已提交)。
- 修改 proto 后:进入 `src/` 运行 `make proto`(需要 `protoc` 33.x);CI 通过 `make proto-check` + `git diff --exit-code` 校验。
- `make install-tools` 在 `src/` 内安装 `protoc-gen-go` 与 `protoc-gen-connect-go`。

## 构建 / 测试 / 运行

```bash
cd src
make setup          # 默认目标:setup-idalib + install-python + build
make build          # go build -o bin/ida-mcp-server ./cmd/ida-mcp-server
make test           # go test ./internal/... ./ida/... + scripts/consistency.sh
make test-all       # 加上 -tags=integration 的所有测试
make integration-test  # 仅 MCP 传输集成测试(StreamableHTTP + SSE)
make run            # build 后启动服务
make restart        # pkill 旧进程后重新启动
make inspector      # 启动 MCP Inspector
make clean          # 删除生成的 pb.go / bin / __pycache__
```

Python 依赖:`pip3 install -r src/python/requirements.txt`,测试依赖 `pip3 install -r src/python/requirements-test.txt`。

## 跨平台说明(Windows / Unix)

当前实现使用 **TCP loopback** 连接 Go server 与 Python worker,以适配 Windows:

- `session.Registry` 通过 `allocatePort()`(绑定 `127.0.0.1:0` 后释放)分配端口,`Session.SocketPath` 形如 `127.0.0.1:<port>`。
- `worker.Manager` 拨号改为 `net.Dial("tcp", addr)`,baseURL 使用 `http://<host:port>`。
- Python `server.py` 用 `serve_on_tcp` 监听 `AF_INET`,启动时写出 `<host>_<port>.ready` 到 `%TEMP%`/`/tmp`。
- worker 命令从 `python3` 改为 `python`(Windows 习惯)。
- `validateConfig` 在 Windows 下跳过 `chmod +x` 检查。
- `GetDefaultDBDir` 用 `os.UserHomeDir()` + `os.TempDir()` 替换硬编码 `$HOME` / `/tmp`。
- `manager_test.go` 中 `processAlive` 改用 `os.FindProcess`(Windows 下 `syscall.Kill(0)` 不可用)。

插件运行时配置使用 `config/config.example.json` -> `config.json`;后者已加入 `.gitignore`,不要提交。
源码运行配置可从 `src/config.example.json` 复制为 `src/config.json`;同样不要提交。

## 持久化与恢复

- `session.Store` 使用 `<DatabaseDirectory>/sessions/<id>.json`,采用 `.tmp + Rename` 原子写入。
- `Session` 持有 mutex,`Touch()` 更新 `LastActivity`,`IsExpired()` 比较 `Timeout`。
- 服务启动调用 `srv.RestoreSessions()` 重建 Registry(socket 端口重新分配,worker 进程不会自动重启)。
- 在 `cmd` 中收到 `SIGINT`/`SIGTERM` 时,先 `httpServer.Shutdown(10s)`,再依次 `workers.Stop(sess.ID)`,触发 `CloseSession(save=true)` 后 Kill 进程并 Wait,避免僵尸。

## 关键不变量与注意事项

- **Worker 寿命独立于 HTTP 请求**:`exec.CommandContext(workerCtx, ...)` 中 `workerCtx` 派生自 `context.Background()`,而非 HTTP 请求 ctx;不要回退成请求 ctx,否则客户端断开会杀掉 worker。
- **进程清理**:`Stop()` 必须 `cmd.Wait()` 才能避免僵尸;`monitorWorker` 也会 Wait,但 `cmd.Wait()` 多次调用是幂等的。
- **会话 ID = uuid 前 8 字符**:足以避免冲突但便于人类阅读,不要直接暴露完整 UUID。
- **Binary 路径双索引**:`Registry.binaryIndex` 用 `filepath.Clean` 后的路径,`open_binary` 同一路径会复用现有 session(语义体现在 server 层,registry 层仅返回错误)。
- **MaxConcurrentSession=0 表示不限**:历史上是 10,`a5a5f64` 移除了任意限制。
- **Connect 协议头**:Python `server.py` 的 HTTP 解析非常简陋(只读到 `\r\n\r\n` + Content-Length),对客户端行为有隐含假设。
- **测试时静音 worker stdout**:`flag.Lookup("test.v")` 检测到测试模式则丢弃,生产则继承 stdout/stderr。

## 添加新 MCP 工具的流程

1. 在 `src/proto/ida/worker/v1/service.proto` 中添加 RPC 与消息。
2. 进入 `src/` 后运行 `make proto` 重新生成 Go/Python 代码。
3. 在 `src/python/worker/ida_wrapper.py` 实现 idalib 调用。
4. 在 `src/python/worker/connect_server.py` 增加 dispatch 分支。
5. 在 `src/internal/server/server.go` 的 `RegisterTools` 注册 MCP 工具,并在合适的 `read.go`/`write.go`/`search.go` 等文件中实现 handler。
6. 在 `src/` 运行 `make test` 跑一致性脚本(`consistency.yaml` 列出的规则)。

## 维护提示

不要把机器本地状态写进本文档。插件运行时默认配置示例放在 `config/config.example.json`;个人覆盖放在被忽略的根目录 `config.json`。
源码改动涉及 Python worker 时,进入 `src/` 运行 `make plugin-sync` 或 `make prebuilt` 同步根目录 `python/worker/`。

最近 5 个提交方向:
- `dc9c56f` import_unflutter 增加耗时统计 + 忽略本地 settings
- `784375f` setup 脚本自动检测最新 IDA 安装
- `e2304bb` 用 unflutter 替换 Blutter 作为 Flutter/Dart 元数据来源
- `eb701bf` protoc 升级到 33.2 重新生成
- `23f340f` Makefile 增加 protoc 检查
