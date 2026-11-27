package notify

import (
	"testing"
	"time"
)

func resetNexHubTestState(t *testing.T, clock func() time.Time) {
	t.Helper()
	prevNow := nexHubNow
	if clock != nil {
		nexHubNow = clock
	}
	nexHubMu.Lock()
	nexHubState = nexHubLeaseState{}
	nexHubMu.Unlock()
	t.Cleanup(func() {
		nexHubMu.Lock()
		nexHubState = nexHubLeaseState{}
		nexHubMu.Unlock()
		nexHubNow = prevNow
	})
}

func TestNexHubDefaultsOfflineSoFallbackWorks(t *testing.T) {
	resetNexHubTestState(t, nil)

	status := CurrentNexHubStatus()
	if status.Active || status.CanNotify || status.Reason != "nex_offline" {
		t.Fatalf("expected default offline hub status, got %+v", status)
	}
	if ShouldSuppressFallback(CategoryActionRequired) {
		t.Fatal("fallback must stay active by default when NEX has not reported a heartbeat")
	}

	rec := &Recorder{}
	n := NewThrottled(rec, time.Minute)
	if err := n.Notify(Payload{
		Title:      "Nexus",
		Body:       "Fallback notification",
		Category:   CategoryActionRequired,
		EpisodeKey: "ep-offline",
	}); err != nil {
		t.Fatal(err)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected 1 delivered fallback notification, got %d", len(rec.Payloads))
	}
}

func TestNexHubSuppressesFallbackWhenActiveAndRespectsForceFallback(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	resetNexHubTestState(t, func() time.Time { return now })

	status := ReportNexHubHeartbeat(true, false, "nex_active", 30*time.Second)
	if !status.Active || !status.CanNotify || status.Reason != "nex_active" {
		t.Fatalf("expected active hub status, got %+v", status)
	}

	rec := &Recorder{}
	n := NewThrottled(rec, time.Minute)
	if err := n.Notify(Payload{
		Title:      "Nexus",
		Body:       "Suppressed while NEX active",
		Category:   CategoryActionRequired,
		EpisodeKey: "ep-suppressed",
	}); err != nil {
		t.Fatal(err)
	}
	if len(rec.Payloads) != 0 {
		t.Fatalf("expected fallback notification to be suppressed while NEX is active, got %d", len(rec.Payloads))
	}

	// Explicit ForceFallback bypasses suppression even while NEX is active.
	if err := n.Notify(Payload{
		Title:         "Nexus",
		Body:          "Emergency fallback",
		Category:      CategoryCriticalFailure,
		EpisodeKey:    "ep-forced",
		ForceFallback: true,
	}); err != nil {
		t.Fatal(err)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected ForceFallback payload to deliver, got %d", len(rec.Payloads))
	}

	// Because "ep-suppressed" was suppressed cleanly, clearing NEX lets later attempts notify if retried.
	ClearNexHub("nex_offline")
	if ShouldSuppressFallback(CategoryActionRequired) {
		t.Fatal("ClearNexHub must restore fallback immediately")
	}
	if err := n.Notify(Payload{
		Title:      "Nexus",
		Body:       "Delivered after NEX went offline",
		Category:   CategoryActionRequired,
		EpisodeKey: "ep-suppressed",
	}); err != nil {
		t.Fatal(err)
	}
	if len(rec.Payloads) != 2 {
		t.Fatalf("expected fallback delivery after ClearNexHub, got %d", len(rec.Payloads))
	}
}

func TestNexHubDNDKeepsFallbackSuppressedPauseRestoresFallbackAndTTLRestoresFallback(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	resetNexHubTestState(t, func() time.Time { return now })

	// User enabled NEX Do Not Disturb: NEX owns notification policy so OS fallback remains quiet.
	dnd := ReportNexHubHeartbeat(true, true, "", 20*time.Second)
	if !dnd.Active || !dnd.DoNotDisturb || dnd.Reason != "nex_dnd" {
		t.Fatalf("expected nex_dnd active lease, got %+v", dnd)
	}
	if !ShouldSuppressFallback(CategoryQuotaExhausted) {
		t.Fatal("NEX DND should keep fallback popups quiet")
	}

	// Paused or hidden NEX restores fallback immediately.
	paused := ReportNexHubHeartbeat(false, false, "nex_paused", 0)
	if paused.Active || paused.Reason != "nex_paused" {
		t.Fatalf("expected paused hub status, got %+v", paused)
	}
	if ShouldSuppressFallback(CategoryActionRequired) {
		t.Fatal("paused NEX must restore fallback")
	}

	hidden := ReportNexHubHeartbeat(false, false, "nex_hidden", 0)
	if hidden.Active || hidden.Reason != "nex_hidden" {
		t.Fatalf("expected hidden hub status, got %+v", hidden)
	}

	// Renew lease and advance clock past TTL to verify fail-safe expiration.
	ReportNexHubHeartbeat(true, false, "", 15*time.Second)
	if !ShouldSuppressFallback(CategoryActionRequired) {
		t.Fatal("expected active suppression immediately after lease renewal")
	}
	now = now.Add(16 * time.Second)
	expired := CurrentNexHubStatus()
	if expired.Active || expired.Reason != "lease_expired" {
		t.Fatalf("expected lease_expired after TTL elapses, got %+v", expired)
	}
	if ShouldSuppressFallback(CategoryActionRequired) {
		t.Fatal("expired lease must restore fallback")
	}
}
