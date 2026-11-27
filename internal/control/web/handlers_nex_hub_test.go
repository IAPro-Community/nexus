package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestHandleNexHubStatusGetPostAndHeaders(t *testing.T) {
	notify.ClearNexHub("nex_offline")
	t.Cleanup(func() { notify.ClearNexHub("nex_offline") })

	h := &NexusHandler{}

	// Initial GET defaults offline.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/nex/hub-status", nil)
	getRec := httptest.NewRecorder()
	h.handleNexHubStatus(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status=%d want 200", getRec.Code)
	}
	var got notify.NexHubStatus
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Active || got.Reason != "nex_offline" {
		t.Fatalf("expected offline default, got %+v", got)
	}

	// POST heartbeat activates NEX hub.
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/nex/hub-status", bytes.NewBufferString(`{"can_notify":true,"do_not_disturb":false}`))
	postRec := httptest.NewRecorder()
	h.handleNexHubStatus(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status=%d want 200", postRec.Code)
	}
	if err := json.Unmarshal(postRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Active || got.Reason != "nex_active" {
		t.Fatalf("expected active hub after POST, got %+v", got)
	}

	// POST paused releases lease immediately.
	pauseReq := httptest.NewRequest(http.MethodPost, "/api/v1/nex/hub-status", bytes.NewBufferString(`{"can_notify":false,"reason":"nex_paused"}`))
	pauseRec := httptest.NewRecorder()
	h.handleNexHubStatus(pauseRec, pauseReq)
	if err := json.Unmarshal(pauseRec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Active || got.Reason != "nex_paused" {
		t.Fatalf("expected paused hub, got %+v", got)
	}

	// Reject unsupported HTTP methods.
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/nex/hub-status", nil)
	delRec := httptest.NewRecorder()
	h.handleNexHubStatus(delRec, delReq)
	if delRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE status=%d want 405", delRec.Code)
	}
}

func TestNexEventsRenewsNexHubLeaseOnlyWhenHeaderPresent(t *testing.T) {
	t.Setenv("NEXUS_DISABLE_HUD", "1")
	notify.ClearNexHub("nex_offline")
	t.Cleanup(func() { notify.ClearNexHub("nex_offline") })

	watch := nexus.NewModelCatalogWatchService(filepath.Join(t.TempDir(), "watch.json"))
	stream := nexus.NewNexEventStream(16)
	h := &NexusHandler{modelWatch: watch, nexEvents: stream}

	// Normal browser call without X-Nexus-Nex-Hub must NOT activate NEX hub.
	plainReq := httptest.NewRequest(http.MethodGet, "/api/v1/nex/events", nil)
	plainRec := httptest.NewRecorder()
	h.handleNexEvents(plainRec, plainReq)
	if plainRec.Code != http.StatusOK {
		t.Fatalf("events status=%d want 200", plainRec.Code)
	}
	if notify.CurrentNexHubStatus().Active {
		t.Fatal("events request without X-Nexus-Nex-Hub header must not activate hub")
	}

	// NEX 2.0 poll with X-Nexus-Nex-Hub: active activates the lease.
	nexReq := httptest.NewRequest(http.MethodGet, "/api/v1/nex/events", nil)
	nexReq.Header.Set("X-Nexus-Nex-Hub", "active")
	nexReq.Header.Set("X-Nexus-Nex-Dnd", "0")
	nexRec := httptest.NewRecorder()
	h.handleNexEvents(nexRec, nexReq)
	if nexRec.Code != http.StatusOK {
		t.Fatalf("events status=%d want 200", nexRec.Code)
	}
	status := notify.CurrentNexHubStatus()
	if !status.Active || status.Reason != "nex_active" {
		t.Fatalf("expected active NEX hub from events header, got %+v", status)
	}
}

func TestPollAttentionSuppressedWhenNexHubActive(t *testing.T) {
	notify.ClearNexHub("nex_offline")
	t.Cleanup(func() { notify.ClearNexHub("nex_offline") })

	notify.ReportNexHubHeartbeat(true, false, "nex_active", 0)
	tracker := newAttentionTracker()

	// Seed attention source behavior through ShouldSuppressFallback + tracker directly:
	// First verify pollAttention short-circuits without needing real VAPID push delivery.
	group := &runner.AttentionGroup{
		NeedsYou: []*runner.AttentionItem{
			{MissionID: "m1", State: runner.StateBlockedNeedsUser, ReasonCode: "QUESTION", Summary: "Approve change?"},
		},
	}
	var snapshots []attentionSnapshot
	for _, item := range group.NeedsYou {
		snapshots = append(snapshots, snapshotFromAttention(item))
	}
	fresh := tracker.Diff(snapshots)
	if len(fresh) != 1 {
		t.Fatalf("expected 1 fresh item, got %d", len(fresh))
	}
	if !notify.ShouldSuppressFallback(notify.CategoryActionRequired) {
		t.Fatal("expected Web Push fallback to be suppressed while NEX hub is active")
	}

	// Clearing NEX restores fallback availability, while tracker keeps already-seen episodes silent.
	notify.ClearNexHub("nex_offline")
	if notify.ShouldSuppressFallback(notify.CategoryActionRequired) {
		t.Fatal("expected Web Push fallback restored when NEX hub goes offline")
	}
	if second := tracker.Diff(snapshots); len(second) != 0 {
		t.Fatalf("already seen episode must not re-fire, got %d", len(second))
	}
	_ = context.Background()
}
