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

// DefaultFallbackCooldown is the heuristic used only when the provider
// supplies no usable Retry-After/ResetAt signal. Provider metadata always wins.
const DefaultFallbackCooldown = 15 * time.Minute

// EffectiveCooldown resolves the authoritative cooldown for a provider failure.
// Precedence: valid ResetAt (absolute) > positive Retry-After > heuristic.
// It never substitutes a known reset with an arbitrary cooldown: callers can
// inspect hasProviderSignal to decide whether overwriting a longer known reset
// is allowed. Malformed inputs (non-positive Retry-After, zero ResetAt) fall
// back to the heuristic with hasProviderSignal=false. An expired ResetAt is a
// valid provider signal (hasProviderSignal=true) so the tracker treats it as
// expired instead of inventing a fresh 15m cooldown.
func EffectiveCooldown(failure model.Failure) (retryAfter time.Duration, resetAt time.Time, hasProviderSignal bool) {
	now := time.Now()
	if failure.ResetAt != nil && !failure.ResetAt.IsZero() {
		reset := *failure.ResetAt
		retry := time.Until(reset)
		if retry < 0 {
			retry = 0
		}
		// An expired absolute reset (retry == 0) keeps ResetAt precedence over a
		// positive relative RetryAfter hint: keep the absolute signal expired
		// and do not invent a fresh cooldown. Callers treat resetAt in the
		// past as not rate-limited.
		return retry, reset, true
	}
	if failure.RetryAfter != nil && *failure.RetryAfter > 0 {
		retry := *failure.RetryAfter
		return retry, now.Add(retry), true
	}
	return DefaultFallbackCooldown, now.Add(DefaultFallbackCooldown), false
}

func key(provider, profile string) string {
	return fmt.Sprintf("%s:%s", provider, profile)
}

func scopeKey(scope model.AccountScope) string { return scope.Key() }

func recordKeyForRecord(rec Record) string {
	if rec.AccountScope.Verifiable() {
		return scopeKey(rec.AccountScope)
	}
	return key(rec.ProviderID, rec.ProfileID)
}

// RecordRateLimitForScope records cooldown under the canonical account key.
// Provider Retry-After/ResetAt always wins over the heuristic fallback. A
// heuristic record never shortens a still-active known reset.
func (t *Tracker) RecordRateLimitForScope(scope model.AccountScope, retryAfter time.Duration, resetAt *time.Time, reason string) {
	if !scope.Verifiable() {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	effectiveRetry, effectiveReset, hasSignal := EffectiveCooldown(model.Failure{RetryAfter: durationPtr(retryAfter), ResetAt: resetAt})
	// durationPtr(nil-equivalent): EffectiveCooldown treats non-positive as no signal.
	// Re-derive without the helper's zero-value confusion when both inputs are empty.
	if retryAfter <= 0 && (resetAt == nil || resetAt.IsZero()) {
		hasSignal = false
		effectiveRetry = DefaultFallbackCooldown
		effectiveReset = now.Add(DefaultFallbackCooldown)
	}
	k := scopeKey(scope)
	if existing, ok := t.records[k]; ok && now.Before(existing.ResetAt) && !hasSignal && effectiveReset.Before(existing.ResetAt) {
		// Never substitute a known provider reset with an arbitrary cooldown.
		return
	}
	t.records[k] = Record{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, AccountScope: scope, RateLimitedAt: now, RetryAfter: effectiveRetry, ResetAt: effectiveReset, Reason: reason}
	_ = t.save()
}

// RecordFailureForScope is the Failure-typed seam used by fallback/scheduler.
// It applies the same precedence rules as RecordRateLimitForScope.
func (t *Tracker) RecordFailureForScope(scope model.AccountScope, failure model.Failure, reason string) {
	if !scope.Verifiable() {
		return
	}
	if reason == "" {
		reason = failure.Message
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	effectiveRetry, effectiveReset, hasSignal := EffectiveCooldown(failure)
	k := scopeKey(scope)
	if existing, ok := t.records[k]; ok && now.Before(existing.ResetAt) && !hasSignal && effectiveReset.Before(existing.ResetAt) {
		return
	}
	t.records[k] = Record{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, AccountScope: scope, RateLimitedAt: now, RetryAfter: effectiveRetry, ResetAt: effectiveReset, Reason: reason}
	_ = t.save()
}

// RecordFailure is the Failure-typed seam for unscoped (legacy provider:profile) keys.
func (t *Tracker) RecordFailure(provider, profile string, failure model.Failure, reason string) {
	if reason == "" {
		reason = failure.Message
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	effectiveRetry, effectiveReset, hasSignal := EffectiveCooldown(failure)
	k := key(provider, profile)
	if existing, ok := t.records[k]; ok && now.Before(existing.ResetAt) && !hasSignal && effectiveReset.Before(existing.ResetAt) {
		return
	}
	t.records[k] = Record{
		ProviderID:    provider,
		ProfileID:     profile,
		RateLimitedAt: now,
		RetryAfter:    effectiveRetry,
		ResetAt:       effectiveReset,
		Reason:        reason,
		InProbe:       false,
	}
	_ = t.save()
}

func durationPtr(d time.Duration) *time.Duration {
	if d <= 0 {
		return nil
	}
	out := d
	return &out
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
// Provider signals win; a heuristic record never shortens a known active reset.
func (t *Tracker) RecordRateLimit(provider, profile string, retryAfter time.Duration, resetAt *time.Time, reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	effectiveRetry, effectiveReset, hasSignal := EffectiveCooldown(model.Failure{RetryAfter: durationPtr(retryAfter), ResetAt: resetAt})
	if retryAfter <= 0 && (resetAt == nil || resetAt.IsZero()) {
		hasSignal = false
		effectiveRetry = DefaultFallbackCooldown
		effectiveReset = now.Add(DefaultFallbackCooldown)
	}

	k := key(provider, profile)
	if existing, ok := t.records[k]; ok && now.Before(existing.ResetAt) && !hasSignal && effectiveReset.Before(existing.ResetAt) {
		return
	}
	t.records[k] = Record{
		ProviderID:    provider,
		ProfileID:     profile,
		RateLimitedAt: now,
		RetryAfter:    effectiveRetry,
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

// ClearForScope removes a scoped cooldown entry.
func (t *Tracker) ClearForScope(scope model.AccountScope) {
	if !scope.Verifiable() {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	k := scopeKey(scope)
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
			t.records[recordKeyForRecord(rec)] = rec
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
