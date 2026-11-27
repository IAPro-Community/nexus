package notify

import (
	"strings"
	"sync"
	"time"
)

// DefaultNexHubLeaseTTL bounds how long the backend suppresses redundant
// OS/Web Push fallback notifications without receiving a fresh heartbeat from NEX.
const DefaultNexHubLeaseTTL = 45 * time.Second

// NexHubStatus reports whether NEX is actively acting as the centralized
// notification hub and whether fallback channels should remain quiet or resume.
type NexHubStatus struct {
	Active       bool      `json:"active"`
	CanNotify    bool      `json:"can_notify"`
	DoNotDisturb bool      `json:"do_not_disturb"`
	Reason       string    `json:"reason"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	LeaseTTLMS   int64     `json:"lease_ttl_ms"`
}

type nexHubLeaseState struct {
	canNotify    bool
	doNotDisturb bool
	reason       string
	lastSeen     time.Time
	expiresAt    time.Time
}

var (
	nexHubMu    sync.RWMutex
	nexHubState nexHubLeaseState
	nexHubNow   = time.Now
)

// ReportNexHubHeartbeat records the latest NEX presence/notification lease state.
func ReportNexHubHeartbeat(canNotify, doNotDisturb bool, reason string, ttl time.Duration) NexHubStatus {
	now := nexHubNow().UTC()
	if ttl <= 0 {
		ttl = DefaultNexHubLeaseTTL
	}
	cleanReason := strings.TrimSpace(reason)

	nexHubMu.Lock()
	if canNotify {
		resolvedReason := "nex_active"
		if doNotDisturb {
			resolvedReason = "nex_dnd"
		}
		nexHubState = nexHubLeaseState{
			canNotify:    true,
			doNotDisturb: doNotDisturb,
			reason:       resolvedReason,
			lastSeen:     now,
			expiresAt:    now.Add(ttl),
		}
	} else {
		if cleanReason == "" {
			cleanReason = "nex_unavailable"
		}
		nexHubState = nexHubLeaseState{
			canNotify:    false,
			doNotDisturb: doNotDisturb,
			reason:       cleanReason,
			lastSeen:     now,
			expiresAt:    time.Time{},
		}
	}
	snapshot := statusFromStateLocked(now)
	nexHubMu.Unlock()
	return snapshot
}

// ClearNexHub releases the active NEX lease so fallback channels resume immediately.
func ClearNexHub(reason string) {
	cleanReason := strings.TrimSpace(reason)
	if cleanReason == "" {
		cleanReason = "nex_offline"
	}
	_ = ReportNexHubHeartbeat(false, false, cleanReason, 0)
}

// CurrentNexHubStatus returns the current effective NEX notification hub lease.
func CurrentNexHubStatus() NexHubStatus {
	now := nexHubNow().UTC()
	nexHubMu.RLock()
	snapshot := statusFromStateLocked(now)
	nexHubMu.RUnlock()
	return snapshot
}

// ShouldSuppressFallback reports whether native OS and Web Push fallback pop-ups
// should be suppressed because NEX is currently handling notifications.
func ShouldSuppressFallback(_ Category) bool {
	return CurrentNexHubStatus().Active
}

func statusFromStateLocked(now time.Time) NexHubStatus {
	active := nexHubState.canNotify && !nexHubState.expiresAt.IsZero() && now.Before(nexHubState.expiresAt)
	reason := nexHubState.reason
	if !active {
		if nexHubState.canNotify && !nexHubState.expiresAt.IsZero() && !now.Before(nexHubState.expiresAt) {
			reason = "lease_expired"
		} else if reason == "" {
			reason = "nex_offline"
		}
	}
	return NexHubStatus{
		Active:       active,
		CanNotify:    active,
		DoNotDisturb: nexHubState.doNotDisturb,
		Reason:       reason,
		LastSeenAt:   nexHubState.lastSeen,
		ExpiresAt:    nexHubState.expiresAt,
		LeaseTTLMS:   DefaultNexHubLeaseTTL.Milliseconds(),
	}
}
