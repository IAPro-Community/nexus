package capacitymonitor

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type fakeJitter struct{}

func (f *fakeJitter) Duration(base time.Duration) time.Duration {
	return base
}

type fakeProbe struct {
	desc quota.ProbeDescriptor
	fn   func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error)
}

func (f *fakeProbe) Descriptor() quota.ProbeDescriptor {
	return f.desc
}

func (f *fakeProbe) Probe(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
	if f.fn != nil {
		return f.fn(ctx, p)
	}
	return model.UsageSnapshot{
		ProviderID: p.Provider,
		ProfileID:  p.Name,
		Status:     model.UsageLive,
		Source:     model.SourceCLIOutput,
		FetchedAt:  time.Now(),
	}, nil
}

type fakeRegistry struct {
	probes map[string]quota.Probe
}

func (r *fakeRegistry) For(provider string) (quota.Probe, bool) {
	p, ok := r.probes[provider]
	return p, ok
}

func (r *fakeRegistry) Providers() []string {
	var list []string
	for k := range r.probes {
		list = append(list, k)
	}
	return list
}

type fakeDetector struct {
	active map[string]bool
}

func (d *fakeDetector) Snapshot(ctx context.Context) (map[string]bool, error) {
	return d.active, nil
}

type fakePower struct {
	low bool
}

func (p *fakePower) LowPower(ctx context.Context) (bool, error) {
	return p.low, nil
}

type fakeEventPublisher struct {
	mu     sync.Mutex
	events []store.CapacityEventRecord
}

func (p *fakeEventPublisher) Publish(ev store.CapacityEventRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, ev)
	return nil
}

func (p *fakeEventPublisher) Events() []store.CapacityEventRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	copied := make([]store.CapacityEventRecord, len(p.events))
	copy(copied, p.events)
	return copied
}

func TestThresholdsValidation(t *testing.T) {
	valid := DefaultThresholds()
	if err := valid.Validate(); err != nil {
		t.Fatalf("default thresholds must be valid: %v", err)
	}

	invalid := Thresholds{
		Warning:   10,
		Critical:  20, // Critical > Warning is invalid
		Exhausted: 0,
		Recovery:  25,
	}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error for Critical >= Warning")
	}
}

func TestEvaluateTransition(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	thresh := DefaultThresholds()

	// 1. Healthy -> Warning
	rem20 := 20.0
	state, _ := EvaluateTransition(StateHealthy, &rem20, false, now, now.Add(time.Hour), now, thresh)
	if state != StateWarning {
		t.Fatalf("state = %q, want %q", state, StateWarning)
	}

	// 2. Warning -> Critical
	rem10 := 9.0
	state, _ = EvaluateTransition(StateWarning, &rem10, false, now, now.Add(time.Hour), now, thresh)
	if state != StateCritical {
		t.Fatalf("state = %q, want %q", state, StateCritical)
	}

	// 3. Critical -> Exhausted
	rem0 := 0.0
	state, _ = EvaluateTransition(StateCritical, &rem0, false, now, now.Add(time.Hour), now, thresh)
	if state != StateExhausted {
		t.Fatalf("state = %q, want %q", state, StateExhausted)
	}

	// 4. Exhausted -> Recovered (at 26% >= 25%)
	rem26 := 26.0
	state, _ = EvaluateTransition(StateExhausted, &rem26, false, now, now.Add(time.Hour), now, thresh)
	if state != StateRecovered {
		t.Fatalf("state = %q, want %q", state, StateRecovered)
	}

	// 5. Recovered -> Healthy
	state, _ = EvaluateTransition(StateRecovered, &rem26, false, now, now.Add(time.Hour), now, thresh)
	if state != StateHealthy {
		t.Fatalf("state = %q, want %q", state, StateHealthy)
	}

	// 6. Stale observation becomes Unknown
	got, _ := EvaluateTransition(StateWarning, &rem20, false, now.Add(-time.Hour), now.Add(-time.Minute), now, thresh)
	if got != StateUnknown {
		t.Fatalf("state = %q, want %q for expired observation", got, StateUnknown)
	}

	// 7. Rate limited becomes Exhausted
	state, _ = EvaluateTransition(StateHealthy, &rem26, true, now, now.Add(time.Hour), now, thresh)
	if state != StateExhausted {
		t.Fatalf("state = %q, want %q for rate limited", state, StateExhausted)
	}
}

func TestMonitorLeadershipAndScheduling(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus_mon_test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	clock := newFakeClock(time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC))
	pub := &fakeEventPublisher{}

	var probeCalls int32
	rem := 15.0
	reg := &fakeRegistry{
		probes: map[string]quota.Probe{
			"agy": &fakeProbe{
				desc: quota.ProbeDescriptor{
					Sources:         []model.UsageSource{model.SourceCLIOutput},
					ReadOnly:        true,
					MinimumInterval: 30 * time.Second,
					Confidence:      model.UsageLive,
					TTL:             5 * time.Minute,
					Granularity:     model.UsageGranularityGroup,
				},
				fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
					atomic.AddInt32(&probeCalls, 1)
					return model.UsageSnapshot{
						ProviderID: "agy",
						ProfileID:  p.Name,
						Status:     model.UsageLive,
						Source:     model.SourceCLIOutput,
						FetchedAt:  clock.Now(),
						ExpiresAt:  func() *time.Time { t := clock.Now().Add(5 * time.Minute); return &t }(),
						Windows: []model.UsageWindow{{
							Kind:             "5h",
							Group:            "gemini",
							RemainingPercent: &rem,
						}},
					}, nil
				},
			},
		},
	}

	deps := Dependencies{
		Profiles: func() ([]model.Profile, error) {
			return []model.Profile{{Provider: "agy", Name: "work"}}, nil
		},
		ProbeRegistry:    reg,
		ActivityDetector: &fakeDetector{active: map[string]bool{"agy": false}},
		PowerState:       &fakePower{low: false},
		Store:            st,
		EventPublisher:   pub,
		Clock:            clock,
		JitterSource:     &fakeJitter{},
		MaxConcurrency:   2,
		OwnerID:          "node-leader",
	}

	mon := New(deps)

	// Tick 1: should acquire lease and perform probe
	ctx := context.Background()
	if err := mon.Tick(ctx); err != nil {
		t.Fatalf("mon tick: %v", err)
	}

	status, err := mon.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.IsLeader {
		t.Fatal("expected monitor to be leader")
	}
	if atomic.LoadInt32(&probeCalls) != 1 {
		t.Fatalf("expected 1 probe call, got %d", atomic.LoadInt32(&probeCalls))
	}

	// Should have emitted a threshold warning event
	events := pub.Events()
	if len(events) != 1 || events[0].Severity != "WARNING" {
		t.Fatalf("expected 1 warning event, got %+v", events)
	}

	// Second instance with different owner should become follower and not probe
	depsFollower := deps
	depsFollower.OwnerID = "node-follower"
	monFollower := New(depsFollower)
	if err := monFollower.Tick(ctx); err != nil {
		t.Fatalf("monFollower tick: %v", err)
	}
	statusF, _ := monFollower.Status(ctx)
	if statusF.IsLeader {
		t.Fatal("follower must not claim leadership while active leader exists")
	}
	if atomic.LoadInt32(&probeCalls) != 1 {
		t.Fatalf("follower must not run probes, call count was %d", atomic.LoadInt32(&probeCalls))
	}
}
