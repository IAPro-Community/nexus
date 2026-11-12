package app

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func continuityFollowRuntime(id, parentID, replacesRuntime string, state registry.RuntimeState, agentID string) registry.RuntimeSession {
	labels := map[string]string{
		"nexus.continuity_replaces_runtime": replacesRuntime,
	}
	return registry.RuntimeSession{
		RuntimeID:       id,
		ParentRuntimeID: parentID,
		AgentID:         agentID,
		ProviderID:      "codex",
		ProfileID:       "work",
		State:           state,
		Labels:          labels,
	}
}

func TestWaitForContinuityReplacementFollowsSuccessor(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/registry.json")
	source := continuityFollowRuntime("rt-a", "", "", registry.StateHandoff, "")
	source.Labels["nexus.continuity_pending"] = "true"
	if err := reg.Register(source); err != nil {
		t.Fatalf("register source: %v", err)
	}
	next := continuityFollowRuntime("rt-b", "rt-a", "rt-a", registry.StateRunning, "")
	if err := reg.Register(next); err != nil {
		t.Fatalf("register next: %v", err)
	}

	got, ok := waitForContinuityReplacementReg(reg, "rt-a", "")
	if !ok || got.RuntimeID != "rt-b" {
		t.Fatalf("follow target=%+v ok=%t, want rt-b", got, ok)
	}
}

func TestWaitForContinuityReplacementWaitsForRunningSuccessor(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/registry.json")
	source := continuityFollowRuntime("rt-a", "", "", registry.StateHandoff, "")
	source.Labels["nexus.continuity_pending"] = "true"
	_ = reg.Register(source)
	// Successor still starting; the follower must wait for a usable state.
	_ = reg.Register(continuityFollowRuntime("rt-b", "rt-a", "rt-a", registry.StateStarting, ""))

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = reg.UpdateState("rt-b", registry.StateRunning)
	}()
	got, ok := waitForContinuityReplacementReg(reg, "rt-a", "")
	if !ok || got.RuntimeID != "rt-b" {
		t.Fatalf("follow target=%+v ok=%t, want rt-b after it started", got, ok)
	}
}

func TestWaitForContinuityReplacementReturnsFastWithoutPendingFlag(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/registry.json")
	source := continuityFollowRuntime("rt-a", "", "", registry.StateStopped, "")
	_ = reg.Register(source)
	_ = reg.Register(continuityFollowRuntime("rt-b", "rt-a", "rt-a", registry.StateRunning, ""))

	started := time.Now()
	got, ok := waitForContinuityReplacementReg(reg, "rt-a", "")
	elapsed := time.Since(started)
	if ok || got.RuntimeID != "" {
		t.Fatalf("normal exit must not follow: target=%+v ok=%t", got, ok)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("normal exit waited %v; follow must be gated on the pending flag", elapsed)
	}
}

func TestWaitForContinuityReplacementPendingFalseDoesNotFollow(t *testing.T) {
	// A handoff successor carries continuity_pending=false: when it ends, an
	// attached CLI must exit immediately even though a replaced label exists.
	reg := registry.NewRegistry(t.TempDir() + "/registry.json")
	source := continuityFollowRuntime("rt-b", "rt-a", "rt-a", registry.StateStopped, "")
	source.Labels["nexus.continuity_pending"] = "false"
	_ = reg.Register(source)
	_ = reg.Register(continuityFollowRuntime("rt-c", "rt-b", "rt-b", registry.StateRunning, ""))

	got, ok := waitForContinuityReplacementReg(reg, "rt-b", "")
	if ok || got.RuntimeID != "" {
		t.Fatalf("successor end must not follow: target=%+v ok=%t", got, ok)
	}
}

func TestWaitForContinuityReplacementTimesOutWithoutSuccessor(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/registry.json")
	source := continuityFollowRuntime("rt-a", "", "", registry.StateHandoff, "")
	source.Labels["nexus.continuity_pending"] = "true"
	_ = reg.Register(source)

	got, ok := waitForContinuityReplacementReg(reg, "rt-a", "")
	if ok || got.RuntimeID != "" {
		t.Fatalf("failed handoff must not follow: target=%+v ok=%t", got, ok)
	}
}

func TestWaitForContinuityReplacementRejectsUnrelatedSuccessor(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/registry.json")
	source := continuityFollowRuntime("rt-a", "", "", registry.StateHandoff, "")
	source.Labels["nexus.continuity_pending"] = "true"
	_ = reg.Register(source)
	// Different lineage: parent and replace labels do not match rt-a.
	_ = reg.Register(continuityFollowRuntime("rt-x", "rt-other", "rt-other", registry.StateRunning, ""))

	got, ok := waitForContinuityReplacementReg(reg, "rt-a", "")
	if ok {
		t.Fatalf("unrelated runtime must not be adopted as follow target: %+v", got)
	}
}
