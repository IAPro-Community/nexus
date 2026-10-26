package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// PAUSED and BLOCKED_NEEDS_USER are non-terminal: reporting them as
// COMPLETED would let callers assume success for a mission that is still
// alive and waiting on a human.
func TestDispatchStatusNeverCompletesNonTerminalStates(t *testing.T) {
	cases := []struct {
		state runner.State
		want  DispatchStatus
	}{
		{runner.StateCompletedVerified, DispatchCompleted},
		{runner.StateFailed, DispatchFailed},
		{runner.StateFailedNoProgress, DispatchFailed},
		{runner.StateCanceledByUser, DispatchCanceled},
		{runner.StateBlockedNeedsUser, DispatchBlocked},
		{runner.StatePaused, DispatchPaused},
		// In-flight states have no honest completion mapping; they must
		// not surface as COMPLETED either.
		{runner.StateExecuting, DispatchFailed},
		{runner.StatePending, DispatchFailed},
	}
	for _, tc := range cases {
		if got := dispatchStatusForRunState(tc.state); got != tc.want {
			t.Errorf("dispatchStatusForRunState(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

// Goal IDs for persistent/distributed entities must be collision-resistant
// and sortable: ULID, not UnixNano.
func TestGeneratedGoalIDIsULID(t *testing.T) {
	id := "goal_" + ids.NewULID()
	if !strings.HasPrefix(id, "goal_") {
		t.Fatalf("goal id %q missing prefix", id)
	}
	if _, ok := ids.DecodeULID(strings.TrimPrefix(id, "goal_")); !ok {
		t.Fatalf("goal id %q suffix is not a valid ULID", id)
	}
	other := "goal_" + ids.NewULID()
	if id == other {
		t.Fatal("consecutive goal IDs collided")
	}
}

func TestStartDispatchDefaultsDelegationOff(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{
		Name:          "Default delegation off",
		CanonicalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := n.StartDispatch(context.Background(), DispatchRequest{
		ProjectID:    project.ID,
		Goal:         "implemente backend e frontend com testes",
		SourcePrompt: "implemente backend e frontend com testes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Status != DispatchRejected {
		t.Fatalf("implicit delegation must not create a Mission: %+v", result)
	}
	if !result.FallbackSafe || !strings.Contains(result.Error, "dispatch brake") {
		t.Fatalf("implicit delegation must fail closed before admission: %+v", result)
	}
}

func TestDispatchTopologySeparatesMissionPromotionFromDelegation(t *testing.T) {
	nonDelegated := DelegationDecision{Delegate: false}
	if dispatchTopologyAllowed(DispatchRequest{}, nonDelegated) {
		t.Fatal("ordinary Conductor dispatch must keep the existing delegation brake")
	}
	if !dispatchTopologyAllowed(DispatchRequest{AutoPromoteToMission: true}, nonDelegated) {
		t.Fatal("an upstream-approved conversation promotion must be allowed to create a single-topology Mission")
	}
	if !dispatchTopologyAllowed(DispatchRequest{}, DelegationDecision{Delegate: true}) {
		t.Fatal("delegated topology must remain dispatchable")
	}
}

func TestWaitDispatchDoesNotCreateOrWaitForRejectedAdmission(t *testing.T) {
	n := openTestNexus(t)
	rejected := &DispatchResult{
		GoalID: "goal_test",
		Status: DispatchRejected,
		Error:  "strict admission blocked",
	}
	got, err := n.WaitDispatch(context.Background(), rejected)
	if err != nil {
		t.Fatal(err)
	}
	if got != rejected {
		t.Fatalf("non-accepted dispatch must be returned unchanged: got=%+v want=%+v", got, rejected)
	}
}

func TestWaitDispatchRejectsNilAcceptedResult(t *testing.T) {
	n := openTestNexus(t)
	if _, err := n.WaitDispatch(context.Background(), nil); err == nil {
		t.Fatal("nil accepted result must be rejected")
	}
}

func TestWaitDispatchFailureIsNeverMarkedFallbackSafeAfterAcceptance(t *testing.T) {
	n := openTestNexus(t)
	got, err := n.WaitDispatch(context.Background(), &DispatchResult{
		GoalID: "goal_test",
		PlanID: "plan_test",
		Status: DispatchAccepted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.FallbackSafe {
		t.Fatalf("post-acceptance failure must never be eligible for Direct fallback: %+v", got)
	}
}

func TestStartDispatchReturnsPreparedPlanOnPreRunAdmissionFailure(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{
		Name:             "Blocked promotion",
		CanonicalPath:    t.TempDir(),
		DefaultIsolation: "project",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := n.StartDispatch(context.Background(), DispatchRequest{
		ProjectID:            project.ID,
		Goal:                 "implemente backend e frontend com testes",
		SourcePrompt:         "implemente backend e frontend com testes",
		DelegationMode:       DelegationAuto,
		AutoPromoteToMission: true,
		PromotionSource:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Status != DispatchRejected {
		t.Fatalf("expected rejected pre-run admission, got %+v", result)
	}
	if !result.AdmissionBlocked || strings.TrimSpace(result.AdmissionStage) == "" || !result.FallbackSafe {
		t.Fatalf("pre-run admission failure must be typed and safe for interactive fallback: %+v", result)
	}
	if strings.TrimSpace(result.PlanID) == "" {
		t.Fatalf("admission failure must expose prepared plan lineage: %+v", result)
	}
	if strings.TrimSpace(result.Error) == "" {
		t.Fatal("admission failure must include an actionable error")
	}
}

func TestStartDispatchExplicitMissionForAtomicFixUsesSingleWorkPlanAndMissionPackage(t *testing.T) {
	seedCodexExecutionFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	root := initAtomicFixExecutionRepo(t)
	goal := "Corrija este teste quebrado como Mission."
	intent := DecideIntent(goal, nil)
	if intent.Strategy != IntentDirect || ComposerIsRequired(intent) || FlowIsRequired(intent) {
		t.Fatalf("atomic fix must stay DIRECT without requiring Composer or Flow: %+v", intent)
	}
	if !strings.Contains(strings.ToLower(goal), "como mission") {
		t.Fatal("the fixture must explicitly request Mission admission for this DIRECT task")
	}

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{
		Name:             "Atomic direct fix",
		CanonicalPath:    root,
		DefaultIsolation: "worktree",
	})
	if err != nil {
		t.Fatal(err)
	}
	executor := &autonomousAtomicFixFixtureExecutor{
		composerMissionFixtureExecutor: &composerMissionFixtureExecutor{},
		nexus:                          n,
	}
	n.runner = runner.NewMissionRunner(newStoreRunRepository(st), executor)

	accepted, err := n.StartDispatch(ctx, DispatchRequest{
		ProjectID:            project.ID,
		Goal:                 goal,
		SourcePrompt:         goal,
		DelegationMode:       DelegationOff,
		AutoPromoteToMission: true,
		PromotionSource:      "p5-atomic-direct-fixture",
	})
	if err != nil {
		t.Fatalf("start explicitly admitted atomic Mission: %v", err)
	}
	if accepted == nil || accepted.Status != DispatchAccepted || accepted.PlanID == "" || accepted.RunID == "" {
		t.Fatalf("expected accepted run with WorkPlan lineage: %+v", accepted)
	}

	plan, err := n.GetWorkPlan(ctx, accepted.PlanID)
	if err != nil {
		t.Fatalf("load canonical WorkPlan: %v", err)
	}
	if plan.StructuredFacts[IntentStrategyFactKey] != string(IntentDirect) {
		t.Fatalf("WorkPlan lost DIRECT intent decision: %+v", plan.StructuredFacts)
	}
	selection, err := ReadMissionTeamSelectionFacts(plan.StructuredFacts)
	if err != nil {
		t.Fatalf("read durable mission team decision: %v", err)
	}
	if !selection.Direct || selection.Preset != "" {
		t.Fatalf("atomic fix must not create a multi-agent team: %+v", selection)
	}
	packageCount := 0
	for _, phase := range plan.Phases {
		packageCount += len(phase.Packages)
		for _, pkg := range phase.Packages {
			if len(pkg.Dependencies) != 0 {
				t.Fatalf("single atomic work package has unexpected dependencies: %+v", pkg)
			}
		}
	}
	if packageCount != 1 {
		t.Fatalf("DIRECT fix generated %d WorkPlan packages; want one", packageCount)
	}

	completed, err := n.WaitDispatch(ctx, accepted)
	if err != nil {
		t.Fatalf("wait for deterministic atomic Mission: %v", err)
	}
	if completed == nil || completed.Status != DispatchCompleted || completed.Run == nil || completed.Run.State != runner.StateCompletedVerified {
		if completed == nil || completed.Run == nil {
			t.Fatalf("atomic DIRECT Mission returned no final run: %+v", completed)
		}
		executed, reviewed := executor.counts()
		t.Fatalf("atomic DIRECT Mission state=%s phase=%s phases=%+v global_verifications=%+v needs_human=%+v paused=%q packages=%+v executed=%d reviewed=%d", completed.Run.State, completed.Run.CurrentPhaseID, completed.Run.PhaseRuns, completed.Run.GlobalVerifications, completed.Run.NeedsHuman, completed.Run.PausedReason, completed.Run.PackageRuns, executed, reviewed)
	}
	if completed.Run.PlanID != plan.ID || completed.Run.PlanRevision != plan.CurrentRevision || len(completed.Run.PackageRuns) != 1 {
		t.Fatalf("MissionRunner did not preserve canonical one-package WorkPlan lineage: %+v", completed.Run)
	}
	if executed, _ := executor.counts(); executed != 1 {
		t.Fatalf("executor dispatched %d packages, want exactly one", executed)
	}
	if duplicates := executor.duplicateDispatches(); duplicates != 0 {
		t.Fatalf("Mission restart/retry duplicated %d atomic package dispatches", duplicates)
	}
	stream, err := st.GetValidationEvidenceStreamByName(project.ID, missionEvidenceStreamName)
	if err != nil {
		t.Fatalf("load durable validation evidence stream: %v", err)
	}
	evidence, err := st.ListAllValidationEvidenceEntries(stream.ID)
	if err != nil {
		t.Fatalf("load durable validation evidence: %v", err)
	}
	if len(evidence) != 2 {
		t.Fatalf("evidence entries=%d, want package and global verification", len(evidence))
	}
}

type autonomousAtomicFixFixtureExecutor struct {
	*composerMissionFixtureExecutor
	nexus *Nexus
}

func (e *autonomousAtomicFixFixtureExecutor) RecordValidationEvidence(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, results []runner.VerificationResult) error {
	return e.nexus.recordMissionValidationEvidence(ctx, run, "mission/p5-atomic-direct/package/"+pkg.PackageID, strings.Join(pkg.VerificationRequirements, " && "), results, "codex", "fixture", "fixture-model", pkg.Attempt)
}

func (e *autonomousAtomicFixFixtureExecutor) RecordGlobalValidationEvidence(ctx context.Context, run *runner.MissionRun, results []runner.VerificationResult) error {
	return e.nexus.recordMissionValidationEvidence(ctx, run, "mission/p5-atomic-direct/global", strings.Join(run.Contract.GlobalVerificationCommands, " && "), results, "", "", "", run.TotalIterations)
}

func initAtomicFixExecutionRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":             "module fixture.local/atomicfix\n\ngo 1.21\n",
		"atomic_fix.go":      "package atomicfix\n\nfunc Value() string { return \"fixed\" }\n",
		"atomic_fix_test.go": "package atomicfix\n\nimport \"testing\"\n\nfunc TestValue(t *testing.T) { if Value() != \"fixed\" { t.Fatal(\"unexpected value\") } }\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
			t.Fatalf("write atomic fix fixture %s: %v", name, err)
		}
	}
	git := func(args ...string) {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		command.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	git("init")
	git("add", "go.mod", "atomic_fix.go", "atomic_fix_test.go")
	git("commit", "-m", "seed atomic fix execution fixture")
	return root
}
