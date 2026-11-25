package runtime

import (
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

// ProfileIsolationEnvDefaults returns standardized environment defaults for
// launching a provider process inside an isolated profile home.
// Writable XDG directories and toolchain caches (Go build cache, Go tmp, npm)
// stay confined inside profileHome while NEXUS_REAL_HOME points to the host
// home so nested Nexus commands can still resolve real user config/data.
func ProfileIsolationEnvDefaults(profileHome string) map[string]string {
	defaults := make(map[string]string)
	home := strings.TrimSpace(profileHome)
	if home != "" {
		defaults["HOME"] = home
		defaults["USERPROFILE"] = home
		defaults["XDG_CONFIG_HOME"] = filepath.Join(home, ".config")
		defaults["XDG_CACHE_HOME"] = filepath.Join(home, ".cache")
		defaults["XDG_DATA_HOME"] = filepath.Join(home, ".local", "share")
		defaults["XDG_STATE_HOME"] = filepath.Join(home, ".local", "state")
		defaults["GOCACHE"] = filepath.Join(home, ".cache", "go-build")
		defaults["NPM_CONFIG_CACHE"] = filepath.Join(home, ".cache", "npm")
		if existingTmp := strings.TrimSpace(os.Getenv("GOTMPDIR")); existingTmp != "" && !isPathWithin(home, existingTmp) {
			goTmp := filepath.Join(home, ".cache", "go-tmp")
			_ = os.MkdirAll(goTmp, 0o700)
			defaults["GOTMPDIR"] = goTmp
		}
	}

	if realHome := strings.TrimSpace(config.BaseHome()); realHome != "" {
		defaults["NEXUS_REAL_HOME"] = realHome
	}
	return defaults
}

// ApplyProfileIsolationEnv merges ProfileIsolationEnvDefaults(profileHome)
// into overrides without overwriting explicit caller values.
func ApplyProfileIsolationEnv(profileHome string, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(overrides)+10)
	maps.Copy(merged, overrides)

	defaults := ProfileIsolationEnvDefaults(profileHome)
	for k, v := range defaults {
		if _, exists := merged[k]; exists {
			continue
		}
		merged[k] = v
	}
	return merged
}

func isPathWithin(base, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
