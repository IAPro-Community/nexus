package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	controlterminal "github.com/IAPro-Community/nexus/internal/control/terminal"
	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/executil"
)

const isolatedSecretServiceScript = `TMPDIR=$(mktemp -d /tmp/nexus-kr-XXXXXX) || exit 1
trap 'rm -rf "$TMPDIR"' EXIT INT TERM
export GNOME_KEYRING_CONTROL="$TMPDIR"
daemon="$1"
shift
"$daemon" --daemonize --components=secrets --control-directory="$TMPDIR" >/dev/null 2>&1 || true
exec "$@"`

// terminalKeyboardModeReset disables optional terminal input protocols that
// can remain enabled after an interrupted TUI session. Those protocols make
// normal keys arrive as CSI text in terminal emulators that do not fully
// negotiate them (notably embedded VS Code terminals).
// The reset also leaves application cursor/keypad mode. Without DECCKM being
// cleared, a TTY recovered after Ctrl+C can keep emitting SS3 arrow reports
// (ESC O A/B), which older or embedded terminal bridges render as literal
// characters in the provider prompt. It also resets all mouse tracking modes
// (?1000, ?1002, ?1003, ?1005, ?1006, ?1007, ?1015) to prevent motion reports from leaking.
const terminalKeyboardModeReset = "\x1b[<u\x1b[>4;0m\x1b[?1l\x1b>\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1004l\x1b[?1005l\x1b[?1006l\x1b[?1007l\x1b[?1015l\x1b[?2004l"

func writeTerminalKeyboardModeReset(out io.Writer) {
	_, _ = io.WriteString(out, terminalKeyboardModeReset)
}

// GuardInteractiveTerminalKeyboardModes resets optional keyboard reporting
// before a provider TUI starts and after it exits. It is a no-op unless both
// ends are real terminals, so pipes and JSON output remain byte-clean.
func GuardInteractiveTerminalKeyboardModes(stdin, stdout *os.File) func() {
	if stdin == nil || stdout == nil || !term.IsTerminal(int(stdin.Fd())) || !term.IsTerminal(int(stdout.Fd())) {
		return func() {}
	}
	writeTerminalKeyboardModeReset(stdout)
	return func() { writeTerminalKeyboardModeReset(stdout) }
}

// ReassertInteractiveTerminalKeyboardModes repairs terminal state after an
// interrupted provider turn. Some TUIs restore their keyboard protocol just
// after consuming Ctrl+C, so reset once immediately and twice after that turn
// transition. No bytes are emitted for pipes or redirected output.
func ReassertInteractiveTerminalKeyboardModes(stdin, stdout *os.File) {
	if stdin == nil || stdout == nil || !term.IsTerminal(int(stdin.Fd())) || !term.IsTerminal(int(stdout.Fd())) {
		return
	}
	writeTerminalKeyboardModeReset(stdout)
	for _, delay := range []time.Duration{75 * time.Millisecond, 300 * time.Millisecond} {
		time.AfterFunc(delay, func() { writeTerminalKeyboardModeReset(stdout) })
	}
}

func alreadyIsolatedSecretServiceArgv(args []string) bool {
	if len(args) >= 4 && args[0] == "--" && args[1] == "/bin/sh" && args[2] == "-c" {
		return true
	}
	if len(args) >= 2 && args[0] == "--" {
		base := filepath.Base(args[1])
		if base != "sh" && base != "dbus-run-session" && args[1] != "--" {
			return true
		}
	}
	return false
}

// WrapWithIsolatedSecretService runs a provider inside an isolated credential context
// (e.g. private D-Bus session with Secret Service component on Linux).
// The original command arguments are passed as positional parameters, avoiding
// shell interpolation of provider arguments.
func WrapWithIsolatedSecretService(bin string, args []string) (string, []string) {
	return DefaultCredentialIsolator().WrapCommand(bin, args)
}

// WrapWithIsolatedSecretServiceChecked refuses an explicit isolation request
// when the native platform has no verified credential-isolation mechanism.
// This prevents Windows/macOS from silently degrading an opt-in security
// boundary into a passthrough launch.
func WrapWithIsolatedSecretServiceChecked(bin string, args []string) (string, []string, error) {
	isolator := DefaultCredentialIsolator()
	capability := isolator.Capability()
	if capability.Status != CredentialSupported {
		return "", nil, fmt.Errorf("credential isolation unavailable on %s (%s): %s", isolator.Platform(), capability.Mechanism, capability.Reason)
	}
	wrappedBin, wrappedArgs := isolator.WrapCommand(bin, args)
	return wrappedBin, wrappedArgs, nil
}

// LookPath searches for an executable in the system PATH and standard developer directories
// (e.g. ~/.local/bin, ~/.bun/bin, ~/.opencode/bin, ~/.cargo/bin, ~/.nvm/versions/node/*/bin).
func LookPath(name string) (string, error) {
	if resolved, err := ResolveCommand(name); err == nil {
		return resolved.ArtifactPath, nil
	}
	// 1. Standard PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	// 2. Proactive developer directories
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		candidates := []string{
			filepath.Join(home, ".local", "bin", name),
			filepath.Join(home, ".bun", "bin", name),
			filepath.Join(home, ".opencode", "bin", name),
			filepath.Join(home, ".cargo", "bin", name),
			filepath.Join(home, ".local", "share", "pnpm", name),
			filepath.Join("/usr", "local", "bin", name),
			filepath.Join("/snap", "bin", name),
		}

		for _, cand := range candidates {
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
				return cand, nil
			}
		}

		// Check NVM directories
		nvmPattern := filepath.Join(home, ".nvm", "versions", "node", "*", "bin", name)
		if matches, err := filepath.Glob(nvmPattern); err == nil && len(matches) > 0 {
			for i := len(matches) - 1; i >= 0; i-- {
				cand := matches[i]
				if fi, err := os.Stat(cand); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
					return cand, nil
				}
			}
		}
	}

	return "", exec.ErrNotFound
}

// EnhancedPATH builds a robust, prioritized and deduplicated PATH string
// including the provider binary directory, active developer toolchains (Node/NVM, Bun, Cargo, PNPM),
// user directories, and standard system paths.
func EnhancedPATH(extraDirs ...string) string {
	home, _ := os.UserHomeDir()
	var candidates []string
	candidates = append(candidates, extraDirs...)

	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, ".cargo", "bin"),
			filepath.Join(home, ".local", "share", "pnpm"),
		)

		// Include all installed NVM node versions (most recent first)
		nvmPattern := filepath.Join(home, ".nvm", "versions", "node", "*", "bin")
		if matches, err := filepath.Glob(nvmPattern); err == nil && len(matches) > 0 {
			for i := len(matches) - 1; i >= 0; i-- {
				candidates = append(candidates, matches[i])
			}
		}

		// Also check fnm and asdf shims
		candidates = append(candidates,
			filepath.Join(home, ".fnm", "current", "bin"),
			filepath.Join(home, ".asdf", "shims"),
		)
	}

	candidates = append(candidates,
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
		"/usr/local/sbin",
		"/usr/sbin",
		"/sbin",
	)

	// Append existing PATH
	existing := os.Getenv("PATH")
	if existing != "" {
		candidates = append(candidates, strings.Split(existing, string(os.PathListSeparator))...)
	}

	// Deduplicate and verify directory exists on disk
	seen := make(map[string]bool)
	var finalDirs []string
	for _, dir := range candidates {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			finalDirs = append(finalDirs, dir)
		}
	}

	return strings.Join(finalDirs, string(os.PathListSeparator))
}

// EnvSet applies environment overrides and removes unset keys from the base environment slice.
func EnvSet(base []string, overrides map[string]string, unset ...string) []string {
	unsetMap := make(map[string]bool)
	for _, k := range unset {
		unsetMap[envKeyForGOOS(k, stdruntime.GOOS)] = true
	}

	overrideKeys := make(map[string]bool)
	for k := range overrides {
		overrideKeys[envKeyForGOOS(k, stdruntime.GOOS)] = true
	}

	var out []string
	seen := make(map[string]bool)
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		key := envKeyForGOOS(parts[0], stdruntime.GOOS)
		if unsetMap[key] || overrideKeys[key] || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, entry)
	}

	for k, v := range overrides {
		out = append(out, k+"="+v)
	}
	return out
}

func envKeyForGOOS(key, goos string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

// DisableSessionSecretService prevents non-interactive provider probes from
// reaching (or autolaunching) the user's desktop Secret Service. Merely
// removing DBUS_SESSION_BUS_ADDRESS is insufficient: D-Bus clients may then
// discover and connect to the user's session bus via autolaunch.
func DisableSessionSecretService(env []string) []string {
	return EnvSet(env, map[string]string{
		// A present, deliberately unreachable address disables D-Bus autolaunch.
		// /dev/null cannot become a Unix socket, so this fails closed on Unix;
		// providers on other platforms simply ignore the variable.
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=/dev/null",
	}, "AI_HOST_DBUS_SESSION_BUS_ADDRESS", "GNOME_KEYRING_CONTROL", "GNOME_KEYRING_PID")
}

// RunInteractive executes an external CLI in full interactive TTY passthrough mode.
func RunInteractive(bin string, args []string, env []string, cwd string) (model.Failure, error) {
	defer GuardInteractiveTerminalKeyboardModes(os.Stdin, os.Stdout)()
	resolved, err := ResolveCommand(bin)
	if err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	cmd := exec.Command(resolved.LauncherPath, append(resolved.PrefixArgs, args...)...)
	// Some embedded/browser terminals expose TERM=dumb even though the child
	// still has an interactive stdin. Modern provider CLIs interpret that value
	// as an unsafe non-TUI terminal and stop for a confirmation prompt whose
	// input cannot be rendered correctly. Give the interactive child a real
	// terminal type while preserving all other caller-provided environment.
	cmd.Env = NormalizeInteractiveEnv(env)
	cmd.Dir = cwd
	executil.HideWindow(cmd)
	cmd.Stdin = os.Stdin

	// Direct provider launches own the caller's TTY just as supervised attach
	// does. Reset private keyboard modes before Start and restore the captured
	// terminal state on every exit path, including provider crashes.
	tty := controlterminal.NewTTYController(os.Stdin, os.Stdout)
	if err := tty.Enter(); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	defer func() { _ = tty.Restore() }()

	var errBuf bytes.Buffer
	output := NewTerminalOutputScrubber(os.Stdout)
	defer func() { _ = output.Flush() }()
	cmd.Stdout = output
	cmd.Stderr = io.MultiWriter(output, &errBuf)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigChan)
	signalDone := make(chan struct{})
	defer close(signalDone)

	if err := cmd.Start(); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}

	go func() {
		for {
			select {
			case <-signalDone:
				return
			case sig := <-sigChan:
				if sig == os.Interrupt {
					_ = tty.Reassert()
				}
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
				if sig == syscall.SIGINT {
					ReassertInteractiveTerminalKeyboardModes(os.Stdin, os.Stdout)
				}
			}
		}
	}()

	err = cmd.Wait()
	// A provider can leave private modes enabled even after a normal exit or a
	// SIGINT-mediated cancellation. Reassert before deferred termios restore so
	// the outer shell always receives a safe terminal.
	_ = tty.Reassert()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fail := classifier.Classify(err, errBuf.String())
			return fail, err
		}
		return model.Failure{Kind: model.FailureUnknown, Message: err.Error()}, err
	}

	return model.Failure{Kind: model.FailureNone}, nil
}

// NormalizeInteractiveEnv prevents provider TUIs from entering their
// TERM=dumb safety prompt when Nexus is itself running inside an interactive
// PTY exposed by the web terminal or a wrapper shell.
func NormalizeInteractiveEnv(env []string) []string {
	for i, entry := range env {
		if entry == "TERM=dumb" {
			copyEnv := append([]string(nil), env...)
			copyEnv[i] = "TERM=xterm-256color"
			return copyEnv
		}
	}
	return env
}

// RunCommandCapture executes a command non-interactively and captures its combined stdout and stderr.
func RunCommandCapture(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
	resolved, err := ResolveCommand(bin)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, resolved.LauncherPath, append(resolved.PrefixArgs, args...)...)
	cmd.Env = env
	cmd.Dir = cwd
	executil.HideConsoleWindow(cmd)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	return buf.String(), err
}

// InternalBinDir ensures helper binaries (such as ai-browser) exist in a runtime dir.
func InternalBinDir() (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	binDir := filepath.Join(dataDir, "runtime", "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		return "", err
	}

	selfExe, err := os.Executable()
	if err != nil {
		return binDir, nil
	}

	dst := BrowserHelperPath(binDir)
	_ = os.Remove(dst)
	if stdruntime.GOOS == "windows" {
		// Never hard-link the running Nexus/test executable on Windows. Windows
		// keeps executing images delete-protected, so a hard link makes the helper
		// impossible for TempDir/test cleanup to remove while the parent process is
		// alive. Use an independent copy instead.
		in, openErr := os.Open(selfExe)
		if openErr != nil {
			return "", openErr
		}
		defer in.Close()
		out, createErr := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
		if createErr != nil {
			return "", createErr
		}
		if _, copyErr := io.Copy(out, in); copyErr != nil {
			_ = out.Close()
			return "", copyErr
		}
		if closeErr := out.Close(); closeErr != nil {
			return "", closeErr
		}
	} else if err := os.Symlink(selfExe, dst); err != nil {
		return "", err
	}

	return binDir, nil
}

// BrowserHelperPath returns the executable path providers should place in BROWSER.
func BrowserHelperPath(binDir string) string {
	name := "ai-browser"
	if stdruntime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir, name)
}

// NoBrowserBinDir returns a directory of no-op browser helpers for background
// provider probes. AGY ignores BROWSER=false in some OAuth fallbacks and calls
// xdg-open directly; putting inert shims first on PATH keeps those probes from
// opening Google login tabs every monitor tick.
func NoBrowserBinDir() (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	binDir := filepath.Join(dataDir, "runtime", "no-browser")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		return "", err
	}

	names := []string{"ai-browser", "xdg-open", "x-www-browser", "gnome-www-browser", "www-browser", "open"}
	if stdruntime.GOOS == "windows" {
		script := "@echo off\r\nrem Nexus no-browser shim: refuse background browser opens.\r\nexit /b 0\r\n"
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(binDir, name+".cmd"), []byte(script), 0600); err != nil {
				return "", err
			}
		}
		return binDir, nil
	}

	script := "#!/bin/sh\n# Nexus no-browser shim: refuse to open URLs from background probes.\nexit 0\n"
	for _, name := range names {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte(script), 0700); err != nil {
			return "", err
		}
	}
	return binDir, nil
}

// NoBrowserHelperPath returns the inert helper providers should place in BROWSER.
func NoBrowserHelperPath(binDir string) string {
	name := "ai-browser"
	if stdruntime.GOOS == "windows" {
		name += ".cmd"
	}
	return filepath.Join(binDir, name)
}
