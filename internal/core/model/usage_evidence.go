package model

import "time"

// ResourceChannel identifies independent capacity/billing domains exposed by
// the same provider account. Subscription quota and API billing must never be
// added together or treated as interchangeable capacity.
type ResourceChannel string

const (
	ResourceChannelSubscription ResourceChannel = "SUBSCRIPTION"
	ResourceChannelAPI          ResourceChannel = "API"
)

// UsageSemantics states what an observed numeric value means at the provider
// boundary. Adapters report source semantics; the quota layer owns normalization.
type UsageSemantics string

const (
	UsageSemanticsUsedPercent        UsageSemantics = "USED_PERCENT"
	UsageSemanticsRemainingPercent   UsageSemantics = "REMAINING_PERCENT"
	UsageSemanticsUsedAbsolute       UsageSemantics = "USED_ABSOLUTE"
	UsageSemanticsRemainingAbsolute  UsageSemantics = "REMAINING_ABSOLUTE"
	UsageSemanticsBinaryAvailability UsageSemantics = "BINARY_AVAILABILITY"
)

// UsageMetric identifies the measured resource without assuming how a provider
// expresses it.
type UsageMetric string

const (
	UsageMetricQuota       UsageMetric = "QUOTA"
	UsageMetricConsumption UsageMetric = "CONSUMPTION"
	UsageMetricCost        UsageMetric = "COST"
	UsageMetricRateLimit   UsageMetric = "RATE_LIMIT"
)

// UsageObservation is provider-boundary evidence before canonical
// normalization. It is intentionally separate from UsageSnapshot so adapters
// can describe provider-native semantics without leaking those semantics to
// schedulers or UI surfaces.
type UsageObservation struct {
	ProviderID   string          `json:"provider_id"`
	ProfileID    string          `json:"profile_id"`
	AccountScope AccountScope    `json:"account_scope,omitempty"`
	Channel      ResourceChannel `json:"channel"`
	Metric       UsageMetric     `json:"metric"`
	Value        float64         `json:"value"`
	Unit         string          `json:"unit,omitempty"`
	Semantics    UsageSemantics  `json:"semantics"`
	ModelGroup   string          `json:"model_group,omitempty"`
	Window       string          `json:"window,omitempty"`
	ObservedAt   time.Time       `json:"observed_at"`
	ResetAt      *time.Time      `json:"reset_at,omitempty"`
	Source       UsageSource     `json:"source"`
	Confidence   UsageConfidence `json:"confidence,omitempty"`
}

// ConsumptionState represents metered API consumption independently from
// subscription quota. Nil values are unknown, never synthetic zero.
type ConsumptionState struct {
	InputTokens     *float64 `json:"input_tokens,omitempty"`
	CachedTokens    *float64 `json:"cached_tokens,omitempty"`
	OutputTokens    *float64 `json:"output_tokens,omitempty"`
	ReasoningTokens *float64 `json:"reasoning_tokens,omitempty"`
	Requests        *float64 `json:"requests,omitempty"`
}

// CostState represents provider-reported or reconciled API spend.
type CostState struct {
	Amount          *float64 `json:"amount,omitempty"`
	Currency        string   `json:"currency,omitempty"`
	Budget          *float64 `json:"budget,omitempty"`
	BudgetRemaining *float64 `json:"budget_remaining,omitempty"`
}

// RateLimitState is throughput evidence. It is deliberately not quota or
// budget: exhausting RPM/TPM is operational throttling, not subscription
// exhaustion or spend-limit exhaustion.
type RateLimitState struct {
	RequestsRemaining *float64   `json:"requests_remaining,omitempty"`
	TokensRemaining   *float64   `json:"tokens_remaining,omitempty"`
	ResetAt           *time.Time `json:"reset_at,omitempty"`
}

// ResourceChannelState is the forward-compatible projection used by Usage /
// Capacity V2. Existing UsageSnapshot consumers remain valid while API-backed
// consumption and billing are added incrementally.
type ResourceChannelState struct {
	Channel      ResourceChannel   `json:"channel"`
	AccountScope AccountScope      `json:"account_scope,omitempty"`
	Status       UsageStatus       `json:"status,omitempty"`
	Source       UsageSource       `json:"source,omitempty"`
	Confidence   UsageConfidence   `json:"confidence,omitempty"`
	ObservedAt   time.Time         `json:"observed_at,omitempty"`
	Quota        *UsageSnapshot    `json:"quota,omitempty"`
	Consumption  *ConsumptionState `json:"consumption,omitempty"`
	Cost         *CostState        `json:"cost,omitempty"`
	RateLimit    *RateLimitState   `json:"rate_limit,omitempty"`
}
