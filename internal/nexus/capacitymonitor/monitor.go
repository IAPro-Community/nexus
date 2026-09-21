package capacitymonitor

import (
	"context"
	"database/sql"
	"encoding/json"
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
// Construct via NewMonitor: it fills Clock, JitterSource, Bus, concurrency,
// OwnerID, LeaseTTL and BaseInterval defaults that Tick/Run/Status rely on
// (nil* dependencies panic on use by design — fail loudly, never silently).
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
	// Constructor defaults cover the normal path; these guards make direct
	// misuse fail loudly instead of panicking on first use.
	if m.deps.Store == nil {
		m.noteFailure()
		m.refreshInterval()
		return errors.New("capacitymonitor: nil store")
	}
	if m.deps.ProbeRegistry == nil {
		m.noteFailure()
		m.refreshInterval()
		return errors.New("capacitymonitor: nil probe registry")
	}
	if m.deps.Profiles == nil {
		m.noteFailure()
		m.refreshInterval()
		return errors.New("capacitymonitor: nil profiles source")
	}
	now := m.deps.Clock.Now().UTC()

	prefs, err := m.deps.Store.GetCapacityPreferences()
	if err != nil {
		m.noteFailure()
		m.refreshInterval()
		return err
	}
	thresholds := prefsToThresholds(prefs)
	if err := thresholds.Validate(); err != nil {
		thresholds = DefaultThresholds()
	}
	reminderInterval := time.Duration(prefs.ReminderIntervalSeconds) * time.Second
	if reminderInterval < 0 {
		reminderInterval = 0
	}
	if !prefs.Enabled {
		return nil
	}

	profiles, err := m.deps.Profiles()
	if err != nil {
		m.noteFailure()
		m.refreshInterval()
		return err
	}

	// The lease TTL always covers the next wait: 3x the last interval clamped
	// to [2m, 30m], so a healthy 10m cadence never laps the 2m default while a
	// crashed leader is still detected within half an hour at most.
	lease, held, err := m.deps.Store.AcquireCapacityLease(m.deps.OwnerID, now, m.leaseTTL())
	if err != nil {
		m.noteFailure()
		m.refreshInterval()
		return err
	}
	m.mu.Lock()
	m.isLeader = held
	m.owner = lease.Owner
	m.mu.Unlock()
	if !held {
		return nil
	}

	workers := m.deps.MaxConcurrency
	if workers <= 0 {
		workers = 4
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var wmu sync.Mutex
	worst := StateUnknown
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
			st, ok := m.evaluateProfile(ctx, p, thresholds, reminderInterval, now)
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
	// Single computation point for the adaptive cadence: both Run and
	// Tick-only callers (via leaseTTL) observe the same interval, so the
	// lease always covers the next wait even under backoff.
	m.refreshIntervalLocked()
	// Extend our just-acquired lease to the NEW interval: the acquire above
	// covered the previous cadence, and a shallow-backoff→healthy recovery
	// would otherwise leave a short TTL over a long sleep, inviting a steal.
	if held {
		if err := m.deps.Store.RenewCapacityLease(m.deps.OwnerID, now, m.ttlForInterval(m.lastInterval)); err != nil {
			m.noteFailure()
		}
	}
	m.mu.Unlock()
	// The pass completed: report cancellation only if it actually landed.
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
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
		interval = m.lastInterval
		if interval <= 0 {
			interval = m.deps.BaseInterval
		}
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

// maxLeaseTTL bounds crash failover: even an explicit operator floor above
// this is clamped, so a typo can never pin leadership for weeks.
const maxLeaseTTL = 24 * time.Hour

// ttlForInterval scales a cadence into lease coverage: 3x, honoring the
// configured LeaseTTL as floor and maxLeaseTTL as absolute cap.
func (m *Monitor) ttlForInterval(interval time.Duration) time.Duration {
	base := interval
	if base <= 0 {
		base = m.deps.BaseInterval
	}
	if base > 10*time.Minute {
		base = 10 * time.Minute
	}
	ttl := 3 * base
	if ttl < m.deps.LeaseTTL {
		ttl = m.deps.LeaseTTL
	}
	if ttl > maxLeaseTTL {
		ttl = maxLeaseTTL
	}
	return ttl
}

// leaseTTL covers the upcoming wait with the last computed cadence.
func (m *Monitor) leaseTTL() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ttlForInterval(m.lastInterval)
}

// refreshIntervalLocked recomputes the adaptive cadence from the current
// worst state and backoff counter. Callers must hold m.mu.
func (m *Monitor) refreshIntervalLocked() {
	m.lastInterval = DetermineCollectionInterval(m.worst, false, false, m.failures, m.minProbeInterval())
}

// refreshInterval recomputes the cadence for early-return paths so a failing
// pass still steers Run onto the retry ladder instead of a stale cadence.
func (m *Monitor) refreshInterval() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshIntervalLocked()
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

// evaluateProfile probes one profile and persists per-window observations and
// transitions. It returns the worst state seen and whether the account
// evaluated cleanly. Any store error fails the account (feeds backoff) instead
// of announcing phantom transitions.
func (m *Monitor) evaluateProfile(ctx context.Context, prof model.Profile, thresholds Thresholds, reminderInterval time.Duration, now time.Time) (State, bool) {
	key := func(group, window string) string { return stateKey(prof.Provider, prof.Name, group, window) }

	probe, ok := m.deps.ProbeRegistry.For(prof.Provider)
	if !ok {
		if err := m.upsertState(prof.Provider, prof.Name, StateUnknown, "no probe for provider", prof.AccountScope, now); err != nil {
			return StateUnknown, false
		}
		return StateUnknown, true
	}
	if err := probe.Descriptor().Validate(); err != nil {
		_ = m.upsertState(prof.Provider, prof.Name, StateUnknown, "unsafe probe descriptor", prof.AccountScope, now)
		return StateUnknown, false
	}
	snap, err := probe.Probe(ctx, prof)
	if err != nil {
		// Best-effort visibility row; the account still counts as failed so
		// persistent probe outages feed the backoff ladder.
		_ = m.upsertState(prof.Provider, prof.Name, StateUnknown, "probe error", prof.AccountScope, now)
		return StateUnknown, false
	}
	scope := scopeKey(snap.AccountScope)

	rateLimited := normalizeStatus(snap.Status) == string(model.UsageRateLimited)
	// Trust set mirrors QuotaDropMonitor.isTrustworthyQuotaStatus, including
	// the legacy "OK" some producers still emit: both monitors must agree on
	// what counts as evidence. Normalization keeps "ok"/" live " honest too.
	trusted := rateLimited
	switch normalizeStatus(snap.Status) {
	case string(model.UsageLive), string(model.UsageCached), "OK":
		trusted = true
	}
	if len(snap.Windows) == 0 {
		// No windows: a single account-level UNKNOWN, persisted once so
		// coverage stays visible instead of silently absent.
		if err := m.upsertState(prof.Provider, prof.Name, StateUnknown, "no windows in snapshot", snap.AccountScope, now); err != nil {
			return StateUnknown, false
		}
		return StateUnknown, true
	}

	worst := StateUnknown
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
		state := StateUnknown
		reason := "untrusted snapshot status"
		rec, found, err := m.loadState(key(w.Group, w.Kind))
		if err != nil {
			failed = true
			continue
		}
		prev := State("")
		prevScope := ""
		if found {
			prev = State(rec.State)
			prevScope = rec.AccountScope
		}
		if trusted {
			state, reason = EvaluateTransition(prev, w.RemainingPercent, rateLimited, snap.FetchedAt, expiresOrZero(snap.ExpiresAt), now, thresholds)
		}
		// Scope drift persists too: re-auth under a new identity refreshes
		// attribution even when the state itself did not move — in place, with
		// no new transition and no announcement.
		if prev != state {
			if err := m.persistTransition(ctx, prof.Provider, prof.Name, w.Group, w.Kind, state, reason, w.RemainingPercent, w.ResetTime, expiresOrZero(snap.ExpiresAt), w.ResetDescription, snap.AccountScope, reminderInterval, now); err != nil {
				failed = true
				continue
			}
		} else if found && prevScope != scopeKey(snap.AccountScope) {
			rec.Reason = reason
			if err := m.deps.Store.RefreshCapacityStateDetail(rec, reason, scopeKey(snap.AccountScope), now); err != nil {
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

// normalizeStatus uppercases/trims provider status text so "ok" and " live "
// match their canonical forms, exactly like the existing quota monitor.
func normalizeStatus(s model.UsageStatus) string {
	return strings.ToUpper(strings.TrimSpace(string(s)))
}

// loadState reads one persisted state row. Not-found means first sight
// (found=false, nil error); any other error fails the account fail-closed.
// Single mapping point for the missing-key contract.
func (m *Monitor) loadState(k string) (store.CapacityStateRecord, bool, error) {
	rec, err := m.deps.Store.GetCapacityState(k)
	if errors.Is(err, sql.ErrNoRows) {
		return store.CapacityStateRecord{}, false, nil
	}
	if err != nil {
		return store.CapacityStateRecord{}, false, err
	}
	return rec, true, nil
}

// persistTransition writes the state, then applies the T5 re-alert rule before
// announcing: a new alert exists only on state change, new window, different
// resource, recovery, or explicit repeat (reminder interval). A write or
// lookup failure skips the announcement: the bus must never claim a
// transition the store cannot prove on restart. The event row is persisted
// BEFORE publishing so a crash-then-restart never duplicates it (restart-safe:
// the next pass sees the same state and stays silent).
func (m *Monitor) persistTransition(ctx context.Context, provider, profile, group, window string, state State, reason string, remaining *float64, resetTime *time.Time, snapshotExpiry time.Time, resetDesc string, scope model.AccountScope, reminderInterval time.Duration, now time.Time) error {
	if err := m.deps.Store.UpsertCapacityState(store.CapacityStateRecord{
		Key:      stateKey(provider, profile, group, window),
		Provider: provider, Profile: profile,
		QuotaGroup: "", WindowKind: "",
		State: string(state), Reason: reason,
		TransitionedAt: now, UpdatedAt: now,
		AccountScope: scopeKey(scope),
	}); err != nil {
		return err
	}
	resetCycle := ResetCycleKey(resetTime, snapshotExpiry)
	fingerprint := DeriveFingerprint(provider, profile, group, window, string(state), resetCycle)
	last, err := lastEventFor(m.deps.Store, fingerprint)
	if err != nil {
		return err
	}
	decision := ShouldDeliver(state, fingerprint, last, now, reminderInterval, false)
	if !decision.Deliver {
		return nil
	}
	// Human triage (T6 ack/snooze endpoints) is enforced here, not just
	// surfaced: an acknowledged fingerprint stays silent on repeats and a
	// snoozed one stays silent until its time passes. New states carry a new
	// fingerprint, so ack/snooze can never suppress a genuinely new condition.
	// A triage lookup failure fails closed to delivery (the bus path, not the
	// human path, is the one that must never lose an alert on store error).
	if triage, ok, terr := m.deps.Store.GetCapacityAlertState(fingerprint); terr == nil && ok {
		if triage.Acked {
			return nil
		}
		if triage.SnoozedUntil != nil && now.Before(*triage.SnoozedUntil) {
			return nil
		}
	}
	// Single dispatcher: when the legacy QuotaDropMonitor already announced
	// this condition in the current cycle, this loop stays silent. The claim
	// happens BEFORE the event row so a suppressed cycle leaves no row that
	// would confuse the next pass. Nil gate allows (legacy tests, unwired).
	dispatchKey := DispatchKey{
		Provider: provider, Profile: profile,
		Group: group, Window: window,
		Class: ClassForState(state),
	}
	if !m.deps.Dispatcher.Claim(OwnerCapacityMonitor, dispatchKey, reminderInterval) {
		return nil
	}
	eventID := fmt.Sprintf("%s@%d", fingerprint, now.UnixNano())
	payload, _ := json.Marshal(map[string]any{
		"state": string(state), "severity": string(state), "reason": reason,
		"reset_cycle": resetCycle,
	})
	if err := m.deps.Store.PutCapacityEvent(store.CapacityEventRecord{
		ID: eventID, Fingerprint: fingerprint,
		EventType: string(eventTypeFor(state)),
		Provider:  provider, Profile: profile,
		QuotaGroup: "", WindowKind: window,
		Severity: string(state), PayloadJSON: string(payload),
		EmittedAt: now,
	}); err != nil {
		// The claim did not materialize: release it so the next pass retries
		// instead of waiting out the TTL for an announcement that never
		// happened. The bus still never claims what the store cannot prove.
		m.deps.Dispatcher.Forget(dispatchKey)
		return err
	}
	m.publish(ctx, provider, profile, group, window, state, reason, remaining, resetDesc, scope, fingerprint)
	// Best-effort receipt: the canonical event is already durable; a receipt
	// write failure must not un-publish it.
	_ = m.deps.Store.PutDeliveryReceipt(store.DeliveryReceipt{
		AlertID: eventID, Channel: "bus", Fingerprint: fingerprint,
		Status: "delivered", DeliveredAt: now,
	})
	return nil
}

// eventTypeFor maps an alert-worthy state to its canonical bus type.
func eventTypeFor(state State) events.EventType {
	switch state {
	case StateExhausted:
		return events.EventQuotaExhausted
	case StateWarning, StateCritical:
		return events.EventQuotaLow
	default:
		return events.EventQuotaMonitorRecovered
	}
}

// upsertState records an account-level fallback state idempotently: identical
// repeats are read-only, so TransitionedAt never drifts on no-change ticks and
// there is no write churn. Changed reason or scope refreshes the diagnostic
// columns without moving the transition moment. It never publishes events.
func (m *Monitor) upsertState(provider, profile string, state State, reason string, scope model.AccountScope, now time.Time) error {
	k := stateKey(provider, profile, "", "")
	rec, found, err := m.loadState(k)
	if err != nil {
		return err
	}
	if found {
		if State(rec.State) == state {
			if rec.Reason == reason && rec.AccountScope == scopeKey(scope) {
				return nil
			}
			rec.Reason = reason
			rec.AccountScope = scopeKey(scope)
			return m.deps.Store.RefreshCapacityStateDetail(rec, reason, scopeKey(scope), now)
		}
	}
	return m.deps.Store.UpsertCapacityState(store.CapacityStateRecord{
		Key: k, Provider: provider, Profile: profile,
		QuotaGroup: "", WindowKind: window,
		State: string(state), Reason: reason, TransitionedAt: now, UpdatedAt: now,
		AccountScope: scopeKey(scope),
	})
}

func (m *Monitor) publish(ctx context.Context, provider, profile, group, window string, state State, reason string, remaining *float64, resetDesc string, scope model.AccountScope, fingerprint string) {
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
		// Scope is excluded on purpose: re-auth under a new identity keeps
		// the same fingerprint so dedup continuity survives credential
		// rotation (a simultaneous cross-account collision is impossible —
		// profile names are unique per provider). The reset cycle IS part
		// of the fingerprint (T5): a new quota window re-alerts even when
		// the severity repeats.
		"fingerprint": fingerprint,
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
	// T4 recommendation bridge: advisory candidate + reason on degraded
	// states only. Nil recommender (or unavailable/error) publishes without
	// it — never a phantom candidate, never an automatic migration.
	if m.deps.Recommender != nil && (state == StateWarning || state == StateCritical || state == StateExhausted) {
		if rec, err := m.deps.Recommender.RecommendForLowResource(ctx,
			ResourceKey{Provider: provider, Profile: profile, Group: group, Window: window}, reason); err == nil && rec.Available {
			data["recommended_provider"] = rec.Provider
			data["recommended_profile"] = rec.Profile
			data["recommended_display_name"] = rec.DisplayName
			data["recommendation_score"] = rec.Score
			data["recommendation_reason"] = rec.Reason
		}
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
