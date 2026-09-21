package profile

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestCapacityProbeRegistryExposesOnlyProductionCollectors(t *testing.T) {
	registry := NewCapacityProbeRegistry()
	want := []string{"codex", "agy", "claude", "gemini"}
	if got := registry.Providers(); !reflect.DeepEqual(got, want) {
		t.Fatalf("providers=%v, want %v", got, want)
	}
	for _, provider := range want {
		probe, ok := registry.For(provider)
		if !ok || probe == nil {
			t.Fatalf("missing production observation probe for %s", provider)
		}
		if err := probe.Descriptor().Validate(); err != nil {
			t.Fatalf("descriptor for %s is unsafe: %v", provider, err)
		}
		if probe.Descriptor().RequiresNetwork {
			t.Fatalf("capacity observation probe for %s must not start live network collection", provider)
		}
	}
	for _, provider := range []string{"cursor", "opencode", "unknown"} {
		if _, ok := registry.For(provider); ok {
			t.Fatalf("%s must remain unsupported until it has trustworthy quota evidence", provider)
		}
	}
}

func TestObservedUsageProbeNeverUpgradesEvidenceOrMutatesInput(t *testing.T) {
	remaining := 74.0
	scope := model.AccountScope{
		ProviderID:      "codex",
		ProfileID:       "work",
		AccountID:       "acct-1",
		IdentityVersion: "v1",
	}
	input := model.Profile{Provider: "codex", Name: "work", AccountScope: scope}
	probe := &observedUsageProbe{
		provider: "codex",
		sources:  []model.UsageSource{model.SourceOfficialAPI},
		read: func(provider, profile string) model.UsageSnapshot {
			if provider != "codex" || profile != "work" {
				t.Fatalf("unexpected lookup %s/%s", provider, profile)
			}
			return model.UsageSnapshot{
				ProviderID: "codex",
				ProfileID:  "work",
				Status:     model.UsageLive,
				Source:     model.SourceOfficialAPI,
				FetchedAt:  time.Now().UTC(),
				Windows: []model.UsageWindow{{
					Kind:             "5h",
					Group:            "codex",
					RemainingPercent: &remaining,
				}},
			}
		},
	}

	got, err := probe.Probe(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.UsageCached {
		t.Fatalf("read-only bridge must downgrade LIVE to CACHED, got %s", got.Status)
	}
	if !got.AccountScope.Verifiable() || got.AccountScope.Key() != scope.Key() {
		t.Fatalf("verified scope was not preserved: %+v", got.AccountScope)
	}
	if input.Provider != "codex" || input.Name != "work" || input.AccountScope.Key() != scope.Key() {
		t.Fatalf("probe mutated input profile: %+v", input)
	}
}

func TestObservedUsageProbeHonorsCancellationAndProviderBoundary(t *testing.T) {
	probe := &observedUsageProbe{
		provider: "codex",
		read: func(string, string) model.UsageSnapshot {
			t.Fatal("reader must not run after cancellation")
			return model.UsageSnapshot{}
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := probe.Probe(ctx, model.Profile{Provider: "codex", Name: "work"}); err == nil {
		t.Fatal("canceled probe must return context error")
	}

	probe.read = func(string, string) model.UsageSnapshot {
		t.Fatal("reader must not run for another provider")
		return model.UsageSnapshot{}
	}
	if _, err := probe.Probe(context.Background(), model.Profile{Provider: "claude", Name: "work"}); err == nil {
		t.Fatal("provider mismatch must fail closed")
	}
}
