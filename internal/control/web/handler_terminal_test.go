package web

import (
	"testing"

	"github.com/kivervinicius/ai-cli/internal/control/events"
)

func TestAttentionFrameFromEventChainsRoutingMetadataWithoutSecretFields(t *testing.T) {
	frame := attentionFrameFromEvent("agent-1", events.Event{
		ID:        "evt-7",
		RuntimeID: "rt-1",
		Summary:   "Approval pending",
		Data: map[string]any{
			"attention_reason": "QUESTION",
			"context":          "Need approval",
			"project_name":     "Payments",
			"dynamic_title":    "Bearer sk-live-secret",
			"project_id":       "prj-1",
			"view":             "work",
			"transcript":       "secret transcript",
			"credentials":      map[string]any{"token": "sk-live-secret"},
		},
	})

	if got := frame["attention_event_id"]; got != "evt-7" {
		t.Fatalf("expected attention event id, got %#v", got)
	}
	if got := frame["project_id"]; got != "prj-1" {
		t.Fatalf("expected project id, got %#v", got)
	}
	if got := frame["view"]; got != "work" {
		t.Fatalf("expected view, got %#v", got)
	}
	if _, ok := frame["transcript"]; ok {
		t.Fatalf("frame must not forward transcript: %#v", frame)
	}
	if _, ok := frame["credentials"]; ok {
		t.Fatalf("frame must not forward credentials: %#v", frame)
	}
}

func TestAttentionFrameFromEventBestEffortProjectRouting(t *testing.T) {
	previous := resolveProjectIDForAgent
	resolveProjectIDForAgent = func(agentID string) string {
		if agentID != "agent-42" {
			t.Fatalf("unexpected agent lookup: %s", agentID)
		}
		return "prj-derived"
	}
	t.Cleanup(func() { resolveProjectIDForAgent = previous })

	frame := attentionFrameFromEvent("agent-42", events.Event{
		ID:        "evt-42",
		RuntimeID: "rt-42",
		Data: map[string]any{
			"attention_reason": "QUESTION",
			"context":          "Need approval",
		},
	})

	if got := frame["project_id"]; got != "prj-derived" {
		t.Fatalf("expected best-effort project id, got %#v", got)
	}
	if got := frame["view"]; got != "work" {
		t.Fatalf("expected best-effort work view, got %#v", got)
	}
}
