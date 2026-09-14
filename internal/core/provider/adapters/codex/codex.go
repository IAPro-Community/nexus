package codex

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// rolloutMaxAge limits how far back we scan rollouts for current quota windows.
const rolloutMaxAge = 30 * 24 * time.Hour

// rolloutMaxCheck caps how many candidate rollout files we open per GetUsage.
const rolloutMaxCheck = 40

// reverseReadBlock is the block size used when reading rollouts from the end.
const reverseReadBlock = 64 * 1024

// usageLiveTTL is how fresh a rollout event must be to report UsageLive. It is
// deliberately the same window the quota engine trusts: a wider TTL here would
// claim LIVE for observations the profile layer then discards as stale.
// Older observations remain useful as UsageEstimated.
const usageLiveTTL = quota.DefaultTTL

type Adapter struct{}

func titlePlan(value string) string {
	value = strings.ToLower(value)
	for i, r := range value {
		return string(unicode.ToUpper(r)) + value[i+len(string(r)):]
	}
	return value
}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID {
	return "codex"
}

func (a *Adapter) Name() string {
	return "Codex"
}

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login:              true,
		Logout:             true,
		Usage:              true,
		Conversations:      true,
		Resume:             true,
		CrossAccountResume: true,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := spec.MustGet("codex").ResolveBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "not found in PATH"}
	}
	out, _ := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    strings.TrimSpace(out),
	}
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	lock, ok, err := TryAcquireMaintenanceLock(home)
	if err != nil {
		return err
	}
	if !ok {
		// A ready profile needs no migration or mutation. Reuse it while one or
		// more interactive TUIs hold shared leases; incomplete profiles fail
		// explicitly so callers can retry after maintenance becomes available.
		if codexProfileReady(home) {
			return nil
		}
		return fmt.Errorf("%w: profile preparation is still required", ErrTUIBusy)
	}
	defer func() { _ = lock.Release() }()
	if err := os.MkdirAll(home, 0700); err != nil {
		return err
	}

	// Apply isolation preset policies
	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	if err := security.ApplyIsolation(home, policy); err != nil {
		return err
	}

	// Single canonical CODEX_HOME: auth and config live at the profile home root.
	// Legacy home/.codex is still readable for migration, but we no longer write
	// a second sessions/sqlite tree that competes with the official CLI.
	configFile := filepath.Join(home, "config.toml")
	authHome := filepath.Join(home, "auth.json")
	legacyAuth := filepath.Join(home, ".codex", "auth.json")
	if data, err := os.ReadFile(authHome); err != nil || len(data) == 0 {
		if data, err := os.ReadFile(legacyAuth); err == nil && len(data) > 0 {
			_ = os.WriteFile(authHome, data, 0600)
		}
	}
	legacyConfig := filepath.Join(home, ".codex", "config.toml")
	if data, err := os.ReadFile(configFile); err != nil || len(data) == 0 {
		if data, err := os.ReadFile(legacyConfig); err == nil && len(data) > 0 {
			_ = os.WriteFile(configFile, data, 0600)
		}
	}
	_ = ensureConfigFile(configFile)

	// Isolate session history per profile so quota attribution is deterministic.
	// Rules/skills/customizations remain shared from the host.
	migrateAwayFromSharedSessions(home)
	hostHome := security.FindHostHome()
	if hostHome != "" {
		hostCodex := filepath.Join(hostHome, ".codex")
		if _, err := os.Stat(hostCodex); err == nil {
			linkSharedCodexItems(home, hostCodex)
		}
		// Keep a durable ledger of which chatgpt_account_id owns host auth.
		if id := readCodexAuthAccountID(filepath.Join(hostCodex, "auth.json")); id != "" {
			recordHostAuthObservation(id)
		}
	}
	_ = os.MkdirAll(filepath.Join(home, "sessions"), 0700)
	// CrossAccountResume: import sibling/host conversation artifacts so the
	// native Codex resume picker can see threads from other Codex accounts.
	_ = seedCrossAccountSessions(p.Name)

	return nil
}

func codexProfileReady(home string) bool {
	if _, err := os.Stat(filepath.Join(home, "config.toml")); err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(home, "sessions"))
	return err == nil && info.IsDir()
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := spec.MustGet("codex").ResolveBinary()
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "codex binary not found"}, err
	}
	home, _ := config.ProfileHome(string(a.ID()), p.Name)
	lock, lockErr := AcquireTUILease(home)
	if lockErr != nil {
		return model.Failure{Kind: model.FailureCommand, Message: lockErr.Error()}, lockErr
	}
	defer func() { _ = lock.Release() }()

	cwd, _ := os.Getwd()
	envOverrides := map[string]string{
		"HOME":             home,
		"CODEX_HOME":       home,
		"CODEX_CONFIG_DIR": home,
		// xterm.js 5.3, used by Nexus, does not support Kitty/CSI-u keyboard
		// enhancement. Keep Codex from leaking those sequences into the prompt.
		"CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT": "1",
		"XDG_CONFIG_HOME":                        filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":                         filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":                          filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":                         filepath.Join(home, ".local", "state"),
		"PATH":                                   runtime.EnhancedPATH(filepath.Dir(bin)),
	}
	env := runtime.EnvSet(os.Environ(), envOverrides)
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, []string{"login"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(home, "auth.json"))
	_ = os.Remove(filepath.Join(home, ".codex", "auth.json"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{
			Status:        "Error",
			Health:        model.HealthUnknown,
			Authenticated: false,
		}
	}

	info := model.AccountInfo{
		Plan:          "ChatGPT Free",
		Status:        "Not authenticated",
		Health:        model.HealthAuthRequired,
		Authenticated: false,
		Usage: model.UsageSnapshot{
			ProviderID: string(a.ID()),
			ProfileID:  p.Name,
			Status:     model.UsageUnknown,
			Source:     model.SourceNone,
		},
	}

	authCandidates := []string{
		filepath.Join(home, "auth.json"),
		filepath.Join(home, ".codex", "auth.json"),
	}
	var data []byte
	for _, ac := range authCandidates {
		if d, err := os.ReadFile(ac); err == nil && len(d) > 0 {
			data = d
			break
		}
	}
	if len(data) == 0 {
		return info
	}

	accountID, email, plan, activeUntil := parseCodexAuthBytes(data)
	if accountID == "" && email == "" {
		return info
	}
	var expiresAt time.Time
	if activeUntil != "" {
		if until, err := time.Parse(time.RFC3339, activeUntil); err == nil {
			expiresAt = until
			if time.Now().After(until) {
				info.Status = "Subscription expired"
				info.Health = model.HealthAuthRequired
				info.Authenticated = false
				info.Email = email
				info.ExternalAccountID = accountID
				info.ExpiresAt = expiresAt
				if plan != "" {
					info.Plan = plan
				}
				return info
			}
		}
	}

	info.Authenticated = true
	info.Status = "Authenticated"
	info.Health = model.HealthHealthy
	info.Email = email
	info.ExternalAccountID = accountID
	info.ExpiresAt = expiresAt
	if plan != "" {
		info.Plan = plan
	} else {
		info.Plan = "ChatGPT Plus"
	}
	return info
}

// GetUsage returns quota for a profile, preferring the official app-server read
// and falling back to local rollout evidence.
func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return a.usage(ctx, p, false)
}

// RefreshUsage bypasses the memoized app-server probe and reads quota now.
func (a *Adapter) RefreshUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return a.usage(ctx, p, true)
}

func (a *Adapter) usage(ctx context.Context, p model.Profile, force bool) model.UsageSnapshot {
	snap := model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}

	home := ""
	if h, err := config.ProfileHome(string(a.ID()), p.Name); err == nil {
		home = h
	}
	tuiBusy := home != "" && IsTUILocked(home)

	// Never mutate session stores from the quota hot path: a concurrent TUI
	// may own thread_history / sessions under this CODEX_HOME.
	if !tuiBusy {
		// 1. Official quota read. Works for any authenticated account without a
		// session, a prompt or quota consumption.
		if apiSnap, ok := a.appServerUsage(ctx, p, force); ok {
			return apiSnap
		}
	}

	// 2. Rollout evidence written by this profile's own sessions.
	if rollSnap, ok := a.getUsageFromRollouts(ctx, p); ok {
		return rollSnap
	}

	// 3. Same chatgpt_account_id as host: adopt the newest eligible host rollout
	// into the isolated store, then re-read. Skip while the TUI owns the home.
	if !tuiBusy && a.tryAdoptLatestHostRollout(ctx, p) {
		if rollSnap, ok := a.getUsageFromRollouts(ctx, p); ok {
			return rollSnap
		}
	}

	info := a.InspectAuth(ctx, p)
	if info.Authenticated {
		snap.Account = info.ExternalAccountID
		if snap.Account == "" {
			snap.Account = info.Email
		}
		if tuiBusy {
			snap.Error = "cota oficial adiada: TUI Codex ativo neste perfil"
		} else {
			snap.Error = "cota oficial indisponível e nenhuma sessão isolada registrada"
		}
	}

	// Persisted observations are loaded by profile.GetUsageSnapshot through the
	// quota engine. Do not read usage.json here: that would bypass account-scope checks.
	return snap
}

type rateLimitWindow struct {
	UsedPercent   float64 `json:"used_percent"`
	WindowMinutes int     `json:"window_minutes"`
	ResetsAt      int64   `json:"resets_at"`
}

type rolloutRateLimits struct {
	Primary   *rateLimitWindow `json:"primary"`
	Secondary *rateLimitWindow `json:"secondary"`
}

type rolloutUsageHit struct {
	limits    rolloutRateLimits
	modelName string
	fetchedAt time.Time
}

func (a *Adapter) getUsageFromRollouts(ctx context.Context, p model.Profile) (model.UsageSnapshot, bool) {
	_ = ctx
	info := a.InspectAuth(ctx, p)
	profileAccountID := strings.TrimSpace(info.ExternalAccountID)
	profileEmail := strings.TrimSpace(info.Email)
	if profileAccountID == "" && profileEmail == "" {
		return model.UsageSnapshot{}, false
	}

	hostHome := security.FindHostHome()
	hostSessions := ""
	hostAccountID := ""
	if hostHome != "" {
		hostSessions = filepath.Join(hostHome, ".codex", "sessions")
		hostAccountID = readCodexAuthAccountID(filepath.Join(hostHome, ".codex", "auth.json"))
		if hostAccountID != "" {
			recordHostAuthObservation(hostAccountID)
		}
	}

	profileHome := ""
	var profileSessionRoots []string
	if h, err := config.ProfileHome(string(a.ID()), p.Name); err == nil {
		profileHome = h
		for _, d := range []string{filepath.Join(h, "sessions"), filepath.Join(h, ".codex", "sessions")} {
			if real, ok := realNonSharedDir(d, hostSessions); ok {
				profileSessionRoots = append(profileSessionRoots, real)
			}
		}
	}

	type fileInfo struct {
		path       string
		modTime    time.Time
		sharedHost bool
	}
	seen := map[string]bool{}
	var rolloutFiles []fileInfo
	cutoff := time.Now().Add(-rolloutMaxAge)

	collect := func(dir string, sharedHost bool) {
		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			realDir = dir
		}
		if _, err := os.Stat(realDir); err != nil {
			return
		}
		_ = filepath.Walk(realDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi == nil || fi.IsDir() {
				return nil
			}
			if !strings.HasPrefix(fi.Name(), "rollout-") || !strings.HasSuffix(fi.Name(), ".jsonl") {
				return nil
			}
			if fi.ModTime().Before(cutoff) {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			rolloutFiles = append(rolloutFiles, fileInfo{path: path, modTime: fi.ModTime(), sharedHost: sharedHost})
			return nil
		})
	}

	for _, d := range profileSessionRoots {
		collect(d, false)
	}
	if hostSessions != "" {
		collect(hostSessions, true)
	}

	if len(rolloutFiles) == 0 {
		return model.UsageSnapshot{}, false
	}

	sort.Slice(rolloutFiles, func(i, j int) bool {
		return rolloutFiles[i].modTime.After(rolloutFiles[j].modTime)
	})

	maxCheck := rolloutMaxCheck
	if len(rolloutFiles) < maxCheck {
		maxCheck = len(rolloutFiles)
	}

	// File mtime is a weak proxy for observation time: a rollout can be appended
	// long after its last rate_limits event, and adoption rewrites mtime entirely.
	// Rank eligible candidates by the timestamp carried by the event itself.
	var best rolloutUsageHit
	var bestAt time.Time
	found := false
	for _, rf := range rolloutFiles[:maxCheck] {
		belongs := rolloutBelongsToProfile(rf.path, rf.modTime, rf.sharedHost, profileHome, profileSessionRoots, profileAccountID, hostAccountID)
		if !belongs {
			continue
		}
		hit, ok := readLatestRateLimitsFromEnd(rf.path)
		if !ok || hit.limits.Primary == nil {
			continue
		}
		if len(buildWindowsFromRateLimits(hit.limits)) == 0 {
			continue
		}
		at := hit.fetchedAt
		if at.IsZero() {
			at = rf.modTime
		}
		if found && !at.After(bestAt) {
			continue
		}
		hit.fetchedAt = at
		best, bestAt, found = hit, at, true
	}
	if !found {
		return model.UsageSnapshot{}, false
	}

	modelName := best.modelName
	if modelName == "" {
		modelName = "gpt-5.6-terra"
	}
	account := profileEmail
	if profileAccountID != "" {
		account = profileAccountID
	}
	status := model.UsageLive
	if time.Since(bestAt) > usageLiveTTL {
		status = model.UsageEstimated
	}
	return model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Account:    account,
		Status:     status,
		Source:     model.SourceObservation,
		ModelName:  modelName,
		FetchedAt:  bestAt,
		Windows:    buildWindowsFromRateLimits(best.limits),
	}, true
}

func realNonSharedDir(dir, hostSessions string) (string, bool) {
	fi, err := os.Lstat(dir)
	if err != nil {
		return "", false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		// Still a shared symlink — not an isolated profile store.
		target, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return "", false
		}
		if hostSessions != "" && (config.FilesystemPathWithin(hostSessions, target) || config.FilesystemPathsEquivalent(target, hostSessions)) {
			return "", false
		}
		return target, true
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		realDir = dir
	}
	if hostSessions != "" && (config.FilesystemPathWithin(hostSessions, realDir) || config.FilesystemPathsEquivalent(realDir, hostSessions)) {
		return "", false
	}
	return realDir, true
}

func rolloutBelongsToProfile(path string, modTime time.Time, sharedHost bool, profileHome string, profileSessionRoots []string, profileAccountID, hostAccountID string) bool {
	// Resume-only imports from other accounts must never feed quota.
	if profileHome != "" && isCrossAccountResumeSession(profileHome, path) {
		return false
	}
	if rolloutAccountID := readRolloutAccountID(path); rolloutAccountID != "" && profileAccountID != "" {
		if !strings.EqualFold(rolloutAccountID, profileAccountID) {
			return false
		}
		// Verifiable account match wins regardless of path locality.
		if !sharedHost {
			return true
		}
	}
	for _, root := range profileSessionRoots {
		if config.FilesystemPathWithin(root, path) {
			// Path under this profile is enough only when the rollout has no
			// foreign account id (already checked above).
			return true
		}
	}
	// Isolated path under profile home that is not the shared host store.
	if !sharedHost && profileHome != "" && config.FilesystemPathWithin(profileHome, path) {
		return true
	}
	if !sharedHost {
		return false
	}
	if profileAccountID == "" || hostAccountID == "" {
		return false
	}
	if !strings.EqualFold(profileAccountID, hostAccountID) {
		return false
	}
	if hostOwnedAt(profileAccountID, modTime, hostAccountID) {
		return true
	}
	// Current host login matches this profile. Allow rollouts at/after the most
	// recent ledger observation for this account (covers omega→gmail switch
	// where hostOwnedAt would still attribute pre-switch ownership).
	return hostCurrentOwnerAllows(profileAccountID, modTime, hostAccountID)
}

// hostCurrentOwnerAllows attributes host rollouts written while the profile's
// account is the current host login, starting at the latest ledger observation
// for that account (or any recent file when the ledger has not yet recorded it).
func hostCurrentOwnerAllows(accountID string, at time.Time, currentHostAccountID string) bool {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" || !strings.EqualFold(accountID, strings.TrimSpace(currentHostAccountID)) {
		return false
	}
	entries := readHostAuthLedger()
	var lastObs time.Time
	found := false
	for _, e := range entries {
		if strings.EqualFold(e.AccountID, accountID) {
			if !found || e.ObservedAt.After(lastObs) {
				lastObs = e.ObservedAt
				found = true
			}
		}
	}
	if !found {
		// Fail closed: being the current host login is not evidence that this
		// account produced a rollout written at an unknown point in the past.
		return false
	}
	return !at.Before(lastObs)
}

func buildWindowsFromRateLimits(limits rolloutRateLimits) []model.UsageWindow {
	type slot struct {
		w    *rateLimitWindow
		fall string
	}
	slots := []slot{
		{limits.Primary, "5h"},
		{limits.Secondary, "weekly"},
	}
	var windows []model.UsageWindow
	now := time.Now()
	for _, s := range slots {
		if s.w == nil {
			continue
		}
		kind := windowKindFromMinutes(s.w.WindowMinutes, s.fall)
		used := s.w.UsedPercent
		rem := 100.0 - used
		if rem < 0 {
			rem = 0
		}
		var resetTime *time.Time
		resetDesc := formatCodexResetTime(s.w.ResetsAt)
		if s.w.ResetsAt > 0 {
			t := time.Unix(s.w.ResetsAt, 0)
			resetTime = &t
			if !t.After(now) {
				resetDesc = "window rolled over"
			}
		}
		windows = append(windows, model.UsageWindow{
			Kind:             kind,
			Group:            "claude_gpt",
			RemainingPercent: &rem,
			UsedPercent:      &used,
			ResetTime:        resetTime,
			ResetDescription: resetDesc,
		})
	}
	return windows
}

func windowKindFromMinutes(minutes int, fallback string) string {
	switch minutes {
	case 300:
		return "5h"
	case 10080:
		return "weekly"
	default:
		if fallback != "" {
			return fallback
		}
		if minutes > 0 && minutes <= 360 {
			return "5h"
		}
		if minutes >= 7*24*60 {
			return "weekly"
		}
		return fallback
	}
}

// readLatestRateLimitsFromEnd scans a rollout JSONL from the end until it finds
// the latest event carrying rate_limits.primary. Avoids reading multi-MB files fully.
func readLatestRateLimitsFromEnd(path string) (rolloutUsageHit, bool) {
	f, err := os.Open(path)
	if err != nil {
		return rolloutUsageHit{}, false
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return rolloutUsageHit{}, false
	}
	size := stat.Size()
	if size == 0 {
		return rolloutUsageHit{}, false
	}

	var (
		hit     rolloutUsageHit
		carry   []byte
		offset  = size
		scanned int64
	)
	for offset > 0 && scanned < 2*1024*1024 {
		chunk := int64(reverseReadBlock)
		if offset < chunk {
			chunk = offset
		}
		offset -= chunk
		buf := make([]byte, chunk)
		if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
			break
		}
		scanned += chunk
		data := append(buf, carry...)
		lines := strings.Split(string(data), "\n")
		// First element may be a partial line — keep as carry for next (earlier) block.
		if offset > 0 {
			carry = []byte(lines[0])
			lines = lines[1:]
		} else {
			carry = nil
		}
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			if !strings.Contains(line, `"rate_limits"`) && !strings.Contains(line, `"thread_settings_applied"`) {
				continue
			}
			var ev struct {
				Timestamp string `json:"timestamp"`
				Type      string `json:"type"`
				Payload   struct {
					Type           string            `json:"type"`
					RateLimits     rolloutRateLimits `json:"rate_limits"`
					ThreadSettings struct {
						Model string `json:"model"`
					} `json:"thread_settings"`
				} `json:"payload"`
			}
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			if ev.Type == "event_msg" && ev.Payload.Type == "thread_settings_applied" && hit.modelName == "" {
				hit.modelName = ev.Payload.ThreadSettings.Model
			}
			// Accept any event carrying primary rate_limits (token_count or splash variants).
			if ev.Payload.RateLimits.Primary != nil {
				hit.limits = ev.Payload.RateLimits
				if t, err := time.Parse(time.RFC3339Nano, ev.Timestamp); err == nil {
					hit.fetchedAt = t
				} else if t, err := time.Parse(time.RFC3339, ev.Timestamp); err == nil {
					hit.fetchedAt = t
				}
				return hit, true
			}
		}
	}
	return hit, false
}

func readCodexAuthAccountID(authPath string) string {
	data, err := os.ReadFile(authPath)
	if err != nil {
		return ""
	}
	accountID, _, _, _ := parseCodexAuthBytes(data)
	return accountID
}

func parseCodexAuthBytes(data []byte) (accountID, email, plan, activeUntil string) {
	var auth struct {
		Tokens struct {
			IDToken   string `json:"id_token"`
			AccountID string `json:"account_id"`
		} `json:"tokens"`
	}
	if json.Unmarshal(data, &auth) != nil {
		return "", "", "", ""
	}
	accountID = strings.TrimSpace(auth.Tokens.AccountID)

	if auth.Tokens.IDToken == "" {
		return accountID, "", "", ""
	}
	parts := strings.Split(auth.Tokens.IDToken, ".")
	if len(parts) < 2 {
		return accountID, "", "", ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return accountID, "", "", ""
		}
	}
	var claims struct {
		Email      string `json:"email"`
		OpenAIAuth struct {
			ChatGPTAccountID string `json:"chatgpt_account_id"`
			PlanType         string `json:"chatgpt_plan_type"`
			ActiveUntil      string `json:"chatgpt_subscription_active_until"`
		} `json:"https://api.openai.com/auth"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return accountID, "", "", ""
	}
	email = strings.TrimSpace(claims.Email)
	if accountID == "" {
		accountID = strings.TrimSpace(claims.OpenAIAuth.ChatGPTAccountID)
	}
	if claims.OpenAIAuth.PlanType != "" {
		if claims.OpenAIAuth.PlanType == "pro" {
			plan = "ChatGPT Pro"
		} else {
			plan = "ChatGPT " + titlePlan(claims.OpenAIAuth.PlanType)
		}
	}
	activeUntil = strings.TrimSpace(claims.OpenAIAuth.ActiveUntil)
	return accountID, email, plan, activeUntil
}

func formatCodexResetTime(epochSec int64) string {
	if epochSec <= 0 {
		return "Quota available"
	}
	t := time.Unix(epochSec, 0)
	now := time.Now()
	if !t.After(now) {
		return "window rolled over"
	}
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return fmt.Sprintf("resets %s", t.Format("15:04"))
	}
	return fmt.Sprintf("resets %s on %d %s", t.Format("15:04"), t.Day(), t.Format("Jan"))
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session

	homeCandidates := []string{}
	if h, err := config.ProfileHome(string(a.ID()), p.Name); err == nil {
		homeCandidates = append(homeCandidates, h, filepath.Join(h, ".codex"))
	}
	if root, err := config.ProfileRoot(string(a.ID()), p.Name); err == nil {
		homeCandidates = append(homeCandidates, root)
	}
	if hostHome := security.FindHostHome(); hostHome != "" {
		homeCandidates = append(homeCandidates, filepath.Join(hostHome, ".codex"), hostHome)
	}
	if userProf := os.Getenv("USERPROFILE"); userProf != "" {
		homeCandidates = append(homeCandidates, filepath.Join(userProf, ".codex"), userProf)
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		homeCandidates = append(homeCandidates, filepath.Join(appData, "codex"), filepath.Join(appData, "OpenAI", "Codex"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		homeCandidates = append(homeCandidates, filepath.Join(localAppData, "codex"), filepath.Join(localAppData, "OpenAI", "Codex"))
	}

	seen := make(map[string]bool)
	for _, h := range homeCandidates {
		indexFile := filepath.Join(h, "session_index.jsonl")
		f, err := os.Open(indexFile)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var entry struct {
				ID         string `json:"id"`
				ThreadName string `json:"thread_name"`
				UpdatedAt  string `json:"updated_at"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil && entry.ID != "" {
				if seen[entry.ID] {
					continue
				}
				seen[entry.ID] = true

				t, _ := time.Parse(time.RFC3339, entry.UpdatedAt)
				title := entry.ThreadName
				if title == "" {
					title = "Codex Thread " + entry.ID[:8]
				}
				sessions = append(sessions, model.Session{
					ProviderID:      string(a.ID()),
					ProfileID:       p.Name,
					ID:              entry.ID,
					Title:           title,
					Workspace:       workspace,
					CreatedAt:       t,
					UpdatedAt:       t,
					ResumeSupported: true,
				})
			}
		}
		f.Close()
	}

	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	if err := adoptSessionIntoProfile(p.Name, sessionID); err != nil {
		// Best-effort: still attempt resume; Codex may find the session via host index.
		_ = err
	}
	resumeArgs := []string{"resume", sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}

func ensureConfigFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte("cli_auth_credentials_store = \"file\"\n"), 0600)
		}
		return err
	}

	content := string(raw)
	if !strings.Contains(content, "cli_auth_credentials_store") {
		content += "\ncli_auth_credentials_store = \"file\"\n"
		return os.WriteFile(path, []byte(content), 0600)
	}
	return nil
}

// linkSharedCodexItems shares non-session artifacts only. Session history must
// stay isolated per profile so quota attribution is deterministic.
func linkSharedCodexItems(profileHome, hostCodex string) {
	items := []string{
		"rules",
		"skills",
		"customizations",
	}

	for _, item := range items {
		src := filepath.Join(hostCodex, item)
		dst := filepath.Join(profileHome, item)

		if _, err := os.Stat(src); err != nil {
			continue
		}

		_ = security.SafeLinkOrCopy(src, dst)
	}
}

// migrateAwayFromSharedSessions replaces a symlink to the host sessions store
// with a real empty directory. Host data is never deleted.
func migrateAwayFromSharedSessions(profileHome string) {
	hostHome := security.FindHostHome()
	hostSessions := ""
	if hostHome != "" {
		hostSessions = filepath.Join(hostHome, ".codex", "sessions")
	}
	for _, name := range []string{"sessions", "session_index.jsonl", "thread_history_1.sqlite", "thread_history_1.sqlite-wal", "thread_history_1.sqlite-shm"} {
		path := filepath.Join(profileHome, name)
		fi, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			target, _ = os.Readlink(path)
		}
		shared := false
		if hostSessions != "" && name == "sessions" {
			shared = config.FilesystemPathWithin(hostSessions, target) || config.FilesystemPathsEquivalent(target, hostSessions)
		}
		if hostHome != "" && strings.HasPrefix(name, "thread_history") {
			hostItem := filepath.Join(hostHome, ".codex", name)
			shared = config.FilesystemPathsEquivalent(target, hostItem)
		}
		if hostHome != "" && name == "session_index.jsonl" {
			shared = config.FilesystemPathsEquivalent(target, filepath.Join(hostHome, ".codex", "session_index.jsonl"))
		}
		if !shared && hostHome != "" {
			// Also treat any symlink whose target lives under host ~/.codex as shared.
			hostCodex := filepath.Join(hostHome, ".codex")
			shared = config.FilesystemPathWithin(hostCodex, target)
		}
		if !shared {
			continue
		}
		_ = os.Remove(path)
		if name == "sessions" {
			_ = os.MkdirAll(path, 0700)
		}
	}
}

// adoptSessionIntoProfile is defined in cross_account_sessions.go.

// tryAdoptLatestHostRollout copies the newest host rollout that belongs to this
// profile's chatgpt_account_id into the isolated sessions store. Returns true
// when a file was adopted (caller should re-scan).
func (a *Adapter) tryAdoptLatestHostRollout(ctx context.Context, p model.Profile) bool {
	info := a.InspectAuth(ctx, p)
	profileAccountID := strings.TrimSpace(info.ExternalAccountID)
	if profileAccountID == "" {
		return false
	}
	hostHome := security.FindHostHome()
	if hostHome == "" {
		return false
	}
	hostAccountID := readCodexAuthAccountID(filepath.Join(hostHome, ".codex", "auth.json"))
	if hostAccountID == "" || !strings.EqualFold(profileAccountID, hostAccountID) {
		return false
	}
	recordHostAuthObservation(hostAccountID)

	hostSessions := filepath.Join(hostHome, ".codex", "sessions")
	type cand struct {
		path    string
		modTime time.Time
	}
	var cands []cand
	cutoff := time.Now().Add(-rolloutMaxAge)
	_ = filepath.Walk(hostSessions, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi == nil || fi.IsDir() {
			return nil
		}
		if !strings.HasPrefix(fi.Name(), "rollout-") || !strings.HasSuffix(fi.Name(), ".jsonl") {
			return nil
		}
		if fi.ModTime().Before(cutoff) {
			return nil
		}
		if !rolloutBelongsToProfile(path, fi.ModTime(), true, "", nil, profileAccountID, hostAccountID) {
			return nil
		}
		hit, ok := readLatestRateLimitsFromEnd(path)
		if !ok || hit.limits.Primary == nil {
			return nil
		}
		cands = append(cands, cand{path: path, modTime: fi.ModTime()})
		return nil
	})
	if len(cands) == 0 {
		return false
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].modTime.After(cands[j].modTime)
	})
	src := cands[0].path
	sessionID := extractSessionIDFromRolloutName(filepath.Base(src))
	if sessionID == "" {
		// Fall back to copying by relative path under host sessions.
		home, err := config.ProfileHome("codex", p.Name)
		if err != nil {
			return false
		}
		destRoot := filepath.Join(home, "sessions")
		_ = os.MkdirAll(destRoot, 0700)
		rel, err := filepath.Rel(hostSessions, src)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = filepath.Base(src)
		}
		dst := filepath.Join(destRoot, rel)
		if _, err := os.Stat(dst); err == nil {
			return false
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return false
		}
		return copyFile(src, dst) == nil
	}
	before := ""
	if home, err := config.ProfileHome("codex", p.Name); err == nil {
		before = findRolloutInTree(filepath.Join(home, "sessions"), sessionID)
	}
	if err := adoptSessionIntoProfile(p.Name, sessionID); err != nil {
		return false
	}
	if home, err := config.ProfileHome("codex", p.Name); err == nil {
		after := findRolloutInTree(filepath.Join(home, "sessions"), sessionID)
		return after != "" && after != before
	}
	return true
}

func extractSessionIDFromRolloutName(name string) string {
	// rollout-2026-09-11T19-50-06-<session-id>.jsonl
	name = strings.TrimSuffix(name, ".jsonl")
	if !strings.HasPrefix(name, "rollout-") {
		return ""
	}
	rest := strings.TrimPrefix(name, "rollout-")
	// Timestamp is YYYY-MM-DDTHH-MM-SS then '-' then session id.
	parts := strings.SplitN(rest, "-", 7)
	if len(parts) < 7 {
		// Try: after the date-time portion (19 chars date + T + time).
		idx := strings.LastIndex(rest, "-")
		// Walk back looking for UUID-like suffix after datetime.
		if idx <= 0 {
			return ""
		}
		// Prefer everything after first 19-ish timestamp chars.
		if len(rest) > 20 && rest[10] == 'T' {
			return rest[20:] // skip "YYYY-MM-DDTHH-MM-SS-"
		}
		return rest[idx+1:]
	}
	// parts: YYYY MM DDTHH MM SS sessionid... when SplitN by '-' on ISO-ish name
	// Actually "2026-09-11T19-50-06-sess1" SplitN("-", 7) =>
	// [2026, 09, 11T19, 50, 06, sess1] — only 6 parts.
	if len(rest) > 20 && rest[10] == 'T' {
		return rest[20:]
	}
	return ""
}

func findRolloutInTree(root, sessionID string) string {
	if root == "" {
		return ""
	}
	var found string
	_ = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi == nil || fi.IsDir() || found != "" {
			return nil
		}
		name := fi.Name()
		if strings.HasPrefix(name, "rollout-") && strings.HasSuffix(name, ".jsonl") && strings.Contains(name, sessionID) {
			found = path
			return io.EOF
		}
		return nil
	})
	return found
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
