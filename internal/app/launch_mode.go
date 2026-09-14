package app

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// LaunchMode represents how a provider should be launched.
type LaunchMode int

const (
	// LaunchModeDirect runs the provider binary directly without Nexus
	// supervision. This is the traditional path for scripts, CI, pipes,
	// headless usage, and explicit --direct.
	LaunchModeDirect LaunchMode = iota

	// LaunchModeSupervised launches the provider through the SessionHost,
	// enabling slash control, detach/attach, handoff, generation tracking,
	// and the full Nexus control plane. This is the default for TTY.
	LaunchModeSupervised
)

func (m LaunchMode) String() string {
	switch m {
	case LaunchModeDirect:
		return "direct"
	case LaunchModeSupervised:
		return "supervised"
	default:
		return "unknown"
	}
}

// LaunchModeResult contains the resolved launch mode plus diagnostic info.
type LaunchModeResult struct {
	Mode   LaunchMode
	Reason string
	Source string
}

// LaunchModeInput captures all signals needed to resolve the launch mode.
type LaunchModeInput struct {
	Args       []string
	StdinIsTTY bool
	Provider   string // optional; used so AGY/Claude -p means --print
}

// LaunchModeResolver centralizes the decision of whether a provider launch
// should be DIRECT or SUPERVISED. The default heuristic:
//
// Explicit --supervised → SUPERVISED
//
//	Explicit --direct    → DIRECT
//	--print              → DIRECT (headless single-shot)
//	TTY interactive      → SUPERVISED (new default)
//	Piped / CI / non-TTY → DIRECT (preserves old behavior)
//
// An explicit NEXUS_LAUNCH_MODE env var overrides all heuristics.
type LaunchModeResolver struct {
	input LaunchModeInput
}

// NewLaunchModeResolver creates a resolver for the given input.
func NewLaunchModeResolver(input LaunchModeInput) *LaunchModeResolver {
	return &LaunchModeResolver{input: input}
}

// Resolve returns the launch mode without diagnostic info.
func (r *LaunchModeResolver) Resolve() LaunchMode {
	return r.ResolveWithReason().Mode
}

// ResolveWithReason returns the launch mode with human-readable explanation.
func (r *LaunchModeResolver) ResolveWithReason() LaunchModeResult {
	// 1. Environment override (highest priority)
	if envMode := strings.ToLower(strings.TrimSpace(os.Getenv("NEXUS_LAUNCH_MODE"))); envMode != "" {
		switch envMode {
		case "supervised":
			return LaunchModeResult{Mode: LaunchModeSupervised, Reason: "NEXUS_LAUNCH_MODE=supervised override", Source: "env"}
		case "direct":
			return LaunchModeResult{Mode: LaunchModeDirect, Reason: "NEXUS_LAUNCH_MODE=direct override", Source: "env"}
		}
		// Unknown env value — fall through to normal resolution
	}

	// 2. Explicit --supervised flag → SUPERVISED
	if hasProviderLaunchFlag(r.input.Args, "--supervised") {
		return LaunchModeResult{Mode: LaunchModeSupervised, Reason: "explicit --supervised flag", Source: "explicit-supervised"}
	}

	// 3. Explicit --direct flag → DIRECT
	if hasProviderLaunchFlag(r.input.Args, "--direct") {
		return LaunchModeResult{Mode: LaunchModeDirect, Reason: "explicit --direct flag", Source: "explicit-direct"}
	}

	// 4. --print flag → DIRECT (headless single-shot, never supervise)
	if hasProviderLaunchFlag(r.input.Args, "--print") {
		return LaunchModeResult{Mode: LaunchModeDirect, Reason: "--print flag implies non-interactive", Source: "print-flag"}
	}
	// AGY/Claude short -p is --print; Codex -p is native --profile and must not force direct.
	prov := strings.ToLower(strings.TrimSpace(r.input.Provider))
	if (prov == "agy" || prov == "claude") && hasProviderLaunchFlag(r.input.Args, "-p") {
		return LaunchModeResult{Mode: LaunchModeDirect, Reason: "provider -p alias for --print", Source: "print-flag"}
	}

	// 5. TTY heuristic: interactive terminal → SUPERVISED by default
	if r.input.StdinIsTTY {
		return LaunchModeResult{Mode: LaunchModeSupervised, Reason: "interactive TTY detected, supervised by default", Source: "tty-default"}
	}

	// 6. Non-interactive / piped / CI → DIRECT (preserves existing behavior)
	return LaunchModeResult{Mode: LaunchModeDirect, Reason: "non-interactive (piped/CI/headless)", Source: "non-interactive"}
}

// StripControlFlags removes Nexus-owned launch flags from args, returning
// only flags/arguments that should be forwarded to the provider.
func (r *LaunchModeResolver) StripControlFlags() []string {
	args := removeProviderLaunchFlag(r.input.Args, "--direct")
	args = removeProviderLaunchFlag(args, "--supervised")
	args = removeProviderLaunchFlag(args, "--lead")
	return args
}

// WantInteractiveLead reports whether the caller opted into Lead Agent
// binding and Mission promotion on the supervised TTY attach path.
func (r *LaunchModeResolver) WantInteractiveLead() bool {
	return wantInteractiveLead(r.input.Args)
}

// IsTerminal reports whether the current process stdin is a terminal.
func IsTerminal() bool {
	return isTerminalFD(int(os.Stdin.Fd()))
}

func isTerminalFD(fd int) bool {
	return term.IsTerminal(fd)
}
