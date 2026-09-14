package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerOperationsRejectCanceledContext(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer Context", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := n.CreateComposerSession(context.Background(), project.ID, "Context contract")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assertCanceled := func(name string, operation func() error) {
		t.Helper()
		if err := operation(); !errors.Is(err, context.Canceled) {
			t.Fatalf("%s error = %v, want context canceled", name, err)
		}
	}
	assertCanceled("create", func() error {
		_, err := n.CreateComposerSessionWithPrompt(ctx, project.ID, "cancel", "")
		return err
	})
	assertCanceled("list", func() error {
		_, err := n.ListComposerSessions(ctx, project.ID)
		return err
	})
	assertCanceled("get", func() error {
		_, err := n.GetComposerSession(ctx, session.Session.ID)
		return err
	})
	assertCanceled("turn", func() error {
		_, err := n.AddComposerTurnExpected(ctx, session.Session.ID, store.ComposerUser, "cancel", 0)
		return err
	})
	assertCanceled("skill", func() error {
		_, err := n.UpdateComposerSkillState(ctx, session.Session.ID, "missing", store.ComposerSkillAccepted)
		return err
	})
	assertCanceled("finalize", func() error {
		_, err := n.FinalizeComposerSession(ctx, session.Session.ID, nil, true)
		return err
	})
	assertCanceled("resolve", func() error {
		_, err := n.ResolveComposerUnknownExpected(ctx, session.Session.ID, "missing", "answer", "ANSWERED", 0)
		return err
	})
}

func TestComposerFinalizationCreatesPromptWithoutWorkPlan(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := n.CreateComposerSession(context.Background(), project.ID, "Help me design test coverage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.AddComposerTurn(context.Background(), session.Session.ID, "USER", "The project is a React dashboard and needs browser regression coverage."); err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), session.Session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Content == "" || artifact.Version != 1 {
		t.Fatalf("invalid artifact: %+v", artifact)
	}
	plans, err := st.ListWorkPlans(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 0 {
		t.Fatalf("finalizing Composer must not create a Flow: %+v", plans)
	}
}

func TestComposerCanApplyBuiltinSkillWithoutMaestro(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer builtin", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSession(context.Background(), project.ID, "Improve test coverage")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, []string{"testing"}, true)
	if err != nil {
		t.Fatalf("builtin skill must be selectable without Maestro: %v", err)
	}
	var skillIDs []string
	if err := json.Unmarshal([]byte(artifact.SkillIDsJSON), &skillIDs); err != nil {
		t.Fatal(err)
	}
	if len(skillIDs) != 1 || skillIDs[0] != "testing" {
		t.Fatalf("unexpected skill IDs: %v", skillIDs)
	}
}

func TestMaterializePromptArtifactPreservesLineage(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Lineage", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSessionWithPrompt(context.Background(), project.ID, "Ship a small feature", "Implement the feature and verify it")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.MaterializePromptArtifactAsFlow(context.Background(), artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StructuredFacts["nexus.source_artifact_id"] != artifact.ID || plan.StructuredFacts["nexus.source_artifact_revision"] != "1" {
		t.Fatalf("missing artifact lineage: %+v", plan.StructuredFacts)
	}
	if got := plan.Phases[0].Packages[0].CompiledPrompt; got != artifact.Content {
		t.Fatalf("materialized flow lost canonical prompt: %q", got)
	}
}

func TestComposerFlowSuitabilityExplainsItsRecommendation(t *testing.T) {
	brief := newComposerBrief("Implement a full-stack feature", "Implement backend and frontend with review")
	brief.Scope.InScope = []string{"backend", "frontend"}
	brief.Quality.Review = []string{"independent review"}
	assessment := AssessComposerFlowSuitability(brief)
	if assessment.Result == FlowDirectFit || len(assessment.Reasons) == 0 || len(assessment.Signals) == 0 {
		t.Fatalf("unexplained suitability: %+v", assessment)
	}
}

func TestReviewComposerPromptReportsGapsAndContradictions(t *testing.T) {
	brief := newComposerBrief("Improve the API", "The endpoint must be fast but must not return quickly")
	review := ReviewComposerPrompt(brief)
	if len(review.Missing) == 0 || len(review.Contradictions) != 1 || len(review.Diff) == 0 {
		t.Fatalf("unexpected review: %+v", review)
	}
}

func TestComposerRevisionConflictIsRejected(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "revision", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(store.ComposerSession{ProjectID: project.ID, BriefJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateComposerSessionExpected(*session, session.Revision); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateComposerSessionExpected(*session, session.Revision); err == nil {
		t.Fatal("expected stale revision conflict")
	}
}

func TestComposerImportedPromptTracksUnknownsAndAppliesSkills(t *testing.T) {
	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus {
		return MaestroStatus{
			Available: true,
			Mode:      MaestroAssist,
			Capabilities: &MaestroCapability{
				Version: "1.0.0",
				Skills: []MaestroSkillDesc{
					{ID: "skill-webapp-testing", Name: "Webapp testing"},
				},
			},
		}
	}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := t.TempDir()
	if err := os.WriteFile(projectRoot+"/AGENTS.md", []byte("# Project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer", CanonicalPath: projectRoot})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSessionWithPrompt(context.Background(), project.ID, "Improve login prompt", `Implement OAuth login with Google for the existing dashboard.
Need backend and frontend changes.`)
	if err != nil {
		t.Fatal(err)
	}
	if view.Brief.Intent.Archetype != PromptArchetypeSoftwareFeature {
		t.Fatalf("unexpected archetype: %+v", view.Brief.Intent)
	}
	if view.Brief.Readiness.State != PromptReadinessNeedsInformation && view.Brief.Readiness.State != PromptReadinessBlocked {
		t.Fatalf("expected non-ready imported prompt, got %+v", view.Brief.Readiness)
	}
	if len(view.Brief.Unknowns) == 0 {
		t.Fatalf("expected imported prompt to produce unknowns: %+v", view.Brief)
	}
	if _, err := st.UpsertComposerSkillProposal(store.ComposerSkillProposal{
		SessionID: view.Session.ID,
		SkillID:   "skill-webapp-testing",
		State:     store.ComposerSkillSuggested,
		Reason:    "Regression coverage",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := n.UpdateComposerSkillState(context.Background(), view.Session.ID, "skill-webapp-testing", store.ComposerSkillAccepted)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Skills[0].State != store.ComposerSkillAccepted {
		t.Fatalf("skill state not persisted: %+v", updated.Skills)
	}
	updated, err = n.AddComposerTurn(context.Background(), view.Session.ID, store.ComposerUser, "Use Go in the backend and React in the frontend. Success means users can sign in with Google, we add regression coverage, and verification runs through go test plus npm test.")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Brief.Readiness.State == PromptReadinessBlocked {
		t.Fatalf("expected progress after answer, got %+v", updated.Brief.Readiness)
	}
	openUnknowns := 0
	for _, unknown := range updated.Brief.Unknowns {
		if unknown.Status == PromptUnknownOpen {
			openUnknowns++
		}
	}
	if openUnknowns == len(updated.Brief.Unknowns) {
		t.Fatalf("expected at least one unknown to close: %+v", updated.Brief.Unknowns)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, []string{"skill-webapp-testing"}, true)
	if err != nil {
		t.Fatal(err)
	}
	var validatedSkills []string
	if err := json.Unmarshal([]byte(artifact.SkillIDsJSON), &validatedSkills); err != nil {
		t.Fatal(err)
	}
	if len(validatedSkills) != 1 || validatedSkills[0] != "skill-webapp-testing" {
		t.Fatalf("unexpected applied skills: %+v", validatedSkills)
	}
	finalView, err := n.GetComposerSession(context.Background(), view.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(finalView.Artifacts) != 1 || finalView.Artifacts[0].Version != 1 {
		t.Fatalf("expected persisted artifact history, got %+v", finalView.Artifacts)
	}
	if finalView.Skills[0].State != store.ComposerSkillApplied {
		t.Fatalf("expected applied skill state, got %+v", finalView.Skills)
	}
}

func TestComposerRefinementCreatesNextVersion(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Refine", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSession(context.Background(), project.ID, "Initial draft for authentication system")
	if err != nil {
		t.Fatal(err)
	}
	v1, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if v1.Version != 1 {
		t.Fatalf("expected version 1, got %d", v1.Version)
	}

	// Refine to create version 2
	v2, err := n.RefineComposerArtifact(context.Background(), view.Session.ID, "Add OAuth2 with Google provider")
	if err != nil {
		t.Fatal(err)
	}
	if v2.Version != 2 {
		t.Fatalf("expected version 2, got %d", v2.Version)
	}
	if v2.ID == v1.ID {
		t.Fatalf("expected distinct artifact ID, got same %s", v2.ID)
	}

	// Verify both versions exist in history
	sessionView, err := n.GetComposerSession(context.Background(), view.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessionView.Artifacts) != 2 {
		t.Fatalf("expected 2 artifacts in history, got %d", len(sessionView.Artifacts))
	}
}

func TestResolveComposerUnknown(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Unknowns", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSession(context.Background(), project.ID, "Fix memory leak in websocket server")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Brief.Unknowns) == 0 {
		t.Fatal("expected at least one unknown in initial brief")
	}
	targetID := view.Brief.Unknowns[0].ID
	updated, err := n.ResolveComposerUnknown(context.Background(), view.Session.ID, targetID, "The leak happens in disconnect handler", "ANSWERED")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range updated.Brief.Unknowns {
		if u.ID == targetID {
			found = true
			if u.Status != PromptUnknownAnswered {
				t.Fatalf("expected status %s, got %s", PromptUnknownAnswered, u.Status)
			}
			if u.Answer != "The leak happens in disconnect handler" {
				t.Fatalf("unexpected answer: %s", u.Answer)
			}
		}
	}
	if !found {
		t.Fatalf("unknown %s not found after resolution", targetID)
	}
}
