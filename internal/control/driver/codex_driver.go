package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/control/terminal"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

type CodexDriver struct{}

func NewCodexDriver() *CodexDriver { return &CodexDriver{} }

func (d *CodexDriver) ProviderID() string { return "codex" }

func (d *CodexDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := spec.MustGet("codex").ResolveBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "codex binary not found in PATH"}, nil
	}
	out, err := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	if err != nil || strings.Contains(out, "No such file or directory") {
		return model.DetectionResult{
			Installed:  false,
			BinaryPath: bin,
			Error:      strings.TrimSpace(out),
		}, nil
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    lines[0],
	}, nil
}

func (d *CodexDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
	det, _ := d.Detect(ctx)
	version := det.Version

	return EffectiveCapabilities{
		Process: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "os/exec",
			Tested:          true,
		},
		Terminal: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       terminal.BackendMechanismForProvider(d.ProviderID()),
			Tested:          true,
		},
		Attach: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "Nexus Control IPC Socket/Pipe",
			Tested:          true,
		},
		StructuredEvents: CapabilityEvidence{
			Status:          CapabilityUnsupported,
			ProviderVersion: version,
			Mechanism:       "codex app-server adapter",
			Reason:          "Structured app-server adapter deferred; see DEV/AI_CONTROL_DEFERRED.md. Running in truthful TERMINAL mode with PTY heuristic fallback only",
			Tested:          false,
		},
		Sessions: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "~/.codex/sessions index",
			Tested:          true,
		},
		Resume: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "codex resume <session-id>",
			Reason:          "resume command supported by signature; not runtime-verified against a live codex session",
			Tested:          false,
		},
		Fork: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Codex CLI does not support session branching/forking",
			Tested: false,
		},
		SubmitPrompt: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "codex exec <prompt>",
			Reason:          "Codex exec provides a verified non-interactive prompt contract",
			Tested:          true,
		},
		CancelTurn: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Ctrl+C signal passthrough",
			Tested:    true,
		},
		Approvals: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Structured approvals require Codex app-server adapter (deferred; DEV/AI_CONTROL_DEFERRED.md); interact via terminal prompt",
			Tested: false,
		},
		NativeUIAttach: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Native GUI attach not implemented for Codex",
			Tested: false,
		},
		Headless: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "codex exec <prompt>",
			Reason:          "Codex exec is available for non-interactive prompts",
			Tested:          true,
		},
		SlashControl: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Universal /ai slash command router",
			Tested:    true,
		},
		ControlLevel: registry.ControlLevelTerminal,
	}
}

func (d *CodexDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}

func (d *CodexDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	s := spec.MustGet("codex")
	bin, err := s.ResolveBinary()
	if err != nil {
		return "", nil, nil, err
	}

	home, err := bootstrapProfile("codex", p)
	if err != nil {
		return "", nil, nil, err
	}
	env := s.OverlayEnv(os.Environ(), home, filepath.Dir(bin), p.Name)
	extraArgs = spec.CodexInteractiveArgs(home, extraArgs)

	return bin, extraArgs, env, nil
}

func (d *CodexDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *CodexDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("cannot resume codex session without valid session ID")
	}
	return spec.MustGet("codex").Resume(providerSessionID), nil
}

func (d *CodexDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return spec.MustGet("codex").Kickoff(kickoffPrompt), nil
}
