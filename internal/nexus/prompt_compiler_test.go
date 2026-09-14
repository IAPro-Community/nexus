package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestCompileAgentPromptWithoutSkills(t *testing.T) {
	prompt := "Run tests and refactor"
	compiled, err := CompileAgentPrompt(prompt, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compiled.CompiledPrompt != prompt {
		t.Fatalf("expected untouched prompt %q, got %q", prompt, compiled.CompiledPrompt)
	}
	if len(compiled.ValidatedSkills) != 0 {
		t.Fatalf("expected no validated skills, got %v", compiled.ValidatedSkills)
	}
	if compiled.PromptHash == "" {
		t.Fatal("expected prompt hash to be calculated")
	}
}

func TestCompileTargetPackagePromptUsesPersistedGenericGuidance(t *testing.T) {
	plan := &store.WorkPlan{StructuredFacts: map[string]string{"framework": "go"}}
	pkg := &store.WorkPackage{
		Title:              "Run verification",
		Goal:               "Verify the implementation",
		Role:               "reviewer",
		TaskRequirements:   `{"guidance":{"enabled":true,"source":"maestro","reference":"advice:verify-1","instructions":["use the independent verification gate"]}}`,
		AcceptanceCriteria: []string{"verification is recorded"},
	}
	compiled, err := compileTargetPackagePromptForAgent(context.Background(), plan, pkg, intelligence.AgentSpec{Role: "reviewer"}, []string{"testing"})
	if err != nil {
		t.Fatalf("compile prompt: %v", err)
	}
	if !strings.Contains(compiled.SystemPrompt, "use the independent verification gate") {
		t.Fatalf("persisted guidance was not included in compiled prompt: %s", compiled.SystemPrompt)
	}
}

func TestCompileAgentPromptWithValidatedSkills(t *testing.T) {
	client := &MaestroClient{
		status: MaestroStatus{
			Available: true,
			Mode:      MaestroAssist,
			Capabilities: &MaestroCapability{
				Version: "1.0.0",
				Skills: []MaestroSkillDesc{
					{ID: "skill-security-review", Name: "Security Review"},
					{ID: "skill-repo-health", Name: "Repo Health"},
				},
			},
		},
	}

	userPrompt := "Revise a segurança deste endpoint"
	compiled, err := CompileAgentPrompt(userPrompt, []string{"skill-security-review"}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(compiled.CompiledPrompt, "Nexus execution context") {
		t.Fatalf("expected context envelope, got: %s", compiled.CompiledPrompt)
	}
	if !strings.Contains(compiled.CompiledPrompt, "Scope: next prompt only") {
		t.Fatalf("expected scope, got: %s", compiled.CompiledPrompt)
	}
	if !strings.Contains(compiled.CompiledPrompt, "skill-security-review") {
		t.Fatalf("expected validated skill in compiled prompt, got: %s", compiled.CompiledPrompt)
	}
	if !strings.Contains(compiled.CompiledPrompt, userPrompt) {
		t.Fatalf("expected user prompt in compiled prompt, got: %s", compiled.CompiledPrompt)
	}
}

func TestCompileAgentPromptRejectsDegradedMaestro(t *testing.T) {
	client := &MaestroClient{
		status: MaestroStatus{
			Available: false,
			Mode:      MaestroOff,
			Error:     "maestro offline",
		},
	}

	_, err := CompileAgentPrompt("test prompt", []string{"skill-security-review"}, client)
	if err == nil {
		t.Fatal("expected failure when maestro is offline and skills are requested")
	}
	if !strings.Contains(err.Error(), "MAESTRO_DEGRADED") {
		t.Fatalf("expected MAESTRO_DEGRADED error, got: %v", err)
	}
}

func TestCompileAgentPromptForProjectResolvesBuiltinSkillWithoutMaestro(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus {
		return MaestroStatus{Available: false, Mode: MaestroOff, Error: "maestro offline"}
	}

	compiled, err := n.CompileAgentPromptForProject("", "run the tests", []string{"testing"})
	if err != nil {
		t.Fatalf("builtin skill resolution must not require Maestro: %v", err)
	}
	if len(compiled.ValidatedSkills) != 1 || compiled.ValidatedSkills[0] != "testing" {
		t.Fatalf("unexpected validated skills: %#v", compiled.ValidatedSkills)
	}
	if !strings.Contains(compiled.CompiledPrompt, "testing") {
		t.Fatalf("compiled prompt omitted the generic skill: %s", compiled.CompiledPrompt)
	}
}

func TestCompilePromptVariantsAreTraceableAndDistinct(t *testing.T) {
	brief := newComposerBrief("Implement a notes feature", "Implement a notes feature with tests")
	variants := CompilePromptVariants(brief, []nexusskills.Skill{{ID: "skill-go", Source: nexusskills.SourceBuiltin, Version: "1.2.0"}}, "codex")
	if len(variants) != 3 {
		t.Fatalf("got %d variants", len(variants))
	}
	seen := map[PromptVariantKind]bool{}
	for _, variant := range variants {
		if variant.Hash == "" || variant.Content == "" || !strings.Contains(variant.Content, "source: builtin") {
			t.Fatalf("variant lacks traceability: %+v", variant)
		}
		seen[variant.Kind] = true
	}
	for _, kind := range []PromptVariantKind{PromptVariantGenericPortable, PromptVariantNexusAgent, PromptVariantFlowHandoff} {
		if !seen[kind] {
			t.Fatalf("missing %s", kind)
		}
	}
}
