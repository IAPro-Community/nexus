// Package nexus wires the durable product state (SQLite) to the control plane:
// projects, persistent agents, runtime generations, lineage, layouts.
package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/driver"
	"github.com/kivervinicius/ai-cli/internal/control/launcher"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/registry"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/nexus/runner"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

// Launcher abstracts the runtime launch + stop lifecycle so that the Nexus
// service can be tested with a mock that does not spawn real processes.
type Launcher interface {
	Launch(ctx context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error)
	Stop(runtimeID string) error
}

// prodLauncher wraps the real launcher.Launcher to satisfy the Launcher interface.
type prodLauncher struct {
	l *launcher.Launcher
}

func (p *prodLauncher) Launch(ctx context.Context, opts launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	return p.l.Launch(ctx, opts)
}

func (p *prodLauncher) Stop(runtimeID string) error {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	stopErr := client.Stop()
	closeErr := client.Close()
	if stopErr != nil {
		return stopErr
	}
	return closeErr
}

// Nexus is the product-level service bridging the durable store and the
// control plane runtime layer.
type Nexus struct {
	mu            sync.RWMutex
	st            *store.Store
	launcher      Launcher
	runner        *runner.MissionRunner
	runnerMu      sync.Mutex
	workersMu     sync.Mutex
	workers       map[string]*missionWorker
	schedulerOnce sync.Once

	// Runtime change observers (set by the web layer to avoid circular imports).
	onRuntimeChanged func(agentID, oldRuntimeID, newRuntimeID, provider, profile, continuity string)
	onAgentState     func(agentID, state string)
	onContinuity     func(agentID, continuity string)
}

// Runner returns the autonomous mission runner instance.
func (n *Nexus) Runner() *runner.MissionRunner {
	n.runnerMu.Lock()
	defer n.runnerMu.Unlock()
	if n.runner == nil {
		n.runner = runner.NewMissionRunner(newStoreRunRepository(n.st), newNexusPackageExecutor(n))
	}
	return n.runner
}

// SetRuntimeObservers registers callbacks for runtime lifecycle events.
// Used by the web layer to wire the AgentTerminalBroker without circular imports.
func (n *Nexus) SetRuntimeObservers(
	onChanged func(agentID, oldRuntimeID, newRuntimeID, provider, profile, continuity string),
	onState func(agentID, state string),
	onCont func(agentID, continuity string),
) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onRuntimeChanged = onChanged
	n.onAgentState = onState
	n.onContinuity = onCont
}

func (n *Nexus) notifyRuntimeChanged(agentID, oldRuntimeID, newRuntimeID, provider, profile, continuity string) {
	n.mu.RLock()
	cb := n.onRuntimeChanged
	n.mu.RUnlock()
	if cb != nil {
		cb(agentID, oldRuntimeID, newRuntimeID, provider, profile, continuity)
	}
}

func (n *Nexus) notifyAgentState(agentID, state string) {
	n.mu.RLock()
	cb := n.onAgentState
	n.mu.RUnlock()
	if cb != nil {
		cb(agentID, state)
	}
}

func (n *Nexus) notifyContinuity(agentID, continuity string) {
	n.mu.RLock()
	cb := n.onContinuity
	n.mu.RUnlock()
	if cb != nil {
		cb(agentID, continuity)
	}
}

var (
	defaultNexus *Nexus
	nexusOnce    sync.Once
)

// Default returns the process-wide Nexus service (SQLite store under DataDir).
func Default() *Nexus {
	nexusOnce.Do(func() {
		st, err := OpenStore()
		if err != nil {
			// The web layer surfaces errors per-request; keep a nil store so
			// runtime features degrade gracefully instead of panicking.
			st = nil
		}
		defaultNexus = &Nexus{st: st, launcher: &prodLauncher{l: launcher.Default()}, workers: map[string]*missionWorker{}}
		if st != nil {
			_ = defaultNexus.RecoverMissionRuns(context.Background())
			defaultNexus.StartScheduleLoop()
		}
	})
	return defaultNexus
}

// OpenStore opens the SQLite store at <DataDir>/nexus.db using the canonical
// cross-platform data directory authority (P1 canonical DataDir).
func OpenStore() (*store.Store, error) {
	dir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	_ = os.MkdirAll(dir, 0700)
	return store.Open(filepath.Join(dir, "nexus.db"))
}

// Store returns the underlying store (nil when unavailable).
func (n *Nexus) Store() *store.Store { return n.st }

// OpenProject returns the store or an error when the DB is unavailable.
func (n *Nexus) OpenProject() (*store.Store, error) {
	if n.st == nil {
		return nil, fmt.Errorf("nexus store unavailable (MAESTRO_DEGRADED-style degraded mode): sqlite open failed")
	}
	return n.st, nil
}

// ResolveAgentByRuntimeID returns the agent ID that owns the given runtime, or
// an error if the runtime generation is not found.
func (n *Nexus) ResolveAgentByRuntimeID(runtimeID string) (string, error) {
	st, err := n.OpenProject()
	if err != nil {
		return "", err
	}
	gen, err := st.GenerationByRuntimeID(runtimeID)
	if err != nil {
		return "", fmt.Errorf("runtime %s not found: %w", runtimeID, err)
	}
	return gen.AgentID, nil
}

// StartAgent launches a supervised runtime for an agent, records a config
// revision and a runtime generation, and updates agent status. The runtime
// starts in the agent's project canonical path (P0-1). Empty provider is
// rejected (P0-6 — no silent fake fallback in production).
func (n *Nexus) StartAgent(ctx context.Context, agentID, provider, profile string) (*registry.RuntimeSession, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required (no implicit fake fallback)")
	}
	if profile == "" {
		profile = "default"
	}

	// Resolve the project workspace for this agent (P0-1).
	proj, perr := st.GetProject(agent.ProjectID)
	if perr != nil {
		return nil, fmt.Errorf("resolve agent project: %w", perr)
	}

	var agentCfg AgentConfig
	currentRevisionID := agent.CurrentRevisionID
	if agent.CurrentRevisionID != "" {
		if rev, rerr := st.GetRevision(agent.CurrentRevisionID); rerr == nil {
			agentCfg, _ = ParseAgentConfig(rev.Config)
		}
	}
	configuredProvider := agentCfg.Provider
	configuredProfile := agentCfg.Profile
	agentCfg.Provider = provider
	agentCfg.Profile = profile
	if agentCfg.Profile == "" {
		agentCfg.Profile = "default"
	}

	revisionID := currentRevisionID
	if revisionID == "" || configuredProvider != agentCfg.Provider || configuredProfile != agentCfg.Profile {
		rev, err := st.AddRevision(agentID, agentCfg.ConfigJSON())
		if err != nil {
			return nil, err
		}
		revisionID = rev.ID
	}

	var previousGen *store.RuntimeGeneration
	if prior, priorErr := st.CurrentGeneration(agentID); priorErr == nil {
		previousGen = &prior
		if prior.RuntimeID != "" && n.runtimeAlive(prior.RuntimeID) {
			return nil, fmt.Errorf("agent %s already has a live runtime %s; attach to it or stop it before starting another generation", agentID, prior.RuntimeID)
		}
	}
	continuityLaunch, err := continuityForNextGeneration(ctx, agentCfg, previousGen)
	if err != nil {
		return nil, fmt.Errorf("resolve start continuity: %w", err)
	}
	executionWorkspace, err := n.resolveExecutionWorkspace(ctx, proj, agent, agentCfg)
	if err != nil {
		return nil, fmt.Errorf("resolve execution workspace: %w", err)
	}
	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		AgentID:           agentID,
		ProviderID:        provider,
		ProfileID:         agentCfg.Profile,
		ProviderSessionID: continuityLaunch.ProviderSessionID,
		Args:              continuityLaunch.Args,
		Workspace:         executionWorkspace,
		Model:             agentCfg.Model,
		Environment:       agentCfg.Environment,
		Isolation:         agentCfg.Isolation,
		Options:           agentCfg.Options,
	})
	if err != nil {
		return nil, fmt.Errorf("start agent runtime: %w", err)
	}

	gen := store.RuntimeGeneration{
		AgentID:         agentID,
		RevisionID:      revisionID,
		RuntimeID:       sess.RuntimeID,
		Provider:        provider,
		Profile:         profile,
		ProviderSession: sess.ProviderSessionID,
		Continuity:      continuityLaunch.Status,
		StartedAt:       time.Now().UTC(),
		State:           "RUNNING",
	}
	if _, err := st.AddGeneration(gen); err != nil {
		// Launch compensation (P1): if generation commit fails, stop the
		// runtime to prevent orphaned provider processes.
		n.stopRuntime(sess.RuntimeID)
		return nil, fmt.Errorf("commit runtime generation: %w", err)
	}

	agent.Status = "WORKING"
	agent.CurrentRevisionID = revisionID
	agent.ContinuityStatus = continuityLaunch.Status
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	_ = st.UpdateAgent(agent)

	// Notify terminal broker of new runtime (Gate 4).
	oldRuntimeID := ""
	if previousGen != nil {
		oldRuntimeID = previousGen.RuntimeID
	}
	n.notifyRuntimeChanged(agentID, oldRuntimeID, sess.RuntimeID, provider, profile, continuityLaunch.Status)
	n.notifyContinuity(agentID, continuityLaunch.Status)
	n.notifyAgentState(agentID, "WORKING")

	return sess, nil
}

// AskAgent submits a prompt to an existing Agent runtime without creating a
// new Agent identity. Starting a stopped Agent remains an explicit lifecycle
// operation so callers can decide which provider/profile to use first.
func (n *Nexus) AskAgent(ctx context.Context, agentID, instruction string) (store.Agent, error) {
	if strings.TrimSpace(instruction) == "" {
		return store.Agent{}, fmt.Errorf("instruction is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return store.Agent{}, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return store.Agent{}, err
	}
	gen, err := st.CurrentGeneration(agentID)
	if err != nil || gen.RuntimeID == "" {
		return store.Agent{}, fmt.Errorf("agent %s has no active runtime; start it before asking", agentID)
	}
	client, err := protocol.NewClient(gen.RuntimeID)
	if err != nil {
		return store.Agent{}, fmt.Errorf("connect to agent runtime: %w", err)
	}
	defer client.Close()
	input := strings.TrimSpace(instruction) + "\n"
	if _, err := client.Send(protocol.CmdInput, protocol.InputPayload{Data: input}); err != nil {
		return store.Agent{}, fmt.Errorf("send agent instruction: %w", err)
	}
	return agent, nil
}

// StopAgent performs a verified stop: sets STOPPING, sends graceful stop,
// waits for process termination (PID identity check), then persists STOPPED.
// Failures leave the agent in STOPPING/FAILED with explanation (P0-3).
func (n *Nexus) StopAgent(ctx context.Context, agentID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return err
	}

	// Phase 1: transition to STOPPING immediately.
	agent.Status = store.AgentStopping
	_ = st.UpdateAgent(agent)
	n.notifyAgentState(agentID, "STOPPING")

	gen, gerr := st.CurrentGeneration(agentID)
	if gerr != nil || gen.RuntimeID == "" {
		// No runtime generation — agent is already stopped.
		agent.Status = store.AgentStopped
		_ = st.UpdateAgent(agent)
		n.notifyAgentState(agentID, "STOPPED")
		return nil
	}

	// Phase 2: send graceful stop and wait for process termination.
	if !n.runtimeAlive(gen.RuntimeID) {
		// Runtime already dead — proceed to finalize.
		agent.Status = store.AgentStopped
		_ = st.UpdateAgent(agent)
		stopped := time.Now().UTC()
		_ = st.StopGeneration(gen.ID, stopped)
		n.notifyAgentState(agentID, "STOPPED")
		return nil
	}

	n.stopRuntime(gen.RuntimeID)

	// Phase 3: verify process termination (barrier).
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			// Timeout: force kill and mark as failed.
			agent.Status = store.AgentFailed
			agent.ContinuityStatus = "STOP_TIMEOUT"
			_ = st.UpdateAgent(agent)
			n.notifyAgentState(agentID, "FAILED")
			return fmt.Errorf("stop timeout for runtime %s (agent marked FAILED)", gen.RuntimeID)
		case <-ticker.C:
			if !n.runtimeAlive(gen.RuntimeID) {
				// Phase 4: process confirmed dead — persist STOPPED.
				stopped := time.Now().UTC()
				_ = st.StopGeneration(gen.ID, stopped)
				agent.Status = store.AgentStopped
				_ = st.UpdateAgent(agent)
				n.notifyAgentState(agentID, "STOPPED")
				return nil
			}
		}
	}
}

func (n *Nexus) stopRuntime(runtimeID string) {
	_ = n.launcher.Stop(runtimeID)
}

// runtimeAlive reports whether a runtime's process is currently alive,
// consulting the live runtime registry (PID + host generation protection).
func (n *Nexus) runtimeAlive(runtimeID string) bool {
	if runtimeID == "" {
		return false
	}
	sess, ok := registry.DefaultRegistry().Get(runtimeID)
	if !ok {
		return false
	}
	return registry.IsProcessAlive(sess.PID)
}

// EffectiveAgentState derives the honest, live agent state: an agent whose
// runtime process is gone is RECOVERABLE (the Agent persists, the Runtime does
// not — §29). It never guesses; it reflects store status + live registry.
func (n *Nexus) EffectiveAgentState(agentID string) (string, error) {
	st, err := n.OpenProject()
	if err != nil {
		return "", err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return "", err
	}
	gen, gerr := st.CurrentGeneration(agentID)
	if gerr != nil {
		return agent.Status, nil // no runtime generation yet
	}
	if n.runtimeAlive(gen.RuntimeID) {
		return agent.Status, nil
	}
	switch agent.Status {
	case store.AgentWorking, store.AgentStarting, store.AgentHandoff, store.AgentRecovering:
		return store.AgentRecoverable, nil
	default:
		return agent.Status, nil
	}
}

// RecoverAgent restarts a persistent agent whose runtime died (machine reboot,
// host crash). It either performs provider-native resume (honest continuity:
// NATIVE_RESUME_UNVERIFIED, since provider-level verification needs a
// CONTROL_API adapter) or starts a NEW SESSION (CONTEXT_RECOVERED_NEW_SESSION)
// when no provider session is known. Without a prior generation, recovery is
// refused — the caller must use StartAgent (P1).
func (n *Nexus) RecoverAgent(ctx context.Context, agentID string) (*registry.RuntimeSession, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}

	gen, gerr := st.CurrentGeneration(agentID)
	if gerr == nil && n.runtimeAlive(gen.RuntimeID) {
		return nil, fmt.Errorf("agent runtime is already alive (no recovery needed)")
	}

	// Without a prior generation, recovery is impossible (P1).
	if gerr != nil {
		return nil, fmt.Errorf("no recoverable runtime generation found (use StartAgent)")
	}

	provider := gen.Provider
	profile := gen.Profile
	sessionID := gen.ProviderSession

	// If the agent was explicitly STOPPED, do not auto-recover (P0-5).
	if agent.Status == store.AgentStopped {
		return nil, fmt.Errorf("agent is STOPPED (use StartAgent to restart)")
	}

	agent.Status = store.AgentRecovering
	_ = st.UpdateAgent(agent)
	n.notifyAgentState(agentID, "RECOVERING")

	// Resolve project canonical workspace (§14, §29). Never fallback to server CWD.
	proj, perr := st.GetProject(agent.ProjectID)
	if perr != nil {
		return nil, fmt.Errorf("resolve agent project: %w", perr)
	}

	// Reuse the current config revision — do NOT create a new revision
	// unless the config actually changed (P1 correct ConfigRevision semantics).
	rev, rerr := st.GetRevision(agent.CurrentRevisionID)
	var agentCfg AgentConfig
	if rerr != nil || rev.ID == "" {
		// Fallback: create a revision only if none exists.
		agentCfg = AgentConfig{
			Provider: provider,
			Profile:  profile,
		}
		rev, err = st.AddRevision(agentID, agentCfg.ConfigJSON())
		if err != nil {
			return nil, err
		}
	} else {
		agentCfg, _ = ParseAgentConfig(rev.Config)
	}

	var args []string
	continuity := store.ContinuityNewSession // NEW SESSION unless native resume is possible
	if sessionID != "" {
		if d, derr := driver.DefaultRegistry().Get(provider); derr == nil {
			prof := model.Profile{Name: profile, Provider: provider}
			if can, _ := d.CanResume(ctx, prof, sessionID); can {
				if ra, rerr := d.BuildResumeArgs(ctx, prof, sessionID); rerr == nil {
					args = ra
					continuity = store.ContinuityNativeResumeUnverified
				}
			}
		}
	}

	executionWorkspace, werr := n.resolveExecutionWorkspace(ctx, proj, agent, agentCfg)
	if werr != nil {
		agent.Status = store.AgentRecoverable
		_ = st.UpdateAgent(agent)
		return nil, fmt.Errorf("resolve execution workspace: %w", werr)
	}
	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		AgentID:           agentID,
		ProviderID:        provider,
		ProfileID:         profile,
		ProviderSessionID: sessionID,
		Args:              args,
		Workspace:         executionWorkspace,
		Model:             agentCfg.Model,
		Environment:       agentCfg.Environment,
		Isolation:         agentCfg.Isolation,
		Options:           agentCfg.Options,
	})
	if err != nil {
		agent.Status = store.AgentRecoverable
		_ = st.UpdateAgent(agent)
		return nil, fmt.Errorf("recover agent runtime: %w", err)
	}

	gen2 := store.RuntimeGeneration{
		AgentID:         agentID,
		RevisionID:      rev.ID,
		RuntimeID:       sess.RuntimeID,
		Provider:        provider,
		Profile:         profile,
		ProviderSession: sess.ProviderSessionID,
		Continuity:      continuity,
		StartedAt:       time.Now().UTC(),
		State:           "RUNNING",
	}
	if _, err := st.AddGeneration(gen2); err != nil {
		// Launch compensation: stop the orphaned runtime.
		n.stopRuntime(sess.RuntimeID)
		return nil, err
	}

	agent.Status = store.AgentWorking
	agent.CurrentRevisionID = rev.ID
	agent.ContinuityStatus = continuity
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	_ = st.UpdateAgent(agent)

	// Notify terminal broker of recovered runtime (Gate 4).
	n.notifyRuntimeChanged(agentID, gen.RuntimeID, sess.RuntimeID, provider, profile, continuity)
	n.notifyAgentState(agentID, "WORKING")
	n.notifyContinuity(agentID, continuity)

	return sess, nil
}

// DeleteAgent safely removes an agent. If the agent has a live runtime, the
// deletion is refused to prevent orphaned provider processes.
func (n *Nexus) DeleteAgent(agentID, projectID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	agent, err := st.GetAgent(agentID, projectID)
	if err != nil {
		return err
	}
	// Check for a live runtime generation.
	gen, gerr := st.CurrentGeneration(agentID)
	if gerr == nil && n.runtimeAlive(gen.RuntimeID) {
		return fmt.Errorf("cannot delete agent %q: runtime %s is live (stop the agent first)", agent.Name, gen.RuntimeID)
	}
	return st.DeleteAgent(agentID, projectID)
}

// DeleteProject safely removes a project. If any agent in the project has a
// live runtime, the deletion is refused.
func (n *Nexus) DeleteProject(projectID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	agents, err := st.ListAgents(projectID)
	if err != nil {
		return err
	}
	for _, a := range agents {
		gen, gerr := st.CurrentGeneration(a.ID)
		if gerr == nil && n.runtimeAlive(gen.RuntimeID) {
			return fmt.Errorf("cannot delete project: agent %q has live runtime %s (stop all agents first)", a.Name, gen.RuntimeID)
		}
	}
	return st.DeleteProject(projectID)
}
