package gemini

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

// Adapter implements the Provider interface for Google Gemini CLI.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID { return model.ProviderGemini }
func (a *Adapter) Name() string         { return "Gemini CLI" }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login:              true,
		Logout:             true,
		Usage:              true,
		Conversations:      false, // ListConversations is not implemented yet
		Resume:             true,
		CrossAccountResume: false,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := spec.MustGet("gemini").ResolveBinary()
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
		filepath.Join(home, ".gemini"),
		filepath.Join(home, ".config", "gemini"),
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
	bin, err := spec.MustGet("gemini").ResolveBinary()
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "gemini binary not found"}, err
	}
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home := filepath.Join(root, "home")
	cwd, _ := os.Getwd()

	envMap := map[string]string{
		"HOME":            home,
		"GEMINI_CLI_HOME": home,
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"PATH":            runtime.EnhancedPATH(filepath.Dir(bin)),
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		envMap["SSH_AUTH_SOCK"] = sock
	}

	env := runtime.EnvSet(os.Environ(), envMap)
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, []string{"--prompt", "hello"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.RemoveAll(filepath.Join(home, ".gemini"))
	return nil
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")

	info := model.AccountInfo{
		Plan:          "Google AI",
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

	credsFile := filepath.Join(home, ".gemini", "oauth_creds.json")
	if data, err := os.ReadFile(credsFile); err == nil {
		var cData struct {
			Email string `json:"email"`
		}
		if json.Unmarshal(data, &cData) == nil && cData.Email != "" {
			info.Email = cData.Email
			info.Authenticated = true
			info.Status = "Authenticated"
			info.Health = model.HealthHealthy
		}
	}

	return info
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	// Cache/last-known only — live HTTP belongs in RefreshUsage.
	if snap, ok := a.readCachedUsage(p); ok {
		return snap
	}
	return unknownUsageSnapshot(p)
}

// RefreshUsage fetches the official Gemini quota without reading usage.json.
func (a *Adapter) RefreshUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	snap, _ := a.fetchLiveUsage(ctx, p)
	return snap
}

func (a *Adapter) readCachedUsage(p model.Profile) (model.UsageSnapshot, bool) {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil || root == "" {
		return model.UsageSnapshot{}, false
	}
	data, err := os.ReadFile(filepath.Join(root, "usage.json"))
	if err != nil {
		return model.UsageSnapshot{}, false
	}
	var snap model.UsageSnapshot
	if json.Unmarshal(data, &snap) != nil || len(snap.Windows) == 0 {
		return model.UsageSnapshot{}, false
	}
	if snap.ProfileID != "" && snap.ProfileID != p.Name {
		return model.UsageSnapshot{}, false
	}
	snap.ProviderID = string(a.ID())
	snap.ProfileID = p.Name
	if snap.Status == model.UsageLive {
		snap.Status = model.UsageCached
	}
	return snap, true
}

func unknownUsageSnapshot(p model.Profile) model.UsageSnapshot {
	return model.UsageSnapshot{
		ProviderID: string(model.ProviderGemini),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	var sessions []model.Session
	return sessions, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	// Gemini CLI resume syntax: gemini -r <SESSION_ID>
	resumeArgs := []string{"-r", sessionID}
	resumeArgs = append(resumeArgs, args...)
	return a.Run(ctx, p, resumeArgs)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}
