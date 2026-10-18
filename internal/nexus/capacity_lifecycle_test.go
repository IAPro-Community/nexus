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

func TestCapacityStatusJoinsCanonicalKeyAndProtectsAccountScope(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "capacity-projection.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UTC().Truncate(time.Minute)
	reset := now.Add(2 * time.Hour)
	for _, tc := range []struct {
		profile    string
		stateScope string
		obsScope   string
		remaining  float64
	}{
		{"account-a", "verified/a", "verified/a", 30},
		{"account-b", "verified/b", "verified/c", 12},
	} {
		used := 100 - tc.remaining
		if err := st.UpsertCapacityObservation(store.ObservationRecord{
			Provider: "codex", Profile: tc.profile, QuotaGroup: "default",
			WindowKind: "5h", Status: "LIVE", Source: "OFFICIAL_API",
			UsedPercent: &used, RemainingPercent: &tc.remaining, FetchedAt: now,
			ExpiresAt: reset, ResetDescription: "2 hours", AccountScope: tc.obsScope,
		}); err != nil {
			t.Fatal(err)
		}
		// Reproduce the persisted legacy state with blank metadata but a
		// canonical pipe-delimited key and an identity-specific scope.
		if err := st.UpsertCapacityState(store.CapacityStateRecord{
			Key: "codex|" + tc.profile + "|default|5h", Provider: "codex",
			Profile: tc.profile, State: "WARNING",
			TransitionedAt: now, UpdatedAt: now, AccountScope: tc.stateScope,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i, percent := range []float64{20, 40, 70} {
		if err := st.RecordCapacityPaceSample(store.CapacityPaceSample{
			Provider: "codex", Profile: "account-a", AccountScope: "verified/a",
			QuotaGroup: "default", WindowKind: "5h", ResetAt: reset,
			ObservedAt: now.Add(-time.Duration(20-i*10) * time.Minute),
			UsedPercent: percent, Source: "OFFICIAL_API",
		}); err != nil {
			t.Fatal(err)
		}
	}
	status, err := NewCapacityApplicationService(st).Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Resources) != 2 {
		t.Fatalf("resources=%d, expected two separate accounts", len(status.Resources))
	}
	for _, resource := range status.Resources {
		if resource.QuotaGroup != "default" || resource.WindowKind != "5h" {
			t.Fatalf("lost group/window metadata for %s: %+v", resource.Profile, resource)
		}
		switch resource.Profile {
		case "account-a":
			if resource.RemainingPercent == nil || *resource.RemainingPercent != 30 ||
				resource.Pace == nil || resource.Pace.Status != "RISING" ||
				resource.ObservationStatus != "LIVE" {
				t.Fatalf("matching observation/forecast not projected: %+v", resource)
			}
		case "account-b":
			if resource.RemainingPercent != nil || resource.Pace != nil ||
				resource.ObservationStatus != "" || resource.Source != "" {
				t.Fatalf("different identity leaked capacity evidence: %+v", resource)
			}
		default:
			t.Fatalf("unexpected account: %+v", resource)
		}
	}
}
