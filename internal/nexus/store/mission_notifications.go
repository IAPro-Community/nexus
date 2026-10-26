package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	MissionNotificationPending    = "PENDING"
	MissionNotificationDelivered  = "DELIVERED"
	MissionNotificationSuperseded = "SUPERSEDED"
	MissionNotificationNative     = "native"
)

// MissionNotificationIntent contains only a safe, display-ready notification
// payload. It records delivery intent, never Mission state or user prompts.
type MissionNotificationIntent struct {
	Key           string
	RunID         string
	ProjectID     string
	SourceID      string
	SourceVersion int
	Kind          string
	Title         string
	Body          string
	CreatedAt     time.Time
}

// MissionNotificationDelivery is one retryable transport attempt state.
type MissionNotificationDelivery struct {
	IntentKey     string
	RunID         string
	Channel       string
	Title         string
	Body          string
	Status        string
	Attempts      int
	LastError     string
	NextAttemptAt time.Time
}

// ObserveMissionNotification records a cursor for every MissionRun state
// transition and atomically enqueues eligible native delivery. The first
// observation establishes a baseline so opening Nexus does not notify about
// all historic terminal Missions.
func (s *Store) ObserveMissionNotification(intent MissionNotificationIntent, eligible bool, now time.Time) (bool, error) {
	if strings.TrimSpace(intent.RunID) == "" || strings.TrimSpace(intent.Key) == "" {
		return false, fmt.Errorf("mission notification run and key are required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if intent.CreatedAt.IsZero() {
		intent.CreatedAt = now
	}
	if intent.SourceID == "" {
		intent.SourceID = intent.RunID
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin mission notification observation: %w", err)
	}
	defer tx.Rollback()

	var previous string
	err = tx.QueryRow(`SELECT last_key FROM mission_notification_cursors WHERE run_id=?`, intent.RunID).Scan(&previous)
	if err == sql.ErrNoRows {
		_, err = tx.Exec(`INSERT INTO mission_notification_cursors(run_id,last_key,updated_at) VALUES(?,?,?)`,
			intent.RunID, intent.Key, formatStoreTime(now))
		if err != nil {
			return false, fmt.Errorf("establish mission notification baseline: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read mission notification cursor: %w", err)
	}
	if previous == intent.Key {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if _, err := tx.Exec(`UPDATE mission_notification_cursors SET last_key=?,updated_at=? WHERE run_id=?`,
		intent.Key, formatStoreTime(now), intent.RunID); err != nil {
		return false, fmt.Errorf("advance mission notification cursor: %w", err)
	}
	if _, err := tx.Exec(`UPDATE mission_notification_deliveries SET status=?,last_error='superseded by newer Mission state',updated_at=?
		WHERE channel=? AND status=? AND intent_key IN (SELECT intent_key FROM mission_notification_intents WHERE run_id=? AND intent_key<>?)`,
		MissionNotificationSuperseded, formatStoreTime(now), MissionNotificationNative, MissionNotificationPending, intent.RunID, intent.Key); err != nil {
		return false, fmt.Errorf("supersede stale mission notifications: %w", err)
	}
	if eligible {
		if strings.TrimSpace(intent.Kind) == "" || strings.TrimSpace(intent.Title) == "" || strings.TrimSpace(intent.Body) == "" {
			return false, fmt.Errorf("eligible mission notification requires kind, title and body")
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO mission_notification_intents
			(intent_key,run_id,project_id,source_id,source_version,kind,title,body,created_at)
			VALUES(?,?,?,?,?,?,?,?,?)`, intent.Key, intent.RunID, intent.ProjectID, intent.SourceID,
			intent.SourceVersion, intent.Kind, intent.Title, intent.Body, formatStoreTime(intent.CreatedAt)); err != nil {
			return false, fmt.Errorf("persist mission notification intent: %w", err)
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO mission_notification_deliveries
			(intent_key,channel,status,attempts,last_error,next_attempt_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
			intent.Key, MissionNotificationNative, MissionNotificationPending, 0, "", formatStoreTime(now), formatStoreTime(now)); err != nil {
			return false, fmt.Errorf("enqueue mission notification delivery: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit mission notification observation: %w", err)
	}
	return eligible, nil
}

// ListPendingMissionNotificationDeliveries returns due transport work in FIFO
// order. It intentionally exposes no event payload beyond the safe copy.
func (s *Store) ListPendingMissionNotificationDeliveries(now time.Time, limit int) ([]MissionNotificationDelivery, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT d.intent_key,i.run_id,d.channel,i.title,i.body,d.status,d.attempts,d.last_error,d.next_attempt_at
		FROM mission_notification_deliveries d
		JOIN mission_notification_intents i ON i.intent_key=d.intent_key
		WHERE d.status=? AND d.next_attempt_at<=?
		ORDER BY d.next_attempt_at ASC,i.created_at ASC LIMIT ?`, MissionNotificationPending, formatStoreTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("list pending mission notification deliveries: %w", err)
	}
	defer rows.Close()
	out := make([]MissionNotificationDelivery, 0)
	for rows.Next() {
		var delivery MissionNotificationDelivery
		var next string
		if err := rows.Scan(&delivery.IntentKey, &delivery.RunID, &delivery.Channel, &delivery.Title,
			&delivery.Body, &delivery.Status, &delivery.Attempts, &delivery.LastError, &next); err != nil {
			return nil, err
		}
		delivery.NextAttemptAt, err = parseStoreTime(next)
		if err != nil {
			return nil, fmt.Errorf("parse mission notification retry time: %w", err)
		}
		out = append(out, delivery)
	}
	return out, rows.Err()
}

// RecordMissionNotificationAttempt persists success or retry state for one
// channel. A failure stays pending with an explicit retry time.
func (s *Store) RecordMissionNotificationAttempt(key, channel, status, safeError string, nextAttemptAt, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if status != MissionNotificationPending && status != MissionNotificationDelivered {
		return fmt.Errorf("unsupported mission notification delivery status %q", status)
	}
	safeError = truncateMissionNotificationText(strings.TrimSpace(safeError), 300)
	var deliveredAt any
	if status == MissionNotificationDelivered {
		deliveredAt = formatStoreTime(now)
		safeError = ""
		nextAttemptAt = now
	} else if nextAttemptAt.IsZero() {
		return fmt.Errorf("pending notification delivery requires next attempt time")
	}
	result, err := s.db.Exec(`UPDATE mission_notification_deliveries SET status=?,attempts=attempts+1,last_error=?,
		next_attempt_at=?,delivered_at=?,updated_at=? WHERE intent_key=? AND channel=?`,
		status, safeError, formatStoreTime(nextAttemptAt), deliveredAt, formatStoreTime(now), key, channel)
	if err != nil {
		return fmt.Errorf("record mission notification attempt: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return fmt.Errorf("mission notification delivery not found")
	}
	return nil
}

// DeferMissionNotificationDelivery keeps an unavailable native channel
// pending without counting a transport attempt.
func (s *Store) DeferMissionNotificationDelivery(key, channel, safeReason string, nextAttemptAt, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if nextAttemptAt.IsZero() {
		return fmt.Errorf("deferred notification delivery requires next attempt time")
	}
	safeReason = truncateMissionNotificationText(strings.TrimSpace(safeReason), 300)
	result, err := s.db.Exec(`UPDATE mission_notification_deliveries SET last_error=?,next_attempt_at=?,updated_at=?
		WHERE intent_key=? AND channel=? AND status=?`, safeReason, formatStoreTime(nextAttemptAt),
		formatStoreTime(now), key, channel, MissionNotificationPending)
	if err != nil {
		return fmt.Errorf("defer mission notification delivery: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return fmt.Errorf("pending mission notification delivery not found")
	}
	return nil
}

func truncateMissionNotificationText(value string, maxRunes int) string {
	value = strings.ToValidUTF8(value, "�")
	runes := []rune(value)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return value
}

// GetMissionNotificationDelivery is intended for diagnostics and tests.
func (s *Store) GetMissionNotificationDelivery(key, channel string) (MissionNotificationDelivery, error) {
	var result MissionNotificationDelivery
	var next string
	err := s.db.QueryRow(`SELECT d.intent_key,i.run_id,d.channel,i.title,i.body,d.status,d.attempts,d.last_error,d.next_attempt_at
		FROM mission_notification_deliveries d JOIN mission_notification_intents i ON i.intent_key=d.intent_key
		WHERE d.intent_key=? AND d.channel=?`, key, channel).Scan(&result.IntentKey, &result.RunID, &result.Channel,
		&result.Title, &result.Body, &result.Status, &result.Attempts, &result.LastError, &next)
	if err != nil {
		return result, err
	}
	result.NextAttemptAt, err = parseStoreTime(next)
	return result, err
}
