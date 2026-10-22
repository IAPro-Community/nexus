// Package flags holds kill-switches for the highest-risk surfaces.
//
// Stable v1 surfaces keep their existing defaults. Nexus 1.1 capabilities are
// introduced behind explicit, independently reversible flags so a beta can
// disable one subsystem without changing the v1 execution path. Every new v1.1
// capability defaults off until its own evidence gate is satisfied.
package flags

import (
	"os"
	"strings"
)

// Name identifies a kill-switchable surface.
type Name string

const (
	// Tunnel gates outbound tunnel / public-URL exposure.
	Tunnel Name = "tunnel"
	// AutonomousMission gates unsupervised mission execution.
	AutonomousMission Name = "autonomous_mission"
	// RemoteAccess gates remote browser/terminal access.
	RemoteAccess Name = "remote_access"
	// ExperimentalIntelligence gates experimental intelligence features.
	ExperimentalIntelligence Name = "experimental_intelligence"
	// HUD gates the read-only HUD projection API; Web HUD visibility remains opt-in.
	HUD Name = "hud"
	// ExecutionWorkspacesV2 gates task-scoped worktrees and execution workspaces.
	ExecutionWorkspacesV2 Name = "execution_workspaces_v2"
	// ReviewCenter gates the source-control and evidence review surface.
	ReviewCenter Name = "review_center"
	// BrowserDesignMode gates browser element inspection and UI context artifacts.
	BrowserDesignMode Name = "browser_design_mode"
	// CandidateRuns gates speculative multi-agent candidate execution.
	CandidateRuns Name = "candidate_runs"
	// RemoteExecution gates execution on SSH/remote targets.
	RemoteExecution Name = "remote_execution"
	// NexusNodes gates enrolled Nexus worker nodes.
	NexusNodes Name = "nexus_nodes"
	// ComputerUse gates desktop observation and input capabilities.
	ComputerUse Name = "computer_use"
	// MobileCompanion gates remote companion surfaces.
	MobileCompanion Name = "mobile_companion"
)

// defaults are the safe production values.
var defaults = map[Name]bool{
	Tunnel:                   true,
	AutonomousMission:        true,
	RemoteAccess:             true,
	ExperimentalIntelligence: false,
	HUD:                      true,
	ExecutionWorkspacesV2:    false,
	ReviewCenter:             false,
	BrowserDesignMode:        false,
	CandidateRuns:            false,
	RemoteExecution:          false,
	NexusNodes:               false,
	ComputerUse:              false,
	MobileCompanion:          false,
}

// All lists every known flag.
func All() []Name {
	return []Name{
		Tunnel,
		AutonomousMission,
		RemoteAccess,
		ExperimentalIntelligence,
		HUD,
		ExecutionWorkspacesV2,
		ReviewCenter,
		BrowserDesignMode,
		CandidateRuns,
		RemoteExecution,
		NexusNodes,
		ComputerUse,
		MobileCompanion,
	}
}

// Source resolves flag values.
type Source interface {
	Enabled(n Name) bool
}

// LocalSource resolves defaults with NEXUS_DISABLE_<NAME> env overrides
// (1/true/yes, case-insensitive, disable; NEXUS_ENABLE_<NAME> to force on).
type LocalSource struct{}

// envKey maps a flag to its override variable stem.
func envKey(n Name, prefix string) string {
	return prefix + strings.ToUpper(strings.ReplaceAll(string(n), "-", "_"))
}

// Enabled implements Source.
func (LocalSource) Enabled(n Name) bool {
	def, ok := defaults[n]
	if !ok {
		return false
	}
	if v := strings.TrimSpace(os.Getenv(envKey(n, "NEXUS_DISABLE_"))); isTruthy(v) {
		return false
	}
	if v := strings.TrimSpace(os.Getenv(envKey(n, "NEXUS_ENABLE_"))); isTruthy(v) {
		return true
	}
	return def
}

func isTruthy(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Enabled is the package-level shortcut over LocalSource.
func Enabled(n Name) bool { return LocalSource{}.Enabled(n) }
