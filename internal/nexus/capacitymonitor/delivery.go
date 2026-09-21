package capacitymonitor

import (
	"database/sql"
	"errors"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// DeliveryDecision explains whether an alert-worthy transition becomes a new
// persisted+persisted event or is suppressed as a duplicate.
type DeliveryDecision struct {
	Deliver bool   `json:"deliver"`
	Reason  string `json:"reason"`
}

// ShouldDeliver applies the re-alert rule: a new alert exists ONLY on
// state change, new window, different resource, recovery, or explicit repeat
// (reminder interval elapsed or explicitRepeat).
//
// The fingerprint already encodes provider/profile/group/window/severity/
// reset-cycle, so any of the first four produce a fingerprint different from
// the last persisted one and deliver. An identical fingerprint delivers only
// as a configured repeat; otherwise it suppresses (no WARNING x4).
func ShouldDeliver(state State, fingerprint string, last *store.CapacityEventRecord, now time.Time, reminderInterval time.Duration, explicitRepeat bool) DeliveryDecision {
	if last == nil {
		return DeliveryDecision{Deliver: true, Reason: "first sight"}
	}
	if explicitRepeat {
		return DeliveryDecision{Deliver: true, Reason: "explicit repeat"}
	}
	if last.Fingerprint != fingerprint {
		return DeliveryDecision{Deliver: true, Reason: "condition changed (state, window, resource, or recovery)"}
	}
	if reminderInterval > 0 && !last.EmittedAt.IsZero() && !now.Before(last.EmittedAt.Add(reminderInterval)) {
		return DeliveryDecision{Deliver: true, Reason: "reminder interval elapsed"}
	}
	_ = state
	return DeliveryDecision{Deliver: false, Reason: "duplicate same-condition alert suppressed"}
}

// lastEventFor returns the latest persisted event for a fingerprint, or nil
// when never emitted. Real store errors propagate (fail-closed is decided by
// the caller); sql.ErrNoRows maps to first sight.
func lastEventFor(st *store.Store, fingerprint string) (*store.CapacityEventRecord, error) {
	rec, err := st.FindCapacityEvent(fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
