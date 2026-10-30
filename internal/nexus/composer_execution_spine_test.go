package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerCompletePromptFlowsToExecutionPreviewWithoutClarification(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer north star", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	sourcePrompt := `Implement OAuth login with Google for the existing React dashboard and Go API.
Scope: implement the backend OAuth callback and the frontend sign-in/session handling.
Security requirements: OAuth state validation, PKCE, HttpOnly SameSite cookies, no secrets in logs.
Out of scope: additional identity providers and deployment changes.
Success means a user can sign in with Google, refresh the dashboard, and keep the authenticated session.
Testing: add Go unit/integration tests, React tests, and browser regression coverage.
Verification: run go test ./..., frontend tests, and the browser authentication scenario.
Independent review must check authentication, authorization, secret handling, and regressions.
Definition of Done: implementation, tests, review findings resolved, and verification evidence recorded.
Deliverables: source changes, tests, review evidence, and verification evidence.
Do not push, merge, deploy, or weaken existing authentication controls.`

	view, err := n.CreateComposerSessionWithPrompt(
		context.Background(),
		project.ID,
		"Implement Google OAuth login safely",
		sourcePrompt,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, unknown := range view.Brief.Unknowns {
		if unknown.Severity == PromptUnknownBlocking && unknown.Status == PromptUnknownOpen {
			t.Fatalf("complete imported prompt must not require blocking clarification: %+v", unknown)
		}
	}

	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, false)
	if err != nil {
		t.Fatalf("complete imported prompt must finalize without confirm-gaps bypass: %v", err)
	}
	composerSemanticFixture(n, false)
	plan, err := n.MaterializePromptArtifactAsFlow(context.Background(), artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StructuredFacts["nexus.source_artifact_id"] != artifact.ID {
		t.Fatalf("artifact lineage missing: %+v", plan.StructuredFacts)
	}
	packageCount := 0
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			packageCount++
			if pkg.TaskRequirements == "" {
				t.Fatalf("package %s lost task requirements", pkg.ID)
			}
		}
	}
	if packageCount < 4 {
		t.Fatalf("complex complete prompt should materialize a delivery pipeline, got %d package(s)", packageCount)
	}

	accounts := []ProviderAccount{
		{
			ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex main",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.82,
			Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED", "testing": "SUPPORTED"},
		},
		{
			ID: "claude-review", Provider: "claude", Profile: "review", DisplayName: "Claude review",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.64,
			Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED", "testing": "SUPPORTED"},
		},
	}

	beforeAgents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	afterAgents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeAgents) != len(afterAgents) {
		t.Fatalf("execution preview must be side-effect free: before=%d after=%d", len(beforeAgents), len(afterAgents))
	}
	if len(preview.Steps) != packageCount {
		t.Fatalf("preview should cover every package: steps=%d packages=%d", len(preview.Steps), packageCount)
	}
	for _, step := range preview.Steps {
		if step.Agent.Reason == "" || step.Resource.Reason == "" {
			t.Fatalf("step %s lacks explainable routing: %+v", step.PackageID, step)
		}
	}
}
