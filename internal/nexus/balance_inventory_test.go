package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func measuredBalanceQuotaView(status string, remaining float64) *quota.QuotaView {
	return &quota.QuotaView{
		Status:    status,
		Source:    "test",
		Available: remaining > 0,
		ModelGroups: []quota.ModelGroup{{
			Key:  "default",
			Name: "Default",
			Windows: []quota.Window{{
				Kind:      "5h",
				Remaining: remaining * 100,
				Status:    status,
			}},
		}},
	}
}

func measuredBalanceAccount(id, provider, profile, status string, remaining float64) ProviderAccount {
	view := measuredBalanceQuotaView(status, remaining)
	view.Provider = provider
	view.Profile = profile
	return ProviderAccount{
		ID:             id,
		Provider:       provider,
		Profile:        profile,
		DisplayName:    provider + " (" + profile + ")",
		Authenticated:  true,
		Available:      view.Available,
		QuotaView:      view,
		QuotaRemaining: remaining,
		QuotaTotal:     1,
		Health:         "healthy",
	}
}

func TestBuildBalanceInventoryUsesOnlyRegisteredMeasuredResources(t *testing.T) {
	live := measuredBalanceAccount("codex:work", "codex", "work", "LIVE", 0.72)
	cached := measuredBalanceAccount("agy:main", "agy", "main", "CACHED", 0.41)
	rateLimited := measuredBalanceAccount("claude:team", "claude", "team", "RATE_LIMITED", 0.08)
	rateLimited.Available = false
	rateLimited.QuotaView.Available = false

	unknown := measuredBalanceAccount("cursor:default", "cursor", "default", "UNKNOWN", 0)
	unsupported := measuredBalanceAccount("opencode:zen", "opencode", "zen", "UNSUPPORTED", 0)
	unauthenticated := measuredBalanceAccount("codex:other", "codex", "other", "LIVE", 0.90)
	unauthenticated.Authenticated = false

	got := BuildBalanceInventory([]ProviderAccount{
		unsupported, unknown, cached, unauthenticated, live, rateLimited,
	})
	if len(got) != 3 {
		t.Fatalf("balance inventory = %#v, want exactly three measured registered entities", got)
	}
	if got[0].ID != "agy:main" || got[1].ID != "claude:team" || got[2].ID != "codex:work" {
		t.Fatalf("balance inventory must be deterministic by provider/profile, got %#v", got)
	}
	if got[1].Available {
		t.Fatalf("rate-limited entity must remain visible as measured evidence but unavailable: %#v", got[1])
	}
	if len(got[2].ModelGroups) != 1 || !got[2].ModelGroups[0].Known {
		t.Fatalf("expected bounded known model-group capacity, got %#v", got[2].ModelGroups)
	}
}

func TestBalanceCandidateAccountsRejectsUnknownAndUnsupportedQuota(t *testing.T) {
	live := measuredBalanceAccount("codex:work", "codex", "work", "LIVE", 0.60)
	unknown := measuredBalanceAccount("cursor:default", "cursor", "default", "UNKNOWN", 0)
	unsupported := measuredBalanceAccount("opencode:zen", "opencode", "zen", "UNSUPPORTED", 0)

	got := BalanceCandidateAccounts([]ProviderAccount{unknown, live, unsupported})
	if len(got) != 1 || got[0].ID != live.ID {
		t.Fatalf("automatic balance candidates = %#v, want only %s", got, live.ID)
	}
}

func TestBalanceInventoryDoesNotInventUnregisteredIdentity(t *testing.T) {
	noProvider := measuredBalanceAccount("ghost", "", "default", "LIVE", 0.50)
	noProfile := measuredBalanceAccount("ghost", "codex", "", "LIVE", 0.50)

	if got := BuildBalanceInventory([]ProviderAccount{noProvider, noProfile}); len(got) != 0 {
		t.Fatalf("unregistered identities must not enter balance inventory: %#v", got)
	}
}

func TestBuildBalanceInventoryPreservesAccountScope(t *testing.T) {
	account := measuredBalanceAccount("codex:work", "codex", "work", "LIVE", 0.72)
	account.Scope = model.AccountScope{
		ProviderID:      "codex",
		ProfileID:       "work",
		AccountID:       "acct-123",
		IdentityVersion: "v4",
	}
	got := BuildBalanceInventory([]ProviderAccount{account})
	if len(got) != 1 || got[0].Scope != account.Scope {
		t.Fatalf("balance account scope = %#v, want %#v", got, account.Scope)
	}
}

func TestBalanceInventoryRejectsSyntheticAggregateWithoutKnownWindow(t *testing.T) {
	account := ProviderAccount{
		ID:             "codex:empty",
		Provider:       "codex",
		Profile:        "empty",
		Authenticated:  true,
		Available:      true,
		Health:         "healthy",
		QuotaRemaining: 0,
		QuotaTotal:     1,
		QuotaView:      &quota.QuotaView{Status: "LIVE"},
	}
	if got := BuildBalanceInventory([]ProviderAccount{account}); len(got) != 0 {
		t.Fatalf("LIVE without a known quota window must not become balance evidence: %#v", got)
	}
}

func TestBalanceInventoryRetainsMeasuredRateLimitForExplanation(t *testing.T) {
	account := ProviderAccount{
		ID:            "codex:limited",
		Provider:      "codex",
		Profile:       "limited",
		Authenticated: true,
		Available:     false,
		RateLimited:   true,
		Health:        "healthy",
		QuotaView:     &quota.QuotaView{Status: "RATE_LIMITED"},
	}
	got := BuildBalanceInventory([]ProviderAccount{account})
	if len(got) != 1 || got[0].Available {
		t.Fatalf("measured rate-limit must remain visible but unavailable: %#v", got)
	}
}
