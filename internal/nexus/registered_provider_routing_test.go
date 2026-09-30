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

func TestResolveRegisteredProviderPreferenceRequiresExplicitRoutingIntent(t *testing.T) {
	if provider, ok := resolveRegisteredProviderPreference("corrija o bug do cursor na tela", []string{"cursor"}); ok || provider != "" {
		t.Fatalf("incidental provider-name mention must not route: provider=%q ok=%t", provider, ok)
	}
	if provider, ok := resolveRegisteredProviderPreference("use cursor para revisar isso", []string{"cursor"}); !ok || provider != "cursor" {
		t.Fatalf("explicit provider request must route: provider=%q ok=%t", provider, ok)
	}
	if provider, ok := resolveRegisteredProviderPreference("revise com codex antes de responder", []string{"codex"}); !ok || provider != "codex" {
		t.Fatalf("task verb + com provider must route: provider=%q ok=%t", provider, ok)
	}
	if provider, ok := resolveRegisteredProviderPreference("codex revise isso", []string{"codex"}); !ok || provider != "codex" {
		t.Fatalf("provider + task verb must route: provider=%q ok=%t", provider, ok)
	}
	if provider, ok := resolveRegisteredProviderPreference("não use codex para isso", []string{"codex"}); ok || provider != "" {
		t.Fatalf("negated provider request must not route: provider=%q ok=%t", provider, ok)
	}
	if provider, ok := resolveRegisteredProviderPreference("use codex, não claude", []string{"codex", "claude"}); !ok || provider != "codex" {
		t.Fatalf("excluded provider must not make the positive request ambiguous: provider=%q ok=%t", provider, ok)
	}
}

func TestApplyRegisteredProviderPreferenceToFlowUsesPreferNotPin(t *testing.T) {
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
	if step.ResourcePolicy != string(PolicyPreferProvider) {
		t.Fatalf("resource policy=%q want %q", step.ResourcePolicy, PolicyPreferProvider)
	}
	if step.Provider != "" || step.Profile != "" {
		t.Fatalf("registered provider preference must not hard-pin the execution resource: %+v", step)
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
		ID:              "step-1",
		Provider:        "claude",
		Profile:         "review",
		ResourcePolicy:  string(PolicyManual),
		TaskRequirements: taskRequirementsJSON("revise o código", "reviewer"),
	}}}
	applyRegisteredProviderPreferenceToFlow(&flow, "codex")
	step := flow.Steps[0]
	if step.Provider != "claude" || step.Profile != "review" || step.ResourcePolicy != string(PolicyManual) {
		t.Fatalf("explicit resource contract was overwritten: %+v", step)
	}
}
