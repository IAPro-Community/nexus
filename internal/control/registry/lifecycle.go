package registry

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CanTransition reports whether a runtime may move between operational
// states. Equal states are idempotent so repeated observations are safe.
// Administrative recovery still uses the explicit STOPPED/FAILED -> STARTING
// paths rather than jumping directly into a live state.
func CanTransition(from, to RuntimeState) bool {
	if from == to {
		return true
	}
	switch from {
	case StateStarting:
		return to == StateRunning || to == StateFailed
	case StateRunning:
		return to == StateWaiting || to == StateApproval || to == StateDetached ||
			to == StateHandoff || to == StateStopping || to == StateStopped || to == StateFailed
	case StateWaiting:
		return to == StateRunning || to == StateApproval || to == StateHandoff ||
			to == StateStopping || to == StateStopped || to == StateFailed
	case StateApproval:
		return to == StateRunning || to == StateWaiting || to == StateHandoff ||
			to == StateStopping || to == StateStopped || to == StateFailed
	case StateDetached:
		return to == StateRunning || to == StateHandoff || to == StateStopping ||
			to == StateStopped || to == StateFailed
	case StateHandoff:
		return to == StateRunning || to == StateStarting || to == StateStopping ||
			to == StateStopped || to == StateFailed
	case StateStopping:
		return to == StateStopped || to == StateFailed
	case StateStopped, StateFailed, StateStale:
		return to == StateStarting
	default:
		return false
	}
}

// HealthStatus classifies one runtime health signal. UNKNOWN means the
// registry cannot prove the signal from durable state; it never means the
// runtime is healthy and it never means the runtime recovered.
type HealthStatus string

const (
	HealthOK      HealthStatus = "OK"
	HealthUnknown HealthStatus = "UNKNOWN"
	HealthError   HealthStatus = "ERROR"
)

// Durable continuity-lease labels and the staleness bound used by the runtime
// health diagnosis. They are declared here so the diagnosis depends only on
// the registry row, not on the continuity claim implementation.
const (
	healthLeaseClaimIDLabel   = "nexus.continuity_claim_id"
	healthLeaseInFlightLabel  = "nexus.continuity_in_flight"
	healthLeaseClaimedAtLabel = "nexus.continuity_claimed_at"
	healthLeaseOwnerPIDLabel  = "nexus.continuity_owner_pid"
	healthLeaseStaleAfter     = 2 * time.Minute
)

// HealthSignal is one diagnosed runtime signal with its evidence detail.
type HealthSignal struct {
	Status HealthStatus `json:"status"`
	Detail string       `json:"detail"`
}

// RuntimeHealth diagnoses lease, PTY, heartbeat, PID and provider capability
// from durable registry state only. It never claims that a runtime recovered:
// failures are reported as ERROR and unverifiable signals as UNKNOWN.
type RuntimeHealth struct {
	RuntimeID          string       `json:"runtime_id"`
	Lease              HealthSignal `json:"lease"`
	PTY                HealthSignal `json:"pty"`
	Heartbeat          HealthSignal `json:"heartbeat"`
	PID                HealthSignal `json:"pid"`
	ProviderCapability HealthSignal `json:"provider_capability"`
}

// DiagnoseRuntimeHealth diagnoses one runtime session. The caller supplies now
// so the heartbeat age is deterministic in tests.
func DiagnoseRuntimeHealth(session RuntimeSession, now time.Time) RuntimeHealth {
	return RuntimeHealth{
		RuntimeID:          session.RuntimeID,
		Lease:              healthLease(session, now),
		PTY:                healthPTY(session),
		Heartbeat:          healthHeartbeat(session, now),
		PID:                healthPID(session),
		ProviderCapability: healthProviderCapability(session),
	}
}

func healthLease(session RuntimeSession, now time.Time) HealthSignal {
	labels := session.Labels
	if labels == nil || labels[healthLeaseClaimIDLabel] == "" {
		return HealthSignal{Status: HealthUnknown, Detail: "no continuity lease recorded; the terminal writer lease is not observable from the registry"}
	}
	ownerPID, err := strconv.Atoi(labels[healthLeaseOwnerPIDLabel])
	if err != nil || ownerPID <= 0 {
		ownerPID = 0
	}
	if labels[healthLeaseInFlightLabel] != "true" {
		return HealthSignal{Status: HealthOK, Detail: "continuity lease released"}
	}
	if healthLeaseIsStale(labels, now) {
		return HealthSignal{Status: HealthError, Detail: fmt.Sprintf("continuity lease heartbeat is stale; owner pid %d liveness is reported separately", ownerPID)}
	}
	return HealthSignal{Status: HealthOK, Detail: fmt.Sprintf("continuity lease held by pid %d", ownerPID)}
}

func healthLeaseIsStale(labels map[string]string, now time.Time) bool {
	claimedAt, err := time.Parse(time.RFC3339Nano, labels[healthLeaseClaimedAtLabel])
	if err != nil {
		return true
	}
	age := now.Sub(claimedAt)
	return age < -time.Minute || age > healthLeaseStaleAfter
}

func healthPTY(session RuntimeSession) HealthSignal {
	if session.ControlLevel != ControlLevelTerminal {
		return HealthSignal{Status: HealthUnknown, Detail: "runtime does not expose terminal control; PTY state is unknown"}
	}
	if session.LastFault == StartupFaultConPTYStartFailed {
		return HealthSignal{Status: HealthError, Detail: "terminal startup failed with CONPTY_START_FAILED"}
	}
	switch session.StartupStage {
	case StartupTerminalReady, StartupProviderStarting, StartupRunning:
		return HealthSignal{Status: HealthOK, Detail: "terminal reported ready"}
	}
	return HealthSignal{Status: HealthUnknown, Detail: fmt.Sprintf("terminal readiness is not recorded (startup stage %q)", session.StartupStage)}
}

func healthHeartbeat(session RuntimeSession, now time.Time) HealthSignal {
	labels := session.Labels
	if labels != nil && labels[healthLeaseClaimIDLabel] != "" {
		claimedAt, err := time.Parse(time.RFC3339Nano, labels[healthLeaseClaimedAtLabel])
		if err != nil {
			return HealthSignal{Status: HealthUnknown, Detail: "continuity lease heartbeat timestamp is unreadable"}
		}
		age := now.Sub(claimedAt)
		if age > healthLeaseStaleAfter {
			return HealthSignal{Status: HealthError, Detail: fmt.Sprintf("continuity lease heartbeat is stale for %s", age.Round(time.Second))}
		}
		return HealthSignal{Status: HealthOK, Detail: fmt.Sprintf("continuity lease heartbeat refreshed %s ago", age.Round(time.Second))}
	}
	if session.UpdatedAt.IsZero() {
		return HealthSignal{Status: HealthUnknown, Detail: "registry row has no heartbeat timestamp"}
	}
	age := now.Sub(session.UpdatedAt)
	if age > healthLeaseStaleAfter {
		return HealthSignal{Status: HealthError, Detail: fmt.Sprintf("registry row was not updated for %s", age.Round(time.Second))}
	}
	return HealthSignal{Status: HealthOK, Detail: fmt.Sprintf("registry row updated %s ago", age.Round(time.Second))}
}

func healthPID(session RuntimeSession) HealthSignal {
	if !session.IsActive() {
		return HealthSignal{Status: HealthUnknown, Detail: fmt.Sprintf("runtime is not active (state %s)", session.State)}
	}
	if session.PID <= 0 {
		return HealthSignal{Status: HealthError, Detail: "provider pid is not recorded"}
	}
	if !session.ProviderLive() {
		return HealthSignal{Status: HealthError, Detail: fmt.Sprintf("provider pid %d is not alive", session.PID)}
	}
	return HealthSignal{Status: HealthOK, Detail: fmt.Sprintf("provider pid %d is alive", session.PID)}
}

func healthProviderCapability(session RuntimeSession) HealthSignal {
	if strings.TrimSpace(session.ProviderID) == "" {
		return HealthSignal{Status: HealthError, Detail: "provider id is not recorded"}
	}
	switch session.LastFault {
	case StartupFaultProviderNotFound:
		return HealthSignal{Status: HealthError, Detail: "provider binary was not found at startup"}
	case StartupFaultProviderExitedEarly:
		return HealthSignal{Status: HealthError, Detail: "provider exited before startup completed"}
	}
	if session.ProviderBinaryPath == "" && session.ProviderBinaryFingerprint == "" {
		return HealthSignal{Status: HealthUnknown, Detail: "provider binary identity is not recorded"}
	}
	return HealthSignal{Status: HealthOK, Detail: "provider binary identity is recorded for " + session.ProviderID}
}
