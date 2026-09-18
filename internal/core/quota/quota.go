package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

const (
	DefaultTTL = 5 * time.Minute
	// LastKnownTTL is intentionally much longer than DefaultTTL. It is used
	// only to recover the last successful observation after a transient CLI or
	// network failure; it must never make an old snapshot trustworthy.
	LastKnownTTL = 30 * 24 * time.Hour
)

var (
	cacheMu sync.RWMutex
)

// Engine manages reading, refreshing, caching, and rendering quotas honestly.
type Engine struct {
	ttl time.Duration
}

// NewEngine creates a new Quota Engine instance.
func NewEngine(ttl time.Duration) *Engine {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Engine{ttl: ttl}
}

// Trustworthy reports whether a cached snapshot has an attributable,
// sufficiently recent observation. A cache file alone is not evidence.
func (e *Engine) Trustworthy(snap model.UsageSnapshot) bool {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if snap.Source == "" || snap.Source == model.SourceNone || snap.FetchedAt.IsZero() || len(snap.Windows) == 0 {
		if debug {
			slog.Debug("Trustworthy: false (missing fields)", "source", snap.Source, "fetchedAt", snap.FetchedAt, "windows", len(snap.Windows))
		}
		return false
	}
	// RATE_LIMITED belongs here: a provider that reports a block alongside exact
	// window percentages has given us attributable data. Excluding it would throw
	// away the most accurate reading we can get for an exhausted account.
	switch snap.Status {
	case model.UsageLive, model.UsageCached, model.UsageEstimated, model.UsageRateLimited:
	default:
		if debug {
			slog.Debug("Trustworthy: false (bad status)", "status", snap.Status)
		}
		return false
	}
	age := time.Since(snap.FetchedAt)
	trustworthy := age >= -time.Minute && age <= e.ttl
	if debug {
		slog.Debug("Trustworthy: checking age", "source", snap.Source, "status", snap.Status, "fetchedAt", snap.FetchedAt, "age", age, "ttl", e.ttl, "trustworthy", trustworthy)
	}
	return trustworthy
}

// GetCachedUsage returns the cached snapshot for a profile without triggering external requests.
func (e *Engine) GetCachedUsage(provider, profileName string) (model.UsageSnapshot, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("GetCachedUsage: starting", "provider", provider, "profile", profileName)
	}

	root, err := config.ProfileRoot(provider, profileName)
	if err != nil {
		if debug {
			slog.Debug("GetCachedUsage: profile root failed", "provider", provider, "profile", profileName, "err", err)
		}
		return model.UsageSnapshot{
			ProviderID: provider,
			ProfileID:  profileName,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
		}, false
	}

	quotaFile := filepath.Join(root, "usage.json")
	data, err := os.ReadFile(quotaFile)
	sourceFile := quotaFile
	if err != nil {
		// Fallback to legacy quota.json if available
		legacyFile := filepath.Join(root, "quota.json")
		data, err = os.ReadFile(legacyFile)
		sourceFile = legacyFile
		if err != nil {
			if debug {
				slog.Debug("GetCachedUsage: no quota files found", "provider", provider, "profile", profileName)
			}
			return model.UsageSnapshot{
				ProviderID: provider,
				ProfileID:  profileName,
				Status:     model.UsageUnknown,
				Source:     model.SourceNone,
			}, false
		}
	}

	if debug {
		slog.Debug("GetCachedUsage: reading file", "provider", provider, "profile", profileName, "file", sourceFile)
	}

	var snap model.UsageSnapshot
	_ = json.Unmarshal(data, &snap)
	// A previous Nexus version could have persisted an UNKNOWN result after a
	// failed refresh. If the profile still has a legacy quota.json, use it as
	// the last-known source instead of losing the only usable observation.
	if sourceFile == quotaFile && (snap.Status == model.UsageUnknown || snap.Status == "" || len(snap.Windows) == 0) {
		legacyFile := filepath.Join(root, "quota.json")
		if legacyData, legacyErr := os.ReadFile(legacyFile); legacyErr == nil {
			data = legacyData
			sourceFile = legacyFile
			snap = model.UsageSnapshot{}
			_ = json.Unmarshal(data, &snap)
		}
	}

	// A cache file is not evidence of current capacity after the trust window.
	// This applies to every provider, especially AGY where selecting a stale
	// default profile can launch against the wrong Google account.
	if statusHasMeasuredPercent(string(snap.Status)) && !e.Trustworthy(snap) {
		if debug {
			slog.Debug("GetCachedUsage: cache not trustworthy", "provider", provider, "profile", profileName, "status", snap.Status, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
		}
		snap.Status = model.UsageUnknown
		snap.Source = model.SourceNone
		snap.Windows = nil
		return snap, false
	}

	if snap.Status == "" || len(snap.Windows) == 0 {
		var leg struct {
			Account   string `json:"account"`
			Email     string `json:"email"`
			ModelName string `json:"model_name"`
			Plan      string `json:"plan"`
			FiveHour  struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"five_hour"`
			Weekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
				ResetsIn    string  `json:"resets_in"`
			} `json:"weekly"`
			ClaudeFiveHour struct {
				PercentLeft *float64 `json:"percent_left"`
				ResetTime   string   `json:"reset_time"`
				ResetsIn    string   `json:"resets_in"`
			} `json:"claude_five_hour"`
			ClaudeWeekly struct {
				PercentLeft *float64 `json:"percent_left"`
				ResetTime   string   `json:"reset_time"`
				ResetsIn    string   `json:"resets_in"`
			} `json:"claude_weekly"`
		}
		if json.Unmarshal(data, &leg) == nil && (leg.FiveHour.PercentLeft > 0 || leg.Weekly.PercentLeft > 0 || leg.FiveHour.ResetTime != "" || leg.Weekly.ResetTime != "" || leg.FiveHour.ResetsIn != "" || leg.Weekly.ResetsIn != "" || leg.ClaudeFiveHour.PercentLeft != nil || leg.ClaudeWeekly.PercentLeft != nil || leg.ClaudeFiveHour.ResetsIn != "" || leg.ClaudeWeekly.ResetsIn != "" || leg.ClaudeFiveHour.ResetTime != "" || leg.ClaudeWeekly.ResetTime != "") {
			if debug {
				slog.Debug("GetCachedUsage: parsing legacy format", "provider", provider, "profile", profileName, "account", firstNonEmpty(leg.Account, leg.Email))
			}
			// AGY and Codex legacy files both store remaining capacity in percent_left
			// (AGY UI label is "Limit Remaining"; a full bar is 100% remaining).
			p5h := clampPercent(leg.FiveHour.PercentLeft)
			u5h := 100.0 - p5h
			pWk := clampPercent(leg.Weekly.PercentLeft)
			uWk := 100.0 - pWk
			primaryGroup := "gemini"
			if provider == "codex" {
				primaryGroup = "claude_gpt"
			}
			r5h := leg.FiveHour.ResetTime
			if r5h == "" {
				r5h = leg.FiveHour.ResetsIn
			}
			rWk := leg.Weekly.ResetTime
			if rWk == "" {
				rWk = leg.Weekly.ResetsIn
			}

			windows := []model.UsageWindow{
				{
					Kind:             "5h",
					Group:            primaryGroup,
					RemainingPercent: &p5h,
					UsedPercent:      &u5h,
					ResetDescription: r5h,
				},
				{
					Kind:             "weekly",
					Group:            primaryGroup,
					RemainingPercent: &pWk,
					UsedPercent:      &uWk,
					ResetDescription: rWk,
				},
			}

			if provider == "agy" && (leg.ClaudeFiveHour.PercentLeft != nil || leg.ClaudeFiveHour.ResetsIn != "" || leg.ClaudeFiveHour.ResetTime != "" || leg.ClaudeWeekly.PercentLeft != nil || leg.ClaudeWeekly.ResetsIn != "" || leg.ClaudeWeekly.ResetTime != "") {
				c5hVal := 0.0
				if leg.ClaudeFiveHour.PercentLeft != nil {
					c5hVal = *leg.ClaudeFiveHour.PercentLeft
				}
				pC5h := clampPercent(c5hVal)
				uC5h := 100.0 - pC5h
				rC5h := leg.ClaudeFiveHour.ResetTime
				if rC5h == "" {
					rC5h = leg.ClaudeFiveHour.ResetsIn
				}
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_5h",
					Group:            "claude_gpt",
					RemainingPercent: &pC5h,
					UsedPercent:      &uC5h,
					ResetDescription: rC5h,
				})

				cWkVal := 0.0
				if leg.ClaudeWeekly.PercentLeft != nil {
					cWkVal = *leg.ClaudeWeekly.PercentLeft
				}
				pCWk := clampPercent(cWkVal)
				uCWk := 100.0 - pCWk
				rCWk := leg.ClaudeWeekly.ResetTime
				if rCWk == "" {
					rCWk = leg.ClaudeWeekly.ResetsIn
				}
				windows = append(windows, model.UsageWindow{
					Kind:             "claude_weekly",
					Group:            "claude_gpt",
					RemainingPercent: &pCWk,
					UsedPercent:      &uCWk,
					ResetDescription: rCWk,
				})
			}

			snap = model.UsageSnapshot{
				ProviderID: provider,
				ProfileID:  profileName,
				Account:    firstNonEmpty(leg.Account, leg.Email),
				Status:     model.UsageCached,
				Source:     model.SourceLocalFiles,
				ModelName:  leg.ModelName,
				FetchedAt:  fileFetchedAt(sourceFile),
				Windows:    windows,
			}
			if !e.Trustworthy(snap) {
				if debug {
					slog.Debug("GetCachedUsage: legacy snapshot not trustworthy", "provider", provider, "profile", profileName, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
				}
				snap.Status = model.UsageUnknown
				snap.Source = model.SourceNone
				snap.Windows = nil
				return snap, false
			}
			if debug {
				slog.Debug("GetCachedUsage: returning legacy snapshot", "provider", provider, "profile", profileName, "status", snap.Status, "fetchedAt", snap.FetchedAt)
			}
			return snap, true
		}
		if debug {
			slog.Debug("GetCachedUsage: no valid legacy format", "provider", provider, "profile", profileName)
		}
		return model.UsageSnapshot{
			ProviderID: provider,
			ProfileID:  profileName,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
		}, false
	}

	// Mark status as CACHED if it was originally LIVE
	if snap.Status == model.UsageLive {
		snap.Status = model.UsageCached
	}

	if debug {
		slog.Debug("GetCachedUsage: returning cached snapshot", "provider", provider, "profile", profileName, "status", snap.Status, "fetchedAt", snap.FetchedAt, "windows", len(snap.Windows))
	}
	return snap, true
}

// GetCachedUsageForScope reads cache only when the caller supplies the exact
// account boundary. The legacy method remains for compatibility with callers
// that have not migrated yet.
func (e *Engine) GetCachedUsageForScope(scope model.AccountScope) (model.UsageSnapshot, bool) {
	if !scope.Verifiable() {
		return model.UsageSnapshot{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, Status: model.UsageUnknown, Source: model.SourceNone}, false
	}
	snap, found := e.GetCachedUsage(scope.ProviderID, scope.ProfileID)
	if !found || !snap.AccountScope.Verifiable() || snap.AccountScope.Key() != scope.Key() {
		return model.UsageSnapshot{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, AccountScope: scope, Status: model.UsageUnknown, Source: model.SourceNone}, false
	}
	return snap, true
}

// GetLastKnownUsageForScope is the scoped equivalent of GetLastKnownUsage.
// It deliberately uses the long retention window while still requiring the
// exact account identity, so stale data cannot cross an identity change.
func (e *Engine) GetLastKnownUsageForScope(scope model.AccountScope) (model.UsageSnapshot, bool) {
	if !scope.Verifiable() {
		return model.UsageSnapshot{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, AccountScope: scope, Status: model.UsageUnknown, Source: model.SourceNone}, false
	}
	snap, found := NewEngine(LastKnownTTL).GetCachedUsage(scope.ProviderID, scope.ProfileID)
	if !found || !snap.AccountScope.Verifiable() || snap.AccountScope.Key() != scope.Key() {
		return model.UsageSnapshot{ProviderID: scope.ProviderID, ProfileID: scope.ProfileID, AccountScope: scope, Status: model.UsageUnknown, Source: model.SourceNone}, false
	}
	return snap, true
}

// GetLastKnownUsage returns the most recent snapshot that was persisted with
// windows, even when it is older than the normal trust window. Callers must
// mark the returned snapshot as degraded/estimated before displaying or using
// it. This prevents a transient refresh failure from erasing useful context
// while keeping freshness guarantees explicit.
func (e *Engine) GetLastKnownUsage(provider, profileName string) (model.UsageSnapshot, bool) {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("GetLastKnownUsage: starting", "provider", provider, "profile", profileName)
	}
	snap, found := NewEngine(LastKnownTTL).GetCachedUsage(provider, profileName)
	if debug {
		slog.Debug("GetLastKnownUsage: result", "provider", provider, "profile", profileName, "found", found, "status", snap.Status, "fetchedAt", snap.FetchedAt, "windows", len(snap.Windows))
	}
	return snap, found
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func fileFetchedAt(path string) time.Time {
	if st, err := os.Stat(path); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// SaveUsage persists a usage snapshot to the profile directory.
func (e *Engine) SaveUsage(snap model.UsageSnapshot) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if snap.FetchedAt.IsZero() {
		snap.FetchedAt = time.Now().UTC()
	}
	if snap.Source == "" || snap.Source == model.SourceNone {
		// Preserve compatibility with older callers while recording an explicit
		// observation origin for future trust decisions.
		snap.Source = model.SourceObservation
	}

	root, err := config.ProfileRoot(snap.ProviderID, snap.ProfileID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	quotaFile := filepath.Join(root, "usage.json")
	tempFile := filepath.Join(root, fmt.Sprintf("usage.tmp.%d", time.Now().UnixNano()))
	if err := os.WriteFile(tempFile, append(data, '\n'), 0600); err != nil {
		return err
	}

	return os.Rename(tempFile, quotaFile)
}

// SaveUsageForScope refuses to persist data without a verifiable account
// identity. This is the safe boundary for new adapters and ephemeral runs.
func (e *Engine) SaveUsageForScope(scope model.AccountScope, snap model.UsageSnapshot) error {
	if !scope.Verifiable() {
		return fmt.Errorf("cannot persist usage without a verifiable account scope")
	}
	snap.AccountScope = scope
	snap.ProviderID = scope.ProviderID
	snap.ProfileID = scope.ProfileID
	return e.SaveUsage(snap)
}

// SaveUsageForExecution is the persistence boundary for managed versus
// unmanaged runs. Ephemeral executions intentionally produce no account-owned
// cache, even when a provider returns useful usage data.
func (e *Engine) SaveUsageForExecution(scope model.AccountScope, snap model.UsageSnapshot, managed bool) error {
	if !managed {
		return nil
	}
	return e.SaveUsageForScope(scope, snap)
}

// FormatFreshness returns human-readable age of a snapshot (e.g. "updated 12s ago").
func FormatFreshness(fetchedAt time.Time) string {
	if fetchedAt.IsZero() {
		return "never updated"
	}
	d := time.Since(fetchedAt)
	if d < 10*time.Second {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// RenderProgressBar builds an honest visual progress bar.
func RenderProgressBar(status model.UsageStatus, remainingPercent *float64, width int) string {
	if width < 6 {
		width = 10
	}

	switch status {
	// RATE_LIMITED now carries exact percentages when it comes from the official
	// provider read. Rendering the bar keeps the consumption visible; the status
	// column is what tells the operator the account is blocked.
	case model.UsageLive, model.UsageCached, model.UsageEstimated, model.UsageRateLimited:
		if remainingPercent == nil {
			if status == model.UsageRateLimited {
				return formatProgressLabel("LIMITED", width)
			}
			return formatProgressLabel("UNKNOWN", width)
		}
		pct := *remainingPercent
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		fillCount := int((pct / 100.0) * float64(width))
		if fillCount > width {
			fillCount = width
		}
		emptyCount := width - fillCount

		// On Windows console (cmd.exe, PowerShell, legacy OEM code pages CP437/CP850/CP1252),
		// UTF-8 block characters (█, ░) frequently render as '???' or mojibake.
		// We use universal ASCII characters (# and -) on Windows to ensure 100% clean rendering.
		if runtime.GOOS == "windows" {
			return "[" + strings.Repeat("#", fillCount) + strings.Repeat("-", emptyCount) + "]"
		}
		return "[" + strings.Repeat("█", fillCount) + strings.Repeat("░", emptyCount) + "]"

	case model.UsageUnsupported:
		return formatProgressLabel("UNSUPPORT", width)

	case model.UsageError:
		return formatProgressLabel("ERROR", width)

	default:
		return formatProgressLabel("UNKNOWN", width)
	}
}

func formatProgressLabel(label string, width int) string {
	if len(label) >= width {
		return "[" + label[:width] + "]"
	}
	pad := width - len(label)
	left := pad / 2
	right := pad - left
	return "[" + strings.Repeat(" ", left) + label + strings.Repeat(" ", right) + "]"
}

// RenderShortStatus returns a compact 1-line representation for tables and status bars.
func RenderShortStatus(status model.UsageStatus, remainingPercent *float64, width int) string {
	bar := RenderProgressBar(status, remainingPercent, width)
	if (status == model.UsageLive || status == model.UsageCached) && remainingPercent != nil {
		return fmt.Sprintf("%s %3.0f%% (%s)", bar, *remainingPercent, string(status))
	}
	return fmt.Sprintf("%s (%s)", bar, string(status))
}

// BatchFetch retrieves usage snapshots for multiple profiles with controlled concurrency.
func (e *Engine) BatchFetch(ctx context.Context, profiles []model.Profile, fetcher func(ctx context.Context, p model.Profile) model.UsageSnapshot, maxWorkers int) map[string]model.UsageSnapshot {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	results := make(map[string]model.UsageSnapshot)
	var mu sync.Mutex

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, p := range profiles {
		wg.Add(1)
		go func(prof model.Profile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			snap := fetcher(ctx, prof)
			if prof.AccountScope.Verifiable() {
				snap.AccountScope = prof.AccountScope
				_ = e.SaveUsageForScope(prof.AccountScope, snap)
			}

			mu.Lock()
			key := prof.Provider + ":" + prof.Name
			if prof.AccountScope.Verifiable() {
				key = prof.AccountScope.Key()
			}
			results[key] = snap
			mu.Unlock()
		}(p)
	}

	wg.Wait()
	return results
}

// FetchBatch is an alias for BatchFetch.
func (e *Engine) FetchBatch(ctx context.Context, profiles []model.Profile, fetcher func(ctx context.Context, p model.Profile) model.UsageSnapshot, maxWorkers int) map[string]model.UsageSnapshot {
	return e.BatchFetch(ctx, profiles, fetcher, maxWorkers)
}
