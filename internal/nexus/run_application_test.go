package nexus

import (
	"context"
	"errors"
	"testing"

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
