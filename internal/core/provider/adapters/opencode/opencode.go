package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/runtime"
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
		// OpenCode supports provider credential management and local loginless execution.
		Login:              true,
		Logout:             true,
		Usage:              false, // Zen has no balance API; Anthropic upstream uses Claude probe.
		Conversations:      true,
		Resume:             true,
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
		filepath.Join(home, ".local", "share", "opencode"),
		filepath.Join(home, ".cache", "opencode"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}

	// Copy host auth.json into profile if profile does not have one yet
	profileAuth := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	if _, err := os.Stat(profileAuth); os.IsNotExist(err) {
		hostHome := config.BaseHome()
		if hostHome == "" {
			hostHome, _ = os.UserHomeDir()
		}
		if hostHome != "" {
			hostAuth := filepath.Join(hostHome, ".local", "share", "opencode", "auth.json")
			if hostData, rerr := os.ReadFile(hostAuth); rerr == nil && len(hostData) > 0 {
				_ = os.WriteFile(profileAuth, hostData, 0600)
			}
		}
	}

	cfgObj, _ := config.LoadConfig()
	policy := security.GetPolicy(cfgObj.IsolationPreset)
	if err := security.ApplyIsolationPreservingConfigDirs(home, policy, "opencode"); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0700)
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
	if err := a.Prepare(ctx, p); err != nil {
		return err
	}
	bin, err := spec.MustGet("opencode").ResolveBinary()
	if err != nil {
		return fmt.Errorf("opencode binary not found: %w", err)
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
	_, runErr := runtime.RunInteractive(bin, []string{"providers", "login"}, env, cwd)
	return runErr
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.Remove(filepath.Join(home, ".local", "share", "opencode", "auth.json"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")

	info := model.AccountInfo{
		Plan:          "OpenCode Local (Ollama/Env)",
		Status:        "Ready (Local / No login required)",
		Health:        model.HealthHealthy,
		Authenticated: true,
		Usage: model.UsageSnapshot{
			ProviderID: string(a.ID()),
			ProfileID:  p.Name,
			Status:     model.UsageUnsupported,
			Source:     model.SourceNone,
		},
	}

	if _, err := spec.MustGet("opencode").ResolveBinary(); err != nil {
		info.Status = "OpenCode binary not found"
		info.Health = model.HealthUnavailable
		info.Authenticated = false
		return info
	}

	authFile := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	data, err := os.ReadFile(authFile)
	if err != nil {
		hostHome := config.BaseHome()
		if hostHome == "" {
			hostHome, _ = os.UserHomeDir()
		}
		if hostHome != "" {
			hostAuthFile := filepath.Join(hostHome, ".local", "share", "opencode", "auth.json")
			data, _ = os.ReadFile(hostAuthFile)
		}
	}

	if len(data) > 0 {
		var auth openCodeAuthFile
		if json.Unmarshal(data, &auth) == nil && len(auth) > 0 {
			if entry, ok := auth["opencode"]; ok && firstNonEmpty(entry.Key, entry.Token, entry.Access) != "" {
				info.Plan = "OpenCode Zen"
				info.Email = "OpenCode Zen"
				info.Status = "Authenticated (Zen)"
				info.Authenticated = true
				info.Health = model.HealthHealthy
				return info
			}
			for provName, entry := range auth {
				if firstNonEmpty(entry.Key, entry.Token, entry.Access) != "" {
					info.Plan = fmt.Sprintf("OpenCode (%s)", provName)
					info.Email = provName
					info.Status = fmt.Sprintf("Configured (%s)", provName)
					info.Authenticated = true
					info.Health = model.HealthHealthy
					return info
				}
			}
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
