package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NexProactivityPreferences are local host-owner preferences. A project scope
// may override the global scope, but cannot enable execution or tool calls.
type NexProactivityPreferences struct {
	Mode               string `json:"mode"`
	QuietStart         string `json:"quiet_start"`
	QuietEnd           string `json:"quiet_end"`
	TimeZone           string `json:"timezone"`
	MinIntervalMinutes int    `json:"min_interval_minutes"`
}

func defaultNexProactivityPreferences() NexProactivityPreferences {
	return NexProactivityPreferences{
		Mode: "manual", QuietStart: "22:00", QuietEnd: "08:00",
		TimeZone: "UTC", MinIntervalMinutes: 30,
	}
}

func ValidateNexProactivityPreferences(p NexProactivityPreferences) error {
	if p.Mode != "manual" && p.Mode != "contextual" && p.Mode != "proactive" {
		return fmt.Errorf("unsupported Nex proactivity mode")
	}
	if _, err := time.Parse("15:04", p.QuietStart); err != nil {
		return fmt.Errorf("invalid quiet start")
	}
	if _, err := time.Parse("15:04", p.QuietEnd); err != nil {
		return fmt.Errorf("invalid quiet end")
	}
	if len(p.TimeZone) == 0 || len(p.TimeZone) > 64 {
		return fmt.Errorf("invalid timezone")
	}
	if _, err := time.LoadLocation(p.TimeZone); err != nil {
		return fmt.Errorf("unknown timezone")
	}
	if p.MinIntervalMinutes < 5 || p.MinIntervalMinutes > 1440 {
		return fmt.Errorf("Nex notification interval must be between 5 and 1440 minutes")
	}
	return nil
}

func NexQuietHours(p NexProactivityPreferences, now time.Time) bool {
	loc, err := time.LoadLocation(p.TimeZone)
	if err != nil {
		return true // fail closed; never notify at an unknown local time
	}
	start, errStart := time.Parse("15:04", p.QuietStart)
	end, errEnd := time.Parse("15:04", p.QuietEnd)
	if errStart != nil || errEnd != nil {
		return true
	}
	a := start.Hour()*60 + start.Minute()
	b := end.Hour()*60 + end.Minute()
	t := now.In(loc).Hour()*60 + now.In(loc).Minute()
	if a == b {
		return false // equal endpoints mean no quiet period
	}
	if a < b {
		return t >= a && t < b
	}
	return t >= a || t < b // overnight quiet hours
}

func (s *Store) GetNexProactivityPreferences(projectID string) (NexProactivityPreferences, error) {
	p := defaultNexProactivityPreferences()
	err := s.db.QueryRow(`SELECT mode,quiet_start,quiet_end,timezone,min_interval_minutes
		FROM nex_proactivity_preferences WHERE project_scope=?`,
		nexConversationScope(projectID)).Scan(&p.Mode, &p.QuietStart, &p.QuietEnd,
		&p.TimeZone, &p.MinIntervalMinutes)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	return p, ValidateNexProactivityPreferences(p)
}

func (s *Store) SetNexProactivityPreferences(projectID string, p NexProactivityPreferences) error {
	if err := ValidateNexProactivityPreferences(p); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO nex_proactivity_preferences
		(project_scope,mode,quiet_start,quiet_end,timezone,min_interval_minutes,updated_at)
		VALUES(?,?,?,?,?,?,?) ON CONFLICT(project_scope) DO UPDATE SET
		mode=excluded.mode,quiet_start=excluded.quiet_start,quiet_end=excluded.quiet_end,
		timezone=excluded.timezone,min_interval_minutes=excluded.min_interval_minutes,
		updated_at=excluded.updated_at`, nexConversationScope(projectID), p.Mode,
		p.QuietStart, p.QuietEnd, p.TimeZone, p.MinIntervalMinutes,
		time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// TryClaimNexSuggestion allocates a single delivery to one of multiple Nexus
// clients. Claim is never an agent action; it only deduplicates user-visible
// suggestions against the local SQLite ledger.
func (s *Store) TryClaimNexSuggestion(projectID, fingerprint string, now time.Time) (bool, error) {
	if len(fingerprint) < 8 || len(fingerprint) > 160 || strings.TrimSpace(fingerprint) != fingerprint {
		return false, fmt.Errorf("invalid Nex suggestion fingerprint")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	scope := nexConversationScope(projectID)
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	p := defaultNexProactivityPreferences()
	err = tx.QueryRow(`SELECT mode,quiet_start,quiet_end,timezone,min_interval_minutes
		FROM nex_proactivity_preferences WHERE project_scope=?`, scope).
		Scan(&p.Mode, &p.QuietStart, &p.QuietEnd, &p.TimeZone, &p.MinIntervalMinutes)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if p.Mode != "proactive" || NexQuietHours(p, now) {
		return false, nil
	}
	var status string
	var snoozed sql.NullString
	err = tx.QueryRow(`SELECT status,snoozed_until FROM nex_proactivity_receipts
		WHERE project_scope=? AND fingerprint=?`, scope, fingerprint).Scan(&status, &snoozed)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if err == nil {
		if status != "SNOOZED" || !snoozed.Valid {
			return false, nil
		}
		until, parseErr := time.Parse(time.RFC3339Nano, snoozed.String)
		if parseErr != nil || now.Before(until) {
			return false, nil
		}
	}
	var last sql.NullString
	if err := tx.QueryRow(`SELECT MAX(updated_at) FROM nex_proactivity_receipts
		WHERE project_scope=? AND status='DELIVERED'`, scope).Scan(&last); err != nil {
		return false, err
	}
	if last.Valid {
		recent, parseErr := time.Parse(time.RFC3339Nano, last.String)
		if parseErr != nil || now.Sub(recent) < time.Duration(p.MinIntervalMinutes)*time.Minute {
			return false, nil
		}
	}
	timestamp := now.UTC().Format(time.RFC3339Nano)
	if _, err := tx.Exec(`INSERT INTO nex_proactivity_receipts
		(project_scope,fingerprint,status,snoozed_until,created_at,updated_at)
		VALUES(?,?,'DELIVERED',NULL,?,?)
		ON CONFLICT(project_scope,fingerprint) DO UPDATE
		SET status='DELIVERED',snoozed_until=NULL,updated_at=excluded.updated_at`,
		scope, fingerprint, timestamp, timestamp); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`DELETE FROM nex_proactivity_receipts WHERE rowid IN (
		SELECT rowid FROM nex_proactivity_receipts WHERE project_scope=?
		ORDER BY updated_at DESC LIMIT -1 OFFSET 500)`, scope); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// SetNexSuggestionDecision persists dismissal or a time-bounded snooze for an
// already claimed suggestion; never creates arbitrary synthetic claims.
func (s *Store) SetNexSuggestionDecision(projectID, fingerprint, decision string, minutes int, now time.Time) error {
	if decision != "DISMISSED" && decision != "SNOOZED" {
		return fmt.Errorf("invalid Nex suggestion decision")
	}
	if decision == "SNOOZED" && (minutes < 5 || minutes > 10080) {
		return fmt.Errorf("invalid Nex snooze duration")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var until any
	if decision == "SNOOZED" {
		until = now.Add(time.Duration(minutes) * time.Minute).UTC().Format(time.RFC3339Nano)
	}
	result, err := s.db.Exec(`UPDATE nex_proactivity_receipts
		SET status=?,snoozed_until=?,updated_at=?
		WHERE project_scope=? AND fingerprint=?`,
		decision, until, now.UTC().Format(time.RFC3339Nano),
		nexConversationScope(projectID), fingerprint)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return ErrNotFound
	}
	return nil
}
