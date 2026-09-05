package codex

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

type Adapter struct{}

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
	bin, err := runtime.LookPath("codex")
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
	dotCodex := filepath.Join(home, ".codex")
	for _, d := range []string{home, dotCodex} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}

	// Apply isolation preset policies
	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	if err := security.ApplyIsolation(home, policy); err != nil {
		return err
	}

	// Configure file auth store inside isolated home and .codex
	configFile := filepath.Join(home, "config.toml")
	_ = ensureConfigFile(configFile)
	dotConfigFile := filepath.Join(dotCodex, "config.toml")
	_ = ensureConfigFile(dotConfigFile)

	// Sync auth.json between home and home/.codex if one exists
	authHome := filepath.Join(home, "auth.json")
	authDot := filepath.Join(dotCodex, "auth.json")
	if data, err := os.ReadFile(authHome); err == nil && len(data) > 0 {
		_ = os.WriteFile(authDot, data, 0600)
	} else if data, err := os.ReadFile(authDot); err == nil && len(data) > 0 {
		_ = os.WriteFile(authHome, data, 0600)
	}

	// Link shared non-credential artifacts (sessions, rules, skills, sqlite indices)
	hostHome := security.FindHostHome()
	if hostHome != "" {
		hostCodex := filepath.Join(hostHome, ".codex")
		if _, err := os.Stat(hostCodex); err == nil {
			linkSharedCodexItems(home, hostCodex)
			linkSharedCodexItems(dotCodex, hostCodex)
		}
	}

	return nil
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := runtime.LookPath("codex")
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "codex binary not found"}, err
	}
	home, _ := config.ProfileHome(string(a.ID()), p.Name)
	cwd, _ := os.Getwd()
	envOverrides := map[string]string{
		"HOME":             home,
		"CODEX_HOME":       home,
		"CODEX_CONFIG_DIR": home,
		"XDG_CONFIG_HOME":  filepath.Join(home, ".config"),
		"XDG_CACHE_HOME":   filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":    filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":   filepath.Join(home, ".local", "state"),
		"PATH":             runtime.EnhancedPATH(filepath.Dir(bin)),
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

	var authStore struct {
		Tokens struct {
			IDToken     string `json:"id_token"`
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
		AuthMode string `json:"auth_mode"`
	}

	if json.Unmarshal(data, &authStore) != nil || authStore.Tokens.IDToken == "" {
		return info
	}

	info.Authenticated = true
	info.Status = "Authenticated"
	info.Health = model.HealthHealthy
	info.Plan = "ChatGPT Plus"

	// Parse JWT claims safely
	parts := strings.Split(authStore.Tokens.IDToken, ".")
	if len(parts) >= 2 {
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err == nil {
			var claims struct {
				Email      string `json:"email"`
				OpenAIAuth struct {
					PlanType    string `json:"chatgpt_plan_type"`
					ActiveUntil string `json:"chatgpt_subscription_active_until"`
				} `json:"https://api.openai.com/auth"`
			}
			if json.Unmarshal(payload, &claims) == nil {
				if claims.Email != "" {
					info.Email = claims.Email
				}
				if claims.OpenAIAuth.PlanType != "" {
					if claims.OpenAIAuth.PlanType == "pro" {
						info.Plan = "ChatGPT Pro"
					} else {
						info.Plan = "ChatGPT " + strings.Title(strings.ToLower(claims.OpenAIAuth.PlanType))
					}
					if claims.OpenAIAuth.ActiveUntil != "" {
						if t, err := time.Parse(time.RFC3339, claims.OpenAIAuth.ActiveUntil); err == nil {
							info.ExpiresAt = t
						}
					}
				}
			}
		}
	}

	return info
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	snap := model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}

	// 1. Try to read live rate limits from recent session rollouts
	if rollSnap, ok := a.getUsageFromRollouts(ctx, p); ok {
		return rollSnap
	}

	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home, _ := config.ProfileHome(string(a.ID()), p.Name)

	candidates := []string{}
	if root != "" {
		candidates = append(candidates, filepath.Join(root, "quota.json"), filepath.Join(root, "usage.json"))
	}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, "usage.json"), filepath.Join(home, "quota.json"))
	}

	for _, file := range candidates {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var s model.UsageSnapshot
		if json.Unmarshal(data, &s) == nil && s.Status != "" && len(s.Windows) > 0 {
			return s
		}

		var leg struct {
			ModelName string `json:"model_name"`
			FiveHour  struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
			} `json:"five_hour"`
			Weekly struct {
				PercentLeft float64 `json:"percent_left"`
				ResetTime   string  `json:"reset_time"`
			} `json:"weekly"`
		}
		if json.Unmarshal(data, &leg) == nil && (leg.FiveHour.PercentLeft > 0 || leg.Weekly.PercentLeft > 0 || leg.FiveHour.ResetTime != "" || leg.Weekly.ResetTime != "") {
			p5h := leg.FiveHour.PercentLeft
			u5h := 100.0 - p5h
			pWk := leg.Weekly.PercentLeft
			uWk := 100.0 - pWk

			return model.UsageSnapshot{
				ProviderID: string(a.ID()),
				ProfileID:  p.Name,
				Status:     model.UsageCached,
				Source:     model.SourceLocalFiles,
				ModelName:  leg.ModelName,
				FetchedAt:  fileModTime(file),
				Windows: []model.UsageWindow{
					{
						Kind:             "5h",
						Group:            "claude_gpt",
						RemainingPercent: &p5h,
						UsedPercent:      &u5h,
						ResetDescription: leg.FiveHour.ResetTime,
					},
					{
						Kind:             "weekly",
						Group:            "claude_gpt",
						RemainingPercent: &pWk,
						UsedPercent:      &uWk,
						ResetDescription: leg.Weekly.ResetTime,
					},
				},
			}
		}
	}

	return snap
}

func (a *Adapter) getUsageFromRollouts(ctx context.Context, p model.Profile) (model.UsageSnapshot, bool) {
	info := a.InspectAuth(ctx, p)
	targetEmail := strings.TrimSpace(info.Email)
	hostHome := security.FindHostHome()
	hostSessions := ""
	hostAuthEmail := ""
	if hostHome != "" {
		hostSessions = filepath.Join(hostHome, ".codex", "sessions")
		hostAuthEmail = readCodexAuthEmail(filepath.Join(hostHome, ".codex", "auth.json"))
	}

	dirs := []string{}
	if h, err := config.ProfileHome(string(a.ID()), p.Name); err == nil {
		dirs = append(dirs, filepath.Join(h, "sessions"), filepath.Join(h, ".codex", "sessions"))
	}
	if hostSessions != "" {
		dirs = append(dirs, hostSessions)
	}

	type fileInfo struct {
		path       string
		modTime    time.Time
		sharedHost bool
	}
	seen := map[string]bool{}
	var rolloutFiles []fileInfo

	for _, dir := range dirs {
		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			realDir = dir
		}
		if _, err := os.Stat(realDir); err != nil {
			continue
		}
		sharedHost := hostSessions != "" && (realDir == hostSessions || strings.HasPrefix(realDir+string(filepath.Separator), hostSessions+string(filepath.Separator)))
		_ = filepath.Walk(realDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi == nil || fi.IsDir() {
				return nil
			}
			if !strings.HasPrefix(fi.Name(), "rollout-") || !strings.HasSuffix(fi.Name(), ".jsonl") {
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

	if len(rolloutFiles) == 0 {
		return model.UsageSnapshot{}, false
	}

	sort.Slice(rolloutFiles, func(i, j int) bool {
		return rolloutFiles[i].modTime.After(rolloutFiles[j].modTime)
	})

	maxCheck := 30
	if len(rolloutFiles) < maxCheck {
		maxCheck = len(rolloutFiles)
	}

	for _, rf := range rolloutFiles[:maxCheck] {
		f, err := os.Open(rf.path)
		if err != nil {
			continue
		}

		var lastRateLimit struct {
			Primary *struct {
				UsedPercent   float64 `json:"used_percent"`
				WindowMinutes int     `json:"window_minutes"`
				ResetsAt      int64   `json:"resets_at"`
			} `json:"primary"`
			Secondary *struct {
				UsedPercent   float64 `json:"used_percent"`
				WindowMinutes int     `json:"window_minutes"`
				ResetsAt      int64   `json:"resets_at"`
			} `json:"secondary"`
		}
		var detectedModel string
		matchedAccount := false

		scanner := bufio.NewScanner(f)
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 10*1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// Check account match if email is present
			if targetEmail != "" && !matchedAccount && strings.Contains(string(line), targetEmail) {
				matchedAccount = true
			}

			if strings.Contains(string(line), `"token_count"`) {
				var ev struct {
					Type    string `json:"type"`
					Payload struct {
						Type       string `json:"type"`
						RateLimits struct {
							Primary *struct {
								UsedPercent   float64 `json:"used_percent"`
								WindowMinutes int     `json:"window_minutes"`
								ResetsAt      int64   `json:"resets_at"`
							} `json:"primary"`
							Secondary *struct {
								UsedPercent   float64 `json:"used_percent"`
								WindowMinutes int     `json:"window_minutes"`
								ResetsAt      int64   `json:"resets_at"`
							} `json:"secondary"`
						} `json:"rate_limits"`
					} `json:"payload"`
				}
				if json.Unmarshal(line, &ev) == nil && ev.Type == "event_msg" && ev.Payload.Type == "token_count" {
					if ev.Payload.RateLimits.Primary != nil {
						lastRateLimit.Primary = ev.Payload.RateLimits.Primary
					}
					if ev.Payload.RateLimits.Secondary != nil {
						lastRateLimit.Secondary = ev.Payload.RateLimits.Secondary
					}
				}
			}

			if detectedModel == "" && strings.Contains(string(line), `"thread_settings_applied"`) {
				var ts struct {
					Payload struct {
						Type           string `json:"type"`
						ThreadSettings struct {
							Model string `json:"model"`
						} `json:"thread_settings"`
					} `json:"payload"`
				}
				if json.Unmarshal(line, &ts) == nil && ts.Payload.ThreadSettings.Model != "" {
					detectedModel = ts.Payload.ThreadSettings.Model
				}
			}
		}
		f.Close()

		// Account matching for shared ~/.codex/sessions (often symlinked from every
		// profile home): modern rollouts rarely embed the email, so claim them only
		// when this profile owns the current host Codex login.
		isProfilePath := strings.Contains(rf.path, p.Name)
		belongs := matchedAccount || isProfilePath
		if !belongs && rf.sharedHost {
			belongs = targetEmail != "" && hostAuthEmail != "" && strings.EqualFold(targetEmail, hostAuthEmail)
		}
		if !belongs {
			continue
		}

		if lastRateLimit.Primary != nil {
			// Codex rate_limits expose used_percent; UI "/status" shows remaining.
			pRem := 100.0 - lastRateLimit.Primary.UsedPercent
			if pRem < 0 {
				pRem = 0
			}
			pUsed := lastRateLimit.Primary.UsedPercent
			r5h := formatCodexResetTime(lastRateLimit.Primary.ResetsAt)

			windows := []model.UsageWindow{
				{
					Kind:             "5h",
					Group:            "claude_gpt",
					RemainingPercent: &pRem,
					UsedPercent:      &pUsed,
					ResetDescription: r5h,
				},
			}

			if lastRateLimit.Secondary != nil {
				sRem := 100.0 - lastRateLimit.Secondary.UsedPercent
				if sRem < 0 {
					sRem = 0
				}
				sUsed := lastRateLimit.Secondary.UsedPercent
				rWk := formatCodexResetTime(lastRateLimit.Secondary.ResetsAt)
				windows = append(windows, model.UsageWindow{
					Kind:             "weekly",
					Group:            "claude_gpt",
					RemainingPercent: &sRem,
					UsedPercent:      &sUsed,
					ResetDescription: rWk,
				})
			}

			if detectedModel == "" {
				detectedModel = "gpt-5.6-terra"
			}

			return model.UsageSnapshot{
				ProviderID: string(a.ID()),
				ProfileID:  p.Name,
				Account:    targetEmail,
				Status:     model.UsageLive,
				Source:     model.SourceObservation,
				ModelName:  detectedModel,
				FetchedAt:  rf.modTime,
				Windows:    windows,
			}, true
		}
	}

	return model.UsageSnapshot{}, false
}

func readCodexAuthEmail(authPath string) string {
	data, err := os.ReadFile(authPath)
	if err != nil {
		return ""
	}
	var auth struct {
		Tokens struct {
			IDToken string `json:"id_token"`
		} `json:"tokens"`
	}
	if json.Unmarshal(data, &auth) != nil || auth.Tokens.IDToken == "" {
		return ""
	}
	parts := strings.Split(auth.Tokens.IDToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some tokens include padding.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}
	var claims struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	return strings.TrimSpace(claims.Email)
}

func fileModTime(path string) time.Time {
	if st, err := os.Stat(path); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}

func formatCodexResetTime(epochSec int64) string {
	if epochSec <= 0 {
		return "Quota available"
	}
	t := time.Unix(epochSec, 0)
	now := time.Now()
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

func linkSharedCodexItems(profileHome, hostCodex string) {
	items := []string{
		"session_index.jsonl",
		"thread_history_1.sqlite",
		"thread_history_1.sqlite-wal",
		"thread_history_1.sqlite-shm",
		"sessions",
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
