package opencode

import (
	"context"
	"encoding/json"
	"fmt"
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

// Adapter implements the Provider interface for OpenCode.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID { return model.ProviderOpenCode }
func (a *Adapter) Name() string         { return "OpenCode" }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		// OpenCode delegates authentication to its configured model provider.
		Login:         false,
		Logout:        false,
		Usage:         false, // Zen has no balance API; Anthropic upstream uses Claude probe.
		Conversations: true,
		Resume:        true,
		// Isolated per-profile session dir only — no sibling/host import path.
		CrossAccountResume: false,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := spec.MustGet("opencode").ResolveBinary()
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
	dirs := []string{
		home,
		filepath.Join(home, ".config", "opencode"),
		filepath.Join(home, ".local", "share", "opencode"),
		filepath.Join(home, ".cache", "opencode"),
	}
	for _, d := range dirs {
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
	bin, err := spec.MustGet("opencode").ResolveBinary()
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "opencode binary not found"}, err
	}
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home := filepath.Join(root, "home")
	cwd, _ := os.Getwd()

	envMap := map[string]string{
		"HOME":                home,
		"XDG_CONFIG_HOME":     filepath.Join(home, ".config"),
		"XDG_DATA_HOME":       filepath.Join(home, ".local", "share"),
		"XDG_CACHE_HOME":      filepath.Join(home, ".cache"),
		"OPENCODE_CONFIG_DIR": filepath.Join(home, ".config", "opencode"),
		"PATH":                runtime.EnhancedPATH(filepath.Dir(bin)),
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		envMap["SSH_AUTH_SOCK"] = sock
	}

	env := runtime.EnvSet(os.Environ(), envMap)
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	return fmt.Errorf("OpenCode authentication is provider-specific; run `opencode auth login <provider>` in the runtime where OpenCode will execute")
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.RemoveAll(filepath.Join(home, ".local", "share", "opencode"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")

	info := model.AccountInfo{
		Plan:          "OpenCode Provider",
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

	authFile := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	if data, err := os.ReadFile(authFile); err == nil {
		var aData struct {
			Email    string `json:"email"`
			Provider string `json:"provider"`
		}
		if json.Unmarshal(data, &aData) == nil && (aData.Email != "" || aData.Provider != "") {
			info.Email = aData.Email
			if info.Email == "" {
				info.Email = aData.Provider
			}
			info.Authenticated = true
			info.Status = "Authenticated"
			info.Health = model.HealthHealthy
		}
	}

	return info
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return nil, err
	}
	sessDir := filepath.Join(root, "home", ".local", "share", "opencode", "sessions")
	entries, err := os.ReadDir(sessDir)
	if err != nil {
		return sessions, nil
	}

	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".json")
		sessions = append(sessions, model.Session{
			ProviderID:      string(a.ID()),
			ProfileID:       p.Name,
			ID:              id,
			Title:           "OpenCode Session " + id,
			Workspace:       workspace,
			UpdatedAt:       time.Now(),
			ResumeSupported: true,
		})
	}

	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	// OpenCode official resume syntax: opencode -s <SESSION_ID>
	resumeArgs := []string{"-s", sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}
