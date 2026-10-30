package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
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
		_, err := n.AddComposerTurnExpected(ctx, session.Session.ID, store.ComposerUser, "cancel", 0, nil)
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
	assertCanceled("resolve wrapper", func() error {
		_, err := n.ResolveComposerUnknown(ctx, session.Session.ID, "missing", "answer", "ANSWERED")
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
	finalized, err := st.GetComposerSession(session.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.State != store.ComposerFinalized {
		t.Fatalf("finalized prompt artifact must persist FINALIZED session state, got %q", finalized.State)
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
	composerSemanticFixture(n, true)
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

func TestComposerSkillRecommendationsFollowBriefWithoutOverwritingUserDecision(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Skill Reconcile", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(store.ComposerSession{
		ProjectID: project.ID,
		Title:     "Skill reconcile",
		BriefJSON: mustJSON(newComposerBrief("security audit", "")),
	})
	if err != nil {
		t.Fatal(err)
	}
	catalog := nexusskills.Catalog{Library: []nexusskills.Skill{
		{ID: "security", Name: "Security", Description: "security audit", Source: nexusskills.SourceBuiltin, Availability: nexusskills.AvailabilityAvailable},
		{ID: "documentation", Name: "Documentation", Description: "documentation readme", Source: nexusskills.SourceBuiltin, Availability: nexusskills.AvailabilityAvailable},
	}}
	reconcileComposerSkillProposals(st, session.ID, newComposerBrief("security audit", ""), catalog)
	if err := st.SetComposerSkillState(session.ID, "security", store.ComposerSkillAccepted); err != nil {
		t.Fatal(err)
	}
	reconcileComposerSkillProposals(st, session.ID, newComposerBrief("documentation readme", ""), catalog)

	proposals, err := st.ListComposerSkillProposals(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]string{}
	for _, proposal := range proposals {
		states[proposal.SkillID] = proposal.State
	}
	if states["security"] != store.ComposerSkillAccepted {
		t.Fatalf("explicit user decision was overwritten: %+v", states)
	}
	if states["documentation"] != store.ComposerSkillSuggested {
		t.Fatalf("new relevant recommendation missing: %+v", states)
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
	if view.Brief.Intent.Archetype != PromptArchetypeSecurity {
		t.Fatalf("unexpected archetype: %+v", view.Brief.Intent)
	}
	// No ritual questions for every missing field: a reviewer may refine this
	// plan without having to complete an arbitrary archetype questionnaire.
	for _, unknown := range view.Brief.Unknowns {
		if unknown.Status == PromptUnknownOpen && (unknown.Source == "ARCHETYPE" || unknown.Source == "REFINERY") {
			t.Fatalf("unexpected static checklist question: %+v", unknown)
		}
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
	var acceptedSkill *store.ComposerSkillProposal
	for i := range updated.Skills {
		if updated.Skills[i].SkillID == "skill-webapp-testing" {
			acceptedSkill = &updated.Skills[i]
			break
		}
	}
	if acceptedSkill == nil || acceptedSkill.State != store.ComposerSkillAccepted {
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
	if len(updated.Brief.Unknowns) > 0 && openUnknowns == len(updated.Brief.Unknowns) {
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
	applied := false
	for _, skill := range finalView.Skills {
		if skill.SkillID == "skill-webapp-testing" && skill.State == store.ComposerSkillApplied {
			applied = true
			break
		}
	}
	if !applied {
		t.Fatalf("expected skill-webapp-testing to be applied, got %+v", finalView.Skills)
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

func TestComposerFinalizeIdempotentWhenRepeatedWithIdenticalContent(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "IdempotentFinalize", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	view, err := n.CreateComposerSession(context.Background(), project.ID, "Improve frontend architecture for notes app")
	if err != nil {
		t.Fatal(err)
	}

	// First finalization
	v1, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, true)
	if err != nil {
		t.Fatalf("first finalization failed: %v", err)
	}
	if v1.Version != 1 {
		t.Fatalf("expected version 1, got %d", v1.Version)
	}

	// Repeated finalization with identical content must succeed idempotently (no UNIQUE constraint failure)
	v2, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, true)
	if err != nil {
		t.Fatalf("repeated finalization failed with constraint error: %v", err)
	}
	if v2.ID != v1.ID {
		t.Fatalf("expected same artifact ID %s, got %s", v1.ID, v2.ID)
	}
	if v2.Version != 1 {
		t.Fatalf("expected version 1, got %d", v2.Version)
	}

	// Refinement with empty/noop goal must also succeed idempotently
	v3, err := n.RefineComposerArtifact(context.Background(), view.Session.ID, "")
	if err != nil {
		t.Fatalf("noop refinement failed with constraint error: %v", err)
	}
	if v3.ID != v1.ID {
		t.Fatalf("expected same artifact ID %s, got %s", v1.ID, v3.ID)
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
	// An actionable question must come from Intelligence, not from a fixed
	// set of seven questions rendered to every user.
	session, err := st.GetComposerSession(view.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	brief := decodeComposerBrief(session.BriefJSON)
	MergeIntelligenceUnknowns(&brief, []intelligence.AmbiguityItem{{
		Key: "context.existing_state", Level: intelligence.AmbiguityBlocking,
		Question: "What behavior is leaking memory?", Rationale: "Identify the failure before modifying code.",
	}})
	refreshComposerBrief(&brief)
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSession(*session); err != nil {
		t.Fatal(err)
	}
	targetID := "context.existing_state"
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

// hangIntelligenceProvider blocks every model call until the context is canceled.
// Mirrors a stuck headless CLI (agy/codex) that never returns stdout.
type hangIntelligenceProvider struct{}

func (hangIntelligenceProvider) Name() string                   { return "hang" }
func (hangIntelligenceProvider) Available(context.Context) bool { return true }
func (hangIntelligenceProvider) AnalyzeIntent(ctx context.Context, _ string, _ map[string]any) (*intelligence.IntentAnalysis, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (hangIntelligenceProvider) EvaluateAmbiguities(ctx context.Context, _ *intelligence.IntentAnalysis) ([]intelligence.AmbiguityItem, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (hangIntelligenceProvider) GeneratePlanOutline(ctx context.Context, _ *intelligence.IntentAnalysis, _ map[string]string, _ map[string]any) ([]intelligence.WorkPackageOutline, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (hangIntelligenceProvider) ExtractBrief(ctx context.Context, _ string, _ map[string]any) (*intelligence.BriefExtraction, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCreateComposerSessionReturnsWhenIntelligenceHangs(t *testing.T) {
	n := openTestNexus(t)
	n.intelligenceProviderForTest = hangIntelligenceProvider{}
	prev := composerBootstrapTimeout
	composerBootstrapTimeout = 120 * time.Millisecond
	t.Cleanup(func() { composerBootstrapTimeout = prev })

	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Hang Guard", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	view, err := n.CreateComposerSessionWithPrompt(
		context.Background(),
		project.ID,
		"Documentar API",
		"Escreva documentação técnica da API REST com exemplos.",
	)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("CreateComposerSessionWithPrompt: %v", err)
	}
	if view == nil || view.Session.ID == "" {
		t.Fatal("expected a persisted session even when intelligence hangs")
	}
	// The 120ms budget applies to provider intelligence, not to the full
	// deterministic bootstrap (Git/context/memory/store). Under -race those
	// local steps are materially slower, so retain a bounded anti-hang guard
	// without turning this test into an end-to-end latency SLA.
	if elapsed > 5*time.Second {
		t.Fatalf("create took %s; hung intelligence did not release within the bounded bootstrap guard", elapsed)
	}
}
