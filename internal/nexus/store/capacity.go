package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// storeTimeFormat renders UTC timestamps fixed-width so lexicographic
// comparison inside SQL always matches chronological order. time.RFC3339Nano
// omits trailing zeros, which breaks string ordering across precisions
// (a second-exact stamp sorts AFTER a fractional one within the same second).
const storeTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

func formatStoreTime(t time.Time) string { return t.UTC().Format(storeTimeFormat) }

// parseStoreTime reads both fixed-width stamps and legacy RFC3339Nano rows.
func parseStoreTime(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }

// T2 persistence for the capacity monitor revival (see
// DEV/SPECS/CAPACITY_MONITOR_REVIVAL.md): observations, threshold states,
// single-leader lease, and user preferences. Event/receipt/recommendation
// persistence lands in T5 alongside delivery; their tables already exist in
// migration 0021 so later tasks add no DDL.

// ObservationRecord represents a stored quota telemetry observation.
type ObservationRecord struct {
	Provider         string    `json:"provider"`
	Profile          string    `json:"profile"`
	QuotaGroup       string    `json:"quota_group"`
	WindowKind       string    `json:"window_kind"`
	Status           string    `json:"status"`
	Source           string    `json:"source"`
	UsedPercent      *float64  `json:"used_percent,omitempty"`
	RemainingPercent *float64  `json:"remaining_percent,omitempty"`
	FetchedAt        time.Time `json:"fetched_at"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	ResetDescription string    `json:"reset_description,omitempty"`
	// AccountScope is the opaque scope key ("", when the source is unattributed).
	AccountScope string `json:"account_scope,omitempty"`
}

// CapacityStateRecord represents the evaluated threshold state for a resource.
type CapacityStateRecord struct {
	Key            string    `json:"key"`
	Provider       string    `json:"provider"`
	Profile        string    `json:"profile"`
	QuotaGroup     string    `json:"quota_group"`
	WindowKind     string    `json:"window_kind"`
	State          string    `json:"state"`
	Reason         string    `json:"reason"`
	TransitionedAt time.Time `json:"transitioned_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// AccountScope is the opaque scope key ("", when the source is unattributed).
	AccountScope string `json:"account_scope,omitempty"`
}

// MonitorLease represents single-leader ownership of the capacity monitor loop.
type MonitorLease struct {
	Owner      string    `json:"owner"`
	AcquiredAt time.Time `json:"acquired_at"`
	RenewedAt  time.Time `json:"renewed_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CapacityPreferences represents user preferences for the capacity monitor.
type CapacityPreferences struct {
	Enabled                     bool      `json:"enabled"`
	BackgroundEnabled           bool      `json:"background_enabled"`
	WarningThreshold            float64   `json:"warning_threshold"`
	CriticalThreshold           float64   `json:"critical_threshold"`
	ExhaustedThreshold          float64   `json:"exhausted_threshold"`
	RecoveryThreshold           float64   `json:"recovery_threshold"`
	ReminderIntervalSeconds     int       `json:"reminder_interval_seconds"`
	QuietHoursEnabled           bool      `json:"quiet_hours_enabled"`
	QuietHoursStartUTC          string    `json:"quiet_hours_start_utc"`
	QuietHoursEndUTC            string    `json:"quiet_hours_end_utc"`
	ExhaustedBypassQuietHours   bool      `json:"exhausted_bypass_quiet_hours"`
	BrowserNotificationsEnabled bool      `json:"browser_notifications_enabled"`
	NativeNotificationsEnabled  bool      `json:"native_notifications_enabled"`
	LockScreenSafeCopy          bool      `json:"lock_screen_safe_copy"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

// DefaultCapacityPreferences returns default safe monitor preferences.
// Background execution stays OFF until the user explicitly opts in.
func DefaultCapacityPreferences() CapacityPreferences {
	return CapacityPreferences{
		Enabled:                     true,
		BackgroundEnabled:           false,
		WarningThreshold:            20.0,
		CriticalThreshold:           10.0,
		ExhaustedThreshold:          0.0,
		RecoveryThreshold:           25.0,
		ReminderIntervalSeconds:     3600,
		QuietHoursEnabled:           false,
		QuietHoursStartUTC:          "22:00",
		QuietHoursEndUTC:            "08:00",
		ExhaustedBypassQuietHours:   true,
		BrowserNotificationsEnabled: true,
		NativeNotificationsEnabled:  true,
		LockScreenSafeCopy:          true,
		UpdatedAt:                   time.Now().UTC(),
	}
}

// UpsertCapacityObservation stores or updates a quota observation.
func (s *Store) UpsertCapacityObservation(record ObservationRecord) error {
	var expiresAtStr *string
	if !record.ExpiresAt.IsZero() {
		str := formatStoreTime(record.ExpiresAt)
		expiresAtStr = &str
	}
	fetchedAtStr := formatStoreTime(record.FetchedAt)

	_, err := s.db.Exec(`
		INSERT INTO capacity_observations(
			provider, profile, quota_group, window_kind, status, source,
			used_percent, remaining_percent, fetched_at, expires_at, reset_description, account_scope
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, profile, quota_group, window_kind) DO UPDATE SET
			status = excluded.status,
			source = excluded.source,
			used_percent = excluded.used_percent,
			remaining_percent = excluded.remaining_percent,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at,
			reset_description = excluded.reset_description,
			account_scope = excluded.account_scope
	`, record.Provider, record.Profile, record.QuotaGroup, record.WindowKind, record.Status, record.Source,
		record.UsedPercent, record.RemainingPercent, fetchedAtStr, expiresAtStr, record.ResetDescription, record.AccountScope)
	return err
}

// ListCapacityObservations returns all current observations.
func (s *Store) ListCapacityObservations() ([]ObservationRecord, error) {
	rows, err := s.db.Query(`
		SELECT provider, profile, quota_group, window_kind, status, source,
		       used_percent, remaining_percent, fetched_at, expires_at, reset_description, account_scope
		FROM capacity_observations
		ORDER BY provider, profile, quota_group, window_kind
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ObservationRecord
	for rows.Next() {
		var r ObservationRecord
		var fetchedAtStr string
		var expiresAtStr sql.NullString
		var usedPct, remPct sql.NullFloat64

		if err := rows.Scan(
			&r.Provider, &r.Profile, &r.QuotaGroup, &r.WindowKind, &r.Status, &r.Source,
			&usedPct, &remPct, &fetchedAtStr, &expiresAtStr, &r.ResetDescription, &r.AccountScope,
		); err != nil {
			return nil, err
		}

		if usedPct.Valid {
			r.UsedPercent = &usedPct.Float64
		}
		if remPct.Valid {
			r.RemainingPercent = &remPct.Float64
		}
		if t, err := parseStoreTime(fetchedAtStr); err == nil {
			r.FetchedAt = t
		}
		if expiresAtStr.Valid {
			if t, err := parseStoreTime(expiresAtStr.String); err == nil {
				r.ExpiresAt = t
			}
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// UpsertCapacityState stores or updates state machine evaluation for a resource.
func (s *Store) UpsertCapacityState(record CapacityStateRecord) error {
	transStr := formatStoreTime(record.TransitionedAt)
	updatedStr := formatStoreTime(record.UpdatedAt)

	_, err := s.db.Exec(`
		INSERT INTO capacity_states(
			key, provider, profile, quota_group, window_kind, state, reason, transitioned_at, updated_at, account_scope
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			provider = excluded.provider,
			profile = excluded.profile,
			quota_group = excluded.quota_group,
			window_kind = excluded.window_kind,
			state = excluded.state,
			reason = excluded.reason,
			transitioned_at = excluded.transitioned_at,
			updated_at = excluded.updated_at,
			account_scope = excluded.account_scope
	`, record.Key, record.Provider, record.Profile, record.QuotaGroup, record.WindowKind,
		record.State, record.Reason, transStr, updatedStr, record.AccountScope)
	return err
}

// GetCapacityState retrieves state for a specific key.
// A missing key returns sql.ErrNoRows so callers can treat absence as UNKNOWN
// without inventing state.
func (s *Store) GetCapacityState(key string) (CapacityStateRecord, error) {
	row := s.db.QueryRow(`
		SELECT key, provider, profile, quota_group, window_kind, state, reason, transitioned_at, updated_at, account_scope
		FROM capacity_states WHERE key = ?
	`, key)

	var r CapacityStateRecord
	var transStr, updatedStr string
	if err := row.Scan(&r.Key, &r.Provider, &r.Profile, &r.QuotaGroup, &r.WindowKind, &r.State, &r.Reason, &transStr, &updatedStr, &r.AccountScope); err != nil {
		return r, err
	}
	var err error
	if r.TransitionedAt, err = parseStoreTime(transStr); err != nil {
		return r, fmt.Errorf("corrupt state transitioned_at for %q: %w", key, err)
	}
	if r.UpdatedAt, err = parseStoreTime(updatedStr); err != nil {
		return r, fmt.Errorf("corrupt state updated_at for %q: %w", key, err)
	}
	return r, nil
}

// RefreshCapacityStateDetail updates reason/scope/updated_at without moving
// TransitionedAt: the transition moment stays honest when only the diagnostic
// cause or the account identity changes. A concurrently deleted row falls back
// to a full insert (a fresh row may honestly start now).
func (s *Store) RefreshCapacityStateDetail(rec CapacityStateRecord, reason, scope string, now time.Time) error {
	res, err := s.db.Exec(`UPDATE capacity_states SET reason = ?, account_scope = ?, updated_at = ? WHERE key = ?`,
		reason, scope, formatStoreTime(now), rec.Key)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		rec.Reason = reason
		rec.AccountScope = scope
		rec.TransitionedAt = now
		rec.UpdatedAt = now
		return s.UpsertCapacityState(rec)
	}
	return nil
}

// ListCapacityStates returns all recorded capacity states.
func (s *Store) ListCapacityStates() ([]CapacityStateRecord, error) {
	rows, err := s.db.Query(`
		SELECT key, provider, profile, quota_group, window_kind, state, reason, transitioned_at, updated_at, account_scope
		FROM capacity_states
		ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CapacityStateRecord
	for rows.Next() {
		var r CapacityStateRecord
		var transStr, updatedStr string
		if err := rows.Scan(&r.Key, &r.Provider, &r.Profile, &r.QuotaGroup, &r.WindowKind, &r.State, &r.Reason, &transStr, &updatedStr, &r.AccountScope); err != nil {
			return nil, err
		}
		// Diagnostics resilience: one corrupt row must not blank the whole
		// listing (decisions use Get, which fails loudly instead).
		var err error
		if r.TransitionedAt, err = parseStoreTime(transStr); err != nil {
			continue
		}
		if r.UpdatedAt, err = parseStoreTime(updatedStr); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// AcquireCapacityLease attempts to acquire or take over the single monitor lease
// in ONE atomic statement: a live lease owned by someone else is never stolen,
// an expired one (or one we already own) is taken over. The single statement
// closes the SELECT-then-UPDATE race where two concurrent acquirers could both
// believe they hold the lease.
func (s *Store) AcquireCapacityLease(owner string, now time.Time, ttl time.Duration) (MonitorLease, bool, error) {
	if ttl <= 0 {
		return MonitorLease{}, false, errors.New("lease ttl must be positive")
	}
	nowStr := formatStoreTime(now)
	expiresStr := formatStoreTime(now.Add(ttl))

	res, err := s.db.Exec(`
		INSERT INTO capacity_monitor_lease(id, owner, acquired_at, renewed_at, expires_at)
		VALUES(1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			owner = excluded.owner,
			acquired_at = CASE WHEN capacity_monitor_lease.owner = excluded.owner
				THEN capacity_monitor_lease.acquired_at ELSE excluded.acquired_at END,
			renewed_at = excluded.renewed_at,
			expires_at = excluded.expires_at
		WHERE capacity_monitor_lease.owner = excluded.owner
		   OR capacity_monitor_lease.expires_at <= excluded.renewed_at
	`, owner, nowStr, nowStr, expiresStr)
	if err != nil {
		return MonitorLease{}, false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return MonitorLease{}, false, err
	}
	cur, err := s.GetCapacityLease()
	if err != nil {
		return MonitorLease{}, false, err
	}
	if cur == nil {
		return MonitorLease{}, false, errors.New("lease vanished after acquire")
	}
	// Re-read instead of echoing inputs so self re-acquire reports the true
	// preserved AcquiredAt rather than now.
	return *cur, affected == 1, nil
}

// RenewCapacityLease refreshes the lease if owned by the caller.
func (s *Store) RenewCapacityLease(owner string, now time.Time, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("lease ttl must be positive")
	}
	nowStr := formatStoreTime(now)
	expiresStr := formatStoreTime(now.Add(ttl))

	res, err := s.db.Exec(`
		UPDATE capacity_monitor_lease
		SET renewed_at = ?, expires_at = ?
		WHERE id = 1 AND owner = ?
	`, nowStr, expiresStr, owner)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("lease renewal failed: lease not held by %s", owner)
	}
	return nil
}

// ReleaseCapacityLease frees the lease. Releasing a lease we do not own is a
// no-op by the owner guard, never an error.
func (s *Store) ReleaseCapacityLease(owner string) error {
	_, err := s.db.Exec(`DELETE FROM capacity_monitor_lease WHERE id = 1 AND owner = ?`, owner)
	return err
}

// GetCapacityLease returns the current monitor lease, or nil when none exists.
func (s *Store) GetCapacityLease() (*MonitorLease, error) {
	var lease MonitorLease
	var acquired, renewed, expires string
	if err := s.db.QueryRow(`SELECT owner, acquired_at, renewed_at, expires_at FROM capacity_monitor_lease WHERE id = 1`).Scan(&lease.Owner, &acquired, &renewed, &expires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var err error
	if lease.AcquiredAt, err = parseStoreTime(acquired); err != nil {
		return nil, fmt.Errorf("corrupt lease acquired_at: %w", err)
	}
	if lease.RenewedAt, err = parseStoreTime(renewed); err != nil {
		return nil, fmt.Errorf("corrupt lease renewed_at: %w", err)
	}
	if lease.ExpiresAt, err = parseStoreTime(expires); err != nil {
		return nil, fmt.Errorf("corrupt lease expires_at: %w", err)
	}
	return &lease, nil
}

// GetCapacityPreferences retrieves user monitor preferences, falling back to
// compiled safe defaults when never configured.
func (s *Store) GetCapacityPreferences() (CapacityPreferences, error) {
	row := s.db.QueryRow(`
		SELECT enabled, background_enabled, warning_threshold, critical_threshold, exhausted_threshold, recovery_threshold,
		       reminder_interval_seconds, quiet_hours_enabled, quiet_hours_start_utc, quiet_hours_end_utc,
		       exhausted_bypass_quiet_hours, browser_notifications_enabled, native_notifications_enabled, lock_screen_safe_copy, updated_at
		FROM capacity_preferences WHERE id = 1
	`)

	var p CapacityPreferences
	var enabled, bg, quiet, bypass, browser, native, lock int
	var updatedStr string
	err := row.Scan(
		&enabled, &bg, &p.WarningThreshold, &p.CriticalThreshold, &p.ExhaustedThreshold, &p.RecoveryThreshold,
		&p.ReminderIntervalSeconds, &quiet, &p.QuietHoursStartUTC, &p.QuietHoursEndUTC,
		&bypass, &browser, &native, &lock, &updatedStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return DefaultCapacityPreferences(), nil
	} else if err != nil {
		return p, err
	}

	p.Enabled = enabled == 1
	p.BackgroundEnabled = bg == 1
	p.QuietHoursEnabled = quiet == 1
	p.ExhaustedBypassQuietHours = bypass == 1
	p.BrowserNotificationsEnabled = browser == 1
	p.NativeNotificationsEnabled = native == 1
	p.LockScreenSafeCopy = lock == 1
	p.UpdatedAt, _ = parseStoreTime(updatedStr)
	return p, nil
}

// PutCapacityPreferences persists user preferences (singleton row, idempotent).
func (s *Store) PutCapacityPreferences(prefs CapacityPreferences) error {
	updatedStr := formatStoreTime(prefs.UpdatedAt)
	if prefs.UpdatedAt.IsZero() {
		updatedStr = formatStoreTime(time.Now())
	}
	bToInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}

	_, err := s.db.Exec(`
		INSERT INTO capacity_preferences(
			id, enabled, background_enabled, warning_threshold, critical_threshold, exhausted_threshold, recovery_threshold,
			reminder_interval_seconds, quiet_hours_enabled, quiet_hours_start_utc, quiet_hours_end_utc,
			exhausted_bypass_quiet_hours, browser_notifications_enabled, native_notifications_enabled, lock_screen_safe_copy, updated_at
		) VALUES(1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			background_enabled = excluded.background_enabled,
			warning_threshold = excluded.warning_threshold,
			critical_threshold = excluded.critical_threshold,
			exhausted_threshold = excluded.exhausted_threshold,
			recovery_threshold = excluded.recovery_threshold,
			reminder_interval_seconds = excluded.reminder_interval_seconds,
			quiet_hours_enabled = excluded.quiet_hours_enabled,
			quiet_hours_start_utc = excluded.quiet_hours_start_utc,
			quiet_hours_end_utc = excluded.quiet_hours_end_utc,
			exhausted_bypass_quiet_hours = excluded.exhausted_bypass_quiet_hours,
			browser_notifications_enabled = excluded.browser_notifications_enabled,
			native_notifications_enabled = excluded.native_notifications_enabled,
			lock_screen_safe_copy = excluded.lock_screen_safe_copy,
			updated_at = excluded.updated_at
	`, bToInt(prefs.Enabled), bToInt(prefs.BackgroundEnabled), prefs.WarningThreshold, prefs.CriticalThreshold,
		prefs.ExhaustedThreshold, prefs.RecoveryThreshold, prefs.ReminderIntervalSeconds, bToInt(prefs.QuietHoursEnabled),
		prefs.QuietHoursStartUTC, prefs.QuietHoursEndUTC, bToInt(prefs.ExhaustedBypassQuietHours),
		bToInt(prefs.BrowserNotificationsEnabled), bToInt(prefs.NativeNotificationsEnabled), bToInt(prefs.LockScreenSafeCopy), updatedStr)
	return err
}

// PruneCapacityHistory removes events and delivery receipts older than before.
// Observations and threshold states are current-value snapshots (upserted, never
// historized), so retention applies only to the append-only event tables.
func (s *Store) PruneCapacityHistory(before time.Time) error {
	beforeStr := formatStoreTime(before)
	if _, err := s.db.Exec(`DELETE FROM capacity_events WHERE emitted_at < ?`, beforeStr); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM capacity_delivery_receipts WHERE delivered_at < ?`, beforeStr); err != nil {
		return err
	}
	return nil
}
