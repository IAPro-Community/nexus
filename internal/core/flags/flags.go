// Package flags holds kill-switches for the highest-risk surfaces.
//
// Only a handful of flags exist on purpose: tunnel, autonomous missions,
// remote access, experimental intelligence, and the experimental Nexus HUD. Each defaults to its safe
// production value and can be forced off via environment for incident
// response. A future remote-config backend implements Source without
// changing call sites; until then the local source is authoritative and
// the app runs fully offline.
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
	// HUD gates the experimental Nexus HUD projection and presentation surfaces.
	HUD Name = "hud"
)

// defaults are the safe production values.
var defaults = map[Name]bool{
	Tunnel:                   true,
	AutonomousMission:        true,
	RemoteAccess:             true,
	ExperimentalIntelligence: false,
	HUD:                      false,
}

// All lists every known flag.
func All() []Name {
	return []Name{Tunnel, AutonomousMission, RemoteAccess, ExperimentalIntelligence, HUD}
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
