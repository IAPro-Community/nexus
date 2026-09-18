package telemetry

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

var (
	logMu sync.Mutex
)

// EventType represents the category of a telemetry event.
type EventType string

const (
	EventProfileSelected   EventType = "PROFILE_SELECTED"
	EventSessionStarted    EventType = "SESSION_STARTED"
	EventRateLimitDetected EventType = "RATE_LIMIT_DETECTED"
	EventFallbackTriggered EventType = "FALLBACK_TRIGGERED"
	EventQuotaRefreshed    EventType = "QUOTA_REFRESHED"
)

// Event records local audit information.
type Event struct {
	Timestamp       time.Time          `json:"timestamp"`
	Type            EventType          `json:"type"`
	ProviderID      string             `json:"provider_id"`
	ProfileID       string             `json:"profile_id"`
	Workspace       string             `json:"workspace,omitempty"`
	Reason          string             `json:"reason,omitempty"`
	DurationMs      int64              `json:"duration_ms,omitempty"`
	FailureKind     model.FailureKind  `json:"failure_kind,omitempty"`
	FallbackProfile string             `json:"fallback_profile,omitempty"`
	AccountScope    model.AccountScope `json:"account_scope,omitempty"`
}

// LogEvent appends a sanitized event record to the local history file.
func LogEvent(ev Event) error {
	logMu.Lock()
	defer logMu.Unlock()

	stateDir, err := config.StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}

	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	logFile := filepath.Join(stateDir, "history.jsonl")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(b, '\n'))
	return err
}

// ReadRecentEvents reads up to limit recent events from history.jsonl.
func ReadRecentEvents(limit int) ([]Event, error) {
	logMu.Lock()
	defer logMu.Unlock()

	stateDir, err := config.StateDir()
	if err != nil {
		return nil, err
	}
	logFile := filepath.Join(stateDir, "history.jsonl")
	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err == nil {
			events = append(events, ev)
		}
	}

	// Reverse to most recent first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

// StatsSummary aggregates local metrics for the given lookback period.
type StatsSummary struct {
	PeriodDays      int            `json:"period_days"`
	TotalSessions   int            `json:"total_sessions"`
	TotalFallbacks  int            `json:"total_fallbacks"`
	TotalRateLimits int            `json:"total_rate_limits"`
	ByProvider      map[string]int `json:"by_provider"`
	ByProfile       map[string]int `json:"by_profile"`
	RateLimits      map[string]int `json:"rate_limits_by_profile"`
}

// ComputeStats aggregates historical metrics over a lookback duration.
func ComputeStats(lookback time.Duration) (StatsSummary, error) {
	events, err := ReadRecentEvents(0)
	if err != nil {
		return StatsSummary{}, err
	}

	cutoff := time.Now().Add(-lookback)
	summary := StatsSummary{
		PeriodDays: int(lookback.Hours() / 24),
		ByProvider: make(map[string]int),
		ByProfile:  make(map[string]int),
		RateLimits: make(map[string]int),
	}

	for _, ev := range events {
		if ev.Timestamp.Before(cutoff) {
			continue
		}
		k := "UNATTRIBUTED"
		if ev.AccountScope.Verifiable() {
			k = ev.AccountScope.Key()
		}
		switch ev.Type {
		case EventSessionStarted:
			summary.TotalSessions++
			summary.ByProvider[ev.ProviderID]++
			summary.ByProfile[k]++
		case EventFallbackTriggered:
			summary.TotalFallbacks++
		case EventRateLimitDetected:
			summary.TotalRateLimits++
			summary.RateLimits[k]++
		}
	}

	return summary, nil
}
