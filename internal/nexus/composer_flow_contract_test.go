package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestApplyBriefExtractionMergesFieldsAndRecordsInferredFacts(t *testing.T) {
	brief := newComposerBrief("Ship feature", "Ship feature")
	before := len(brief.Facts)
	applyBriefExtraction(&brief, &intelligence.BriefExtraction{
		InScope:            []string{"backend", "frontend"},
		Risks:              []string{"data loss"},
		Verification:       []string{"go test ./..."},
		AcceptanceCriteria: []string{"tests pass"},
	})
	if len(brief.Scope.InScope) != 2 || brief.Scope.InScope[0] != "backend" {
		t.Fatalf("in_scope not merged: %#v", brief.Scope.InScope)
	}
	if len(brief.Risks) != 1 || len(brief.Quality.Verification) != 1 || len(brief.Quality.AcceptanceCriteria) != 1 {
		t.Fatalf("brief fields not merged: %+v", brief)
	}
	inferred := 0
	for _, fact := range brief.Facts {
		if fact.SourceType == "INTELLIGENCE" && fact.Status == ComposerFactInferred {
			inferred++
		}
	}
	if inferred == 0 || len(brief.Facts) <= before {
		t.Fatalf("expected inferred intelligence facts, got %d facts (before %d)", len(brief.Facts), before)
	}

	// Re-applying the same extraction must be idempotent (no duplicate facts/values).
	factsAfterFirst := len(brief.Facts)
	applyBriefExtraction(&brief, &intelligence.BriefExtraction{InScope: []string{"backend", "frontend"}})
	if len(brief.Scope.InScope) != 2 || len(brief.Facts) != factsAfterFirst {
		t.Fatalf("re-applying extraction was not idempotent: scope=%#v facts=%d", brief.Scope.InScope, len(brief.Facts))
	}
}

func TestMaterializePromptArtifactDecomposesWhenFlowBeneficial(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Decompose", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSessionWithPrompt(context.Background(), project.ID, "Ship full feature", "Implement backend and frontend with review")
	if err != nil {
		t.Fatal(err)
	}
	// Persist a flow-beneficial brief (multiple in-scope workstreams + review)
	// before finalizing: the artifact freezes the brief, so updating it after
	// finalization would only affect future sessions, not this materialization.
	session, err := st.GetComposerSession(view.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	brief := LivingBrief{
		Goal:    "Implement backend and frontend with review coverage",
		Intent:  BriefIntent{Objective: "Implement backend and frontend with review coverage"},
		Scope:   BriefScope{InScope: []string{"backend API", "frontend UI"}},
		Quality: BriefQuality{Review: []string{"independent review"}, AcceptanceCriteria: []string{"tests pass"}},
	}
	if assessment := AssessComposerFlowSuitability(brief); assessment.Result == FlowDirectFit {
		t.Fatalf("test fixture must be flow-beneficial, got %+v", assessment)
	}
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSession(*session); err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	composerSemanticFixture(n, false)
	plan, err := n.MaterializePromptArtifactAsFlow(context.Background(), artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Description != "Materialized from Composer PromptArtifact via Flow decomposition" {
		t.Fatalf("expected flow decomposition path, got description %q", plan.Description)
	}
	if plan.StructuredFacts["nexus.source_artifact_id"] != artifact.ID || plan.StructuredFacts["nexus.source_artifact_revision"] != "1" {
		t.Fatalf("decomposed plan lost Composer lineage: %+v", plan.StructuredFacts)
	}
}

func TestComposerArtifactExecutionContextExcludesDroppedAndReferenceOnlySources(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Bounded Flow", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	previousPlan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Previous mission"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := st.UpsertMissionRun(store.MissionRunRecord{
		ID: "run-reference-only", PlanID: previousPlan.ID, ProjectID: project.ID,
		State: "COMPLETED_VERIFIED", PayloadJSON: `{"secret":"must-not-flow"}`, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	view, err := n.CreateComposerSessionWithPrompt(context.Background(), project.ID, "Review authentication safely", "Review authentication safely with tests")
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.GetComposerSession(view.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	brief := decodeComposerBrief(session.BriefJSON)
	for i := 0; i < 20; i++ {
		brief.Context.ReferencedFiles = append(brief.Context.ReferencedFiles, ContextFileRef{
			Path:    fmt.Sprintf("context/file-%02d.txt", i),
			Excerpt: strings.Repeat(fmt.Sprintf("evidence-%02d ", i), 500),
			Hash:    fmt.Sprintf("hash-%02d", i),
		})
	}
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSession(*session); err != nil {
		t.Fatal(err)
	}

	artifact, err := n.FinalizeComposerSession(context.Background(), session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	var envelope composerArtifactContextEnvelope
	if err := json.Unmarshal([]byte(artifact.ContextJSON), &envelope); err != nil {
		t.Fatal(err)
	}
	var droppedPath string
	var keptPath string
	for _, item := range envelope.ContextManifest.Items {
		switch {
		case item.Kind == "REFERENCED_FILE" && item.Decision == ComposerContextDrop && droppedPath == "":
			droppedPath = item.Locator
		case item.Kind == "REFERENCED_FILE" && item.Decision == ComposerContextKeep && keptPath == "":
			keptPath = item.Locator
		}
	}
	if droppedPath == "" || keptPath == "" {
		t.Fatalf("fixture must produce kept and dropped files: %+v", envelope.ContextManifest.Budget)
	}
	if strings.Contains(artifact.Content, droppedPath) {
		t.Fatalf("dropped file %q leaked into executable PromptArtifact", droppedPath)
	}
	if !strings.Contains(artifact.Content, keptPath) {
		t.Fatalf("kept file %q missing from executable PromptArtifact", keptPath)
	}

	// Mutating the live session after artifact creation must not alter an older
	// artifact's materialized Flow.
	latest, err := st.GetComposerSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	mutated := decodeComposerBrief(latest.BriefJSON)
	mutated.Goal = "MUTATED AFTER ARTIFACT"
	latest.BriefJSON = mustJSON(mutated)
	if err := st.UpdateComposerSession(*latest); err != nil {
		t.Fatal(err)
	}

	plan, err := n.MaterializePromptArtifactAsFlow(context.Background(), artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	executionContext := plan.StructuredFacts["composer.context"]
	if strings.Contains(executionContext, droppedPath) {
		t.Fatalf("dropped file %q leaked into Flow execution context", droppedPath)
	}
	if strings.Contains(executionContext, "run-reference-only") || strings.Contains(executionContext, "must-not-flow") {
		t.Fatal("reference-only Mission data leaked into Flow execution context")
	}
	if strings.Contains(executionContext, "MUTATED AFTER ARTIFACT") {
		t.Fatal("Flow used mutable session state instead of immutable PromptArtifact context")
	}
	if plan.StructuredFacts["nexus.source_context_manifest_fingerprint"] != envelope.ContextManifest.Fingerprint {
		t.Fatalf("Flow lost manifest lineage: %+v", plan.StructuredFacts)
	}
}

func TestComposerAndFlowRemainOptionalAcrossIntentPipeline(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	if decision.Strategy != IntentDirect {
		t.Fatalf("C1 must be DIRECT: %+v", decision)
	}
	if ComposerIsRequired(decision) || FlowIsRequired(decision) {
		t.Fatal("DIRECT must not require Composer or Flow")
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := contextsnapshot.Discover(root, contextsnapshot.Metadata{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	improve := DecideIntent("quero melhorar os testes", &snapshot)
	if ComposerIsRequired(improve) {
		t.Fatalf("C2 must not force Composer: %+v", improve)
	}

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "optional-composer", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Fix test", "atomic", []store.PlanPhase{{
		ID: "phase", Title: "Fix", Order: 1, Packages: []store.WorkPackage{{
			ID: "pkg", Title: "Fix", Goal: "corrija este teste quebrado", Status: "READY",
			AcceptanceCriteria: []string{"tests pass"}, VerificationRequirements: []string{"true"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow := FlowFromWorkPlan(*plan)
	roundTrip := WorkPlanFromFlow(flow)
	if roundTrip.ID != plan.ID || roundTrip.Title != plan.Title {
		t.Fatalf("Flow projection mutated WorkPlan identity")
	}
	semantic := WorkPlanFromFlow(flow)
	flow.UpdatedAt = flow.UpdatedAt.Add(time.Second)
	if WorkPlanFromFlow(flow).Phases[0].Packages[0].Goal != semantic.Phases[0].Packages[0].Goal {
		t.Fatal("non-semantic Flow metadata must not change WorkPlan goal")
	}

	spec, err := planToRunnerSpec(plan, "snap")
	if err != nil {
		t.Fatal(err)
	}
	if spec.ID != plan.ID {
		t.Fatal("WorkPlan must remain the execution spec source")
	}
	_ = runner.DefaultAutonomyContract()
}

func TestFlowSemanticEditProducesWorkPlanRevisionPayload(t *testing.T) {
	plan := canonicalFlowPlanFixture()
	flow := FlowFromWorkPlan(plan)
	flow.Steps[0].Goal = "Plan work with an explicit acceptance gate"
	revised := WorkPlanFromFlow(flow)
	if revised.Phases[0].Packages[0].Goal == plan.Phases[0].Packages[0].Goal {
		t.Fatal("semantic Flow edit must change WorkPlan goal")
	}
	if revised.ID != plan.ID {
		t.Fatal("revision must keep WorkPlan identity")
	}
}
