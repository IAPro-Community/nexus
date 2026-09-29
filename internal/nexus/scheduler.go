package nexus

import (
	"fmt"
	"sort"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

// ProviderAccount represents a provider account/profile that can run agents.
type ProviderAccount struct {
	ID             string              `json:"id"`
	Provider       string              `json:"provider"`
	Profile        string              `json:"profile"`
	DisplayName    string              `json:"display_name"`
	Authenticated  bool                `json:"authenticated"`
	IsDefault      bool                `json:"is_default"`
	Available      bool                `json:"available"`
	AvailReasons   *quota.AvailReasons `json:"avail_reasons,omitempty"`
	QuotaView      *quota.QuotaView    `json:"quota_view,omitempty"`
	Eligibility    *quota.Eligibility  `json:"eligibility,omitempty"`
	Capabilities     map[string]string            `json:"capabilities,omitempty"`
	ResourceChannels []model.ResourceChannelState `json:"resource_channels,omitempty"`
	QuotaRemaining   float64                      `json:"quota_remaining"` // 0-1 literal remaining quota of the best usable pool
	QuotaTotal     float64             `json:"quota_total"`
	RateLimited    bool                `json:"rate_limited"`
	NetworkError   bool                `json:"network_error,omitempty"`
	CooldownUntil  *time.Time          `json:"cooldown_until,omitempty"`
	Health         string              `json:"health"` // "healthy" | "degraded" | "unhealthy" | "unknown"
	Scope          model.AccountScope  `json:"account_scope,omitempty"`
	LastChecked    time.Time           `json:"last_checked"`
}

// UsageSnapshot captures the current resource usage state.
type UsageSnapshot struct {
	Freshness   string  `json:"freshness"`  // "LIVE" | "CACHED" | "ESTIMATED" | "UNKNOWN"
	Confidence  float64 `json:"confidence"` // 0-1
	Remaining   float64 `json:"remaining"`
	Total       float64 `json:"total"`
	RateLimited bool    `json:"rate_limited"`
	Unavailable bool    `json:"unavailable"`
}

// SchedulerPolicy determines how providers are selected for agents.
type SchedulerPolicy string

const (
	PolicyBalanced       SchedulerPolicy = "BALANCED"
	PolicyPreserveQuota  SchedulerPolicy = "PRESERVE_QUOTA"
	PolicyPreferProvider SchedulerPolicy = "PREFER_PROVIDER"
	PolicyManual         SchedulerPolicy = "MANUAL"
)

// SchedulerDecision describes why a provider was selected.
type SchedulerDecision struct {
	Selected    ProviderAccount `json:"selected"`
	Policy      SchedulerPolicy `json:"policy"`
	Reason      string          `json:"reason"`
	Score       float64         `json:"score"`
	Rejected    []Rejection     `json:"rejected,omitempty"`
	ExplainPath []string        `json:"explain_path"`
}

// Rejection records a provider that was considered but rejected.
type Rejection struct {
	Account ProviderAccount `json:"account"`
	Reason  string          `json:"reason"`
}

// ResourceScheduler selects provider accounts for agents based on policies,
// quotas, health, continuity, and project affinity.
type ResourceScheduler struct {
	accounts []ProviderAccount
	policy   SchedulerPolicy
}

// NewResourceScheduler creates a scheduler with the given accounts and policy.
func NewResourceScheduler(accounts []ProviderAccount, policy SchedulerPolicy) *ResourceScheduler {
	if policy == "" {
		policy = PolicyBalanced
	}
	return &ResourceScheduler{
		accounts: accounts,
		policy:   policy,
	}
}

// Select chooses the best provider account for an agent, returning an
// explainable decision. Implements the Gate 5 scheduling algorithm.
func (s *ResourceScheduler) Select(
	preferProvider string,
	currentProvider string,
	continuity string,
	projectAffinity map[string]int, // provider → recent usage count
) SchedulerDecision {
	decision := SchedulerDecision{
		Policy:      s.policy,
		ExplainPath: []string{fmt.Sprintf("policy=%s", s.policy)},
	}

	candidates := s.filterEligible()
	decision.ExplainPath = append(decision.ExplainPath, fmt.Sprintf("eligible=%d", len(candidates)))

	// Manual policy with no explicit preference: reject all.
	if s.policy == PolicyManual && preferProvider == "" {
		decision.Reason = "manual policy requires explicit provider preference"
		return decision
	}

	if len(candidates) == 0 {
		decision.Reason = "no eligible provider accounts"
		return decision
	}

	// Score each candidate.
	type scoredEntry struct {
		account ProviderAccount
		score   float64
		reason  string
	}
	var entries []scoredEntry

	for _, acc := range candidates {
		score, reason := s.scoreAccount(acc, preferProvider, currentProvider, continuity, projectAffinity)
		entries = append(entries, scoredEntry{account: acc, score: score, reason: reason})
	}

	// Sort by score descending.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	if len(entries) > 0 {
		decision.Selected = entries[0].account
		decision.Score = entries[0].score
		decision.Reason = entries[0].reason
		decision.ExplainPath = append(decision.ExplainPath,
			fmt.Sprintf("selected=%s score=%.2f", entries[0].account.ID, entries[0].score),
		)
	}

	// Record rejections.
	for _, e := range entries[1:] {
		decision.Rejected = append(decision.Rejected, Rejection{
			Account: e.account,
			Reason:  e.reason,
		})
	}

	return decision
}

// filterEligible returns accounts that pass the shared quota.EvaluateEligibility
// gate (auth, network, health, cooldown, rate-limit, UNKNOWN, exhausted).
func (s *ResourceScheduler) filterEligible() []ProviderAccount {
	var out []ProviderAccount
	for _, acc := range s.accounts {
		elig := quota.EvaluateEligibility(accountEligibilityInput(acc, false))
		if !elig.Eligible {
			continue
		}
		out = append(out, acc)
	}
	return out
}

// scoreAccount returns a 0-1 score and explanation for a single account.
func (s *ResourceScheduler) scoreAccount(
	acc ProviderAccount,
	preferProvider, currentProvider, continuity string,
	projectAffinity map[string]int,
) (float64, string) {
	score := 0.5 // base score
	reasons := []string{}

	// Health bonus.
	switch acc.Health {
	case "healthy":
		score += 0.2
		reasons = append(reasons, "healthy")
	case "degraded":
		score -= 0.1
		reasons = append(reasons, "degraded")
	}

	// Quota availability.
	if acc.QuotaTotal > 0 {
		quotaRatio := acc.QuotaRemaining / acc.QuotaTotal
		score += quotaRatio * 0.2
		reasons = append(reasons, fmt.Sprintf("quota=%.0f%%", quotaRatio*100))
	}

	// Policy-specific scoring.
	switch s.policy {
	case PolicyPreserveQuota:
		// Prefer accounts with more remaining quota.
		if acc.QuotaTotal > 0 {
			score += (acc.QuotaRemaining / acc.QuotaTotal) * 0.3
		}
		reasons = append(reasons, "preserve-quota")

	case PolicyPreferProvider:
		// Strongly prefer the requested provider.
		if preferProvider != "" && acc.Provider == preferProvider {
			score += 0.4
			reasons = append(reasons, fmt.Sprintf("prefer=%s", preferProvider))
		}

	case PolicyManual:
		// Only use if explicitly matched.
		if preferProvider != "" && acc.Provider == preferProvider {
			score += 0.5
			reasons = append(reasons, "manual-match")
		} else {
			score = 0
			reasons = append(reasons, "manual-no-match")
		}
	}

	// Universal preference bonus (all policies except Manual).
	// Manual policy already handled above; skip to avoid double-scoring.
	if s.policy != PolicyManual && preferProvider != "" && acc.Provider == preferProvider {
		score += 0.15
		reasons = append(reasons, fmt.Sprintf("user-prefer=%s", preferProvider))
	}

	// Continuity: prefer same provider for native resume.
	if continuity == "NATIVE_RESUME_VERIFIED" || continuity == "NATIVE_RESUME_UNVERIFIED" {
		if acc.Provider == currentProvider {
			score += 0.15
			reasons = append(reasons, "continuity-same-provider")
		}
	}

	// Project affinity: prefer providers recently used in this project.
	if projectAffinity != nil {
		if count, ok := projectAffinity[acc.Provider]; ok && count > 0 {
			score += 0.1
			reasons = append(reasons, fmt.Sprintf("project-affinity=%d", count))
		}
	}

	return score, joinReasons(reasons)
}

func joinReasons(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "; " + p
	}
	return result
}
