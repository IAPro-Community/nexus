package quota

import (
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// CodexOperationalMaxAge is deliberately shorter than LastKnownTTL. Historical
// evidence can remain available for diagnostics for 30 days, but must not rank
// accounts or be presented as current capacity once it is older than 15 min.
const CodexOperationalMaxAge = 15 * time.Minute

// CodexEvidenceExpired prevents historical rollout/RPC snapshots from being
// mistaken for actionable provider quota. Rechecking usage is required before
// automatic routing; UNKNOWN is not proof that a profile is exhausted.
func CodexEvidenceExpired(snap model.UsageSnapshot, now time.Time) bool {
	if snap.ProviderID != "codex" || len(snap.Windows) == 0 {
		return false
	}
	if snap.FetchedAt.IsZero() {
		return true
	}
	age := now.Sub(snap.FetchedAt)
	return age < -time.Minute || age > CodexOperationalMaxAge
}
