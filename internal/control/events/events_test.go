package events

import (
	"sync"
	"testing"
	"time"
)

func TestEventBusPubSubAndHistory(t *testing.T) {
	bus := NewBus(10)

	runtimeID := "rt-test-events"
	ch, unsub := bus.Subscribe(runtimeID)
	defer unsub()

	evt1 := NewEvent(runtimeID, "codex", "work", EventAgentWorking, "Refactoring code", nil)
	bus.Publish(evt1)

	select {
	case received := <-ch:
		if received.Type != EventAgentWorking || received.Summary != "Refactoring code" {
			t.Errorf("unexpected event received: %+v", received)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for event on channel")
	}

	// Verify history
	hist := bus.GetHistory(runtimeID, 5)
	if len(hist) != 1 || hist[0].ID != evt1.ID {
		t.Errorf("unexpected history: %+v", hist)
	}
}

func TestNewEventWithCorrelationPreservesTimelineIdentity(t *testing.T) {
	e := NewEventWithCorrelation("run-123", "runtime-1", "codex", "work", EventProcessStarted, "started", nil)
	if e.CorrelationID != "run-123" || e.RuntimeID != "runtime-1" {
		t.Fatalf("event identity mismatch: %+v", e)
	}
}

func TestRuntimeLifecycleEventTypesAreStable(t *testing.T) {
	if EventRuntimeStarted != "RUNTIME_STARTED" || EventRuntimeStopped != "RUNTIME_STOPPED" || EventRuntimeFailed != "RUNTIME_FAILED" {
		t.Fatalf("runtime lifecycle contract changed: %q %q %q", EventRuntimeStarted, EventRuntimeStopped, EventRuntimeFailed)
	}
}

func TestHandoffTimelineCorrelationUsesLineageID(t *testing.T) {
	lineageID := "lineage-123"
	e := NewEventWithCorrelation(lineageID, "runtime-target", "codex", "work", EventHandoffCompleted, "handoff completed", map[string]any{
		"source_id": "runtime-source",
	})
	if e.CorrelationID != lineageID {
		t.Fatalf("expected handoff event correlation %q, got %q", lineageID, e.CorrelationID)
	}
	if e.RuntimeID != "runtime-target" || e.Type != EventHandoffCompleted {
		t.Fatalf("handoff event lost runtime identity: %+v", e)
	}
}

func TestEventBusDurableRecorderHook(t *testing.T) {
	bus := NewBus(10)

	var recorded []Event
	var mu sync.Mutex

	bus.SetRecorder(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		recorded = append(recorded, e)
	})

	evt1 := NewEvent("rt-1", "gemini", "default", EventSessionStarted, "Session started", nil)
	evt2 := NewEvent("rt-1", "gemini", "default", EventAgentWorking, "Writing code", nil)

	bus.Publish(evt1)
	bus.Publish(evt2)

	mu.Lock()
	defer mu.Unlock()
	if len(recorded) != 2 {
		t.Fatalf("expected 2 recorded events, got %d", len(recorded))
	}
	if recorded[0].ID != evt1.ID || recorded[1].ID != evt2.ID {
		t.Errorf("mismatched recorded event IDs: %+v", recorded)
	}
}
