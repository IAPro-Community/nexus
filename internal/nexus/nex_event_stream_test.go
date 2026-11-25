package nexus

import (
	"testing"
	"time"
)

func TestNexEventStreamCursorDedupeAndNotificationPolicy(t *testing.T) {
	stream := NewNexEventStream(3)
	now := time.Date(2026, 11, 25, 12, 30, 0, 0, time.UTC)

	state := HUDState{
		Alerts: []HUDAlert{
			{Fingerprint: "fp-1", Severity: "WARNING", Provider: "claude", Reason: "Quota < 20%"},
			{Fingerprint: "fp-acked", Severity: "CRITICAL", Acked: true},
		},
		RuntimeAttention: []HUDRuntimeAttention{
			{RuntimeID: "rt-1", Provider: "codex", Reason: "Approval needed", Fingerprint: "att-1"},
		},
	}
	changes := []ModelCatalogChange{
		{Fingerprint: "mc-1", Kind: ModelChangeAdded, Provider: "claude", ModelID: "claude-sonnet-4"},
	}

	first := stream.IngestHUDState(state, changes, now)
	if first != 3 {
		t.Fatalf("expected 3 events recorded (acked alert excluded), got %d", first)
	}
	// Re-ingesting identical HUD state must produce 0 duplicates.
	second := stream.IngestHUDState(state, changes, now.Add(time.Second))
	if second != 0 {
		t.Fatalf("expected 0 duplicate events on repeat ingest, got %d", second)
	}

	page0 := stream.Since(0)
	if page0.ReplayTruncated || len(page0.Events) != 3 || page0.Revision != 3 {
		t.Fatalf("unexpected page0: %+v", page0)
	}

	// Exceed ring capacity (3) so earliest retained revision becomes 3 (> cursor 1 + 1).
	stream.Record(NexEventEnvelope{Fingerprint: "ev-4", Kind: "MODEL_CATALOG_ADDED"})
	if stillContiguous := stream.Since(1); stillContiguous.ReplayTruncated || len(stillContiguous.Events) != 3 {
		t.Fatalf("expected non-truncated contiguous tail from cursor=1, got %+v", stillContiguous)
	}
	stream.Record(NexEventEnvelope{Fingerprint: "ev-5", Kind: "MODEL_CATALOG_ADDED"})
	truncatedPage := stream.Since(1)
	if !truncatedPage.ReplayTruncated {
		t.Fatalf("expected ReplayTruncated when cursor=1 and earliest retained revision is 3: %+v", truncatedPage)
	}

	// Notification policy suppresses DND/quiet hours and caps bursts.
	if got := FilterForNotification(page0.Events, NexNotificationPolicy{DoNotDisturb: true}); len(got) != 0 {
		t.Fatalf("DND must suppress notifications, got %d", len(got))
	}
	if got := FilterForNotification(page0.Events, NexNotificationPolicy{QuietHoursActive: true}); len(got) != 0 {
		t.Fatalf("Quiet hours must suppress notifications, got %d", len(got))
	}
	batched := FilterForNotification(append(page0.Events, page0.Events[0]), NexNotificationPolicy{MaxBatch: 2})
	if len(batched) != 2 {
		t.Fatalf("expected deduplicated burst capped to 2, got %d", len(batched))
	}
}
