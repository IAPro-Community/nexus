package profile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// Re-export common types for compatibility
type Profile = model.Profile
type Config = config.Config

func DataDir() (string, error) {
	return config.DataDir()
}

func ConfigDir() (string, error) {
	return config.ConfigDir()
}

func Root(provider, name string) (string, error) {
	return config.ProfileRoot(provider, name)
}

func Home(provider, name string) (string, error) {
	return config.ProfileHome(provider, name)
}

func ValidateName(s string) error {
	return config.ValidateProfileName(s)
}

func ValidateProvider(p string) error {
	switch strings.ToLower(p) {
	case "codex", "agy", "claude", "opencode", "gemini", "cursor":
		return nil
	case "omp":
		if os.Getenv("NEXUS_EXPERIMENTAL_HARNESSES") == "1" {
			return nil
		}
		return fmt.Errorf("OMP requires NEXUS_EXPERIMENTAL_HARNESSES=1 while issue #98 remains experimental")
	default:
		return fmt.Errorf("unsupported provider %q (supported: codex, agy, claude, opencode, gemini, cursor; experimental: omp)", p)
	}
}

func Exists(provider, name string) bool {
	root, err := config.ProfileRoot(provider, name)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(root, "profile.json"))
	return err == nil
}

func Create(provider, name string) (model.Profile, error) {
	var p model.Profile
	if err := ValidateProvider(provider); err != nil {
		return p, err
	}
	if err := ValidateName(name); err != nil {
		return p, err
	}
	if Exists(provider, name) {
		return p, fmt.Errorf("profile %s:%s already exists", provider, name)
	}
	root, err := config.ProfileRoot(provider, name)
	if err != nil {
		return p, err
	}
	home := filepath.Join(root, "home")
	for _, d := range []string{root, home} {
		if err := os.MkdirAll(d, 0700); err != nil {
			return p, err
		}
	}
	p = model.Profile{Provider: provider, Name: name, CreatedAt: time.Now()}
	b, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "profile.json"), append(b, '\n'), 0600); err != nil {
		return p, err
	}
	if scope, scopeErr := AccountScope(provider, name, ""); scopeErr == nil {
		p.AccountScope = scope
		b, _ = json.MarshalIndent(p, "", "  ")
		if err := os.WriteFile(filepath.Join(root, "profile.json"), append(b, '\n'), 0600); err != nil {
			return p, err
		}
	}
	// A profile born without a credential adopts the host CLI login once, so
	// the first run does not end in an unexplained "awaiting authentication"
	// state. Quota probes still validate the account identity per profile.
	_, _ = codex.AdoptHostAuth(context.Background(), p)
	return p, nil
}

func Delete(provider, name string) error {
	root, err := config.ProfileRoot(provider, name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Defaults[provider] == name {
		delete(cfg.Defaults, provider)
		ps, _ := List()
		for _, p := range ps {
			if p.Provider == provider {
				cfg.Defaults[provider] = p.Name
				break
			}
		}
		_ = config.SaveConfig(cfg)
	}
	return nil
}

func Rename(provider, oldName, newName string) error {
	if err := ValidateName(newName); err != nil {
		return err
	}
	if !Exists(provider, oldName) {
		return fmt.Errorf("profile %s:%s does not exist", provider, oldName)
	}
	if Exists(provider, newName) {
		return fmt.Errorf("profile %s:%s already exists", provider, newName)
	}

	oldRoot, err := config.ProfileRoot(provider, oldName)
	if err != nil {
		return err
	}
	newRoot, err := config.ProfileRoot(provider, newName)
	if err != nil {
		return err
	}

	if err := os.Rename(oldRoot, newRoot); err != nil {
		return err
	}

	p, err := Get(provider, newName)
	if err == nil {
		p.Name = newName
		b, _ := json.MarshalIndent(p, "", "  ")
		_ = os.WriteFile(filepath.Join(newRoot, "profile.json"), append(b, '\n'), 0600)
	}

	// Update defaults and bindings
	cfg, err := config.LoadConfig()
	if err == nil {
		if cfg.Defaults[provider] == oldName {
			cfg.Defaults[provider] = newName
		}
		for ws, b := range cfg.Bindings {
			if b[provider] == oldName {
				cfg.Bindings[ws][provider] = newName
			}
		}
		_ = config.SaveConfig(cfg)
	}

	return nil
}

// recoverLegacyProfile rebuilds profile.json for a profile root that was left
// without metadata, for example when the process stopped between creating the
// isolated home and writing the profile. The directory only exists because a
// profile was requested, so restoring its metadata is safer than hiding an
// account from `profiles`, `usage` and every other listing.
func recoverLegacyProfile(provider, name string) error {
	root, err := config.ProfileRoot(provider, name)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("profile %s:%s has no profile root", provider, name)
	}
	// Only recover directories that already look like a profile root. An
	// arbitrary directory must never become a profile by accident.
	looksLikeProfile := false
	for _, marker := range []string{"account-scope.json", "home"} {
		if _, statErr := os.Stat(filepath.Join(root, marker)); statErr == nil {
			looksLikeProfile = true
			break
		}
	}
	if !looksLikeProfile {
		return fmt.Errorf("profile %s:%s is not a profile directory", provider, name)
	}
	p := model.Profile{Provider: provider, Name: name, CreatedAt: info.ModTime()}
	if scope, scopeErr := AccountScope(provider, name, ""); scopeErr == nil {
		p.AccountScope = scope
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), append(b, '\n'), 0600); err != nil {
		return err
	}
	// A recovered root is a profile that never finished being created, so it has
	// no credential of its own. Adopt the host CLI login exactly like Create
	// does, recording provenance, instead of surfacing an account that can never
	// report quota.
	_, _ = codex.AdoptHostAuth(context.Background(), p)
	return nil
}

func List() ([]model.Profile, error) {
	data, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(data, "profiles")
	var out []model.Profile
	providers := []string{"agy", "codex", "claude", "cursor", "opencode", "gemini"}

	cfg, _ := config.LoadConfig()

	for _, provider := range providers {
		entries, err := os.ReadDir(filepath.Join(base, provider))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(base, provider, e.Name(), "profile.json"))
			if err != nil {
				if repairErr := recoverLegacyProfile(provider, e.Name()); repairErr != nil {
					continue
				}
				raw, err = os.ReadFile(filepath.Join(base, provider, e.Name(), "profile.json"))
				if err != nil {
					continue
				}
			}
			var p model.Profile
			if json.Unmarshal(raw, &p) == nil {
				// Backfill the opaque scope for legacy profile.json files. This is
				// metadata-only and does not attribute any old quota snapshot.
				if scope, scopeErr := AccountScope(p.Provider, p.Name, ""); scopeErr == nil {
					p.AccountScope = scope
				}
				if cfg.Disabled[p.Provider] != nil && cfg.Disabled[p.Provider][p.Name] {
					p.Disabled = true
				}
				if cfg.Priorities[p.Provider] != nil {
					p.Priority = cfg.Priorities[p.Provider][p.Name]
				}
				if cfg.Labels[p.Provider] != nil {
					p.Labels = cfg.Labels[p.Provider][p.Name]
				}
				out = append(out, p)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider == out[j].Provider {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		}
		return out[i].Provider < out[j].Provider
	})
	return out, nil
}

func Get(provider, name string) (model.Profile, error) {
	var p model.Profile
	if err := ValidateProvider(provider); err != nil {
		return p, err
	}
	root, err := config.ProfileRoot(provider, name)
	if err != nil {
		return p, err
	}
	raw, err := os.ReadFile(filepath.Join(root, "profile.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return p, fmt.Errorf("profile %s:%s does not exist", provider, name)
		}
		return p, err
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	if scope, scopeErr := AccountScope(p.Provider, p.Name, ""); scopeErr == nil {
		p.AccountScope = scope
	}
	cfg, _ := config.LoadConfig()
	if cfg.Disabled[p.Provider] != nil && cfg.Disabled[p.Provider][p.Name] {
		p.Disabled = true
	}
	if cfg.Priorities[p.Provider] != nil {
		p.Priority = cfg.Priorities[p.Provider][p.Name]
	}
	return p, nil
}

func SetDisabled(provider, name string, disabled bool) error {
	if !Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist", provider, name)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.Disabled[provider] == nil {
		cfg.Disabled[provider] = make(map[string]bool)
	}
	cfg.Disabled[provider][name] = disabled
	return config.SaveConfig(cfg)
}

func SetPriority(provider, name string, priority int) error {
	if !Exists(provider, name) {
		return fmt.Errorf("profile %s:%s does not exist", provider, name)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.Priorities[provider] == nil {
		cfg.Priorities[provider] = make(map[string]int)
	}
	cfg.Priorities[provider][name] = priority
	return config.SaveConfig(cfg)
}

func LoadConfig() (config.Config, error) {
	return config.LoadConfig()
}

func SaveConfig(cfg config.Config) error {
	return config.SaveConfig(cfg)
}

func SetDefault(provider, name string) error {
	return config.SetDefaultProfile(provider, name)
}

func Default(provider string) (string, error) {
	return config.GetDefaultProfile(provider)
}

type ProfileInfo struct {
	Profile       model.Profile     `json:"profile"`
	IsDefault     bool              `json:"is_default"`
	RootPath      string            `json:"root_path"`
	HomePath      string            `json:"home_path"`
	ConfigDir     string            `json:"config_dir"`
	DataDir       string            `json:"data_dir"`
	BinaryPath    string            `json:"binary_path"`
	CWD           string            `json:"cwd"`
	UID           int               `json:"uid"`
	GID           int               `json:"gid"`
	IsolationVars map[string]string `json:"isolation_vars"`
	Details       map[string]string `json:"details"`
}

func Inspect(provider, name string) (*ProfileInfo, error) {
	p, err := Get(provider, name)
	if err != nil {
		return nil, err
	}
	root, _ := config.ProfileRoot(provider, name)
	home, _ := config.ProfileHome(provider, name)
	d, _ := config.GetDefaultProfile(provider)
	cfgDir, _ := config.ConfigDir()
	dataDir, _ := config.DataDir()
	cwd, _ := os.Getwd()
	uid := os.Getuid()
	gid := os.Getgid()

	binPath := "(not found in PATH)"
	if s, ok := spec.Get(provider); ok {
		if resolved, err := s.ResolveBinary(); err == nil && resolved != "" {
			binPath = resolved
		}
	} else if looked, err := runtime.LookPath(provider); err == nil && looked != "" {
		binPath = looked
	}

	info := &ProfileInfo{
		Profile:       p,
		IsDefault:     d == name,
		RootPath:      root,
		HomePath:      home,
		ConfigDir:     cfgDir,
		DataDir:       dataDir,
		BinaryPath:    binPath,
		CWD:           cwd,
		UID:           uid,
		GID:           gid,
		IsolationVars: make(map[string]string),
		Details:       make(map[string]string),
	}

	switch provider {
	case "codex":
		info.IsolationVars["CODEX_HOME"] = home
	case "agy":
		info.IsolationVars["HOME"] = home
		info.IsolationVars["XDG_CONFIG_HOME"] = filepath.Join(home, ".config")
		info.IsolationVars["XDG_CACHE_HOME"] = filepath.Join(home, ".cache")
		info.IsolationVars["XDG_DATA_HOME"] = filepath.Join(home, ".local", "share")
	case "claude":
		info.IsolationVars["HOME"] = home
		info.IsolationVars["CLAUDE_CONFIG_DIR"] = filepath.Join(home, ".claude")
	case "opencode":
		info.IsolationVars["HOME"] = home
		info.IsolationVars["OPENCODE_CONFIG_DIR"] = filepath.Join(home, ".config", "opencode")
	case "gemini":
		info.IsolationVars["HOME"] = home
		info.IsolationVars["GEMINI_CLI_HOME"] = home
	}

	return info, nil
}

func EnsureRandomSecret(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(buf)+"\n"), 0600)
}
