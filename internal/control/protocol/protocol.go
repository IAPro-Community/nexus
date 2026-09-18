package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

const ProtocolVersion = 1

type CommandType string

const (
	CmdPing            CommandType = "ping"
	CmdStatus          CommandType = "status"
	CmdMetadata        CommandType = "metadata"
	CmdAttach          CommandType = "attach"
	CmdDetach          CommandType = "detach"
	CmdResize          CommandType = "resize"
	CmdInput           CommandType = "input"
	CmdSubmitPrompt    CommandType = "submit_prompt"
	CmdStop            CommandType = "stop"
	CmdTerminate       CommandType = "terminate"
	CmdHandoff         CommandType = "handoff"
	CmdContinue        CommandType = "continue"
	CmdEvents          CommandType = "events"
	CmdUsage           CommandType = "usage"
	CmdSlash           CommandType = "slash"
	CmdLeaseAcquire    CommandType = "lease_acquire"
	CmdLeaseRelease    CommandType = "lease_release"
	CmdRuntimeChanged  CommandType = "runtime_changed"  // Gate 4: notify WS clients of generation switch
	CmdAgentState      CommandType = "agent_state"      // Gate 4: push effective agent state
	CmdContinuityState CommandType = "continuity_state" // Gate 4: push continuity status
)

// Request is a versioned command request sent to a SessionHost.
type Request struct {
	Version   int             `json:"version"`
	ID        string          `json:"id,omitempty"`
	Command   CommandType     `json:"command"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// Response is the structured reply from a SessionHost.
type Response struct {
	Version       int             `json:"version"`
	ID            string          `json:"id,omitempty"`
	OK            bool            `json:"ok"`
	Code          string          `json:"code,omitempty"`
	RuntimeID     string          `json:"runtime_id,omitempty"`
	Action        string          `json:"action,omitempty"`
	State         string          `json:"state,omitempty"`
	Message       string          `json:"message,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Data          json.RawMessage `json:"data,omitempty"`
	Error         string          `json:"error,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// ControlResult is the stable, redacted result envelope for explicit control
// actions. It is intentionally separate from InputPayload: terminal bytes
// must never be interpreted as control protocol.
type ControlResult struct {
	OK            bool   `json:"ok"`
	Code          string `json:"code,omitempty"`
	RuntimeID     string `json:"runtime_id,omitempty"`
	Action        string `json:"action,omitempty"`
	State         string `json:"state,omitempty"`
	Message       string `json:"message,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// ResizePayload specifies terminal window dimensions.
type ResizePayload struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// InputPayload contains raw terminal input bytes.
type InputPayload struct {
	Data string `json:"data"` // Base64 or plain string
}

// EventsPayload bounds a read-only event query.
type EventsPayload struct {
	Limit int `json:"limit,omitempty"`
}

// EventData is the redacted transport representation returned by CmdEvents.
// Keeping this type in protocol avoids coupling IPC clients to the event bus.
type EventData struct {
	ID            string    `json:"id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	RuntimeID     string    `json:"runtime_id"`
	Provider      string    `json:"provider"`
	Profile       string    `json:"profile"`
	Type          string    `json:"type"`
	Summary       string    `json:"summary"`
	Timestamp     time.Time `json:"timestamp"`
}

// UsageData is a bounded, read-only quota snapshot for monitor/control UIs.
type UsageData struct {
	RuntimeID     string  `json:"runtime_id"`
	ProviderID    string  `json:"provider_id"`
	ProfileID     string  `json:"profile_id"`
	Status        string  `json:"status"`
	PercentLeft   float64 `json:"percent_left,omitempty"`
	FetchedAtUnix int64   `json:"fetched_at_unix,omitempty"`
}

// SubmitPromptPayload is a high-level prompt for an existing supervised runtime.
// Unlike CmdInput it does not participate in the interactive single-writer lease
// and is forwarded literally, including a leading slash.
type SubmitPromptPayload struct {
	Prompt string `json:"prompt"`
}

// SlashPayload contains an explicit control command. It is not accepted from
// CmdInput and exists only for command-palette/CLI compatibility.
type SlashPayload struct {
	RawCommand string `json:"raw_command"`
}

// HandoffPayload specifies target profile for same-provider handoff.
type HandoffPayload struct {
	TargetProfile string `json:"target_profile"`
}

// ContinuePayload specifies target provider & profile for cross-provider context handoff.
type ContinuePayload struct {
	TargetProvider string `json:"target_provider"`
	TargetProfile  string `json:"target_profile,omitempty"`
}

// RuntimeChangedPayload notifies WS clients that the agent's runtime generation changed (Gate 4).
type RuntimeChangedPayload struct {
	AgentID      string `json:"agent_id"`
	OldRuntimeID string `json:"old_runtime_id,omitempty"`
	NewRuntimeID string `json:"new_runtime_id"`
	Provider     string `json:"provider"`
	Profile      string `json:"profile"`
	Continuity   string `json:"continuity"`
}

// AgentStatePayload pushes the honest effective agent state to WS clients (Gate 4).
type AgentStatePayload struct {
	AgentID string `json:"agent_id"`
	State   string `json:"state"`
}

// ContinuityStatePayload pushes continuity status updates (Gate 4).
type ContinuityStatePayload struct {
	AgentID    string `json:"agent_id"`
	Continuity string `json:"continuity"`
}

// StatusData contains runtime status reported by SessionHost.
type StatusData struct {
	RuntimeID           string    `json:"runtime_id"`
	ProviderID          string    `json:"provider_id"`
	ProfileID           string    `json:"profile_id"`
	ProviderSessionID   string    `json:"provider_session_id,omitempty"`
	Workspace           string    `json:"workspace"`
	PID                 int       `json:"pid"`
	State               string    `json:"state"`
	StartupStage        string    `json:"startup_stage,omitempty"`
	StageChangedAt      time.Time `json:"stage_changed_at,omitempty"`
	LastFault           string    `json:"last_fault,omitempty"`
	ControlLevel        string    `json:"control_level"`
	StartedAt           time.Time `json:"started_at"`
	QuotaStatus         string    `json:"quota_status,omitempty"`
	QuotaPercent        float64   `json:"quota_percent,omitempty"`
	DroppedOutputChunks uint64    `json:"dropped_output_chunks,omitempty"`
	AttentionReason     string    `json:"attention_reason,omitempty"`
	AttentionContext    string    `json:"attention_context,omitempty"`
	AttentionKind       string    `json:"attention_kind,omitempty"`
	PromptKind          string    `json:"prompt_kind,omitempty"`
	Continuity          string    `json:"continuity,omitempty"`
}

// NewRequest creates a standard request.
func NewRequest(cmd CommandType, payload any) (Request, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return Request{}, err
		}
		raw = b
	}
	return Request{
		Version:   ProtocolVersion,
		ID:        fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Command:   cmd,
		Payload:   raw,
		Timestamp: time.Now(),
	}, nil
}

// NewResponse creates a successful response.
func NewResponse(data any) (Response, error) {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return Response{}, err
		}
		raw = b
	}
	return Response{
		Version:   ProtocolVersion,
		OK:        true,
		Data:      raw,
		Timestamp: time.Now(),
	}, nil
}

// NewControlResponse creates a structured response for a typed control
// command. The fields are duplicated at the top level for clients that do not
// decode Data, while Data contains the same stable envelope for compatibility.
func NewControlResponse(result ControlResult) Response {
	raw, _ := json.Marshal(result)
	return Response{
		Version:       ProtocolVersion,
		OK:            result.OK,
		Code:          result.Code,
		RuntimeID:     result.RuntimeID,
		Action:        result.Action,
		State:         result.State,
		Message:       result.Message,
		CorrelationID: result.CorrelationID,
		Data:          raw,
		Timestamp:     time.Now(),
	}
}

// NewErrorResponse creates an error response.
func NewErrorResponse(errMsg string) Response {
	return Response{
		Version:   ProtocolVersion,
		OK:        false,
		Code:      errMsg,
		Message:   errMsg,
		Error:     errMsg,
		Timestamp: time.Now(),
	}
}
