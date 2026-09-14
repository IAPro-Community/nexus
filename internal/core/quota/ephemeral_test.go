package quota

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestEphemeralExecutionDoesNotPersistUsage(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	engine := NewEngine(time.Minute)
	scope := model.AccountScope{ProviderID: "codex", ProfileID: "ephemeral", AccountID: "unmanaged", IdentityVersion: "1"}
	err := engine.SaveUsageForExecution(scope, model.UsageSnapshot{Status: model.UsageLive, Source: model.SourceObservation, Windows: []model.UsageWindow{{Kind: "5h"}}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := engine.GetCachedUsageForScope(scope); found {
		t.Fatal("ephemeral execution contaminated account usage cache")
	}
}
