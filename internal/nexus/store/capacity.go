package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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
}

// MonitorLease represents single-leader ownership of the capacity monitor loop.
type MonitorLease struct {
	Owner      string    `json:"owner"`
	AcquiredAt time.Time `json:"acquired_at"`
	RenewedAt  time.Time `json:"renewed_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CapacityEventRecord represents an immutable canonical event emitted by the monitor.
type CapacityEventRecord struct {
	ID          string    `json:"id"`
	Fingerprint string    `json:"fingerprint"`
	EventType   string    `json:"event_type"`
	Provider    string    `json:"provider"`
	Profile     string    `json:"profile"`
	QuotaGroup  string    `json:"quota_group"`
	WindowKind  string    `json:"window_kind"`
	Severity    string    `json:"severity"`
	PayloadJSON string    `json:"payload_json"`
	EmittedAt   time.Time `json:"emitted_at"`
}

// DeliveryReceipt records a delivery attempt on a specific notification channel.
type DeliveryReceipt struct {
	ID          int64     `json:"id,omitempty"`
	AlertID     string    `json:"alert_id"`
	Channel     string    `json:"channel"`
	Fingerprint string    `json:"fingerprint"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	DeliveredAt time.Time `json:"delivered_at"`
}

// CapacityRecommendationRecord stores the latest advisory recommendation for a fingerprint.
type CapacityRecommendationRecord struct {
	Fingerprint        string    `json:"fingerprint"`
	Provider           string    `json:"provider"`
	Profile            string    `json:"profile"`
	QuotaGroup         string    `json:"quota_group"`
	RecommendationJSON string    `json:"recommendation_json"`
	CreatedAt          time.Time `json:"created_at"`
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
		str := record.ExpiresAt.UTC().Format(time.RFC3339Nano)
		expiresAtStr = &str
	}
	fetchedAtStr := record.FetchedAt.UTC().Format(time.RFC3339Nano)

	_, err := s.db.Exec(`
		INSERT INTO capacity_observations(
			provider, profile, quota_group, window_kind, status, source,
			used_percent, remaining_percent, fetched_at, expires_at, reset_description
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, profile, quota_group, window_kind) DO UPDATE SET
			status = excluded.status,
			source = excluded.source,
			used_percent = excluded.used_percent,
			remaining_percent = excluded.remaining_percent,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at,
			reset_description = excluded.reset_description
	`, record.Provider, record.Profile, record.QuotaGroup, record.WindowKind, record.Status, record.Source,
		record.UsedPercent, record.RemainingPercent, fetchedAtStr, expiresAtStr, record.ResetDescription)
	return err
}

// ListCapacityObservations returns all current observations.
func (s *Store) ListCapacityObservations() ([]ObservationRecord, error) {
	rows, err := s.db.Query(`
		SELECT provider, profile, quota_group, window_kind, status, source,
		       used_percent, remaining_percent, fetched_at, expires_at, reset_description
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
			&usedPct, &remPct, &fetchedAtStr, &expiresAtStr, &r.ResetDescription,
		); err != nil {
			return nil, err
		}

		if usedPct.Valid {
			r.UsedPercent = &usedPct.Float64
		}
		if remPct.Valid {
			r.RemainingPercent = &remPct.Float64
		}
		if t, err := time.Parse(time.RFC3339Nano, fetchedAtStr); err == nil {
			r.FetchedAt = t
		}
		if expiresAtStr.Valid {
			if t, err := time.Parse(time.RFC3339Nano, expiresAtStr.String); err == nil {
				r.ExpiresAt = t
			}
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// UpsertCapacityState stores or updates state machine evaluation for a resource.
func (s *Store) UpsertCapacityState(record CapacityStateRecord) error {
	transStr := record.TransitionedAt.UTC().Format(time.RFC3339Nano)
	updatedStr := record.UpdatedAt.UTC().Format(time.RFC3339Nano)

	_, err := s.db.Exec(`
		INSERT INTO capacity_states(
			key, provider, profile, quota_group, window_kind, state, reason, transitioned_at, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			provider = excluded.provider,
			profile = excluded.profile,
			quota_group = excluded.quota_group,
			window_kind = excluded.window_kind,
			state = excluded.state,
			reason = excluded.reason,
			transitioned_at = excluded.transitioned_at,
			updated_at = excluded.updated_at
	`, record.Key, record.Provider, record.Profile, record.QuotaGroup, record.WindowKind,
		record.State, record.Reason, transStr, updatedStr)
	return err
}

// GetCapacityState retrieves state for a specific key.
func (s *Store) GetCapacityState(key string) (CapacityStateRecord, error) {
	row := s.db.QueryRow(`
		SELECT key, provider, profile, quota_group, window_kind, state, reason, transitioned_at, updated_at
		FROM capacity_states WHERE key = ?
	`, key)

	var r CapacityStateRecord
	var transStr, updatedStr string
	if err := row.Scan(&r.Key, &r.Provider, &r.Profile, &r.QuotaGroup, &r.WindowKind, &r.State, &r.Reason, &transStr, &updatedStr); err != nil {
		return r, err
	}
	r.TransitionedAt, _ = time.Parse(time.RFC3339Nano, transStr)
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
	return r, nil
}

// ListCapacityStates returns all recorded capacity states.
func (s *Store) ListCapacityStates() ([]CapacityStateRecord, error) {
	rows, err := s.db.Query(`
		SELECT key, provider, profile, quota_group, window_kind, state, reason, transitioned_at, updated_at
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
		if err := rows.Scan(&r.Key, &r.Provider, &r.Profile, &r.QuotaGroup, &r.WindowKind, &r.State, &r.Reason, &transStr, &updatedStr); err != nil {
			return nil, err
		}
		r.TransitionedAt, _ = time.Parse(time.RFC3339Nano, transStr)
		r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
		records = append(records, r)
	}
	return records, rows.Err()
}

// AcquireCapacityLease attempts to acquire or take over the single monitor lease.
func (s *Store) AcquireCapacityLease(owner string, now time.Time, ttl time.Duration) (MonitorLease, bool, error) {
	nowStr := now.UTC().Format(time.RFC3339Nano)
	expiresStr := now.Add(ttl).UTC().Format(time.RFC3339Nano)

	tx, err := s.db.Begin()
	if err != nil {
		return MonitorLease{}, false, err
	}
	defer tx.Rollback()

	var currentOwner string
	var currentExpiresStr string
	err = tx.QueryRow(`SELECT owner, expires_at FROM capacity_monitor_lease WHERE id = 1`).Scan(&currentOwner, &currentExpiresStr)
	if errors.Is(err, sql.ErrNoRows) {
		// Insert new lease
		_, err := tx.Exec(`
			INSERT INTO capacity_monitor_lease(id, owner, acquired_at, renewed_at, expires_at)
			VALUES(1, ?, ?, ?, ?)
		`, owner, nowStr, nowStr, expiresStr)
		if err != nil {
			return MonitorLease{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return MonitorLease{}, false, err
		}
		return MonitorLease{
			Owner:      owner,
			AcquiredAt: now,
			RenewedAt:  now,
			ExpiresAt:  now.Add(ttl),
		}, true, nil
	} else if err != nil {
		return MonitorLease{}, false, err
	}

	currentExpires, _ := time.Parse(time.RFC3339Nano, currentExpiresStr)
	// If existing lease is still valid and owned by someone else
	if currentOwner != owner && now.Before(currentExpires) {
		return MonitorLease{
			Owner:     currentOwner,
			ExpiresAt: currentExpires,
		}, false, nil
	}

	// Lease expired or already owned: take over / update
	_, err = tx.Exec(`
		UPDATE capacity_monitor_lease
		SET owner = ?, acquired_at = CASE WHEN owner = ? THEN acquired_at ELSE ? END, renewed_at = ?, expires_at = ?
		WHERE id = 1
	`, owner, owner, nowStr, nowStr, expiresStr)
	if err != nil {
		return MonitorLease{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return MonitorLease{}, false, err
	}

	return MonitorLease{
		Owner:      owner,
		AcquiredAt: now,
		RenewedAt:  now,
		ExpiresAt:  now.Add(ttl),
	}, true, nil
}

// RenewCapacityLease refreshes the lease if owned by the caller.
func (s *Store) RenewCapacityLease(owner string, now time.Time, ttl time.Duration) error {
	nowStr := now.UTC().Format(time.RFC3339Nano)
	expiresStr := now.Add(ttl).UTC().Format(time.RFC3339Nano)

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

// ReleaseCapacityLease frees the lease.
func (s *Store) ReleaseCapacityLease(owner string) error {
	_, err := s.db.Exec(`DELETE FROM capacity_monitor_lease WHERE id = 1 AND owner = ?`, owner)
	return err
}

// PutCapacityEvent persists a canonical capacity event.
func (s *Store) PutCapacityEvent(event CapacityEventRecord) error {
	var existing string
	if err := s.db.QueryRow(`SELECT id FROM capacity_events WHERE fingerprint = ? LIMIT 1`, event.Fingerprint).Scan(&existing); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	emittedStr := event.EmittedAt.UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`
		INSERT INTO capacity_events(id, fingerprint, event_type, provider, profile, quota_group, window_kind, severity, payload_json, emitted_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, event.ID, event.Fingerprint, event.EventType, event.Provider, event.Profile,
		event.QuotaGroup, event.WindowKind, event.Severity, event.PayloadJSON, emittedStr)
	return err
}

// FindCapacityEvent retrieves the most recent event matching a fingerprint.
func (s *Store) FindCapacityEvent(fingerprint string) (*CapacityEventRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, fingerprint, event_type, provider, profile, quota_group, window_kind, severity, payload_json, emitted_at
		FROM capacity_events
		WHERE fingerprint = ?
		ORDER BY emitted_at DESC LIMIT 1
	`, fingerprint)

	var ev CapacityEventRecord
	var emittedStr string
	if err := row.Scan(&ev.ID, &ev.Fingerprint, &ev.EventType, &ev.Provider, &ev.Profile,
		&ev.QuotaGroup, &ev.WindowKind, &ev.Severity, &ev.PayloadJSON, &emittedStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ev.EmittedAt, _ = time.Parse(time.RFC3339Nano, emittedStr)
	return &ev, nil
}

// ListCapacityEvents retrieves recent events in descending emission order.
func (s *Store) ListCapacityEvents(limit int) ([]CapacityEventRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`
		SELECT id, fingerprint, event_type, provider, profile, quota_group, window_kind, severity, payload_json, emitted_at
		FROM capacity_events
		ORDER BY emitted_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CapacityEventRecord
	for rows.Next() {
		var ev CapacityEventRecord
		var emittedStr string
		if err := rows.Scan(&ev.ID, &ev.Fingerprint, &ev.EventType, &ev.Provider, &ev.Profile,
			&ev.QuotaGroup, &ev.WindowKind, &ev.Severity, &ev.PayloadJSON, &emittedStr); err != nil {
			return nil, err
		}
		ev.EmittedAt, _ = time.Parse(time.RFC3339Nano, emittedStr)
		events = append(events, ev)
	}
	return events, rows.Err()
}

// PutDeliveryReceipt records delivery status for an alert.
func (s *Store) PutDeliveryReceipt(receipt DeliveryReceipt) error {
	deliveredStr := receipt.DeliveredAt.UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`
		INSERT INTO capacity_delivery_receipts(alert_id, channel, fingerprint, status, error, delivered_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, receipt.AlertID, receipt.Channel, receipt.Fingerprint, receipt.Status, receipt.Error, deliveredStr)
	return err
}

// ListDeliveryReceipts returns all delivery receipts for an alert ID.
func (s *Store) ListDeliveryReceipts(alertID string) ([]DeliveryReceipt, error) {
	rows, err := s.db.Query(`
		SELECT id, alert_id, channel, fingerprint, status, error, delivered_at
		FROM capacity_delivery_receipts
		WHERE alert_id = ?
		ORDER BY delivered_at ASC
	`, alertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DeliveryReceipt
	for rows.Next() {
		var rec DeliveryReceipt
		var deliveredStr string
		if err := rows.Scan(&rec.ID, &rec.AlertID, &rec.Channel, &rec.Fingerprint, &rec.Status, &rec.Error, &deliveredStr); err != nil {
			return nil, err
		}
		rec.DeliveredAt, _ = time.Parse(time.RFC3339Nano, deliveredStr)
		list = append(list, rec)
	}
	return list, rows.Err()
}

// PutCapacityRecommendation stores an explainable recommendation.
func (s *Store) PutCapacityRecommendation(rec CapacityRecommendationRecord) error {
	createdStr := rec.CreatedAt.UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`
		INSERT INTO capacity_recommendations(fingerprint, provider, profile, quota_group, recommendation_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(fingerprint) DO UPDATE SET
			recommendation_json = excluded.recommendation_json,
			created_at = excluded.created_at
	`, rec.Fingerprint, rec.Provider, rec.Profile, rec.QuotaGroup, rec.RecommendationJSON, createdStr)
	return err
}

// GetCapacityRecommendation retrieves a stored recommendation.
func (s *Store) GetCapacityRecommendation(fingerprint string) (*CapacityRecommendationRecord, error) {
	row := s.db.QueryRow(`
		SELECT fingerprint, provider, profile, quota_group, recommendation_json, created_at
		FROM capacity_recommendations WHERE fingerprint = ?
	`, fingerprint)

	var rec CapacityRecommendationRecord
	var createdStr string
	if err := row.Scan(&rec.Fingerprint, &rec.Provider, &rec.Profile, &rec.QuotaGroup, &rec.RecommendationJSON, &createdStr); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
	return &rec, nil
}

// GetCapacityPreferences retrieves user monitor preferences, falling back to defaults if not set.
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
	p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
	return p, nil
}

// PutCapacityPreferences persists user preferences.
func (s *Store) PutCapacityPreferences(prefs CapacityPreferences) error {
	updatedStr := prefs.UpdatedAt.UTC().Format(time.RFC3339Nano)
	if prefs.UpdatedAt.IsZero() {
		updatedStr = time.Now().UTC().Format(time.RFC3339Nano)
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

// PruneCapacityHistory removes events and delivery receipts older than before timestamp.
func (s *Store) PruneCapacityHistory(before time.Time) error {
	beforeStr := before.UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`DELETE FROM capacity_events WHERE emitted_at < ?`, beforeStr); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM capacity_delivery_receipts WHERE delivered_at < ?`, beforeStr); err != nil {
		return err
	}
	return nil
}
