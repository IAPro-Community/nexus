package nexus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/capacitymonitor"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// CapacityApplicationService is the thin Web/API boundary over the capacity
// monitor store. It owns no loop, no scheduler and no event bus: status is a
// read-model projection, preferences are validated persistence, and ack/snooze
// are human triage rows consumed by monitor delivery. Unknown stays UNKNOWN:
// absent rows project as empty lists, never as healthy.
type CapacityApplicationService struct {
	st *store.Store
}

// NewCapacityApplicationService builds the service over an opened store.
// A nil store fails every call closed (service unavailable, never fabricated).
func NewCapacityApplicationService(st *store.Store) *CapacityApplicationService {
	return &CapacityApplicationService{st: st}
}

func (s *CapacityApplicationService) requireStore() (*store.Store, error) {
	if s == nil || s.st == nil {
		return nil, errors.New("capacity store unavailable")
	}
	return s.st, nil
}

// CapacityResourceStatus is one monitored resource with triage attached.
type CapacityResourceStatus struct {
	Key               string     `json:"key"`
	Provider          string     `json:"provider"`
	Profile           string     `json:"profile"`
	QuotaGroup        string     `json:"quota_group"`
	WindowKind        string     `json:"window_kind"`
	State             string     `json:"state"`
	Fingerprint       string     `json:"fingerprint,omitempty"`
	Reason            string     `json:"reason,omitempty"`
	TransitionedAt    time.Time  `json:"transitioned_at"`
	LastObservedAt    time.Time  `json:"last_observed_at"`
	RemainingPercent  *float64   `json:"remaining_percent,omitempty"`
	ResetDescription  string     `json:"reset_description,omitempty"`
	Source            string     `json:"source,omitempty"`
	ObservationStatus string     `json:"observation_status,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	Acked             bool       `json:"acked"`
	SnoozedUntil      *time.Time `json:"snoozed_until,omitempty"`
	Pace              *capacitymonitor.PaceForecast `json:"pace,omitempty"`
}

// CapacityStatus is the GET /api/v1/capacity/status projection.
type CapacityStatus struct {
	Enabled            bool                      `json:"enabled"`
	MonitoredResources int                       `json:"monitored_resources"`
	Resources          []CapacityResourceStatus  `json:"resources"`
	LeaseOwner         string                    `json:"lease_owner,omitempty"`
	LeaseExpiresAt     *time.Time                `json:"lease_expires_at,omitempty"`
	Preferences        store.CapacityPreferences `json:"preferences"`
}

// Status projects observations+states+lease+preferences. Never invents health:
// with no rows the resource list is empty and Enabled reflects preferences.
func (s *CapacityApplicationService) Status(ctx context.Context) (CapacityStatus, error) {
	if err := ctx.Err(); err != nil {
		return CapacityStatus{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return CapacityStatus{}, err
	}
	prefs, err := st.GetCapacityPreferences()
	if err != nil {
		return CapacityStatus{}, err
	}
	out := CapacityStatus{Enabled: prefs.Enabled, Preferences: prefs}
	obs, err := st.ListCapacityObservations()
	if err != nil {
		return CapacityStatus{}, err
	}
	byKey := make(map[string]store.ObservationRecord, len(obs))
	for _, o := range obs {
		byKey[capacityStateKey(o.Provider, o.Profile, o.QuotaGroup, o.WindowKind)] = o
	}
	states, err := st.ListCapacityStates()
	if err != nil {
		return CapacityStatus{}, err
	}
	// Avoid a global sample cap starving other accounts. Forecast per
	// verified provider/profile/account/group/window and keep failures advisory.
	now := time.Now().UTC()
	for _, rec := range states {
		rs := CapacityResourceStatus{
			Key: rec.Key, Provider: rec.Provider, Profile: rec.Profile,
			QuotaGroup: rec.QuotaGroup, WindowKind: rec.WindowKind,
			State: rec.State, Reason: rec.Reason,
			TransitionedAt: rec.TransitionedAt, LastObservedAt: rec.UpdatedAt,
		}
		if o, ok := byKey[rec.Key]; ok {
			rs.RemainingPercent = o.RemainingPercent
			rs.ResetDescription = o.ResetDescription
			rs.Source = o.Source
			rs.ObservationStatus = o.Status
			if !o.ExpiresAt.IsZero() {
				expires := o.ExpiresAt
				rs.ExpiresAt = &expires
			}
			rs.LastObservedAt = o.FetchedAt
			if rec.AccountScope != "" && rec.AccountScope == o.AccountScope {
				if samples, paceErr := st.ListCapacityPaceSamplesForResource(o, now.Add(-24*time.Hour)); paceErr == nil {
					rs.Pace = capacitymonitor.ForecastForObservation(samples, o, now)
				}
			}
			fp := capacitymonitor.DeriveFingerprint(o.Provider, o.Profile, o.QuotaGroup, o.WindowKind, rec.State,
				capacitymonitor.ResetCycleKey(nil, o.ExpiresAt))
			rs.Fingerprint = fp
			if triage, ok, terr := st.GetCapacityAlertState(fp); terr == nil && ok {
				rs.Acked = triage.Acked
				rs.SnoozedUntil = triage.SnoozedUntil
			}
		}
		out.Resources = append(out.Resources, rs)
	}
	if out.Resources == nil {
		out.Resources = []CapacityResourceStatus{}
	}
	out.MonitoredResources = len(out.Resources)
	if lease, lerr := st.GetCapacityLease(); lerr == nil && lease != nil {
		out.LeaseOwner = lease.Owner
		exp := lease.ExpiresAt
		out.LeaseExpiresAt = &exp
	}
	return out, nil
}

// capacityStateKey mirrors capacitymonitor.stateKey (provider/profile/group/window).
func capacityStateKey(provider, profile, group, window string) string {
	return strings.Join([]string{provider, profile, group, window}, "/")
}

// CapacityHistoryEvent is one durable alert with triage attached.
type CapacityHistoryEvent struct {
	store.CapacityEventRecord
	Acked        bool       `json:"acked"`
	SnoozedUntil *time.Time `json:"snoozed_until,omitempty"`
}

// History returns recent durable alert events, newest first. Limit is clamped
// by the store (default 50, max 200).
func (s *CapacityApplicationService) History(ctx context.Context, limit int) ([]CapacityHistoryEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := s.requireStore()
	if err != nil {
		return nil, err
	}
	rows, err := st.ListCapacityEvents(limit)
	if err != nil {
		return nil, err
	}
	out := make([]CapacityHistoryEvent, 0, len(rows))
	for _, r := range rows {
		ev := CapacityHistoryEvent{CapacityEventRecord: r}
		if triage, ok, terr := st.GetCapacityAlertState(r.Fingerprint); terr == nil && ok {
			ev.Acked = triage.Acked
			ev.SnoozedUntil = triage.SnoozedUntil
		}
		out = append(out, ev)
	}
	return out, nil
}

// Preferences returns the persisted monitor preferences (defaults when fresh).
func (s *CapacityApplicationService) Preferences(ctx context.Context) (store.CapacityPreferences, error) {
	if err := ctx.Err(); err != nil {
		return store.CapacityPreferences{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return store.CapacityPreferences{}, err
	}
	return st.GetCapacityPreferences()
}

// UpdatePreferences validates and persists monitor preferences. Threshold
// ordering (warning > critical > exhausted, recovery above critical) is
// enforced so the UI cannot configure a hysteresis inversion.
func (s *CapacityApplicationService) UpdatePreferences(ctx context.Context, prefs store.CapacityPreferences) (store.CapacityPreferences, error) {
	if err := ctx.Err(); err != nil {
		return store.CapacityPreferences{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return store.CapacityPreferences{}, err
	}
	if !(prefs.WarningThreshold > prefs.CriticalThreshold && prefs.CriticalThreshold > prefs.ExhaustedThreshold) {
		return store.CapacityPreferences{}, fmt.Errorf("thresholds must order warning > critical > exhausted")
	}
	if prefs.RecoveryThreshold <= prefs.CriticalThreshold {
		return store.CapacityPreferences{}, fmt.Errorf("recovery must sit above critical")
	}
	if prefs.ReminderIntervalSeconds < 0 {
		return store.CapacityPreferences{}, fmt.Errorf("reminder interval must not be negative")
	}
	if prefs.QuietHoursEnabled {
		if err := validateDayTime(prefs.QuietHoursStartUTC); err != nil {
			return store.CapacityPreferences{}, fmt.Errorf("quiet hours start: %w", err)
		}
		if err := validateDayTime(prefs.QuietHoursEndUTC); err != nil {
			return store.CapacityPreferences{}, fmt.Errorf("quiet hours end: %w", err)
		}
	}
	prefs.UpdatedAt = time.Now().UTC()
	if err := st.PutCapacityPreferences(prefs); err != nil {
		return store.CapacityPreferences{}, err
	}
	return prefs, nil
}

func validateDayTime(v string) error {
	if _, err := time.Parse("15:04", strings.TrimSpace(v)); err != nil {
		return err
	}
	return nil
}

// Ack records human acknowledgement for one alert fingerprint. It suppresses
// repeats of the same fingerprint, never new states (those carry a new
// fingerprint and still alert).
func (s *CapacityApplicationService) Ack(ctx context.Context, fingerprint string) (store.AlertState, error) {
	if err := ctx.Err(); err != nil {
		return store.AlertState{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return store.AlertState{}, err
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return store.AlertState{}, errors.New("fingerprint is required")
	}
	cur, ok, err := st.GetCapacityAlertState(fingerprint)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return store.AlertState{}, err
	}
	now := time.Now().UTC()
	if !ok {
		cur = store.AlertState{Fingerprint: fingerprint}
	}
	cur.Acked = true
	cur.SnoozedUntil = nil
	cur.UpdatedAt = now
	if err := st.PutCapacityAlertState(cur); err != nil {
		return store.AlertState{}, err
	}
	return cur, nil
}

// Snooze suppresses delivery of one alert fingerprint until the given time
// (RFC3339, must be in the future, max 30 days). Zero/empty clears snooze.
func (s *CapacityApplicationService) Snooze(ctx context.Context, fingerprint string, until time.Time) (store.AlertState, error) {
	if err := ctx.Err(); err != nil {
		return store.AlertState{}, err
	}
	st, err := s.requireStore()
	if err != nil {
		return store.AlertState{}, err
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return store.AlertState{}, errors.New("fingerprint is required")
	}
	cur, ok, err := st.GetCapacityAlertState(fingerprint)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return store.AlertState{}, err
	}
	now := time.Now().UTC()
	if !ok {
		cur = store.AlertState{Fingerprint: fingerprint}
	}
	if until.IsZero() {
		cur.SnoozedUntil = nil
	} else {
		until = until.UTC()
		if !until.After(now) {
			return store.AlertState{}, errors.New("snooze time must be in the future")
		}
		if until.Sub(now) > 30*24*time.Hour {
			return store.AlertState{}, errors.New("snooze is capped at 30 days")
		}
		cur.SnoozedUntil = &until
	}
	cur.UpdatedAt = now
	if err := st.PutCapacityAlertState(cur); err != nil {
		return store.AlertState{}, err
	}
	return cur, nil
}
