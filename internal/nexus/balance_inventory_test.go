package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func measuredAccount(id, provider, profile, status string, remaining float64) ProviderAccount {
	view := quota.QuotaView{
		Provider: provider,
		Profile:  profile,
		Status:   status,
		Source:   "test",
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
	return ProviderAccount{
		ID:             id,
		Provider:       provider,
		Profile:        profile,
		DisplayName:    provider + " (" + profile + ")",
		Authenticated:  true,
		Available:      view.Available,
		QuotaView:      &view,
		QuotaRemaining: remaining,
		QuotaTotal:     1,
		Health:         "healthy",
	}
}

func TestBuildBalanceInventoryUsesOnlyRegisteredMeasuredResources(t *testing.T) {
	live := measuredAccount("codex:work", "codex", "work", "LIVE", 0.72)
	cached := measuredAccount("agy:main", "agy", "main", "CACHED", 0.41)
	rateLimited := measuredAccount("claude:team", "claude", "team", "RATE_LIMITED", 0.08)
	rateLimited.Available = false
	rateLimited.QuotaView.Available = false

	unknown := measuredAccount("cursor:default", "cursor", "default", "UNKNOWN", 0)
	unsupported := measuredAccount("opencode:zen", "opencode", "zen", "UNSUPPORTED", 0)
	unauthenticated := measuredAccount("codex:other", "codex", "other", "LIVE", 0.90)
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
	live := measuredAccount("codex:work", "codex", "work", "LIVE", 0.60)
	unknown := measuredAccount("cursor:default", "cursor", "default", "UNKNOWN", 0)
	unsupported := measuredAccount("opencode:zen", "opencode", "zen", "UNSUPPORTED", 0)

	got := BalanceCandidateAccounts([]ProviderAccount{unknown, live, unsupported})
	if len(got) != 1 || got[0].ID != live.ID {
		t.Fatalf("automatic balance candidates = %#v, want only %s", got, live.ID)
	}
}

func TestBalanceInventoryDoesNotInventUnregisteredIdentity(t *testing.T) {
	noProvider := measuredAccount("ghost", "", "default", "LIVE", 0.50)
	noProfile := measuredAccount("ghost", "codex", "", "LIVE", 0.50)

	if got := BuildBalanceInventory([]ProviderAccount{noProvider, noProfile}); len(got) != 0 {
		t.Fatalf("unregistered identities must not enter balance inventory: %#v", got)
	}
}
