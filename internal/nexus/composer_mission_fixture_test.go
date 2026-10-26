package nexus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// TestComposerApprovedFlowReachesMissionRunner uses a temporary authenticated
// provider profile and a deterministic PackageExecutor. It exercises the same
// Composer, WorkPlan, Execution Preview, approval, and MissionRunner boundaries
// as production without reading host credentials or dispatching provider work.
func TestComposerApprovedFlowReachesMissionRunner(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the isolated Codex CLI fixture uses a POSIX executable")
	}
	seedCodexExecutionFixture(t)

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Composer mission fixture", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	const sourcePrompt = `Implement OAuth login with Google for the existing React dashboard and Go API.
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
	view, err := n.CreateComposerSessionWithPrompt(context.Background(), project.ID, "Implement Google OAuth login safely", sourcePrompt)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := n.FinalizeComposerSession(context.Background(), view.Session.ID, nil, false)
	if err != nil {
		t.Fatalf("finalize complete Composer brief: %v", err)
	}
	plan, err := n.MaterializePromptArtifactAsFlow(context.Background(), artifact.ID)
	if err != nil {
		t.Fatalf("materialize Composer artifact as Flow: %v", err)
	}
	if plan.StructuredFacts["nexus.source_artifact_id"] != artifact.ID {
		t.Fatalf("Composer artifact lineage missing from WorkPlan: %+v", plan.StructuredFacts)
	}

	// The fixture owns its verification command, so no command from the imported
	// prompt or host repository can be executed during this deterministic test.
	for phaseIndex := range plan.Phases {
		for packageIndex := range plan.Phases[phaseIndex].Packages {
			plan.Phases[phaseIndex].Packages[packageIndex].VerificationRequirements = []string{"exit 0"}
		}
	}
	plan, _, err = n.UpdateWorkPlan(context.Background(), *plan, "Use isolated deterministic verification fixture")
	if err != nil {
		t.Fatalf("prepare deterministic verification requirements: %v", err)
	}

	preview, err := n.PreviewPlanExecution(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("execution preview: %v", err)
	}
	if !preview.Ready || preview.Revision != plan.CurrentRevision || strings.TrimSpace(preview.ApprovalToken) == "" {
		t.Fatalf("expected a ready preview bound to the current revision: %+v", preview)
	}
	for _, step := range preview.Steps {
		if !step.Ready || step.Resource.Provider != "codex" || step.Resource.Profile != "fixture" {
			t.Fatalf("preview did not resolve the isolated Codex fixture: %+v", step)
		}
	}

	executor := &composerMissionFixtureExecutor{}
	executor.failBeforeDispatch = 1
	n.runner = runner.NewMissionRunner(newStoreRunRepository(st), executor)
	contract := runner.DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.VerificationCommands = nil
	contract.GlobalVerificationCommands = nil
	approvedToken := preview.ApprovalToken
	packageID := plan.Phases[0].Packages[0].ID
	editedGoal := strings.TrimSpace(plan.Phases[0].Packages[0].Goal) + "\nKeep the existing account and do not change provider credentials."
	plan.Phases[0].Packages[0].Goal = editedGoal
	plan, _, err = n.UpdateWorkPlan(context.Background(), *plan, "Clarify the existing-account constraint")
	if err != nil {
		t.Fatalf("persist semantic WorkPlan edit: %v", err)
	}
	if plan.CurrentRevision != preview.Revision+1 {
		t.Fatalf("semantic edit revision=%d, expected %d", plan.CurrentRevision, preview.Revision+1)
	}

	_, staleApprovalErr := n.StartMissionRunApprovedWithOptions(
		context.Background(), plan.ID, plan.CurrentRevision, "", contract, false,
		MissionAdmissionOptions{ExecutionPreviewToken: approvedToken},
	)
	if staleApprovalErr == nil || !strings.Contains(staleApprovalErr.Error(), "execution preview changed after approval") {
		t.Fatalf("stale preview token after semantic edit returned %v", staleApprovalErr)
	}
	runsAfterStaleApproval, err := n.Runner().ListRuns(context.Background())
	if err != nil {
		t.Fatalf("list runs after stale approval rejection: %v", err)
	}
	if len(runsAfterStaleApproval) != 0 {
		t.Fatalf("stale approval created %d MissionRuns", len(runsAfterStaleApproval))
	}

	preview, err = n.PreviewPlanExecution(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("refresh preview after semantic edit: %v", err)
	}
	if !preview.Ready || preview.Revision != plan.CurrentRevision || strings.TrimSpace(preview.ApprovalToken) == "" {
		t.Fatalf("expected refreshed ready preview for semantic revision: %+v", preview)
	}
	if preview.ApprovalToken == approvedToken {
		t.Fatal("semantic edit must invalidate the previous execution preview token")
	}
	if plan.Phases[0].Packages[0].ID != packageID {
		t.Fatalf("semantic edit unexpectedly replaced package identity: got %q, want %q", plan.Phases[0].Packages[0].ID, packageID)
	}
	run, err := n.StartMissionRunApprovedWithOptions(
		context.Background(), plan.ID, plan.CurrentRevision, "", contract, false,
		MissionAdmissionOptions{ExecutionPreviewToken: preview.ApprovalToken},
	)
	if err != nil {
		t.Fatalf("start approved MissionRun from current preview: %v", err)
	}
	if run.PlanRevision != plan.CurrentRevision || run.ExecutionSnapshotID == "" {
		t.Fatalf("MissionRun did not freeze the approved revision: %+v", run)
	}
	if len(run.PackageRuns) < 4 || len(run.PackageRuns) != len(preview.Steps) {
		t.Fatalf("MissionRunner package count=%d, preview step count=%d", len(run.PackageRuns), len(preview.Steps))
	}
	snapshot, err := n.loadMissionSnapshot(run)
	if err != nil {
		t.Fatalf("load immutable MissionRun snapshot: %v", err)
	}
	if snapshot.Plan.CurrentRevision != plan.CurrentRevision {
		t.Fatalf("snapshot revision=%d, approved revision=%d", snapshot.Plan.CurrentRevision, plan.CurrentRevision)
	}
	if got := snapshot.Plan.Phases[0].Packages[0].Goal; got != editedGoal {
		t.Fatalf("MissionRun snapshot lost the approved semantic edit: got %q, want %q", got, editedGoal)
	}

	sawPreDispatchFailure := false
	for step := 0; step < 100 && run.State != runner.StateCompletedVerified; step++ {
		updated, _, stepErr := n.Runner().ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if !sawPreDispatchFailure && hasDispatchState(run, runner.DispatchFailedBeforeDispatch) {
			if stepErr != nil {
				t.Fatalf("proven pre-dispatch failure should remain safely resumable: %v", stepErr)
			}
			if run.State == runner.StateBlockedNeedsUser {
				t.Fatalf("proven pre-dispatch failure unexpectedly blocked the mission: %+v", run.NeedsHuman)
			}
			if failures := executor.preDispatchFailures(); failures != 1 {
				t.Fatalf("pre-dispatch failure count=%d, want 1", failures)
			}
			if err := st.Close(); err != nil {
				t.Fatalf("close mission store before restart: %v", err)
			}
			reopened, err := store.Open(filepath.Join(os.Getenv("AI_CLI_DATA_DIR"), "nexus.db"))
			if err != nil {
				t.Fatalf("reopen mission store: %v", err)
			}
			defer reopened.Close()
			n.st = reopened
			n.runner = runner.NewMissionRunner(newStoreRunRepository(reopened), executor)
			run, err = n.Runner().GetRun(context.Background(), run.ID)
			if err != nil {
				t.Fatalf("reload Composer MissionRun after restart: %v", err)
			}
			if !hasDispatchState(run, runner.DispatchFailedBeforeDispatch) {
				t.Fatal("restart lost the proven pre-dispatch failure state")
			}
			restartedSnapshot, err := n.loadMissionSnapshot(run)
			if err != nil {
				t.Fatalf("reload immutable execution snapshot after restart: %v", err)
			}
			if restartedSnapshot.Plan.CurrentRevision != plan.CurrentRevision || restartedSnapshot.Plan.Phases[0].Packages[0].Goal != editedGoal {
				t.Fatalf("restart changed the approved semantic snapshot: %+v", restartedSnapshot.Plan)
			}
			sawPreDispatchFailure = true
			continue
		}
		if stepErr != nil {
			t.Fatalf("deterministic MissionRunner step %d: %v (state=%s)", step, stepErr, run.State)
		}
	}
	if !sawPreDispatchFailure {
		t.Fatal("Composer MissionRun did not exercise a proven pre-dispatch failure and restart")
	}
	if run.State != runner.StateCompletedVerified {
		t.Fatalf("Composer-created MissionRun did not complete through MissionRunner: %s", run.State)
	}
	executed, reviewed := executor.counts()
	if executed == 0 || reviewed == 0 {
		t.Fatalf("expected deterministic work and review stages, executed=%d reviewed=%d", executed, reviewed)
	}
	if duplicates := executor.duplicateDispatches(); duplicates != 0 {
		t.Fatalf("restart repeated %d successful package dispatches", duplicates)
	}
	for _, packageRun := range run.PackageRuns {
		if packageRun.State != runner.StateVerified {
			t.Fatalf("package %s lacks verified terminal state: %s", packageRun.PackageID, packageRun.State)
		}
	}
}

func seedCodexExecutionFixture(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("AI_MANAGER_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("AI_CLI_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("NEXUS_CONFIG_DIR", filepath.Join(root, "config"))
	t.Setenv("AI_MANAGER_CONFIG_DIR", filepath.Join(root, "config"))
	t.Setenv("AI_CLI_CONFIG_DIR", filepath.Join(root, "config"))

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	codexPath := filepath.Join(binDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nif [ \"${1:-}\" = \"--version\" ]; then printf '%s\\n' 'codex fixture 1.0'; exit 0; fi\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := profile.Create("codex", "fixture"); err != nil {
		t.Fatalf("create temporary Codex profile: %v", err)
	}
	home, err := profile.Home("codex", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"email":                       "fixture@example.test",
		"https://api.openai.com/auth": map[string]string{"chatgpt_account_id": "acct-fixture"},
	})
	if err != nil {
		t.Fatal(err)
	}
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".fixture-signature"
	authData, err := json.Marshal(map[string]any{"tokens": map[string]string{
		"id_token": jwt, "access_token": "fixture-access", "refresh_token": "fixture-refresh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "auth.json"), authData, 0o600); err != nil {
		t.Fatal(err)
	}

	scope, err := profile.AccountScope("codex", "fixture", "acct-fixture")
	if err != nil {
		t.Fatal(err)
	}
	remaining := 90.0
	if err := quota.NewEngine(time.Minute).SaveUsageForScope(scope, model.UsageSnapshot{
		ProviderID: "codex", ProfileID: "fixture", Status: model.UsageLive,
		Source: model.SourceOfficialAPI, FetchedAt: time.Now().UTC(), Account: "acct-fixture",
		Windows: []model.UsageWindow{{Kind: "5h", RemainingPercent: &remaining}},
	}); err != nil {
		t.Fatalf("seed scoped fixture quota: %v", err)
	}
}

type composerMissionFixtureExecutor struct {
	mu                   sync.Mutex
	executed             int
	reviewed             int
	failBeforeDispatch   int
	failedBeforeDispatch int
	executedPackages     map[string]struct{}
	duplicatePackages    int
}

func (*composerMissionFixtureExecutor) Allocate(_ context.Context, run *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	return runner.AllocationResult{AgentID: "agent-fixture", Workspace: run.Workspace}, nil
}

func (*composerMissionFixtureExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "fixture-prompt-v1", Content: "deterministic Composer mission fixture"}, nil
}

func (e *composerMissionFixtureExecutor) Execute(_ context.Context, _ *runner.MissionRun, pkg *runner.PackageRun, _ string) (runner.ExecutionResult, error) {
	e.mu.Lock()
	if e.failBeforeDispatch > 0 {
		e.failBeforeDispatch--
		e.failedBeforeDispatch++
		e.mu.Unlock()
		return runner.ExecutionResult{}, runner.MarkDispatchNotSent(errors.New("fixture failed before provider dispatch"))
	}
	e.executed++
	if e.executedPackages == nil {
		e.executedPackages = make(map[string]struct{})
	}
	if _, exists := e.executedPackages[pkg.PackageID]; exists {
		e.duplicatePackages++
	} else {
		e.executedPackages[pkg.PackageID] = struct{}{}
	}
	e.mu.Unlock()
	return runner.ExecutionResult{RuntimeID: "runtime-fixture"}, nil
}

func (e *composerMissionFixtureExecutor) duplicateDispatches() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.duplicatePackages
}

func (e *composerMissionFixtureExecutor) preDispatchFailures() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.failedBeforeDispatch
}

func hasDispatchState(run *runner.MissionRun, state runner.DispatchState) bool {
	for _, packageRun := range run.PackageRuns {
		if packageRun.DispatchState == state {
			return true
		}
	}
	return false
}

func (e *composerMissionFixtureExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	e.mu.Lock()
	e.reviewed++
	e.mu.Unlock()
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-fixture"}, nil
}

func (e *composerMissionFixtureExecutor) counts() (executed, reviewed int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.executed, e.reviewed
}
