package server

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PreflightRequest struct {
	IDAPath string `json:"ida_path,omitempty" mcp:"optional IDA install directory to use for idalib initialization"`
}

func (s *Server) preflight(ctx context.Context, req *mcp.CallToolRequest, args PreflightRequest) (*mcp.CallToolResult, any, error) {
	s.logToolInvocation("preflight", "", nil)

	pythonCmd := findPython()
	if pythonCmd == "" {
		return textResult("preflight FAILED\n\nPython not found on PATH. Install Python 3.10+ and ensure 'python' or 'python3' is available."), nil, nil
	}

	var lines []string

	pyVer, err := runQuiet(ctx, pythonCmd, "--version")
	if err != nil {
		lines = append(lines, fmt.Sprintf("python: FAIL (%v)", err))
	} else {
		lines = append(lines, fmt.Sprintf("python: OK (%s)", pyVer))
	}

	idalibVer, err := runQuiet(ctx, pythonCmd, "-c",
		"import idapro; v=idapro.get_library_version(); print(f'{v[0]}.{v[1]}')")
	if err == nil {
		lines = append(lines, fmt.Sprintf("idalib: OK (version %s)", idalibVer))
		return textResult("preflight OK\n\n" + strings.Join(lines, "\n")), nil, nil
	}

	lines = append(lines, "idalib: MISSING — attempting initialization...")

	launchPy := findLaunchScript()
	if launchPy == "" {
		lines = append(lines, "init: SKIP (launch.py not found)")
		lines = append(lines, "\nAction required: tell the user to run /ida-init --ida-path \"<IDA install dir>\"")
		return textResult("preflight FAILED\n\n" + strings.Join(lines, "\n")), nil, nil
	}

	initArgs := []string{launchPy, "init", "--skip-build"}
	if args.IDAPath != "" {
		initArgs = append(initArgs, "--ida-path", args.IDAPath)
	}

	initCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	initCmd := exec.CommandContext(initCtx, pythonCmd, initArgs...)
	var initBuf bytes.Buffer
	initCmd.Stdout = &initBuf
	initCmd.Stderr = &initBuf
	initErr := initCmd.Run()

	if initErr != nil {
		lines = append(lines, fmt.Sprintf("init: FAILED (%v)", initErr))
		output := initBuf.String()
		if output != "" {
			lines = append(lines, "init output:", output)
		}
		lines = append(lines, "\nAction required: tell the user to run /ida-init --ida-path \"<IDA install dir>\"")
		return textResult("preflight FAILED\n\n" + strings.Join(lines, "\n")), nil, nil
	}

	idalibVer, err = runQuiet(ctx, pythonCmd, "-c",
		"import idapro; v=idapro.get_library_version(); print(f'{v[0]}.{v[1]}')")
	if err != nil {
		lines = append(lines, "init: completed but idalib still not importable")
		lines = append(lines, "\nAction required: tell the user to run /ida-init --ida-path \"<IDA install dir>\"")
		return textResult("preflight FAILED\n\n" + strings.Join(lines, "\n")), nil, nil
	}

	lines = append(lines, fmt.Sprintf("idalib: OK after init (version %s)", idalibVer))
	return textResult("preflight OK\n\n" + strings.Join(lines, "\n")), nil, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func findPython() string {
	if p, err := exec.LookPath("python"); err == nil {
		return p
	}
	if p, err := exec.LookPath("python3"); err == nil {
		return p
	}
	return ""
}

func runQuiet(ctx context.Context, name string, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func findLaunchScript() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(exeDir, "..", "scripts", "launch.py"),
		filepath.Join(exeDir, "scripts", "launch.py"),
	}
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return ""
}
