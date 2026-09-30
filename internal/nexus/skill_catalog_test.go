package nexus

import (
	"context"
	"testing"

	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

func TestGenericSkillIDsResolveWithoutMaestro(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus {
		return MaestroStatus{Available: false, Mode: MaestroOff, Error: "offline for test"}
	}

	got, err := validateGenericSkillIDs(n, "", []string{"coding", "testing", "coding"})
	if err != nil {
		t.Fatalf("builtin skills must resolve without Maestro: %v", err)
	}
	if len(got) != 2 || got[0] != "coding" || got[1] != "testing" {
		t.Fatalf("generic skill IDs were not deterministic/deduplicated: %v", got)
	}
	if _, err := validateGenericSkillIDs(n, "", []string{"missing-skill"}); err == nil {
		t.Fatal("unknown generic skill must be rejected")
	}
}

func TestResolveSkillReferencesPreservesSelectionProvenance(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }

	resolutions, err := resolveSkillReferences(n, "", []string{"testing", "testing"})
	if err != nil {
		t.Fatalf("resolve skill references: %v", err)
	}
	if len(resolutions) != 1 {
		t.Fatalf("expected deterministic deduplicated resolution, got %+v", resolutions)
	}
	resolution := resolutions[0]
	if resolution.RequestedCapability != "testing" || len(resolution.Candidates) != 1 || resolution.Selected != "testing" || resolution.Source != nexusskills.SourceBuiltin || resolution.Version == "" || resolution.Hash == "" || resolution.Reason == "" {
		t.Fatalf("skill provenance is incomplete: %+v", resolution)
	}
}

func TestBuiltinCatalogCoversCoreExecutionSkillsWithoutMaestro(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }

	requested := []string{"verification", "agent-selection", "resource-selection", "model-selection"}
	got, err := validateGenericSkillIDs(n, "", requested)
	if err != nil {
		t.Fatalf("core execution skills must resolve without Maestro: %v", err)
	}
	if len(got) != len(requested) {
		t.Fatalf("resolved skills=%v, want %v", got, requested)
	}
}

func TestMaestroSkillSourceIsOptionalAndGeneric(t *testing.T) {
	offline := &MaestroClient{status: MaestroStatus{Available: false, Mode: MaestroOff}}
	source := NewMaestroSkillSource(offline)
	if source.Available(context.Background()) {
		t.Fatal("offline Maestro must not make its source available")
	}
	if _, err := source.Discover(context.Background()); err != nexusskills.ErrSourceUnavailable {
		t.Fatalf("expected optional source error, got %v", err)
	}

	online := &MaestroClient{status: MaestroStatus{
		Available: true, Mode: MaestroAssist,
		Capabilities: &MaestroCapability{Version: "1.0.0", Skills: []MaestroSkillDesc{{ID: "debug", Name: "Debug", Prompt: "Use evidence."}}},
	}}
	items, err := NewMaestroSkillSource(online).Discover(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("generic Maestro discovery failed: %v %+v", err, items)
	}
	if items[0].Source != nexusskills.SourceMaestro || items[0].Instructions != "Use evidence." {
		t.Fatalf("Maestro-specific type leaked or contract lost: %+v", items[0])
	}
}


func TestAutomaticSkillResolutionUsesKnownCatalogOnly(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }

	got, err := n.ResolveAutomaticSkillIDs(context.Background(), "", "estou fazendo review e quero validar as skills")
	if err != nil {
		t.Fatalf("automatic skill resolution: %v", err)
	}
	if len(got) != 2 || got[0] != "review" || got[1] != "verification" {
		t.Fatalf("automatic skills=%v, want [review verification]", got)
	}
}

func TestAutomaticSkillResolutionIsBoundedAndDeterministic(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }

	got, err := n.ResolveAutomaticSkillIDs(context.Background(), "", "refatore frontend backend com testes, validação, segurança e arquitetura")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > maxAutomaticSkills {
		t.Fatalf("automatic skill set must stay bounded, got %v", got)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one known automatic skill")
	}
	for _, id := range got {
		if _, err := validateGenericSkillIDs(n, "", []string{id}); err != nil {
			t.Fatalf("automatic resolver selected unknown skill %q: %v", id, err)
		}
	}
}


func TestAutomaticSkillResolutionPrefersRegisteredSpecialistsWithoutDuplicateBuiltin(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus {
		return MaestroStatus{
			Available: true,
			Mode:      MaestroAssist,
			Capabilities: &MaestroCapability{Version: "1.0.0", Skills: []MaestroSkillDesc{
				{
					ID:          "code-review",
					Name:        "Code Review",
					Description: "Run a comprehensive code review.",
				},
				{
					ID:          "skill-quality-gate",
					Name:        "Quality Gate",
					Description: "Avalia skills, plugins e MCPs antes da adoção.",
					Triggers:    []string{"quality gate", "skill gate", "avaliar skill", "auditar plugin"},
				},
			}},
		}
	}

	got, err := n.ResolveAutomaticSkillIDs(context.Background(), "", "estou fazendo review e quero validar as skills")
	if err != nil {
		t.Fatal(err)
	}
	has := func(id string) bool {
		for _, item := range got {
			if item == id {
				return true
			}
		}
		return false
	}
	if !has("code-review") || !has("skill-quality-gate") || !has("verification") {
		t.Fatalf("registered specialist resolution incomplete: %v", got)
	}
	if has("review") {
		t.Fatalf("builtin review must not duplicate registered code-review in bounded context: %v", got)
	}
}
