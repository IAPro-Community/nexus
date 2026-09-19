package nexus

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func testCapacityNexus(t *testing.T) *Nexus {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Nexus{st: st}
}

// Disabled by default: BackgroundEnabled=false means no loop is hosted.
func TestStartCapacityMonitorDisabledByDefault(t *testing.T) {
	n := testCapacityNexus(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n.StartCapacityMonitor(ctx)
	if n.CapacityMonitorRunning() {
		t.Fatal("loop must not start with background disabled (consent-gated)")
	}
	n.StopCapacityMonitor() // safe when never started
}

// Opt-in starts exactly one loop; double start is idempotent; stop releases.
func TestStartCapacityMonitorOptInIdempotent(t *testing.T) {
	n := testCapacityNexus(t)
	prefs := store.DefaultCapacityPreferences()
	prefs.BackgroundEnabled = true
	if err := n.st.PutCapacityPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n.StartCapacityMonitor(ctx)
	if !n.CapacityMonitorRunning() {
		t.Fatal("loop must start after explicit background opt-in")
	}
	n.StartCapacityMonitor(ctx) // idempotent: still exactly one
	if !n.CapacityMonitorRunning() {
		t.Fatal("loop must survive double start")
	}
	n.StopCapacityMonitor()
	if n.CapacityMonitorRunning() {
		t.Fatal("stop must release the hosted loop")
	}
	// Restart-safe: starting again after stop works.
	n.StartCapacityMonitor(ctx)
	if !n.CapacityMonitorRunning() {
		t.Fatal("restart after stop must work")
	}
	n.StopCapacityMonitor()
}

// Enabled but not background: still no loop.
func TestStartCapacityMonitorEnabledWithoutBackground(t *testing.T) {
	n := testCapacityNexus(t)
	prefs := store.DefaultCapacityPreferences()
	prefs.BackgroundEnabled = false
	if err := n.st.PutCapacityPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n.StartCapacityMonitor(ctx)
	if n.CapacityMonitorRunning() {
		t.Fatal("enabled alone must not start the loop without background opt-in")
	}
}
