package nexus

import (
	"context"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type restartProbeExecutor struct {
	executed atomic.Int32
}

func (e *restartProbeExecutor) Allocate(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.AllocationResult, error) {
	return runner.AllocationResult{AgentID: "agent-restart", Workspace: "."}, nil
}

func (e *restartProbeExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "prompt-restart", Content: "resume"}, nil
}

func (e *restartProbeExecutor) Execute(context.Context, *runner.MissionRun, *runner.PackageRun, string) (runner.ExecutionResult, error) {
	e.executed.Add(1)
	return runner.ExecutionResult{RuntimeID: "runtime-restart"}, nil
}

func (e *restartProbeExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-restart", ReviewedAt: time.Now().UTC()}, nil
}

func TestInterventionResolutionSurvivesStoreReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nexus.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	project, err := st.CreateProject(store.Project{Name: "Resume", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Resume plan"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	contract := runner.DefaultAutonomyContract()
	contract.RequireVerification = false
	run := &runner.MissionRun{
		ID: "run-restart", PlanID: plan.ID, PlanRevision: plan.CurrentRevision, ProjectID: project.ID,
		State: runner.StateBlockedNeedsUser, Contract: contract, StartedAt: now, UpdatedAt: now,
		PackageRuns: []runner.PackageRun{{
			ID: "pkg", PackageID: "pkg", State: runner.StateFailed, DispatchState: runner.DispatchFailedBeforeDispatch,
			RoutingDecisionJSON: `{"task_id":"pkg","selected_provider":"agy","selected_profile":"frontend","selected_model":"medium","reason":"task-aware fallback"}`,
		}},
		NeedsHuman: &runner.HumanIntervention{
			ID: "intervention-restart", Version: 7, MissionID: "run-restart", TaskID: "pkg", Timestamp: now,
			Options: []runner.InterventionOption{{ID: "replan-package", Operation: runner.InterventionReplanPackage, PackageID: "pkg"}},
		},
	}
	repo := newStoreRunRepository(st)
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	first, created, err := runner.NewMissionRunner(repo, nil).ResolveInterventionWithOutcome(ctx, run.ID, "intervention-restart", 7, "replan-package", "user-1")
	if err != nil || !created {
		t.Fatalf("first resolution failed: created=%v err=%v", created, err)
	}
	if first.ResumeRequest == nil || first.ResumeRequest.Status != "PENDING" {
		t.Fatalf("resume request was not durably pending: %+v", first.ResumeRequest)
	}
	events, err := st.ListMissionRunEvents(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].EventType != "human_intervention.resolved" || events[1].EventType != "mission.resume_requested" {
		t.Fatalf("expected atomic resolution outbox, got %+v", events)
	}
	// Explicit close before reopen to simulate process restart.
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopenedRepo := newStoreRunRepository(reopened)
	loaded, err := reopenedRepo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NeedsHuman == nil || loaded.NeedsHuman.Resolution == nil || loaded.NeedsHuman.Resolution.OptionID != "replan-package" {
		t.Fatalf("canonical resolution did not survive restart: %+v", loaded.NeedsHuman)
	}
	if loaded.ResumeRequest == nil || loaded.ResumeRequest.IdempotencyKey == "" || loaded.ResumeRequest.Status != "PENDING" {
		t.Fatalf("durable resume request did not survive restart: %+v", loaded.ResumeRequest)
	}
	if loaded.PackageRuns[0].RoutingDecisionJSON == "" {
		t.Fatal("persisted routing decision was lost across restart")
	}
	report, err := BuildRoutingDecisionReport(loaded)
	if err != nil {
		t.Fatalf("reloaded routing decision was not explainable: %v", err)
	}
	if len(report.Decisions) != 1 || report.Decisions[0].Decision.SelectedModel != "medium" {
		t.Fatalf("unexpected reloaded routing report: %+v", report)
	}
	duplicate, created, err := runner.NewMissionRunner(reopenedRepo, nil).ResolveInterventionWithOutcome(ctx, run.ID, "intervention-restart", 7, "replan-package", "another-retry")
	if err != nil || created {
		t.Fatalf("duplicate resolution was not semantic idempotent: created=%v err=%v", created, err)
	}
	if duplicate.NeedsHuman.Resolution.IdempotencyKey != loaded.NeedsHuman.Resolution.IdempotencyKey {
		t.Fatal("duplicate resolution changed its semantic identity")
	}
	events, err = reopened.ListMissionRunEvents(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("duplicate resolution emitted additional durable events: %d", len(events))
	}

	probe := &restartProbeExecutor{}
	recovered := &Nexus{st: reopened, runner: runner.NewMissionRunner(reopenedRepo, probe)}
	if err := recovered.RecoverMissionRuns(ctx); err != nil {
		t.Fatal(err)
	}
	var final *runner.MissionRun
	for attempts := 0; attempts < 10000; attempts++ {
		final, err = reopenedRepo.GetRun(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if final.State == runner.StateCompletedVerified {
			break
		}
		runtime.Gosched()
	}
	if final.State != runner.StateCompletedVerified || final.ResumeRequest == nil || final.ResumeRequest.Status != "COMPLETED" {
		t.Fatalf("recovered mission did not complete its pending resume: %+v", final)
	}
	if probe.executed.Load() != 1 {
		t.Fatalf("restart performed %d external continuations, want exactly one", probe.executed.Load())
	}
}
