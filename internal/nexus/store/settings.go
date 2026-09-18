package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GetSetting reads one entry from the global nexus_settings singleton table.
// A missing key returns ("", sql.ErrNoRows-equivalent ErrNotFound) so callers
// can fall back to compiled safe defaults without inventing state.
func (s *Store) GetSetting(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("setting key is required")
	}
	var value string
	err := s.db.QueryRow(`SELECT value FROM nexus_settings WHERE key=?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read setting %s: %w", key, err)
	}
	return value, nil
}

// PutSetting upserts one entry in the global nexus_settings singleton table.
// The write is idempotent: repeating the same key/value is a no-op in effect.
func (s *Store) PutSetting(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("setting key is required")
	}
	if value == "" {
		value = "{}"
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`INSERT INTO nexus_settings(key, value, updated_at) VALUES(?,?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, value, now)
	if err != nil {
		return fmt.Errorf("write setting %s: %w", key, err)
	}
	return nil
}
