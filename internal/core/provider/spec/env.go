package spec

import (
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

func overlayEnv(base []string, profileHome string, overrides map[string]string, unset []string) []string {
	merged := make(map[string]string, len(overrides)+8)
	maps.Copy(merged, overrides)
	applySharedToolchainCacheOverrides(base, merged)
	return runtime.EnvSet(base, runtime.ApplyProfileIsolationEnv(profileHome, merged), unset...)
}

// applySharedToolchainCacheOverrides shares strictly non-credential compiler
// and package-manager caches across isolated profiles so multi-account usage
// never duplicates gigabytes of go-build/npm/yarn/playwright artifacts.
// Opt out via NEXUS_ISOLATE_BUILD_CACHES=1.
func applySharedToolchainCacheOverrides(base []string, overrides map[string]string) {
	if os.Getenv("NEXUS_ISOLATE_BUILD_CACHES") == "1" {
		return
	}
	baseHome := strings.TrimSpace(config.BaseHome())
	if baseHome == "" {
		return
	}
	sharedRoot := filepath.Join(baseHome, ".cache", "nexus-toolchains")
	mappings := [][2]string{
		{"GOCACHE", filepath.Join(sharedRoot, "go-build")},
		{"GOMODCACHE", filepath.Join(sharedRoot, "go-mod")},
		{"NPM_CONFIG_CACHE", filepath.Join(sharedRoot, "npm")},
		{"npm_config_cache", filepath.Join(sharedRoot, "npm")},
		{"YARN_CACHE_FOLDER", filepath.Join(sharedRoot, "yarn")},
		{"PLAYWRIGHT_BROWSERS_PATH", filepath.Join(sharedRoot, "ms-playwright")},
		{"CYPRESS_CACHE_FOLDER", filepath.Join(sharedRoot, "Cypress")},
	}
	for _, pair := range mappings {
		key, dir := pair[0], pair[1]
		if _, exists := overrides[key]; exists {
			continue
		}
		if existing, ok := envLookupNonEmpty(base, key); ok && config.StripProfileSandboxPath(existing) == existing {
			overrides[key] = existing
			continue
		}
		overrides[key] = dir
	}
}

func envLookupNonEmpty(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			val := strings.TrimSpace(strings.TrimPrefix(entry, prefix))
			if val != "" {
				return val, true
			}
		}
	}
	return "", false
}
