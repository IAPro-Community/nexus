package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestFaultInjectionStoreReopenPreservesIntervention(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRunRepository()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_reopen", ProjectID: "p", State: StateBlockedNeedsUser,
		NeedsHuman: &HumanIntervention{
			ID: "intv", ReasonCode: "SECURITY", Summary: "auth", Question: "Approve?",
			MissionID: "run_reopen", TaskID: "pkg", Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg"}},
		},
		UpdatedAt: now, LastProgressAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewMemoryRunRepository()
	var copy MissionRun
	if err := json.Unmarshal(raw, &copy); err != nil {
		t.Fatal(err)
	}
	if err := reopened.SaveRun(ctx, &copy); err != nil {
		t.Fatal(err)
	}
	got, err := reopened.GetRun(ctx, "run_reopen")
	if err != nil {
		t.Fatal(err)
	}
	if got.NeedsHuman == nil || got.NeedsHuman.ID != "intv" || got.State != StateBlockedNeedsUser {
		t.Fatalf("reopen lost intervention: %+v", got)
	}
	center := NewAttentionCenter(reopened)
	first, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := center.ListAttention(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.NeedsYou) != 1 || len(second.NeedsYou) != 1 {
		t.Fatalf("restart must not duplicate attention: first=%d second=%d", len(first.NeedsYou), len(second.NeedsYou))
	}
	if AttentionDedupKey(got) != AttentionDedupKey(run) {
		t.Fatalf("dedup key must survive reopen: %q vs %q", AttentionDedupKey(got), AttentionDedupKey(run))
	}
}

func TestFaultInjectionStaleRoutingJSONFailsClosed(t *testing.T) {
	pkg := PackageRun{PackageID: "pkg", RoutingDecisionJSON: "{not-json"}
	if json.Valid([]byte(pkg.RoutingDecisionJSON)) {
		t.Fatal("fixture must be invalid JSON")
	}
}

func TestFaultInjectionDuplicateInterventionResolvesOnce(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, &fakeExecutor{})
	ctx := context.Background()
	now := time.Now().UTC()
	run := &MissionRun{
		ID: "run_dup", State: StateBlockedNeedsUser,
		PackageRuns: []PackageRun{{ID: "pkg-1", PackageID: "pkg-1", State: StateFailed, DispatchState: DispatchFailedBeforeDispatch}},
		NeedsHuman: &HumanIntervention{
			ID: "intv_dup", ReasonCode: "SECURITY", Summary: "x", Question: "?",
			MissionID: "run_dup", TaskID: "pkg-1", Source: "runner", Timestamp: now, Version: 1,
			Options: []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, PackageID: "pkg-1"}},
		},
		Contract: DefaultAutonomyContract(), UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveIntervention(ctx, "run_dup", "intv_dup", 1, "replan-package", "tester"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveIntervention(ctx, "run_dup", "intv_dup", 1, "replan-package", "tester"); err != nil {
		t.Fatalf("duplicate resolve must be idempotent: %v", err)
	}
}

func TestFaultInjectionQuotaFailoverIsNotAttention(t *testing.T) {
	run := &MissionRun{ID: "run_q", State: StateAllocating, UpdatedAt: time.Now().UTC()}
	if item := AttentionFromRun(run); item != nil {
		t.Fatalf("quota/allocation recovery must stay silent: %+v", item)
	}
}

// failoverResource is one allocatable provider/profile account for the
// fault-injection executor. exhausted mirrors the real scheduler semantics
// after a typed quota/rate-limit failure: the failing resource stops being a
// candidate (mission_executor.go filterOutFailingResource).
type failoverResource struct {
	agentID   string
	provider  string
	profile   string
	workspace string
	exhausted bool
}

// failoverAttributionExecutor injects one quota exhaustion at the provider
// surface and otherwise follows the real routing contract of
// nexusPackageExecutor.Allocate: after the typed quota failure the runner
// re-enters StateAllocating, the allocator filters out the exhausted
// provider/profile and REWRITES pkg.Provider/pkg.Profile to whoever will
// actually execute the next attempt. There is no hot mid-run model switch in
// the runner: re-allocation on re-entry is the reassignment mechanism.
type failoverAttributionExecutor struct {
	fakeExecutor
	mu           sync.Mutex
	providers    []failoverResource
	allocations  []string
	executions   []string
	quotaFailing bool
}

func (f *failoverAttributionExecutor) Allocate(_ context.Context, _ *MissionRun, pkg *PackageRun) (AllocationResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.providers {
		res := &f.providers[i]
		if res.exhausted {
			continue
		}
		pkg.Provider, pkg.Profile = res.provider, res.profile
		f.allocations = append(f.allocations, pkg.PackageID+"->"+res.provider)
		return AllocationResult{AgentID: res.agentID, Workspace: res.workspace}, nil
	}
	return AllocationResult{}, fmt.Errorf("no eligible runtime account for package %s: every allocated resource exhausted its quota", pkg.PackageID)
}

func (f *failoverAttributionExecutor) Execute(_ context.Context, _ *MissionRun, pkg *PackageRun, _ string) (ExecutionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	runtimeID := "rt-" + pkg.Provider
	f.executions = append(f.executions, pkg.PackageID+"@"+pkg.Provider+":"+runtimeID)
	if !f.quotaFailing {
		f.quotaFailing = true
		for i := range f.providers {
			if f.providers[i].provider == pkg.Provider {
				f.providers[i].exhausted = true
			}
		}
		return ExecutionResult{
			RuntimeID: runtimeID,
			Failure:   model.Failure{Kind: model.FailureQuota},
		}, fmt.Errorf("provider status 429: quota exceeded for %s:%s", pkg.Provider, pkg.Profile)
	}
	return ExecutionResult{RuntimeID: runtimeID, Output: "implemented under " + pkg.Provider}, nil
}

func failoverPackageRun(run *MissionRun, packageID string) *PackageRun {
	for i := range run.PackageRuns {
		if run.PackageRuns[i].PackageID == packageID {
			return &run.PackageRuns[i]
		}
	}
	return nil
}

// TestFaultInjectionQuotaFailureAfterDispatchProhibitsReplay pins the safe
// behavior when a provider reports quota exhaustion after receiving a task.
// Without a provider receipt proving that no output or effects occurred, the
// runner preserves attribution and blocks instead of allocating another account.
func TestFaultInjectionQuotaFailureAfterDispatchProhibitsReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := t.TempDir()
	workspaceA := filepath.Join(root, "workspace-provider-a")
	workspaceB := filepath.Join(root, "workspace-provider-b")
	for _, dir := range []string{workspaceA, workspaceB} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	exec := &failoverAttributionExecutor{
		fakeExecutor: fakeExecutor{reviewOK: true},
		providers: []failoverResource{
			{agentID: "agt-provider-a", provider: "provider-a", profile: "profile-a", workspace: workspaceA},
			{agentID: "agt-provider-b", provider: "provider-b", profile: "profile-b", workspace: workspaceB},
		},
	}
	r := NewMissionRunner(NewMemoryRunRepository(), exec)
	contract := DefaultAutonomyContract()
	contract.VerificationCommands = []string{verificationSuccessCommand()}
	plan := PlanSpec{ID: "plan-quota-failure", ProjectID: "proj-1", Revision: 1, Packages: []PackageSpec{
		{ID: "pkg-1", Title: "First", Goal: "first"},
		{ID: "pkg-2", Title: "Second", Goal: "second", Dependencies: []string{"pkg-1"}},
	}}
	run, err := r.StartMissionRun(ctx, plan, root, contract, "")
	if err != nil {
		t.Fatal(err)
	}

	for step := 0; step < 12 && run.State != StateBlockedNeedsUser; step++ {
		run, _, _ = r.ExecuteNextStep(ctx, run.ID)
		if run == nil {
			t.Fatalf("step %d returned no run", step)
		}
	}
	if run.State != StateBlockedNeedsUser || run.NeedsHuman == nil {
		t.Fatalf("post-dispatch quota failure must request reconciliation, got state=%s", run.State)
	}
	pkg := failoverPackageRun(run, "pkg-1")
	if pkg == nil {
		t.Fatalf("missing failed package: %+v", run.PackageRuns)
	}
	if pkg.FailureKind != model.FailureQuota || pkg.DispatchState != DispatchFailed || !pkg.ReplayProhibited {
		t.Fatalf("expected preserved quota failure with replay prohibited: %+v", pkg)
	}
	if pkg.AssignedRuntime != "rt-provider-a" || pkg.Attempt != 1 {
		t.Fatalf("failed attempt attribution must remain with provider-a: %+v", pkg)
	}
	if len(exec.allocations) != 1 || exec.allocations[0] != "pkg-1->provider-a" {
		t.Fatalf("ambiguous provider failure must not trigger reallocation: %v", exec.allocations)
	}
	if len(exec.executions) != 1 || exec.executions[0] != "pkg-1@provider-a:rt-provider-a" {
		t.Fatalf("ambiguous provider failure must not be replayed: %v", exec.executions)
	}
}

func TestOvernightSoakHarnessBounded(t *testing.T) {
	duration := 15 * time.Second
	if raw := os.Getenv("OVERNIGHT_SOAK_DURATION"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			t.Fatal(err)
		}
		duration = parsed
	}
	if duration <= 0 {
		t.Fatalf("soak duration must be positive, got %s", duration)
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	workspace := t.TempDir()
	cycles := 0
	const cyclePause = 250 * time.Millisecond

soakLoop:
	for {
		select {
		case <-ctx.Done():
			break soakLoop
		default:
		}

		exec := &overnightAcceptanceExecutor{globalRepairExecutor: globalRepairExecutor{fakeExecutor: fakeExecutor{reviewOK: true}}}
		r := NewMissionRunner(NewMemoryRunRepository(), exec)
		contract := DefaultAutonomyContract()
		contract.VerificationCommands = []string{verificationSuccessCommand()}
		contract.GlobalVerificationCommands = []string{verificationSuccessCommand()}
		plan := PlanSpec{ID: "soak", ProjectID: "p", Revision: 1, Packages: []PackageSpec{
			{ID: "a", Title: "a", Goal: "a", ParallelGroup: "g"},
			{ID: "b", Title: "b", Goal: "b", Dependencies: []string{"a"}},
		}}
		run, err := r.StartMissionRun(ctx, plan, workspace, contract, "")
		if err != nil {
			if ctx.Err() != nil {
				break soakLoop
			}
			t.Fatalf("soak cycle %d start failed: %v", cycles+1, err)
		}
		completed, err := r.RunToTerminal(ctx, run.ID)
		if err != nil && ctx.Err() == nil {
			t.Fatalf("soak cycle %d failed: %v", cycles+1, err)
		}
		if completed == nil {
			if ctx.Err() != nil {
				break soakLoop
			}
			t.Fatalf("soak cycle %d returned no run", cycles+1)
		}
		switch completed.State {
		case StateCompletedVerified, StateFailedNoProgress, StateBlockedNeedsUser, StateCanceledByUser, StateFailed, StateFailedBudgetExceeded, StateFailedVerification:
		default:
			t.Fatalf("soak cycle %d must end in an explicit terminal state, got %s", cycles+1, completed.State)
		}
		cycles++

		timer := time.NewTimer(cyclePause)
		select {
		case <-ctx.Done():
			timer.Stop()
			break soakLoop
		case <-timer.C:
		}
	}
	if cycles == 0 {
		t.Fatalf("soak completed no cycles during %s", duration)
	}
	t.Logf("completed %d mission cycles during %s", cycles, duration)
}
