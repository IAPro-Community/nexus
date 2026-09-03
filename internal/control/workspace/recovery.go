package workspace

import (
	"time"
)

// RecoveryState represents the recovery state of an agent.
type RecoveryState string

const (
	RecoveryStateRunning   RecoveryState = "RUNNING"
	RecoveryStateRecoverable RecoveryState = "RECOVERABLE"
	RecoveryStateLost      RecoveryState = "LOST"
	RecoveryStateStopped   RecoveryState = "STOPPED"
)

// AgentRecoveryInfo contains information about an agent's recovery status.
type AgentRecoveryInfo struct {
	AgentID           string        `json:"agentId"`
	AgentTitle        string        `json:"agentTitle"`
	ProjectID         string        `json:"projectId"`
	ProviderID        string        `json:"providerId"`
	ProfileID         string        `json:"profileId"`
	RecoveryState     RecoveryState `json:"recoveryState"`
	ResumeCapability  string        `json:"resumeCapability"` // SUPPORTED, PARTIAL, UNSUPPORTED, UNKNOWN
	ProviderSessionID string        `json:"providerSessionId,omitempty"`
	LastRuntimeID     string        `json:"lastRuntimeId,omitempty"`
	LastKnownPID      int           `json:"lastKnownPid,omitempty"`
	StoppedAt         *time.Time    `json:"stoppedAt,omitempty"`
	Message           string        `json:"message,omitempty"`
}

// RecoveryAction represents an action that can be taken to recover an agent.
type RecoveryAction string

const (
	RecoveryActionRecover     RecoveryAction = "recover"
	RecoveryActionStartFresh  RecoveryAction = "start_fresh"
	RecoveryActionOpenShell   RecoveryAction = "open_shell"
	RecoveryActionDismiss     RecoveryAction = "dismiss"
)

// RecoveryCenter manages agent recovery after restart.
type RecoveryCenter struct {
	agents map[string]*AgentRecoveryInfo // agentID -> info
}

// NewRecoveryCenter creates a new recovery center.
func NewRecoveryCenter() *RecoveryCenter {
	return &RecoveryCenter{
		agents: make(map[string]*AgentRecoveryInfo),
	}
}

// RegisterAgent registers an agent for recovery tracking.
func (rc *RecoveryCenter) RegisterAgent(info *AgentRecoveryInfo) {
	rc.agents[info.AgentID] = info
}

// UpdateState updates the recovery state of an agent.
func (rc *RecoveryCenter) UpdateState(agentID string, state RecoveryState, message string) {
	if info, ok := rc.agents[agentID]; ok {
		info.RecoveryState = state
		info.Message = message
	}
}

// GetAgent returns recovery info for an agent.
func (rc *RecoveryCenter) GetAgent(agentID string) *AgentRecoveryInfo {
	return rc.agents[agentID]
}

// ListAgents returns all agents with their recovery states.
func (rc *RecoveryCenter) ListAgents() []*AgentRecoveryInfo {
	result := make([]*AgentRecoveryInfo, 0, len(rc.agents))
	for _, info := range rc.agents {
		result = append(result, info)
	}
	return result
}

// RecoverableCount returns the number of agents in RECOVERABLE state.
func (rc *RecoveryCenter) RecoverableCount() int {
	count := 0
	for _, info := range rc.agents {
		if info.RecoveryState == RecoveryStateRecoverable {
			count++
		}
	}
	return count
}

// AllRecoverable returns all agents in RECOVERABLE state.
func (rc *RecoveryCenter) AllRecoverable() []*AgentRecoveryInfo {
	result := make([]*AgentRecoveryInfo, 0)
	for _, info := range rc.agents {
		if info.RecoveryState == RecoveryStateRecoverable {
			result = append(result, info)
		}
	}
	return result
}

// MarkRunning marks an agent as running (recovery successful).
func (rc *RecoveryCenter) MarkRunning(agentID string, runtimeID string) {
	if info, ok := rc.agents[agentID]; ok {
		info.RecoveryState = RecoveryStateRunning
		info.LastRuntimeID = runtimeID
		info.Message = ""
	}
}

// MarkLost marks an agent as lost (cannot recover).
func (rc *RecoveryCenter) MarkLost(agentID string, message string) {
	if info, ok := rc.agents[agentID]; ok {
		info.RecoveryState = RecoveryStateLost
		info.Message = message
		now := time.Now()
		info.StoppedAt = &now
	}
}

// MarkStopped marks an agent as stopped.
func (rc *RecoveryCenter) MarkStopped(agentID string) {
	if info, ok := rc.agents[agentID]; ok {
		info.RecoveryState = RecoveryStateStopped
		now := time.Now()
		info.StoppedAt = &now
	}
}
