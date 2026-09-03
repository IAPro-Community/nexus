package workspace

import (
	"testing"
	"time"
)

func TestRecoveryCenterRegisterAndList(t *testing.T) {
	rc := NewRecoveryCenter()

	now := time.Now()
	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-1",
		AgentTitle:    "Backend Developer",
		ProjectID:     "project-1",
		ProviderID:    "codex",
		ProfileID:     "kivergmail",
		RecoveryState: RecoveryStateRecoverable,
		StoppedAt:     &now,
	})

	agents := rc.ListAgents()
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].AgentID != "agent-1" {
		t.Fatalf("expected agent-1, got %s", agents[0].AgentID)
	}
}

func TestRecoveryCenterRecoverableCount(t *testing.T) {
	rc := NewRecoveryCenter()

	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-1",
		RecoveryState: RecoveryStateRecoverable,
	})
	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-2",
		RecoveryState: RecoveryStateRunning,
	})
	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-3",
		RecoveryState: RecoveryStateRecoverable,
	})

	if rc.RecoverableCount() != 2 {
		t.Fatalf("expected 2 recoverable agents, got %d", rc.RecoverableCount())
	}
}

func TestRecoveryCenterMarkRunning(t *testing.T) {
	rc := NewRecoveryCenter()

	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-1",
		RecoveryState: RecoveryStateRecoverable,
		Message:       "runtime lost",
	})

	rc.MarkRunning("agent-1", "runtime-new")

	info := rc.GetAgent("agent-1")
	if info.RecoveryState != RecoveryStateRunning {
		t.Fatalf("expected RUNNING, got %s", info.RecoveryState)
	}
	if info.LastRuntimeID != "runtime-new" {
		t.Fatalf("expected runtime-new, got %s", info.LastRuntimeID)
	}
	if info.Message != "" {
		t.Fatalf("expected empty message, got %s", info.Message)
	}
}

func TestRecoveryCenterMarkLost(t *testing.T) {
	rc := NewRecoveryCenter()

	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-1",
		RecoveryState: RecoveryStateRecoverable,
	})

	rc.MarkLost("agent-1", "provider session expired")

	info := rc.GetAgent("agent-1")
	if info.RecoveryState != RecoveryStateLost {
		t.Fatalf("expected LOST, got %s", info.RecoveryState)
	}
	if info.Message != "provider session expired" {
		t.Fatalf("expected provider session expired, got %s", info.Message)
	}
	if info.StoppedAt == nil {
		t.Fatal("expected stoppedAt to be set")
	}
}

func TestRecoveryCenterAllRecoverable(t *testing.T) {
	rc := NewRecoveryCenter()

	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-1",
		RecoveryState: RecoveryStateRecoverable,
		ResumeCapability: "SUPPORTED",
	})
	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-2",
		RecoveryState: RecoveryStateRunning,
	})
	rc.RegisterAgent(&AgentRecoveryInfo{
		AgentID:       "agent-3",
		RecoveryState: RecoveryStateRecoverable,
		ResumeCapability: "UNSUPPORTED",
	})

	recoverable := rc.AllRecoverable()
	if len(recoverable) != 2 {
		t.Fatalf("expected 2 recoverable agents, got %d", len(recoverable))
	}
}
