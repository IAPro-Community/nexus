package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/agy"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/claude"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/cursor"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/gemini"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/opencode"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

// usageRefreshTimeout caps live provider probes so TUI quit / HTTP cancel
// always terminate quota fetches.
const usageRefreshTimeout = 15 * time.Second

type LimitWindow struct {
	PercentLeft float64 `json:"percent_left"`
	ResetsIn    string  `json:"resets_in"`
	ResetTime   string  `json:"reset_time"`
	ProgressBar string  `json:"progress_bar"`
	Status      string  `json:"status,omitempty"`
}

type QuotaDetails struct {
	Provider    string      `json:"provider"`
	ProfileName string      `json:"profile_name"`
	Account     string      `json:"account"`
	Plan        string      `json:"plan"`
	Status      string      `json:"status"`
	ModelName   string      `json:"model_name"`
	FiveHour    LimitWindow `json:"five_hour"`
	Weekly      LimitWindow `json:"weekly"`
	ClaudeFiveH LimitWindow `json:"claude_five_hour,omitempty"`
	ClaudeWeek  LimitWindow `json:"claude_weekly,omitempty"`
}

// GetUsageSnapshot returns cached/last-known quota only. It never spawns a
// provider CLI, HTTP probe, or Codex app-server — interactive surfaces such as
// `nexus usage` and Web ListResources must stay responsive. Use
// RefreshUsageSnapshot for live probes.
func GetUsageSnapshot(providerName, name string) model.UsageSnapshot {
	return loadUsageSnapshot(context.Background(), providerName, name, false, true)
}

// PeekUsageSnapshot returns last-known/cached quota without spawning a provider
// CLI or Codex app-server. Launch paths use this so `nexus agy` / `nexus codex`
// start instead of looking frozen while quota probes run.
func PeekUsageSnapshot(providerName, name string) model.UsageSnapshot {
	return loadUsageSnapshot(context.Background(), providerName, name, false, true)
}

// RefreshUsageSnapshot forces a provider fetch and persists the result when live.
func RefreshUsageSnapshot(providerName, name string) model.UsageSnapshot {
	return RefreshUsageSnapshotContext(context.Background(), providerName, name)
}

// RefreshUsageSnapshotContext forces a live provider fetch bounded by ctx and
// an overall usageRefreshTimeout.
func RefreshUsageSnapshotContext(ctx context.Context, providerName, name string) model.UsageSnapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	return loadUsageSnapshot(ctx, providerName, name, true, false)
}

func loadUsageSnapshot(ctx context.Context, providerName, name string, refresh, skipLive bool) model.UsageSnapshot {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("loadUsageSnapshot: starting", "provider", providerName, "profile", name, "refresh", refresh, "skipLive", skipLive)
	}

	qEng := quota.NewEngine(5 * time.Minute)
	expectedAccount := expectedProfileAccount(providerName, name)
	scope, scopeOK := usageScope(providerName, name, expectedAccount)
	var lastKnown model.UsageSnapshot
	var hasLastKnown bool
	if scopeOK {
		lastKnown, hasLastKnown = qEng.GetLastKnownUsageForScope(scope)
	}
	if !hasLastKnown {
		fallback, ok := qEng.GetLastKnownUsage(providerName, name)
		if ok && snapshotBelongsToProfile(fallback, providerName, name, expectedAccount) {
			lastKnown, hasLastKnown = fallback, true
		}
	}
	if hasLastKnown && !snapshotBelongsToProfile(lastKnown, providerName, name, expectedAccount) {
		lastKnown = model.UsageSnapshot{ProviderID: providerName, ProfileID: name, Status: model.UsageUnknown, Source: model.SourceNone}
		hasLastKnown = false
	}
	if debug {
		slog.Debug("loadUsageSnapshot: GetLastKnownUsage", "provider", providerName, "profile", name, "hasLastKnown", hasLastKnown, "lastKnownStatus", lastKnown.Status, "lastKnownFetchedAt", lastKnown.FetchedAt, "lastKnownWindows", len(lastKnown.Windows))
	}

	var snap model.UsageSnapshot
	var found bool
	if scopeOK {
		snap, found = qEng.GetCachedUsageForScope(scope)
	}
	if found && !snapshotBelongsToProfile(snap, providerName, name, expectedAccount) {
		snap = model.UsageSnapshot{ProviderID: providerName, ProfileID: name, Status: model.UsageUnknown, Source: model.SourceNone}
		found = false
	}
	if debug {
		slog.Debug("loadUsageSnapshot: GetCachedUsage", "provider", providerName, "profile", name, "found", found, "snapStatus", snap.Status, "snapFetchedAt", snap.FetchedAt, "snapWindows", len(snap.Windows), "snapAccount", snap.Account)
	}

	// AGY profiles are per-Google-account. Reject cached snapshots whose
	// recorded account does not match the profile's authenticated email.
	// This prevents stale cross-profile cache from being served.
	if providerName == "agy" && found && snap.Account != "" {
		authEmail := expectedAccount
		if debug {
			slog.Debug("loadUsageSnapshot: AGY account check", "provider", providerName, "profile", name, "cacheAccount", snap.Account, "authEmail", authEmail)
		}
		if authEmail != "" && !strings.EqualFold(authEmail, snap.Account) {
			if debug {
				slog.Debug("loadUsageSnapshot: AGY account mismatch, rejecting cache", "provider", providerName, "profile", name, "cacheAccount", snap.Account, "authEmail", authEmail)
			}
			snap = model.UsageSnapshot{
				ProviderID: providerName,
				ProfileID:  name,
				Status:     model.UsageUnknown,
				Source:     model.SourceNone,
			}
			found = false
		}
	}

	useCache := !refresh && found && len(snap.Windows) > 0 && snap.Status != model.UsageUnknown && qEng.Trustworthy(snap)
	// While an interactive Codex TUI owns this profile home, prefer last-known
	// official quota as CACHED instead of spawning a competing app-server.
	if !useCache && !refresh && providerName == "codex" {
		if home, err := config.ProfileHome("codex", name); err == nil && codex.IsTUILocked(home) {
			if found && snap.Source == model.SourceOfficialAPI && len(snap.Windows) > 0 {
				snap.Status = model.UsageCached
				useCache = true
			} else if hasLastKnown && lastKnown.Source == model.SourceOfficialAPI && len(lastKnown.Windows) > 0 {
				lastKnown.Status = model.UsageCached
				lastKnown.Error = "TUI Codex ativo; usando última cota oficial"
				return lastKnown
			}
		}
	}
	if debug {
		slog.Debug("loadUsageSnapshot: cache decision", "provider", providerName, "profile", name, "useCache", useCache, "trustworthy", qEng.Trustworthy(snap))
	}

	if useCache {
		if snap.Status == model.UsageLive {
			snap.Status = model.UsageCached
		}
		return snap
	}
	if skipLive || !refresh {
		if hasLastKnown && len(lastKnown.Windows) > 0 {
			lastKnown.ProviderID = providerName
			lastKnown.ProfileID = name
			if lastKnown.Status != model.UsageRateLimited {
				lastKnown.Status = model.UsageEstimated
				if lastKnown.Error == "" {
					lastKnown.Error = fmt.Sprintf("última observação conhecida de %s", quota.FormatFreshness(lastKnown.FetchedAt))
				}
			}
			return lastKnown
		}
		// Cache miss with no last-known: UNSUPPORTED providers stay honest.
		switch providerName {
		case "cursor", "opencode":
			return model.UsageSnapshot{
				ProviderID: providerName,
				ProfileID:  name,
				Status:     model.UsageUnsupported,
				Source:     model.SourceNone,
				FetchedAt:  time.Now(),
			}
		}
		return model.UsageSnapshot{
			ProviderID: providerName,
			ProfileID:  name,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
			FetchedAt:  time.Now(),
		}
	}

	if ctx == nil {
		ctx = context.Background()
	}
	probeCtx, cancel := context.WithTimeout(ctx, usageRefreshTimeout)
	defer cancel()
	p := model.Profile{Provider: providerName, Name: name, AccountScope: scope}
	switch providerName {
	case "codex":
		snap = codex.New().RefreshUsage(probeCtx, p)
	case "agy":
		snap = agy.New().RefreshUsage(probeCtx, p)
	case "claude":
		snap = claude.New().RefreshUsage(probeCtx, p)
	case "opencode":
		snap = opencode.New().RefreshUsage(probeCtx, p)
	case "gemini":
		snap = gemini.New().RefreshUsage(probeCtx, p)
	case "cursor":
		snap = cursor.New().RefreshUsage(probeCtx, p)
	default:
		snap = model.UsageSnapshot{
			ProviderID: providerName,
			ProfileID:  name,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
			FetchedAt:  time.Now(),
		}
	}
	if scopeOK && !snap.AccountScope.Verifiable() {
		snap.AccountScope = scope
	}
	if !snapshotBelongsToProfile(snap, providerName, name, expectedAccount) {
		snap = model.UsageSnapshot{ProviderID: providerName, ProfileID: name, Status: model.UsageUnknown, Source: model.SourceNone, FetchedAt: time.Now()}
	}

	if debug {
		slog.Debug("loadUsageSnapshot: after adapter", "provider", providerName, "profile", name, "status", snap.Status, "source", snap.Source, "fetchedAt", snap.FetchedAt, "windows", len(snap.Windows), "account", snap.Account)
	}

	// Reject stale snapshots from adapters: a quota file older than the
	// trust window is not evidence of live capacity. Keep processing below so
	// the last-known fallback can preserve context without treating it as live.
	if snap.Status != model.UsageUnknown && snap.Status != model.UsageError && len(snap.Windows) > 0 {
		// An adapter that already labeled its observation ESTIMATED is telling
		// the truth about freshness. Discarding it would turn real evidence into
		// SEM DADOS for any observation just past the trust window.
		if snap.Status == model.UsageEstimated && withinLastKnownWindow(snap.FetchedAt) {
			if snap.Error == "" {
				snap.Error = fmt.Sprintf("observação de %s", quota.FormatFreshness(snap.FetchedAt))
			}
			if scopeOK {
				snap.AccountScope = scope
				_ = qEng.SaveUsageForScope(scope, snap)
			}
			if debug {
				slog.Debug("loadUsageSnapshot: preserving adapter ESTIMATED", "provider", providerName, "profile", name, "fetchedAt", snap.FetchedAt)
			}
			return snap
		}
		if !qEng.Trustworthy(snap) {
			if debug {
				slog.Debug("loadUsageSnapshot: rejecting stale adapter snapshot", "provider", providerName, "profile", name, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
			}
			snap.Status = model.UsageUnknown
			snap.Source = model.SourceNone
			snap.Windows = nil
		} else {
			// Persist only trustworthy data so the scheduler and other
			// consumers read current quota instead of stale cache files.
			if scopeOK {
				snap.AccountScope = scope
				eng := quota.NewEngine(5 * time.Minute)
				_ = eng.SaveUsageForScope(scope, snap)
			}
			if debug {
				slog.Debug("loadUsageSnapshot: returning trustworthy adapter snapshot", "provider", providerName, "profile", name, "status", snap.Status)
			}
			return snap
		}
	}

	// A failed refresh must not destroy the last successful observation. Keep
	// official RATE_LIMITED blocks intact so AUTO/PREFER cannot select a
	// backend-blocked account. Other last-known snapshots are shown as
	// ESTIMATED with an explicit diagnostic.
	if hasLastKnown && len(lastKnown.Windows) > 0 {
		lastKnown.ProviderID = providerName
		lastKnown.ProfileID = name
		if lastKnown.Status == model.UsageRateLimited {
			if lastKnown.Error == "" {
				lastKnown.Error = fmt.Sprintf("official rate limit still in effect from %s", quota.FormatFreshness(lastKnown.FetchedAt))
			}
			if scopeOK {
				_ = qEng.SaveUsageForScope(scope, lastKnown)
			}
			if debug {
				slog.Debug("loadUsageSnapshot: preserving RATE_LIMITED lastKnown", "provider", providerName, "profile", name, "lastKnownFetchedAt", lastKnown.FetchedAt)
			}
			return lastKnown
		}
		lastKnown.Status = model.UsageEstimated
		lastKnown.Error = fmt.Sprintf("live usage refresh failed; showing last known observation from %s", quota.FormatFreshness(lastKnown.FetchedAt))
		if scopeOK {
			_ = qEng.SaveUsageForScope(scope, lastKnown)
		}
		if debug {
			slog.Debug("loadUsageSnapshot: returning lastKnown as ESTIMATED", "provider", providerName, "profile", name, "lastKnownFetchedAt", lastKnown.FetchedAt, "lastKnownAccount", lastKnown.Account)
		}
		return lastKnown
	}
	if debug {
		slog.Debug("loadUsageSnapshot: returning final snap", "provider", providerName, "profile", name, "status", snap.Status)
	}
	return snap
}

// withinLastKnownWindow bounds how old an explicitly ESTIMATED observation may
// be before it stops being useful context at all.
func withinLastKnownWindow(at time.Time) bool {
	if at.IsZero() {
		return false
	}
	age := time.Since(at)
	return age >= -time.Minute && age <= quota.LastKnownTTL
}

func expectedProfileAccount(providerName, profileName string) string {
	if providerName == "agy" {
		if email := resolveAGYAuthenticatedEmail(profileName); email != "" {
			return email
		}
		// google_accounts.json / jetski may be missing after isolation changes;
		// InspectAuth still recovers the Google email from the oauth id_token.
		return strings.TrimSpace(GetAccountInfo("agy", profileName).Email)
	}
	info := GetAccountInfo(providerName, profileName)
	if providerName == "codex" && strings.TrimSpace(info.ExternalAccountID) != "" {
		return strings.TrimSpace(info.ExternalAccountID)
	}
	return strings.TrimSpace(info.Email)
}

func usageScope(providerName, profileName, identity string) (model.AccountScope, bool) {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return model.AccountScope{}, false
	}
	scope, err := AccountScope(providerName, profileName, identity)
	return scope, err == nil && scope.Verifiable()
}

func snapshotBelongsToProfile(snap model.UsageSnapshot, providerName, profileName, expectedAccount string) bool {
	if snap.ProviderID != "" && snap.ProviderID != providerName {
		return false
	}
	if snap.ProfileID != "" && snap.ProfileID != profileName {
		return false
	}
	// A resolved account may only consume snapshots carrying the exact same
	// identity scope. Legacy files without this metadata remain untrusted and
	// must be refreshed before they can become account-owned usage.
	if scope, ok := usageScope(providerName, profileName, expectedAccount); ok {
		// Fresh adapter observations often omit AccountScope. Reject only when
		// a verifiable scope is present and belongs to a different identity.
		if snap.AccountScope.Verifiable() && snap.AccountScope.Key() != scope.Key() {
			// Codex may bump IdentityVersion when migrating email → chatgpt_account_id,
			// but only a persisted historical identity can prove continuity.
			sameDurable := providerName == "codex" &&
				snap.AccountScope.AccountID == scope.AccountID &&
				snap.AccountScope.ProviderID == scope.ProviderID &&
				snap.AccountScope.ProfileID == scope.ProfileID
			if !sameDurable || !persistedScopeContainsIdentity(providerName, profileName, scope.AccountID, snap.Account) {
				return false
			}
		}
	}
	if expectedAccount == "" {
		// Without an authenticated identity, no snapshot can be attributed.
		return false
	}
	if snap.Account == "" {
		// Snapshot omitted account but already passed provider/profile/scope checks.
		return true
	}
	if strings.EqualFold(expectedAccount, snap.Account) {
		return true
	}
	// A current-version snapshot may still carry the previous Codex identity
	// after it was re-associated with the current durable account scope.
	if providerName == "codex" {
		if scope, ok := usageScope(providerName, profileName, expectedAccount); ok &&
			snap.AccountScope.Verifiable() &&
			snap.AccountScope.ProviderID == scope.ProviderID &&
			snap.AccountScope.ProfileID == scope.ProfileID &&
			snap.AccountScope.AccountID == scope.AccountID &&
			persistedScopeContainsIdentity(providerName, profileName, scope.AccountID, snap.Account) {
			return true
		}
	}
	return false
}

// GetQuotaDetails returns usage and quota metrics without fabricating 100% data.
func GetQuotaDetails(providerName, name, plan, email string) QuotaDetails {
	snap := GetUsageSnapshot(providerName, name)

	q := QuotaDetails{
		Provider:    providerName,
		ProfileName: name,
		Account:     email,
		Plan:        plan,
		Status:      string(snap.Status),
		ModelName:   snap.ModelName,
	}

	if q.ModelName == "" {
		switch providerName {
		case "codex":
			q.ModelName = "gpt-5.6-sol"
		case "agy":
			q.ModelName = "Gemini 2.5 Flash / Pro"
		case "claude":
			q.ModelName = "Claude 3.7 Sonnet"
		case "opencode":
			q.ModelName = "OpenCode Provider"
		case "gemini":
			q.ModelName = "Gemini Pro"
		}
	}

	// Map windows if present
	for _, w := range snap.Windows {
		var pct float64
		if w.RemainingPercent != nil {
			pct = *w.RemainingPercent
		}
		bar := quota.RenderProgressBar(snap.Status, w.RemainingPercent, 10)
		if snap.Status == model.UsageLive || snap.Status == model.UsageCached {
			bar = fmt.Sprintf("%s %2.0f%%", bar, pct)
		}
		lw := LimitWindow{
			PercentLeft: pct,
			ResetTime:   w.ResetDescription,
			ResetsIn:    w.ResetDescription,
			ProgressBar: bar,
			Status:      string(snap.Status),
		}
		switch w.Kind {
		case "5h", "daily":
			q.FiveHour = lw
		case "weekly":
			q.Weekly = lw
		case "claude_5h", "claude_five_hour":
			q.ClaudeFiveH = lw
		case "claude_weekly":
			q.ClaudeWeek = lw
		}
	}

	if len(snap.Windows) == 0 {
		q.FiveHour = LimitWindow{
			ProgressBar: quota.RenderProgressBar(snap.Status, nil, 10),
			Status:      string(snap.Status),
			ResetTime:   "Status: " + string(snap.Status),
		}
		q.Weekly = LimitWindow{
			ProgressBar: quota.RenderProgressBar(snap.Status, nil, 10),
			Status:      string(snap.Status),
			ResetTime:   "Status: " + string(snap.Status),
		}
	}

	return q
}

// GetSanitizedUsageSnapshot returns the exact quota evidence allowed to reach
// public capacity surfaces. It preserves GetUsageSnapshot's non-blocking
// behavior while applying account-identity and freshness guards before the
// snapshot is projected into either QuotaView or resource_channels.
func GetSanitizedUsageSnapshot(providerName, name, email string) model.UsageSnapshot {
	debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"
	if debug {
		slog.Debug("GetSanitizedUsageSnapshot: starting", "provider", providerName, "profile", name, "email", email)
	}

	snap := GetUsageSnapshot(providerName, name)
	if debug {
		slog.Debug("GetSanitizedUsageSnapshot: after GetUsageSnapshot", "provider", providerName, "profile", name, "status", snap.Status, "account", snap.Account, "windows", len(snap.Windows))
	}

	// A cached quota can outlive an account switch. Do not show or score data
	// whose recorded identity differs from the profile's authenticated identity.
	if snapshotAccount := strings.TrimSpace(snap.Account); snapshotAccount != "" && strings.TrimSpace(email) != "" && !strings.EqualFold(snapshotAccount, strings.TrimSpace(email)) {
		mismatch := true
		if providerName == "codex" {
			info := GetAccountInfo(providerName, name)
			if info.ExternalAccountID != "" && strings.EqualFold(snapshotAccount, info.ExternalAccountID) {
				mismatch = false
			}
			if info.Email != "" && strings.EqualFold(snapshotAccount, info.Email) {
				mismatch = false
			}
		}
		if mismatch {
			if debug {
				slog.Debug("GetSanitizedUsageSnapshot: account mismatch detected", "provider", providerName, "profile", name, "snapshotAccount", snapshotAccount, "profileEmail", email)
			}
			snap.Status = model.UsageUnknown
			snap.Source = model.SourceNone
			snap.Windows = nil
			snap.Error = fmt.Sprintf("quota belongs to %s, profile is authenticated as %s", snapshotAccount, strings.TrimSpace(email))
		}
	}

	// Stale local files must not score as fresh CACHED (e.g. August quota.json).
	qEng := quota.NewEngine(quota.DefaultTTL)
	if len(snap.Windows) > 0 && snap.Status != model.UsageUnknown && !qEng.Trustworthy(snap) {
		if debug {
			slog.Debug("GetSanitizedUsageSnapshot: marking stale as ESTIMATED", "provider", providerName, "profile", name, "fetchedAt", snap.FetchedAt, "age", time.Since(snap.FetchedAt))
		}
		if snap.Status == model.UsageCached || snap.Status == model.UsageLive {
			snap.Status = model.UsageEstimated
		}
	}

	// Apply default model names when the adapter didn't set one.
	if snap.ModelName == "" {
		snap.ModelName = defaultModelName(providerName)
	}

	return snap
}

// GetQuotaView returns the omnibus QuotaView for any consumer (TUI, Web, CLI, Scheduler).
// This is the preferred entry point for quota display and scoring.
func GetQuotaView(providerName, name, plan, email string) quota.QuotaView {
	snap := GetSanitizedUsageSnapshot(providerName, name, email)
	return quota.BuildQuotaView(snap, email, plan)
}

// defaultModelName returns a fallback model name for providers that don't set one.
func defaultModelName(provider string) string {
	switch provider {
	case "codex":
		return "gpt-5.6-sol"
	case "agy":
		return "Gemini 2.5 Flash / Pro"
	case "claude":
		return "Claude 3.7 Sonnet"
	case "opencode":
		return "OpenCode Provider"
	case "gemini":
		return "Gemini Pro"
	default:
		return provider
	}
}

// RenderBar generates a clean progress bar.
func RenderBar(percent float64, width int) string {
	p := percent
	return quota.RenderProgressBar(model.UsageLive, &p, width)
}

func RenderShortBar(percent float64) string {
	p := percent
	return quota.RenderShortStatus(model.UsageLive, &p, 10)
}

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// resolveAGYAuthenticatedEmail reads the actual Google account email for an
// AGY profile from google_accounts.json or jetski_state.pbtxt.
// Returns "" when no authoritative email can be determined.
func resolveAGYAuthenticatedEmail(profileName string) string {
	root, err := config.ProfileRoot("agy", profileName)
	if err != nil {
		return ""
	}
	home := filepath.Join(root, "home")

	// 1. google_accounts.json in profile home.
	accountsFile := filepath.Join(home, ".gemini", "google_accounts.json")
	if email := readActiveEmailFromAccountsFile(accountsFile); email != "" {
		return email
	}

	// 2. jetski_state.pbtxt in profile home.
	jetskiFile := filepath.Join(home, ".gemini", "antigravity-cli", "jetski_state.pbtxt")
	if email := readEmailFromJetski(jetskiFile); email != "" {
		return email
	}

	return ""
}

func readActiveEmailFromAccountsFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var acc struct {
		Active string `json:"active"`
	}
	if json.Unmarshal(data, &acc) == nil {
		return acc.Active
	}
	return ""
}

func readEmailFromJetski(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	if matches := emailRegex.FindAllString(string(data), -1); len(matches) > 0 {
		return matches[0]
	}
	return ""
}
