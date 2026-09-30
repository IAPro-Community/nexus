package nexus

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// --- P0-1: Project execution isolation ---

func TestStartAgentUsesProjectWorkspace(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projDir := t.TempDir()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: projDir})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatalf("StartAgent: %v", err)
	}
	if sess.Workspace != projDir {
		t.Errorf("runtime workspace = %q, want project canonical path %q", sess.Workspace, projDir)
	}
}
func TestStartAgentNeverUsesServerCWD(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	projDir := t.TempDir()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: projDir})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	orig, _ := os.Getwd()
	altDir := t.TempDir()
	_ = os.Chdir(altDir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	if sess.Workspace != projDir {
		t.Errorf("runtime workspace = %q (server CWD %q), want %q", sess.Workspace, altDir, projDir)
	}
}

// --- P0-5: Effective state reconciliation ---

func TestEffectiveStateReconciledInList(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	dead := &store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: "rev_x",
		RuntimeID:  "rt_dead_xyz",
		Provider:   "fake",
		Profile:    "default",
		StartedAt:  time.Now().UTC(),
		State:      "RUNNING",
	}
	_, _ = st.AddGeneration(*dead)
	agent.Status = store.AgentWorking
	_ = st.UpdateAgent(agent)

	state, err := n.EffectiveAgentState(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state != store.AgentRecoverable {
		t.Errorf("effective state = %q, want RECOVERABLE", state)
	}
}

// --- P0-6: No silent fake provider fallback ---

func TestStartAgentRejectsEmptyProvider(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	_, err := n.StartAgent(context.Background(), agent.ID, "", "default")
	if err == nil {
		t.Fatal("expected error when starting with empty provider, got nil")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

// --- P0-3: StopAgent verified barrier ---

func TestStopAgentVerifiedBarrier(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	_, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}

	err = n.StopAgent(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("StopAgent: %v", err)
	}

	a, _ := st.GetAgent(agent.ID, "")
	if a.Status != store.AgentStopped {
		t.Errorf("agent status after stop = %q, want STOPPED", a.Status)
	}

	gen, err := st.CurrentGeneration(agent.ID)
	if err == nil && gen.StoppedAt == nil {
		t.Error("generation stopped_at should be set after stop")
	}
}

// --- P0-4: Lifecycle-aware delete ---

func TestDeleteAgentWithLiveRuntimeRefuses(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	_, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}

	err = n.DeleteAgent(agent.ID, proj.ID)
	if err == nil {
		t.Fatal("expected error when deleting live agent, got nil")
	}
	if !strings.Contains(err.Error(), "live") && !strings.Contains(err.Error(), "runtime") {
		t.Errorf("error should mention live runtime, got: %v", err)
	}
}

func TestDeleteProjectWithLiveAgentsRefuses(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	_, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}

	err = n.DeleteProject(proj.ID)
	if err == nil {
		t.Fatal("expected error when deleting project with live agents, got nil")
	}
}

// --- P1: ConfigRevision semantics ---

func TestConfigRevisionNotDuplicatedOnRestart(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	revs1, _ := st.ListRevisions(agent.ID)
	if len(revs1) != 1 {
		t.Fatalf("expected 1 revision after first start, got %d", len(revs1))
	}

	// Simulate runtime crash: remove from live registry so runtimeAlive returns false.
	registry.DefaultRegistry().Delete(sess.RuntimeID)
	// Re-read agent to get the updated CurrentRevisionID from StartAgent.
	agent, _ = st.GetAgent(agent.ID, "")
	agent.Status = store.AgentWorking
	_ = st.UpdateAgent(agent)

	// Recover reuses existing revision.
	_, err = n.RecoverAgent(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("RecoverAgent: %v", err)
	}
	revs2, _ := st.ListRevisions(agent.ID)
	if len(revs2) != 1 {
		t.Errorf("expected still 1 revision after recover (config unchanged), got %d", len(revs2))
	}
}

// --- Effective state during STOPPING ---

func TestEffectiveStateDuringStop(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	dead := &store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: "rev_x",
		RuntimeID:  "rt_dead_xyz",
		Provider:   "fake",
		Profile:    "default",
		StartedAt:  time.Now().UTC(),
		State:      "STOPPING",
	}
	_, _ = st.AddGeneration(*dead)
	agent.Status = store.AgentStopping
	_ = st.UpdateAgent(agent)

	state, err := n.EffectiveAgentState(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state == store.AgentRecoverable {
		t.Error("STOPPING agent should not be RECOVERABLE")
	}
}

// --- Recover without prior generation ---

func TestRecoverWithoutGenerationFails(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	_, err := n.RecoverAgent(context.Background(), agent.ID)
	if err == nil {
		t.Fatal("expected error when recovering agent with no prior generation")
	}
}

func TestRecoverAlreadyAliveReturnsSession(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	started, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := n.RecoverAgent(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("RecoverAgent while alive: %v", err)
	}
	if recovered == nil || recovered.RuntimeID != started.RuntimeID {
		t.Fatalf("expected idempotent recover to return live session %s, got %#v", started.RuntimeID, recovered)
	}
}

// --- Recover stopped agent ---

func TestRecoverStoppedAgentFails(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	// Agent was explicitly STOPPED (never started) — recover should refuse.
	_, err := n.RecoverAgent(context.Background(), agent.ID)
	if err == nil {
		t.Fatal("expected error when recovering explicitly STOPPED agent")
	}
}

// --- Effective state: STARTING with dead runtime ---

func TestEffectiveStateStartingDeadRuntime(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	dead := &store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: "rev_x",
		RuntimeID:  "rt_dead_starting",
		Provider:   "fake",
		Profile:    "default",
		StartedAt:  time.Now().UTC(),
		State:      "STARTING",
	}
	_, _ = st.AddGeneration(*dead)
	agent.Status = store.AgentStarting
	_ = st.UpdateAgent(agent)

	state, err := n.EffectiveAgentState(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state != store.AgentRecoverable {
		t.Errorf("STARTING agent with dead runtime: got %q, want RECOVERABLE", state)
	}
}

func TestRecoverAgentDeadRuntimeStartsNewSessionUnlessNative(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	started, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}

	// Simulate dead runtime (host reboot / process killed)
	registry.DefaultRegistry().Delete(started.RuntimeID)

	recovered, err := n.RecoverAgent(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("RecoverAgent: %v", err)
	}
	if recovered.RuntimeID == started.RuntimeID {
		t.Fatalf("expected new runtime session after dead runtime recovery, got same %s", recovered.RuntimeID)
	}

	gen, err := st.CurrentGeneration(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gen.Continuity != store.ContinuityNewSession {
		t.Errorf("continuity = %q, want %q", gen.Continuity, store.ContinuityNewSession)
	}
}

func TestRecoverAgentStaleRuntimeStartsNewSession(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Dev"})

	started, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.DefaultRegistry().UpdateState(started.RuntimeID, registry.StateStale); err != nil {
		t.Fatal(err)
	}

	recovered, err := n.RecoverAgent(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("RecoverAgent: %v", err)
	}
	if recovered.RuntimeID == started.RuntimeID {
		t.Fatalf("expected new runtime session after stale runtime recovery, got same %s", started.RuntimeID)
	}
}
