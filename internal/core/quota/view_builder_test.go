package quota

import (
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestBuildQuotaViewDoesNotInventModelName(t *testing.T) {
	snap := model.UsageSnapshot{
		ProviderID: "agy",
		ProfileID:  "work",
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{{
			Kind:  "5h",
			Group: "gemini",
		}},
	}

	qv := BuildQuotaView(snap, "test@example.com", "Pro")
	if len(qv.ModelGroups) == 0 {
		t.Fatal("expected model groups")
	}
	// Group is "gemini", display name should be "Gemini" or group name, not fabricated model
	if qv.ModelGroups[0].Name == "Gemini 2.5 Flash / Pro" {
		t.Fatalf("fabricated model name found in ModelGroup: %q", qv.ModelGroups[0].Name)
	}
}
