package events

import (
	"fmt"
	"sync/atomic"
	"time"
)

type EventType string

const (
	EventProcessStarted   EventType = "PROCESS_STARTED"
	EventProcessExited    EventType = "PROCESS_EXITED"
	EventSessionStarted   EventType = "SESSION_STARTED"
	EventSessionResumed   EventType = "SESSION_RESUMED"
	EventSessionEnded     EventType = "SESSION_ENDED"
	EventAgentWorking     EventType = "AGENT_WORKING"
	EventAgentWaiting     EventType = "AGENT_WAITING"
	EventToolStarted      EventType = "TOOL_STARTED"
	EventToolFinished     EventType = "TOOL_FINISHED"
	EventApprovalRequired EventType = "APPROVAL_REQUIRED"
	EventApproved         EventType = "APPROVED"
	EventRejected         EventType = "REJECTED"
	EventRateLimited      EventType = "RATE_LIMITED"
	EventHandoffCompleted EventType = "HANDOFF_COMPLETED"
	EventError            EventType = "ERROR"

	// Canonical Capacity Monitor Event Types
	EventQuotaObserved             EventType = "quota.observed"
	EventQuotaThresholdCrossed     EventType = "quota.threshold_crossed"
	EventQuotaRecovered            EventType = "quota.recovered"
	EventRecommendationChanged     EventType = "recommendation.changed"
	EventMonitorDegraded           EventType = "monitor.degraded"
	EventMonitorLeadershipChanged  EventType = "monitor.leadership_changed"
)

var eventCounter uint64

// Event represents a structured, normalized lifecycle or execution event.
type Event struct {
	ID        string         `json:"id"`
	RuntimeID string         `json:"runtime_id"`
	Provider  string         `json:"provider"`
	Profile   string         `json:"profile"`
	Type      EventType      `json:"type"`
	Summary   string         `json:"summary"`
	Data      map[string]any `json:"data,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// NewEvent creates a timestamped normalized event.
func NewEvent(runtimeID, provider, profile string, eventType EventType, summary string, data map[string]any) Event {
	id := atomic.AddUint64(&eventCounter, 1)
	return Event{
		ID:        fmt.Sprintf("evt-%d", id),
		RuntimeID: runtimeID,
		Provider:  provider,
		Profile:   profile,
		Type:      eventType,
		Summary:   summary,
		Data:      data,
		Timestamp: time.Now(),
	}
}
