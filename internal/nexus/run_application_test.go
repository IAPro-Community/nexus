package nexus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestRunApplicationServiceRejectsCanceledContext(t *testing.T) {
	service := NewRunApplicationService(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wantCanceled := func(t *testing.T, err error) {
		t.Helper()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	}

	_, err := service.List(ctx)
	wantCanceled(t, err)
	_, err = service.Get(ctx, "run-1")
	wantCanceled(t, err)
	_, err = service.Start(ctx, RunStartRequest{PlanID: "plan-1", Revision: 1})
	wantCanceled(t, err)
	_, err = service.Step(ctx, "run-1")
	wantCanceled(t, err)
	_, err = service.Pause(ctx, "run-1", "test")
	wantCanceled(t, err)
	_, err = service.TakeControl(ctx, "run-1", "test")
	wantCanceled(t, err)
	_, err = service.Resume(ctx, "run-1")
	wantCanceled(t, err)
	_, err = service.ReturnToMission(ctx, "run-1")
	wantCanceled(t, err)
	_, err = service.Cancel(ctx, "run-1", "test")
	wantCanceled(t, err)
	_, err = service.SubmitCoordination(ctx, "run-1", runner.CoordinationRequest{}, false)
	wantCanceled(t, err)
	_, err = service.MaterializeCoordination(ctx, "run-1", "coord-1")
	wantCanceled(t, err)
	_, err = service.ResolveCoordination(ctx, "run-1", "coord-1", runner.CoordinationResolved, "", "")
	wantCanceled(t, err)
}

func TestRunApplicationServicePublishesCorrelatedMissionTimeline(t *testing.T) {
	bus := events.NewBus(10)
	service := NewRunApplicationServiceWithBus(nil, bus)
	run := &runner.MissionRun{ID: "run-123", ProjectID: "project-1", CurrentPkgIndex: 0, PackageRuns: []runner.PackageRun{{AssignedRuntime: "runtime-1", AssignedAgent: "agent-1", Provider: "codex", Profile: "work"}}}

	service.publishRunEvent(run, events.EventMissionStepStarted, "step started", map[string]any{"step": "A"})
	history := bus.GetHistory("runtime-1", 1)
	if len(history) != 1 || history[0].CorrelationID != "run-123" || history[0].Type != events.EventMissionStepStarted {
		t.Fatalf("unexpected correlated mission timeline: %+v", history)
	}
	if history[0].Data["project_id"] != "project-1" || history[0].Data["agent_id"] != "agent-1" {
		t.Fatalf("mission event lacks durable ownership identity: %+v", history[0].Data)
	}
}

func TestRunApplicationServicePublishesCoordinationTransition(t *testing.T) {
	bus := events.NewBus(10)
	service := NewRunApplicationServiceWithBus(nil, bus)
	before := &runner.MissionRun{
		ID:              "run-coord",
		ProjectID:       "project-1",
		CurrentPkgIndex: 0,
		PackageRuns:     []runner.PackageRun{{AssignedRuntime: "runtime-1", AssignedAgent: "agent-1"}},
		CoordinationRequests: []runner.CoordinationRequest{{
			ID:     "coord-1",
			Status: runner.CoordinationPending,
		}},
	}
	after := &runner.MissionRun{
		ID:              "run-coord",
		ProjectID:       "project-1",
		CurrentPkgIndex: 0,
		PackageRuns:     []runner.PackageRun{{AssignedRuntime: "runtime-1", AssignedAgent: "agent-1"}},
		CoordinationRequests: []runner.CoordinationRequest{{
			ID:              "coord-1",
			Status:          runner.CoordinationResolved,
			ChildPackageID:  "coord-child",
			ResultReceiptID: "receipt-1",
		}},
	}

	service.publishCoordinationTransitions(before, after)
	history := bus.GetHistory("runtime-1", 5)
	if len(history) != 1 {
		t.Fatalf("expected one coordination transition event, got %d", len(history))
	}
	if history[0].CorrelationID != "run-coord" || history[0].Data["request_id"] != "coord-1" {
		t.Fatalf("unexpected coordination transition event: %+v", history[0])
	}
	if history[0].Data["coordination_phase"] != "RESOLVED" || history[0].Data["receipt_id"] != "receipt-1" {
		t.Fatalf("coordination transition event lacks durable lineage: %+v", history[0].Data)
	}
}

func TestRunApplicationServiceResolvesArtifactCoordinationFromReceipt(t *testing.T) {
	ctx := context.Background()
	repo := runner.NewMemoryRunRepository()
	missionRunner := runner.NewMissionRunner(repo, nil)
	n := &Nexus{runner: missionRunner}
	service := NewRunApplicationService(n)
	now := time.Now().UTC()
	run := &runner.MissionRun{
		ID:           "run-artifact",
		PlanID:       "plan-artifact",
		PlanRevision: 1,
		ProjectID:    "project-artifact",
		Workspace:    "/tmp/project",
		State:        runner.StateExecuting,
		Contract:     runner.DefaultAutonomyContract(),
		PackageRuns: []runner.PackageRun{{
			ID:        "pkgrun-source",
			PackageID: "source",
			State:     runner.StateVerified,
			StartedAt: now,
			WorkReceipt: &runner.WorkReceipt{
				ID:          "receipt-source",
				RunID:       "run-artifact",
				StepID:      "source",
				Status:      "VERIFIED",
				Summary:     "schema artifact produced",
				Artifacts:   []string{"artifacts/schema.json"},
				StartedAt:   now,
				CompletedAt: now,
			},
		}},
		CoordinationRequests: []runner.CoordinationRequest{{
			ID:              "coord-artifact",
			RunID:           "run-artifact",
			SourcePackageID: "source",
			Operation:       runner.CoordinationArtifactRequest,
			Subject:         "schema",
			ArtifactRefs:    []string{"artifacts/schema.json"},
			Status:          runner.CoordinationPending,
			CreatedAt:       now,
		}},
		StartedAt: now,
		UpdatedAt: now,
	}
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}

	resolved, source, ok := service.resolveCoordinationFromDurableContext(
		ctx,
		run,
		&run.CoordinationRequests[0],
	)
	if !ok || source != "WORK_RECEIPT" {
		t.Fatalf("expected WorkReceipt resolution, ok=%v source=%q", ok, source)
	}
	if resolved.Status != runner.CoordinationResolved || resolved.ResultReceiptID != "receipt-source" {
		t.Fatalf("unexpected durable resolution: %+v", resolved)
	}
	loaded, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CoordinationRequests[0].Status != runner.CoordinationResolved {
		t.Fatalf("resolution was not persisted: %+v", loaded.CoordinationRequests[0])
	}
}
