# CLAUDE.md

项目级指引文件,用于在新会话中快速建立对本仓库的理解。

## 项目身份

**ida-headless-mcp** —— 通过 Model Context Protocol(MCP)对外暴露 IDA Pro 的无头(headless)二进制分析能力。
Go 语言负责对外的 MCP 服务、多会话编排、Worker 生命周期管理与持久化;Python 进程承载真正的 IDA(`idalib`)调用。每个分析会话独立进程,彼此隔离。

- Go module:`github.com/zboralski/ida-headless-mcp`(go 1.25)
- 主要依赖:`connectrpc.com/connect`、`github.com/modelcontextprotocol/go-sdk`、`google.golang.org/protobuf`、`github.com/google/uuid`
- 默认监听端口:`17300`(`/` 为 Streamable HTTP,`/sse` 为 SSE 兼容端点)
- License:MIT

## 架构

```
┌─────────────────────────────┐
│   MCP Client (HTTP/SSE)     │  Claude Desktop / Claude Code / CLI
└──────────────┬──────────────┘
               │  http://localhost:17300/
┌──────────────▼──────────────┐
│   Go Server (cmd + internal)│
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
├── cmd/ida-mcp-server/main.go     # 入口:解析 flags、加载 config、启动 HTTP 服务、注册信号处理
├── internal/
│   ├── server/
│   │   ├── server.go              # Config、defaults、env overrides、所有 MCP 工具注册(52 个)
│   │   ├── http.go                # /sse 与 / 的 mux,Streamable HTTP + SSE handler
│   │   ├── session.go             # open_binary / close_binary / list_sessions / save_database
│   │   ├── read.go                # 反汇编、伪代码、函数列表、xref 等只读工具
│   │   ├── write.go               # rename / set_comment / set_*_type 等写入工具
│   │   ├── search.go              # find_binary / find_text / 正则过滤
│   │   ├── flutter.go             # import_flutter (unflutter 元数据)
│   │   ├── il2cpp.go              # import_il2cpp (Il2CppDumper 元数据)
│   │   ├── progress.go            # auto-analysis 进度快照
│   │   ├── cache.go               # session 级别的缓存
│   │   ├── params.go / types.go / util.go
│   │   ├── golden_test.go         # 黄金文件比对
│   │   ├── transport_test.go      # MCP 传输集成测试(StreamableHTTP / SSE)
│   │   └── testdata/golden/
│   ├── session/
│   │   ├── registry.go            # 内存 Registry(按 ID 与 binary 路径双索引)
│   │   └── store.go               # JSON 元数据持久化(原子写入 .tmp + Rename)
│   └── worker/
│       ├── manager.go             # Worker 进程拉起、TCP 探活、Connect 客户端组装、监控/Stop
│       └── manager_test.go        # 进程生命周期独立性测试
├── proto/ida/worker/v1/
│   ├── service.proto              # 三大服务 + 所有消息(权威来源)
│   └── generate.go                # //go:generate 触发 protoc
├── ida/worker/v1/                 # protoc 生成的 Go 代码(*.pb.go + connect 客户端)
├── python/worker/
│   ├── server.py                  # TCP 监听 + 极简 HTTP/1.1 解析 + 路由到 ConnectServer
│   ├── connect_server.py          # Connect RPC 路由(SessionControl/AnalysisTools/Healthcheck)
│   ├── ida_wrapper.py             # idalib 封装(46KB,核心业务在此)
│   └── gen/ida/worker/v1/         # protoc 生成的 Python pb2 代码
├── contrib/il2cpp/                # Il2CppDumper 配套脚本(MIT,第三方代码)
├── samples/                       # ls_arm64e 示例二进制 + .i64
├── scripts/
│   ├── setup_idalib.sh            # 自动检测最新 IDA 安装并配置 idalib
│   ├── consistency.sh             # `make test` 调用,校验 proto/工具/Go 代码一致性
│   ├── gen_proto.sh
│   └── inspector.sh               # MCP Inspector 快捷启动
├── Makefile                       # 默认目标 = setup
├── consistency.yaml               # consistency.sh 使用的规则
├── config.json                    # 本地配置(已加入 .gitignore,工作树有未追踪副本)
├── go.mod / go.sum
└── README.md
```

> 注意:`tests/` 在 README 中提到但当前仓库**不存在**;只有 `internal/server/testdata/` 和 Go test 文件。

## 配置

`server.Config` 字段(`internal/server/server.go`):

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

- 权威来源:`proto/ida/worker/v1/service.proto`(三个服务 `SessionControl` / `AnalysisTools` / `Healthcheck`)。
- Go 生成产物:`ida/worker/v1/*.pb.go` + `workerconnect/`(已提交到仓库)。
- Python 生成产物:`python/worker/gen/ida/worker/v1/*_pb2.py`(已提交)。
- 修改 proto 后:`make proto`(需要 `protoc` 33.x);CI 通过 `make proto-check` + `git diff --exit-code` 校验。
- `make install-tools` 安装 `protoc-gen-go` 与 `protoc-gen-connect-go`。

## 构建 / 测试 / 运行

```bash
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

Python 依赖:`pip3 install -r python/requirements.txt`,测试依赖 `pip3 install -r requirements-test.txt`。

## 跨平台说明(Windows / Unix)

工作树中(已修改但**未提交**)将传输从 Unix Domain Socket 切换为 **TCP loopback**,以适配 Windows:

- `session.Registry` 通过 `allocatePort()`(绑定 `127.0.0.1:0` 后释放)分配端口,`Session.SocketPath` 形如 `127.0.0.1:<port>`。
- `worker.Manager` 拨号改为 `net.Dial("tcp", addr)`,baseURL 使用 `http://<host:port>`。
- Python `server.py` 用 `serve_on_tcp` 监听 `AF_INET`,启动时写出 `<host>_<port>.ready` 到 `%TEMP%`/`/tmp`。
- worker 命令从 `python3` 改为 `python`(Windows 习惯)。
- `validateConfig` 在 Windows 下跳过 `chmod +x` 检查。
- `GetDefaultDBDir` 用 `os.UserHomeDir()` + `os.TempDir()` 替换硬编码 `$HOME` / `/tmp`。
- `manager_test.go` 中 `processAlive` 改用 `os.FindProcess`(Windows 下 `syscall.Kill(0)` 不可用)。

> 这些改动尚未 commit,代表当前正在适配 Windows 的工作。`config.json` 也是本地未追踪文件(`.gitignore` 已覆盖)。

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

1. 在 `proto/ida/worker/v1/service.proto` 中添加 RPC 与消息。
2. `make proto` 重新生成 Go/Python 代码。
3. 在 `python/worker/ida_wrapper.py` 实现 idalib 调用。
4. 在 `python/worker/connect_server.py` 增加 dispatch 分支。
5. 在 `internal/server/server.go` 的 `RegisterTools` 注册 MCP 工具,并在合适的 `read.go`/`write.go`/`search.go` 等文件中实现 handler。
6. `make test` 跑一致性脚本(`consistency.yaml` 列出的规则)。

## 当前工作树状态(2026-04-28)

未提交改动:`cmd/ida-mcp-server/main.go`、`internal/server/server.go`、`internal/session/registry.go`、`internal/worker/manager.go`、`internal/worker/manager_test.go`、`python/worker/server.py`,主题是 **从 Unix socket 迁移到 TCP loopback 以支持 Windows**。
未追踪:`config.json`(本地配置)。

最近 5 个提交方向:
- `dc9c56f` import_unflutter 增加耗时统计 + 忽略本地 settings
- `784375f` setup 脚本自动检测最新 IDA 安装
- `e2304bb` 用 unflutter 替换 Blutter 作为 Flutter/Dart 元数据来源
- `eb701bf` protoc 升级到 33.2 重新生成
- `23f340f` Makefile 增加 protoc 检查
