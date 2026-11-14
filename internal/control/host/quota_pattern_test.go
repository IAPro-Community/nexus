package host

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

func newTestQuotaWatcher(t *testing.T, published *[]events.Event) *QuotaLimitWatcher {
	t.Helper()
	return NewQuotaLimitWatcher(
		"rt-quota",
		"codex",
		"work",
		"lineage-1",
		time.Now().Add(-time.Minute), // past the min-uptime guard
		func(e events.Event) {
			*published = append(*published, e)
		},
	)
}

func TestQuotaLimitWatcherPublishesOnLimitPattern(t *testing.T) {
	var published []events.Event
	w := newTestQuotaWatcher(t, &published)

	w.ProcessChunk([]byte("✗ You've hit your usage limit. Your limit will reset at 5pm.\r\n"))

	if len(published) != 1 {
		t.Fatalf("published %d events, want exactly 1", len(published))
	}
	e := published[0]
	if e.Type != events.EventQuotaExhausted {
		t.Fatalf("event type = %s", e.Type)
	}
	if e.Data["nexus.quota_trigger"] != "pty_pattern" {
		t.Fatalf("quota_trigger = %v", e.Data["nexus.quota_trigger"])
	}
	if e.Data["affected_runtime_id"] != "rt-quota" {
		t.Fatalf("affected_runtime_id = %v", e.Data["affected_runtime_id"])
	}
	if e.Data["failure_kind"] != string(model.FailureRateLimit) {
		t.Fatalf("failure_kind = %v", e.Data["failure_kind"])
	}
	if e.RuntimeID != "rt-quota" || e.Provider != "codex" || e.Profile != "work" {
		t.Fatalf("event ownership mismatch: %+v", e)
	}
}

func TestQuotaLimitWatcherPublishesBillingExhaustionAsQuotaKind(t *testing.T) {
	var published []events.Event
	w := newTestQuotaWatcher(t, &published)

	w.ProcessChunk([]byte("error: insufficient credits for this request\r\n"))

	if len(published) != 1 {
		t.Fatalf("published %d events, want 1", len(published))
	}
	if published[0].Data["failure_kind"] != string(model.FailureQuota) {
		t.Fatalf("failure_kind = %v", published[0].Data["failure_kind"])
	}
}

func TestQuotaLimitWatcherIgnoresBenignOutput(t *testing.T) {
	var published []events.Event
	w := newTestQuotaWatcher(t, &published)

	// Agent-authored prose about APIs must never look like a provider banner.
	w.ProcessChunk([]byte("The agent discussed http 429 semantics and rate limiting in the blog post draft.\r\n"))
	w.ProcessChunk([]byte("def retry_policy(): return 'exponential backoff' # 429 handling example\r\n"))
	w.ProcessChunk([]byte("normal TUI repaint bytes\r\n"))

	if len(published) != 0 {
		t.Fatalf("benign output must not publish; got %d events", len(published))
	}
}

func TestQuotaLimitWatcherFiresOnlyOncePerRuntime(t *testing.T) {
	var published []events.Event
	w := newTestQuotaWatcher(t, &published)

	w.ProcessChunk([]byte("Error: usage limit reached.\r\n"))
	w.ProcessChunk([]byte("Error: usage limit reached.\r\n"))
	w.ProcessChunk([]byte("usage limit reached again\r\n"))

	if len(published) != 1 {
		t.Fatalf("watcher must dedupe to 1 event per runtime; got %d", len(published))
	}
}

func TestQuotaLimitWatcherRespectsMinUptime(t *testing.T) {
	var published []events.Event
	w := NewQuotaLimitWatcher("rt-quota", "codex", "work", "lineage-1", time.Now(), func(e events.Event) {
		published = append(published, e)
	})

	// Resumed-session banners replay at startup; before min uptime they must
	// not trigger the evidence-check acceleration.
	w.ProcessChunk([]byte("usage limit reached (banner replay from resume)\r\n"))
	if len(published) != 0 {
		t.Fatalf("startup replay must not trigger; got %d events", len(published))
	}

	// Simulate uptime passing and a fresh pattern.
	w.startedAt = time.Now().Add(-2 * quotaPatternMinUptime)
	w.ProcessChunk([]byte("usage limit reached\r\n"))
	if len(published) != 1 {
		t.Fatalf("post-uptime pattern must trigger; got %d events", len(published))
	}
}

func TestQuotaLimitWatcherNilSafe(t *testing.T) {
	var w *QuotaLimitWatcher
	w.ProcessChunk([]byte("usage limit reached")) // must not panic
}

func TestSessionHostObserveOutputChunkKeepsDetectorBehavior(t *testing.T) {
	// The unified observation point must keep feeding the attention detector:
	// a nil watcher must not change detector behavior.
	sh := &SessionHost{detector: NewAttentionDetectorWithProject(
		"rt-x", "codex", "work", "/tmp", "", "",
		func(reason, context, dynamicTitle string, state registry.RuntimeState) {},
	)}
	sh.observeOutputChunk([]byte("✗ usage limit reached\n"))
	if sh.quotaWatcher != nil {
		t.Fatal("bare struct must not fabricate a watcher")
	}
}

func TestQuotaLimitWatcherPublishesAgyWeeklyQuotaBanner(t *testing.T) {
	published := &[]events.Event{}
	w := NewQuotaLimitWatcher(
		"rt-agy",
		"agy",
		"work",
		"lineage-2",
		time.Now().Add(-time.Minute), // past the min-uptime guard
		func(e events.Event) {
			*published = append(*published, e)
		},
	)

	w.ProcessChunk([]byte("⚠ Individual quota reached. Please upgrade your subscription to increase your limits.\r\n"))
	if len(*published) != 1 {
		t.Fatalf("published %d events, want 1", len(*published))
	}
	if (*published)[0].Data["failure_kind"] != string(model.FailureQuota) {
		t.Fatalf("failure_kind = %v", (*published)[0].Data["failure_kind"])
	}
}
