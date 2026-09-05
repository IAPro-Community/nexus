package capacitymonitor

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// EvaluateTransition evaluates the current capacity observation against thresholds
// and returns the target state along with the transition reason.
func EvaluateTransition(
	previous State,
	remaining *float64,
	rateLimited bool,
	observedAt, expiresAt time.Time,
	now time.Time,
	thresholds Thresholds,
) (State, string) {
	if rateLimited {
		return StateExhausted, "rate limit active"
	}

	if remaining == nil || observedAt.IsZero() {
		return StateUnknown, "no valid observation"
	}

	if !expiresAt.IsZero() && now.After(expiresAt) {
		return StateUnknown, "observation expired"
	}

	rem := *remaining

	if rem <= thresholds.Exhausted {
		return StateExhausted, fmt.Sprintf("quota exhausted (%.0f%% remaining <= %.0f%%)", rem, thresholds.Exhausted)
	}

	if rem <= thresholds.Critical {
		return StateCritical, fmt.Sprintf("quota critical (%.0f%% remaining <= %.0f%%)", rem, thresholds.Critical)
	}

	if rem <= thresholds.Warning {
		return StateWarning, fmt.Sprintf("quota warning (%.0f%% remaining <= %.0f%%)", rem, thresholds.Warning)
	}

	// Above Warning threshold:
	if (previous == StateWarning || previous == StateCritical || previous == StateExhausted) && rem >= thresholds.Recovery {
		return StateRecovered, fmt.Sprintf("quota recovered (%.0f%% remaining >= %.0f%%)", rem, thresholds.Recovery)
	}

	return StateHealthy, fmt.Sprintf("quota healthy (%.0f%% remaining)", rem)
}

// DeriveFingerprint computes a stable fingerprint for an alert based on canonical properties.
// It explicitly excludes email, filesystem paths, prompt content, and tokens.
func DeriveFingerprint(provider, profile, group, window, severity, resetCycle string) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s", provider, profile, group, window, severity, resetCycle)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("alert-%x", sum[:12])
}
