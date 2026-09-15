package capacitymonitor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// Monitor is the SQLite-backed adaptive capacity loop. It probes each profile,
// persists per-window observations, evaluates hysteresis transitions against
// stored state, and publishes canonical bus events on transitions. Alert
// delivery and dedup belong to T5; this loop only records and announces.
// It never duplicates QuotaDropMonitor: that monitor keeps its own polling and
// dispatch path untouched (see slice doc).
type Monitor struct {
	deps Dependencies

	mu       sync.Mutex
	running  bool
	isLeader bool
	owner    string
	lastTick time.Time
	failures int
	worst    State
}

// NewMonitor fills sane defaults for nil dependencies.
func NewMonitor(d Dependencies) *Monitor {
	if d.Clock == nil {
		d.Clock = RealClock{}
	}
	if d.JitterSource == nil {
		d.JitterSource = RealJitter{}
	}
	if d.Bus == nil {
		d.Bus = events.DefaultBus()
	}
	if d.MaxConcurrency <= 0 {
		d.MaxConcurrency = 4
	}
	if strings.TrimSpace(d.OwnerID) == "" {
		d.OwnerID = fmt.Sprintf("capacity-%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	if d.LeaseTTL <= 0 {
		d.LeaseTTL = 2 * time.Minute
	}
	if d.BaseInterval <= 0 {
		d.BaseInterval = time.Minute
	}
	return &Monitor{deps: d, worst: StateUnknown}
}

// Tick runs a single collection pass. It is a no-op (nil) when preferences
// disable the monitor or when another process holds the leader lease.
// Per-account probe failures are counted for backoff, never fatal.
func (m *Monitor) Tick(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := m.deps.Clock.Now().UTC()

	prefs, err := m.deps.Store.GetCapacityPreferences()
	if err != nil {
		m.noteFailure()
		return err
	}
	thresholds := prefsToThresholds(prefs)
	if err := thresholds.Validate(); err != nil {
		thresholds = DefaultThresholds()
	}
	if !prefs.Enabled {
		return nil
	}

	profiles, err := m.deps.Profiles()
	if err != nil {
		m.noteFailure()
		return err
	}

	lease, held, err := m.deps.Store.AcquireCapacityLease(m.deps.OwnerID, now, m.deps.LeaseTTL)
	if err != nil {
		m.noteFailure()
		return err
	}
	m.mu.Lock()
	m.isLeader = held
	m.owner = lease.Owner
	m.mu.Unlock()
	if !held {
		return nil
	}

	sem := make(chan struct{}, m.deps.MaxConcurrency)
	var wg sync.WaitGroup
	var wmu sync.Mutex
	worst := StateHealthy
	tickFailures := 0
	for _, prof := range profiles {
		if err := ctx.Err(); err != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(p model.Profile) {
			defer wg.Done()
			defer func() { <-sem }()
			st, failed := m.evaluateProfile(ctx, p, thresholds, now)
			wmu.Lock()
			if failed {
				tickFailures++
			}
			if severityRank(st) > severityRank(worst) {
				worst = st
			}
			wmu.Unlock()
		}(prof)
	}
	wg.Wait()

	m.mu.Lock()
	if tickFailures > 0 {
		m.failures += tickFailures
	} else {
		m.failures = 0
	}
	m.worst = worst
	m.lastTick = now
	m.mu.Unlock()
	return ctx.Err()
}

// Run loops Tick on an adaptive jittered interval until ctx ends.
func (m *Monitor) Run(ctx context.Context) {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	interval := m.deps.BaseInterval
	_ = m.Tick(ctx)
	for {
		wait := m.deps.JitterSource.Duration(interval)
		if wait <= 0 {
			wait = m.deps.BaseInterval
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		_ = m.Tick(ctx)
		m.mu.Lock()
		interval = DetermineCollectionInterval(m.worst, false, false, m.failures, m.minProbeIntervalLocked())
		m.mu.Unlock()
	}
}

// Status snapshots live diagnostics plus persisted resource states.
func (m *Monitor) Status() Status {
	m.mu.Lock()
	st := Status{
		IsLeader:    m.isLeader,
		LeaderOwner: m.owner,
		Running:     m.running,
		LastTickAt:  m.lastTick,
	}
	m.mu.Unlock()

	obs, _ := m.deps.Store.ListCapacityObservations()
	byKey := make(map[string]store.ObservationRecord, len(obs))
	for _, o := range obs {
		byKey[stateKey(o.Provider, o.Profile, o.QuotaGroup, o.WindowKind)] = o
	}
	states, _ := m.deps.Store.ListCapacityStates()
	for _, s := range states {
		rs := ResourceState{
			Key: s.Key, Provider: s.Provider, Profile: s.Profile,
			QuotaGroup: s.QuotaGroup, WindowKind: s.WindowKind,
			State: State(s.State), TransitionedAt: s.TransitionedAt,
			LastObservedAt: s.UpdatedAt,
		}
		if o, ok := byKey[s.Key]; ok {
			rs.RemainingPercent = o.RemainingPercent
			rs.ResetDescription = o.ResetDescription
			rs.LastObservedAt = o.FetchedAt
		}
		st.Resources = append(st.Resources, rs)
	}
	st.MonitoredResources = len(st.Resources)
	return st
}

func (m *Monitor) noteFailure() {
	m.mu.Lock()
	m.failures++
	m.mu.Unlock()
}

func (m *Monitor) minProbeInterval() time.Duration {
	var out time.Duration
	if m.deps.ProbeRegistry == nil {
		return 0
	}
	for _, provider := range m.deps.ProbeRegistry.Providers() {
		if p, ok := m.deps.ProbeRegistry.For(provider); ok {
			if d := p.Descriptor().MinimumInterval; d > out {
				out = d
			}
		}
	}
	return out
}

func (m *Monitor) minProbeIntervalLocked() time.Duration { return m.minProbeInterval() }

// evaluateProfile probes one profile and persists per-window observations and
// transitions. It returns the worst state seen and whether the account failed.
func (m *Monitor) evaluateProfile(ctx context.Context, prof model.Profile, thresholds Thresholds, now time.Time) (State, bool) {
	key := func(group, window string) string { return stateKey(prof.Provider, prof.Name, group, window) }

	probe, ok := m.deps.ProbeRegistry.For(prof.Provider)
	if !ok {
		m.upsertState(prof.Provider, prof.Name, "", "", StateUnknown, "no probe for provider", now)
		return StateUnknown, false
	}
	snap, err := probe.Probe(ctx, prof)
	if err != nil {
		m.upsertState(prof.Provider, prof.Name, "", "", StateUnknown, "probe error", now)
		return StateUnknown, true
	}

	rateLimited := snap.Status == model.UsageRateLimited
	trusted := snap.Status == model.UsageLive || snap.Status == model.UsageCached ||
		snap.Status == "OK" || rateLimited
	if len(snap.Windows) == 0 {
		var remaining *float64
		var state State = StateUnknown
		reason := "no valid observation"
		if trusted {
			prev := State("")
			if rec, err := m.deps.Store.GetCapacityState(key("", "")); err == nil {
				prev = State(rec.State)
			}
			// remaining stays nil without windows: always UNKNOWN, persisted once.
			state, reason = EvaluateTransition(prev, remaining, rateLimited, snap.FetchedAt, expiresOrZero(snap.ExpiresAt), now, thresholds)
			if prev != state {
				m.persistTransition(prof.Provider, prof.Name, "", "", state, reason, remaining, "", now)
			}
		}
		return state, false
	}

	worst := StateHealthy
	failed := false
	for _, w := range snap.Windows {
		m.deps.Store.UpsertCapacityObservation(store.ObservationRecord{
			Provider: prof.Provider, Profile: prof.Name,
			QuotaGroup: w.Group, WindowKind: w.Kind,
			Status: string(snap.Status), Source: string(snap.Source),
			UsedPercent: w.UsedPercent, RemainingPercent: w.RemainingPercent,
			FetchedAt: snap.FetchedAt, ExpiresAt: expiresOrZero(snap.ExpiresAt),
			ResetDescription: w.ResetDescription,
		})
		var state State = StateUnknown
		reason := "untrusted snapshot status"
		prev := State("")
		if rec, err := m.deps.Store.GetCapacityState(key(w.Group, w.Kind)); err == nil {
			prev = State(rec.State)
		}
		if trusted {
			state, reason = EvaluateTransition(prev, w.RemainingPercent, rateLimited, snap.FetchedAt, expiresOrZero(snap.ExpiresAt), now, thresholds)
		}
		if prev != state {
			m.persistTransition(prof.Provider, prof.Name, w.Group, w.Kind, state, reason, w.RemainingPercent, w.ResetDescription, now)
		}
		if severityRank(state) > severityRank(worst) {
			worst = state
		}
	}
	return worst, failed
}

// persistTransition writes the state and publishes alert-worthy bus events.
// Callers compare against the stored value first, so repeat observations stay
// read-only: no write churn, no duplicate announcements.
func (m *Monitor) persistTransition(provider, profile, group, window string, state State, reason string, remaining *float64, resetDesc string, now time.Time) {
	_ = m.deps.Store.UpsertCapacityState(store.CapacityStateRecord{
		Key:      stateKey(provider, profile, group, window),
		Provider: provider, Profile: profile,
		QuotaGroup: group, WindowKind: window,
		State: string(state), Reason: reason,
		TransitionedAt: now, UpdatedAt: now,
	})
	m.publish(provider, profile, group, window, state, reason, remaining, resetDesc)
}

// upsertState records a state without transition comparison (account-level
// fallbacks such as missing probes). It never publishes events.
func (m *Monitor) upsertState(provider, profile, group, window string, state State, reason string, now time.Time) {
	_ = m.deps.Store.UpsertCapacityState(store.CapacityStateRecord{
		Key:      stateKey(provider, profile, group, window),
		Provider: provider, Profile: profile, QuotaGroup: group, WindowKind: window,
		State: string(state), Reason: reason, TransitionedAt: now, UpdatedAt: now,
	})
}

func (m *Monitor) publish(provider, profile, group, window string, state State, reason string, remaining *float64, resetDesc string) {
	var typ events.EventType
	switch state {
	case StateExhausted:
		typ = events.EventQuotaExhausted
	case StateWarning, StateCritical:
		typ = events.EventQuotaLow
	case StateRecovered:
		typ = events.EventQuotaMonitorRecovered
	default:
		return
	}
	data := map[string]any{
		"provider": provider, "profile": profile,
		"group": group, "window": window,
		"state": string(state), "severity": string(state), "reason": reason,
		"fingerprint": DeriveFingerprint(provider, profile, group, window, string(state), ""),
	}
	if remaining != nil {
		data["remaining_percent"] = *remaining
	}
	if resetDesc != "" {
		data["reset_desc"] = resetDesc
	}
	m.deps.Bus.Publish(events.NewEvent("", provider, profile, typ, reason, data))
}

func stateKey(provider, profile, group, window string) string {
	return strings.Join([]string{provider, profile, group, window}, "|")
}

func prefsToThresholds(p store.CapacityPreferences) Thresholds {
	return Thresholds{
		Warning:   p.WarningThreshold,
		Critical:  p.CriticalThreshold,
		Exhausted: p.ExhaustedThreshold,
		Recovery:  p.RecoveryThreshold,
	}
}

func expiresOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func severityRank(s State) int {
	switch s {
	case StateExhausted:
		return 5
	case StateCritical:
		return 4
	case StateWarning:
		return 3
	case StateRecovered:
		return 2
	case StateHealthy:
		return 1
	default:
		return 0
	}
}
