package cooldown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

// Record tracks active rate-limit status and cooldown timers for a profile.
type Record struct {
	ProviderID    string             `json:"provider_id"`
	ProfileID     string             `json:"profile_id"`
	RateLimitedAt time.Time          `json:"rate_limited_at"`
	RetryAfter    time.Duration      `json:"retry_after"`
	ResetAt       time.Time          `json:"reset_at"`
	Reason        string             `json:"reason"`
	InProbe       bool               `json:"in_probe,omitempty"`
	AccountScope  model.AccountScope `json:"account_scope,omitempty"`
}

// Tracker coordinates rate-limit tracking and cooldowns across all profiles.
type Tracker struct {
	mu      sync.RWMutex
	records map[string]Record // key: provider:profile
}

// NewTracker creates a new Cooldown Tracker and loads persisted state.
func NewTracker() *Tracker {
	t := &Tracker{
		records: make(map[string]Record),
	}
	_ = t.load()
	return t
}

func key(provider, profile string) string {
	return fmt.Sprintf("%s:%s", provider, profile)
}

func scopeKey(scope model.AccountScope) string { return scope.Key() }

// RecordRateLimitForScope records cooldown under the canonical account key.
func (t *Tracker) RecordRateLimitForScope(scope model.AccountScope, retryAfter time.Duration, resetAt *time.Time, reason string) {
	if !scope.Verifiable() {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	effectiveReset := now.Add(retryAfter)
	if resetAt != nil && !resetAt.IsZero() {
		effectiveReset = *resetAt
	}
	if retryAfter <= 0 {
		retryAfter = time.Until(effectiveReset)
	}
	t.records[scopeKey(scope)] = Record{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, AccountScope: scope, RateLimitedAt: now, RetryAfter: retryAfter, ResetAt: effectiveReset, Reason: reason}
	_ = t.save()
}

// IsRateLimitedForScope checks only the requested account's cooldown.
func (t *Tracker) IsRateLimitedForScope(scope model.AccountScope) (bool, *Record) {
	if !scope.Verifiable() {
		return false, nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	rec, ok := t.records[scopeKey(scope)]
	if !ok || time.Now().After(rec.ResetAt) {
		return false, &rec
	}
	return true, &rec
}

// RecordRateLimit registers a rate limit occurrence for a profile.
func (t *Tracker) RecordRateLimit(provider, profile string, retryAfter time.Duration, resetAt *time.Time, reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	var effectiveReset time.Time
	if resetAt != nil && !resetAt.IsZero() {
		effectiveReset = *resetAt
	} else if retryAfter > 0 {
		effectiveReset = now.Add(retryAfter)
	} else {
		// Default conservative cooldown: 15 minutes
		retryAfter = 15 * time.Minute
		effectiveReset = now.Add(retryAfter)
	}

	k := key(provider, profile)
	t.records[k] = Record{
		ProviderID:    provider,
		ProfileID:     profile,
		RateLimitedAt: now,
		RetryAfter:    retryAfter,
		ResetAt:       effectiveReset,
		Reason:        reason,
		InProbe:       false,
	}

	_ = t.save()
}

// IsRateLimited checks if a profile is currently blocked by an active rate-limit cooldown.
func (t *Tracker) IsRateLimited(provider, profile string) (bool, *Record) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	k := key(provider, profile)
	rec, exists := t.records[k]
	if !exists {
		return false, nil
	}

	if time.Now().After(rec.ResetAt) {
		// Cooldown has expired
		return false, &rec
	}

	return true, &rec
}

// Clear removes a cooldown entry (e.g. after successful test probe).
func (t *Tracker) Clear(provider, profile string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	k := key(provider, profile)
	if _, exists := t.records[k]; exists {
		delete(t.records, k)
		_ = t.save()
	}
}

// ListActive returns all currently active rate-limited profile records.
func (t *Tracker) ListActive() []Record {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	var active []Record
	for _, rec := range t.records {
		if now.Before(rec.ResetAt) {
			active = append(active, rec)
		}
	}
	return active
}

func (t *Tracker) load() error {
	stateDir, err := config.StateDir()
	if err != nil {
		return err
	}
	path := filepath.Join(stateDir, "cooldowns.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var list []Record
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	now := time.Now()
	for _, rec := range list {
		if now.Before(rec.ResetAt) {
			t.records[key(rec.ProviderID, rec.ProfileID)] = rec
		}
	}
	return nil
}

func (t *Tracker) save() error {
	stateDir, err := config.StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	var list []Record
	now := time.Now()
	for _, rec := range t.records {
		if now.Before(rec.ResetAt) {
			list = append(list, rec)
		}
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	target := filepath.Join(stateDir, "cooldowns.json")
	temp := filepath.Join(stateDir, fmt.Sprintf("cooldowns.tmp.%d", time.Now().UnixNano()))
	if err := os.WriteFile(temp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(temp, target)
}
