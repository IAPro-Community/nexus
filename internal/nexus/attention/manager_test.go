package attention

import (
	"testing"
	"time"
)

func TestManagerIngestAndSubscribe(t *testing.T) {
	manager := NewManager(nil)
	sub := manager.Subscribe("project-1")

	event := AttentionEvent{
		ID:        "evt-1",
		ProjectID: "project-1",
		State:     StateWorking,
		Reason:    "building",
		Source:    SourceTerminalHeuristic,
		OccurredAt: time.Now(),
	}

	_, inserted, err := manager.Ingest(event)
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
	if !inserted {
		t.Fatal("expected event to be inserted")
	}

	select {
	case received := <-sub:
		if received.ID != "evt-1" {
			t.Fatalf("expected event evt-1, got %s", received.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestManagerIngestDuplicateIsIdempotent(t *testing.T) {
	manager := NewManager(nil)

	event := AttentionEvent{
		ID:        "evt-1",
		ProjectID: "project-1",
		State:     StateWorking,
		Reason:    "building",
		Source:    SourceTerminalHeuristic,
		OccurredAt: time.Now(),
	}

	_, inserted, err := manager.Ingest(event)
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
	if !inserted {
		t.Fatal("expected first event to be inserted")
	}

	_, inserted, err = manager.Ingest(event)
	if err != nil {
		t.Fatalf("Ingest duplicate error: %v", err)
	}
	if inserted {
		t.Fatal("expected duplicate event to not be inserted")
	}
}

func TestManagerProjectSummary(t *testing.T) {
	manager := NewManager(nil)
	base := time.Now()

	manager.Ingest(AttentionEvent{
		ID:        "evt-1",
		ProjectID: "project-1",
		State:     StateWorking,
		Reason:    "building",
		Source:    SourceTerminalHeuristic,
		OccurredAt: base,
	})

	manager.Ingest(AttentionEvent{
		ID:        "evt-2",
		ProjectID: "project-1",
		State:     StateFailed,
		Reason:    "tests failed",
		Source:    SourceSystem,
		OccurredAt: base.Add(time.Second),
	})

	summary := manager.ProjectSummary("project-1")
	if summary.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.PriorityState != StateFailed {
		t.Fatalf("expected failed priority, got %s", summary.PriorityState)
	}
}

func TestManagerSubscribeAll(t *testing.T) {
	manager := NewManager(nil)
	sub := manager.SubscribeAll()

	manager.Ingest(AttentionEvent{
		ID:        "evt-1",
		ProjectID: "project-1",
		State:     StateWorking,
		Reason:    "building",
		Source:    SourceTerminalHeuristic,
		OccurredAt: time.Now(),
	})

	select {
	case received := <-sub:
		if received.ID != "evt-1" {
			t.Fatalf("expected event evt-1, got %s", received.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestManagerUnsubscribe(t *testing.T) {
	manager := NewManager(nil)
	sub := manager.Subscribe("project-1")
	manager.Unsubscribe("project-1", sub)

	// After unsubscribe, channel should be closed
	_, ok := <-sub
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

func TestManagerAllProjectSummaries(t *testing.T) {
	manager := NewManager(nil)
	base := time.Now()

	manager.Ingest(AttentionEvent{
		ID:        "evt-1",
		ProjectID: "project-1",
		State:     StateWorking,
		Reason:    "building",
		Source:    SourceTerminalHeuristic,
		OccurredAt: base,
	})

	manager.Ingest(AttentionEvent{
		ID:        "evt-2",
		ProjectID: "project-2",
		State:     StateFailed,
		Reason:    "crashed",
		Source:    SourceSystem,
		OccurredAt: base.Add(time.Second),
	})

	summaries := manager.AllProjectSummaries()
	if len(summaries) != 2 {
		t.Fatalf("expected 2 project summaries, got %d", len(summaries))
	}
}
