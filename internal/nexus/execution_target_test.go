package nexus

import (
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestModelCatalogNeverInventsModels(t *testing.T) {
	catalog := NewModelCatalog([]ModelCatalogEntry{{
		Candidate:  ModelCandidate{Provider: "opencode", Profile: "work", Model: "observed-model"},
		Source:     "test-observation",
		ObservedAt: time.Now().UTC(),
	}})
	if !catalog.Has("opencode", "work", "observed-model") {
		t.Fatal("observed model missing from catalog")
	}
	if catalog.Has("opencode", "work", "kimi") || catalog.Has("opencode", "work", "glm-5.3") {
		t.Fatal("ModelCatalog must not invent named models")
	}
}

func TestExecutionTargetTreatsUnsupportedQuotaAsOpaqueEligible(t *testing.T) {
	view := quota.QuotaView{Status: string(model.UsageUnsupported)}
	account := ProviderAccount{
		Provider:      "opencode",
		Profile:       "work",
		Authenticated: true,
		Health:        "healthy",
		QuotaView:     &view,
	}
	targets := BuildExecutionTargets([]ProviderAccount{account}, nil)
	if len(targets) != 1 {
		t.Fatalf("targets=%d want 1", len(targets))
	}
	if !targets[0].RuntimeReady {
		t.Fatalf("OPAQUE quota must remain eligible: %+v", targets[0])
	}
	if targets[0].CapacityEvidence != "OPAQUE" {
		t.Fatalf("capacity evidence=%q want OPAQUE", targets[0].CapacityEvidence)
	}
}

func TestCapacityDomainCollapsesSameVerifiedAccount(t *testing.T) {
	scope := model.AccountScope{ProviderID: "codex", ProfileID: "a", AccountID: "acct-1", IdentityVersion: "v1"}
	a := ProviderAccount{Provider: "codex", Profile: "a", Scope: scope}
	scope.ProfileID = "b"
	b := ProviderAccount{Provider: "codex", Profile: "b", Scope: scope}
	left := capacityDomainForAccount(a)
	right := capacityDomainForAccount(b)
	if left.ID != right.ID {
		t.Fatalf("same upstream account must share capacity domain: %q != %q", left.ID, right.ID)
	}
}

func TestResolveExecutionTargetRejectsMissingPinnedModel(t *testing.T) {
	view := quota.QuotaView{Status: string(model.UsageUnsupported)}
	account := ProviderAccount{
		Provider:      "opencode",
		Profile:       "work",
		Authenticated: true,
		Health:        "healthy",
		QuotaView:     &view,
	}
	_, _, err := ResolveExecutionTarget(
		TaskRequirements{TaskKind: "coding"},
		RuntimeAffinityPolicy{Model: AffinityPreference{Mode: AffinityPin, Value: "missing-model"}},
		[]ProviderAccount{account},
		NewModelCatalog(nil),
	)
	if err == nil || !strings.Contains(err.Error(), "absent from ModelCatalog") {
		t.Fatalf("missing pinned model must fail clearly, got %v", err)
	}
}

func TestBuildExecutionTargetsHonorsExplicitCapacityDomainEvidence(t *testing.T) {
	view := quota.QuotaView{Status: string(model.UsageLive), Available: true}
	account := ProviderAccount{
		Provider:      "agy",
		Profile:       "work",
		Authenticated: true,
		Available:     true,
		Health:        "healthy",
		QuotaView:     &view,
	}
	catalog := NewModelCatalog([]ModelCatalogEntry{{
		Candidate: ModelCandidate{
			Provider: "agy", Profile: "work", Model: "real-model",
			Healthy: true, Authenticated: true, QuotaAvailable: true,
		},
		Source:           "provider-discovery",
		CapacityDomainID: "agy/account-1/family-a",
	}})
	targets := BuildExecutionTargets([]ProviderAccount{account}, catalog)
	if len(targets) != 1 || targets[0].CapacityDomainID != "agy/account-1/family-a" {
		t.Fatalf("explicit capacity-domain evidence lost: %+v", targets)
	}
}

func TestSpecialistPreferenceDoesNotOverrideExplicitPin(t *testing.T) {
	catalog := NewModelCatalog([]ModelCatalogEntry{{
		Candidate: ModelCandidate{Provider: "opencode", Model: "kimi"},
		Source:    "test",
	}})
	affinity := RuntimeAffinityPolicy{
		Provider: AffinityPreference{Mode: AffinityPin, Value: "codex"},
		Model:    AffinityPreference{Mode: AffinityAuto},
	}
	got := applySpecialistExecutionPreferences(TaskRequirements{Role: "frontend-engineer"}, affinity, catalog)
	if got.Provider.Mode != AffinityPin || got.Provider.Value != "codex" {
		t.Fatalf("specialist preference overrode explicit provider PIN: %+v", got.Provider)
	}
}

func TestSpecialistPreferenceUsesCatalogBackedModelWhenAutomatic(t *testing.T) {
	catalog := NewModelCatalog([]ModelCatalogEntry{{
		Candidate: ModelCandidate{Provider: "opencode", Model: "kimi"},
		Source:    "test",
	}})
	got := applySpecialistExecutionPreferences(
		TaskRequirements{Role: "frontend-engineer"},
		RuntimeAffinityPolicy{
			Provider: AffinityPreference{Mode: AffinityAuto},
			Model:    AffinityPreference{Mode: AffinityAuto},
		},
		catalog,
	)
	if got.Provider.Mode != AffinityPrefer || got.Provider.Value != "opencode" {
		t.Fatalf("provider preference=%+v want opencode PREFER", got.Provider)
	}
	if got.Model.Mode != AffinityPrefer || got.Model.Value != "kimi" {
		t.Fatalf("model preference=%+v want kimi PREFER", got.Model)
	}
}
