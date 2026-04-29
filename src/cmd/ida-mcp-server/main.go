package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zboralski/ida-headless-mcp/internal/server"
	"github.com/zboralski/ida-headless-mcp/internal/session"
	"github.com/zboralski/ida-headless-mcp/internal/worker"
)

const version = "0.2.0"

var (
	configPath   = flag.String("config", "config.json", "Path to server config")
	portFlag     = flag.Int("port", 0, "HTTP port (overrides config)")
	pythonWorker = flag.String("worker", "", "Python worker script (overrides config)")
	maxSessions  = flag.Int("max-sessions", 0, "Max concurrent sessions (overrides config)")
	timeoutFlag  = flag.Duration("session-timeout", 0, "Session idle timeout (overrides config)")
	debugFlag    = flag.Bool("debug", false, "Enable verbose debug logging")
	stdioFlag    = flag.Bool("stdio", false, "Run as stdio MCP server (for Claude Code / Codex plugin install)")
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			os.Exit(runInit(os.Args[2:]))
		case "print-config":
			os.Exit(runPrintConfig(os.Args[2:]))
		case "version", "--version", "-v":
			fmt.Printf("ida-mcp-server %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	flag.Parse()
	runServe()
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `ida-mcp-server %s — Headless IDA Pro binary analysis via MCP

Usage:
  ida-mcp-server [flags]              Run HTTP MCP server (default)
  ida-mcp-server --stdio               Run stdio MCP server (for plugin install)
  ida-mcp-server init [flags]          Detect IDA, install deps, build binary
  ida-mcp-server print-config <target> Print MCP config for a client
  ida-mcp-server version               Show version
  ida-mcp-server help                  Show this help

Print-config targets:
  claude-desktop   JSON snippet for claude_desktop_config.json
  claude-code      JSON snippet for ~/.claude/settings.json
  codex            TOML snippet for ~/.codex/config.toml
  codex-add        Shell command using 'codex mcp add'

Flags (serve mode):
`, version)
	flag.PrintDefaults()
}

func runServe() {
	// Stdio mode requires logs go to stderr to avoid corrupting JSON-RPC on stdout
	logOut := os.Stdout
	if *stdioFlag {
		logOut = os.Stderr
	}
	logger := log.New(logOut, "[MCP] ", log.LstdFlags)
	logger.Printf("Starting IDA Headless MCP Server %s", version)

	cfg, err := server.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	server.ApplyEnvOverrides(&cfg)

	if *portFlag > 0 {
		cfg.Port = *portFlag
	}
	if *pythonWorker != "" {
		cfg.PythonWorkerPath = *pythonWorker
	}
	if *maxSessions > 0 {
		cfg.MaxConcurrentSession = *maxSessions
	}

	sessionTimeout := time.Duration(cfg.SessionTimeoutMin) * time.Minute
	if *timeoutFlag > 0 {
		sessionTimeout = *timeoutFlag
	}

	if *debugFlag {
		cfg.Debug = true
	}

	resolvePythonWorker(&cfg)

	if err := validateConfig(&cfg); err != nil {
		logger.Fatalf("invalid configuration: %v", err)
	}

	registry := session.NewRegistry(cfg.MaxConcurrentSession)
	workers := worker.NewManager(cfg.PythonWorkerPath, logger)
	stateDir := filepath.Join(cfg.DatabaseDirectory, "sessions")
	store, err := session.NewStore(stateDir)
	if err != nil {
		logger.Fatalf("failed to initialize session store: %v", err)
	}

	srv := server.New(registry, workers, logger, sessionTimeout, cfg.Debug, store)
	srv.RestoreSessions()
	go srv.Watchdog()

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "ida-headless",
		Version: version,
	}, nil)
	srv.RegisterTools(mcpServer)

	if *stdioFlag {
		runStdio(mcpServer, registry, workers, logger)
		return
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	mux := srv.HTTPMux(mcpServer)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Printf("Listening on %s", addr)
	logger.Printf("HTTP transport at http://localhost:%d/", cfg.Port)
	logger.Printf("SSE transport at http://localhost:%d/sse", cfg.Port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Shutting down gracefully...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Printf("HTTP server shutdown error: %v", err)
		}
		for _, sess := range registry.List() {
			if err := workers.Stop(sess.ID); err != nil {
				logger.Printf("Failed to stop worker %s: %v", sess.ID, err)
			}
		}
		logger.Println("Shutdown complete")
		os.Exit(0)
	}()

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal(err)
	}
}

func runStdio(mcpServer *mcp.Server, registry *session.Registry, workers worker.Controller, logger *log.Logger) {
	logger.Println("Running in stdio mode (MCP over stdin/stdout)")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := mcpServer.Run(ctx, &mcp.StdioTransport{})
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Printf("MCP server error: %v", err)
	}

	logger.Println("Shutting down workers...")
	for _, sess := range registry.List() {
		if stopErr := workers.Stop(sess.ID); stopErr != nil {
			logger.Printf("Failed to stop worker %s: %v", sess.ID, stopErr)
		}
	}
	logger.Println("Shutdown complete")
}

// resolvePythonWorker looks for the worker script in plausible locations relative
// to the binary, so platform-specific prebuilt binaries work when called from
// outside the repo by the Claude Code/Codex launchers.
func resolvePythonWorker(cfg *server.Config) {
	if cfg.PythonWorkerPath == "" {
		cfg.PythonWorkerPath = "python/worker/server.py"
	}
	if _, err := os.Stat(cfg.PythonWorkerPath); err == nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(exeDir, "..", "python", "worker", "server.py"),
		filepath.Join(exeDir, "python", "worker", "server.py"),
	}
	for _, c := range candidates {
		if abs, absErr := filepath.Abs(c); absErr == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				cfg.PythonWorkerPath = abs
				return
			}
		}
	}
}

func validateConfig(cfg *server.Config) error {
	if cfg.MaxConcurrentSession < 0 {
		return fmt.Errorf("max_concurrent_sessions must be non-negative, got %d (use 0 for unlimited)", cfg.MaxConcurrentSession)
	}

	if cfg.PythonWorkerPath == "" {
		return fmt.Errorf("python_worker_path is required")
	}

	absPath, err := filepath.Abs(cfg.PythonWorkerPath)
	if err != nil {
		return fmt.Errorf("invalid python_worker_path %q: %w", cfg.PythonWorkerPath, err)
	}
	cfg.PythonWorkerPath = absPath

	info, err := os.Stat(cfg.PythonWorkerPath)
	if err != nil {
		return fmt.Errorf("python_worker_path %q not found: %w", cfg.PythonWorkerPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("python_worker_path %q is a directory, expected a Python script", cfg.PythonWorkerPath)
	}

	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		return fmt.Errorf("python_worker_path %q is not executable (try: chmod +x %s)", cfg.PythonWorkerPath, cfg.PythonWorkerPath)
	}

	return nil
}
