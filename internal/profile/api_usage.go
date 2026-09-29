package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	coreusage "github.com/IAPro-Community/nexus/internal/core/usage"
)

const apiUsageCacheTTL = 5 * time.Minute

// GetAPIResourceChannel returns only persisted API telemetry attributable to the
// exact current account. It never performs network I/O.
func GetAPIResourceChannel(providerName, profileName string, expectedScope model.AccountScope) (model.ResourceChannelState, bool) {
	var empty model.ResourceChannelState
	if !expectedScope.Verifiable() {
		return empty, false
	}
	root, err := config.ProfileRoot(providerName, profileName)
	if err != nil {
		return empty, false
	}
	data, err := os.ReadFile(filepath.Join(root, "api-usage.json"))
	if err != nil {
		return empty, false
	}
	var state model.ResourceChannelState
	if json.Unmarshal(data, &state) != nil || state.Channel != model.ResourceChannelAPI {
		return empty, false
	}
	if !state.AccountScope.Verifiable() || state.AccountScope.Key() != expectedScope.Key() {
		return empty, false
	}
	if state.ObservedAt.IsZero() {
		return empty, false
	}
	age := time.Since(state.ObservedAt)
	if age < -time.Minute || age > 30*24*time.Hour {
		return empty, false
	}
	if age <= apiUsageCacheTTL {
		if state.Status == model.UsageLive {
			state.Status = model.UsageCached
		}
		return state, true
	}
	if state.Status != model.UsageRateLimited {
		state.Status = model.UsageEstimated
		state.Confidence = model.UsageConfidenceLow
	}
	return state, true
}

// RefreshAPIResourceChannelContext refreshes an explicitly configured API usage
// binding. Missing bindings are not errors and do not affect subscription quota.
func RefreshAPIResourceChannelContext(ctx context.Context, providerName, profileName string) (model.ResourceChannelState, bool, error) {
	var empty model.ResourceChannelState
	cfg, err := config.LoadConfig()
	if err != nil {
		return empty, false, err
	}
	binding, ok := apiUsageBinding(cfg, providerName, profileName)
	if !ok {
		return empty, false, nil
	}

	identity := expectedProfileAccount(providerName, profileName)
	scope, scopeOK := usageScope(providerName, profileName, identity)
	if !scopeOK {
		return empty, false, fmt.Errorf("API usage binding requires a verifiable account scope for %s:%s", providerName, profileName)
	}
	if cached, found := GetAPIResourceChannel(providerName, profileName, scope); found &&
		!cached.ObservedAt.IsZero() && time.Since(cached.ObservedAt) <= apiUsageCacheTTL {
		return cached, true, nil
	}

	apiKeyEnv := strings.TrimSpace(binding.APIKeyEnv)
	if apiKeyEnv == "" {
		apiKeyEnv = "OPENAI_ADMIN_KEY"
	}
	adminKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if adminKey == "" {
		return empty, false, fmt.Errorf("%s is not configured", apiKeyEnv)
	}

	switch strings.ToLower(strings.TrimSpace(binding.Provider)) {
	case "openai":
		state, collectErr := coreusage.CollectOpenAIAdminUsage(ctx, adminKey, coreusage.OpenAIAdminFilter{
			ProjectIDs: binding.ProjectIDs,
			UserIDs:    binding.UserIDs,
			APIKeyIDs:  binding.APIKeyIDs,
		}, scope)
		if collectErr != nil {
			return empty, false, collectErr
		}
		if saveErr := saveAPIResourceChannel(providerName, profileName, state); saveErr != nil {
			return empty, false, saveErr
		}
		return state, true, nil
	default:
		return empty, false, fmt.Errorf("unsupported API usage provider %q", binding.Provider)
	}
}

func apiUsageBinding(cfg config.Config, providerName, profileName string) (config.APIUsageBinding, bool) {
	for provider, profiles := range cfg.APIUsageBindings {
		if !strings.EqualFold(strings.TrimSpace(provider), strings.TrimSpace(providerName)) {
			continue
		}
		for name, binding := range profiles {
			if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(profileName)) {
				return binding, true
			}
		}
	}
	return config.APIUsageBinding{}, false
}

func saveAPIResourceChannel(providerName, profileName string, state model.ResourceChannelState) error {
	if state.Channel != model.ResourceChannelAPI {
		return fmt.Errorf("refusing to persist non-API resource channel in api-usage cache")
	}
	if !state.AccountScope.Verifiable() {
		return fmt.Errorf("refusing to persist API usage without account scope")
	}
	root, err := config.ProfileRoot(providerName, profileName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(root, "api-usage.json")
	temp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(temp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
