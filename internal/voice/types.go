package voice

import "time"

type ProviderKind string

const (
	ProviderGroq       ProviderKind = "groq"
	ProviderCloudflare ProviderKind = "cloudflare"
)

const (
	StatusSuccess                = "SUCCESS"
	StatusRateLimited            = "RATE_LIMITED"
	StatusQuotaExhausted         = "QUOTA_EXHAUSTED"
	StatusAuthError              = "AUTH_ERROR"
	StatusTemporarilyUnavailable = "TEMPORARILY_UNAVAILABLE"
	StatusMisconfigured          = "MISCONFIGURED"
	StatusInvalidRequest         = "INVALID_REQUEST"
	StatusUnknown                = "UNKNOWN"
)

type ProviderConfig struct {
	ID           string       `json:"id"`
	Kind         ProviderKind `json:"kind"`
	Label        string       `json:"label"`
	Enabled      bool         `json:"enabled"`
	Priority     int          `json:"priority"`
	Model        string       `json:"model"`
	AccountID    string       `json:"account_id,omitempty"`
	AccountIDEnv string       `json:"account_id_env,omitempty"`
	BaseURL      string       `json:"base_url,omitempty"`
	APIKeyEnv    string       `json:"api_key_env,omitempty"`
	APIKeyFile   string       `json:"api_key_file,omitempty"`
	FreeEligible bool         `json:"free_eligible"`
}

type Settings struct {
	Version              int              `json:"version"`
	Enabled              bool             `json:"enabled"`
	Language             string           `json:"language"`
	TerminalEnabled      bool             `json:"terminal_enabled"`
	NexusControlEnabled  bool             `json:"nexus_control_enabled"`
	AutomaticFallback    bool             `json:"automatic_fallback"`
	MaxCaptureSeconds    int              `json:"max_capture_seconds"`
	Providers            []ProviderConfig `json:"providers"`
}

type ProviderUpdate struct {
	ProviderConfig
	APIKey      string `json:"api_key,omitempty"`
	ClearAPIKey bool   `json:"clear_api_key,omitempty"`
}

type SettingsUpdate struct {
	Enabled             bool             `json:"enabled"`
	Language            string           `json:"language"`
	TerminalEnabled     bool             `json:"terminal_enabled"`
	NexusControlEnabled bool             `json:"nexus_control_enabled"`
	AutomaticFallback   bool             `json:"automatic_fallback"`
	MaxCaptureSeconds   int              `json:"max_capture_seconds"`
	Providers           []ProviderUpdate `json:"providers"`
}

type ProviderUsage struct {
	ProviderID      string    `json:"provider_id"`
	Requests        int64     `json:"requests"`
	Successes       int64     `json:"successes"`
	Failures        int64     `json:"failures"`
	ObservedAudioMS int64     `json:"observed_audio_ms"`
	LastStatus      string    `json:"last_status,omitempty"`
	LastUsedAt      time.Time `json:"last_used_at,omitempty"`
}

type ProviderView struct {
	ProviderConfig
	HasAPIKey bool          `json:"has_api_key"`
	Configured bool         `json:"configured"`
	Usage      ProviderUsage `json:"usage"`
}

type SettingsView struct {
	Version             int            `json:"version"`
	Enabled             bool           `json:"enabled"`
	Language            string         `json:"language"`
	TerminalEnabled     bool           `json:"terminal_enabled"`
	NexusControlEnabled bool           `json:"nexus_control_enabled"`
	AutomaticFallback   bool           `json:"automatic_fallback"`
	MaxCaptureSeconds   int            `json:"max_capture_seconds"`
	Providers           []ProviderView `json:"providers"`
}

type TranscribeRequest struct {
	Audio      []byte
	Filename   string
	MIMEType   string
	Language   string
	DurationMS int64
}

type ProviderAttempt struct {
	ProviderID string `json:"provider_id"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Message    string `json:"message,omitempty"`
}

type TranscribeResult struct {
	Text       string            `json:"text"`
	ProviderID string            `json:"provider_id"`
	Provider   string            `json:"provider"`
	Model      string            `json:"model"`
	Attempts   []ProviderAttempt `json:"attempts"`
	DurationMS int64             `json:"duration_ms"`
}

type TranscriptionError struct {
	Message  string
	Attempts []ProviderAttempt
}

func (e *TranscriptionError) Error() string { return e.Message }
