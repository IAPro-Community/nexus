package attention

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/kivervinicius/ai-cli/internal/core/security"
)

const (
	DefaultReplayLimit = 64
	DefaultStreamLimit = 256
	MaxMessageRunes    = 280
)

type State string

const (
	StateWorking         State = "WORKING"
	StateWaitingInput    State = "WAITING_INPUT"
	StateWaitingApproval State = "WAITING_APPROVAL"
	StateCompleted       State = "COMPLETED"
	StateFailed          State = "FAILED"
	StateRateLimited     State = "RATE_LIMITED"
	StateRecoverable     State = "RECOVERABLE"
)

func (s State) Valid() bool {
	switch s {
	case StateWorking, StateWaitingInput, StateWaitingApproval, StateCompleted, StateFailed, StateRateLimited, StateRecoverable:
		return true
	default:
		return false
	}
}

type Source string

const (
	SourceStructuredProviderEvent Source = "STRUCTURED_PROVIDER_EVENT"
	SourceProviderHook            Source = "PROVIDER_HOOK"
	SourceTerminalHeuristic       Source = "TERMINAL_HEURISTIC"
	SourceProcessExit             Source = "PROCESS_EXIT"
	SourceFlowRunner              Source = "FLOW_RUNNER"
	SourceSystem                  Source = "SYSTEM"
)

func (s Source) Valid() bool {
	switch s {
	case SourceStructuredProviderEvent, SourceProviderHook, SourceTerminalHeuristic, SourceProcessExit, SourceFlowRunner, SourceSystem:
		return true
	default:
		return false
	}
}

type AttentionEvent struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"projectId"`
	AgentID        string     `json:"agentId,omitempty"`
	RuntimeID      string     `json:"runtimeId,omitempty"`
	FlowRunID      string     `json:"flowRunId,omitempty"`
	ViewLogicalKey string     `json:"viewLogicalKey,omitempty"`
	State          State      `json:"state"`
	Reason         string     `json:"reason"`
	Message        string     `json:"message,omitempty"`
	Source         Source     `json:"source"`
	OccurredAt     time.Time  `json:"occurredAt"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
}

type ReplayFilter struct {
	ProjectID      string
	AgentID        string
	RuntimeID      string
	FlowRunID      string
	ViewLogicalKey string
	Limit          int
}

type ProjectAttentionSummary struct {
	ProjectID        string    `json:"projectId"`
	Total            int       `json:"total"`
	Working          int       `json:"working"`
	WaitingInput     int       `json:"waitingInput"`
	WaitingApproval  int       `json:"waitingApproval"`
	Completed        int       `json:"completed"`
	CompletedUnread  int       `json:"completedUnread"`
	Failed           int       `json:"failed"`
	RateLimited      int       `json:"rateLimited"`
	Recoverable      int       `json:"recoverable"`
	PriorityState    State     `json:"priorityState,omitempty"`
	PriorityReason   string    `json:"priorityReason,omitempty"`
	PriorityEventID  string    `json:"priorityEventId,omitempty"`
	LatestOccurredAt time.Time `json:"latestOccurredAt,omitempty"`
}

func NormalizeEvent(event AttentionEvent) (AttentionEvent, error) {
	event.ID = strings.TrimSpace(event.ID)
	event.ProjectID = strings.TrimSpace(event.ProjectID)
	event.AgentID = strings.TrimSpace(event.AgentID)
	event.RuntimeID = strings.TrimSpace(event.RuntimeID)
	event.FlowRunID = strings.TrimSpace(event.FlowRunID)
	event.ViewLogicalKey = strings.TrimSpace(event.ViewLogicalKey)
	event.Reason = strings.TrimSpace(event.Reason)
	event.Message = sanitizeMessage(event.Message)
	event.OccurredAt = event.OccurredAt.UTC()

	if event.AcknowledgedAt != nil {
		ack := event.AcknowledgedAt.UTC()
		event.AcknowledgedAt = &ack
	}

	switch {
	case event.ID == "":
		return AttentionEvent{}, fmt.Errorf("attention event id is required")
	case event.ProjectID == "":
		return AttentionEvent{}, fmt.Errorf("attention event projectId is required")
	case !event.State.Valid():
		return AttentionEvent{}, fmt.Errorf("attention event state %q is invalid", event.State)
	case event.Reason == "":
		return AttentionEvent{}, fmt.Errorf("attention event reason is required")
	case !event.Source.Valid():
		return AttentionEvent{}, fmt.Errorf("attention event source %q is invalid", event.Source)
	case event.OccurredAt.IsZero():
		return AttentionEvent{}, fmt.Errorf("attention event occurredAt is required")
	case event.AcknowledgedAt != nil && event.AcknowledgedAt.Before(event.OccurredAt):
		return AttentionEvent{}, fmt.Errorf("attention event acknowledgedAt cannot be before occurredAt")
	default:
		return event, nil
	}
}

func sanitizeMessage(message string) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}

	redacted := security.Redact(message)
	clean := strings.Map(func(r rune) rune {
		if r == unicode.ReplacementChar {
			return -1
		}
		if unicode.IsControl(r) {
			switch r {
			case '\n', '\r', '\t':
				return ' '
			default:
				return -1
			}
		}
		return r
	}, redacted)
	clean = strings.Join(strings.Fields(clean), " ")

	return clampRunes(clean, MaxMessageRunes)
}

func clampRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
