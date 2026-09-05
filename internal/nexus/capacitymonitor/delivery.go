package capacitymonitor

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// Action defines an actionable user option attached to an alert.
type Action struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"` // "acknowledge" | "snooze" | "switch" | "copy_command"
	URL   string `json:"url,omitempty"`
}

// RecommendationSummary provides concise replacement guidance for notification payloads.
type RecommendationSummary struct {
	Provider         string  `json:"provider"`
	Profile          string  `json:"profile"`
	DisplayName      string  `json:"display_name"`
	Score            float64 `json:"score"`
	ContinuityImpact string  `json:"continuity_impact,omitempty"`
	Fingerprint      string  `json:"fingerprint"`
}

// CanonicalAlert represents the single backend authority event rendered across all notification channels.
type CanonicalAlert struct {
	ID                  string                 `json:"id"`
	Fingerprint         string                 `json:"fingerprint"`
	Provider            string                 `json:"provider"`
	Profile             string                 `json:"profile"`
	Group               string                 `json:"group"`
	Window              string                 `json:"window"`
	Severity            string                 `json:"severity"` // "WARNING" | "CRITICAL" | "EXHAUSTED" | "RECOVERED"
	RemainingPercent    *float64               `json:"remaining_percent,omitempty"`
	ResetAt             string                 `json:"reset_at,omitempty"`
	Freshness           string                 `json:"freshness,omitempty"`
	Confidence          string                 `json:"confidence,omitempty"`
	Subject             string                 `json:"subject"`
	ModelName           string                 `json:"model_name,omitempty"`
	Recommendation      *RecommendationSummary `json:"recommendation,omitempty"`
	LockScreenSafeTitle string                 `json:"lock_screen_safe_title"`
	LockScreenSafeBody  string                 `json:"lock_screen_safe_body"`
	Actions             []Action               `json:"actions,omitempty"`
}

// Channel abstracts delivery to in-app, browser, or native OS notification surfaces.
type Channel interface {
	Deliver(ctx context.Context, alert CanonicalAlert) error
}

// DeliveryRouter coordinates multi-channel delivery, quiet hours filtering, and receipt logging.
type DeliveryRouter struct {
	store    *store.Store
	channels map[string]Channel
	mu       sync.RWMutex
}

// NewDeliveryRouter constructs a DeliveryRouter with the specified channel adapters.
func NewDeliveryRouter(st *store.Store, channels map[string]Channel) *DeliveryRouter {
	if channels == nil {
		channels = make(map[string]Channel)
	}
	return &DeliveryRouter{
		store:    st,
		channels: channels,
	}
}

// Route delivers a canonical alert to all enabled channels respecting preferences and quiet hours.
func (r *DeliveryRouter) Route(
	ctx context.Context,
	alert CanonicalAlert,
	prefs store.CapacityPreferences,
	now time.Time,
) error {
	if !prefs.Enabled {
		return nil
	}

	// Quiet hours check
	if isQuietHours(prefs, now) {
		// Exhausted can bypass if explicitly permitted
		if alert.Severity != "EXHAUSTED" || !prefs.ExhaustedBypassQuietHours {
			return nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for channelName, ch := range r.channels {
		// Channel preference filtering
		if channelName == "browser" && !prefs.BrowserNotificationsEnabled {
			continue
		}
		if channelName == "native" && !prefs.NativeNotificationsEnabled {
			continue
		}

		err := ch.Deliver(ctx, alert)

		receiptStatus := "DELIVERED"
		errStr := ""
		if err != nil {
			receiptStatus = "FAILED"
			errStr = err.Error()
		}

		if r.store != nil {
			_ = r.store.PutDeliveryReceipt(store.DeliveryReceipt{
				AlertID:     alert.ID,
				Channel:     channelName,
				Fingerprint: alert.Fingerprint,
				Status:      receiptStatus,
				Error:       errStr,
				DeliveredAt: now,
			})
		}
	}

	return nil
}

func isQuietHours(prefs store.CapacityPreferences, now time.Time) bool {
	if !prefs.QuietHoursEnabled {
		return false
	}
	startMins := parseTimeMinutes(prefs.QuietHoursStartUTC)
	endMins := parseTimeMinutes(prefs.QuietHoursEndUTC)

	currentMins := now.UTC().Hour()*60 + now.UTC().Minute()

	if startMins <= endMins {
		return currentMins >= startMins && currentMins < endMins
	}
	// Crosses midnight (e.g. 22:00 -> 08:00)
	return currentMins >= startMins || currentMins < endMins
}

func parseTimeMinutes(s string) int {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

// MetricsSnapshot provides privacy-safe bounded telemetry counters and percentiles.
type MetricsSnapshot struct {
	ProbeSuccess              uint64        `json:"probe_success"`
	ProbeFailure              uint64        `json:"probe_failure"`
	StaleResources            uint64        `json:"stale_resources"`
	DeliveryFailure           uint64        `json:"delivery_failure"`
	DeduplicatedAlerts        uint64        `json:"deduplicated_alerts"`
	RecommendationAvailable   uint64        `json:"recommendation_available"`
	ProbeDurationP50          time.Duration `json:"probe_duration_p50"`
	ProbeDurationP95          time.Duration `json:"probe_duration_p95"`
	AverageCollectionInterval time.Duration `json:"average_collection_interval"`
}
