package driver

import (
	"context"
	"time"

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

// IsSupportedAndTested reports whether this evidence is safe to project as an
// enabled capability in consumers that cannot preserve the full evidence.
func (e CapabilityEvidence) IsSupportedAndTested() bool {
	return e.Status == CapabilitySupported && e.Tested
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
	ReadOnlyReview   CapabilityEvidence    `json:"read_only_review"`
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
	ReadOnlyReview   bool                  `json:"read_only_review"`
	ControlLevel     registry.ControlLevel `json:"control_level"`
}

// ToBooleanCaps converts EffectiveCapabilities to legacy ControlCapabilities.
func (ec EffectiveCapabilities) ToBooleanCaps() ControlCapabilities {
	// The legacy boolean contract has no field for confidence or verification.
	// Only project a capability as enabled when its evidence is both supported
	// and tested; richer clients can inspect EffectiveCapabilities directly.
	return ControlCapabilities{
		Process:          ec.Process.IsSupportedAndTested(),
		Terminal:         ec.Terminal.IsSupportedAndTested(),
		Attach:           ec.Attach.IsSupportedAndTested(),
		StructuredEvents: ec.StructuredEvents.IsSupportedAndTested(),
		Sessions:         ec.Sessions.IsSupportedAndTested(),
		Resume:           ec.Resume.IsSupportedAndTested(),
		Fork:             ec.Fork.IsSupportedAndTested(),
		SubmitPrompt:     ec.SubmitPrompt.IsSupportedAndTested(),
		CancelTurn:       ec.CancelTurn.IsSupportedAndTested(),
		Approvals:        ec.Approvals.IsSupportedAndTested(),
		NativeUIAttach:   ec.NativeUIAttach.IsSupportedAndTested(),
		Headless:         ec.Headless.IsSupportedAndTested(),
		SlashControl:     ec.SlashControl.IsSupportedAndTested(),
		ReadOnlyReview:   ec.ReadOnlyReview.IsSupportedAndTested(),
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

// SessionIDDiscoverer is an optional driver capability for supervised terminal
// runtimes that cannot report a provider session id over a structured channel.
// It probes the provider's own session store for the live session id of a
// runtime. Implementations must be strict read-only observers: no PTY writes,
// no profile mutations, no provider commands, and no contention with an
// interactive TUI (no locks, no probes that race the child process).
type SessionIDDiscoverer interface {
	DiscoverSessionID(ctx context.Context, p model.Profile, workspace string, since time.Time) (string, error)
}
