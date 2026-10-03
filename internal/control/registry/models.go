package registry

import (
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

type RuntimeState string

type StartupStage string

const (
	StartupCreated          StartupStage = "CREATED"
	StartupHostStarting     StartupStage = "HOST_STARTING"
	StartupIPCBinding       StartupStage = "IPC_BINDING"
	StartupIPCBound         StartupStage = "IPC_BOUND"
	StartupProtocolReady    StartupStage = "PROTOCOL_READY"
	StartupTerminalStarting StartupStage = "TERMINAL_STARTING"
	StartupTerminalReady    StartupStage = "TERMINAL_READY"
	StartupProviderStarting StartupStage = "PROVIDER_STARTING"
	StartupRunning          StartupStage = "RUNNING"
)

type StartupFault string

const (
	StartupFaultIPCBindFailed       StartupFault = "IPC_BIND_FAILED"
	StartupFaultIPCTimeout          StartupFault = "IPC_TIMEOUT"
	StartupFaultProtocolError       StartupFault = "PROTOCOL_ERROR"
	StartupFaultConPTYStartFailed   StartupFault = "CONPTY_START_FAILED"
	StartupFaultProviderNotFound    StartupFault = "PROVIDER_NOT_FOUND"
	StartupFaultProviderExitedEarly StartupFault = "PROVIDER_EXITED_EARLY"
	StartupFaultWorkspaceInvalid    StartupFault = "WORKSPACE_INVALID"
	StartupFaultPermissionDenied    StartupFault = "PERMISSION_DENIED"
	StartupFaultProcessSupervision  StartupFault = "PROCESS_SUPERVISION_FAILED"
)

const (
	StateStarting RuntimeState = "STARTING"
	StateRunning  RuntimeState = "RUNNING"
	StateWaiting  RuntimeState = "WAITING"
	StateApproval RuntimeState = "APPROVAL"
	StateDetached RuntimeState = "DETACHED"
	StateHandoff  RuntimeState = "HANDOFF"
	StateStopping RuntimeState = "STOPPING"
	StateStopped  RuntimeState = "STOPPED"
	StateFailed   RuntimeState = "FAILED"
	StateStale    RuntimeState = "STALE"
)

const (
	TransportIPC             = "ipc"
	TransportMock            = "mock"
	TransportManagedSSHProxy = "managed-ssh-proxy"
)

// IsRemoteRuntimeTransport reports transports whose provider process identity
// belongs to another machine and must never be interpreted as a local PID.
func IsRemoteRuntimeTransport(transport string) bool {
	return transport == TransportManagedSSHProxy
}

type ControlLevel string

const (
	ControlLevelAPI      ControlLevel = "CONTROL_API"
	ControlLevelEvents   ControlLevel = "EVENTS"
	ControlLevelTerminal ControlLevel = "TERMINAL"
	ControlLevelProcess  ControlLevel = "PROCESS"
)

// RuntimeSession represents a managed execution session under AI Control.
// Note: Secrets, OAuth tokens, and API keys are NEVER stored in this struct or persisted.
type RuntimeSession struct {
	RuntimeID                 string             `json:"runtime_id"`
	AgentID                   string             `json:"agent_id,omitempty"`
	Title                     string             `json:"title,omitempty"`
	ProviderID                string             `json:"provider_id"`
	ProfileID                 string             `json:"profile_id"`
	AccountScope              model.AccountScope `json:"account_scope,omitempty"`
	ProviderSessionID         string             `json:"provider_session_id,omitempty"`
	Model                     string             `json:"model,omitempty"`
	Workspace                 string             `json:"workspace"`
	PID                       int                `json:"pid"`
	ProviderGeneration        int64              `json:"provider_generation,omitempty"`
	HostPID                   int                `json:"host_pid,omitempty"`
	HostGeneration            int64              `json:"host_generation,omitempty"`
	HostBinaryPath            string             `json:"host_binary_path,omitempty"`
	HostBinaryFingerprint     string             `json:"host_binary_fingerprint,omitempty"`
	ProviderBinaryPath        string             `json:"provider_binary_path,omitempty"`
	ProviderBinaryFingerprint string             `json:"provider_binary_fingerprint,omitempty"`
	Binary                    string             `json:"-"`
	Args                      []string           `json:"-"`
	Env                       []string           `json:"-"`
	State                     RuntimeState       `json:"state"`
	StartupStage              StartupStage       `json:"startup_stage,omitempty"`
	StageChangedAt            time.Time          `json:"stage_changed_at,omitempty"`
	LastFault                 StartupFault       `json:"last_fault,omitempty"`
	ControlLevel              ControlLevel       `json:"control_level"`
	StartedAt                 time.Time          `json:"started_at"`
	UpdatedAt                 time.Time          `json:"updated_at"`
	ControlEndpoint           string             `json:"control_endpoint"`
	ParentRuntimeID           string             `json:"parent_runtime_id,omitempty"`
	HandoffType               string             `json:"handoff_type,omitempty"`
	Continuity                string             `json:"continuity,omitempty"`
	LineageID                 string             `json:"lineage_id,omitempty"`
	Labels                    map[string]string  `json:"labels,omitempty"`
	MachineID                 string             `json:"machine_id,omitempty"`
	Location                  string             `json:"location,omitempty"`
	Transport                 string             `json:"transport,omitempty"`
	AttentionReason           string             `json:"attention_reason,omitempty"`
	AttentionContext          string             `json:"attention_context,omitempty"`
	PromptKind                string             `json:"prompt_kind,omitempty"`    // yn | choice | free_text | none
	AttentionKind             string             `json:"attention_kind,omitempty"` // needs_user | working | completed | error | idle
	AttentionFingerprint      string             `json:"attention_fingerprint,omitempty"`
	ProjectID                 string             `json:"project_id,omitempty"`
	ProjectName               string             `json:"project_name,omitempty"`
	LastTaskSummary           string             `json:"last_task_summary,omitempty"`
	DynamicTitle              string             `json:"dynamic_title,omitempty"`
}

// IsActive returns true if the runtime is in an operational lifecycle state.
func (s RuntimeSession) IsActive() bool {
	switch s.State {
	case StateStarting, StateRunning, StateWaiting, StateApproval, StateDetached, StateHandoff:
		return true
	default:
		return false
	}
}

// ProviderLive reports provider-process liveness using the provider generation
// token when available. Legacy rows without ProviderGeneration remain readable,
// but callers must not use their raw PID as a destructive kill target.
func (s RuntimeSession) ProviderLive() bool {
	if s.PID <= 0 {
		return false
	}
	if s.ProviderGeneration > 0 {
		return IsProcessAliveWithGeneration(s.PID, s.ProviderGeneration)
	}
	return IsProcessAlive(s.PID)
}

// HostLive reports whether the SessionHost process behind this registry row is
// still usable. After a machine or service restart, runtimes.json still lists
// the previous generation; the host PID and host-generation checks reject
// those zombies so the terminal WS can 404 instead of upgrading into a black
// screen. The provider PID is intentionally not used for generation matching:
// HostGeneration belongs to the SessionHost, not its child process.
//
// STARTING without a PID is not treated as live: callers must wait for the
// control endpoint (named pipe / unix socket) before advertising attachability.
// A live PID alone is still not proof the endpoint exists — HandleWebSocket
// dials before upgrade.
func (s RuntimeSession) HostLive() bool {
	if s.Transport == TransportMock {
		return true
	}
	// A provider child is never a substitute for SessionHost identity. Legacy
	// rows without HostPID can still be reached through an already-responsive
	// control endpoint by callers that probe IPC first, but raw provider
	// liveness must not keep an unreachable runtime classified as healthy.
	if s.HostPID <= 0 {
		return false
	}
	if s.HostGeneration > 0 {
		return IsProcessAliveWithGeneration(s.HostPID, s.HostGeneration)
	}
	return IsProcessAlive(s.HostPID)
}
