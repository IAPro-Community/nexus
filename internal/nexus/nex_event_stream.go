package nexus

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type NexEventEnvelope struct {
	Revision    uint64    `json:"revision"`
	EventID     string    `json:"event_id"`
	Fingerprint string    `json:"fingerprint"`
	Kind        string    `json:"kind"`
	Severity    string    `json:"severity"`
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	Detail      string    `json:"detail,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Profile     string    `json:"profile,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type NexNotificationPolicy struct {
	DoNotDisturb     bool `json:"do_not_disturb"`
	QuietHoursActive bool `json:"quiet_hours_active"`
	MaxBatch         int  `json:"max_batch"`
}

type NexEventStreamPage struct {
	Revision        uint64             `json:"revision"`
	ReplayTruncated bool               `json:"replay_truncated"`
	Events          []NexEventEnvelope `json:"events"`
}

// NexEventStream maintains a bounded, deduplicated cursor-indexed event buffer
// so reconnects can resume safely without replaying duplicate alerts.
type NexEventStream struct {
	mu               sync.Mutex
	maxEvents        int
	revision         uint64
	events           []NexEventEnvelope
	seenFingerprints map[string]struct{}
}

func NewNexEventStream(maxEvents int) *NexEventStream {
	if maxEvents <= 0 {
		maxEvents = 256
	}
	return &NexEventStream{
		maxEvents:        maxEvents,
		events:           make([]NexEventEnvelope, 0),
		seenFingerprints: make(map[string]struct{}),
	}
}

// Record stores an event once per unique fingerprint and assigns a monotonic
// revision. Repeated events with an already-recorded fingerprint are ignored.
func (s *NexEventStream) Record(ev NexEventEnvelope) (NexEventEnvelope, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fp := strings.TrimSpace(ev.Fingerprint)
	if fp != "" {
		if _, exists := s.seenFingerprints[fp]; exists {
			return NexEventEnvelope{}, false
		}
		s.seenFingerprints[fp] = struct{}{}
	}
	s.revision++
	ev.Revision = s.revision
	if strings.TrimSpace(ev.EventID) == "" {
		ev.EventID = fmt.Sprintf("nex-ev-%d", ev.Revision)
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	} else {
		ev.CreatedAt = ev.CreatedAt.UTC()
	}
	s.events = append(s.events, ev)
	if len(s.events) > s.maxEvents {
		s.events = append([]NexEventEnvelope{}, s.events[len(s.events)-s.maxEvents:]...)
	}
	return ev, true
}

// Since returns all retained events strictly newer than cursor, marking
// ReplayTruncated when older events have rolled out of the ring buffer.
func (s *NexEventStream) Since(cursor uint64) NexEventStreamPage {
	s.mu.Lock()
	defer s.mu.Unlock()

	truncated := false
	if cursor > 0 && cursor < s.revision {
		if len(s.events) == 0 || s.events[0].Revision > cursor+1 {
			truncated = true
		}
	}

	out := make([]NexEventEnvelope, 0)
	for _, ev := range s.events {
		if ev.Revision > cursor {
			out = append(out, ev)
		}
	}
	return NexEventStreamPage{
		Revision:        s.revision,
		ReplayTruncated: truncated,
		Events:          out,
	}
}

// IngestHUDState translates canonical operational HUD facts and model catalog
// changes into deduplicated Nex event envelopes.
func (s *NexEventStream) IngestHUDState(state HUDState, modelChanges []ModelCatalogChange, now time.Time) int {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	recorded := 0
	for _, alert := range state.Alerts {
		if alert.Acked || (alert.SnoozedUntil != nil && alert.SnoozedUntil.After(now)) {
			continue
		}
		_, ok := s.Record(NexEventEnvelope{
			Fingerprint: fmt.Sprintf("alert:%s:%s", alert.Fingerprint, strings.ToUpper(alert.Severity)),
			Kind:        "CAPACITY_ALERT",
			Severity:    strings.ToUpper(alert.Severity),
			Source:      "capacity_monitor",
			Title:       firstNonEmptyStr(alert.Provider, alert.Category, "Capacity"),
			Detail:      alert.Reason,
			Provider:    alert.Provider,
			Profile:     alert.Profile,
			CreatedAt:   now,
		})
		if ok {
			recorded++
		}
	}
	for _, att := range state.RuntimeAttention {
		_, ok := s.Record(NexEventEnvelope{
			Fingerprint: fmt.Sprintf("runtime_attention:%s:%s", att.RuntimeID, att.Fingerprint),
			Kind:        "RUNTIME_ATTENTION",
			Severity:    "WARNING",
			Source:      "runtime_registry",
			Title:       firstNonEmptyStr(att.Provider, "Runtime"),
			Detail:      att.Reason,
			Provider:    att.Provider,
			CreatedAt:   now,
		})
		if ok {
			recorded++
		}
	}
	for _, mission := range state.Missions {
		if !mission.NeedsUser {
			continue
		}
		_, ok := s.Record(NexEventEnvelope{
			Fingerprint: fmt.Sprintf("mission:%s:%s", mission.RunID, mission.CurrentStage),
			Kind:        "MISSION_ATTENTION",
			Severity:    "WARNING",
			Source:      "mission_runner",
			Title:       firstNonEmptyStr(mission.CurrentTaskTitle, mission.RunID),
			Detail:      mission.InterventionReason,
			Provider:    mission.Provider,
			Profile:     mission.Profile,
			CreatedAt:   now,
		})
		if ok {
			recorded++
		}
	}
	for _, ch := range modelChanges {
		_, ok := s.Record(NexEventEnvelope{
			Fingerprint: fmt.Sprintf("model:%s", ch.Fingerprint),
			Kind:        "MODEL_CATALOG_" + ch.Kind,
			Severity:    "INFO",
			Source:      "model_catalog_watch",
			Title:       ch.Provider + " · " + ch.ModelID,
			Detail:      ch.Kind,
			Provider:    ch.Provider,
			Profile:     ch.Profile,
			CreatedAt:   now,
		})
		if ok {
			recorded++
		}
	}
	return recorded
}

// FilterForNotification enforces DND/quiet hours and bounded burst grouping
// while deduplicating by fingerprint.
func FilterForNotification(events []NexEventEnvelope, policy NexNotificationPolicy) []NexEventEnvelope {
	if policy.DoNotDisturb || policy.QuietHoursActive || len(events) == 0 {
		return []NexEventEnvelope{}
	}
	maxBatch := policy.MaxBatch
	if maxBatch <= 0 {
		maxBatch = 3
	}
	out := make([]NexEventEnvelope, 0, maxBatch)
	seen := make(map[string]struct{}, len(events))
	for _, ev := range events {
		if ev.Fingerprint != "" {
			if _, ok := seen[ev.Fingerprint]; ok {
				continue
			}
			seen[ev.Fingerprint] = struct{}{}
		}
		out = append(out, ev)
		if len(out) >= maxBatch {
			break
		}
	}
	return out
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
