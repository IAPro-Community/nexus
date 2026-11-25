package nexus

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/security"
)

// BuildNexContextSummary constructs a bounded, redacted operational summary
// combining live HUD counts with opted-in @personal facts without leaking
// source code, filesystem paths, or credentials.
func BuildNexContextSummary(state HUDState, personalSummary string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 600
	}
	var parts []string
	parts = append(parts, fmt.Sprintf(
		"providers=%d agents=%d missions=%d alerts=%d runtime_attention=%d",
		len(state.Providers),
		len(state.Agents),
		len(state.Missions),
		len(state.Alerts),
		len(state.RuntimeAttention),
	))
	if state.Cost != nil {
		parts = append(parts, fmt.Sprintf("cost_budget=%s cost_evidence=%s", state.Cost.BudgetLevel, state.Cost.Evidence))
	}
	cleanPersonal := strings.TrimSpace(security.Redact(personalSummary))
	if cleanPersonal != "" {
		parts = append(parts, "personal="+cleanPersonal)
	}
	joined := strings.TrimSpace(security.Redact(strings.Join(parts, "; ")))
	if len(joined) > maxChars {
		return strings.TrimSpace(strings.ToValidUTF8(joined[:maxChars], ""))
	}
	return joined
}
