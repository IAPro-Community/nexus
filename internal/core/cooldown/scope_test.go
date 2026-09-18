package cooldown

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestCooldownIsolatedByAccountScope(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	tracker := NewTracker()
	a := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "a", IdentityVersion: "1"}
	b := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "b", IdentityVersion: "1"}
	tracker.RecordRateLimitForScope(a, time.Minute, nil, "rate limit")
	if limited, _ := tracker.IsRateLimitedForScope(b); limited {
		t.Fatal("account B inherited account A cooldown")
	}
	if limited, _ := tracker.IsRateLimitedForScope(a); !limited {
		t.Fatal("account A cooldown was not recorded")
	}
}
