package profile

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestAPIResourceChannelCacheIsAccountScoped(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)

	scope := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "acct-a", IdentityVersion: "1"}
	state := model.ResourceChannelState{
		Channel: model.ResourceChannelAPI, AccountScope: scope,
		Status: model.UsageLive, Source: model.SourceOfficialAPI,
		Confidence: model.UsageConfidenceHigh, ObservedAt: time.Now().UTC(),
		Consumption: &model.ConsumptionState{InputTokens: 1234},
	}
	if err := saveAPIResourceChannel("codex", "work", state); err != nil {
		t.Fatal(err)
	}

	got, ok := GetAPIResourceChannel("codex", "work", scope)
	if !ok || got.Status != model.UsageCached || got.Consumption == nil || got.Consumption.InputTokens != 1234 {
		t.Fatalf("cached state=%+v ok=%v", got, ok)
	}

	other := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "acct-b", IdentityVersion: "1"}
	if leaked, ok := GetAPIResourceChannel("codex", "work", other); ok {
		t.Fatalf("API telemetry leaked across account scopes: %+v", leaked)
	}
}

func TestAPIResourceChannelOldCacheBecomesEstimated(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)

	scope := model.AccountScope{ProviderID: "codex", ProfileID: "work", AccountID: "acct-a", IdentityVersion: "1"}
	state := model.ResourceChannelState{
		Channel: model.ResourceChannelAPI, AccountScope: scope,
		Status: model.UsageLive, Source: model.SourceOfficialAPI,
		Confidence: model.UsageConfidenceHigh, ObservedAt: time.Now().Add(-10 * time.Minute).UTC(),
	}
	if err := saveAPIResourceChannel("codex", "work", state); err != nil {
		t.Fatal(err)
	}
	got, ok := GetAPIResourceChannel("codex", "work", scope)
	if !ok || got.Status != model.UsageEstimated || got.Confidence != model.UsageConfidenceLow {
		t.Fatalf("old API cache=%+v ok=%v", got, ok)
	}
}
