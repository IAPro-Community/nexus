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
