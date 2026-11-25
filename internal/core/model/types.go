package model

import (
	"time"

	sdkmodel "github.com/IAPro-Community/IAPro-SDK/model"
)

// Execution owner constants identify the single runtime allowed to supervise a
// Mission run. NEXUS runs are executed by the local MissionRunner; MAESTRO
// runs are dispatched only by the external Maestro worker.
const (
	ExecutionOwnerNexus   = "NEXUS"
	ExecutionOwnerMaestro = "MAESTRO"
)

// AccountScope is the identity boundary for account-owned state. Provider and
// profile names are routing labels, not sufficient identities: one profile can
// be re-authenticated and two accounts may have the same display email.
// AccountScope is the SDK's canonical provider account identity contract.
type AccountScope = sdkmodel.AccountScope

// ProviderID identifies a supported AI CLI provider (e.g. "codex", "agy", "claude", "opencode", "gemini", "cursor").
type ProviderID = sdkmodel.ProviderID

const (
	ProviderCodex               = sdkmodel.ProviderCodex
	ProviderAGY                 = sdkmodel.ProviderAGY
	ProviderClaude              = sdkmodel.ProviderClaude
	ProviderOpenCode            = sdkmodel.ProviderOpenCode
	ProviderGemini              = sdkmodel.ProviderGemini
	ProviderCursor              = sdkmodel.ProviderCursor
	ProviderZCode    ProviderID = "zcode" // Nexus-only provider; not yet in the shared catalog.
)

// FailureKind categorizes the cause of a CLI execution failure.
type FailureKind = sdkmodel.FailureKind

const (
	FailureNone      = sdkmodel.FailureNone
	FailureAuth      = sdkmodel.FailureAuth
	FailureQuota     = sdkmodel.FailureQuota
	FailureRateLimit = sdkmodel.FailureRateLimit
	FailureNetwork   = sdkmodel.FailureNetwork
	FailureProvider  = sdkmodel.FailureProvider
	FailureCommand   = sdkmodel.FailureCommand
	FailureUser      = sdkmodel.FailureUser
	FailureUnknown   = sdkmodel.FailureUnknown
)

// UsageStatus represents the state and confidence of quota/usage data.
type UsageStatus = sdkmodel.UsageStatus

const (
	UsageLive        = sdkmodel.UsageLive
	UsageCached      = sdkmodel.UsageCached
	UsageEstimated   = sdkmodel.UsageEstimated
	UsageUnknown     = sdkmodel.UsageUnknown
	UsageUnsupported = sdkmodel.UsageUnsupported
	UsageRateLimited = sdkmodel.UsageRateLimited
	UsageError       = sdkmodel.UsageError
)

// UsageSource represents the origin of usage data.
type UsageSource = sdkmodel.UsageSource

const (
	SourceOfficialAPI    = sdkmodel.SourceOfficialAPI
	SourceCLIOutput      = sdkmodel.SourceCLIOutput
	SourceLocalFiles     = sdkmodel.SourceLocalFiles
	SourceResponseHeader = sdkmodel.SourceResponseHeader
	SourceObservation    = sdkmodel.SourceObservation
	SourceNone           = sdkmodel.SourceNone
)

// UsageConfidence describes how strongly Nexus can support an observation.
type UsageConfidence = sdkmodel.UsageConfidence

const (
	UsageConfidenceHigh    = sdkmodel.UsageConfidenceHigh
	UsageConfidenceMedium  = sdkmodel.UsageConfidenceMedium
	UsageConfidenceLow     = sdkmodel.UsageConfidenceLow
	UsageConfidenceUnknown = sdkmodel.UsageConfidenceUnknown
)

// ProviderHealth represents the operational health of a provider or profile.
type ProviderHealth = sdkmodel.ProviderHealth

const (
	HealthHealthy      = sdkmodel.HealthHealthy
	HealthDegraded     = sdkmodel.HealthDegraded
	HealthUnavailable  = sdkmodel.HealthUnavailable
	HealthAuthRequired = sdkmodel.HealthAuthRequired
	HealthRateLimited  = sdkmodel.HealthRateLimited
	HealthUnknown      = sdkmodel.HealthUnknown
)

// UsageWindow represents usage metrics for a specific time window (e.g. 5h, weekly).
// Group clusters windows by model family when a provider exposes separate quotas
// (e.g. AGY: "gemini" for Gemini models, "claude_gpt" for Claude/GPT models).
type UsageWindow = sdkmodel.UsageWindow

// UsageSnapshot captures point-in-time usage metrics for a profile.
type UsageSnapshot = sdkmodel.UsageSnapshot

// Capabilities declares supported features for a provider adapter.
type Capabilities = sdkmodel.Capabilities

// Profile represents a local profile entity.
type Profile struct {
	Provider     string       `json:"provider"`
	Name         string       `json:"name"`
	AccountScope AccountScope `json:"account_scope,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	Disabled     bool         `json:"disabled,omitempty"`
	Priority     int          `json:"priority,omitempty"`
	Labels       []string     `json:"labels,omitempty"`
}

// AuthSource records how a profile obtained its provider credential. It never
// carries the credential itself, only provenance that the UI can explain.
type AuthSource struct {
	Origin        string    `json:"origin"`
	LinkedAt      time.Time `json:"linked_at"`
	AccountIDHash string    `json:"account_id_hash,omitempty"`
	Plan          string    `json:"plan,omitempty"`
}

// AuthSourceOriginHost marks a credential adopted from the host CLI login.
const AuthSourceOriginHost = "host"

// HostAuthDetection summarizes a detected authentication session on the host machine.
type HostAuthDetection struct {
	Provider  string `json:"provider,omitempty"`
	Installed bool   `json:"installed"`
	Available bool   `json:"available"`
	Email     string `json:"email,omitempty"`
	Plan      string `json:"plan,omitempty"`
	AuthType  string `json:"auth_type,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}

// AccountInfo summarizes identity and status information for a profile.
type AccountInfo struct {
	AccountScope      AccountScope   `json:"account_scope,omitempty"`
	Email             string         `json:"email"`
	ExternalAccountID string         `json:"external_account_id,omitempty"`
	Plan              string         `json:"plan"`
	Status            string         `json:"status"`
	Health            ProviderHealth `json:"health"`
	Authenticated     bool           `json:"authenticated"`
	ExpiresAt         time.Time      `json:"expires_at,omitempty"`
	Limits            []string       `json:"limits,omitempty"`
	Usage             UsageSnapshot  `json:"usage"`
}

// Session represents a universal session index entry across providers.
type Session struct {
	ProviderID      string    `json:"provider_id"`
	ProfileID       string    `json:"profile_id"`
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Workspace       string    `json:"workspace"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ResumeSupported bool      `json:"resume_supported"`
	Pinned          bool      `json:"pinned,omitempty"`
}

// WorkspaceInfo groups active sessions and bound profiles for a workspace directory.
type WorkspaceInfo struct {
	Path      string            `json:"path"`
	Bindings  map[string]string `json:"bindings"` // provider -> profile
	Sessions  []Session         `json:"sessions"`
	LastTouch time.Time         `json:"last_touch"`
}

// DetectionResult indicates whether a CLI provider binary is available locally.
type DetectionResult struct {
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	BinaryPath string `json:"binary_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Failure classifies an execution failure.
type Failure = sdkmodel.Failure

// IsolationPreset defines security isolation level.
type IsolationPreset string

const (
	IsolationStrict    IsolationPreset = "strict"
	IsolationDeveloper IsolationPreset = "developer"
	IsolationCompat    IsolationPreset = "compat"
)
