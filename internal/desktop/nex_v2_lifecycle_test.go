package desktop

import (
	"testing"
	"time"
)

func TestResolveNexV2LifecycleAndRendererIsolation(t *testing.T) {
	standalone := ResolveNexV2Lifecycle(false, true)
	if standalone.Mode != NexV2LifecycleStandalone || !standalone.MainCloseKeepsCore || !standalone.RendererCrashSafe {
		t.Fatalf("unexpected standalone lifecycle: %+v", standalone)
	}

	linked := ResolveNexV2Lifecycle(true, false)
	if linked.Mode != NexV2LifecycleLinked || linked.MainCloseKeepsCore || !linked.RendererCrashSafe {
		t.Fatalf("unexpected linked lifecycle: %+v", linked)
	}

	if ShouldTerminateCoreOnRendererError() {
		t.Fatal("renderer failure must never terminate Core or user Agents")
	}
}

func TestRendererRecoverySupervisorBoundedBackoffAndReset(t *testing.T) {
	current := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sup := NewRendererRecoverySupervisor(func() time.Time { return current })

	wantDelays := []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}
	for i, want := range wantDelays {
		ok, delay := sup.RecordFailure()
		if !ok || delay != want {
			t.Fatalf("attempt %d: ok=%v delay=%v, want ok=true delay=%v", i+1, ok, delay, want)
		}
		if sup.StaticFallbackActive() {
			t.Fatalf("fallback should not be active before budget exhaustion (attempt %d)", i+1)
		}
	}

	// Fourth failure exceeds maxRestarts=3 and activates static fallback.
	ok, delay := sup.RecordFailure()
	if ok || delay != 0 || !sup.StaticFallbackActive() {
		t.Fatalf("expected static fallback after 3 restarts, got ok=%v delay=%v fallback=%v", ok, delay, sup.StaticFallbackActive())
	}

	// Advancing past the healthy window resets the restart budget.
	current = current.Add(6 * time.Minute)
	sup.RecordHealthy()
	if sup.StaticFallbackActive() || sup.RestartCount() != 0 {
		t.Fatalf("expected budget reset after healthy window, count=%d fallback=%v", sup.RestartCount(), sup.StaticFallbackActive())
	}
}
