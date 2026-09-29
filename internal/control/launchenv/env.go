package launchenv

import (
	"os"
	stdruntime "runtime"
	"strings"
)

// Isolation keys must stay under driver control. Agent Environment must never
// point Codex/Claude/etc. back at the host home (broken plugins, credential bleed).
var protectedIsolationKeys = map[string]struct{}{
	"HOME":             {},
	"CODEX_HOME":       {},
	"CODEX_CONFIG_DIR": {},
	"CURSOR_HOME":      {},
	"GEMINI_CLI_HOME":  {},
	"OPENCODE_HOME":    {},
	"XDG_DATA_HOME":    {},
	"XDG_CONFIG_HOME":  {},
	"XDG_STATE_HOME":   {},
	"XDG_CACHE_HOME":   {},
}

// Merge canonicalizes environment overrides and optionally prepends directories
// to the PATH already produced by a provider driver. This avoids duplicate keys
// whose precedence differs across operating systems/process launchers.
func Merge(base []string, overrides map[string]string, pathPrepend []string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	displayKeys := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))
	seen := map[string]bool{}
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		canonicalKey := envKeyForGOOS(parts[0], stdruntime.GOOS)
		if !seen[canonicalKey] {
			order = append(order, canonicalKey)
			displayKeys[canonicalKey] = parts[0]
			seen[canonicalKey] = true
		}
		values[canonicalKey] = parts[1]
	}
	for key, value := range overrides {
		if key == "" {
			continue
		}
		canonicalKey := envKeyForGOOS(key, stdruntime.GOOS)
		if _, protected := protectedIsolationKeys[canonicalKey]; protected {
			continue
		}
		if !seen[canonicalKey] {
			order = append(order, canonicalKey)
			displayKeys[canonicalKey] = key
			seen[canonicalKey] = true
		}
		values[canonicalKey] = value
	}
	if len(pathPrepend) > 0 {
		parts := make([]string, 0, len(pathPrepend)+1)
		for _, dir := range pathPrepend {
			if strings.TrimSpace(dir) != "" {
				parts = append(parts, dir)
			}
		}
		pathKey := envKeyForGOOS("PATH", stdruntime.GOOS)
		if existing := values[pathKey]; existing != "" {
			parts = append(parts, existing)
		}
		values[pathKey] = strings.Join(parts, string(os.PathListSeparator))
		if !seen[pathKey] {
			order = append(order, pathKey)
			displayKeys[pathKey] = "PATH"
			seen[pathKey] = true
		}
	}
	out := make([]string, 0, len(order))
	for _, canonicalKey := range order {
		key := displayKeys[canonicalKey]
		if key == "" {
			key = canonicalKey
		}
		out = append(out, key+"="+values[canonicalKey])
	}
	return out
}

func envKeyForGOOS(key, goos string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}
