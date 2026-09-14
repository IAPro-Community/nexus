package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestPromptIsShallowAndRefineryUnknowns(t *testing.T) {
	shallow := newComposerBrief("melhorar o dashboard", "")
	if !PromptIsShallow(shallow) {
		t.Fatalf("vague goal should be shallow: %+v", shallow)
	}
	refreshComposerBrief(&shallow)
	for _, u := range shallow.Unknowns {
		if u.Source == "ARCHETYPE" && u.Severity == PromptUnknownBlocking && u.Status == PromptUnknownOpen {
			t.Fatalf("shallow prompt must not keep ritual ARCHETYPE as BLOCKING open: %+v", u)
		}
	}
	deep := newComposerBrief("Implemente login OAuth com CSRF e testes e2e verdes no fluxo de auth", "")
	if PromptIsShallow(deep) {
		t.Fatalf("complete goal should not be shallow")
	}
	refreshComposerBrief(&deep)
	for _, u := range deep.Unknowns {
		if u.Source == "ARCHETYPE" && u.Severity == PromptUnknownBlocking && u.Status == PromptUnknownOpen {
			t.Fatalf("deep prompt must not keep ritual BLOCKING open: %+v", u)
		}
	}
}

func TestApplyMemoriesToBriefSkipsVerificationRitual(t *testing.T) {
	brief := newComposerBrief("melhorar o dashboard", "")
	ApplyMemoriesToBrief(&brief, map[string]string{
		"quality.verification": "make quality",
		"quality.testing":      "make test:frontend",
	})
	refreshComposerBrief(&brief)
	for _, u := range brief.Unknowns {
		if u.Field == "quality.verification" && u.Status == PromptUnknownOpen {
			t.Fatalf("memory should cover verification: %+v", u)
		}
		if u.Field == "quality.testing" && u.Status == PromptUnknownOpen {
			t.Fatalf("memory should cover testing: %+v", u)
		}
	}
}

func TestMergeIntelligenceUnknownsPreservesAnswers(t *testing.T) {
	brief := LivingBrief{Goal: "ship feature"}
	brief.Unknowns = []PromptUnknown{{
		ID: "scope", Field: "scope.in_scope", Question: "old", Status: PromptUnknownAnswered,
		Answer: "backend only", Severity: PromptUnknownBlocking, Source: "USER",
	}}
	MergeIntelligenceUnknowns(&brief, []intelligence.AmbiguityItem{{
		Key: "scope.in_scope", Level: intelligence.AmbiguityBlocking, Question: "new question?",
		SuggestedOptions: []string{"a", "b"},
	}})
	if brief.Unknowns[0].Status != PromptUnknownAnswered || brief.Unknowns[0].Answer != "backend only" {
		t.Fatalf("resolved answer must be preserved: %+v", brief.Unknowns[0])
	}
	MergeIntelligenceUnknowns(&brief, []intelligence.AmbiguityItem{{
		Key: "quality.acceptance_criteria", Level: intelligence.AmbiguityBlocking, Question: "What is done?",
		SuggestedOptions: []string{"tests pass", "UI ships"},
	}})
	found := false
	for _, u := range brief.Unknowns {
		if u.Field == "quality.acceptance_criteria" {
			found = true
			if len(u.SuggestedOptions) != 2 {
				t.Fatalf("expected options: %+v", u)
			}
		}
	}
	if !found {
		t.Fatal("expected intelligence unknown to be merged")
	}
}

func TestPromoteHabitMemorySecondAnswer(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Mem Proj", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := n.PromoteHabitMemory(ctx, project.ID, "quality.verification", "make quality", MemorySourceComposer, false); err != nil {
		t.Fatal(err)
	}
	if err := n.PromoteHabitMemory(ctx, project.ID, "quality.verification", "make quality", MemorySourceComposer, false); err != nil {
		t.Fatal(err)
	}
	facts, err := n.MemoryFactsMap(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if facts["quality.verification"] != "make quality" {
		t.Fatalf("expected durable habit memory, got %#v", facts)
	}
}

func TestLooksLikeSecretBlocksMemory(t *testing.T) {
	if !LooksLikeSecret("api_key=sk-secret") {
		t.Fatal("expected secret detection")
	}
	if LooksLikeSecret("make quality") {
		t.Fatal("make quality is not a secret")
	}
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Secret Proj", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = n.UpsertIntelligenceMemory(context.Background(), MemoryScopeProject, project.ID, "quality.verification", "api_key=secret", MemorySourceExplicit)
	if err == nil {
		t.Fatal("expected secret rejection")
	}
}
