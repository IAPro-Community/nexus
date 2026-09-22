package nexus

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerContextManifestIsBoundedAndExplainsDroppedItems(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Context Budget", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	brief := newComposerBrief("debug authentication security regression", "")
	for i := 0; i < 20; i++ {
		brief.Context.ReferencedFiles = append(brief.Context.ReferencedFiles, ContextFileRef{
			Path:    "context/file-" + string(rune('a'+i)) + ".txt",
			Excerpt: strings.Repeat("bounded evidence ", 300),
			Hash:    "hash",
		})
	}
	manifest, err := n.BuildComposerContextManifest(context.Background(), project.ID, brief)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Budget.UsedEstimatedTokens > manifest.Budget.MaxEstimatedTokens {
		t.Fatalf("budget exceeded: %+v", manifest.Budget)
	}
	if manifest.Budget.KeptItems == 0 || manifest.Budget.DroppedItems == 0 {
		t.Fatalf("expected both kept and dropped context under pressure: %+v", manifest.Budget)
	}
	for _, item := range manifest.Items {
		if item.Decision == ComposerContextDrop && !strings.Contains(item.Reason, "budget") {
			t.Fatalf("dropped item lacks explicit reason: %+v", item)
		}
	}
}

func TestComposerContextManifestKeepsMissionHistoryReferenceOnly(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	project, err := st.CreateProject(store.Project{Name: "Mission Context", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := st.UpsertMissionRun(store.MissionRunRecord{
		ID: "run-context-manifest", PlanID: "plan-context-manifest", ProjectID: project.ID,
		State: "COMPLETED_VERIFIED", PayloadJSON: "{}", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	manifest, err := n.BuildComposerContextManifest(context.Background(), project.ID, newComposerBrief("review prior mission evidence", ""))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range manifest.Items {
		if item.Kind != "MISSION_REFERENCE" || item.Locator != "run-context-manifest" {
			continue
		}
		found = true
		if item.Decision != ComposerContextReferenceOnly {
			t.Fatalf("mission history leaked into model context: %+v", item)
		}
	}
	if !found {
		t.Fatalf("mission reference missing: %+v", manifest.Items)
	}
	projection := manifest.IntelligenceContext()
	for _, raw := range projection["items"].([]map[string]any) {
		if raw["kind"] == "MISSION_REFERENCE" {
			t.Fatal("reference-only mission was included in intelligence context")
		}
	}
}

func TestRankComposerSkillCandidatesLoadsOnlyRelevantSkills(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }
	catalog, err := n.skillCatalogForProject("")
	if err != nil {
		t.Fatal(err)
	}
	ranked := rankComposerSkillCandidates(catalog, newComposerBrief("debug security authentication tests", ""))
	if len(ranked) == 0 {
		t.Fatal("expected relevant builtin skill candidates")
	}
	seen := map[string]bool{}
	for _, candidate := range ranked {
		seen[candidate.Skill.ID] = true
		if candidate.Score <= 0 || len(candidate.Reasons) == 0 {
			t.Fatalf("candidate is not explainable: %+v", candidate)
		}
	}
	if !seen["debugging"] && !seen["security"] && !seen["testing"] {
		t.Fatalf("relevant skills were not surfaced: %+v", ranked)
	}
}
