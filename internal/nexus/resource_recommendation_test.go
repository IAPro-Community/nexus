package nexus

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestResourceRecommendation_Balanced(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "codex-default",
			Provider:       "codex",
			Profile:        "default",
			DisplayName:    "OpenAI Codex (Default)",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.85,
			Health:         "healthy",
		},
		{
			ID:             "claude-default",
			Provider:       "claude",
			Profile:        "default",
			DisplayName:    "Anthropic Claude",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.40,
			Health:         "healthy",
		},
		{
			ID:            "gemini-unauthed",
			Provider:      "gemini",
			Profile:       "default",
			DisplayName:   "Google Gemini",
			Authenticated: false,
			Available:     false,
			Health:        "unknown",
		},
	}

	req := TaskRequirements{
		TaskKind: "coding",
		Role:     "implementer",
	}

	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended == nil {
		t.Fatalf("expected recommended account, got nil")
	}

	if res.Recommended.Account.Provider != "codex" {
		t.Errorf("expected codex to be recommended due to higher quota, got %s", res.Recommended.Account.Provider)
	}

	if len(res.Candidates) != 3 {
		t.Errorf("expected 3 candidates, got %d", len(res.Candidates))
	}

	// Unauthenticated account must be ineligible
	for _, c := range res.Candidates {
		if c.Account.Provider == "gemini" && c.Eligible {
			t.Errorf("unauthenticated gemini must be marked ineligible")
		}
	}
}

func TestResourceRecommendation_ProviderPriorityControlsFallbackOrder(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "agy", Provider: "agy", Profile: "one", DisplayName: "AGY", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.95},
		{ID: "codex", Provider: "codex", Profile: "one", DisplayName: "Codex", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.20},
	}
	res := RecommendResources(accounts, TaskRequirements{ProviderPriorities: map[string]int{"codex": 1, "agy": 2}}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.Provider != "codex" {
		t.Fatalf("expected configured provider priority to win fallback order, got %+v", res.Recommended)
	}
}

func TestResourceRecommendation_PreserveQuota(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "codex-low",
			Provider:       "codex",
			Profile:        "default",
			DisplayName:    "Codex Low Quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.15, // < 30%
			Health:         "healthy",
		},
		{
			ID:             "claude-high",
			Provider:       "claude",
			Profile:        "default",
			DisplayName:    "Claude High Quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.90, // > 70%
			Health:         "healthy",
		},
	}

	req := TaskRequirements{
		TaskKind: "coding",
		Role:     "implementer",
	}

	res := RecommendResources(accounts, req, PolicyPreserveQuota)
	if res.Recommended == nil || res.Recommended.Account.Provider != "claude" {
		t.Errorf("PreserveQuota policy must choose claude with 90%% quota over codex with 15%%, got %v", res.Recommended)
	}
}

func TestResourceRecommendation_CooldownRejection(t *testing.T) {
	future := time.Now().Add(5 * time.Minute)
	accounts := []ProviderAccount{
		{
			ID:            "codex-cooldown",
			Provider:      "codex",
			Profile:       "default",
			DisplayName:   "Codex in Cooldown",
			Authenticated: true,
			Available:     true,
			CooldownUntil: &future,
			Health:        "degraded",
		},
	}

	res := RecommendResources(accounts, TaskRequirements{}, PolicyBalanced)
	if res.Recommended != nil {
		t.Errorf("account in cooldown must not be recommended")
	}
	if res.Candidates[0].Eligible {
		t.Errorf("account in cooldown must be marked ineligible")
	}
}

func TestResourceRecommendation_UnknownQuotaDoesNotOutrankKnownLiveQuota(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "unknown",
			Provider:       "codex",
			Profile:        "unknown",
			DisplayName:    "Unknown quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "UNKNOWN"},
			QuotaRemaining: 0,
			Health:         "healthy",
		},
		{
			ID:             "known",
			Provider:       "claude",
			Profile:        "known",
			DisplayName:    "Known quota",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.50,
			Health:         "healthy",
		},
	}

	res := RecommendResources(accounts, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.ID != "known" {
		t.Fatalf("known LIVE quota must outrank UNKNOWN quota, got %#v", res.Recommended)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.ID == "unknown" {
			if candidate.Eligible {
				t.Fatal("UNKNOWN quota must be ineligible for automatic allocation")
			}
			if candidate.Confidence != "UNKNOWN" {
				t.Fatalf("unknown quota confidence = %q, want UNKNOWN", candidate.Confidence)
			}
		}
	}
}

func TestResourceRecommendation_UnknownQuotaHardGate(t *testing.T) {
	accounts := []ProviderAccount{{
		ID:            "only-unknown",
		Provider:      "codex",
		Profile:       "default",
		Authenticated: true,
		Available:     true,
		QuotaView:     &quota.QuotaView{Status: "UNKNOWN"},
		Health:        "healthy",
	}}
	res := RecommendResources(accounts, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if res.Recommended != nil {
		t.Fatalf("UNKNOWN-only pool must not recommend, got %#v", res.Recommended)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].Eligible {
		t.Fatalf("expected ineligible UNKNOWN candidate, got %#v", res.Candidates)
	}
}

func TestResourceRecommendation_ConfidenceIsExplicitLiveCachedUnknown(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "live", Provider: "codex", Profile: "default", DisplayName: "Codex live", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.80, Health: "healthy"},
		{ID: "cached", Provider: "agy", Profile: "default", DisplayName: "AGY cached", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "CACHED"}, QuotaRemaining: 0.40, Health: "healthy"},
		{ID: "unknown", Provider: "gemini", Profile: "default", DisplayName: "Gemini unknown", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "UNKNOWN"}, QuotaRemaining: 0.99, Health: "healthy"},
	}
	res := RecommendResources(accounts, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	confidence := map[string]string{}
	for _, candidate := range res.Candidates {
		confidence[candidate.Account.ID] = candidate.Confidence
	}
	if confidence["live"] != "LIVE" {
		t.Fatalf("LIVE quota confidence = %q, want LIVE", confidence["live"])
	}
	if confidence["cached"] != "CACHED" {
		t.Fatalf("CACHED quota confidence = %q, want CACHED", confidence["cached"])
	}
	if confidence["unknown"] != "UNKNOWN" {
		t.Fatalf("UNKNOWN quota confidence = %q, want UNKNOWN", confidence["unknown"])
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.ID == "unknown" && candidate.Eligible {
			t.Fatalf("UNKNOWN quota must stay ineligible: %#v", candidate)
		}
	}
}

func TestResourceRecommendation_PinnedProviderExcludesOtherAccounts(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "codex", Provider: "codex", Profile: "default", DisplayName: "Codex", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.20, Health: "healthy"},
		{ID: "agy", Provider: "agy", Profile: "default", DisplayName: "AGY", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.95, Health: "healthy"},
	}
	req := TaskRequirements{
		TaskKind:        "coding",
		RuntimeAffinity: &RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPin, Value: "codex"}},
	}
	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended == nil {
		t.Fatalf("pinned provider must still be recommended, got %#v", res)
	}
	if res.Recommended.Account.Provider != "codex" {
		t.Fatalf("pinned provider drifted to %s", res.Recommended.Account.Provider)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.Provider == "agy" {
			if candidate.Eligible {
				t.Fatalf("account outside the PIN must not stay eligible: %#v", candidate)
			}
			if !strings.Contains(candidate.RejectionReason, "PIN") {
				t.Fatalf("PIN rejection must be explicit, got %q", candidate.RejectionReason)
			}
		}
	}
}

func TestResourceRecommendation_PinnedProviderUnavailableFailsClosed(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "agy", Provider: "agy", Profile: "default", DisplayName: "AGY", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.95, Health: "healthy"},
	}
	req := TaskRequirements{
		TaskKind:        "coding",
		RuntimeAffinity: &RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPin, Value: "codex"}},
	}
	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended != nil {
		t.Fatalf("PIN must never drift to another provider, got %#v", res.Recommended)
	}
	if !strings.Contains(res.Explanation, "PIN") {
		t.Fatalf("explanation must expose the PIN rejection, got %q", res.Explanation)
	}
}

func TestResourceRecommendation_PinnedProfileExcludesOtherProfiles(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "codex-default", Provider: "codex", Profile: "default", DisplayName: "Codex default", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.90, Health: "healthy"},
		{ID: "codex-work", Provider: "codex", Profile: "work", DisplayName: "Codex work", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.10, Health: "healthy"},
	}
	req := TaskRequirements{
		TaskKind:        "coding",
		RuntimeAffinity: &RuntimeAffinityPolicy{Profile: AffinityPreference{Mode: AffinityPin, Value: "work"}},
	}
	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.Profile != "work" {
		t.Fatalf("pinned profile must win regardless of quota, got %#v", res.Recommended)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.Profile == "default" && candidate.Eligible {
			t.Fatalf("profile outside the PIN must not stay eligible: %#v", candidate)
		}
	}
}

func TestResourceRecommendation_PreferModeKeepsPoolEligible(t *testing.T) {
	accounts := []ProviderAccount{
		{ID: "codex", Provider: "codex", Profile: "default", DisplayName: "Codex", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.90, Health: "healthy"},
		{ID: "agy", Provider: "agy", Profile: "default", DisplayName: "AGY", Authenticated: true, Available: true, QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.20, Health: "healthy"},
	}
	req := TaskRequirements{
		TaskKind:        "coding",
		RuntimeAffinity: &RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}},
	}
	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended == nil {
		t.Fatalf("PREFER must keep the pool eligible, got %#v", res)
	}
	for _, candidate := range res.Candidates {
		if !candidate.Eligible {
			t.Fatalf("PREFER must not exclude accounts: %#v", candidate)
		}
	}
}

func TestResourceRecommendation_RequiredCapabilitiesAreHardGate(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:             "no-resume",
			Provider:       "agy",
			Profile:        "default",
			DisplayName:    "AGY",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.95,
			Health:         "healthy",
			Capabilities: map[string]string{
				"resume": "UNSUPPORTED",
			},
		},
		{
			ID:             "has-resume",
			Provider:       "codex",
			Profile:        "default",
			DisplayName:    "Codex",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.40,
			Health:         "healthy",
			Capabilities: map[string]string{
				"resume": "SUPPORTED",
			},
		},
	}

	res := RecommendResources(accounts, TaskRequirements{RequiredCapabilities: []string{"resume"}}, PolicyBalanced)
	if res.Recommended == nil || res.Recommended.Account.ID != "has-resume" {
		t.Fatalf("required capability must reject unsupported candidate, got %#v", res.Recommended)
	}
	for _, candidate := range res.Candidates {
		if candidate.Account.ID == "no-resume" && candidate.Eligible {
			t.Fatal("candidate without required capability must be ineligible")
		}
	}
}

func TestResourceRecommendationExplainsCapabilityRejection(t *testing.T) {
	accounts := []ProviderAccount{{
		ID:             "codex-review",
		Provider:       "codex",
		Profile:        "review",
		Authenticated:  true,
		Available:      true,
		QuotaView:      &quota.QuotaView{Status: "LIVE"},
		QuotaRemaining: 0.5,
		Health:         "healthy",
		Capabilities: map[string]string{
			"headless":         "SUPPORTED",
			"submit_prompt":    "SUPPORTED",
			"read_only_review": "UNKNOWN",
		},
	}}
	result := RecommendResources(accounts, TaskRequirements{
		RequiredCapabilities: []string{"headless", "submit_prompt", "read_only_review"},
	}, PolicyBalanced)
	if result.Recommended != nil {
		t.Fatalf("unsupported review capability must reject the resource: %+v", result.Recommended)
	}
	for _, want := range []string{"codex:review", "read_only_review", "UNKNOWN"} {
		if !strings.Contains(result.Explanation, want) {
			t.Fatalf("explanation %q does not identify rejection detail %q", result.Explanation, want)
		}
	}
	if strings.Contains(result.Explanation, "sem autenticação") {
		t.Fatalf("explanation must not mislabel a capability rejection: %q", result.Explanation)
	}
}

func TestProviderResourceRequirementsKeepTaskSpecialtiesOutOfControlGate(t *testing.T) {
	req := TaskRequirements{RequiredCapabilities: []string{
		"headless", "submit_prompt", "security", "code_review", "read_only_review",
	}}
	got := providerResourceRequirements(req)
	want := []string{"headless", "submit_prompt", "read_only_review"}
	if !reflect.DeepEqual(got.RequiredCapabilities, want) {
		t.Fatalf("provider capabilities=%v, want %v", got.RequiredCapabilities, want)
	}
	if !reflect.DeepEqual(req.RequiredCapabilities, []string{"headless", "submit_prompt", "security", "code_review", "read_only_review"}) {
		t.Fatalf("helper mutated original task requirements: %v", req.RequiredCapabilities)
	}
}

// TestCapabilityBasedRoleScoring verifies that accounts with relevant
// capabilities receive higher role_fit scores than those without, without
// any provider-name matching.
func TestCapabilityBasedRoleScoring(t *testing.T) {
	lowCaps := ProviderAccount{
		ID:             "low-caps",
		Provider:       "future-provider",
		Profile:        "default",
		Authenticated:  true,
		Available:      true,
		QuotaView:      &quota.QuotaView{Status: "LIVE"},
		QuotaRemaining: 0.85,
		Health:         "healthy",
		Capabilities: map[string]string{
			"headless":      "SUPPORTED",
			"submit_prompt": "SUPPORTED",
		},
	}
	highCaps := ProviderAccount{
		ID:             "high-caps",
		Provider:       "another-future-provider",
		Profile:        "default",
		Authenticated:  true,
		Available:      true,
		QuotaView:      &quota.QuotaView{Status: "LIVE"},
		QuotaRemaining: 0.40,
		Health:         "healthy",
		Capabilities: map[string]string{
			"headless":          "SUPPORTED",
			"submit_prompt":     "SUPPORTED",
			"sessions":          "SUPPORTED",
			"resume":            "SUPPORTED",
			"structured_events": "SUPPORTED",
			"approvals":         "SUPPORTED",
		},
	}
	t.Run("reviewer role prefers structured_events/approvals capabilities", func(t *testing.T) {
		res := RecommendResources([]ProviderAccount{lowCaps, highCaps}, TaskRequirements{Role: "reviewer"}, PolicyBalanced)
		if res.Recommended == nil {
			t.Fatal("expected a recommendation")
		}
		// high-caps has structured_events + approvals → role_fit=20, low-caps → role_fit=10
		// even though low-caps has more quota, the role bonus should tip high-caps
		var lowRoleFit, highRoleFit float64
		for _, c := range res.Candidates {
			if c.Account.ID == "low-caps" {
				lowRoleFit = c.ScoreBreakdown["role_fit"]
			}
			if c.Account.ID == "high-caps" {
				highRoleFit = c.ScoreBreakdown["role_fit"]
			}
		}
		if highRoleFit <= lowRoleFit {
			t.Errorf("high-caps reviewer role_fit (%v) must exceed low-caps (%v)", highRoleFit, lowRoleFit)
		}
	})
	t.Run("architect role prefers sessions+resume capabilities", func(t *testing.T) {
		var lowRoleFit, highRoleFit float64
		for _, c := range []ResourceCandidate{
			evaluateCandidate(lowCaps, TaskRequirements{Role: "architect"}, PolicyBalanced),
			evaluateCandidate(highCaps, TaskRequirements{Role: "architect"}, PolicyBalanced),
		} {
			if c.Account.ID == "low-caps" {
				lowRoleFit = c.ScoreBreakdown["role_fit"]
			}
			if c.Account.ID == "high-caps" {
				highRoleFit = c.ScoreBreakdown["role_fit"]
			}
		}
		if highRoleFit <= lowRoleFit {
			t.Errorf("high-caps architect role_fit (%v) must exceed low-caps (%v)", highRoleFit, lowRoleFit)
		}
	})
}

// TestAccountSupportsHeadlessReview verifies the capability-based reviewer
// check replaces the old provider name whitelist.
func TestAccountSupportsHeadlessReview(t *testing.T) {
	cases := []struct {
		name     string
		caps     map[string]string
		expected bool
	}{
		{
			name:     "both supported",
			caps:     map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
			expected: true,
		},
		{
			name:     "missing headless",
			caps:     map[string]string{"submit_prompt": "SUPPORTED"},
			expected: false,
		},
		{
			name:     "missing submit_prompt",
			caps:     map[string]string{"headless": "SUPPORTED"},
			expected: false,
		},
		{
			name:     "partial headless",
			caps:     map[string]string{"headless": "PARTIAL", "submit_prompt": "SUPPORTED"},
			expected: false,
		},
		{
			name:     "empty capabilities",
			caps:     map[string]string{},
			expected: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			acc := ProviderAccount{Capabilities: tc.caps}
			got := accountSupportsHeadlessReview(acc)
			if got != tc.expected {
				t.Errorf("accountSupportsHeadlessReview(%v) = %v, want %v", tc.caps, got, tc.expected)
			}
		})
	}
}

func TestResourceRecommendation_RejectsUnsupportedQuotaForAutomaticChoice(t *testing.T) {
	opaque := ProviderAccount{
		ID:            "opencode-default",
		Provider:      "opencode",
		Profile:       "default",
		DisplayName:   "OpenCode",
		Authenticated: true,
		Available:     false,
		QuotaView: &quota.QuotaView{
			Status:       "UNSUPPORTED",
			Available:    false,
			AvailReasons: quota.AvailReasons{UnknownQuota: true},
		},
		AvailReasons: &quota.AvailReasons{UnknownQuota: true},
		Health:       "healthy",
	}
	res := RecommendResources([]ProviderAccount{opaque}, TaskRequirements{TaskKind: "coding"}, PolicyBalanced)
	if res.Recommended != nil {
		t.Fatalf("unsupported quota must not authorize automatic recommendation: %+v", res.Recommended)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].Eligible || !strings.Contains(res.Candidates[0].RejectionReason, "Quota não suportada") {
		t.Fatalf("expected explicit unsupported-quota rejection, got %+v", res.Candidates)
	}
}
