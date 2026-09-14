package intelligence

import (
	"context"
	"fmt"
	"os"
	"strings"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
)

// ProviderFromConfig creates provider types that do not require the Nexus runtime adapter.
// CLI mode is built by nexus.ConfiguredIntelligenceProvider because it must validate real driver capabilities.
func ProviderFromConfig(cfg coreconfig.IntelligenceConfig) (IntelligenceProvider, error) {
	switch cfg.Mode {
	case "", coreconfig.IntelligenceOff:
		return nil, nil
	case coreconfig.IntelligenceOpenAICompatible:
		key, err := resolveAPIKey(cfg)
		if err != nil {
			return nil, err
		}
		p := NewOpenAIProvider(cfg.BaseURL, key, cfg.Model)
		if !p.Available(context.TODO()) {
			return nil, ErrIntelligenceUnavailable
		}
		return p, nil
	case coreconfig.IntelligenceCLI:
		return nil, fmt.Errorf("CLI intelligence requires Nexus driver adapter")
	default:
		return nil, fmt.Errorf("unsupported intelligence mode %q", cfg.Mode)
	}
}

func resolveAPIKey(cfg coreconfig.IntelligenceConfig) (string, error) {
	if cfg.APIKeyFile != "" {
		b, err := os.ReadFile(cfg.APIKeyFile)
		if err != nil {
			return "", fmt.Errorf("read intelligence api key file: %w", err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	if cfg.APIKeyEnv != "" {
		return strings.TrimSpace(os.Getenv(cfg.APIKeyEnv)), nil
	}
	return "", nil
}
