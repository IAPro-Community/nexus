package zcode

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/authcopy"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// Adapter implements the Provider interface for ZCode CLI.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) ID() model.ProviderID { return model.ProviderZCode }
func (a *Adapter) Name() string         { return "ZCode" }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login:              true,
		Logout:             true,
		Usage:              false,
		Conversations:      false,
		Resume:             true,
		CrossAccountResume: false,
		HotAccountSwitch:   false,
		IsolatedRuntime:    true,
		ProjectBinding:     true,
	}
}

func zcodeBinary() (string, error) {
	return spec.MustGet("zcode").ResolveBinary()
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := zcodeBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "zcode binary not found in PATH"}
	}
	out, runErr := runtime.RunCommandCapture(ctx, bin, []string{"version"}, os.Environ(), "")
	version := strings.TrimSpace(out)
	if runErr != nil && version == "" {
		version = "unknown"
	}
	return model.DetectionResult{Installed: true, BinaryPath: bin, Version: version}
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	for _, dir := range []string{
		home,
		filepath.Join(home, ".zcode"),
		filepath.Join(home, ".zcode", "cli"),
		filepath.Join(home, ".zcode", "v2"),
		filepath.Join(home, ".config"),
	} {
		if err := os.MkdirAll(dir, 0700); err != nil {
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
	bin, err := zcodeBinary()
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "zcode binary not found"}, err
	}
	root, _ := config.ProfileRoot(string(a.ID()), p.Name)
	home := filepath.Join(root, "home")
	cwd, _ := os.Getwd()

	envMap := map[string]string{
		"HOME":            home,
		"USERPROFILE":     home,
		"ZCODE_HOME":      home,
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"AI_PROFILE":      p.Name,
		"AI_PROVIDER":     string(a.ID()),
		"PATH":            runtime.EnhancedPATH(filepath.Dir(bin)),
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		envMap["SSH_AUTH_SOCK"] = sock
	}

	env := runtime.EnvSet(os.Environ(), envMap)
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	_, err := a.Run(ctx, p, []string{"login", "zai"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	home := filepath.Join(root, "home")
	_ = os.RemoveAll(filepath.Join(home, ".zcode"))
	return nil
}

func validZCodeCredentials(data []byte) bool {
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return false
	}
	for _, key := range []string{"oauth:active_provider", "oauth:zai:access_token", "zcodejwttoken"} {
		if val, ok := m[key]; ok && val != nil && val != "" {
			return true
		}
	}
	return false
}

// DetectHostAuth inspects the host machine for active ZCode credentials.
func (a *Adapter) DetectHostAuth(ctx context.Context) (model.HostAuthDetection, bool) {
	det := model.HostAuthDetection{
		Provider: string(a.ID()),
	}
	detectResult := a.Detect(ctx)
	det.Installed = detectResult.Installed

	hostHome := security.FindHostHome()
	if hostHome == "" {
		return det, false
	}

	credsPath := filepath.Join(hostHome, ".zcode", "v2", "credentials.json")
	data, err := os.ReadFile(credsPath)
	if err != nil || len(data) == 0 || !validZCodeCredentials(data) {
		return det, false
	}

	det.Available = true
	det.AuthType = "oauth"
	det.Plan = "Z.AI Individual Plan"

	// Try reading provider_config.json for model info
	cfgPath := filepath.Join(hostHome, ".zcode", "v2", "provider_config.json")
	if cfgData, err := os.ReadFile(cfgPath); err == nil {
		var cfg struct {
			Config struct {
				DefaultModelSelection struct {
					ModelID string `json:"modelId"`
				} `json:"defaultModelSelection"`
			} `json:"config"`
		}
		if json.Unmarshal(cfgData, &cfg) == nil && cfg.Config.DefaultModelSelection.ModelID != "" {
			det.Plan = "Z.AI (" + cfg.Config.DefaultModelSelection.ModelID + ")"
		}
	}

	return det, true
}

// ImportHostAuth copies ZCode CLI host credentials and config into the isolated profile home.
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
	home := filepath.Join(root, "home")

	// 1. Copy credentials.json
	srcCreds := filepath.Join(hostHome, ".zcode", "v2", "credentials.json")
	dstCreds := filepath.Join(home, ".zcode", "v2", "credentials.json")
	copied, err := authcopy.CopyValidatedFile(srcCreds, dstCreds, validZCodeCredentials)
	if err != nil || !copied {
		return copied, err
	}

	// 2. Copy provider_config.json if present
	srcProv := filepath.Join(hostHome, ".zcode", "v2", "provider_config.json")
	dstProv := filepath.Join(home, ".zcode", "v2", "provider_config.json")
	if data, err := os.ReadFile(srcProv); err == nil {
		_ = os.WriteFile(dstProv, data, 0600)
	}

	// 3. Copy setting.json if present
	srcSet := filepath.Join(hostHome, ".zcode", "cli", "setting.json")
	dstSet := filepath.Join(home, ".zcode", "cli", "setting.json")
	if data, err := os.ReadFile(srcSet); err == nil {
		_ = os.WriteFile(dstSet, data, 0600)
	}

	if !a.InspectAuth(ctx, p).Authenticated {
		return false, errors.New("host ZCode session could not be verified in the Nexus profile")
	}
	return true, nil
}

// InspectAuth reports whether the profile has a valid ZCode session.
func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	root, err := config.ProfileRoot(string(a.ID()), p.Name)
	if err != nil {
		return model.AccountInfo{Status: "Error", Health: model.HealthUnknown, Authenticated: false}
	}
	home := filepath.Join(root, "home")
	credsPath := filepath.Join(home, ".zcode", "v2", "credentials.json")
	data, err := os.ReadFile(credsPath)
	if err != nil || !validZCodeCredentials(data) {
		return model.AccountInfo{
			Status:        "Not authenticated",
			Health:        model.HealthAuthRequired,
			Authenticated: false,
		}
	}

	plan := "Z.AI Individual Plan"
	cfgPath := filepath.Join(home, ".zcode", "v2", "provider_config.json")
	if cfgData, err := os.ReadFile(cfgPath); err == nil {
		var cfg struct {
			Config struct {
				DefaultModelSelection struct {
					ModelID string `json:"modelId"`
				} `json:"defaultModelSelection"`
			} `json:"config"`
		}
		if json.Unmarshal(cfgData, &cfg) == nil && cfg.Config.DefaultModelSelection.ModelID != "" {
			plan = "Z.AI (" + cfg.Config.DefaultModelSelection.ModelID + ")"
		}
	}

	return model.AccountInfo{
		Plan:          plan,
		Status:        "Authenticated",
		Health:        model.HealthHealthy,
		Authenticated: true,
	}
}
