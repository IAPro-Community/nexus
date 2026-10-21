// Package hudlink launches the tray-resident Nexus HUD as a companion process
// of a running Nexus workspace surface (CLI Web core or Desktop shell) so the
// two share one lifecycle: started together, stopped together.
package hudlink

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/executil"
)

// SpawnEnv opts the linked HUD out entirely (servers, CI, automated harnesses).
// Unset or any value other than 0/false/off/no keeps the companion enabled.
const SpawnEnv = "NEXUS_HUD_SPAWN"

// LinkedEnv is set on the HUD process started by Spawn so the child can tell
// a linked launch apart from a manual or login-autostart launch.
const LinkedEnv = "NEXUS_HUD_LINKED"

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "nexus-hud.exe"
	}
	return "nexus-hud"
}

// Enabled reports whether the linked HUD companion is allowed.
func Enabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(SpawnEnv))) {
	case "0", "false", "off", "no", "disabled":
		return false
	}
	return true
}

// BinaryPath returns the nexus-hud executable installed next to the current
// executable, mirroring the installer layout of the workspace tools.
func BinaryPath() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	return binaryPathForExe(exe)
}

// binaryPathForExe resolves the HUD companion for a given host executable.
func binaryPathForExe(exe string) (string, bool) {
	if strings.TrimSpace(exe) == "" {
		return "", false
	}
	dir := filepath.Dir(exe)
	if path, ok := binaryPathIn(dir); ok {
		return path, true
	}
	for _, candidate := range companionDirs(dir) {
		if path, ok := binaryPathIn(candidate); ok {
			return path, true
		}
	}
	return "", false
}

// hudBundleName is the companion bundle shipped by the macOS installer next to
// the desktop app.
const hudBundleName = "IAPro Nexus HUD.app"

// companionDirs lists additional install layouts that host the HUD executable.
func companionDirs(dir string) []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	// macOS ships two sibling bundles:
	//   <parent>/IAPro Nexus.app/Contents/MacOS/nexus-desktop
	//   <parent>/IAPro Nexus HUD.app/Contents/MacOS/nexus-hud
	contents := filepath.Dir(dir)
	bundle := filepath.Dir(contents)
	if filepath.Base(dir) != "MacOS" || filepath.Base(contents) != "Contents" {
		return nil
	}
	return []string{filepath.Join(filepath.Dir(bundle), hudBundleName, "Contents", "MacOS")}
}

func binaryPathIn(dir string) (string, bool) {
	if strings.TrimSpace(dir) == "" {
		return "", false
	}
	path := filepath.Join(dir, binaryName())
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

// Linked reports whether the current process is the HUD half of a linked pair,
// either via the marker environment variable or the --linked flag.
func Linked(args []string, getenv func(string) string) bool {
	if getenv != nil {
		switch strings.ToLower(strings.TrimSpace(getenv(LinkedEnv))) {
		case "1", "true":
			return true
		}
	}
	for _, arg := range args {
		if arg == "--linked" {
			return true
		}
	}
	return false
}

func linkedCommand(path string) *exec.Cmd {
	cmd := executil.Command(path, "--linked")
	cmd.Env = append(os.Environ(), LinkedEnv+"=1")
	return cmd
}

// Process supervises a HUD process started by Spawn. Close terminates it; the
// HUD also watches its core and exits on its own when the core disappears.
type Process struct {
	cmd  *exec.Cmd
	once sync.Once
}

// Close terminates the companion HUD. It is safe to call more than once and
// after the process already exited.
func (p *Process) Close() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	p.once.Do(func() {
		_ = p.cmd.Process.Kill()
	})
}

// Spawn starts the linked HUD when it is installed next to this executable and
// not disabled. A nil Process with a nil error means "nothing to do".
func Spawn() (*Process, error) {
	if !Enabled() {
		return nil, nil
	}
	path, ok := BinaryPath()
	if !ok {
		return nil, nil
	}
	cmd := linkedCommand(path)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(path), err)
	}
	proc := &Process{cmd: cmd}
	go func() { _ = cmd.Wait() }()
	return proc, nil
}
