package driver

import (
	"context"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

type CapabilityStatus string

const (
	CapabilitySupported   CapabilityStatus = "SUPPORTED"
	CapabilityPartial     CapabilityStatus = "PARTIAL"
	CapabilityUnsupported CapabilityStatus = "UNSUPPORTED"
	CapabilityUnknown     CapabilityStatus = "UNKNOWN"
	CapabilityNotTested   CapabilityStatus = "NOT_TESTED"
)

// CapabilityEvidence provides truthful evidence for a specific capability.
type CapabilityEvidence struct {
	Status          CapabilityStatus `json:"status"`
	ProviderVersion string           `json:"provider_version,omitempty"`
	Mechanism       string           `json:"mechanism,omitempty"`
	Reason          string           `json:"reason,omitempty"`
	Tested          bool             `json:"tested"`
}

// EffectiveCapabilities holds truthful derived capabilities for a supervised provider driver.
type EffectiveCapabilities struct {
	Process          CapabilityEvidence    `json:"process"`
	Terminal         CapabilityEvidence    `json:"terminal"`
	Attach           CapabilityEvidence    `json:"attach"`
	StructuredEvents CapabilityEvidence    `json:"structured_events"`
	Sessions         CapabilityEvidence    `json:"sessions"`
	Resume           CapabilityEvidence    `json:"resume"`
	Fork             CapabilityEvidence    `json:"fork"`
	SubmitPrompt     CapabilityEvidence    `json:"submit_prompt"`
	CancelTurn       CapabilityEvidence    `json:"cancel_turn"`
	Approvals        CapabilityEvidence    `json:"approvals"`
	NativeUIAttach   CapabilityEvidence    `json:"native_ui_attach"`
	Headless         CapabilityEvidence    `json:"headless"`
	SlashControl     CapabilityEvidence    `json:"slash_control"`
	ControlLevel     registry.ControlLevel `json:"control_level"`
}

// ControlCapabilities is preserved as a convenient boolean view computed from EffectiveCapabilities.
type ControlCapabilities struct {
	Process          bool                  `json:"process"`
	Terminal         bool                  `json:"terminal"`
	Attach           bool                  `json:"attach"`
	StructuredEvents bool                  `json:"structured_events"`
	Sessions         bool                  `json:"sessions"`
	Resume           bool                  `json:"resume"`
	Fork             bool                  `json:"fork"`
	SubmitPrompt     bool                  `json:"submit_prompt"`
	CancelTurn       bool                  `json:"cancel_turn"`
	Approvals        bool                  `json:"approvals"`
	NativeUIAttach   bool                  `json:"native_ui_attach"`
	Headless         bool                  `json:"headless"`
	SlashControl     bool                  `json:"slash_control"`
	ControlLevel     registry.ControlLevel `json:"control_level"`
}

// ToBooleanCaps converts EffectiveCapabilities to legacy ControlCapabilities.
func (ec EffectiveCapabilities) ToBooleanCaps() ControlCapabilities {
	return ControlCapabilities{
		Process:          ec.Process.Status == CapabilitySupported,
		Terminal:         ec.Terminal.Status == CapabilitySupported,
		Attach:           ec.Attach.Status == CapabilitySupported,
		StructuredEvents: ec.StructuredEvents.Status == CapabilitySupported,
		Sessions:         ec.Sessions.Status == CapabilitySupported,
		Resume:           ec.Resume.Status == CapabilitySupported,
		Fork:             ec.Fork.Status == CapabilitySupported,
		SubmitPrompt:     ec.SubmitPrompt.Status == CapabilitySupported,
		CancelTurn:       ec.CancelTurn.Status == CapabilitySupported,
		Approvals:        ec.Approvals.Status == CapabilitySupported,
		NativeUIAttach:   ec.NativeUIAttach.Status == CapabilitySupported,
		Headless:         ec.Headless.Status == CapabilitySupported,
		SlashControl:     ec.SlashControl.Status == CapabilitySupported,
		ControlLevel:     ec.ControlLevel,
	}
}

// ControlDriver defines the interface for creating supervised runtime configurations for a provider.
type ControlDriver interface {
	ProviderID() string
	Detect(ctx context.Context) (model.DetectionResult, error)
	Capabilities(ctx context.Context, p model.Profile) ControlCapabilities
	EffectiveCaps(ctx context.Context, p model.Profile) EffectiveCapabilities
	BuildCommand(ctx context.Context, p model.Profile, extraArgs []string) (binary string, args []string, env []string, err error)
	CanResume(ctx context.Context, p model.Profile, providerSessionID string) (bool, string)
	BuildResumeArgs(ctx context.Context, p model.Profile, providerSessionID string) ([]string, error)
	BuildKickoffArgs(ctx context.Context, p model.Profile, kickoffPrompt string) ([]string, error)
}
