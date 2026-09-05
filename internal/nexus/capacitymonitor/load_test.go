package capacitymonitor

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestCapacityMonitor_LoadSimulation50Profiles30Minutes(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "load_sim.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer st.Close()

	clock := newFakeClock(time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC))

	var concurrentProbes atomic.Int64
	var maxObservedConcurrency atomic.Int64
	var totalProbes atomic.Int64
	rem := 40.0

	reg := &fakeRegistry{
		probes: map[string]quota.Probe{
			"codex": &fakeProbe{
				desc: quota.ProbeDescriptor{Granularity: model.UsageGranularityGroup, ReadOnly: true},
				fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
					totalProbes.Add(1)
					curr := concurrentProbes.Add(1)
					for {
						oldMax := maxObservedConcurrency.Load()
						if curr <= oldMax || maxObservedConcurrency.CompareAndSwap(oldMax, curr) {
							break
						}
					}
					// Simulate small execution latency
					time.Sleep(200 * time.Microsecond)
					concurrentProbes.Add(-1)

					return model.UsageSnapshot{
						ProviderID: p.Provider,
						ProfileID:  p.Name,
						Status:     model.UsageLive,
						Source:     model.SourceLocalFiles,
						FetchedAt:  clock.Now(),
						Windows: []model.UsageWindow{
							{
								Kind:             "sliding_5h",
								Group:            "requests",
								RemainingPercent: &rem,
							},
						},
					}, nil
				},
			},
		},
	}

	const profileCount = 50
	profiles := make([]model.Profile, profileCount)
	for i := 0; i < profileCount; i++ {
		profiles[i] = model.Profile{
			Provider: "codex",
			Name:     fmt.Sprintf("profile-%02d", i),
		}
	}

	initialGoroutines := runtime.NumGoroutine()

	deps := Dependencies{
		Profiles: func() ([]model.Profile, error) {
			return profiles, nil
		},
		ProbeRegistry:  reg,
		Store:          st,
		Clock:          clock,
		JitterSource:   &fakeJitter{},
		MaxConcurrency: 2,
		OwnerID:        "load-node-1",
	}

	mon := New(deps)
	ctx := context.Background()

	// Simulate 60 ticks (30 minutes of simulated time with 30s ticks)
	const simulatedTicks = 60
	for tick := 0; tick < simulatedTicks; tick++ {
		if err := mon.Tick(ctx); err != nil {
			t.Fatalf("tick %d failed: %v", tick, err)
		}
		clock.Advance(30 * time.Second)
	}

	// Assertions
	if maxObs := maxObservedConcurrency.Load(); maxObs > 2 {
		t.Fatalf("concurrency limit exceeded: expected <= 2, got %d", maxObs)
	}

	if tot := totalProbes.Load(); tot == 0 {
		t.Fatalf("expected probes to be executed during load simulation, got 0")
	}

	// Verify states in database
	states, err := st.ListCapacityStates()
	if err != nil {
		t.Fatalf("failed to list capacity states: %v", err)
	}
	if len(states) != profileCount {
		t.Fatalf("expected %d capacity states in db, got %d", profileCount, len(states))
	}

	// Give runtime a moment to settle goroutines
	time.Sleep(10 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()
	growth := finalGoroutines - initialGoroutines
	if growth > 15 {
		t.Fatalf("goroutine leak suspected: initial %d, final %d (growth: %d)", initialGoroutines, finalGoroutines, growth)
	}
}
