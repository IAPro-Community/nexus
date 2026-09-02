package attention

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeEventSanitizesBoundedRedactedMessage(t *testing.T) {
	occurredAt := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rawMessage := strings.Repeat(" step", 80) + " Authorization: Bearer sk-proj-1234567890abcdef1234567890abcdef1234567890 \x00 \n"

	event, err := NormalizeEvent(AttentionEvent{
		ID:             " event-1 ",
		ProjectID:      " project-1 ",
		AgentID:        " agent-1 ",
		RuntimeID:      " runtime-1 ",
		FlowRunID:      " flow-1 ",
		ViewLogicalKey: " view-1 ",
		State:          StateWorking,
		Reason:         " waiting on external compile ",
		Message:        rawMessage,
		Source:         SourceStructuredProviderEvent,
		OccurredAt:     occurredAt,
	})
	if err != nil {
		t.Fatalf("NormalizeEvent returned error: %v", err)
	}

	if event.ID != "event-1" || event.ProjectID != "project-1" {
		t.Fatalf("expected trimmed IDs, got %#v", event)
	}
	if event.Reason != "waiting on external compile" {
		t.Fatalf("expected trimmed reason, got %q", event.Reason)
	}
	if strings.Contains(event.Message, "sk-proj-1234567890abcdef1234567890abcdef1234567890") {
		t.Fatalf("expected message redaction, got %q", event.Message)
	}
	if strings.ContainsRune(event.Message, '\x00') {
		t.Fatalf("expected control characters removed, got %q", event.Message)
	}
	if len([]rune(event.Message)) > MaxMessageRunes {
		t.Fatalf("expected bounded message length <= %d, got %d", MaxMessageRunes, len([]rune(event.Message)))
	}
}

func TestNormalizeEventSanitizesBoundedRedactedReason(t *testing.T) {
	occurredAt := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rawReason := strings.Repeat(" stalled", 80) + " Authorization: Bearer sk-proj-1234567890abcdef1234567890abcdef1234567890 "

	event, err := NormalizeEvent(AttentionEvent{
		ID:         "event-reason",
		ProjectID:  "project-1",
		State:      StateWaitingInput,
		Reason:     rawReason,
		Source:     SourceStructuredProviderEvent,
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("NormalizeEvent returned error: %v", err)
	}

	if strings.Contains(event.Reason, "sk-proj-1234567890abcdef1234567890abcdef1234567890") {
		t.Fatalf("expected reason redaction, got %q", event.Reason)
	}
	if len([]rune(event.Reason)) > MaxMessageRunes {
		t.Fatalf("expected bounded reason length <= %d, got %d", MaxMessageRunes, len([]rune(event.Reason)))
	}
}

func TestNormalizeEventRejectsInvalidStateAndSource(t *testing.T) {
	occurredAt := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	_, err := NormalizeEvent(AttentionEvent{
		ID:         "event-1",
		ProjectID:  "project-1",
		State:      State("BROKEN"),
		Reason:     "bad state",
		Source:     SourceSystem,
		OccurredAt: occurredAt,
	})
	if err == nil || !strings.Contains(err.Error(), "state") {
		t.Fatalf("expected invalid state error, got %v", err)
	}

	_, err = NormalizeEvent(AttentionEvent{
		ID:         "event-2",
		ProjectID:  "project-1",
		State:      StateWorking,
		Reason:     "bad source",
		Source:     Source("BROKEN"),
		OccurredAt: occurredAt,
	})
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("expected invalid source error, got %v", err)
	}
}

func TestMemoryStreamIngestIsIdempotentAndReplayIsBounded(t *testing.T) {
	stream := NewMemoryStream(2)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	first, inserted, err := stream.Ingest(AttentionEvent{
		ID:         "event-1",
		ProjectID:  "project-1",
		RuntimeID:  "runtime-1",
		State:      StateWorking,
		Reason:     "building",
		Source:     SourceTerminalHeuristic,
		OccurredAt: base,
	})
	if err != nil || !inserted {
		t.Fatalf("expected first event inserted, got inserted=%v err=%v", inserted, err)
	}

	duplicate, inserted, err := stream.Ingest(first)
	if err != nil || inserted {
		t.Fatalf("expected duplicate event to be ignored, got inserted=%v err=%v", inserted, err)
	}
	if duplicate.ID != first.ID {
		t.Fatalf("expected duplicate to return stored event, got %#v", duplicate)
	}

	_, _, err = stream.Ingest(AttentionEvent{
		ID:         "event-2",
		ProjectID:  "project-1",
		RuntimeID:  "runtime-2",
		State:      StateRecoverable,
		Reason:     "runtime lost",
		Source:     SourceProcessExit,
		OccurredAt: base.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("expected second event inserted: %v", err)
	}

	_, _, err = stream.Ingest(AttentionEvent{
		ID:         "event-3",
		ProjectID:  "project-2",
		RuntimeID:  "runtime-3",
		State:      StateFailed,
		Reason:     "crashed",
		Source:     SourceSystem,
		OccurredAt: base.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("expected third event inserted: %v", err)
	}

	events := stream.Replay(ReplayFilter{Limit: 10})
	if len(events) != 2 {
		t.Fatalf("expected bounded replay of 2 events, got %d", len(events))
	}
	if events[0].ID != "event-2" || events[1].ID != "event-3" {
		t.Fatalf("expected oldest retained first, got %#v", events)
	}
}

func TestMemoryStreamRetainsSameEventIDAcrossProjects(t *testing.T) {
	stream := NewMemoryStream(8)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	first := mustIngest(t, stream, AttentionEvent{
		ID:         "event-1",
		ProjectID:  "project-1",
		RuntimeID:  "runtime-1",
		State:      StateWorking,
		Reason:     "building",
		Source:     SourceFlowRunner,
		OccurredAt: base,
	})
	second, inserted, err := stream.Ingest(AttentionEvent{
		ID:         "event-1",
		ProjectID:  "project-2",
		RuntimeID:  "runtime-2",
		State:      StateFailed,
		Reason:     "failed",
		Source:     SourceSystem,
		OccurredAt: base.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("expected second project event inserted: %v", err)
	}
	if !inserted {
		t.Fatalf("expected same event id in another project to be retained, got %#v", second)
	}

	projectOne := stream.Replay(ReplayFilter{ProjectID: "project-1", Limit: 8})
	if len(projectOne) != 1 || projectOne[0].ProjectID != first.ProjectID {
		t.Fatalf("expected independent project-1 event, got %#v", projectOne)
	}
	projectTwo := stream.Replay(ReplayFilter{ProjectID: "project-2", Limit: 8})
	if len(projectTwo) != 1 || projectTwo[0].ProjectID != "project-2" {
		t.Fatalf("expected independent project-2 event, got %#v", projectTwo)
	}
}

func TestProjectSummaryCountsLatestStatePerLogicalSubject(t *testing.T) {
	stream := NewMemoryStream(16)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	mustIngest(t, stream, AttentionEvent{
		ID:         "event-1",
		ProjectID:  "project-1",
		AgentID:    "agent-1",
		State:      StateWorking,
		Reason:     "building",
		Source:     SourceFlowRunner,
		OccurredAt: base,
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-2",
		ProjectID:  "project-1",
		AgentID:    "agent-1",
		State:      StateFailed,
		Reason:     "tests failed",
		Source:     SourceFlowRunner,
		OccurredAt: base.Add(time.Second),
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-3",
		ProjectID:  "project-1",
		AgentID:    "agent-2",
		State:      StateWaitingApproval,
		Reason:     "approval required",
		Source:     SourceStructuredProviderEvent,
		OccurredAt: base.Add(2 * time.Second),
	})
	mustIngest(t, stream, AttentionEvent{
		ID:             "event-4",
		ProjectID:      "project-1",
		ViewLogicalKey: "view:build",
		State:          StateCompleted,
		Reason:         "build complete",
		Source:         SourceSystem,
		OccurredAt:     base.Add(3 * time.Second),
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-5",
		ProjectID:  "project-1",
		RuntimeID:  "runtime-3",
		State:      StateRecoverable,
		Reason:     "runtime exited",
		Source:     SourceProcessExit,
		OccurredAt: base.Add(4 * time.Second),
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-6",
		ProjectID:  "project-1",
		RuntimeID:  "runtime-4",
		State:      StateRateLimited,
		Reason:     "provider 429",
		Source:     SourceProviderHook,
		OccurredAt: base.Add(5 * time.Second),
	})

	summary := stream.ProjectSummary("project-1")
	if summary.Total != 5 {
		t.Fatalf("expected 5 active subjects, got %#v", summary)
	}
	if summary.Working != 0 || summary.Failed != 1 || summary.WaitingApproval != 1 || summary.Completed != 1 || summary.CompletedUnread != 1 || summary.Recoverable != 1 || summary.RateLimited != 1 {
		t.Fatalf("unexpected counters: %#v", summary)
	}
	if summary.PriorityState != StateFailed {
		t.Fatalf("expected failed priority, got %#v", summary)
	}
	if summary.PriorityReason != "tests failed" {
		t.Fatalf("expected failed reason to win priority, got %#v", summary)
	}
}

func TestAcknowledgeCompletedEventDropsUnreadCompletedPriority(t *testing.T) {
	stream := NewMemoryStream(16)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	mustIngest(t, stream, AttentionEvent{
		ID:             "event-1",
		ProjectID:      "project-1",
		ViewLogicalKey: "view:task",
		State:          StateCompleted,
		Reason:         "done",
		Source:         SourceSystem,
		OccurredAt:     base,
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-2",
		ProjectID:  "project-1",
		RuntimeID:  "runtime-1",
		State:      StateRecoverable,
		Reason:     "reattach needed",
		Source:     SourceProcessExit,
		OccurredAt: base.Add(time.Second),
	})

	before := stream.ProjectSummary("project-1")
	if before.PriorityState != StateCompleted || before.CompletedUnread != 1 {
		t.Fatalf("expected unread completed priority before ack, got %#v", before)
	}

	ackAt := base.Add(2 * time.Second)
	acknowledged, changed := stream.Acknowledge("project-1", "event-1", ackAt)
	if !changed {
		t.Fatal("expected acknowledge to change event state")
	}
	if acknowledged.AcknowledgedAt == nil || !acknowledged.AcknowledgedAt.Equal(ackAt) {
		t.Fatalf("expected acknowledged timestamp, got %#v", acknowledged)
	}

	after := stream.ProjectSummary("project-1")
	if after.CompletedUnread != 0 {
		t.Fatalf("expected completed unread to drop after ack, got %#v", after)
	}
	if after.PriorityState != StateRecoverable {
		t.Fatalf("expected recoverable priority after ack, got %#v", after)
	}
}

func TestAcknowledgeTargetsMatchingProjectAndEventID(t *testing.T) {
	stream := NewMemoryStream(16)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	mustIngest(t, stream, AttentionEvent{
		ID:             "event-shared",
		ProjectID:      "project-1",
		ViewLogicalKey: "view:project-1",
		State:          StateCompleted,
		Reason:         "done project 1",
		Source:         SourceSystem,
		OccurredAt:     base,
	})
	mustIngest(t, stream, AttentionEvent{
		ID:             "event-shared",
		ProjectID:      "project-2",
		ViewLogicalKey: "view:project-2",
		State:          StateCompleted,
		Reason:         "done project 2",
		Source:         SourceSystem,
		OccurredAt:     base.Add(time.Second),
	})

	ackAt := base.Add(2 * time.Second)
	acknowledged, changed := stream.Acknowledge("project-1", "event-shared", ackAt)
	if !changed {
		t.Fatal("expected acknowledge to update the matching project event")
	}
	if acknowledged.ProjectID != "project-1" {
		t.Fatalf("expected acknowledged event from project-1, got %#v", acknowledged)
	}
	if acknowledged.AcknowledgedAt == nil || !acknowledged.AcknowledgedAt.Equal(ackAt) {
		t.Fatalf("expected acknowledged timestamp on project-1 event, got %#v", acknowledged)
	}

	projectOne := stream.Replay(ReplayFilter{ProjectID: "project-1", Limit: 8})
	if len(projectOne) != 1 || projectOne[0].AcknowledgedAt == nil || !projectOne[0].AcknowledgedAt.Equal(ackAt) {
		t.Fatalf("expected project-1 event acknowledged, got %#v", projectOne)
	}

	projectTwo := stream.Replay(ReplayFilter{ProjectID: "project-2", Limit: 8})
	if len(projectTwo) != 1 {
		t.Fatalf("expected one project-2 event, got %#v", projectTwo)
	}
	if projectTwo[0].AcknowledgedAt != nil {
		t.Fatalf("expected project-2 event to stay unacknowledged, got %#v", projectTwo[0])
	}
}

func TestReplayFiltersByProjectAndLimit(t *testing.T) {
	stream := NewMemoryStream(16)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	mustIngest(t, stream, AttentionEvent{
		ID:         "event-1",
		ProjectID:  "project-1",
		AgentID:    "agent-1",
		State:      StateWorking,
		Reason:     "building",
		Source:     SourceFlowRunner,
		OccurredAt: base,
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-2",
		ProjectID:  "project-2",
		AgentID:    "agent-2",
		State:      StateFailed,
		Reason:     "failed",
		Source:     SourceSystem,
		OccurredAt: base.Add(time.Second),
	})
	mustIngest(t, stream, AttentionEvent{
		ID:         "event-3",
		ProjectID:  "project-1",
		AgentID:    "agent-3",
		State:      StateWaitingInput,
		Reason:     "input",
		Source:     SourceTerminalHeuristic,
		OccurredAt: base.Add(2 * time.Second),
	})

	events := stream.Replay(ReplayFilter{
		ProjectID: "project-1",
		Limit:     1,
	})
	if len(events) != 1 {
		t.Fatalf("expected one filtered event, got %d", len(events))
	}
	if events[0].ID != "event-3" {
		t.Fatalf("expected newest matching event, got %#v", events)
	}
}

func TestMemoryStreamOutputsDoNotExposeStoredAcknowledgedAtPointers(t *testing.T) {
	stream := NewMemoryStream(16)
	base := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	ingestedAck := base.Add(10 * time.Second)

	stored, inserted, err := stream.Ingest(AttentionEvent{
		ID:             "event-ack",
		ProjectID:      "project-1",
		ViewLogicalKey: "view:ack",
		State:          StateCompleted,
		Reason:         "done",
		Source:         SourceSystem,
		OccurredAt:     base,
		AcknowledgedAt: &ingestedAck,
	})
	if err != nil || !inserted {
		t.Fatalf("expected acknowledged event inserted, got inserted=%v err=%v", inserted, err)
	}

	mutatedIngestAck := base.Add(20 * time.Second)
	*stored.AcknowledgedAt = mutatedIngestAck
	replayed := stream.Replay(ReplayFilter{ProjectID: "project-1", Limit: 8})
	if len(replayed) != 1 {
		t.Fatalf("expected one replayed event, got %#v", replayed)
	}
	if replayed[0].AcknowledgedAt == nil || !replayed[0].AcknowledgedAt.Equal(ingestedAck) {
		t.Fatalf("expected stored acknowledgedAt isolated from ingest output mutation, got %#v", replayed[0])
	}

	acknowledged, changed := stream.Acknowledge("project-1", "event-live-ack", base.Add(31*time.Second))
	if changed {
		t.Fatalf("expected missing event not to acknowledge, got %#v", acknowledged)
	}

	mustIngest(t, stream, AttentionEvent{
		ID:             "event-live-ack",
		ProjectID:      "project-1",
		ViewLogicalKey: "view:live-ack",
		State:          StateCompleted,
		Reason:         "complete",
		Source:         SourceSystem,
		OccurredAt:     base.Add(30 * time.Second),
	})

	acknowledged, changed = stream.Acknowledge("project-1", "event-live-ack", base.Add(31*time.Second))
	if !changed || acknowledged.AcknowledgedAt == nil {
		t.Fatalf("expected acknowledge to set timestamp, got changed=%v event=%#v", changed, acknowledged)
	}

	mutatedReturnedAck := base.Add(40 * time.Second)
	*acknowledged.AcknowledgedAt = mutatedReturnedAck
	replayed = stream.Replay(ReplayFilter{ProjectID: "project-1", ViewLogicalKey: "view:live-ack", Limit: 1})
	if len(replayed) != 1 {
		t.Fatalf("expected one acknowledged replayed event, got %#v", replayed)
	}
	if replayed[0].AcknowledgedAt == nil || !replayed[0].AcknowledgedAt.Equal(base.Add(31*time.Second)) {
		t.Fatalf("expected stored acknowledgedAt isolated from acknowledge output mutation, got %#v", replayed[0])
	}
}

func mustIngest(t *testing.T, stream *MemoryStream, event AttentionEvent) AttentionEvent {
	t.Helper()
	stored, inserted, err := stream.Ingest(event)
	if err != nil {
		t.Fatalf("Ingest(%q) error: %v", event.ID, err)
	}
	if !inserted {
		t.Fatalf("expected %q inserted", event.ID)
	}
	return stored
}
