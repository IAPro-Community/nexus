package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

const isolatedSecretServiceScript = `TMPDIR=$(mktemp -d /tmp/nexus-kr-XXXXXX) || exit 1
trap 'rm -rf "$TMPDIR"' EXIT INT TERM
export GNOME_KEYRING_CONTROL="$TMPDIR"
daemon="$1"
shift
"$daemon" --daemonize --components=secrets --control-directory="$TMPDIR" >/dev/null 2>&1 || true
exec "$@"`

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
		unsetMap[k] = true
	}

	overrideKeys := make(map[string]bool)
	for k := range overrides {
		overrideKeys[k] = true
	}

	var out []string
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 0 {
			continue
		}
		k := parts[0]
		if unsetMap[k] || overrideKeys[k] {
			continue
		}
		out = append(out, entry)
	}

	for k, v := range overrides {
		out = append(out, k+"="+v)
	}
	return out
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
	cmd.Stdin = os.Stdin

	var errBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}

	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err = cmd.Wait()
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

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err = cmd.Run()
	return buf.String(), err
}

// InternalBinDir ensures helper binaries (such as ai-browser / xdg-open shim) exist in a runtime dir.
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

	// Create ai-browser shim (entry point for $BROWSER calls from agy/gemini).
	// Do NOT create an xdg-open symlink — it would shadow the real system
	// xdg-open in PATH and cause infinite recursion when browser.Open()
	// resolves "xdg-open" back to the nexus binary.
	dst := filepath.Join(binDir, "ai-browser")
	_ = os.Remove(dst)
	_ = os.Symlink(selfExe, dst)

	return binDir, nil
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
	script := "#!/bin/sh\n# Nexus no-browser shim: refuse to open URLs from background probes.\nexit 0\n"
	for _, name := range []string{"ai-browser", "xdg-open", "x-www-browser", "gnome-www-browser", "www-browser", "open"} {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte(script), 0700); err != nil {
			return "", err
		}
	}
	return binDir, nil
}
