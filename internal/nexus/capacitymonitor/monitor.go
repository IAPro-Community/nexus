package capacitymonitor

import (
	"context"
	"database/sql"
	"errors"
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
//
// Dual-publish note: event types intentionally match the ones QuotaDropMonitor
// emits (QUOTA_LOW/QUOTA_EXHAUSTED) because downstream consumers key on them.
// Only ONE loop may run per process: this package ships no wiring (no service
// start, no main change), so no duplication exists today. Whoever wires Run
// must coordinate with QuotaMonitorService (single dispatcher or distinct
// types) — see slice doc.
type Monitor struct {
	deps Dependencies

	mu           sync.Mutex
	running      bool
	isLeader     bool
	owner        string
	lastTick     time.Time
	failures     int
	worst        State
	lastInterval time.Duration
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
// A nil probe registry is a configuration error and fails loudly.
// Per-account failures never abort the pass; only a fully-failed pass feeds
// the global backoff counter.
func (m *Monitor) Tick(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.deps.ProbeRegistry == nil {
		m.noteFailure()
		return errors.New("capacitymonitor: nil probe registry")
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

	// The lease TTL always covers the next wait: 3x the last interval clamped
	// to [2m, 30m], so a healthy 10m cadence never laps the 2m default while a
	// crashed leader is still detected within half an hour at most.
	lease, held, err := m.deps.Store.AcquireCapacityLease(m.deps.OwnerID, now, m.leaseTTL())
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
	evaluated := 0
	failed := 0
tickLoop:
	for _, prof := range profiles {
		if err := ctx.Err(); err != nil {
			break
		}
		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Done()
			break tickLoop
		}
		go func(p model.Profile) {
			defer wg.Done()
			defer func() { <-sem }()
			st, ok := m.evaluateProfile(ctx, p, thresholds, now)
			wmu.Lock()
			if ok {
				evaluated++
			} else {
				failed++
			}
			if severityRank(st) > severityRank(worst) {
				worst = st
			}
			wmu.Unlock()
		}(prof)
	}
	wg.Wait()

	m.mu.Lock()
	// Backoff follows fully-failed passes only: one sick account must not pin
	// healthy accounts to the 15m ladder. Any evaluated account resets it.
	if len(profiles) > 0 && evaluated == 0 {
		m.failures++
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
		m.lastInterval = interval
		m.mu.Unlock()
	}
}

// Status snapshots live diagnostics plus persisted resource states.
func (m *Monitor) Status() Status {
	m.mu.Lock()
	st := Status{
		IsLeader:            m.isLeader,
		LeaderOwner:         m.owner,
		Running:             m.running,
		LastTickAt:          m.lastTick,
		ConsecutiveFailures: m.failures,
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

// leaseTTL scales with the last collection interval so slow healthy cadences
// never outrun the lease while crash failover stays bounded.
func (m *Monitor) leaseTTL() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	base := m.lastInterval
	if base <= 0 {
		base = m.deps.BaseInterval
	}
	ttl := 3 * base
	if ttl < 2*time.Minute {
		ttl = 2 * time.Minute
	}
	if ttl > 30*time.Minute {
		ttl = 30 * time.Minute
	}
	return ttl
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
// transitions. It returns the worst state seen and whether the account
// evaluated cleanly. Any store error fails the account (feeds backoff) instead
// of announcing phantom transitions.
func (m *Monitor) evaluateProfile(ctx context.Context, prof model.Profile, thresholds Thresholds, now time.Time) (State, bool) {
	key := func(group, window string) string { return stateKey(prof.Provider, prof.Name, group, window) }

	probe, ok := m.deps.ProbeRegistry.For(prof.Provider)
	if !ok {
		if err := m.upsertState(prof.Provider, prof.Name, "", "", StateUnknown, "no probe for provider", now); err != nil {
			return StateUnknown, false
		}
		return StateUnknown, true
	}
	snap, err := probe.Probe(ctx, prof)
	if err != nil {
		// Best-effort visibility row; the account still counts as failed so
		// persistent probe outages feed the backoff ladder.
		_ = m.upsertState(prof.Provider, prof.Name, "", "", StateUnknown, "probe error", now)
		return StateUnknown, false
	}
	scope := scopeKey(snap.AccountScope)

	rateLimited := snap.Status == model.UsageRateLimited
	trusted := snap.Status == model.UsageLive || snap.Status == model.UsageCached ||
		snap.Status == "OK" || rateLimited
	if len(snap.Windows) == 0 {
		// No windows: a single account-level UNKNOWN, persisted once so
		// coverage stays visible instead of silently absent.
		if err := m.upsertState(prof.Provider, prof.Name, "", "", StateUnknown, "no windows in snapshot", now); err != nil {
			return StateUnknown, false
		}
		return StateUnknown, true
	}

	worst := StateHealthy
	failed := false
	for _, w := range snap.Windows {
		if err := m.deps.Store.UpsertCapacityObservation(store.ObservationRecord{
			Provider: prof.Provider, Profile: prof.Name,
			QuotaGroup: w.Group, WindowKind: w.Kind,
			Status: string(snap.Status), Source: string(snap.Source),
			UsedPercent: w.UsedPercent, RemainingPercent: w.RemainingPercent,
			FetchedAt: snap.FetchedAt, ExpiresAt: expiresOrZero(snap.ExpiresAt),
			ResetDescription: w.ResetDescription, AccountScope: scope,
		}); err != nil {
			failed = true
			continue
		}
		var state State = StateUnknown
		reason := "untrusted snapshot status"
		prev, err := m.storedState(key(w.Group, w.Kind))
		if err != nil {
			failed = true
			continue
		}
		if trusted {
			state, reason = EvaluateTransition(prev, w.RemainingPercent, rateLimited, snap.FetchedAt, expiresOrZero(snap.ExpiresAt), now, thresholds)
		}
		if prev != state {
			if err := m.persistTransition(prof.Provider, prof.Name, w.Group, w.Kind, state, reason, w.RemainingPercent, w.ResetDescription, snap.AccountScope, now); err != nil {
				failed = true
				continue
			}
		}
		if severityRank(state) > severityRank(worst) {
			worst = state
		}
	}
	return worst, !failed
}

// storedState reads the persisted state; sql.ErrNoRows means first sight
// (empty previous), any other error fails the account fail-closed.
func (m *Monitor) storedState(k string) (State, error) {
	rec, err := m.deps.Store.GetCapacityState(k)
	if errors.Is(err, sql.ErrNoRows) {
		return State(""), nil
	}
	if err != nil {
		return State(""), err
	}
	return State(rec.State), nil
}

// persistTransition writes the state and publishes alert-worthy bus events.
// A write failure skips the announcement: the bus must never claim a
// transition the store cannot prove on restart.
func (m *Monitor) persistTransition(provider, profile, group, window string, state State, reason string, remaining *float64, resetDesc string, scope model.AccountScope, now time.Time) error {
	if err := m.deps.Store.UpsertCapacityState(store.CapacityStateRecord{
		Key:      stateKey(provider, profile, group, window),
		Provider: provider, Profile: profile,
		QuotaGroup: group, WindowKind: window,
		State: string(state), Reason: reason,
		TransitionedAt: now, UpdatedAt: now,
		AccountScope: scopeKey(scope),
	}); err != nil {
		return err
	}
	m.publish(provider, profile, group, window, state, reason, remaining, resetDesc, scope)
	return nil
}

// upsertState records an account-level fallback state idempotently: identical
// repeats are read-only, so TransitionedAt never drifts on no-change ticks and
// there is no write churn. It never publishes events.
func (m *Monitor) upsertState(provider, profile, group, window string, state State, reason string, now time.Time) error {
	k := stateKey(provider, profile, group, window)
	prev, err := m.storedState(k)
	if err != nil {
		return err
	}
	if prev == state {
		return nil
	}
	return m.deps.Store.UpsertCapacityState(store.CapacityStateRecord{
		Key: k, Provider: provider, Profile: profile,
		QuotaGroup: group, WindowKind: window,
		State: string(state), Reason: reason, TransitionedAt: now, UpdatedAt: now,
	})
}

func (m *Monitor) publish(provider, profile, group, window string, state State, reason string, remaining *float64, resetDesc string, scope model.AccountScope) {
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
		// resetCycle stays empty: T5 owns reset-aware dedup and will refine
		// the fingerprint inputs alongside event persistence.
		"fingerprint": DeriveFingerprint(provider, profile, group, window, string(state), ""),
	}
	if remaining != nil {
		data["remaining_percent"] = *remaining
	}
	if resetDesc != "" {
		data["reset_desc"] = resetDesc
	}
	if key := scopeKey(scope); key != "" {
		data["account_scope"] = key
	}
	m.deps.Bus.Publish(events.NewEvent("", provider, profile, typ, reason, data).WithAccountScope(scope))
}

func stateKey(provider, profile, group, window string) string {
	return strings.Join([]string{provider, profile, group, window}, "|")
}

// scopeKey renders the opaque account scope key, or "" when the snapshot
// carries no verifiable identity. Empty means unattributed, never unknown-account.
func scopeKey(scope model.AccountScope) string {
	if !scope.Verifiable() {
		return ""
	}
	return scope.Key()
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
