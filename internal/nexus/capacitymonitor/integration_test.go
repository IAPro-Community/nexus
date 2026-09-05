package capacitymonitor

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestCapacityMonitor_FullLifecycleIntegration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_lifecycle.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	defer st.Close()

	clock := newFakeClock(time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC))

	var probeVal atomic.Int64
	probeVal.Store(80) // start healthy at 80% remaining

	reg := &fakeRegistry{
		probes: map[string]quota.Probe{
			"codex": &fakeProbe{
				desc: quota.ProbeDescriptor{Granularity: model.UsageGranularityGroup, ReadOnly: true},
				fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
					pct := float64(probeVal.Load())
					return model.UsageSnapshot{
						ProviderID: p.Provider,
						ProfileID:  p.Name,
						Status:     model.UsageLive,
						Source:     model.SourceLocalFiles,
						FetchedAt:  clock.Now(),
						Windows: []model.UsageWindow{
							{
								Kind:             "sliding_5h",
								Group:            "general",
								RemainingPercent: &pct,
							},
						},
					}, nil
				},
			},
		},
	}

	profiles := []model.Profile{
		{
			Provider: "codex",
			Name:     "default",
		},
	}

	deps := Dependencies{
		Profiles: func() ([]model.Profile, error) {
			return profiles, nil
		},
		ProbeRegistry:  reg,
		Store:          st,
		Clock:          clock,
		JitterSource:   &fakeJitter{},
		MaxConcurrency: 2,
		OwnerID:        "node-integration-1",
	}

	mon := New(deps)
	ctx := context.Background()

	// Pass 1: Healthy
	if err := mon.Tick(ctx); err != nil {
		t.Fatalf("pass 1 failed: %v", err)
	}

	states, err := st.ListCapacityStates()
	if err != nil {
		t.Fatalf("list states failed: %v", err)
	}
	if len(states) != 1 || states[0].State != "HEALTHY" {
		t.Fatalf("expected 1 HEALTHY state, got: %+v", states)
	}

	// Pass 2: Escalate to Warning (18% remaining)
	probeVal.Store(18)
	clock.Advance(1 * time.Minute)
	if err := mon.Tick(ctx); err != nil {
		t.Fatalf("pass 2 failed: %v", err)
	}

	states, _ = st.ListCapacityStates()
	if states[0].State != "WARNING" {
		t.Fatalf("expected WARNING state, got: %s", states[0].State)
	}

	events, err := st.ListCapacityEvents(10)
	if err != nil {
		t.Fatalf("list events failed: %v", err)
	}
	if len(events) == 0 || events[0].Severity != "WARNING" {
		t.Fatalf("expected WARNING event emitted, got: %+v", events)
	}

	// Pass 3: Escalate to Critical (8% remaining)
	probeVal.Store(8)
	clock.Advance(1 * time.Minute)
	if err := mon.Tick(ctx); err != nil {
		t.Fatalf("pass 3 failed: %v", err)
	}

	states, _ = st.ListCapacityStates()
	if states[0].State != "CRITICAL" {
		t.Fatalf("expected CRITICAL state, got: %s", states[0].State)
	}

	// Pass 4: Escalate to Exhausted (0% remaining)
	probeVal.Store(0)
	clock.Advance(1 * time.Minute)
	if err := mon.Tick(ctx); err != nil {
		t.Fatalf("pass 4 failed: %v", err)
	}

	states, _ = st.ListCapacityStates()
	if states[0].State != "EXHAUSTED" {
		t.Fatalf("expected EXHAUSTED state, got: %s", states[0].State)
	}

	// Pass 5: Recover (30% remaining > 25% recovery threshold)
	probeVal.Store(30)
	clock.Advance(1 * time.Minute)
	if err := mon.Tick(ctx); err != nil {
		t.Fatalf("pass 5 failed: %v", err)
	}

	states, _ = st.ListCapacityStates()
	if states[0].State != "RECOVERED" {
		t.Fatalf("expected RECOVERED state, got: %s", states[0].State)
	}
}
