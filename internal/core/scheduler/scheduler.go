package scheduler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/cooldown"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/core/telemetry"
)

// CandidateEvaluation holds the scoring breakdown for a single candidate profile.
type CandidateEvaluation struct {
	Profile        model.Profile       `json:"profile"`
	Account        model.AccountInfo   `json:"account"`
	Usage          model.UsageSnapshot `json:"usage"`
	Score          float64             `json:"score"`
	Eligible       bool                `json:"eligible"`
	RejectReason   string              `json:"reject_reason,omitempty"`
	ScoreBreakdown []string            `json:"score_breakdown"`
	Bound          bool                `json:"bound"`
	IsDefault      bool                `json:"is_default"`
}

// SelectionResult represents the outcome of the smart account selection algorithm.
type SelectionResult struct {
	SelectedProfile *model.Profile        `json:"selected_profile,omitempty"`
	Reason          string                `json:"reason"`
	Evaluations     []CandidateEvaluation `json:"evaluations"`
}

// Selector scores and picks the optimal account for an execution request.
type Selector struct {
	cfg      config.Config
	quotaEng *quota.Engine
	cooldown *cooldown.Tracker
}

// NewSelector creates a new Selector instance.
func NewSelector(cfg config.Config, q *quota.Engine, cd *cooldown.Tracker) *Selector {
	if cd == nil {
		cd = cooldown.NewTracker()
	}
	return &Selector{
		cfg:      cfg,
		quotaEng: q,
		cooldown: cd,
	}
}

// SelectBestProfile evaluates candidate profiles and selects the optimal account.
func (s *Selector) SelectBestProfile(ctx context.Context, provider string, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excludeProfiles []string) (*SelectionResult, error) {
	evals := s.EvaluateAll(provider, workspace, candidates, accounts, excludeProfiles)

	var eligible []CandidateEvaluation
	for _, ev := range evals {
		if ev.Eligible {
			eligible = append(eligible, ev)
		}
	}

	if len(eligible) == 0 {
		// Check if non-disabled profiles exist where auth can be attempted on launch
		var fallbackCandidates []CandidateEvaluation
		for _, ev := range evals {
			if !ev.Profile.Disabled && ev.RejectReason == "authentication required" {
				fallbackCandidates = append(fallbackCandidates, ev)
			}
		}

		if len(fallbackCandidates) > 0 {
			var best CandidateEvaluation
			foundBest := false
			for _, c := range fallbackCandidates {
				if c.Bound {
					best = c
					foundBest = true
					break
				}
			}
			if !foundBest {
				for _, c := range fallbackCandidates {
					if c.IsDefault {
						best = c
						foundBest = true
						break
					}
				}
			}
			if !foundBest {
				best = fallbackCandidates[0]
			}

			return &SelectionResult{
				SelectedProfile: &best.Profile,
				Reason:          "default profile fallback",
				Evaluations:     evals,
			}, nil
		}

		var rejectSummaries []string
		for _, ev := range evals {
			rejectSummaries = append(rejectSummaries, fmt.Sprintf("%s (%s)", ev.Profile.Name, ev.RejectReason))
		}
		if len(rejectSummaries) == 0 {
			// Never return a nil result: callers rely on a non-nil SelectionResult.
			return &SelectionResult{Evaluations: evals}, fmt.Errorf("no %s profiles configured", provider)
		}
		return &SelectionResult{
			Evaluations: evals,
		}, fmt.Errorf("no usable %s profiles available: %s", provider, strings.Join(rejectSummaries, ", "))
	}

	// When no candidate has a current quota observation, rotate healthy
	// authenticated profiles by least-recently-selected instead of favoring a
	// stale default. ESTIMATED windows remain visible to the caller but are not
	// evidence for automatic account selection.
	hasQuotaEvidence := false
	for _, ev := range eligible {
		if len(ev.Usage.Windows) > 0 && (ev.Usage.Status == model.UsageLive || ev.Usage.Status == model.UsageCached) {
			hasQuotaEvidence = true
			break
		}
	}
	if !hasQuotaEvidence {
		lastUsed := selectedAt(provider)
		sort.SliceStable(eligible, func(i, j int) bool {
			left, right := lastUsed[eligible[i].Profile.Name], lastUsed[eligible[j].Profile.Name]
			if !left.Equal(right) {
				return left.Before(right)
			}
			return eligible[i].Profile.Name < eligible[j].Profile.Name
		})
		best := eligible[0]
		return &SelectionResult{SelectedProfile: &best.Profile, Reason: "quota UNKNOWN; LRU among healthy authenticated profiles", Evaluations: evals}, nil
	}

	// Sort eligible candidates by Score descending
	sort.Slice(eligible, func(i, j int) bool {
		iFresh := currentQuotaEvidence(eligible[i].Usage.Status)
		jFresh := currentQuotaEvidence(eligible[j].Usage.Status)
		if iFresh != jFresh {
			return iFresh
		}
		if eligible[i].Score != eligible[j].Score {
			return eligible[i].Score > eligible[j].Score
		}
		return eligible[i].Profile.Name < eligible[j].Profile.Name
	})

	best := eligible[0]
	reason := strings.Join(best.ScoreBreakdown, ", ")
	if reason == "" {
		reason = "highest available score"
	}

	return &SelectionResult{
		SelectedProfile: &best.Profile,
		Reason:          reason,
		Evaluations:     evals,
	}, nil
}

func currentQuotaEvidence(status model.UsageStatus) bool {
	return status == model.UsageLive || status == model.UsageCached
}

func selectedAt(provider string) map[string]time.Time {
	result := make(map[string]time.Time)
	events, err := telemetry.ReadRecentEvents(0)
	if err != nil {
		return result
	}
	for _, ev := range events {
		if ev.Type == telemetry.EventProfileSelected && ev.ProviderID == provider {
			if old, ok := result[ev.ProfileID]; !ok || ev.Timestamp.After(old) {
				result[ev.ProfileID] = ev.Timestamp
			}
		}
	}
	return result
}

// EvaluateAll scores all candidate profiles for a provider.
func (s *Selector) EvaluateAll(provider string, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo, excludeProfiles []string) []CandidateEvaluation {
	boundProfile := config.GetBinding(workspace, provider)
	defaultProfile := s.cfg.Defaults[provider]

	var evals []CandidateEvaluation
	for _, p := range candidates {
		if p.Provider != provider {
			continue
		}

		acc, hasAcc := accounts[p.Name]
		if !hasAcc {
			acc = model.AccountInfo{
				Authenticated: false,
				Health:        model.HealthUnknown,
			}
		}

		snap := acc.Usage
		scope := p.AccountScope
		if !scope.Verifiable() {
			scope = acc.AccountScope
		}
		if snap.ProviderID == "" || snap.ProfileID == "" {
			if scope.Verifiable() {
				snap, _ = s.quotaEng.GetCachedUsageForScope(scope)
			} else {
				// Compatibility for callers constructing in-memory profiles. All
				// persisted profiles are backfilled with AccountScope by profile.List.
				snap, _ = s.quotaEng.GetCachedUsage(provider, p.Name)
			}
		}
		isExcluded := false
		for _, ex := range excludeProfiles {
			if ex == p.Name {
				isExcluded = true
				break
			}
		}

		ev := CandidateEvaluation{
			Profile:   p,
			Account:   acc,
			Usage:     snap,
			Bound:     boundProfile == p.Name,
			IsDefault: defaultProfile == p.Name,
		}

		// 1. Check Hard Filters
		if isExcluded {
			ev.Eligible = false
			ev.RejectReason = "excluded in current fallback cycle"
			evals = append(evals, ev)
			continue
		}

		if p.Disabled || (s.cfg.Disabled[provider] != nil && s.cfg.Disabled[provider][p.Name]) {
			ev.Eligible = false
			ev.RejectReason = "profile is disabled"
			evals = append(evals, ev)
			continue
		}

		if !acc.Authenticated {
			ev.Eligible = false
			ev.RejectReason = "authentication required"
			evals = append(evals, ev)
			continue
		}

		var isLimited bool
		var rec *cooldown.Record
		if scope.Verifiable() {
			isLimited, rec = s.cooldown.IsRateLimitedForScope(scope)
		} else {
			isLimited, rec = s.cooldown.IsRateLimited(provider, p.Name)
		}
		qv := quota.BuildQuotaView(snap, acc.Email, acc.Plan)
		elig := quota.EvaluateEligibility(quota.EligibilityInput{
			View:          qv,
			Authenticated: acc.Authenticated,
			Health:        string(acc.Health),
			InCooldown:    isLimited,
		})
		if !elig.Eligible {
			ev.Eligible = false
			if elig.Reason == quota.ReasonCooldown && rec != nil {
				ev.RejectReason = fmt.Sprintf("rate limited until %s (%s)", rec.ResetAt.Format("15:04"), rec.Reason)
			} else if elig.Message != "" {
				ev.RejectReason = elig.Message
			} else {
				ev.RejectReason = fmt.Sprintf("unavailable: %s", qv.AvailabilityLabel())
			}
			evals = append(evals, ev)
			continue
		}

		ev.Eligible = true

		// 2. Score Calculation
		score := 100.0 // Base availability score
		var breakdown []string
		breakdown = append(breakdown, "authenticated")

		// UNKNOWN quota is not a hard block, but flag it for scoring penalty.
		if qv.AvailReasons.UnknownQuota {
			score -= 10.0
			breakdown = append(breakdown, "unknown quota (-10.0)")
		}

		// Capacity / Quota Score via QuotaView bottleneck
		effectiveCapacity, bottleneckKind, avgRemaining := quota.BottleneckScore(&qv)
		hasWindows := false
		for _, window := range qv.AllWindows() {
			if window.Kind != "unknown" {
				hasWindows = true
				break
			}
		}

		if hasWindows {
			capScore := effectiveCapacity * 10.0 // Up to +1000 points for 100% capacity
			score += capScore

			minPct, _ := qv.Bottleneck()
			if len(qv.AllWindows()) == 1 {
				breakdown = append(breakdown, fmt.Sprintf("%.0f%% capacity (+%.1f)", effectiveCapacity, capScore))
			} else {
				breakdown = append(breakdown, fmt.Sprintf("%.0f%% eff capacity (min: %.0f%% [%s], avg: %.0f%%) (+%.1f)", effectiveCapacity, minPct, bottleneckKind, avgRemaining, capScore))
			}
		} else {
			// Unknown quota remains eligible for resilience, but must never
			// outrank an account with a known positive quota.
			breakdown = append(breakdown, "unknown capacity (+0.0)")
		}

		// User Configured Priority
		priority := p.Priority
		if s.cfg.Priorities[provider] != nil {
			priority = s.cfg.Priorities[provider][p.Name]
		}
		if priority != 0 {
			pScore := float64(priority) * 10.0
			score += pScore
			breakdown = append(breakdown, fmt.Sprintf("priority %d (+%.1f)", priority, pScore))
		}

		// Project / Workspace Binding Boost
		if ev.Bound {
			score += 50.0
			breakdown = append(breakdown, "workspace bound (+50.0)")
		}

		// Default Profile Boost (Used as minor tie-breaker)
		if ev.IsDefault {
			score += 1.0
			breakdown = append(breakdown, "default profile (+1.0)")
		}

		// Recency / Plan Type (Used as minor tie-breaker)
		if acc.Plan == "ChatGPT Pro" || acc.Plan == "Google AI Pro" {
			score += 2.0
			breakdown = append(breakdown, "pro tier (+2.0)")
		}

		ev.Score = score
		ev.ScoreBreakdown = breakdown
		evals = append(evals, ev)
	}

	return evals
}

// ExplainSelection produces a formatted textual explanation of candidate evaluations.
func (s *Selector) ExplainSelection(ctx context.Context, provider string, workspace string, candidates []model.Profile, accounts map[string]model.AccountInfo) string {
	res, _ := s.SelectBestProfile(ctx, provider, workspace, candidates, accounts, nil)
	if res == nil || len(res.Evaluations) == 0 {
		return fmt.Sprintf("No profiles configured for provider %s.", provider)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "=== Smart Account Selection Explanation: %s ===\n", strings.ToUpper(provider))
	sb.WriteString("Evaluation of all candidate profiles:\n\n")
	if res.SelectedProfile != nil {
		fmt.Fprintf(&sb, "Optimal Choice: %s (Reason: %s)\n\n", res.SelectedProfile.Name, res.Reason)
	}

	fmt.Fprintf(&sb, "%-18s %-10s %-8s %s\n", "PROFILE", "ELIGIBLE", "SCORE", "BREAKDOWN / REJECTION")
	for _, ev := range res.Evaluations {
		elig := "YES"
		if !ev.Eligible {
			elig = "NO"
		}
		detail := strings.Join(ev.ScoreBreakdown, ", ")
		if !ev.Eligible {
			detail = "REJECTED: " + ev.RejectReason
		}
		fmt.Fprintf(&sb, "%-18s %-10s %-8.1f %s\n", ev.Profile.Name, elig, ev.Score, detail)
	}

	return sb.String()
}
