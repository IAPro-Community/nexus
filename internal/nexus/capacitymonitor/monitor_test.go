package capacitymonitor

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

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
