package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// runInit executes the friendly setup flow:
//  1. Detect Python and Go toolchains
//  2. Detect IDA Pro installation
//  3. Install idalib (pip + activation script)
//  4. Install Python worker dependencies
//  5. Build the Go binary
//
// Each step is skippable so users with partial setups can advance one piece at a time.
func runInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	idaPath := fs.String("ida-path", "", "Path to IDA installation directory (defaults to auto-detect, env IDA_PATH)")
	skipBuild := fs.Bool("skip-build", false, "Skip building the Go binary")
	skipPython := fs.Bool("skip-python", false, "Skip installing Python worker dependencies")
	skipIDA := fs.Bool("skip-ida", false, "Skip idalib detection and activation")
	yes := fs.Bool("yes", false, "Skip confirmation prompts")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: ida-mcp-server init [flags]")
		fmt.Fprintln(fs.Output(), "Friendly initializer: detect IDA, install deps, build binary.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	_ = yes

	root, err := repoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot determine repository root: %v\n", err)
		return 1
	}

	header()

	pyCmd, pyVer, ok := detectPython()
	step(1, 5, "Python toolchain", ok, pyVer, "Install Python 3.10+ from https://python.org and re-run init.")
	if !ok {
		return 1
	}

	goVer, ok := detectGo()
	if *skipBuild {
		stepSkipped(2, 5, "Go toolchain", "build skipped via --skip-build")
	} else {
		step(2, 5, "Go toolchain", ok, goVer, "Install Go 1.21+ from https://go.dev/dl and re-run init, or use --skip-build.")
		if !ok {
			return 1
		}
	}

	if *skipIDA {
		stepSkipped(3, 5, "idalib", "skipped via --skip-ida")
	} else {
		path, err := resolveIDA(*idaPath)
		if err != nil {
			step(3, 5, "IDA installation", false, "", err.Error())
			fmt.Println()
			fmt.Println("  Hint: pass --ida-path \"/path/to/IDA\" or set IDA_PATH env var.")
			return 1
		}
		fmt.Printf("[3/5] idalib       … using %s\n", path)
		if err := activateIdalib(pyCmd, path); err != nil {
			fmt.Printf("       FAILED: %v\n", err)
			return 1
		}
		fmt.Println("       ✓ idalib activated")
	}

	if *skipPython {
		stepSkipped(4, 5, "Python deps", "skipped via --skip-python")
	} else {
		fmt.Print("[4/5] Python deps  … ")
		req := filepath.Join(root, "python", "requirements.txt")
		if err := runStream(pyCmd, "-m", "pip", "install", "-r", req); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			return 1
		}
		fmt.Println("       ✓ requirements installed")
	}

	if *skipBuild {
		stepSkipped(5, 5, "Go binary", "skipped via --skip-build")
	} else {
		fmt.Print("[5/5] Go binary    … ")
		buildRoot := sourceRootForBuild(root)
		out := filepath.Join(root, "bin", "ida-mcp-server")
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			return 1
		}
		buildCmd := exec.Command("go", "build", "-o", out, "./cmd/ida-mcp-server")
		buildCmd.Dir = buildRoot
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			return 1
		}
		fmt.Printf("       ✓ %s\n", out)
	}

	fmt.Println()
	fmt.Println("Init complete.")
	fmt.Println()
	printNextStepsAfterInit(root)
	return 0
}

func header() {
	fmt.Printf("ida-headless-mcp init   v%s\n", version)
	fmt.Println(strings.Repeat("=", 50))
}

func step(n, total int, label string, ok bool, detail, hint string) {
	if ok {
		fmt.Printf("[%d/%d] %-12s … ✓ %s\n", n, total, label, detail)
	} else {
		fmt.Printf("[%d/%d] %-12s … ✗ MISSING\n", n, total, label)
		if hint != "" {
			fmt.Printf("       %s\n", hint)
		}
	}
}

func stepSkipped(n, total int, label, reason string) {
	fmt.Printf("[%d/%d] %-12s … - %s\n", n, total, label, reason)
}

func detectPython() (string, string, bool) {
	candidates := []string{"python3", "python"}
	if runtime.GOOS == "windows" {
		candidates = []string{"python", "python3", "py"}
	}
	for _, name := range candidates {
		out, err := exec.Command(name, "--version").CombinedOutput()
		if err == nil && strings.Contains(strings.ToLower(string(out)), "python") {
			return name, strings.TrimSpace(string(out)), true
		}
	}
	return "", "", false
}

func detectGo() (string, bool) {
	out, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func resolveIDA(explicit string) (string, error) {
	if explicit != "" {
		if isDir(explicit) {
			return explicit, nil
		}
		return "", fmt.Errorf("--ida-path %q is not a directory", explicit)
	}
	if env := os.Getenv("IDA_PATH"); env != "" {
		if isDir(env) {
			return env, nil
		}
		return "", fmt.Errorf("IDA_PATH=%q is not a directory", env)
	}

	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		matches, _ := filepath.Glob("/Applications/IDA*.app/Contents/MacOS")
		candidates = append(candidates, matches...)
	case "linux":
		for _, pat := range []string{"/opt/ida*", "/opt/idapro*", "/opt/IDA*"} {
			matches, _ := filepath.Glob(pat)
			for _, m := range matches {
				if isDir(m) {
					candidates = append(candidates, m)
				}
			}
		}
		if home, _ := os.UserHomeDir(); home != "" {
			matches, _ := filepath.Glob(filepath.Join(home, "ida*"))
			for _, m := range matches {
				if isDir(m) {
					candidates = append(candidates, m)
				}
			}
		}
	case "windows":
		for _, pat := range []string{
			`C:\Program Files\IDA Pro*`,
			`C:\Program Files\IDA Essential*`,
			`C:\Program Files\IDA*`,
		} {
			matches, _ := filepath.Glob(pat)
			for _, m := range matches {
				if isDir(m) {
					candidates = append(candidates, m)
				}
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no IDA installation auto-detected on %s", runtime.GOOS)
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i] > candidates[j] })
	return candidates[0], nil
}

func activateIdalib(pyCmd, idaPath string) error {
	idalibDir := filepath.Join(idaPath, "idalib")
	if !isDir(idalibDir) {
		return fmt.Errorf("idalib directory not found in %s (requires IDA Pro 9.0+ or Essential 9.2+)", idaPath)
	}
	pythonDir := filepath.Join(idalibDir, "python")

	var pkgArg string
	wheels, _ := filepath.Glob(filepath.Join(pythonDir, "*.whl"))
	if len(wheels) > 0 {
		pkgArg = wheels[0]
	} else if _, err := os.Stat(filepath.Join(pythonDir, "setup.py")); err == nil {
		pkgArg = pythonDir
	} else {
		return fmt.Errorf("no wheel or setup.py found in %s", pythonDir)
	}

	if err := runStream(pyCmd, "-m", "pip", "install", "--force-reinstall", pkgArg); err != nil {
		return fmt.Errorf("pip install %s: %w", pkgArg, err)
	}

	activate := filepath.Join(pythonDir, "py-activate-idalib.py")
	if _, err := os.Stat(activate); err != nil {
		return fmt.Errorf("activation script not found: %s", activate)
	}
	if err := runStream(pyCmd, activate, "-d", idaPath); err != nil {
		return fmt.Errorf("py-activate-idalib failed: %w", err)
	}

	if err := runStream(pyCmd, "-c", "import idapro; v=idapro.get_library_version(); print(f'idalib {v[0]}.{v[1]} ready')"); err != nil {
		return fmt.Errorf("idalib import test failed: %w", err)
	}
	return nil
}

func runStream(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func repoRoot() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		// When running from a built binary in `bin/`, the repo root is its parent.
		exeDir := filepath.Dir(exe)
		if isDir(filepath.Join(exeDir, "..", "cmd", "ida-mcp-server")) {
			return filepath.Abs(filepath.Join(exeDir, ".."))
		}
		if isDir(filepath.Join(exeDir, "..", "src", "cmd", "ida-mcp-server")) {
			return filepath.Abs(filepath.Join(exeDir, ".."))
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "src", "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd, nil
		}
		dir = parent
	}
}

func sourceRootForBuild(root string) string {
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return root
	}
	src := filepath.Join(root, "src")
	if _, err := os.Stat(filepath.Join(src, "go.mod")); err == nil {
		return src
	}
	return root
}

func printNextStepsAfterInit(root string) {
	binPath := currentBinaryPath()
	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Println("  Install as Claude Code plugin (recommended):")
	fmt.Println("    /plugin marketplace add killaragorn/ida-headless-mcp")
	fmt.Println("    /plugin install ida-headless-mcp@ida-headless-mcp")
	fmt.Println()
	fmt.Println("  Or register manually with Codex CLI:")
	fmt.Printf("    codex mcp add ida-headless -- \"%s\" --stdio\n", binPath)
	fmt.Println()
	fmt.Println("  Or run the HTTP server directly:")
	fmt.Printf("    \"%s\"\n", binPath)
	fmt.Println()
	fmt.Println("  Print client config snippets:")
	fmt.Println("    ida-mcp-server print-config claude-desktop")
	fmt.Println("    ida-mcp-server print-config claude-code")
	fmt.Println("    ida-mcp-server print-config codex")
}
