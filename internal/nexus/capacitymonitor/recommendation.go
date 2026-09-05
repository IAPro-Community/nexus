package capacitymonitor

import (
	"context"
)

// ResourceKey identifies a provider account and optional quota group.
type ResourceKey struct {
	Provider   string `json:"provider"`
	Profile    string `json:"profile"`
	QuotaGroup string `json:"quota_group,omitempty"`
}

// RecommendationRequest contains criteria for finding a replacement resource.
type RecommendationRequest struct {
	AtRiskResource       ResourceKey `json:"at_risk_resource"`
	TaskKind             string      `json:"task_kind,omitempty"`
	Role                 string      `json:"role,omitempty"`
	RequiredCapabilities []string    `json:"required_capabilities,omitempty"`
	AgentPreference      string      `json:"agent_preference,omitempty"`
	PreferProvider       string      `json:"prefer_provider,omitempty"`
	ProjectPolicy        string      `json:"project_policy,omitempty"`
	CurrentProvider      string      `json:"current_provider,omitempty"`
	CurrentProfile       string      `json:"current_profile,omitempty"`
}

// RecommendationCandidate is an evaluated option with score and explanation.
type RecommendationCandidate struct {
	Provider        string             `json:"provider"`
	Profile         string             `json:"profile"`
	DisplayName     string             `json:"display_name"`
	Rank            int                `json:"rank"`
	TotalScore      float64            `json:"total_score"`
	Confidence      string             `json:"confidence"`
	Eligible        bool               `json:"eligible"`
	RejectionReason string             `json:"rejection_reason,omitempty"`
	ScoreBreakdown  map[string]float64 `json:"score_breakdown,omitempty"`
	Pros            []string           `json:"pros,omitempty"`
	Cons            []string           `json:"cons,omitempty"`
}

// Recommendation holds the best candidate, alternatives, and continuity impact.
type Recommendation struct {
	Fingerprint          string                    `json:"fingerprint"`
	Selected             *RecommendationCandidate  `json:"selected,omitempty"`
	Candidates           []RecommendationCandidate `json:"candidates"`
	ResetInformation     string                    `json:"reset_information,omitempty"`
	RejectedAlternatives []string                  `json:"rejected_alternatives,omitempty"`
	ContinuityImpact     string                    `json:"continuity_impact,omitempty"`
	Explanation          string                    `json:"explanation"`
}

// RecommendationProvider abstracts recommendation computation without importing the parent nexus package.
type RecommendationProvider interface {
	Recommend(ctx context.Context, req RecommendationRequest) (Recommendation, error)
}
