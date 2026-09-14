package quota

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestScopedUsageRejectsOtherIdentity(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	engine := NewEngine(time.Minute)
	a := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "account-a", IdentityVersion: "1"}
	b := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "account-b", IdentityVersion: "1"}
	snap := model.UsageSnapshot{Status: model.UsageLive, Source: model.SourceObservation, FetchedAt: time.Now(), Windows: []model.UsageWindow{{Kind: "5h"}}}
	if err := engine.SaveUsageForScope(a, snap); err != nil {
		t.Fatal(err)
	}
	if _, found := engine.GetCachedUsageForScope(b); found {
		t.Fatal("account B must not read account A usage")
	}
	got, found := engine.GetCachedUsageForScope(a)
	if !found || got.AccountScope.Key() != a.Key() {
		t.Fatalf("account A scope not recovered: found=%v got=%+v", found, got.AccountScope)
	}
}
