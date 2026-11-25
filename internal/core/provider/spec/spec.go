package spec

import (
	"fmt"
	"strings"

	sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

type QuotaSource = sdkprovider.QuotaSource
type IsolationGuarantee = sdkprovider.IsolationGuarantee
type GroupTier = sdkprovider.GroupTier
type ModelGroupSpec = sdkprovider.ModelGroupSpec
type AuthKind = sdkprovider.AuthKind
type QuotaPolicy = sdkprovider.QuotaPolicy
type HistorySyncMode = sdkprovider.HistorySyncMode
type HistorySyncPolicy = sdkprovider.HistorySyncPolicy

const (
	IsolationUnverified              = sdkprovider.IsolationUnverified
	IsolationWorkspaceWriteNoNetwork = sdkprovider.IsolationWorkspaceWriteNoNetwork
	IsolationReadOnlyNoNetwork       = sdkprovider.IsolationReadOnlyNoNetwork

	QuotaLiveHTTP    = sdkprovider.QuotaLiveHTTP
	QuotaLiveRPC     = sdkprovider.QuotaLiveRPC
	QuotaLiveCLI     = sdkprovider.QuotaLiveCLI
	QuotaUnsupported = sdkprovider.QuotaUnsupported

	GroupTierPrimary   = sdkprovider.GroupTierPrimary
	GroupTierAuxiliary = sdkprovider.GroupTierAuxiliary

	AuthKindOAuthCLI    = sdkprovider.AuthKindOAuthCLI
	AuthKindAPIKey      = sdkprovider.AuthKindAPIKey
	AuthKindHostSession = sdkprovider.AuthKindHostSession

	HistorySyncNone            = sdkprovider.HistorySyncNone
	HistorySyncLink            = sdkprovider.HistorySyncLink
	HistorySyncIncrementalCopy = sdkprovider.HistorySyncIncrementalCopy
)

// Register registers a product-specific or test provider specification in the shared SDK registry.
func Register(s Spec) error {
	return sdkprovider.Register(s.Spec)
}

// Extend updates an existing specification in the shared SDK registry.
func Extend(id string, fn func(*sdkprovider.Spec)) error {
	return sdkprovider.Extend(id, fn)
}

// Unregister removes a registered specification from the shared SDK registry.
func Unregister(id string) {
	sdkprovider.Unregister(id)
}

type FlagTable = sdkprovider.FlagTable

// Spec embeds the SDK contract. Its methods below preserve Nexus's injectable
// executable lookup and host-specific environment isolation.
type Spec struct {
	sdkprovider.Spec
}

// BinaryLookPath remains injectable for hermetic adapter tests.
var BinaryLookPath = runtime.LookPath

func adaptSDKSpec(source sdkprovider.Spec) Spec {
	originalEnv := source.EnvKeys
	source.EnvKeys = func(profileHome, binDir, profileName string) map[string]string {
		env := originalEnv(profileHome, binDir, profileName)
		if source.ID == "agy" {
			internalBin, _ := runtime.InternalBinDir()
			pathBins := make([]string, 0, 2)
			if strings.TrimSpace(internalBin) != "" {
				pathBins = append(pathBins, internalBin)
			}
			if strings.TrimSpace(binDir) != "" {
				pathBins = append(pathBins, binDir)
			}
			env["PATH"] = runtime.EnhancedPATH(pathBins...)
			if strings.TrimSpace(internalBin) != "" {
				env["BROWSER"] = runtime.BrowserHelperPath(internalBin)
			} else if strings.TrimSpace(binDir) != "" {
				env["BROWSER"] = runtime.BrowserHelperPath(binDir)
			}
		} else {
			env["PATH"] = runtime.EnhancedPATH(binDir)
		}
		if source.ID == "codex" {
			env[runtime.EnvCodexDisableKeyboardEnhancement] = "1"
		}
		return env
	}
	return Spec{Spec: source}
}

func Get(id string) (Spec, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	shared, ok := sdkprovider.Get(id)
	if !ok {
		return Spec{}, false
	}
	return adaptSDKSpec(shared), true
}

func MustGet(id string) Spec {
	spec, ok := Get(id)
	if !ok {
		panic(fmt.Sprintf("unknown provider spec %q", id))
	}
	return spec
}

func All() []Spec {
	shared := sdkprovider.All()
	result := make([]Spec, 0, len(shared))
	for _, item := range shared {
		result = append(result, adaptSDKSpec(item))
	}
	return result
}

func (s Spec) AutonomousIsolationLevel() IsolationGuarantee {
	return s.Spec.AutonomousIsolationLevel()
}

func (s Spec) ReviewIsolationLevel() IsolationGuarantee {
	return s.Spec.ReviewIsolationLevel()
}

func (s Spec) ResolveBinary() (string, error) {
	for _, candidate := range s.BinaryCandidates {
		if path, err := BinaryLookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s binary not found (tried %s)", s.ID, strings.Join(s.BinaryCandidates, ", "))
}

func (s Spec) OverlayEnv(base []string, profileHome, binDir, profileName string) []string {
	if s.EnvKeys == nil {
		return append([]string(nil), base...)
	}
	return overlayEnv(base, profileHome, s.EnvKeys(profileHome, binDir, profileName), s.EnvDelete)
}

func (s Spec) NormalizeFlags(args []string) []string {
	return Normalize(s.ID, args, nil)
}
