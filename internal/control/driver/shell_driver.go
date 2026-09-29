package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/executil"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

// ShellDriver exposes a normal local project shell through the same supervised
// PTY/ConPTY runtime used by provider terminals. It intentionally advertises no
// AI/session/prompt capabilities.
type ShellDriver struct{}

func NewShellDriver() *ShellDriver        { return &ShellDriver{} }
func (d *ShellDriver) ProviderID() string { return "shell" }

func resolveSystemShell() (string, error) {
	if runtime.GOOS == "windows" {
		if configured := strings.TrimSpace(os.Getenv("COMSPEC")); configured != "" {
			if shellRuns(configured, "/c", "exit", "0") {
				return configured, nil
			}
		}
		for _, candidate := range []string{"pwsh.exe", "powershell.exe", "cmd.exe"} {
			if path, err := exec.LookPath(candidate); err == nil && shellRuns(path, "/c", "exit", "0") {
				return path, nil
			}
		}
		return "", fmt.Errorf("no Windows shell found")
	}
	if configured := strings.TrimSpace(os.Getenv("SHELL")); configured != "" {
		if path, err := exec.LookPath(configured); err == nil && shellRuns(path, "-c", "exit 0") {
			return path, nil
		}
	}
	for _, candidate := range []string{"bash", "zsh", "sh"} {
		if path, err := exec.LookPath(candidate); err == nil && shellRuns(path, "-c", "exit 0") {
			return path, nil
		}
	}
	return "", fmt.Errorf("no local shell found")
}

// shellRuns catches stale or non-executable SHELL values before a detached
// SessionHost is registered. A path can exist while its interpreter or loader
// is unavailable in the host environment, which otherwise surfaces later as a
// misleading IPC timeout.
func shellRuns(binary string, args ...string) bool {
	cmd := exec.Command(binary, args...)
	cmd.Env = nexusruntime.SanitizeChildEnv(os.Environ())
	executil.HideConsoleWindow(cmd)
	return cmd.Run() == nil
}

func (d *ShellDriver) Detect(context.Context) (model.DetectionResult, error) {
	binary, err := resolveSystemShell()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: err.Error()}, nil
	}
	return model.DetectionResult{Installed: true, BinaryPath: binary, Version: "system-shell"}, nil
}

func shellCap(status CapabilityStatus, mechanism string, _ bool) CapabilityEvidence {
	return CapabilityEvidence{Status: status, Mechanism: mechanism, Tested: true}
}

func (d *ShellDriver) EffectiveCaps(context.Context, model.Profile) EffectiveCapabilities {
	unsupported := shellCap(CapabilityUnsupported, "not an AI provider", true)
	return EffectiveCapabilities{
		Process:          shellCap(CapabilitySupported, "os/exec", true),
		Terminal:         shellCap(CapabilitySupported, "PTY / ConPTY", true),
		Attach:           shellCap(CapabilitySupported, "Nexus IPC", true),
		StructuredEvents: unsupported, Sessions: unsupported, Resume: unsupported, Fork: unsupported,
		SubmitPrompt: unsupported, CancelTurn: unsupported, Approvals: unsupported, NativeUIAttach: unsupported,
		Headless:     unsupported,
		SlashControl: shellCap(CapabilitySupported, "SessionHost slash control", true),
		ControlLevel: registry.ControlLevelTerminal,
	}
}
func (d *ShellDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}
func (d *ShellDriver) BuildCommand(context.Context, model.Profile, []string) (string, []string, []string, error) {
	binary, err := resolveSystemShell()
	if err != nil {
		return "", nil, nil, err
	}
	return binary, nil, nexusruntime.SanitizeChildEnv(os.Environ()), nil
}
func (d *ShellDriver) CanResume(context.Context, model.Profile, string) (bool, string) {
	return false, "project shells are not provider sessions"
}
func (d *ShellDriver) BuildResumeArgs(context.Context, model.Profile, string) ([]string, error) {
	return nil, fmt.Errorf("project shells do not support provider resume")
}
func (d *ShellDriver) BuildKickoffArgs(context.Context, model.Profile, string) ([]string, error) {
	return nil, fmt.Errorf("project shells do not support AI kickoff prompts")
}
