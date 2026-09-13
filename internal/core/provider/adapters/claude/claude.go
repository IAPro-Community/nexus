package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/spec"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

// Adapter implements the Provider interface for Anthropic Claude Code.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID { return model.ProviderClaude }
func (a *Adapter) Name() string         { return "Claude Code" }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login:              true,
		Logout:             true,
		Usage:              true,
		Conversations:      true,
		Resume:             true,
		CrossAccountResume: false,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := spec.MustGet("claude").ResolveBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "binary not found in PATH"}
	}
	out, err := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	version := strings.TrimSpace(out)
	if err != nil && version == "" {
		return model.DetectionResult{Installed: true, BinaryPath: bin, Version: "unknown"}
	}
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    version,
	}
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	claudeDir := filepath.Join(home, ".claude")
	configDir := filepath.Join(home, ".config", "claude")
	for _, d := range []string{home, claudeDir, configDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}

	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	return security.ApplyIsolation(home, policy)
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := spec.MustGet("claude").ResolveBinary()
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "claude binary not found"}, err
	}
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home := filepath.Join(root, "home")
	cwd, _ := os.Getwd()

	envMap := map[string]string{
		"HOME":              home,
		"CLAUDE_CONFIG_DIR": filepath.Join(home, ".claude"),
		"XDG_CONFIG_HOME":   filepath.Join(home, ".config"),
		"PATH":              runtime.EnhancedPATH(filepath.Dir(bin)),
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		envMap["SSH_AUTH_SOCK"] = sock
	}

	env := runtime.EnvSet(os.Environ(), envMap)
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, []string{"login"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.Remove(filepath.Join(home, ".claude.json"))
	_ = os.RemoveAll(filepath.Join(home, ".claude"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")

	info := model.AccountInfo{
		Plan:          "Claude Pro / Team",
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

	// Identity from ~/.claude.json is informative but not proof of a live token.
	claudeJson := filepath.Join(home, ".claude.json")
	if data, err := os.ReadFile(claudeJson); err == nil {
		var cData struct {
			OAuthAccount struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"oauthAccount"`
			UserID string `json:"userID"`
		}
		if json.Unmarshal(data, &cData) == nil {
			if cData.OAuthAccount.Email != "" {
				info.Email = cData.OAuthAccount.Email
			} else if cData.UserID != "" {
				info.Email = cData.UserID
			}
		}
	}

	cred, ok := readClaudeCredentials(home)
	if !ok {
		// Preserve prior fail-open for identity-only installs that never wrote
		// a credentials file: email/userID in .claude.json still counts.
		if info.Email != "" {
			info.Authenticated = true
			info.Status = "Authenticated"
			info.Health = model.HealthHealthy
		}
		return info
	}
	if cred.Email != "" {
		info.Email = cred.Email
	}
	hasToken := cred.AccessToken != "" || cred.RefreshToken != ""
	if !hasToken && info.Email == "" {
		return info
	}
	if !hasToken {
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
		return info
	}

	applyClaudeTokenExpiry(&info, cred)
	return info
}

type claudeCredentialBlob struct {
	Email        string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	HasExpiry    bool
}

func readClaudeCredentials(home string) (claudeCredentialBlob, bool) {
	candidates := []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(home, ".claude", "credentials.json"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			continue
		}
		if cred, ok := parseClaudeCredentialsJSON(data); ok {
			return cred, true
		}
	}
	return claudeCredentialBlob{}, false
}

func parseClaudeCredentialsJSON(data []byte) (claudeCredentialBlob, bool) {
	// Flat schema used by older Nexus writes / some Claude builds.
	var flat struct {
		OAuthAccount struct {
			Email string `json:"email"`
		} `json:"oauthAccount"`
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		Email        string `json:"email"`
		ExpiresAt    any    `json:"expiresAt"`
		Expiry       any    `json:"expiry"`
	}
	if json.Unmarshal(data, &flat) == nil {
		email := strings.TrimSpace(flat.OAuthAccount.Email)
		if email == "" {
			email = strings.TrimSpace(flat.Email)
		}
		access := strings.TrimSpace(flat.AccessToken)
		refresh := strings.TrimSpace(flat.RefreshToken)
		exp, hasExp := parseClaudeExpiry(flat.ExpiresAt)
		if !hasExp {
			exp, hasExp = parseClaudeExpiry(flat.Expiry)
		}
		if email != "" || access != "" || refresh != "" {
			return claudeCredentialBlob{
				Email: email, AccessToken: access, RefreshToken: refresh,
				ExpiresAt: exp, HasExpiry: hasExp,
			}, true
		}
	}

	// Nested Claude Code schema: { "claudeAiOauth": { ... } }
	var nested struct {
		ClaudeAiOauth struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresAt    any    `json:"expiresAt"`
			Email        string `json:"email"`
		} `json:"claudeAiOauth"`
	}
	if json.Unmarshal(data, &nested) == nil {
		access := strings.TrimSpace(nested.ClaudeAiOauth.AccessToken)
		refresh := strings.TrimSpace(nested.ClaudeAiOauth.RefreshToken)
		email := strings.TrimSpace(nested.ClaudeAiOauth.Email)
		exp, hasExp := parseClaudeExpiry(nested.ClaudeAiOauth.ExpiresAt)
		if email != "" || access != "" || refresh != "" {
			return claudeCredentialBlob{
				Email: email, AccessToken: access, RefreshToken: refresh,
				ExpiresAt: exp, HasExpiry: hasExp,
			}, true
		}
	}
	return claudeCredentialBlob{}, false
}

func parseClaudeExpiry(raw any) (time.Time, bool) {
	switch v := raw.(type) {
	case nil:
		return time.Time{}, false
	case float64:
		// Claude Code commonly stores epoch milliseconds.
		if v > 1e12 {
			return time.UnixMilli(int64(v)).UTC(), true
		}
		if v > 1e9 {
			return time.Unix(int64(v), 0).UTC(), true
		}
		return time.Time{}, false
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return time.Time{}, false
		}
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t.UTC(), true
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.UTC(), true
		}
		return time.Time{}, false
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return time.Time{}, false
		}
		return parseClaudeExpiry(f)
	default:
		return time.Time{}, false
	}
}

func applyClaudeTokenExpiry(info *model.AccountInfo, cred claudeCredentialBlob) {
	if !cred.HasExpiry {
		// No parseable expiry: keep prior fail-open so we do not invent a verdict.
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
		return
	}
	if time.Now().Before(cred.ExpiresAt) {
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
		info.ExpiresAt = cred.ExpiresAt
		return
	}
	info.ExpiresAt = cred.ExpiresAt
	if strings.TrimSpace(cred.RefreshToken) != "" {
		info.Authenticated = true
		info.Status = "Token expired (refreshable by Claude CLI)"
		info.Health = model.HealthDegraded
		return
	}
	info.Authenticated = false
	info.Status = "Token expired"
	info.Health = model.HealthAuthRequired
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	// Cache/last-known only — live Anthropic probes belong in RefreshUsage.
	if snap, ok := readProfileUsageJSON(string(a.ID()), p.Name); ok {
		return snap
	}
	snap := newClaudeUsageSnapshot(p)
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	_, found, credErr := readClaudeOAuthCredentials(home)
	if credErr != nil {
		snap.Error = credErr.Error()
		return snap
	}
	if !found {
		snap.Status = model.UsageUnsupported
	}
	return snap
}

// RefreshUsage always probes Anthropic's official OAuth usage endpoint.
func (a *Adapter) RefreshUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return a.getLiveUsage(ctx, p, true)
}

func readProfileUsageJSON(provider, profileName string) (model.UsageSnapshot, bool) {
	root, err := config.ProfileRoot(provider, profileName)
	if err != nil || root == "" {
		return model.UsageSnapshot{}, false
	}
	data, err := os.ReadFile(filepath.Join(root, "usage.json"))
	if err != nil {
		return model.UsageSnapshot{}, false
	}
	var snap model.UsageSnapshot
	if json.Unmarshal(data, &snap) != nil || snap.Status == "" || len(snap.Windows) == 0 {
		return model.UsageSnapshot{}, false
	}
	if snap.Status == model.UsageLive {
		snap.Status = model.UsageCached
	}
	snap.ProviderID = provider
	snap.ProfileID = profileName
	return snap, true
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return nil, err
	}
	projectsDir := filepath.Join(root, "home", ".claude", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return sessions, nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sessions = append(sessions, model.Session{
			ProviderID:      string(a.ID()),
			ProfileID:       p.Name,
			ID:              e.Name(),
			Title:           "Claude Session " + e.Name(),
			Workspace:       workspace,
			UpdatedAt:       time.Now(),
			ResumeSupported: true,
		})
	}

	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	// Claude official resume syntax: claude --resume <SESSION_ID>
	resumeArgs := []string{"--resume", sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}
