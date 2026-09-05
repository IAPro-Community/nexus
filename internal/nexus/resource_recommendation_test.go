package nexus

import (
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/quota"
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
		if candidate.Account.ID == "unknown" && candidate.Confidence != "UNKNOWN" {
			t.Fatalf("unknown quota confidence = %q, want UNKNOWN", candidate.Confidence)
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

func TestRecommendationExplainsRejectionsAndContinuity(t *testing.T) {
	accounts := []ProviderAccount{
		{
			ID:            "unauthed",
			Provider:      "codex",
			Profile:       "work",
			DisplayName:   "Codex Work",
			Authenticated: false,
			Available:     false,
		},
		{
			ID:             "authed-claude",
			Provider:       "claude",
			Profile:        "personal",
			DisplayName:    "Claude Personal",
			Authenticated:  true,
			Available:      true,
			QuotaView:      &quota.QuotaView{Status: "LIVE"},
			QuotaRemaining: 0.8,
			Health:         "healthy",
		},
	}

	req := TaskRequirements{
		CurrentProvider: "codex",
		CurrentProfile:  "work",
	}

	res := RecommendResources(accounts, req, PolicyBalanced)
	if res.Recommended == nil {
		t.Fatal("expected recommendation")
	}
	if len(res.RejectedAlternatives) == 0 {
		t.Fatal("expected rejected alternatives to list unauthed account")
	}
	if res.ContinuityImpact == "" {
		t.Fatal("expected continuity impact explanation for provider change")
	}
	if res.Fingerprint == "" {
		t.Fatal("expected non-empty recommendation fingerprint")
	}
}
