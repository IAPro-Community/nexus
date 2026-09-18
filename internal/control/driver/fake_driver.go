package driver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

type FakeDriver struct {
	CustomCaps *EffectiveCapabilities
}

func NewFakeDriver() *FakeDriver { return &FakeDriver{} }

func (d *FakeDriver) ProviderID() string { return "fake" }

func (d *FakeDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: "/bin/sh",
		Version:    "1.0.0-fake",
	}, nil
}

func (d *FakeDriver) EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities {
	if d.CustomCaps != nil {
		return *d.CustomCaps
	}

	return EffectiveCapabilities{
		Process: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: "1.0.0-fake",
			Mechanism:       "os/exec",
			Tested:          true,
		},
		Terminal: CapabilityEvidence{
			Status:          CapabilitySupported,
			ProviderVersion: "1.0.0-fake",
			Mechanism:       "PTY / ConPTY",
			Tested:          true,
		},
		Attach: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "IPC",
			Tested:    true,
		},
		StructuredEvents: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		Sessions: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "in-memory session map",
			Tested:    true,
		},
		Resume: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "fake resume <session-id>",
			Tested:    true,
		},
		Fork: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		SubmitPrompt: CapabilityEvidence{
			Status: CapabilitySupported,
			Tested: true,
		},
		CancelTurn: CapabilityEvidence{
			Status:    CapabilitySupported,
			Mechanism: "Ctrl+C",
			Tested:    true,
		},
		Approvals: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		NativeUIAttach: CapabilityEvidence{
			Status: CapabilityUnsupported,
			Tested: false,
		},
		Headless: CapabilityEvidence{
			Status: CapabilitySupported,
			Tested: true,
		},
		SlashControl: CapabilityEvidence{
			Status: CapabilitySupported,
			Tested: true,
		},
		ControlLevel: registry.ControlLevelTerminal,
	}
}

func (d *FakeDriver) Capabilities(ctx context.Context, p model.Profile) ControlCapabilities {
	return d.EffectiveCaps(ctx, p).ToBooleanCaps()
}

func (d *FakeDriver) BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (string, []string, []string, error) {
	// Fake driver runs a simple interactive shell or echo command
	bin := "cat"
	args := []string{}
	if len(extraArgs) > 0 {
		args = extraArgs
	}
	return bin, args, os.Environ(), nil
}

func (d *FakeDriver) CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string) {
	if strings.TrimSpace(providerSessionID) == "" {
		return false, "Provider session ID is empty or unknown"
	}
	return true, ""
}

func (d *FakeDriver) BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("cannot resume fake session without valid session ID")
	}
	return []string{"resume", providerSessionID}, nil
}

func (d *FakeDriver) BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error) {
	return []string{kickoffPrompt}, nil
}
