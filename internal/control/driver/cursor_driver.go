package driver

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// CursorDriver controls the Cursor CLI Agent (cursor-agent / agent).
type CursorDriver struct{}

// NewCursorDriver creates a new CursorDriver.
func NewCursorDriver() *CursorDriver { return &CursorDriver{} }

// ProviderID returns "cursor".
func (d *CursorDriver) ProviderID() string { return "cursor" }

// Detect checks if cursor-agent or agent is installed in PATH or developer paths.
func (d *CursorDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	return detectProviderCLI(ctx, "cursor", "cursor-agent binary not found in PATH", "--version")
}

// EffectiveCaps returns the live capability matrix for Cursor.
func (d *CursorDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
	det, _ := d.Detect(ctx)
	version := det.Version

	status := CapabilitySupported
	if !det.Installed {
		status = CapabilityUnsupported
	}

	return EffectiveCapabilities{
		Process: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "os/exec",
			Tested:          true,
		},
		Terminal: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "PTY / ConPTY TerminalBackend",
			Tested:          true,
		},
		Attach: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "Nexus Control IPC Socket/Pipe",
			Tested:          true,
		},
		StructuredEvents: CapabilityEvidence{
			Status:          CapabilityUnsupported,
			ProviderVersion: version,
			Reason:          "Cursor CLI hooks not wired to daemon; running in TERMINAL mode",
			Tested:          false,
		},
		Sessions: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "cursor-agent --resume / local state",
			Tested:          true,
		},
		Resume: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "cursor-agent --resume <id>",
			Tested:          true,
		},
		Fork: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		SubmitPrompt: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "cursor-agent <prompt>",
			Tested:          true,
		},
		CancelTurn: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "SIGINT (Ctrl+C) via PTY",
			Tested:          true,
		},
		Approvals: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "cursor-agent terminal prompts",
			Tested:          true,
		},
		NativeUIAttach: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		Headless: CapabilityEvidence{
			Status:          status,
			ProviderVersion: version,
			Mechanism:       "cursor-agent -p / --print",
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

func (d *CursorDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}

func (d *CursorDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	s := spec.MustGet("cursor")
	bin, err := s.ResolveBinary()
	if err != nil {
		return "", nil, nil, err
	}

	home, err := config.ProfileHome("cursor", p.Name)
	if err != nil {
		return "", nil, nil, err
	}
	_ = os.MkdirAll(home, 0700)

	cfgObj, _ := config.LoadConfig()
	_ = security.ApplyIsolation(home, security.GetPolicy(cfgObj.IsolationPreset))

	env := s.OverlayEnv(os.Environ(), home, filepath.Dir(bin), p.Name)

	return bin, extraArgs, env, nil
}

func (d *CursorDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *CursorDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) != "" {
		return spec.MustGet("cursor").Resume(providerSessionID), nil
	}
	return []string{"--continue"}, nil
}

func (d *CursorDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return spec.MustGet("cursor").Kickoff(kickoffPrompt), nil
}
