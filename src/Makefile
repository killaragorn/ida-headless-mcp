.PHONY: proto proto-check clean build plugin-sync prebuilt test run restart integration-test inspector setup setup-idalib init plugin-check stdio

GO_BUILD_FLAGS ?= -trimpath
GO_LDFLAGS ?= -s -w
PLUGIN_DIR := ..

# Protobuf generation
proto:
	@command -v protoc >/dev/null 2>&1 || { echo "Error: protoc not found. Install protobuf (brew install protobuf on macOS)"; exit 1; }
	PATH="$(shell go env GOPATH)/bin:$${PATH}" go generate ./proto/ida/worker/v1

proto-check:
	@command -v protoc >/dev/null 2>&1 || { echo "Error: protoc not found. Install protobuf (brew install protobuf on macOS)"; exit 1; }
	PATH="$(shell go env GOPATH)/bin:$${PATH}" go generate ./proto/ida/worker/v1
	git diff --exit-code proto ida python/worker/gen

# Build server binary (proto files are committed, no need to regenerate)
build:
	@mkdir -p bin
	go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o bin/ida-mcp-server ./cmd/ida-mcp-server

# Sync source runtime files into the isolated plugin package.
plugin-sync:
	@rm -rf $(PLUGIN_DIR)/python
	@mkdir -p $(PLUGIN_DIR)/python
	cp -R python/worker $(PLUGIN_DIR)/python/worker
	cp python/requirements.txt $(PLUGIN_DIR)/python/requirements.txt

# Rebuild all committed plugin binaries in the isolated plugin package.
prebuilt: plugin-sync
	@mkdir -p $(PLUGIN_DIR)/bin
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(PLUGIN_DIR)/bin/ida-mcp-server-windows-amd64.exe ./cmd/ida-mcp-server
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(PLUGIN_DIR)/bin/ida-mcp-server-linux-amd64 ./cmd/ida-mcp-server
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(PLUGIN_DIR)/bin/ida-mcp-server-linux-arm64 ./cmd/ida-mcp-server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(PLUGIN_DIR)/bin/ida-mcp-server-darwin-amd64 ./cmd/ida-mcp-server
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -ldflags '$(GO_LDFLAGS)' -o $(PLUGIN_DIR)/bin/ida-mcp-server-darwin-arm64 ./cmd/ida-mcp-server

# Run the friendly initializer (detect IDA, install deps, build binary)
init: build
	./bin/ida-mcp-server init

# Run server in stdio mode (for plugin install testing)
stdio: build
	./bin/ida-mcp-server --stdio

# Sanity-check the Claude Code plugin manifest is valid JSON
plugin-check:
	@command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1 || { echo "python required for plugin-check"; exit 1; }
	@python3 -c "import json; [json.load(open(p)) for p in ['$(PLUGIN_DIR)/.claude-plugin/plugin.json','$(PLUGIN_DIR)/.codex-plugin/plugin.json','$(PLUGIN_DIR)/.codex-plugin/mcp.json','$(PLUGIN_DIR)/.agents/plugins/marketplace.json','$(PLUGIN_DIR)/.claude/settings.json','$(PLUGIN_DIR)/config/config.example.json','config.example.json']]" 2>/dev/null || python -c "import json; [json.load(open(p)) for p in ['$(PLUGIN_DIR)/.claude-plugin/plugin.json','$(PLUGIN_DIR)/.codex-plugin/plugin.json','$(PLUGIN_DIR)/.codex-plugin/mcp.json','$(PLUGIN_DIR)/.agents/plugins/marketplace.json','$(PLUGIN_DIR)/.claude/settings.json','$(PLUGIN_DIR)/config/config.example.json','config.example.json']]"
	@echo "plugin JSON is valid"

# Run fast unit tests only (excludes integration tests)
test:
	go test -v ./internal/... ./ida/...
	@./scripts/consistency.sh

# Run all tests including IDA integration tests
test-all:
	go test -v -tags=integration ./...
	@./scripts/consistency.sh

# Run Go integration tests (MCP transports)
integration-test:
	go test ./internal/server -run TestStreamableHTTPTransportLifecycle -v
	go test ./internal/server -run TestSSETransportLifecycle -v

# Run server
run: build
	./bin/ida-mcp-server

# Restart server (kill existing, rebuild, start)
restart: build
	@pkill -f 'bin/ida-mcp-server' 2>/dev/null || true
	@pkill -f 'python3.*ida-headless-mcp/python/worker/server.py' 2>/dev/null || true
	@sleep 1
	./bin/ida-mcp-server

# Clean generated files
clean:
	rm -rf ida/worker/v1/*.pb.go
	rm -rf bin/
	find python/worker -name "*.pyc" -delete
	find python/worker -name "__pycache__" -delete

# Install Go tools
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# Install Python dependencies
install-python:
	pip3 install -r python/requirements.txt

# Setup idalib (required for IDA integration)
setup-idalib:
	@./scripts/setup_idalib.sh

# Full setup: idalib + python deps + build
setup: setup-idalib install-python build
	@echo "Setup complete. Run: ./bin/ida-mcp-server"

# Start MCP Inspector for testing
inspector:
	@./scripts/inspector.sh

.DEFAULT_GOAL := setup
