package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func runPrintConfig(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ida-mcp-server print-config <target>")
		fmt.Fprintln(os.Stderr, "Targets: claude-desktop, claude-code, codex, codex-add")
		return 2
	}

	binPath := currentBinaryPath()
	binPathEsc := jsonEscape(binPath)

	switch strings.ToLower(args[0]) {
	case "claude-desktop":
		fmt.Print(claudeDesktopSnippet(binPathEsc))
	case "claude-code":
		fmt.Print(claudeCodeSnippet(binPathEsc))
	case "codex":
		fmt.Print(codexTOMLSnippet(binPath))
	case "codex-add":
		fmt.Print(codexAddSnippet(binPath))
	default:
		fmt.Fprintf(os.Stderr, "Unknown target %q. Valid: claude-desktop, claude-code, codex, codex-add\n", args[0])
		return 2
	}
	return 0
}

func currentBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		// Fallback to relative path
		if runtime.GOOS == "windows" {
			return filepath.Join("bin", "ida-mcp-server.exe")
		}
		return filepath.Join("bin", "ida-mcp-server")
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return exe
	}
	return abs
}

func claudeDesktopSnippet(bin string) string {
	return fmt.Sprintf(`Add this to your Claude Desktop config:

  macOS:   ~/Library/Application Support/Claude/claude_desktop_config.json
  Windows: %%APPDATA%%\Claude\claude_desktop_config.json
  Linux:   ~/.config/Claude/claude_desktop_config.json

{
  "mcpServers": {
    "ida-headless": {
      "command": "%s",
      "args": ["--stdio"]
    }
  }
}

Restart Claude Desktop after editing.
`, bin)
}

func claudeCodeSnippet(bin string) string {
	return fmt.Sprintf(`Recommended: install as a plugin
  /plugin install killaragorn/ida-headless-mcp

Or add manually to ~/.claude/settings.json:

{
  "mcpServers": {
    "ida-headless": {
      "command": "%s",
      "args": ["--stdio"]
    }
  }
}
`, bin)
}

func codexTOMLSnippet(bin string) string {
	binTOML := tomlEscape(bin)
	return fmt.Sprintf(`Add this to ~/.codex/config.toml:

[mcp_servers.ida-headless]
command = "%s"
args = ["--stdio"]

Or use 'codex mcp add' (see: ida-mcp-server print-config codex-add).
`, binTOML)
}

func codexAddSnippet(bin string) string {
	return fmt.Sprintf(`Run this to register with Codex CLI:

  codex mcp add ida-headless -- "%s" --stdio

Verify:

  codex mcp list
`, bin)
}

func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func tomlEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
