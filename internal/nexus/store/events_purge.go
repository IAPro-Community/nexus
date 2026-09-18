package store

import (
	"time"
)

// PurgeExpiredEventMetadata removes activity events older than the provided
// cutoff. The operation is idempotent: running it repeatedly with the same
// cutoff is safe and harmless.
func (s *Store) PurgeExpiredEventMetadata(cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, nil
	}
	res, err := s.db.Exec(`DELETE FROM events_metadata WHERE ts<?`, cutoff.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PurgeExpiredEventsSince is the public API used by the conductor startup
// path: it removes events older than 90 days from the given reference time.
func (s *Store) PurgeExpiredEventsSince(now time.Time, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	return s.PurgeExpiredEventMetadata(cutoff)
}
