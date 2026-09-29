package model

import "time"

// ResourceChannel separates product/subscription capacity from metered API
// consumption. A healthy API channel never implies subscription quota and a
// depleted subscription never implies that an API budget is exhausted.
type ResourceChannel string

const (
	ResourceChannelSubscription ResourceChannel = "SUBSCRIPTION"
	ResourceChannelAPI          ResourceChannel = "API"
)

// UsageMetric identifies what an observation measures.
type UsageMetric string

const (
	UsageMetricQuota    UsageMetric = "QUOTA"
	UsageMetricTokens   UsageMetric = "TOKENS"
	UsageMetricRequests UsageMetric = "REQUESTS"
	UsageMetricCost     UsageMetric = "COST"
)

// UsageSemantics declares what the provider meant by Value at the source.
type UsageSemantics string

const (
	SemanticsUsedPercent        UsageSemantics = "USED_PERCENT"
	SemanticsRemainingPercent   UsageSemantics = "REMAINING_PERCENT"
	SemanticsUsedAbsolute       UsageSemantics = "USED_ABSOLUTE"
	SemanticsRemainingAbsolute  UsageSemantics = "REMAINING_ABSOLUTE"
	SemanticsBinaryAvailability UsageSemantics = "BINARY_AVAILABILITY"
)

// UsageObservation is provider evidence before Nexus projects it into quota,
// consumption, cost, or rate-limit views.
type UsageObservation struct {
	ProviderID   string          `json:"provider_id"`
	ProfileID    string          `json:"profile_id"`
	AccountScope AccountScope    `json:"account_scope,omitempty"`
	Channel      ResourceChannel `json:"channel"`
	Metric       UsageMetric     `json:"metric"`
	Semantics    UsageSemantics  `json:"semantics"`
	Value        float64         `json:"value"`
	Unit         string          `json:"unit,omitempty"`
	ModelGroup   string          `json:"model_group,omitempty"`
	WindowKind   string          `json:"window_kind,omitempty"`
	Status       UsageStatus     `json:"status"`
	Source       UsageSource     `json:"source"`
	Confidence   UsageConfidence `json:"confidence,omitempty"`
	ObservedAt   time.Time       `json:"observed_at"`
	ResetAt      *time.Time      `json:"reset_at,omitempty"`
}

// ConsumptionState is metered activity. It is deliberately not a quota.
type ConsumptionState struct {
	InputTokens     float64 `json:"input_tokens,omitempty"`
	CachedTokens    float64 `json:"cached_tokens,omitempty"`
	OutputTokens    float64 `json:"output_tokens,omitempty"`
	ReasoningTokens float64 `json:"reasoning_tokens,omitempty"`
	Requests        float64 `json:"requests,omitempty"`
}

// CostState represents attributable monetary usage for an API channel.
type CostState struct {
	Currency        string   `json:"currency,omitempty"`
	Amount          float64  `json:"amount,omitempty"`
	Budget          *float64 `json:"budget,omitempty"`
	BudgetRemaining *float64 `json:"budget_remaining,omitempty"`
}

// RateLimitState represents throughput limits independently from subscription
// quota and monetary budgets.
type RateLimitState struct {
	Kind      string     `json:"kind,omitempty"`
	Limit     *float64   `json:"limit,omitempty"`
	Remaining *float64   `json:"remaining,omitempty"`
	Unit      string     `json:"unit,omitempty"`
	ResetAt   *time.Time `json:"reset_at,omitempty"`
}

// ResourceChannelState is the additive V2 projection exposed by resource APIs.
// Legacy quota_remaining remains available while clients migrate.
type ResourceChannelState struct {
	Channel        ResourceChannel   `json:"channel"`
	Status         UsageStatus       `json:"status"`
	Source         UsageSource       `json:"source,omitempty"`
	Confidence     UsageConfidence   `json:"confidence,omitempty"`
	ObservedAt     time.Time         `json:"observed_at,omitempty"`
	QuotaRemaining *float64          `json:"quota_remaining,omitempty"` // 0..1 literal remaining capacity
	Consumption    *ConsumptionState `json:"consumption,omitempty"`
	Cost           *CostState        `json:"cost,omitempty"`
	RateLimits     []RateLimitState  `json:"rate_limits,omitempty"`
}
