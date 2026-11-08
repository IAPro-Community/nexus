package cursor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/authcopy"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// Adapter implements the classic direct-execution provider contract for
// Cursor CLI. The supervised control driver and this adapter intentionally use
// the same isolated profile HOME semantics.
type Adapter struct{}

func New() *Adapter                     { return &Adapter{} }
func (a *Adapter) ID() model.ProviderID { return model.ProviderCursor }
func (a *Adapter) Name() string         { return "Cursor CLI" }

func (a *Adapter) Capabilities() model.Capabilities {
	return model.Capabilities{
		Login: true, Logout: true, Usage: false, Conversations: false, Resume: true,
		CrossAccountResume: false, HotAccountSwitch: false, IsolatedRuntime: true, ProjectBinding: true,
	}
}

const cursorAuthCacheTTL = 3 * time.Minute

type cursorAuthCacheEntry struct {
	info      model.AccountInfo
	fetchedAt time.Time
}

var (
	cursorAuthCache   = map[string]cursorAuthCacheEntry{}
	cursorAuthCacheMu sync.Mutex
	cursorStatusRun   = runtime.RunCommandCapture
)

func cursorStatusAuthenticated(output string) bool {
	text := strings.ToLower(strings.TrimSpace(output))
	if text == "" {
		return false
	}

	for _, marker := range []string{
		"not authenticated",
		"unauthenticated",
		"not logged in",
		"logged out",
		"login required",
		"authentication required",
		"please login",
		"please log in",
	} {
		if strings.Contains(text, marker) {
			return false
		}
	}

	for _, marker := range []string{
		"login successful",
		"logged in as",
		"authentication successful",
		"authenticated",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func cursorStatusUnauthenticated(output string) bool {
	text := strings.ToLower(strings.TrimSpace(output))
	if text == "" {
		return false
	}
	for _, marker := range []string{
		"not authenticated",
		"unauthenticated",
		"not logged in",
		"logged out",
		"login required",
		"authentication required",
		"please login",
		"please log in",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func cursorBinary() (string, error) {
	return spec.MustGet("cursor").ResolveBinary()
}

func (a *Adapter) Detect(ctx context.Context) model.DetectionResult {
	bin, err := cursorBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: "agent/cursor-agent binary not found in PATH"}
	}
	out, runErr := runtime.RunCommandCapture(ctx, bin, []string{"--version"}, os.Environ(), "")
	version := strings.TrimSpace(out)
	if runErr != nil && version == "" {
		version = "unknown"
	}
	return model.DetectionResult{Installed: true, BinaryPath: bin, Version: version}
}

type cursorCLIConfig struct {
	AuthInfo struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		AuthID      string `json:"authId"`
	} `json:"authInfo"`
	ServerConfigCache struct {
		AuthCacheKey string `json:"authCacheKey"`
	} `json:"serverConfigCache"`
}

func validHostCursorAuth(data []byte) bool {
	var cfg cursorCLIConfig
	if json.Unmarshal(data, &cfg) != nil {
		return false
	}
	return strings.TrimSpace(cfg.AuthInfo.Email) != "" || strings.TrimSpace(cfg.ServerConfigCache.AuthCacheKey) != ""
}

// DetectHostAuth inspects the host machine for Cursor CLI credentials and installation.
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
	configPath := filepath.Join(hostHome, ".cursor", "cli-config.json")
	data, err := os.ReadFile(configPath)
	if err == nil && validHostCursorAuth(data) {
		var cfg cursorCLIConfig
		_ = json.Unmarshal(data, &cfg)
		det.Available = true
		det.AuthType = "session"
		det.Plan = "Cursor"
		det.AccountID = cfg.AuthInfo.Email
		det.Email = cfg.AuthInfo.Email
		return det, true
	}

	// Fallback to checking cursor status command directly on host if binary installed
	if det.Installed {
		bin, err := cursorBinary()
		if err == nil {
			statusCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			out, err := cursorStatusRun(statusCtx, bin, []string{"status"}, os.Environ(), "")
			if err == nil && cursorStatusAuthenticated(out) {
				det.Available = true
				det.AuthType = "session"
				det.Plan = "Cursor"
				return det, true
			}
		}
	}

	return det, false
}

// ImportHostAuth copies Cursor CLI configuration and auth into the isolated profile home.
func (a *Adapter) ImportHostAuth(ctx context.Context, p model.Profile) (bool, error) {
	hostHome := security.FindHostHome()
	if hostHome == "" {
		return false, nil
	}
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return false, err
	}
	if err := a.Prepare(ctx, p); err != nil {
		return false, err
	}
	source := filepath.Join(hostHome, ".cursor", "cli-config.json")
	destination := filepath.Join(home, ".cursor", "cli-config.json")
	copied, err := authcopy.CopyValidatedFile(source, destination, validHostCursorAuth)
	if err != nil || !copied {
		return copied, err
	}
	invalidateCursorAuthCache(p.Name)
	return a.InspectAuth(ctx, p).Authenticated, nil
}

func (a *Adapter) Prepare(ctx context.Context, p model.Profile) error {
	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0700); err != nil {
		return err
	}
	cfg, _ := config.LoadConfig()
	return security.ApplyIsolation(home, security.GetPolicy(cfg.IsolationPreset))
}

func (a *Adapter) Run(ctx context.Context, p model.Profile, args []string) (model.Failure, error) {
	if err := a.Prepare(ctx, p); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	bin, err := cursorBinary()
	if err != nil {
		return model.Failure{Kind: model.FailureProvider, Message: "cursor agent binary not found"}, err
	}
	home, _ := config.ProfileHome(string(a.ID()), p.Name)
	cwd, _ := os.Getwd()
	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME": home, "CURSOR_HOME": home, "CURSOR_CONFIG_DIR": filepath.Join(home, ".cursor"),
		"AI_PROFILE": p.Name, "AI_PROVIDER": string(a.ID()), "PATH": runtime.EnhancedPATH(filepath.Dir(bin)),
	})
	return runtime.RunInteractive(bin, args, env, cwd)
}

func (a *Adapter) Login(ctx context.Context, p model.Profile) error {
	invalidateCursorAuthCache(p.Name)
	_, err := a.Run(ctx, p, []string{"login"})
	return err
}

func (a *Adapter) Logout(ctx context.Context, p model.Profile) error {
	invalidateCursorAuthCache(p.Name)
	_, err := a.Run(ctx, p, []string{"logout"})
	return err
}

func invalidateCursorAuthCache(profile string) {
	cursorAuthCacheMu.Lock()
	defer cursorAuthCacheMu.Unlock()
	delete(cursorAuthCache, profile)
}

func (a *Adapter) InspectAuth(ctx context.Context, p model.Profile) model.AccountInfo {
	cursorAuthCacheMu.Lock()
	if entry, ok := cursorAuthCache[p.Name]; ok && time.Since(entry.fetchedAt) < cursorAuthCacheTTL {
		info := entry.info
		cursorAuthCacheMu.Unlock()
		return info
	}
	cursorAuthCacheMu.Unlock()

	info := model.AccountInfo{
		Plan: "Cursor", Status: "Not authenticated", Health: model.HealthAuthRequired,
		Usage: model.UsageSnapshot{ProviderID: string(a.ID()), ProfileID: p.Name, Status: model.UsageUnknown, Source: model.SourceNone},
	}
	if err := a.Prepare(ctx, p); err != nil {
		info.Status = err.Error()
		info.Health = model.HealthUnknown
		return info
	}
	bin, err := cursorBinary()
	if err != nil {
		return info
	}
	home, _ := config.ProfileHome(string(a.ID()), p.Name)
	env := runtime.EnvSet(os.Environ(), map[string]string{
		"HOME": home, "CURSOR_HOME": home, "CURSOR_CONFIG_DIR": filepath.Join(home, ".cursor"), "PATH": runtime.EnhancedPATH(filepath.Dir(bin)),
	})
	statusCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := cursorStatusRun(statusCtx, bin, []string{"status"}, env, "")
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		info.Status = "Authentication status could not be verified"
		info.Health = model.HealthUnknown
		return info
	}
	if err == nil && cursorStatusAuthenticated(out) {
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
		for _, line := range strings.Split(out, "\n") {
			lowerLine := strings.ToLower(line)
			if idx := strings.Index(lowerLine, "logged in as "); idx != -1 {
				candidate := strings.TrimSpace(line[idx+len("logged in as "):])
				if candidate != "" && strings.Contains(candidate, "@") {
					info.Email = candidate
					break
				}
			}
		}
	}
	if !info.Authenticated && !cursorStatusUnauthenticated(out) {
		info.Status = "Authentication status could not be verified"
		info.Health = model.HealthUnknown
		return info
	}

	cursorAuthCacheMu.Lock()
	cursorAuthCache[p.Name] = cursorAuthCacheEntry{info: info, fetchedAt: time.Now()}
	cursorAuthCacheMu.Unlock()
	return info
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnsupported,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
}

func (a *Adapter) RefreshUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return a.GetUsage(ctx, p)
}

func (a *Adapter) ListConversations(ctx context.Context, p model.Profile, workspace string) ([]model.Session, error) {
	// Cursor exposes `agent ls` as an interactive picker. Until Cursor exposes a
	// stable machine-readable list contract, Nexus does not scrape terminal UI.
	return nil, nil
}

func (a *Adapter) Resume(ctx context.Context, p model.Profile, sessionID string, args []string) (model.Failure, error) {
	resume := []string{"--resume", sessionID}
	resume = append(resume, args...)
	return a.Run(ctx, p, resume)
}

func (a *Adapter) ClassifyError(err error, output string) model.Failure {
	return classifier.Classify(err, output)
}
