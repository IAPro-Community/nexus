package claude

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	sdkclaude "github.com/IAPro-Community/IAPro-SDK/providers/claude"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/historysync"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/authcopy"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/provider/versionprobe"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/runtime"
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
		CrossAccountResume: true,
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
	return versionprobe.Detect(ctx, bin)
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
	if err := security.ApplyIsolation(home, policy); err != nil {
		return err
	}
	if err := historysync.SyncProviderHistory(string(a.ID()), p.Name, home); err != nil {
		slog.Warn("Claude history sync warning", "profile", p.Name, "err", err)
	}
	return nil
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

	env := runtime.EnvSet(os.Environ(), runtime.ApplyProfileIsolationEnv(home, envMap))
	return runtime.RunInteractive(bin, args, env, cwd, string(a.ID()))
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, spec.MustGet("claude").LoginCommand())
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

// ImportHostAuth copies Claude Code's file-backed OAuth credential into an
// isolated Nexus profile. It deliberately excludes Claude history and config.
func (a *Adapter) ImportHostAuth(ctx context.Context, p model.Profile) (bool, error) {
	hostHome := security.FindHostHome()
	if hostHome == "" {
		return false, nil
	}
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return false, err
	}
	if err := a.Prepare(ctx, p); err != nil {
		return false, err
	}
	destination := filepath.Join(root, "home", ".claude", ".credentials.json")
	for _, relative := range []string{
		filepath.Join(".claude", ".credentials.json"),
		filepath.Join(".claude", "credentials.json"),
	} {
		copied, err := authcopy.CopyValidatedFile(
			filepath.Join(hostHome, relative), destination, validHostClaudeAuth,
		)
		if err != nil || !copied {
			if err != nil || errors.Is(err, authcopy.ErrDestinationExists) {
				return copied, err
			}
			continue
		}
		if !a.InspectAuth(ctx, p).Authenticated {
			return false, errors.New("host Claude session could not be verified in the Nexus profile")
		}
		// Extract minimal, non-sensitive identity metadata from host ~/.claude.json
		var data []byte
		if d, err := readClaudeIdentityJSON(hostHome); err == nil {
			data = d
		}
		if data != nil {
			var cData struct {
				OAuthAccount struct {
					Email        string `json:"email"`
					EmailAddress string `json:"emailAddress"`
					Name         string `json:"name"`
				} `json:"oauthAccount"`
				UserID string `json:"userID"`
			}
			if json.Unmarshal(data, &cData) == nil {
				email := cData.OAuthAccount.EmailAddress
				if email == "" {
					email = cData.OAuthAccount.Email
				}
				if email != "" || cData.UserID != "" {
					sanitized := map[string]any{
						"oauthAccount": map[string]any{
							"email":        email,
							"emailAddress": email,
							"name":         cData.OAuthAccount.Name,
						},
						"userID": cData.UserID,
					}
					if outBytes, err := json.MarshalIndent(sanitized, "", "  "); err == nil {
						_ = os.WriteFile(filepath.Join(root, "home", ".claude.json"), append(outBytes, '\n'), 0600)
					}
				}
			}
		}
		return true, nil
	}
	return false, nil
}

// DetectHostAuth inspects the host machine for active Claude credentials
// without modifying any profile or exposing secrets.
func (a *Adapter) DetectHostAuth(ctx context.Context) (model.HostAuthDetection, bool) {
	hostHome := security.FindHostHome()
	if hostHome == "" {
		return model.HostAuthDetection{}, false
	}
	cred, ok := readClaudeCredentials(hostHome)
	if !ok || (!cred.HasAccess && !cred.HasRefresh) {
		return model.HostAuthDetection{}, false
	}
	email := cred.Email
	if email == "" {
		if data, err := readClaudeIdentityJSON(hostHome); err == nil {
			var cData struct {
				OAuthAccount struct {
					Email        string `json:"email"`
					EmailAddress string `json:"emailAddress"`
				} `json:"oauthAccount"`
				UserID string `json:"userID"`
			}
			if json.Unmarshal(data, &cData) == nil {
				if cData.OAuthAccount.EmailAddress != "" {
					email = cData.OAuthAccount.EmailAddress
				} else if cData.OAuthAccount.Email != "" {
					email = cData.OAuthAccount.Email
				} else if cData.UserID != "" {
					email = cData.UserID
				}
			}
		}
	}
	installed := a.Detect(ctx).Installed
	return model.HostAuthDetection{
		Provider:  string(a.ID()),
		Installed: installed,
		Available: true,
		Email:     email,
		Plan:      "Claude Pro / Team",
	}, true
}

func validHostClaudeAuth(data []byte) bool {
	credential, ok := parseClaudeCredentialsJSON(data)
	if !ok || (!credential.HasAccess && !credential.HasRefresh) {
		return false
	}
	info := model.AccountInfo{}
	applyClaudeTokenExpiry(&info, credential)
	return info.Authenticated
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
	// Both known locations are accepted: recent Claude Code builds nest the
	// file under .claude/, legacy builds wrote it at the home root.
	if data, err := readClaudeIdentityJSON(home); err == nil {
		var cData struct {
			OAuthAccount struct {
				Email        string `json:"email"`
				EmailAddress string `json:"emailAddress"`
				Name         string `json:"name"`
			} `json:"oauthAccount"`
			UserID string `json:"userID"`
		}
		if json.Unmarshal(data, &cData) == nil {
			if cData.OAuthAccount.EmailAddress != "" {
				info.Email = cData.OAuthAccount.EmailAddress
			} else if cData.OAuthAccount.Email != "" {
				info.Email = cData.OAuthAccount.Email
			} else if cData.UserID != "" {
				info.Email = cData.UserID
			}
		}
	}

	cred, ok := readClaudeCredentials(home)
	if !ok {
		// Profile metadata is useful for display, but only OAuth credentials
		// prove that this isolated profile can authenticate.
		return info
	}
	if cred.Email != "" {
		info.Email = cred.Email
	}
	hasToken := cred.HasAccess || cred.HasRefresh
	if !hasToken {
		return info
	}

	applyClaudeTokenExpiry(&info, cred)
	return info
}

type claudeCredentialBlob = sdkclaude.CredentialHints

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
	return sdkclaude.ParseCredentialHints(data)
}

func parseClaudeExpiry(raw any) (time.Time, bool) {
	return sdkclaude.ParseCredentialExpiry(raw)
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
	if cred.HasRefresh {
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
