package voice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
)

const currentSettingsVersion = 1

var storeMu sync.Mutex

type Store struct{}

func NewStore() *Store { return &Store{} }

func DefaultSettings() Settings {
	return Settings{
		Version:             currentSettingsVersion,
		Enabled:             false,
		Language:            "pt",
		TerminalEnabled:     true,
		NexusControlEnabled: true,
		AutomaticFallback:   true,
		MaxCaptureSeconds:   60,
		Providers: []ProviderConfig{
			{
				ID:           "groq-primary",
				Kind:         ProviderGroq,
				Label:        "Groq · Whisper Large V3 Turbo",
				Enabled:      true,
				Priority:     1,
				Model:        "whisper-large-v3-turbo",
				BaseURL:      "https://api.groq.com",
				APIKeyEnv:    "GROQ_API_KEY",
				FreeEligible: true,
			},
			{
				ID:           "cloudflare-fallback",
				Kind:         ProviderCloudflare,
				Label:        "Cloudflare Workers AI · Whisper Large V3 Turbo",
				Enabled:      true,
				Priority:     2,
				Model:        "@cf/openai/whisper-large-v3-turbo",
				BaseURL:      "https://api.cloudflare.com/client/v4",
				APIKeyEnv:    "CLOUDFLARE_API_TOKEN",
				AccountIDEnv: "CLOUDFLARE_ACCOUNT_ID",
				FreeEligible: true,
			},
		},
	}
}

func (s *Store) paths() (settingsPath, secretsPath, usagePath string, err error) {
	dir, err := coreconfig.ConfigDir()
	if err != nil {
		return "", "", "", err
	}
	return filepath.Join(dir, "voice-settings.json"),
		filepath.Join(dir, "voice-secrets.json"),
		filepath.Join(dir, "voice-usage.json"), nil
}

func (s *Store) LoadSettings() (Settings, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	return s.loadSettingsUnlocked()
}

func (s *Store) loadSettingsUnlocked() (Settings, error) {
	settingsPath, _, _, err := s.paths()
	if err != nil {
		return Settings{}, err
	}
	cfg := DefaultSettings()
	data, err := os.ReadFile(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Settings{}, fmt.Errorf("decode voice settings: %w", err)
	}
	return normalizeSettings(cfg)
}

func normalizeSettings(cfg Settings) (Settings, error) {
	if cfg.Version == 0 {
		cfg.Version = currentSettingsVersion
	}
	if cfg.Version != currentSettingsVersion {
		return Settings{}, fmt.Errorf("unsupported voice settings version %d", cfg.Version)
	}
	cfg.Language = normalizeLanguage(cfg.Language)
	if cfg.MaxCaptureSeconds < 5 {
		cfg.MaxCaptureSeconds = 5
	}
	if cfg.MaxCaptureSeconds > 120 {
		cfg.MaxCaptureSeconds = 120
	}
	if len(cfg.Providers) == 0 {
		cfg.Providers = DefaultSettings().Providers
	}
	seen := make(map[string]struct{}, len(cfg.Providers))
	for i := range cfg.Providers {
		p := &cfg.Providers[i]
		p.ID = strings.TrimSpace(p.ID)
		if p.ID == "" {
			return Settings{}, errors.New("voice provider id is required")
		}
		if _, exists := seen[p.ID]; exists {
			return Settings{}, fmt.Errorf("duplicate voice provider id %q", p.ID)
		}
		seen[p.ID] = struct{}{}
		if p.Priority <= 0 {
			p.Priority = i + 1
		}
		switch p.Kind {
		case ProviderGroq:
			if p.Label == "" {
				p.Label = "Groq · Whisper Large V3 Turbo"
			}
			if p.Model == "" {
				p.Model = "whisper-large-v3-turbo"
			}
			// Built-in providers are pinned to canonical endpoints. Settings are
			// remotely writable, so accepting arbitrary base URLs or key files here
			// would turn STT configuration into SSRF/local-file credential exfiltration.
			p.BaseURL = "https://api.groq.com"
			p.APIKeyEnv = "GROQ_API_KEY"
			p.APIKeyFile = ""
			p.AccountIDEnv = ""
		case ProviderCloudflare:
			if p.Label == "" {
				p.Label = "Cloudflare Workers AI · Whisper Large V3 Turbo"
			}
			if p.Model == "" {
				p.Model = "@cf/openai/whisper-large-v3-turbo"
			}
			p.BaseURL = "https://api.cloudflare.com/client/v4"
			p.APIKeyEnv = "CLOUDFLARE_API_TOKEN"
			p.APIKeyFile = ""
			p.AccountIDEnv = "CLOUDFLARE_ACCOUNT_ID"
		default:
			return Settings{}, fmt.Errorf("unsupported voice provider kind %q", p.Kind)
		}
	}
	sort.SliceStable(cfg.Providers, func(i, j int) bool {
		return cfg.Providers[i].Priority < cfg.Providers[j].Priority
	})
	for i := range cfg.Providers {
		cfg.Providers[i].Priority = i + 1
	}
	return cfg, nil
}

func normalizeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return ""
	case "pt", "pt-br", "pt_br":
		return "pt"
	case "en", "en-us", "en_us":
		return "en"
	case "es", "es-es", "es_es":
		return "es"
	default:
		v := strings.TrimSpace(value)
		if len(v) > 8 {
			return ""
		}
		return v
	}
}

func (s *Store) SaveUpdate(update SettingsUpdate) (Settings, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	cfg := Settings{
		Version:             currentSettingsVersion,
		Enabled:             update.Enabled,
		Language:            update.Language,
		TerminalEnabled:     update.TerminalEnabled,
		NexusControlEnabled: update.NexusControlEnabled,
		AutomaticFallback:   update.AutomaticFallback,
		MaxCaptureSeconds:   update.MaxCaptureSeconds,
		Providers:           make([]ProviderConfig, 0, len(update.Providers)),
	}
	for _, p := range update.Providers {
		cfg.Providers = append(cfg.Providers, p.ProviderConfig)
	}
	var err error
	cfg, err = normalizeSettings(cfg)
	if err != nil {
		return Settings{}, err
	}

	settingsPath, secretsPath, _, err := s.paths()
	if err != nil {
		return Settings{}, err
	}
	secrets, err := loadSecrets(secretsPath)
	if err != nil {
		return Settings{}, err
	}
	for _, p := range update.Providers {
		key := strings.TrimSpace(p.APIKey)
		switch {
		case p.ClearAPIKey:
			delete(secrets, p.ID)
		case key != "":
			secrets[p.ID] = key
		}
	}
	if err := writeJSONAtomic(settingsPath, cfg); err != nil {
		return Settings{}, err
	}
	if err := writeJSONAtomic(secretsPath, secrets); err != nil {
		return Settings{}, err
	}
	return cfg, nil
}

func (s *Store) ResolveAPIKey(p ProviderConfig) (string, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	_, secretsPath, _, err := s.paths()
	if err != nil {
		return "", err
	}
	secrets, err := loadSecrets(secretsPath)
	if err != nil {
		return "", err
	}
	if key := strings.TrimSpace(secrets[p.ID]); key != "" {
		return key, nil
	}
	if p.APIKeyEnv != "" {
		if key := strings.TrimSpace(os.Getenv(p.APIKeyEnv)); key != "" {
			return key, nil
		}
	}
	if p.APIKeyFile != "" {
		data, err := os.ReadFile(p.APIKeyFile)
		if err == nil {
			if key := strings.TrimSpace(string(data)); key != "" {
				return key, nil
			}
		}
	}
	return "", nil
}

func (s *Store) ResolveAccountID(p ProviderConfig) string {
	if value := strings.TrimSpace(p.AccountID); value != "" {
		return value
	}
	if p.AccountIDEnv != "" {
		return strings.TrimSpace(os.Getenv(p.AccountIDEnv))
	}
	return ""
}

func (s *Store) LoadUsage() (map[string]ProviderUsage, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	_, _, path, err := s.paths()
	if err != nil {
		return nil, err
	}
	out := map[string]ProviderUsage{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode voice usage: %w", err)
	}
	return out, nil
}

func (s *Store) RecordUsage(providerID, status string, durationMS int64, success, requestSent bool) {
	if providerID == "" {
		return
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	_, _, path, err := s.paths()
	if err != nil {
		return
	}
	usage := map[string]ProviderUsage{}
	if data, readErr := os.ReadFile(path); readErr == nil {
		_ = json.Unmarshal(data, &usage)
	}
	entry := usage[providerID]
	entry.ProviderID = providerID
	if requestSent {
		entry.Requests++
		if success {
			entry.Successes++
			if durationMS > 0 {
				entry.ObservedAudioMS += durationMS
			}
		} else {
			entry.Failures++
		}
	}
	entry.LastStatus = status
	entry.LastUsedAt = time.Now().UTC()
	usage[providerID] = entry
	_ = writeJSONAtomic(path, usage)
}

func loadSecrets(path string) (map[string]string, error) {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode voice secrets: %w", err)
	}
	return out, nil
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil && runtime.GOOS != "windows" {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
