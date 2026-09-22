package events

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

type EventType string

const (
	EventProcessStarted                 EventType = "PROCESS_STARTED"
	EventProcessExited                  EventType = "PROCESS_EXITED"
	EventRuntimeStarted                 EventType = "RUNTIME_STARTED"
	EventRuntimeStopped                 EventType = "RUNTIME_STOPPED"
	EventRuntimeFailed                  EventType = "RUNTIME_FAILED"
	EventSessionStarted                 EventType = "SESSION_STARTED"
	EventSessionResumed                 EventType = "SESSION_RESUMED"
	EventSessionEnded                   EventType = "SESSION_ENDED"
	EventAgentWorking                   EventType = "AGENT_WORKING"
	EventAgentWaiting                   EventType = "AGENT_WAITING"
	EventToolStarted                    EventType = "TOOL_STARTED"
	EventToolFinished                   EventType = "TOOL_FINISHED"
	EventApprovalRequired               EventType = "APPROVAL_REQUIRED"
	EventApproved                       EventType = "APPROVED"
	EventRejected                       EventType = "REJECTED"
	EventRateLimited                    EventType = "RATE_LIMITED"
	EventQuotaLow                       EventType = "QUOTA_LOW"
	EventQuotaExhausted                 EventType = "QUOTA_EXHAUSTED"
	EventQuotaFailoverRequested         EventType = "QUOTA_FAILOVER_REQUESTED"
	EventQuotaFailoverCompleted         EventType = "QUOTA_FAILOVER_COMPLETED"
	EventQuotaFailoverFailed            EventType = "QUOTA_FAILOVER_FAILED"
	EventQuotaMonitorDegraded           EventType = "QUOTA_MONITOR_DEGRADED"
	EventQuotaMonitorRecovered          EventType = "QUOTA_MONITOR_RECOVERED"
	EventHandoffCompleted               EventType = "HANDOFF_COMPLETED"
	EventMissionStarted                 EventType = "MISSION_STARTED"
	EventMissionStepStarted             EventType = "MISSION_STEP_STARTED"
	EventMissionStepCompleted           EventType = "MISSION_STEP_COMPLETED"
	EventMissionPaused                  EventType = "MISSION_PAUSED"
	EventMissionResumed                 EventType = "MISSION_RESUMED"
	EventMissionCanceled                EventType = "MISSION_CANCELED"
	EventMissionCompleted               EventType = "MISSION_COMPLETED"
	EventMissionFailed                  EventType = "MISSION_FAILED"
	EventHumanInterventionCreated       EventType = "HUMAN_INTERVENTION_CREATED"
	EventHumanInterventionResolved      EventType = "HUMAN_INTERVENTION_RESOLVED"
	EventHumanInterventionStaleRejected EventType = "HUMAN_INTERVENTION_STALE_REJECTED"
	EventMissionResumeRequested         EventType = "MISSION_RESUME_REQUESTED"
	EventMissionCoordinationRequested   EventType = "MISSION_COORDINATION_REQUESTED"
	EventMissionCoordinationMaterialized EventType = "MISSION_COORDINATION_MATERIALIZED"
	EventMissionCoordinationResolved    EventType = "MISSION_COORDINATION_RESOLVED"
	EventError                          EventType = "ERROR"
)

var eventCounter uint64

// Event represents a structured, normalized lifecycle or execution event.
type Event struct {
	ID            string             `json:"id"`
	CorrelationID string             `json:"correlation_id,omitempty"`
	RuntimeID     string             `json:"runtime_id"`
	Provider      string             `json:"provider"`
	Profile       string             `json:"profile"`
	Type          EventType          `json:"type"`
	Summary       string             `json:"summary"`
	Data          map[string]any     `json:"data,omitempty"`
	Timestamp     time.Time          `json:"timestamp"`
	AccountScope  model.AccountScope `json:"account_scope,omitempty"`
}

// WithAccountScope attaches the verified account boundary to an event without
// changing the compatibility constructor used by older producers.
func (e Event) WithAccountScope(scope model.AccountScope) Event {
	if scope.Verifiable() {
		e.AccountScope = scope
	}
	return e
}

// NewEvent creates a timestamped normalized event.
func NewEvent(runtimeID, provider, profile string, eventType EventType, summary string, data map[string]any) Event {
	return NewEventWithCorrelation("", runtimeID, provider, profile, eventType, summary, data)
}

// NewEventWithCorrelation creates an event linked to a broader execution
// timeline, such as a MissionRun or handoff operation.
func NewEventWithCorrelation(correlationID, runtimeID, provider, profile string, eventType EventType, summary string, data map[string]any) Event {
	id := atomic.AddUint64(&eventCounter, 1)
	return Event{
		ID:            fmt.Sprintf("evt-%d", id),
		CorrelationID: correlationID,
		RuntimeID:     runtimeID,
		Provider:      provider,
		Profile:       profile,
		Type:          eventType,
		Summary:       summary,
		Data:          data,
		Timestamp:     time.Now(),
	}
}
