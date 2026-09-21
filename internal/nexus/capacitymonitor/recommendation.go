package capacitymonitor

import "context"

// ResourceKey identifies the at-risk resource a recommendation refers to.
type ResourceKey struct {
	Provider string `json:"provider"`
	Profile  string `json:"profile"`
	Group    string `json:"group,omitempty"`
	Window   string `json:"window,omitempty"`
}

// Recommendation is an ADVISORY fallback suggestion for a low resource.
// It never migrates workloads automatically: the monitor only publishes the
// candidate and the reason; any switch requires explicit user confirmation
// through the existing allocation flow (MANUAL policy).
type Recommendation struct {
	Provider    string  `json:"provider"`
	Profile     string  `json:"profile"`
	DisplayName string  `json:"display_name"`
	Score       float64 `json:"score"`
	Reason      string  `json:"reason"`
	Available   bool    `json:"available"`
}

// Recommender ranks fallback resources for an at-risk key.
//
// The interface lives in capacitymonitor (dependency-neutral) so the parent
// nexus package can implement it around the EXISTING RecommendResources
// without a capacitymonitor -> nexus import cycle: injection via
// Dependencies.Recommender. Implementations must be side-effect free.
type Recommender interface {
	RecommendForLowResource(ctx context.Context, atRisk ResourceKey, reason string) (Recommendation, error)
}
