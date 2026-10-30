package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
)

const CurrentConfigVersion = 1

type IntelligenceMode string

const (
	IntelligenceOff              IntelligenceMode = "OFF"
	IntelligenceCLI              IntelligenceMode = "CLI"
	IntelligenceOpenAICompatible IntelligenceMode = "OPENAI_COMPATIBLE"
)

// IntelligenceConfig stores non-secret routing metadata for Nexus Intelligence.
// API keys are resolved at runtime from APIKeyEnv or APIKeyFile; raw secrets are never persisted here.
type IntelligenceConfig struct {
	Mode       IntelligenceMode `json:"mode"`
	Provider   string           `json:"provider,omitempty"`
	Profile    string           `json:"profile,omitempty"`
	BaseURL    string           `json:"base_url,omitempty"`
	Model      string           `json:"model,omitempty"`
	APIKeyEnv  string           `json:"api_key_env,omitempty"`
	APIKeyFile string           `json:"api_key_file,omitempty"`
}

// IntelligenceMemory is a visible, deletable habit fact for this Nexus installation.
// It is never used to train a model — only to skip repeated Composer/Flow questions.
type IntelligenceMemory struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Source    string `json:"source,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Config represents persistent application settings and user preferences.
type Config struct {
	ConfigVersion        int                            `json:"config_version"`
	Defaults             map[string]string              `json:"defaults"`                      // provider -> profile
	Priorities           map[string]map[string]int      `json:"priorities,omitempty"`          // provider -> profile -> priority
	ProviderPriorities   map[string]int                 `json:"provider_priorities,omitempty"` // provider -> lower-is-first fallback priority
	Disabled             map[string]map[string]bool     `json:"disabled,omitempty"`            // provider -> profile -> true
	Labels               map[string]map[string][]string `json:"labels,omitempty"`              // provider -> profile -> tags
	Strategy             string                         `json:"strategy"`                      // best-capacity, least-used, round-robin, sticky
	StickyTTL            string                         `json:"sticky_ttl,omitempty"`          // e.g. "30m"
	IsolationPreset      model.IsolationPreset          `json:"isolation_preset"`              // developer, strict, compat
	Bindings             map[string]map[string]string   `json:"bindings,omitempty"`            // workspace -> provider -> profile
	AutomaticFallback    bool                           `json:"automatic_fallback"`
	MaxConcurrency       int                            `json:"max_concurrency"`
	Language             string                         `json:"language,omitempty"`
	Intelligence         IntelligenceConfig             `json:"intelligence,omitempty"`
	IntelligenceMemories []IntelligenceMemory           `json:"intelligence_memories,omitempty"`
	FlagAliases          map[string]map[string][]string `json:"flag_aliases,omitempty"` // alias -> provider -> flags
	installMemoryBaseline string `json:"-"`
}

var configMu sync.RWMutex

// NewDefaultConfig returns a well-configured default Configuration.
func NewDefaultConfig() Config {
	return Config{
		ConfigVersion:      CurrentConfigVersion,
		Defaults:           make(map[string]string),
		Priorities:         make(map[string]map[string]int),
		ProviderPriorities: map[string]int{"codex": 1, "agy": 2, "opencode": 3},
		Disabled:           make(map[string]map[string]bool),
		Labels:             make(map[string]map[string][]string),
		Strategy:           "best-capacity",
		StickyTTL:          "30m",
		IsolationPreset:    model.IsolationDeveloper,
		Bindings:           make(map[string]map[string]string),
		AutomaticFallback:  true,
		MaxConcurrency:     4,
		Language:           "auto",
		Intelligence:       IntelligenceConfig{Mode: IntelligenceOff},
	}
}

func getBaseHome() string {
	if v := os.Getenv("NEXUS_REAL_HOME"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if v := os.Getenv("AI_REAL_HOME"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if v := os.Getenv("USERPROFILE"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v
		}
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		norm := filepath.ToSlash(h)
		if idx := strings.Index(norm, "/.local/share/nexus/profiles"); idx != -1 {
			return filepath.FromSlash(norm[:idx])
		}
		if idx := strings.Index(norm, "/.local/share/ai-manager/profiles"); idx != -1 {
			return filepath.FromSlash(norm[:idx])
		}
		if idx := strings.Index(norm, "/.local/share/ai-cli/profiles"); idx != -1 {
			return filepath.FromSlash(norm[:idx])
		}
		if idx := strings.Index(norm, "/AppData/Local/nexus/profiles"); idx != -1 {
			return filepath.FromSlash(norm[:idx])
		}
		if idx := strings.Index(norm, "/AppData/Local/ai-manager/profiles"); idx != -1 {
			return filepath.FromSlash(norm[:idx])
		}
		if idx := strings.Index(norm, "/AppData/Local/ai-cli/profiles"); idx != -1 {
			return filepath.FromSlash(norm[:idx])
		}
		return h
	}
	if d, p := os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"); d != "" && p != "" {
		full := filepath.Join(d, p)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			return full
		}
	}
	if u := os.Getenv("USERNAME"); u != "" {
		p := filepath.Join(`C:\Users`, u)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if u := os.Getenv("USER"); u != "" {
		p := filepath.Join("/home", u)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

// BaseHome returns the real host user home directory, stripping any profile sandbox prefixes.
func BaseHome() string {
	return getBaseHome()
}

// ConfigDir returns the XDG/Windows-compliant config directory with ai-manager fallback.
func ConfigDir() (string, error) {
	if v := os.Getenv("NEXUS_CONFIG_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_MANAGER_CONFIG_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_CLI_CONFIG_DIR"); v != "" {
		return filepath.Abs(v)
	}

	home := getBaseHome()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	var candidates []string

	// Check XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" && !strings.Contains(filepath.ToSlash(xdg), "/profiles/") {
		candidates = append(candidates, filepath.Join(xdg, "nexus"), filepath.Join(xdg, "ai-cli"), filepath.Join(xdg, "ai-manager"))
	}

	// Check APPDATA on Windows
	if appData := os.Getenv("APPDATA"); appData != "" {
		candidates = append(candidates, filepath.Join(appData, "nexus"), filepath.Join(appData, "ai-cli"), filepath.Join(appData, "ai-manager"))
	}

	// Standard ~/.config
	configBase := filepath.Join(home, ".config")
	candidates = append(candidates,
		filepath.Join(configBase, "nexus"),
		filepath.Join(configBase, "ai-cli"),
		filepath.Join(configBase, "ai-manager"),
	)

	// Check if any candidate has config.json
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "config.json")); err == nil {
			return c, nil
		}
	}

	// Default to first candidate
	return candidates[0], nil
}

// DataDir returns the XDG/Windows-compliant data directory with ai-manager fallback.
func DataDir() (string, error) {
	if v := os.Getenv("NEXUS_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_MANAGER_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_CLI_DATA_DIR"); v != "" {
		return filepath.Abs(v)
	}

	home := getBaseHome()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	var candidates []string

	// Check XDG_DATA_HOME
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && !strings.Contains(filepath.ToSlash(xdg), "/profiles/") {
		candidates = append(candidates, filepath.Join(xdg, "nexus"), filepath.Join(xdg, "ai-cli"), filepath.Join(xdg, "ai-manager"))
	}

	// Check LOCALAPPDATA / APPDATA on Windows
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "nexus"), filepath.Join(localAppData, "ai-cli"), filepath.Join(localAppData, "ai-manager"))
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		candidates = append(candidates, filepath.Join(appData, "nexus"), filepath.Join(appData, "ai-cli"), filepath.Join(appData, "ai-manager"))
	}

	// Standard ~/.local/share
	dataBase := filepath.Join(home, ".local", "share")
	candidates = append(candidates,
		filepath.Join(dataBase, "nexus"),
		filepath.Join(dataBase, "ai-cli"),
		filepath.Join(dataBase, "ai-manager"),
	)

	// Check if any candidate has a profiles directory
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "profiles")); err == nil {
			return c, nil
		}
	}

	return candidates[0], nil
}

// StateDir returns the XDG/Windows-compliant state directory.
func StateDir() (string, error) {
	if v := os.Getenv("NEXUS_STATE_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_MANAGER_STATE_DIR"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("AI_CLI_STATE_DIR"); v != "" {
		return filepath.Abs(v)
	}

	home := getBaseHome()
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	var candidates []string

	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" && !strings.Contains(filepath.ToSlash(xdg), "/profiles/") {
		candidates = append(candidates, filepath.Join(xdg, "nexus"), filepath.Join(xdg, "ai-cli"), filepath.Join(xdg, "ai-manager"))
	}

	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "nexus", "state"), filepath.Join(localAppData, "ai-cli", "state"), filepath.Join(localAppData, "ai-manager", "state"))
	}

	stateBase := filepath.Join(home, ".local", "state")
	candidates = append(candidates,
		filepath.Join(stateBase, "nexus"),
		filepath.Join(stateBase, "ai-cli"),
		filepath.Join(stateBase, "ai-manager"),
	)

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return candidates[0], nil
}

// ProfileRoot returns the profile root directory.
func ProfileRoot(provider, name string) (string, error) {
	data, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "profiles", provider, name), nil
}

// ProfileHome returns the isolated home directory for a profile.
func ProfileHome(provider, name string) (string, error) {
	root, err := ProfileRoot(provider, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "home"), nil
}

// LoadConfig reads the configuration file with migration support.
func LoadConfig() (Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	cfg := NewDefaultConfig()
	dir, err := ConfigDir()
	if err != nil {
		return cfg, err
	}

	configPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.installMemoryBaseline = installMemoryBaseline(cfg.IntelligenceMemories)
			return cfg, nil
		}
		return cfg, err
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err == nil {
		if rawMap["defaults"] != nil {
			if defs, ok := rawMap["defaults"].(map[string]interface{}); ok {
				for k, v := range defs {
					if strVal, ok := v.(string); ok {
						cfg.Defaults[k] = strVal
					}
				}
			}
		}
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if cfg.Defaults == nil {
		cfg.Defaults = make(map[string]string)
	}
	if cfg.Priorities == nil {
		cfg.Priorities = make(map[string]map[string]int)
	}
	if cfg.ProviderPriorities == nil {
		cfg.ProviderPriorities = map[string]int{"codex": 1, "agy": 2, "opencode": 3}
	}
	if cfg.Disabled == nil {
		cfg.Disabled = make(map[string]map[string]bool)
	}
	if cfg.Labels == nil {
		cfg.Labels = make(map[string]map[string][]string)
	}
	if cfg.Bindings == nil {
		cfg.Bindings = make(map[string]map[string]string)
	}
	if cfg.IsolationPreset == "" {
		cfg.IsolationPreset = model.IsolationDeveloper
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "best-capacity"
	}
	if cfg.StickyTTL == "" {
		cfg.StickyTTL = "30m"
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 4
	}
	if cfg.Language == "" {
		cfg.Language = "auto"
	}
	if cfg.Intelligence.Mode == "" {
		cfg.Intelligence.Mode = IntelligenceOff
	}

	cfg.installMemoryBaseline = installMemoryBaseline(cfg.IntelligenceMemories)
	return cfg, nil
}

// SaveConfig atomically persists configuration to disk.
func SaveConfig(cfg Config) error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	return memoryrecovery.WithLegacyWriter(dataDir, func() error {
		configMu.Lock()
		defer configMu.Unlock()
		return saveConfigLocked(cfg)
	})
}

// saveConfigLocked is called with configMu held and with the cross-process
// memory recovery writer fence owned by the caller.
func saveConfigLocked(cfg Config) error {

	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	if err := checkInstallMemoryBaseline(configPath, cfg); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".nexus-config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, configPath)
}

// GetDefaultProfile returns the default profile for a provider.
func GetDefaultProfile(provider string) (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.Defaults[provider], nil
}

// SetDefaultProfile sets the default profile for a provider.
func SetDefaultProfile(provider, name string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	cfg.Defaults[provider] = name
	return SaveConfig(cfg)
}

// BindWorkspace associates a workspace with a provider profile.
func BindWorkspace(workspace, provider, profileName string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		absWs = workspace
	}
	if cfg.Bindings[absWs] == nil {
		cfg.Bindings[absWs] = make(map[string]string)
	}
	cfg.Bindings[absWs][provider] = profileName
	return SaveConfig(cfg)
}

// UnbindWorkspace removes workspace binding for a provider.
func UnbindWorkspace(workspace, provider string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		absWs = workspace
	}
	if cfg.Bindings[absWs] != nil {
		delete(cfg.Bindings[absWs], provider)
		if len(cfg.Bindings[absWs]) == 0 {
			delete(cfg.Bindings, absWs)
		}
		return SaveConfig(cfg)
	}
	return nil
}

// GetBinding returns the bound profile for a given workspace and provider.
func GetBinding(workspace, provider string) string {
	cfg, err := LoadConfig()
	if err != nil {
		return ""
	}
	absWs, err := filepath.Abs(workspace)
	if err != nil {
		absWs = workspace
	}
	if b, ok := cfg.Bindings[absWs]; ok {
		return b[provider]
	}
	return ""
}

// ValidateProfileName validates that a profile name uses allowed characters.
func ValidateProfileName(name string) error {
	if name == "" {
		return errors.New("profile name cannot be empty")
	}
	if len(name) > 64 {
		return errors.New("profile name too long (max 64 characters)")
	}
	if strings.HasPrefix(name, "-") {
		return errors.New("profile name cannot start with '-' (looks like a CLI flag)")
	}
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.' {
			continue
		}
		return fmt.Errorf("invalid character %q in profile name (allowed: a-z, A-Z, 0-9, -, _, .)", string(ch))
	}
	return nil
}

// Validate performs structural validation of configuration settings.
func (c Config) Validate() []string {
	var issues []string
	if c.Strategy != "best-capacity" && c.Strategy != "least-used" && c.Strategy != "round-robin" && c.Strategy != "sticky" {
		issues = append(issues, fmt.Sprintf("unknown strategy %q (allowed: best-capacity, least-used, round-robin, sticky)", c.Strategy))
	}
	if c.IsolationPreset != model.IsolationStrict && c.IsolationPreset != model.IsolationDeveloper && c.IsolationPreset != model.IsolationCompat {
		issues = append(issues, fmt.Sprintf("unknown isolation preset %q (allowed: developer, strict, compat)", c.IsolationPreset))
	}
	if c.MaxConcurrency < 1 || c.MaxConcurrency > 32 {
		issues = append(issues, "max_concurrency must be between 1 and 32")
	}
	if c.Language != "" && c.Language != "auto" && c.Language != "pt-BR" && c.Language != "en" && c.Language != "es" {
		issues = append(issues, fmt.Sprintf("unknown language %q (allowed: auto, pt-BR, en, es)", c.Language))
	}
	if c.Intelligence.Mode != IntelligenceOff && c.Intelligence.Mode != IntelligenceCLI && c.Intelligence.Mode != IntelligenceOpenAICompatible {
		issues = append(issues, fmt.Sprintf("unknown intelligence mode %q (allowed: OFF, CLI, OPENAI_COMPATIBLE)", c.Intelligence.Mode))
	}
	return issues
}
