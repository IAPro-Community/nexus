package spec

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/runtime"
)

// QuotaSource describes how a provider obtains attributable usage data.
type QuotaSource string
type IsolationGuarantee string

const (
	IsolationUnverified             IsolationGuarantee = "UNVERIFIED"
	IsolationWorkspaceWriteNoNetwork IsolationGuarantee = "WORKSPACE_WRITE_NO_NETWORK"
	IsolationReadOnlyNoNetwork       IsolationGuarantee = "READ_ONLY_NO_NETWORK"

	QuotaLiveHTTP    QuotaSource = "LIVE_HTTP"
	QuotaLiveRPC     QuotaSource = "LIVE_RPC"
	QuotaLiveCLI     QuotaSource = "LIVE_CLI"
	QuotaUnsupported QuotaSource = "UNSUPPORTED"
)

// FlagTable maps canonical Nexus aliases to provider-native flags.
type FlagTable map[string]CanonicalAlias

// Spec is the single source of truth for a provider's binary, env, flags and
// mission launch contracts. Adapters, drivers and mission execution must derive
// from Spec instead of maintaining parallel tables.
type Spec struct {
	ID               string
	BinaryCandidates []string
	EnvKeys          func(profileHome, binDir, profileName string) map[string]string
	EnvDelete        []string
	Flags            FlagTable
	Kickoff          func(prompt string) []string
	Resume           func(sessionID string) []string
	AutoApproval     []string
	ReviewMode          []string
	AutonomousIsolation IsolationGuarantee
	ReviewIsolation     IsolationGuarantee
	// ClearLine is the provider-native control sequence used before a prompt
	// promoted from an interactive terminal is replaced by Nexus' banner.
	ClearLine     string
	Quota         QuotaSource
	KnownVersions []string
}

var registry = map[string]Spec{
	"agy":      agySpec,
	"claude":   claudeSpec,
	"codex":    codexSpec,
	"cursor":   cursorSpec,
	"gemini":   geminiSpec,
	"opencode": opencodeSpec,
}

var orderedIDs = []string{"agy", "claude", "codex", "cursor", "gemini", "opencode"}

func (s Spec) AutonomousIsolationLevel() IsolationGuarantee {
	if s.AutonomousIsolation == "" {
		return IsolationUnverified
	}
	return s.AutonomousIsolation
}

func (s Spec) ReviewIsolationLevel() IsolationGuarantee {
	if s.ReviewIsolation == "" {
		return IsolationUnverified
	}
	return s.ReviewIsolation
}

// Get returns the Spec for a provider id.
func Get(id string) (Spec, bool) {
	value, ok := registry[strings.ToLower(strings.TrimSpace(id))]
	return value, ok
}

// MustGet returns the Spec or panics.
func MustGet(id string) Spec {
	value, ok := Get(id)
	if !ok {
		panic(fmt.Sprintf("unknown provider spec %q", id))
	}
	return value
}

// All returns Specs in stable order.
func All() []Spec {
	result := make([]Spec, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		result = append(result, registry[id])
	}
	return result
}

// BinaryLookPath resolves provider binaries. It defaults to the
// PATH-plus-developer-dirs lookup and is a variable (not a direct call) so
// tests can stub binary presence without depending on host state.
var BinaryLookPath = runtime.LookPath

// ResolveBinary tries BinaryCandidates via BinaryLookPath.
func (s Spec) ResolveBinary() (string, error) {
	for _, candidate := range s.BinaryCandidates {
		if bin, err := BinaryLookPath(candidate); err == nil {
			return bin, nil
		}
	}
	return "", fmt.Errorf("%s binary not found (tried %s)", s.ID, strings.Join(s.BinaryCandidates, ", "))
}

// OverlayEnv applies Spec env keys and deletions onto base.
func (s Spec) OverlayEnv(base []string, profileHome, binDir, profileName string) []string {
	if s.EnvKeys == nil {
		return append([]string(nil), base...)
	}
	return overlayEnv(base, s.EnvKeys(profileHome, binDir, profileName), s.EnvDelete)
}

// NormalizeFlags maps canonical aliases for this provider.
func (s Spec) NormalizeFlags(args []string) []string {
	return Normalize(s.ID, args, nil)
}
