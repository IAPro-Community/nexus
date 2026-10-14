package store

import (
	"errors"
	"math"
	"strings"
	"time"
)

// CapacityPaceSample is read-only quota telemetry, never billing or authoritative
// capacity. The account scope and absolute reset separate identities/cycles.
type CapacityPaceSample struct {
	Provider    string    `json:"provider"`
	Profile     string    `json:"profile"`
	AccountScope string   `json:"-"`
	QuotaGroup  string    `json:"quota_group"`
	WindowKind  string    `json:"window_kind"`
	ResetAt     time.Time `json:"reset_at"`
	ObservedAt  time.Time `json:"observed_at"`
	UsedPercent float64   `json:"used_percent"`
	Source      string    `json:"source"`
}

// RecordCapacityPaceSample is opt-in through the existing monitor, not a new
// probe. Unknown identities, stale measurements and invalid reset times do not
// become forecast evidence. A 5-minute bucket bounds writes and history size.
func (s *Store) RecordCapacityPaceSample(v CapacityPaceSample) error {
	if s == nil || s.db == nil {
		return errors.New("capacity pace: store unavailable")
	}
	if strings.TrimSpace(v.Provider) == "" || strings.TrimSpace(v.Profile) == "" ||
		strings.TrimSpace(v.AccountScope) == "" || strings.TrimSpace(v.WindowKind) == "" ||
		v.ObservedAt.IsZero() || v.ResetAt.IsZero() || !v.ResetAt.After(v.ObservedAt) ||
		math.IsNaN(v.UsedPercent) || math.IsInf(v.UsedPercent, 0) ||
		v.UsedPercent < 0 || v.UsedPercent > 100 {
		return nil // no trustworthy observation; absence is intentional
	}
	bucket := v.ObservedAt.UTC().Truncate(5 * time.Minute)
	_, err := s.db.Exec(`INSERT INTO capacity_pace_samples (
		provider, profile, account_scope, quota_group, window_kind, reset_at,
		observed_at, used_percent, source
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(provider, profile, account_scope, quota_group, window_kind, reset_at, observed_at)
	DO UPDATE SET used_percent=excluded.used_percent, source=excluded.source`,
		v.Provider, v.Profile, v.AccountScope, v.QuotaGroup, v.WindowKind,
		formatStoreTime(v.ResetAt), formatStoreTime(bucket), v.UsedPercent, v.Source)
	return err
}

// ListCapacityPaceSamples reads at most a bounded amount of the retained history.
// Callers must match provider/profile/scope/group/window/reset before forecasting.
func (s *Store) ListCapacityPaceSamples(since time.Time) ([]CapacityPaceSample, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("capacity pace: store unavailable")
	}
	rows, err := s.db.Query(`SELECT provider, profile, account_scope, quota_group, window_kind,
		reset_at, observed_at, used_percent, source FROM capacity_pace_samples
		WHERE observed_at >= ? ORDER BY observed_at DESC LIMIT 5000`, formatStoreTime(since))
	if err != nil { return nil, err }
	defer rows.Close()
	result := []CapacityPaceSample{}
	for rows.Next() {
		var v CapacityPaceSample
		var reset, observed string
		if err := rows.Scan(&v.Provider, &v.Profile, &v.AccountScope, &v.QuotaGroup,
			&v.WindowKind, &reset, &observed, &v.UsedPercent, &v.Source); err != nil {
			return nil, err
		}
		v.ResetAt, err = parseStoreTime(reset)
		if err != nil { return nil, err }
		v.ObservedAt, err = parseStoreTime(observed)
		if err != nil { return nil, err }
		result = append(result, v)
	}
	return result, rows.Err()
}

// PruneCapacityPaceSamples bounds disk usage without touching current quota.
func (s *Store) PruneCapacityPaceSamples(before time.Time) (int64, error) {
	if s == nil || s.db == nil { return 0, errors.New("capacity pace: store unavailable") }
	result, err := s.db.Exec(`DELETE FROM capacity_pace_samples WHERE observed_at < ?`,
		formatStoreTime(before))
	if err != nil { return 0, err }
	return result.RowsAffected()
}
