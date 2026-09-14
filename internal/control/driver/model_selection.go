package driver

import (
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

const (
	// These are intentionally conservative defaults: the medium Flash model is
	// the preferred AGY Gemini route, while Sonnet is the first fallback pool.
	DefaultAGYGeminiModel = "gemini-3.8-flash-medium"
	DefaultAGYClaudeModel = "claude-sonnet-4-6"
)

// ResolveLaunchModel chooses AGY's model family without overriding an
// explicit model configured by the user. AGY exposes independent Gemini and
// Claude/GPT quota pools, so the first available pool must be selected before
// the CLI is started; otherwise AGY may silently choose the wrong family.
func ResolveLaunchModel(provider, configured string, usage model.UsageSnapshot) string {
	configured = strings.TrimSpace(configured)
	if configured != "" || !strings.EqualFold(strings.TrimSpace(provider), "agy") {
		return configured
	}

	geminiAvailable := agyQuotaGroupAvailable(usage, "gemini")
	claudeAvailable := agyQuotaGroupAvailable(usage, "claude_gpt")
	if geminiAvailable {
		return DefaultAGYGeminiModel
	}
	if claudeAvailable {
		return DefaultAGYClaudeModel
	}

	// Unknown quota must not make AGY fall back to its own implicit default.
	// Keep Gemini as the deterministic primary route. If both known pools are
	// exhausted, this also makes the provider return its quota error explicitly.
	return DefaultAGYGeminiModel
}

func agyQuotaGroupAvailable(usage model.UsageSnapshot, group string) bool {
	minRemaining := 100.0
	found := false
	for _, window := range usage.Windows {
		if !strings.EqualFold(strings.TrimSpace(window.Group), group) || window.RemainingPercent == nil {
			continue
		}
		found = true
		if *window.RemainingPercent < minRemaining {
			minRemaining = *window.RemainingPercent
		}
	}
	return found && minRemaining > 0
}
