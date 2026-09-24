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

type OpenCodeDriver struct{}

func NewOpenCodeDriver() *OpenCodeDriver { return &OpenCodeDriver{} }

func (d *OpenCodeDriver) ProviderID() string { return "opencode" }

func (d *OpenCodeDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, err := spec.MustGet("opencode").ResolveBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "opencode binary not found in PATH"}, nil
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

func (d *OpenCodeDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
	det, _ := d.Detect(ctx)
	version := det.Version

	// OpenCode enforces read-only through per-agent permission config. The
	// current terminal/headless adapter does not materialize such an agent yet,
	// so direct read-only review is intentionally not advertised.
	readOnlyReviewStatus := CapabilityUnsupported
	readOnlyReviewReason := "read-only review requires a permission-scoped OpenCode agent; Agent Projection is not implemented yet"

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
			Mechanism:       terminal.BackendMechanism(),
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
			Mechanism:       "opencode serve HTTP/events adapter",
			Reason:          "Server adapter disabled; running in truthful TERMINAL mode",
			Tested:          false,
		},
		Sessions: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "opencode session list / database",
			Tested:          true,
		},
		Resume: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "opencode -s <session-id>",
			Reason:          "resume command supported by signature; not runtime-verified against a live opencode session",
			Tested:          false,
		},
		Fork: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "opencode --fork",
			Tested:          true,
		},
		SubmitPrompt: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "opencode run [message..] / --prompt",
			Tested:          true,
		},
		CancelTurn: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Ctrl+C signal passthrough",
			Tested:    true,
		},
		Approvals: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Reason: "Structured approvals require server adapter; interact via terminal",
			Tested: false,
		},
		NativeUIAttach: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "opencode attach <url>",
			Tested:          false,
		},
		Headless: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: version,
			Mechanism:       "opencode serve / opencode run",
			Tested:          true,
		},
		SlashControl: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Universal /ai slash command router",
			Tested:    true,
		},
		ReadOnlyReview: CapabilityEvidence{
			Status:          readOnlyReviewStatus,
			ProviderVersion: version,
			Mechanism:       "OpenCode per-agent permission.edit=deny (not projected by Nexus yet)",
			Reason:          readOnlyReviewReason,
			Tested:          false,
		},
		ControlLevel: registry.ControlLevelTerminal,
	}
}

func (d *OpenCodeDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}

func (d *OpenCodeDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	s := spec.MustGet("opencode")
	bin, err := s.ResolveBinary()
	if err != nil {
		return "", nil, nil, err
	}

	home, err := bootstrapProfile("opencode", p)
	if err != nil {
		return "", nil, nil, err
	}
	env := s.OverlayEnv(os.Environ(), home, filepath.Dir(bin), p.Name)

	return bin, extraArgs, env, nil
}

func (d *OpenCodeDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *OpenCodeDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("cannot resume opencode session without valid session ID")
	}
	return spec.MustGet("opencode").Resume(providerSessionID), nil
}

func (d *OpenCodeDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return spec.MustGet("opencode").Kickoff(kickoffPrompt), nil
}

