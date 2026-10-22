package nexus

import (
	"encoding/json"
	"testing"

	"github.com/IAPro-Community/nexus/internal/profile"
)

func TestResolveRegisteredProviderPreferenceUsesOnlyProfileRegistry(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())

	if _, err := profile.Create("codex", "work"); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.Create("claude", "review"); err != nil {
		t.Fatal(err)
	}

	n := &Nexus{}
	provider, ok, err := n.ResolveRegisteredProviderPreference("estou fazendo review; chame o codex para validar as skills")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || provider != "codex" {
		t.Fatalf("registered codex request = %q, %t; want codex,true", provider, ok)
	}

	if provider, ok, err = n.ResolveRegisteredProviderPreference("peça ao Codex para revisar isso"); err != nil || !ok || provider != "codex" {
		t.Fatalf("localized explicit provider request = %q, %t, err=%v; want codex,true", provider, ok, err)
	}

	if provider, ok, err = n.ResolveRegisteredProviderPreference("a documentação do Codex diz isso"); err != nil || ok || provider != "" {
		t.Fatalf("incidental provider mention must not become routing authority: provider=%q ok=%t err=%v", provider, ok, err)
	}

	if provider, ok, err = n.ResolveRegisteredProviderPreference("chame o grok para validar"); err != nil || ok || provider != "" {
		t.Fatalf("unregistered provider must not become a routing candidate: provider=%q ok=%t err=%v", provider, ok, err)
	}

	if provider, ok, err = n.ResolveRegisteredProviderPreference("compare codex e claude para esse review"); err != nil || ok || provider != "" {
		t.Fatalf("multiple registered providers must not invent a winner: provider=%q ok=%t err=%v", provider, ok, err)
	}
}

func TestResolveRegisteredProviderPreferenceSupportsNormalizedSpelling(t *testing.T) {
	provider, ok := resolveRegisteredProviderPreference("use open-code para isso", []string{"opencode"})
	if !ok || provider != "opencode" {
		t.Fatalf("provider=%q ok=%t, want opencode,true", provider, ok)
	}
	provider, ok = resolveRegisteredProviderPreference("use open code para isso", []string{"opencode"})
	if !ok || provider != "opencode" {
		t.Fatalf("spaced provider=%q ok=%t, want opencode,true", provider, ok)
	}
}

func TestResolveExplicitReadOnlyProviderDelegateKeepsStickyLeadByDefault(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	if _, err := profile.Create("codex", "review"); err != nil {
		t.Fatal(err)
	}
	n := &Nexus{}
	provider, delegated, err := n.ResolveExplicitReadOnlyProviderDelegate("peça ao Codex para revisar isso")
	if err != nil || !delegated || provider != "codex" {
		t.Fatalf("expected explicit review delegation to codex, got %q %t %v", provider, delegated, err)
	}
	for _, prompt := range []string{
		"analise a documentação do Codex",
		"use Codex para implementar uma API",
		"compare Codex com Claude",
		"corrija o teste unitário",
	} {
		provider, delegated, err = n.ResolveExplicitReadOnlyProviderDelegate(prompt)
		if err != nil || delegated || provider != "" {
			t.Fatalf("sticky lead must KEEP %q: provider=%q delegated=%t err=%v", prompt, provider, delegated, err)
		}
	}
}

func TestApplyRegisteredProviderPreferenceToFlowPinsExplicitProvider(t *testing.T) {
	flow := FlowDefinition{Steps: []FlowStep{{
		ID:                 "step-1",
		TaskRequirements:   taskRequirementsJSON("revise o código", "reviewer"),
		AssignmentStrategy: FlowAssignmentAuto,
	}}}
	applyRegisteredProviderPreferenceToFlow(&flow, "codex")
	step := flow.Steps[0]
	if step.DesiredProvider != "codex" {
		t.Fatalf("desired provider=%q want codex", step.DesiredProvider)
	}
	if step.ResourcePolicy != string(PolicyManual) {
		t.Fatalf("resource policy=%q want %q", step.ResourcePolicy, PolicyManual)
	}
	if step.Provider != "" || step.Profile != "" {
		t.Fatalf("provider PIN should leave profile/account selection to the scheduler: %+v", step)
	}
	var req TaskRequirements
	if err := json.Unmarshal([]byte(step.TaskRequirements), &req); err != nil {
		t.Fatal(err)
	}
	if req.PreferProvider != "codex" {
		t.Fatalf("task requirement prefer_provider=%q want codex", req.PreferProvider)
	}
}

func TestApplyRegisteredProviderPreferenceDoesNotOverwriteExplicitResource(t *testing.T) {
	flow := FlowDefinition{Steps: []FlowStep{{
		ID:               "step-1",
		Provider:         "claude",
		Profile:          "review",
		ResourcePolicy:   string(PolicyManual),
		TaskRequirements: taskRequirementsJSON("revise o código", "reviewer"),
	}}}
	applyRegisteredProviderPreferenceToFlow(&flow, "codex")
	step := flow.Steps[0]
	if step.Provider != "claude" || step.Profile != "review" || step.ResourcePolicy != string(PolicyManual) {
		t.Fatalf("explicit resource contract was overwritten: %+v", step)
	}
}
