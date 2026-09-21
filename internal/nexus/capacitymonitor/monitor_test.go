package capacitymonitor

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

var errTestBoom = errors.New("test boom")

type stubClock struct{ now time.Time }

func (c stubClock) Now() time.Time { return c.now }

type stubProbe struct {
	desc quota.ProbeDescriptor
	snap model.UsageSnapshot
	err  error
}

func (p *stubProbe) Descriptor() quota.ProbeDescriptor { return p.desc }

func (p *stubProbe) Probe(ctx context.Context, prof model.Profile) (model.UsageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.UsageSnapshot{}, err
	}
	if p.err != nil {
		return model.UsageSnapshot{}, p.err
	}
	out := p.snap
	out.ProviderID = prof.Provider
	out.ProfileID = prof.Name
	return out, nil
}

type stubRegistry struct{ probes map[string]quota.Probe }

func (r *stubRegistry) For(provider string) (quota.Probe, bool) {
	p, ok := r.probes[provider]
	return p, ok
}

func (r *stubRegistry) Providers() []string {
	out := make([]string, 0, len(r.probes))
	for k := range r.probes {
		out = append(out, k)
	}
	return out
}

type busCounter struct {
	mu     sync.Mutex
	events []events.Event
}

func (c *busCounter) record(e events.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *busCounter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

func (c *busCounter) types() []events.EventType {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]events.EventType, 0, len(c.events))
	for _, e := range c.events {
		out = append(out, e.Type)
	}
	return out
}

func liveSnapshot(remaining float64, status model.UsageStatus) model.UsageSnapshot {
	return model.UsageSnapshot{
		Status: status, Source: model.SourceOfficialAPI,
		FetchedAt: time.Now().UTC().Add(-time.Second),
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "claude_gpt", RemainingPercent: &remaining},
		},
	}
}

func testMonitor(t *testing.T, profiles []model.Profile, reg ProbeRegistry, prefs *store.CapacityPreferences) (*Monitor, *store.Store, *busCounter) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if prefs != nil {
		if err := s.PutCapacityPreferences(*prefs); err != nil {
			t.Fatalf("seed prefs: %v", err)
		}
	}
	bus := events.NewBus(64)
	counter := &busCounter{}
	bus.SetRecorder(counter.record)
	m := NewMonitor(Dependencies{
		Profiles:      func() ([]model.Profile, error) { return profiles, nil },
		ProbeRegistry: reg,
		Store:         s,
		Bus:           bus,
		Clock:         stubClock{now: time.Now().UTC()},
		JitterSource:  RealJitter{},
		OwnerID:       "test-owner",
		LeaseTTL:      time.Hour,
		BaseInterval:  time.Minute,
	})
	return m, s, counter
}

func TestTickExhaustedEmitsExactlyOnce(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(0, model.UsageLive),
	}
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 1 {
		t.Fatalf("events = %d, want exactly 1", counter.count())
	}
	if typs := counter.types(); typs[0] != events.EventQuotaExhausted {
		t.Fatalf("event = %s, want QUOTA_EXHAUSTED", typs[0])
	}
	rec, err := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if err != nil || rec.State != "EXHAUSTED" {
		t.Fatalf("state = %+v,%v, want EXHAUSTED", rec, err)
	}
	first := rec.TransitionedAt

	// Identical repeat tick: no new event, no state rewrite.
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("second Tick() = %v", err)
	}
	if counter.count() != 1 {
		t.Fatalf("events after repeat = %d, want still 1", counter.count())
	}
	rec2, _ := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if !rec2.TransitionedAt.Equal(first) {
		t.Fatalf("TransitionedAt moved on no-change tick: %v -> %v", first, rec2.TransitionedAt)
	}
	// Same check with an advanced clock: frozen time would mask a rewrite.
	later := time.Now().UTC().Add(2 * time.Hour)
	lateBus := events.NewBus(8)
	lateCounter := &busCounter{}
	lateBus.SetRecorder(lateCounter.record)
	mLate := NewMonitor(Dependencies{
		Profiles:      func() ([]model.Profile, error) { return []model.Profile{{Provider: "codex", Name: "kiver"}}, nil },
		ProbeRegistry: &stubRegistry{probes: map[string]quota.Probe{"codex": probe}},
		Store:         s,
		Bus:           lateBus,
		Clock:         stubClock{now: later},
		JitterSource:  RealJitter{},
		OwnerID:       "test-owner",
		LeaseTTL:      time.Hour,
	})
	if err := mLate.Tick(context.Background()); err != nil {
		t.Fatalf("late tick = %v", err)
	}
	if lateCounter.count() != 0 {
		t.Fatalf("late tick published %d events, want 0 (no-change)", lateCounter.count())
	}
	rec3, _ := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if !rec3.TransitionedAt.Equal(first) {
		t.Fatalf("TransitionedAt drifted on late no-change tick: %v, want %v", rec3.TransitionedAt, first)
	}
}

func TestTickHealthyIsSilent(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(80, model.UsageLive),
	}
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0 for healthy", counter.count())
	}
	rec, err := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if err != nil || rec.State != "HEALTHY" {
		t.Fatalf("state = %+v,%v, want HEALTHY persisted", rec, err)
	}
	// Configured LeaseTTL is a floor the adaptive TTL never shrinks:
	// testMonitor sets 1h, so the lease must outlive now+55m.
	lease, err := s.GetCapacityLease()
	if err != nil || lease == nil {
		t.Fatalf("lease = %+v,%v, want held", lease, err)
	}
	if lease.ExpiresAt.Before(time.Now().UTC().Add(55 * time.Minute)) {
		t.Fatalf("lease expires = %v, want >= now+55m (configured 1h floor)", lease.ExpiresAt)
	}
}

func TestTickWarningThenRecovered(t *testing.T) {
	rem := 15.0
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(rem, model.UsageLive),
	}
	m, _, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("tick1 = %v", err)
	}
	if typs := counter.types(); len(typs) != 1 || typs[0] != events.EventQuotaLow {
		t.Fatalf("events = %v, want single QUOTA_LOW", typs)
	}
	// Climb past recovery: WARNING -> RECOVERED with its own event.
	rem = 30.0
	probe.snap = liveSnapshot(rem, model.UsageLive)
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("tick2 = %v", err)
	}
	if typs := counter.types(); len(typs) != 2 || typs[1] != events.EventQuotaMonitorRecovered {
		t.Fatalf("events = %v, want QUOTA_LOW then QUOTA_MONITOR_RECOVERED", typs)
	}
}

func TestTickDisabledIsNoop(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(0, model.UsageLive),
	}
	prefs := store.DefaultCapacityPreferences()
	prefs.Enabled = false
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, &prefs)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0 while disabled", counter.count())
	}
	states, err := s.ListCapacityStates()
	if err != nil || len(states) != 0 {
		t.Fatalf("states = %+v,%v, want none while disabled", states, err)
	}
}

func TestTickMissingProbeStaysUnknown(t *testing.T) {
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "mystery", Name: "ghost"}},
		&stubRegistry{probes: map[string]quota.Probe{}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0 for unknown", counter.count())
	}
	rec, err := s.GetCapacityState("mystery|ghost||")
	if err != nil || rec.State != "UNKNOWN" {
		t.Fatalf("state = %+v,%v, want UNKNOWN", rec, err)
	}
}

func TestTickRejectsUnsafeProbeDescriptor(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{
			ReadOnly:    false,
			Granularity: quota.ProbeGranularityGroup,
		},
		snap: liveSnapshot(0, model.UsageLive),
	}
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "unsafe"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err == nil {
		t.Fatal("unsafe probe descriptor must fail the monitor tick")
	}
	if counter.count() != 0 {
		t.Fatalf("unsafe probe published %d events, want 0", counter.count())
	}
	rec, err := s.GetCapacityState("codex|unsafe||")
	if err != nil {
		t.Fatal(err)
	}
	if rec.State != string(StateUnknown) || rec.Reason != "unsafe probe descriptor" {
		t.Fatalf("unsafe probe state=%+v, want UNKNOWN with explicit reason", rec)
	}
}

func TestTickUntrustedSnapshotSilent(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(0, model.UsageEstimated),
	}
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0 for estimated data", counter.count())
	}
	obs, err := s.ListCapacityObservations()
	if err != nil || len(obs) != 1 {
		t.Fatalf("telemetry must still persist: %+v,%v", obs, err)
	}
	rec, err := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if err != nil || rec.State != "UNKNOWN" {
		t.Fatalf("state = %+v,%v, want UNKNOWN for estimated", rec, err)
	}
}

func TestTickLeaseContention(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(80, model.UsageLive),
	}
	reg := &stubRegistry{probes: map[string]quota.Probe{"codex": probe}}
	profiles := []model.Profile{{Provider: "codex", Name: "kiver"}}
	mA, s, _ := testMonitor(t, profiles, reg, nil)
	if err := mA.Tick(context.Background()); err != nil {
		t.Fatalf("owner tick = %v", err)
	}
	mB := NewMonitor(Dependencies{
		Profiles:      func() ([]model.Profile, error) { return profiles, nil },
		ProbeRegistry: reg,
		Store:         s,
		Bus:           events.NewBus(8),
		Clock:         stubClock{now: time.Now().UTC()},
		JitterSource:  RealJitter{},
		OwnerID:       "other-owner",
		LeaseTTL:      time.Hour,
	})
	before, _ := s.ListCapacityStates()
	if err := mB.Tick(context.Background()); err != nil {
		t.Fatalf("contender tick = %v", err)
	}
	if mB.Status().IsLeader {
		t.Fatal("contender must not be leader while lease is live")
	}
	after, _ := s.ListCapacityStates()
	if len(before) != len(after) {
		t.Fatalf("contender wrote states: %d -> %d", len(before), len(after))
	}
}

func TestStatusSnapshot(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(5, model.UsageLive),
	}
	m, _, _ := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	st := m.Status()
	if !st.IsLeader || st.MonitoredResources != 1 {
		t.Fatalf("status = %+v, want leader with 1 resource", st)
	}
	if len(st.Resources) != 1 || st.Resources[0].State != StateCritical {
		t.Fatalf("resources = %+v, want single CRITICAL", st.Resources)
	}
	if st.Resources[0].RemainingPercent == nil || *st.Resources[0].RemainingPercent != 5 {
		t.Fatalf("resource remaining = %+v, want 5", st.Resources[0])
	}
}

func TestTickExpiredSnapshotUnknown(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	snap := liveSnapshot(0, model.UsageLive)
	snap.ExpiresAt = &past
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: snap,
	}
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0 for expired data", counter.count())
	}
	obs, err := s.ListCapacityObservations()
	if err != nil || len(obs) != 1 {
		t.Fatalf("expired telemetry must persist: %+v,%v", obs, err)
	}
	rec, err := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if err != nil || rec.State != "UNKNOWN" {
		t.Fatalf("state = %+v,%v, want UNKNOWN for expired", rec, err)
	}
	if m.worst != StateUnknown {
		t.Fatalf("worst = %s, want UNKNOWN on all-untrusted pass", m.worst)
	}
}

func TestTickRecordsNextInterval(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(80, model.UsageLive),
	}
	m, s, _ := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if got := m.lastInterval; got != 10*time.Minute {
		t.Fatalf("lastInterval = %v, want 10m for healthy idle", got)
	}
	// The test helper configures a 1h LeaseTTL floor, which wins over the
	// adaptive 30m: the lease must honor explicit config verbatim here.
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("tick2 = %v", err)
	}
	lease, err := s.GetCapacityLease()
	if err != nil || lease == nil {
		t.Fatalf("lease = %+v,%v, want held", lease, err)
	}
	want := time.Now().UTC().Add(60 * time.Minute)
	if lease.ExpiresAt.Before(want.Add(-2*time.Minute)) || lease.ExpiresAt.After(want.Add(2*time.Minute)) {
		t.Fatalf("lease expires = %v, want ≈ now+60m (1h floor)", lease.ExpiresAt)
	}
}

// funcClock is a mutable clock for tests that advance time between ticks.
type funcClock func() time.Time

func (f funcClock) Now() time.Time { return f() }

func TestTickRenewsLeaseForNewInterval(t *testing.T) {
	now := time.Now().UTC()
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(80, model.UsageLive),
	}
	reg := &stubRegistry{probes: map[string]quota.Probe{"codex": probe}}
	profiles := []model.Profile{{Provider: "codex", Name: "kiver"}}
	m := NewMonitor(Dependencies{
		Profiles:      func() ([]model.Profile, error) { return profiles, nil },
		ProbeRegistry: reg,
		Store:         s,
		Bus:           events.NewBus(8),
		Clock:         funcClock(func() time.Time { return now }),
		JitterSource:  RealJitter{},
		OwnerID:       "test-owner",
		BaseInterval:  time.Minute,
		// No LeaseTTL override: 2m default floor, adaptive 30m wins when healthy.
	})
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("tick1 = %v", err)
	}
	first, err := s.GetCapacityLease()
	if err != nil || first == nil {
		t.Fatalf("lease after tick1 = %+v,%v", first, err)
	}
	// Advance past the cold-start TTL but well within the renewed one, then
	// tick again: the leader must extend, never lapse.
	now = now.Add(5 * time.Minute)
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("tick2 = %v", err)
	}
	second, err := s.GetCapacityLease()
	if err != nil || second == nil {
		t.Fatalf("lease after tick2 = %+v,%v", second, err)
	}
	if !second.ExpiresAt.After(first.ExpiresAt) {
		t.Fatalf("lease not extended: %v -> %v", first.ExpiresAt, second.ExpiresAt)
	}
	if second.Owner != "test-owner" {
		t.Fatalf("owner = %q, want test-owner (no steal across renew)", second.Owner)
	}
}

func TestRunTicksUntilCancel(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(80, model.UsageLive),
	}
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	m := NewMonitor(Dependencies{
		Profiles: func() ([]model.Profile, error) {
			return []model.Profile{{Provider: "codex", Name: "kiver"}}, nil
		},
		ProbeRegistry: &stubRegistry{probes: map[string]quota.Probe{"codex": probe}},
		Store:         s,
		Bus:           events.NewBus(8),
		Clock:         stubClock{now: time.Now().UTC()},
		JitterSource:  RealJitter{},
		OwnerID:       "test-owner",
		LeaseTTL:      time.Hour,
		BaseInterval:  5 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
	st := m.Status()
	if st.LastTickAt.IsZero() {
		t.Fatal("LastTickAt is zero, Run never ticked")
	}
	if st.MonitoredResources != 1 {
		t.Fatalf("resources = %d, want 1 after Run ticks", st.MonitoredResources)
	}
}

func TestScopeDriftRefreshesWithoutRepublish(t *testing.T) {
	scopeA := model.AccountScope{ProviderID: "codex", ProfileID: "kiver", AccountID: "a1", IdentityVersion: "v1"}
	scopeB := model.AccountScope{ProviderID: "codex", ProfileID: "kiver", AccountID: "a9", IdentityVersion: "v1"}
	snap := liveSnapshot(5, model.UsageLive)
	snap.AccountScope = scopeA
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: snap,
	}
	now := time.Now().UTC()
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	bus := events.NewBus(8)
	counter := &busCounter{}
	bus.SetRecorder(counter.record)
	profiles := []model.Profile{{Provider: "codex", Name: "kiver"}}
	mk := func() *Monitor {
		return NewMonitor(Dependencies{
			Profiles:      func() ([]model.Profile, error) { return profiles, nil },
			ProbeRegistry: &stubRegistry{probes: map[string]quota.Probe{"codex": probe}},
			Store:         s,
			Bus:           bus,
			Clock:         funcClock(func() time.Time { return now }),
			JitterSource:  RealJitter{},
			OwnerID:       "test-owner",
			LeaseTTL:      time.Hour,
		})
	}
	if err := mk().Tick(context.Background()); err != nil {
		t.Fatalf("tick1 = %v", err)
	}
	if counter.count() != 1 {
		t.Fatalf("events = %d, want 1 (CRITICAL first sight)", counter.count())
	}
	first, _ := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	// Re-auth under a new identity, same quota: attribution refreshes in
	// place — no new transition, no second announcement.
	now = now.Add(time.Hour)
	snap.AccountScope = scopeB
	probe.snap = snap
	if err := mk().Tick(context.Background()); err != nil {
		t.Fatalf("tick2 = %v", err)
	}
	if counter.count() != 1 {
		t.Fatalf("events = %d, want still 1 (scope drift is silent)", counter.count())
	}
	got, err := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AccountScope != "codex/a9/v1" {
		t.Fatalf("scope = %q, want codex/a9/v1", got.AccountScope)
	}
	if !got.TransitionedAt.Equal(first.TransitionedAt) {
		t.Fatalf("TransitionedAt moved on scope-only drift: %v, want %v", got.TransitionedAt, first.TransitionedAt)
	}
}

func TestTickNilDependenciesFailLoudly(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	base := Dependencies{
		Profiles:      func() ([]model.Profile, error) { return nil, nil },
		ProbeRegistry: &stubRegistry{probes: map[string]quota.Probe{}},
		Store:         s,
		Bus:           events.NewBus(8),
		Clock:         stubClock{now: time.Now().UTC()},
		JitterSource:  RealJitter{},
		OwnerID:       "test-owner",
	}
	for name, mutate := range map[string]func(*Dependencies){
		"nil store":    func(d *Dependencies) { d.Store = nil },
		"nil registry": func(d *Dependencies) { d.ProbeRegistry = nil },
		"nil profiles": func(d *Dependencies) { d.Profiles = nil },
	} {
		t.Run(name, func(t *testing.T) {
			d := base
			mutate(&d)
			if err := NewMonitor(d).Tick(context.Background()); err == nil {
				t.Fatalf("%s: expected loud error, got nil", name)
			}
		})
	}
}

func TestTickClosedStoreFailsClosed(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	bus := events.NewBus(8)
	counter := &busCounter{}
	bus.SetRecorder(counter.record)
	m := NewMonitor(Dependencies{
		Profiles:      func() ([]model.Profile, error) { return []model.Profile{{Provider: "codex", Name: "kiver"}}, nil },
		ProbeRegistry: &stubRegistry{probes: map[string]quota.Probe{}},
		Store:         s,
		Bus:           bus,
		Clock:         stubClock{now: time.Now().UTC()},
		JitterSource:  RealJitter{},
		OwnerID:       "test-owner",
	})
	_ = s.Close()
	if err := m.Tick(context.Background()); err == nil {
		t.Fatal("expected error on dead store, got nil")
	}
	if counter.count() != 0 {
		t.Fatalf("events on DB outage = %d, want 0 (no phantom announcements)", counter.count())
	}
}

func TestMissingProbeTransitionStable(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	bus := events.NewBus(8)
	counter := &busCounter{}
	bus.SetRecorder(counter.record)
	profiles := []model.Profile{{
		Provider: "mystery", Name: "ghost",
		AccountScope: model.AccountScope{ProviderID: "mystery", ProfileID: "ghost", AccountID: "a9", IdentityVersion: "v1"},
	}}
	reg := &stubRegistry{probes: map[string]quota.Probe{}}
	at := time.Now().UTC()
	t0 := at
	mk := func() *Monitor {
		return NewMonitor(Dependencies{
			Profiles:      func() ([]model.Profile, error) { return profiles, nil },
			ProbeRegistry: reg,
			Store:         s,
			Bus:           bus,
			Clock:         stubClock{now: at},
			JitterSource:  RealJitter{},
			OwnerID:       "test-owner",
			LeaseTTL:      time.Hour,
		})
	}
	if err := mk().Tick(context.Background()); err != nil {
		t.Fatalf("tick1 = %v", err)
	}
	// Advance the clock well past any timestamp: a no-change repeat must
	// neither rewrite TransitionedAt nor announce.
	at = at.Add(2 * time.Hour)
	if err := mk().Tick(context.Background()); err != nil {
		t.Fatalf("tick2 = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0", counter.count())
	}
	states, err := s.ListCapacityStates()
	if err != nil || len(states) != 1 {
		t.Fatalf("states = %+v,%v, want single UNKNOWN row (no churn)", states, err)
	}
	if !states[0].TransitionedAt.Equal(t0) {
		t.Fatalf("TransitionedAt drifted on no-change tick: %v, want %v", states[0].TransitionedAt, t0)
	}
	if states[0].AccountScope != "mystery/a9/v1" {
		t.Fatalf("fallback scope = %q, want mystery/a9/v1", states[0].AccountScope)
	}
	// Same UNKNOWN state, new cause (probe now exists but errors): the reason
	// text refreshes while the transition moment stays put, still silent.
	reg.probes["mystery"] = &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		err:  errTestBoom,
	}
	at = at.Add(time.Hour)
	if err := mk().Tick(context.Background()); err != nil {
		t.Fatalf("tick3 = %v", err)
	}
	if counter.count() != 0 {
		t.Fatalf("events = %d, want 0", counter.count())
	}
	flapped, err := s.GetCapacityState("mystery|ghost||")
	if err != nil {
		t.Fatalf("get after reason flap: %v", err)
	}
	if flapped.Reason != "probe error" {
		t.Fatalf("reason = %q, want refreshed 'probe error'", flapped.Reason)
	}
	if !flapped.TransitionedAt.Equal(t0) {
		t.Fatalf("TransitionedAt moved on reason-only refresh: %v, want %v", flapped.TransitionedAt, t0)
	}
}

func TestFailuresTrackedInStatus(t *testing.T) {
	newCustom := func(t *testing.T, profiles func() ([]model.Profile, error), reg ProbeRegistry) (*Monitor, *busCounter) {
		t.Helper()
		s, err := store.Open(filepath.Join(t.TempDir(), "cap.db"))
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { _ = s.Close() })
		bus := events.NewBus(8)
		counter := &busCounter{}
		bus.SetRecorder(counter.record)
		return NewMonitor(Dependencies{
			Profiles: profiles, ProbeRegistry: reg, Store: s, Bus: bus,
			Clock: stubClock{now: time.Now().UTC()}, JitterSource: RealJitter{},
			OwnerID: "test-owner", LeaseTTL: time.Hour,
		}), counter
	}
	healthyProbe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(80, model.UsageLive),
	}
	badProbe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		err:  errTestBoom,
	}
	reg := &stubRegistry{probes: map[string]quota.Probe{"good": healthyProbe, "bad": badProbe}}

	// One sick account among healthy ones must not pin the backoff counter.
	m, _ := newCustom(t,
		func() ([]model.Profile, error) {
			return []model.Profile{{Provider: "good", Name: "a"}, {Provider: "bad", Name: "b"}}, nil
		}, reg)
	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("mixed tick = %v", err)
	}
	if got := m.Status().ConsecutiveFailures; got != 0 {
		t.Fatalf("failures with partial success = %d, want 0", got)
	}

	// A profiles-listing failure is fully failed and surfaces in status.
	m2, _ := newCustom(t,
		func() ([]model.Profile, error) { return nil, errTestBoom },
		&stubRegistry{probes: map[string]quota.Probe{}})
	if err := m2.Tick(context.Background()); err == nil {
		t.Fatal("expected profiles error, got nil")
	}
	if got := m2.Status().ConsecutiveFailures; got != 1 {
		t.Fatalf("failures after profiles error = %d, want 1", got)
	}
}

func TestScopePropagatesToStoreAndEvents(t *testing.T) {
	scope := model.AccountScope{ProviderID: "codex", ProfileID: "kiver", AccountID: "acct-1", IdentityVersion: "v3"}
	snap := liveSnapshot(5, model.UsageLive)
	snap.AccountScope = scope
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: snap,
	}
	m, s, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	obs, err := s.ListCapacityObservations()
	if err != nil || len(obs) != 1 || obs[0].AccountScope != "codex/acct-1/v3" {
		t.Fatalf("observation scope = %+v,%v, want codex/acct-1/v3", obs, err)
	}
	rec, err := s.GetCapacityState("codex|kiver|claude_gpt|5h")
	if err != nil || rec.AccountScope != "codex/acct-1/v3" {
		t.Fatalf("state scope = %+v,%v, want codex/acct-1/v3", rec, err)
	}
	if counter.count() != 1 {
		t.Fatalf("events = %d, want 1", counter.count())
	}
	got, ok := counter.events[0].Data["account_scope"]
	if !ok || got != "codex/acct-1/v3" {
		t.Fatalf("event account_scope = %v, want codex/acct-1/v3", counter.events[0].Data)
	}
}
