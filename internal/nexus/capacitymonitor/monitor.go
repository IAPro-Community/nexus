package capacitymonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// Monitor coordinates single-leader capacity polling, threshold evaluation, and event emission.
type Monitor struct {
	deps Dependencies

	mu            sync.RWMutex
	ownerID       string
	isLeader      bool
	leaderOwner   string
	running       bool
	lastTickAt    time.Time
	consecErrors  map[string]int
	profileLocks  map[string]*sync.Mutex
	profileLockMu sync.Mutex
	nextDue       map[string]time.Time
}

// New constructs a new Monitor instance.
func New(deps Dependencies) *Monitor {
	if deps.Clock == nil {
		deps.Clock = RealClock{}
	}
	if deps.JitterSource == nil {
		deps.JitterSource = RealJitter{}
	}
	if deps.MaxConcurrency <= 0 {
		deps.MaxConcurrency = 2
	}
	owner := deps.OwnerID
	if owner == "" {
		host, _ := os.Hostname()
		owner = fmt.Sprintf("nexus-%s-%d", host, os.Getpid())
	}

	return &Monitor{
		deps:         deps,
		ownerID:      owner,
		consecErrors: make(map[string]int),
		profileLocks: make(map[string]*sync.Mutex),
		nextDue:      make(map[string]time.Time),
	}
}

func (m *Monitor) getProfileLock(key string) *sync.Mutex {
	m.profileLockMu.Lock()
	defer m.profileLockMu.Unlock()
	lock, ok := m.profileLocks[key]
	if !ok {
		lock = &sync.Mutex{}
		m.profileLocks[key] = lock
	}
	return lock
}

// Run executes the monitoring loop until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) error {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.running = false
		isLeader := m.isLeader
		m.mu.Unlock()
		if isLeader && m.deps.Store != nil {
			_ = m.deps.Store.ReleaseCapacityLease(m.ownerID)
		}
	}()

	for {
		interval := m.nextWakeInterval()
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
			if err := m.tick(ctx, false); err != nil && ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
}

func (m *Monitor) nextWakeInterval() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := m.deps.Clock.Now()
	if len(m.nextDue) == 0 {
		return time.Second
	}
	interval := 10 * time.Minute
	for _, due := range m.nextDue {
		if due.IsZero() || !due.After(now) {
			return time.Second
		}
		if wait := due.Sub(now); wait < interval {
			interval = wait
		}
	}
	return interval
}

// Tick executes a single evaluation pass across registered profiles.
func (m *Monitor) Tick(ctx context.Context) error {
	return m.tick(ctx, true)
}

func (m *Monitor) tick(ctx context.Context, force bool) error {
	now := m.deps.Clock.Now()

	m.mu.Lock()
	m.lastTickAt = now
	m.mu.Unlock()

	// 1. Leader election
	leaseTTL := 30 * time.Second
	lease, acquired, err := m.deps.Store.AcquireCapacityLease(m.ownerID, now, leaseTTL)
	if err != nil {
		return fmt.Errorf("acquire capacity lease: %w", err)
	}

	m.mu.Lock()
	m.isLeader = acquired
	m.leaderOwner = lease.Owner
	m.mu.Unlock()

	if !acquired {
		// Follower mode: do not perform collections
		return nil
	}

	// A pass may contain multiple slow provider probes. Renew the lease while it runs
	// so another daemon cannot take leadership during a long collection.
	renewCtx, cancelRenew := context.WithCancel(ctx)
	defer cancelRenew()
	renewErr := make(chan error, 1)
	go m.renewLeaseLoop(renewCtx, renewErr)

	// 2. Read preferences
	prefs, err := m.deps.Store.GetCapacityPreferences()
	if err != nil {
		prefs = store.DefaultCapacityPreferences()
	}
	if !prefs.Enabled {
		return nil
	}

	thresholds := Thresholds{
		Warning:   prefs.WarningThreshold,
		Critical:  prefs.CriticalThreshold,
		Exhausted: prefs.ExhaustedThreshold,
		Recovery:  prefs.RecoveryThreshold,
	}
	if err := thresholds.Validate(); err != nil {
		thresholds = DefaultThresholds()
	}

	// 3. Environment & activity discovery
	var activityMap map[string]bool
	if m.deps.ActivityDetector != nil {
		activityMap, _ = m.deps.ActivityDetector.Snapshot(ctx)
	}
	var lowPower bool
	if m.deps.PowerState != nil {
		lowPower, _ = m.deps.PowerState.LowPower(ctx)
	}

	// 4. Retrieve profiles
	if m.deps.Profiles == nil {
		return nil
	}
	profiles, err := m.deps.Profiles()
	if err != nil {
		return fmt.Errorf("fetch profiles: %w", err)
	}

	// 5. Bounded concurrency execution
	sem := make(chan struct{}, m.deps.MaxConcurrency)
	var wg sync.WaitGroup

	for _, p := range profiles {
		if p.Disabled {
			continue
		}
		probe, ok := m.deps.ProbeRegistry.For(p.Provider)
		if !ok || probe == nil {
			continue
		}

		profileKey := fmt.Sprintf("%s:%s", p.Provider, p.Name)
		if !force && !m.profileDue(profileKey, now) {
			continue
		}
		pLock := m.getProfileLock(profileKey)

		wg.Add(1)
		go func(prof model.Profile, pr quota.Probe, lock *sync.Mutex) {
			defer wg.Done()

			// Enforce per-profile singleflight
			lock.Lock()
			defer lock.Unlock()

			// Acquire global concurrency semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			state := m.evaluateProfile(ctx, prof, pr, thresholds, activityMap, lowPower, now)
			interval := DetermineCollectionInterval(state, activityMap[prof.Provider] || activityMap[prof.Name], lowPower, m.failureCount(prof), pr.Descriptor().MinimumInterval)
			interval = m.deps.JitterSource.Duration(interval)
			m.setNextDue(fmt.Sprintf("%s:%s", prof.Provider, prof.Name), now.Add(interval))
		}(p, probe, pLock)
	}

	wg.Wait()
	select {
	case err := <-renewErr:
		return fmt.Errorf("renew capacity lease: %w", err)
	default:
	}
	return nil
}

func (m *Monitor) renewLeaseLoop(ctx context.Context, errs chan<- error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.deps.Store.RenewCapacityLease(m.ownerID, m.deps.Clock.Now(), 30*time.Second); err != nil {
				select {
				case errs <- err:
				default:
				}
				return
			}
		}
	}
}

func (m *Monitor) profileDue(key string, now time.Time) bool {
	m.mu.RLock()
	due := m.nextDue[key]
	m.mu.RUnlock()
	return due.IsZero() || !now.Before(due)
}

func (m *Monitor) setNextDue(key string, due time.Time) {
	m.mu.Lock()
	m.nextDue[key] = due
	m.mu.Unlock()
}

func (m *Monitor) failureCount(prof model.Profile) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.consecErrors[fmt.Sprintf("%s:%s", prof.Provider, prof.Name)]
}

func (m *Monitor) evaluateProfile(
	ctx context.Context,
	prof model.Profile,
	pr quota.Probe,
	thresholds Thresholds,
	activityMap map[string]bool,
	lowPower bool,
	now time.Time,
) State {
	snap, err := pr.Probe(ctx, prof)

	m.mu.Lock()
	profKey := fmt.Sprintf("%s:%s", prof.Provider, prof.Name)
	if err != nil {
		m.consecErrors[profKey]++
	} else {
		m.consecErrors[profKey] = 0
	}
	m.mu.Unlock()

	if err != nil {
		return StateUnknown
	}

	// If snapshot has no windows, create a synthetic single group window if status is unknown/unsupported
	windows := snap.Windows
	if len(windows) == 0 {
		windows = []model.UsageWindow{{
			Kind:             "default",
			Group:            snap.QuotaGroup,
			RemainingPercent: nil,
		}}
	}

	resultState := StateHealthy
	for _, w := range windows {
		var expiresAt time.Time
		if snap.ExpiresAt != nil {
			expiresAt = *snap.ExpiresAt
		}

		obsRecord := store.ObservationRecord{
			Provider:         prof.Provider,
			Profile:          prof.Name,
			QuotaGroup:       w.Group,
			WindowKind:       w.Kind,
			Status:           string(snap.Status),
			Source:           string(snap.Source),
			UsedPercent:      w.UsedPercent,
			RemainingPercent: w.RemainingPercent,
			FetchedAt:        snap.FetchedAt,
			ExpiresAt:        expiresAt,
			ResetDescription: w.ResetDescription,
		}
		_ = m.deps.Store.UpsertCapacityObservation(obsRecord)

		stateKey := fmt.Sprintf("%s:%s:%s:%s", prof.Provider, prof.Name, w.Group, w.Kind)
		prevStateRecord, err := m.deps.Store.GetCapacityState(stateKey)
		prevState := StateUnknown
		if err == nil {
			prevState = State(prevStateRecord.State)
		}

		rateLimited := snap.Status == model.UsageRateLimited
		newState, reason := EvaluateTransition(
			prevState,
			w.RemainingPercent,
			rateLimited,
			snap.FetchedAt,
			expiresAt,
			now,
			thresholds,
		)
		if newState == StateExhausted || newState == StateCritical || (newState == StateWarning && resultState == StateHealthy) {
			resultState = newState
		} else if resultState == StateHealthy && newState != StateHealthy {
			resultState = newState
		}

		stateRecord := store.CapacityStateRecord{
			Key:            stateKey,
			Provider:       prof.Provider,
			Profile:        prof.Name,
			QuotaGroup:     w.Group,
			WindowKind:     w.Kind,
			State:          string(newState),
			Reason:         reason,
			TransitionedAt: now,
			UpdatedAt:      now,
		}
		if prevState != newState {
			stateRecord.TransitionedAt = now
		} else if err == nil {
			stateRecord.TransitionedAt = prevStateRecord.TransitionedAt
		}
		_ = m.deps.Store.UpsertCapacityState(stateRecord)

		// Emit canonical event if state changed to a significant threshold or recovery
		if prevState != newState {
			var eventType string
			switch newState {
			case StateWarning, StateCritical, StateExhausted:
				eventType = "quota.threshold_crossed"
			case StateRecovered:
				eventType = "quota.recovered"
			}

			if eventType != "" {
				resetDesc := w.ResetDescription
				fp := DeriveFingerprint(prof.Provider, prof.Name, w.Group, w.Kind, string(newState), resetDesc)

				payload := map[string]interface{}{
					"provider":          prof.Provider,
					"profile":           prof.Name,
					"quota_group":       w.Group,
					"window_kind":       w.Kind,
					"previous_state":    string(prevState),
					"new_state":         string(newState),
					"remaining_percent": w.RemainingPercent,
					"reason":            reason,
					"reset_description": resetDesc,
				}
				payloadBytes, _ := json.Marshal(payload)

				evRecord := store.CapacityEventRecord{
					ID:          fmt.Sprintf("ev-%d-%s", now.UnixNano(), fp),
					Fingerprint: fp,
					EventType:   eventType,
					Provider:    prof.Provider,
					Profile:     prof.Name,
					QuotaGroup:  w.Group,
					WindowKind:  w.Kind,
					Severity:    string(newState),
					PayloadJSON: string(payloadBytes),
					EmittedAt:   now,
				}
				_ = m.deps.Store.PutCapacityEvent(evRecord)
				if m.deps.EventPublisher != nil {
					_ = m.deps.EventPublisher.Publish(evRecord)
				}
			}
		}
	}
	return resultState
}

// Status returns the current diagnostic status of the monitor and monitored resources.
func (m *Monitor) Status(ctx context.Context) (Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stat := Status{
		IsLeader:    m.isLeader,
		LeaderOwner: m.leaderOwner,
		Running:     m.running,
		LastTickAt:  m.lastTickAt,
	}

	if m.deps.Store != nil {
		states, err := m.deps.Store.ListCapacityStates()
		if err == nil {
			stat.MonitoredResources = len(states)
			obsList, _ := m.deps.Store.ListCapacityObservations()
			obsMap := make(map[string]store.ObservationRecord)
			for _, o := range obsList {
				k := fmt.Sprintf("%s:%s:%s:%s", o.Provider, o.Profile, o.QuotaGroup, o.WindowKind)
				obsMap[k] = o
			}

			for _, s := range states {
				rs := ResourceState{
					Key:            s.Key,
					Provider:       s.Provider,
					Profile:        s.Profile,
					QuotaGroup:     s.QuotaGroup,
					WindowKind:     s.WindowKind,
					State:          State(s.State),
					TransitionedAt: s.TransitionedAt,
				}
				if obs, ok := obsMap[s.Key]; ok {
					rs.RemainingPercent = obs.RemainingPercent
					rs.ResetDescription = obs.ResetDescription
					rs.LastObservedAt = obs.FetchedAt
				}
				stat.Resources = append(stat.Resources, rs)
			}
		}
	}

	return stat, nil
}
