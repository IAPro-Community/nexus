package runner

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestRemediationPersistsStrategyChangeBeforeRetry(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{})
	run := &MissionRun{Contract: AutonomyContract{MaxNoProgress: 4}}
	pkg := &PackageRun{PackageID: "pkg"}
	if err := r.markRemediation(run, pkg, StateCompiling, "same failure"); err != nil {
		t.Fatal(err)
	}
	if pkg.StrategyVariant != "ALTERNATE_APPROACH" || pkg.RemediationContext == "same failure" {
		t.Fatalf("first remediation did not persist strategy change: %+v", pkg)
	}
	if err := r.markRemediation(run, pkg, StateCompiling, "same failure"); err != nil {
		t.Fatal(err)
	}
	if pkg.StrategyVariant != "REPLAN_DECOMPOSE" {
		t.Fatalf("second identical failure must force replan strategy, got %q", pkg.StrategyVariant)
	}
}

func TestNeedsHumanContractIsStructuredAndDurable(t *testing.T) {
	r := NewMissionRunner(NewMemoryRunRepository(), &fakeExecutor{})
	run := &MissionRun{ID: "mission-1"}
	pkg := &PackageRun{PackageID: "task-1", RemediationContext: "strategy=REPLAN_DECOMPOSE"}
	r.blockNeedsHuman(run, pkg, "NO_PROGRESS", "verification kept failing", "Choose a different strategy", []string{"Review evidence"})
	if run.State != StateBlockedNeedsUser || run.NeedsHuman == nil {
		t.Fatalf("expected structured human blocker: %+v", run)
	}
	if run.NeedsHuman.ReasonCode != "NO_PROGRESS" || run.NeedsHuman.MissionID != run.ID || run.NeedsHuman.TaskID != pkg.PackageID {
		t.Fatalf("human blocker lost identity: %+v", run.NeedsHuman)
	}
	if len(run.NeedsHuman.RecommendedActions) != 1 || run.NeedsHuman.Context == "" || run.NeedsHuman.Timestamp.IsZero() {
		t.Fatalf("human blocker is not actionable: %+v", run.NeedsHuman)
	}
	if !run.NeedsHuman.Timestamp.After(time.Now().Add(-time.Minute)) {
		t.Fatalf("human blocker timestamp is too old: %v", run.NeedsHuman.Timestamp)
	}
}

func TestMissionRunnerLifecycleCompletesOnlyAfterVerificationAndReview(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"echo OK"}
	plan := PlanSpec{ID: "plan-1", ProjectID: "proj-1", Revision: 1, Packages: []PackageSpec{{ID: "pkg-1", Title: "Unit Tests", Goal: "Write tests"}}}

	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10 && run.State != StateCompletedVerified; i++ {
		run, _, err = r.ExecuteNextStep(context.Background(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("expected completed verified, got %s", run.State)
	}
	if len(run.PackageRuns[0].Verifications) == 0 {
		t.Fatal("verification evidence missing")
	}
	if len(run.PackageRuns[0].Verdicts) == 0 {
		t.Fatal("review evidence missing")
	}
}

func TestMissionRunnerBoundedRetries(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.MaxRetries = 2
	contract.VerificationCommands = []string{verificationFailureCommand()}
	plan := PlanSpec{ID: "plan-fail", ProjectID: "proj-1", Revision: 1, Packages: []PackageSpec{{ID: "pkg-fail", Title: "Always fails", Goal: "Fail"}}}

	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	var finalErr error
	for i := 0; i < 20; i++ {
		updated, _, stepErr := r.ExecuteNextStep(context.Background(), run.ID)
		if updated != nil {
			run = updated
		}
		if stepErr != nil && run.PackageRuns[0].State == StateFailed {
			finalErr = stepErr
			break
		}
	}
	if finalErr == nil {
		t.Fatal("expected bounded retry failure")
	}
	if run.State != StateBlockedNeedsUser && run.State != StateFailedNoProgress {
		t.Fatalf("unexpected run state %s", run.State)
	}
}

func TestMissionRunnerPreservesFlowStepExecutionContracts(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	spec := PackageSpec{
		ID: "step", Title: "Step", Goal: "Goal", Role: "tester",
		AssignmentStrategy: "AUTO", ResourcePolicy: "PRESERVE_QUOTA", Provider: "codex", Profile: "fast",
		SkillIDs: []string{"testing"}, MaestroSkills: []string{"verification"}, RelevantPaths: []string{"internal/nexus"},
		VerificationRequirements: []string{"echo step-specific"}, AcceptanceCriteria: []string{"verified"},
	}
	run, err := r.StartMissionRun(context.Background(), PlanSpec{ID: "flow", ProjectID: "p", Revision: 2, Packages: []PackageSpec{spec}}, t.TempDir(), contract, "legacy-default")
	if err != nil {
		t.Fatal(err)
	}
	pkg := run.PackageRuns[0]
	if pkg.AssignedAgent != "" {
		t.Fatalf("AUTO must not inherit legacy default Agent: %q", pkg.AssignedAgent)
	}
	if pkg.AssignmentStrategy != "AUTO" || pkg.ResourcePolicy != "PRESERVE_QUOTA" || pkg.Provider != "codex" || pkg.Profile != "fast" {
		t.Fatalf("flow execution contract lost: %+v", pkg)
	}
	if !reflect.DeepEqual(pkg.SkillIDs, []string{"testing"}) || !reflect.DeepEqual(pkg.MaestroSkills, []string{"verification"}) || !reflect.DeepEqual(pkg.RelevantPaths, []string{"internal/nexus"}) || !reflect.DeepEqual(pkg.VerificationRequirements, []string{"echo step-specific"}) {
		t.Fatalf("bounded step context lost: %+v", pkg)
	}
}

func TestMissionRunnerUsesStepVerificationRequirementsBeforeGlobalCommands(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &fakeExecutor{reviewOK: true}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{verificationFailureCommand()}
	plan := PlanSpec{ID: "flow", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal", VerificationRequirements: []string{"echo step-ok"}}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10 && run.State != StateCompletedVerified; i++ {
		run, _, err = r.ExecuteNextStep(context.Background(), run.ID)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	if run.State != StateCompletedVerified {
		t.Fatalf("step-specific verification should allow completion, got %s", run.State)
	}
	if got := run.PackageRuns[0].Verifications; len(got) == 0 || got[0].Command != "echo step-ok" {
		t.Fatalf("wrong verification evidence: %+v", got)
	}
}

type quotaFailExecutor struct {
	fakeExecutor
	failFirst     bool
	failureKind   model.FailureKind
	failureText   string
	allocateCount int
}

func (q *quotaFailExecutor) Allocate(ctx context.Context, run *MissionRun, pkg *PackageRun) (AllocationResult, error) {
	q.allocateCount++
	return q.fakeExecutor.Allocate(ctx, run, pkg)
}

func (q *quotaFailExecutor) Execute(ctx context.Context, run *MissionRun, pkg *PackageRun, prompt string) (ExecutionResult, error) {
	if q.failFirst {
		q.failFirst = false
		return ExecutionResult{
			RuntimeID: "runtime-classified-failure",
			Failure:   model.Failure{Kind: q.failureKind},
		}, fmt.Errorf("%s", q.failureText)
	}
	return q.fakeExecutor.Execute(ctx, run, pkg, prompt)
}

func TestMissionRunner_QuotaErrorRoutesToAllocatingForFailover(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &quotaFailExecutor{
		fakeExecutor: fakeExecutor{reviewOK: true},
		failFirst:    true,
		failureKind:  model.FailureRateLimit,
		failureText:  "provider status 429: rate limit exhausted",
	}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"echo OK"}
	plan := PlanSpec{ID: "flow", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Ready -> Allocating
	_, _, _ = r.ExecuteNextStep(context.Background(), run.ID)
	// 2. Allocating -> Compiling (allocateCount = 1)
	run, _, _ = r.ExecuteNextStep(context.Background(), run.ID)
	// 3. Compiling -> Executing
	run, _, _ = r.ExecuteNextStep(context.Background(), run.ID)
	// 4. Executing -> 429 error! Must route to StateAllocating via StateRemediating
	run, _, _ = r.ExecuteNextStep(context.Background(), run.ID)

	pkg := run.PackageRuns[0]
	if pkg.State != StateRemediating {
		t.Fatalf("expected StateRemediating, got %s", pkg.State)
	}
	if pkg.RetryFrom != StateAllocating {
		t.Fatalf("expected RetryFrom StateAllocating for 429 quota error, got %s", pkg.RetryFrom)
	}
	if pkg.FailureKind != model.FailureRateLimit {
		t.Fatalf("expected typed failure to be persisted, got %s", pkg.FailureKind)
	}

	// 5. Remediating -> routes to StateAllocating
	run, _, _ = r.ExecuteNextStep(context.Background(), run.ID)
	if run.PackageRuns[0].State != StateAllocating {
		t.Fatalf("expected package to return to StateAllocating for failover, got %s", run.PackageRuns[0].State)
	}

	// 6. Allocating re-runs! (allocateCount should now be 2)
	_, _, _ = r.ExecuteNextStep(context.Background(), run.ID)
	if exec.allocateCount != 2 {
		t.Fatalf("expected 2 allocations due to failover, got %d", exec.allocateCount)
	}
}

func TestMissionRunner_QuotaWordDoesNotTriggerFailoverWithoutTypedFailure(t *testing.T) {
	repo := NewMemoryRunRepository()
	exec := &quotaFailExecutor{
		fakeExecutor: fakeExecutor{reviewOK: true},
		failFirst:    true,
		failureKind:  model.FailureUnknown,
		failureText:  "failed to write usage.json quota cache",
	}
	r := NewMissionRunner(repo, exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{"echo OK"}
	plan := PlanSpec{ID: "flow-false-positive", ProjectID: "p", Revision: 1, Packages: []PackageSpec{{ID: "step", Title: "Step", Goal: "Goal"}}}
	run, err := r.StartMissionRun(context.Background(), plan, t.TempDir(), contract, "")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 4; i++ {
		run, _, _ = r.ExecuteNextStep(context.Background(), run.ID)
	}

	pkg := run.PackageRuns[0]
	if pkg.FailureKind != model.FailureUnknown {
		t.Fatalf("expected unknown typed failure, got %s", pkg.FailureKind)
	}
	if pkg.RetryFrom != StateCompiling {
		t.Fatalf("generic error mentioning quota must not trigger failover, got retry from %s", pkg.RetryFrom)
	}
	if exec.allocateCount != 1 {
		t.Fatalf("generic quota text must not reallocate, got %d allocations", exec.allocateCount)
	}
}
