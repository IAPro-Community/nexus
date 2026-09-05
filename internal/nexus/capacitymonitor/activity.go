package capacitymonitor

import (
	"context"
	"path/filepath"
	"strings"
)

var knownProviderBinaries = map[string]string{
	"codex":    "codex",
	"agy":      "agy",
	"claude":   "claude",
	"opencode": "opencode",
	"gemini":   "gemini",
	"cursor":   "cursor",
}

// NormalizeProviderActivity maps an executable name or resolved executable path
// to a normalized provider ID. It NEVER inspects arguments, environment, or session content.
func NormalizeProviderActivity(executable, resolvedPath string) (string, bool) {
	candidates := []string{executable, resolvedPath}
	for _, cand := range candidates {
		cand = strings.TrimSpace(cand)
		if cand == "" {
			continue
		}
		base := filepath.Base(cand)
		base = strings.TrimSuffix(base, ".exe")
		base = strings.ToLower(base)

		if provider, ok := knownProviderBinaries[base]; ok {
			return provider, true
		}
	}
	return "", false
}

// ActivityDetector inspects the local machine for running provider CLI processes
// without reading session content, arguments, or environment variables.
type ActivityDetector interface {
	Snapshot(ctx context.Context) (map[string]bool, error)
}
