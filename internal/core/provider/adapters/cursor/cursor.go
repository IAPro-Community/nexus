package cursor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/classifier"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
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

func cursorBinary() (string, error) {
	if bin, err := runtime.LookPath("agent"); err == nil {
		return bin, nil
	}
	return runtime.LookPath("cursor-agent")
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
	text := strings.ToLower(out)
	if err == nil && !strings.Contains(text, "not authenticated") && !strings.Contains(text, "login") {
		info.Authenticated = true
		info.Status = "Authenticated"
		info.Health = model.HealthHealthy
		// Cursor status output is provider-owned and not guaranteed structured;
		// avoid scraping an email into an unstable contract.
	}

	cursorAuthCacheMu.Lock()
	cursorAuthCache[p.Name] = cursorAuthCacheEntry{info: info, fetchedAt: time.Now()}
	cursorAuthCacheMu.Unlock()
	return info
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return model.UsageSnapshot{ProviderID: string(a.ID()), ProfileID: p.Name, Status: model.UsageUnknown, Source: model.SourceNone, FetchedAt: time.Now()}
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
