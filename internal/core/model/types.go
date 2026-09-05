package model

import "time"

// ProviderID identifies a supported AI CLI provider (e.g. "codex", "agy", "claude", "opencode", "gemini", "cursor").
type ProviderID string

const (
	ProviderCodex    ProviderID = "codex"
	ProviderAGY      ProviderID = "agy"
	ProviderClaude   ProviderID = "claude"
	ProviderOpenCode ProviderID = "opencode"
	ProviderGemini   ProviderID = "gemini"
	ProviderCursor   ProviderID = "cursor"
)

// FailureKind categorizes the cause of a CLI execution failure.
type FailureKind string

const (
	FailureNone      FailureKind = "NONE"
	FailureAuth      FailureKind = "AUTH_FAILURE"
	FailureQuota     FailureKind = "QUOTA_FAILURE"
	FailureRateLimit FailureKind = "RATE_LIMIT_FAILURE"
	FailureNetwork   FailureKind = "NETWORK_FAILURE"
	FailureProvider  FailureKind = "PROVIDER_FAILURE"
	FailureCommand   FailureKind = "COMMAND_FAILURE"
	FailureUser      FailureKind = "USER_FAILURE"
	FailureUnknown   FailureKind = "UNKNOWN_FAILURE"
)

// UsageGranularity indicates the specificity of quota evidence.
type UsageGranularity string

const (
	UsageGranularityProvider UsageGranularity = "PROVIDER"
	UsageGranularityProfile  UsageGranularity = "PROFILE"
	UsageGranularityGroup    UsageGranularity = "GROUP"
	UsageGranularityModel    UsageGranularity = "MODEL"
)

// UsageStatus represents the state and confidence of quota/usage data.
type UsageStatus string

const (
	UsageLive        UsageStatus = "LIVE"
	UsageCached      UsageStatus = "CACHED"
	UsageEstimated   UsageStatus = "ESTIMATED"
	UsageUnknown     UsageStatus = "UNKNOWN"
	UsageUnsupported UsageStatus = "UNSUPPORTED"
	UsageRateLimited UsageStatus = "RATE_LIMITED"
	UsageError       UsageStatus = "ERROR"
)

// UsageSource represents the origin of usage data.
type UsageSource string

const (
	SourceOfficialAPI    UsageSource = "OFFICIAL_API"
	SourceCLIOutput      UsageSource = "CLI_OUTPUT"
	SourceLocalFiles     UsageSource = "LOCAL_FILES"
	SourceResponseHeader UsageSource = "RESPONSE_HEADERS"
	SourceObservation    UsageSource = "OBSERVATION"
	SourceNone           UsageSource = "NONE"
)

// ProviderHealth represents the operational health of a provider or profile.
type ProviderHealth string

const (
	HealthHealthy      ProviderHealth = "HEALTHY"
	HealthDegraded     ProviderHealth = "DEGRADED"
	HealthUnavailable  ProviderHealth = "UNAVAILABLE"
	HealthAuthRequired ProviderHealth = "AUTH_REQUIRED"
	HealthRateLimited  ProviderHealth = "RATE_LIMITED"
	HealthUnknown      ProviderHealth = "UNKNOWN"
)

// UsageWindow represents usage metrics for a specific time window (e.g. 5h, weekly).
// Group clusters windows by model family when a provider exposes separate quotas
// (e.g. AGY: "gemini" for Gemini models, "claude_gpt" for Claude/GPT models).
type UsageWindow struct {
	Kind             string     `json:"kind"`
	Group            string     `json:"group,omitempty"`
	UsedPercent      *float64   `json:"used_percent,omitempty"`
	RemainingPercent *float64   `json:"remaining_percent,omitempty"`
	ResetTime        *time.Time `json:"reset_time,omitempty"`
	ResetDescription string     `json:"reset_description,omitempty"`
}

// UsageSnapshot captures point-in-time usage metrics for a profile.
type UsageSnapshot struct {
	ProviderID  string           `json:"provider_id"`
	ProfileID   string           `json:"profile_id"`
	Status      UsageStatus      `json:"status"`
	Source      UsageSource      `json:"source"`
	FetchedAt   time.Time        `json:"fetched_at"`
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
	Windows     []UsageWindow    `json:"windows"`
	ModelName   string           `json:"model_name,omitempty"`
	QuotaGroup  string           `json:"quota_group,omitempty"`
	Granularity UsageGranularity `json:"granularity,omitempty"`
	Account     string           `json:"account,omitempty"`
	Plan        string           `json:"plan,omitempty"`
	Error       string           `json:"error,omitempty"`
}

// Capabilities declares supported features for a provider adapter.
type Capabilities struct {
	Login              bool `json:"login"`
	Logout             bool `json:"logout"`
	Usage              bool `json:"usage"`
	Conversations      bool `json:"conversations"`
	Resume             bool `json:"resume"`
	CrossAccountResume bool `json:"cross_account_resume"`
	HotAccountSwitch   bool `json:"hot_account_switch"`
	IsolatedRuntime    bool `json:"isolated_runtime"`
	ProjectBinding     bool `json:"project_binding"`
}

// Profile represents a local profile entity.
type Profile struct {
	Provider  string    `json:"provider"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Disabled  bool      `json:"disabled,omitempty"`
	Priority  int       `json:"priority,omitempty"`
	Labels    []string  `json:"labels,omitempty"`
}

// AccountInfo summarizes identity and status information for a profile.
type AccountInfo struct {
	Email         string         `json:"email"`
	Plan          string         `json:"plan"`
	Status        string         `json:"status"`
	Health        ProviderHealth `json:"health"`
	Authenticated bool           `json:"authenticated"`
	ExpiresAt     time.Time      `json:"expires_at,omitempty"`
	Limits        []string       `json:"limits,omitempty"`
	Usage         UsageSnapshot  `json:"usage"`
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
type Failure struct {
	Kind       FailureKind    `json:"kind"`
	Message    string         `json:"message"`
	RetryAfter *time.Duration `json:"retry_after,omitempty"`
	ResetAt    *time.Time     `json:"reset_at,omitempty"`
}

// IsolationPreset defines security isolation level.
type IsolationPreset string

const (
	IsolationStrict    IsolationPreset = "strict"
	IsolationDeveloper IsolationPreset = "developer"
	IsolationCompat    IsolationPreset = "compat"
)
