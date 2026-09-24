package nexus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestPreviewPlanExecutionUsesExistingMatcherAndResourceSchedulerWithoutMutation(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer preview", CanonicalPath: t.TempDir(), DefaultIsolation: "worktree"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Frontend specialist", Role: "frontend-engineer"})
	if err != nil {
		t.Fatal(err)
	}

	requirements := TaskRequirements{
		TaskKind:             "coding",
		Role:                 "frontend-engineer",
		PreferredRoles:       []string{"frontend-engineer"},
		AcceptableRoles:      []string{"frontend-engineer", "fullstack-engineer"},
		RequiredCapabilities: []string{"headless", "submit_prompt"},
		Confidence:           "CLASSIFIED",
		Source:               "composer",
	}
	rawRequirements, err := json.Marshal(requirements)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Improve login", "", []store.PlanPhase{{
		ID: "phase-build", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "frontend", Title: "Frontend", Goal: "Improve the React login flow", Status: "READY",
			Role: "implementer", TaskRequirements: string(rawRequirements), AssignmentStrategy: "AUTO",
			ResourcePolicy: "BALANCED", AcceptanceCriteria: []string{"login flow works"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}

	accounts := []ProviderAccount{
		{
			ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex main",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.80,
			Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
		},
		{
			ID: "claude-low", Provider: "claude", Profile: "default", DisplayName: "Claude low",
			Authenticated: true, Available: true, Health: "healthy",
			QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.25,
			Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
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

	if !preview.Ready || len(preview.Steps) != 1 {
		t.Fatalf("preview=%+v", preview)
	}
	step := preview.Steps[0]
	if step.Agent.AgentID != agent.ID || step.Agent.WillCreate {
		t.Fatalf("agent preview=%+v", step.Agent)
	}
	if step.Resource.Provider != "codex" || step.Resource.Profile != "main" {
		t.Fatalf("resource preview=%+v", step.Resource)
	}
	if step.Resource.Freshness != "LIVE" {
		t.Fatalf("freshness=%q, want LIVE", step.Resource.Freshness)
	}
	if len(beforeAgents) != len(afterAgents) {
		t.Fatalf("execution preview mutated Agent count: before=%d after=%d", len(beforeAgents), len(afterAgents))
	}
}

func TestPreviewAgentIsolationMatchesAutonomousRuntimePolicy(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Isolation preview", CanonicalPath: t.TempDir(), DefaultIsolation: "worktree"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Unsafe existing", Role: "implementer"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, (AgentConfig{Isolation: "project"}).ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}

	pinned := store.WorkPackage{
		ID: "pinned", Title: "Pinned", Goal: "Pinned", Role: "implementer",
		AssignmentStrategy: string(FlowAssignmentExisting), AgentAllocation: agent.ID,
	}
	_, _, decision, blocker := previewAgentForPackage(st, project, pinned, TaskRequirements{}, []store.Agent{agent}, true)
	if blocker == "" || decision.AgentID != agent.ID {
		t.Fatalf("pinned unsafe Agent must be an actionable blocker: decision=%+v blocker=%q", decision, blocker)
	}

	auto := pinned
	auto.ID = "auto"
	auto.AssignmentStrategy = string(FlowAssignmentAuto)
	auto.AgentAllocation = ""
	selected, _, decision, blocker := previewAgentForPackage(st, project, auto, TaskRequirements{}, []store.Agent{agent}, true)
	if blocker != "" || selected.ID != "" || !decision.WillCreate {
		t.Fatalf("AUTO should skip unsafe Agent and plan a worktree specialist: selected=%+v decision=%+v blocker=%q", selected, decision, blocker)
	}
}

func TestPreviewPlanExecutionExplainsSpecialistCreationWithoutStartingAgent(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer specialist preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	requirements := TaskRequirements{
		TaskKind:             "review",
		Role:                 "reviewer",
		PreferredRoles:       []string{"reviewer"},
		AcceptableRoles:      []string{"reviewer"},
		RequiredCapabilities: []string{"headless", "submit_prompt"},
	}
	rawRequirements, _ := json.Marshal(requirements)
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Review", "", []store.PlanPhase{{
		ID: "phase-review", Title: "Review", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "review", Title: "Independent review", Goal: "Review the change", Status: "READY",
			Role: "reviewer", TaskRequirements: string(rawRequirements), AssignmentStrategy: "AUTO",
			ResourcePolicy: "BALANCED", AcceptanceCriteria: []string{"review completed"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	accounts := []ProviderAccount{{
		ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex main",
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.70,
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}}

	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready || len(preview.Steps) != 1 {
		t.Fatalf("preview=%+v", preview)
	}
	if !preview.Steps[0].Agent.WillCreate {
		t.Fatalf("expected preview to explain specialist creation, got %+v", preview.Steps[0].Agent)
	}
	agents, err := st.ListAgents(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 0 {
		t.Fatalf("preview must not create specialist Agent, got %d", len(agents))
	}
}

func TestPreviewPlanExecutionBlocksWhenNoEligibleResourceExists(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer blocked preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Blocked", "", []store.PlanPhase{{
		ID: "phase", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "step", Title: "Build", Goal: "Build safely", Status: "READY", Role: "implementer",
			AssignmentStrategy: "AUTO", ResourcePolicy: "BALANCED",
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	accounts := []ProviderAccount{{
		ID: "unknown", Provider: "codex", Profile: "main", DisplayName: "Codex unknown",
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView:    &quota.QuotaView{Status: "UNKNOWN"},
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}}

	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Ready || len(preview.Blockers) == 0 {
		t.Fatalf("UNKNOWN-only resource pool must block automatic preview: %+v", preview)
	}
}

func TestPreviewPlanExecutionRequiresCostConfirmationWithoutClaimingHeadroom(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Cost preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Budgeted plan", "", []store.PlanPhase{{
		ID: "phase", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "step", Title: "Build", Goal: "Build safely", Status: "READY", Role: "implementer",
			AssignmentStrategy: "AUTO", ResourcePolicy: "BALANCED",
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prefs := store.DefaultCostPreferences()
	prefs.HardLimitMicroUSD = 1_000_000
	if err := st.PutCostPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	accounts := []ProviderAccount{{
		ID: "codex-main", Provider: "codex", Profile: "main", DisplayName: "Codex main",
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.80,
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}}

	preview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready {
		t.Fatalf("cost confirmation is an admission decision, not a resource blocker: %+v", preview)
	}
	if !preview.CostConfirmationRequired || preview.CostConfirmationToken == "" {
		t.Fatalf("preview must require an evidence-bound confirmation token: %+v", preview)
	}
	if preview.CostBudget.Level != CostBudgetUnknown {
		t.Fatalf("budget level=%s, want UNKNOWN because cost headroom is unmeasured", preview.CostBudget.Level)
	}
	if len(preview.Warnings) == 0 {
		t.Fatal("cost uncertainty must be visible in preview warnings")
	}
}

func TestValidateApprovedExecutionPreviewFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		preview *PlanExecutionPreview
		wantErr bool
	}{
		{name: "missing", preview: nil, wantErr: true},
		{name: "wrong plan", preview: &PlanExecutionPreview{PlanID: "other", Revision: 3, Ready: true}, wantErr: true},
		{name: "stale revision", preview: &PlanExecutionPreview{PlanID: "plan", Revision: 2, Ready: true}, wantErr: true},
		{name: "no eligible route", preview: &PlanExecutionPreview{PlanID: "plan", Revision: 3, Ready: false, Blockers: []string{"no provider account"}}, wantErr: true},
		{name: "fresh and ready", preview: &PlanExecutionPreview{PlanID: "plan", Revision: 3, Ready: true}, wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateApprovedExecutionPreview(tc.preview, "plan", 3)
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestExecutionPreviewAdmissionUsesApprovedAutonomyContract(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Contract preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Explicit contract", "", []store.PlanPhase{{
		ID: "phase", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "step", Title: "Build", Goal: "Build with explicit permissions", Status: "READY",
			Role: "implementer", AssignmentStrategy: "AUTO", ResourcePolicy: "BALANCED",
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	accounts := []ProviderAccount{{
		ID: "gemini-main", Provider: "gemini", Profile: "default", DisplayName: "Gemini",
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.8,
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}}

	defaultPreview, err := n.previewPlanExecution(context.Background(), st, plan, accounts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if defaultPreview.Ready {
		t.Fatal("default autonomous preview must reject unverified provider isolation")
	}

	unrestricted := runner.DefaultAutonomyContract()
	unrestricted.DisallowDestructiveGit = false
	unrestricted.AllowGitPush = true
	unrestricted.AllowDeploy = true
	unrestricted.AllowExternalNetwork = true
	unrestricted.AllowSecretAccess = true
	unrestricted.AllowPaidServices = true

	approvedPreview, err := n.previewPlanExecutionWithContract(
		context.Background(), st, plan, accounts, nil, unrestricted, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !approvedPreview.Ready {
		t.Fatalf("explicit unrestricted contract should be previewed as approved: %+v", approvedPreview.Blockers)
	}

	interactivePreview, err := n.previewPlanExecutionWithContract(
		context.Background(), st, plan, accounts, nil, runner.DefaultAutonomyContract(), false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !interactivePreview.Ready {
		t.Fatalf("non-autonomous run should not require autonomous provider isolation: %+v", interactivePreview.Blockers)
	}
}

func TestExecutionPreviewApprovalTokenBindsSemanticRoute(t *testing.T) {
	base := &PlanExecutionPreview{
		PlanID: "plan-1", ProjectID: "project-1", Revision: 7, Ready: true,
		Steps: []StepExecutionPreview{{
			PackageID: "pkg-1", PhaseID: "phase-1", Assignment: "AUTO", ResourcePolicy: "BALANCED",
			Agent:    PreviewAgentDecision{AgentID: "agent-1", Role: "developer", Reason: "matched"},
			Resource: PreviewResourceDecision{AccountID: "account-1", Provider: "codex", Profile: "work", Reason: "eligible"},
			Model:    PreviewModelDecision{Model: "gpt", Reason: "selected", Evidence: "LIVE"},
			Skills:   []string{"testing"}, Ready: true,
		}},
		CostBudget:  CostBudgetAssessment{Level: CostBudgetOK, Evidence: CostEvidenceMeasured},
		CostAnomaly: CostAnomalyAssessment{Status: CostAnomalyNormal},
	}
	first := executionPreviewApprovalToken(base)
	if first == "" {
		t.Fatal("expected execution preview approval token")
	}
	copyPreview := *base
	copyPreview.GeneratedAt = time.Now().UTC().Add(time.Hour)
	if got := executionPreviewApprovalToken(&copyPreview); got != first {
		t.Fatalf("generated_at must not destabilize approval token: got %s want %s", got, first)
	}
	copyPreview.Steps = append([]StepExecutionPreview(nil), base.Steps...)
	copyPreview.Steps[0] = base.Steps[0]
	copyPreview.Steps[0].Resource = base.Steps[0].Resource
	copyPreview.Steps[0].Resource.Profile = "fallback"
	if got := executionPreviewApprovalToken(&copyPreview); got == first {
		t.Fatal("routing change must invalidate execution preview approval token")
	}
}
