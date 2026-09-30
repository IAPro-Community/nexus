// Package nexus wires the durable product state (SQLite) to the control plane:
// projects, persistent agents, runtime generations, lineage, layouts.
package nexus

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/device"
	"github.com/IAPro-Community/nexus/internal/core/entitlement"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/capacitymonitor"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	profilepkg "github.com/IAPro-Community/nexus/internal/profile"
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
	return p.l.Stop(runtimeID)
}

// Nexus is the product-level service bridging the durable store and the
// control plane runtime layer.
type Nexus struct {
	mu                         sync.RWMutex
	st                         *store.Store
	launcher                   Launcher
	drivers                    *driver.Registry
	runtimeReg                 *registry.Registry
	runner                     *runner.MissionRunner
	runnerMu                   sync.Mutex
	submitPrompt               func(runtimeID, prompt string) error
	maestroStatus              func() MaestroStatus
	workersMu                  sync.Mutex
	workers                    map[string]*missionWorker
	schedulerOnce              sync.Once
	scheduleCancel             context.CancelFunc
	quotaMonitor               *QuotaMonitorService
	capacityMonitor            *capacitymonitor.Monitor
	capacityMonitorCancel      context.CancelFunc
	attentionOnce              sync.Once
	attentionCenter            *runner.AttentionCenter
	projectIntelligenceMu      sync.Mutex
	projectIntelligenceRunning map[string]bool
	entitlementService         entitlement.Service
	entitlementSubjectID       string

	// Runtime change observers (set by the web layer to avoid circular imports).
	onRuntimeChanged func(agentID, oldRuntimeID, newRuntimeID, provider, profile, continuity string)
	onAgentState     func(agentID, state string)
	onContinuity     func(agentID, continuity string)

	// Optional test-only intelligence provider. When set, ConfiguredIntelligenceProvider
	// returns it instead of resolving from config — used to prove Create does not hang.
	intelligenceProviderForTest intelligence.IntelligenceProvider
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

// AttentionCenter returns the read-model query surface for mission attention items.
func (n *Nexus) AttentionCenter() *runner.AttentionCenter {
	n.attentionOnce.Do(func() {
		n.attentionCenter = runner.NewAttentionCenter(newStoreRunRepository(n.st))
	})
	return n.attentionCenter
}

// ResolveMissionIntervention validates and persists a human decision for a
// blocked mission, then resumes autonomous execution.
func (n *Nexus) ResolveMissionIntervention(ctx context.Context, runID, interventionID string, version int, optionID, resolvedBy string) (*runner.MissionRun, error) {
	run, _, err := n.ResolveMissionInterventionWithOutcome(ctx, runID, interventionID, version, optionID, resolvedBy)
	return run, err
}

func (n *Nexus) ResolveMissionInterventionWithOutcome(ctx context.Context, runID, interventionID string, version int, optionID, resolvedBy string) (*runner.MissionRun, bool, error) {
	run, created, err := n.Runner().ResolveInterventionWithOutcome(ctx, runID, interventionID, version, optionID, resolvedBy)
	if err != nil {
		return nil, false, err
	}
	if created {
		n.restartMissionWorker(runID)
	} else {
		n.StartMissionWorker(runID)
	}
	return run, created, nil
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
		defaultNexus = &Nexus{
			st: st, launcher: &prodLauncher{l: launcher.Default()}, drivers: driver.DefaultRegistry(),
			runtimeReg: registry.DefaultRegistry(), workers: map[string]*missionWorker{},
			projectIntelligenceRunning: map[string]bool{},
			entitlementService:         entitlement.StaticService{Grants: entitlement.BetaGrants()},
		}
		defaultNexus.quotaMonitor = NewQuotaMonitorService(DefaultQuotaDropMonitor(), events.DefaultBus())
		if st != nil {
			events.DefaultBus().SetRecorder(func(e events.Event) {
				projectID, _ := e.Data["project_id"].(string)
				agentID, _ := e.Data["agent_id"].(string)
				_, _ = st.RecordEventMetadata(store.EventMetadata{
					ID:            e.ID,
					AgentID:       agentID,
					ProjectID:     projectID,
					CorrelationID: e.CorrelationID,
					Kind:          string(e.Type),
					Timestamp:     e.Timestamp,
					Summary:       e.Summary,
					AccountScope:  e.AccountScope,
				})
			})
			_ = defaultNexus.RecoverMissionRuns(context.Background())
			defaultNexus.StartScheduleLoop()
			// Purge expired events at startup (90d default).
			go defaultNexus.purgeExpiredEvents()
		}
	})
	return defaultNexus
}

// SetEntitlementService replaces the capability resolver. The beta product uses
// a local StaticService; future signed/server-backed grants plug into the same
// boundary without changing feature call sites.
func (n *Nexus) SetEntitlementService(service entitlement.Service) {
	if n == nil {
		return
	}
	n.mu.Lock()
	n.entitlementService = service
	n.mu.Unlock()
}

// RequireCapability is the server-side product gate. UI visibility is never
// treated as authorization. The installation identity is privacy-safe and is
// used only as the local entitlement subject until account identity exists.
func (n *Nexus) RequireCapability(ctx context.Context, capability entitlement.Capability) error {
	if n == nil {
		return fmt.Errorf("entitlement service unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	n.mu.RLock()
	service := n.entitlementService
	subject := n.entitlementSubjectID
	n.mu.RUnlock()
	if service == nil {
		service = entitlement.StaticService{Grants: entitlement.BetaGrants()}
	}
	if strings.TrimSpace(subject) == "" {
		dataDir, err := config.DataDir()
		if err != nil {
			return fmt.Errorf("resolve entitlement data dir: %w", err)
		}
		subject, err = device.LoadOrCreate(dataDir)
		if err != nil {
			return fmt.Errorf("resolve entitlement installation identity: %w", err)
		}
		n.mu.Lock()
		if n.entitlementSubjectID == "" {
			n.entitlementSubjectID = subject
		} else {
			subject = n.entitlementSubjectID
		}
		n.mu.Unlock()
	}
	grants, err := service.GrantsFor(ctx, subject)
	if err != nil {
		return fmt.Errorf("resolve entitlement: %w", err)
	}
	if !entitlement.Require(grants, capability) {
		return fmt.Errorf("capability %s is not granted", capability)
	}
	return nil
}

func (n *Nexus) runtimeRegistry() *registry.Registry {
	if n != nil && n.runtimeReg != nil {
		return n.runtimeReg
	}
	return registry.DefaultRegistry()
}

// RuntimeRegistry exposes the Nexus-owned runtime registry to transport
// adapters without forcing them back to the process-wide singleton.
func (n *Nexus) RuntimeRegistry() *registry.Registry { return n.runtimeRegistry() }

// controlDrivers returns the Nexus-owned provider control registry. The
// fallback preserves compatibility for tests and legacy constructors that
// still build a Nexus value directly.
func (n *Nexus) controlDrivers() *driver.Registry {
	if n != nil && n.drivers != nil {
		return n.drivers
	}
	return driver.DefaultRegistry()
}

// ResetDefaultForTest releases the process-wide singleton between tests that
// replace the configured data directory. Production code should use Default.
func ResetDefaultForTest() {
	if defaultNexus != nil {
		defaultNexus.Shutdown()
		defaultNexus = nil
	}
	nexusOnce = sync.Once{}
}

// purgeExpiredEvents removes activity events older than the retention period.
// It runs once at startup and is safe to call from a background goroutine.
func (nx *Nexus) purgeExpiredEvents() {
	if nx == nil {
		return
	}
	st, err := nx.OpenProject()
	if err != nil {
		return
	}
	policy, err := GetConductorPolicy(st)
	if err != nil || policy == nil || policy.RetentionDays == nil {
		return
	}
	purged, err := st.PurgeExpiredEventsSince(time.Now().UTC(), *policy.RetentionDays)
	if err == nil && purged > 0 {
		slog.Debug("nexus conductor: purged expired events", "count", purged)
	}
}

// StartQuotaMonitor starts the process-resident quota service exactly once.
func (n *Nexus) StartQuotaMonitor(ctx context.Context) {
	n.mu.Lock()
	if n.quotaMonitor == nil {
		n.quotaMonitor = NewQuotaMonitorService(DefaultQuotaDropMonitor(), events.DefaultBus())
	}
	service := n.quotaMonitor
	n.mu.Unlock()
	service.Start(ctx)
}

// StopQuotaMonitor stops the resident service without affecting runtimes.
func (n *Nexus) StopQuotaMonitor() {
	n.mu.RLock()
	service := n.quotaMonitor
	n.mu.RUnlock()
	if service != nil {
		service.Stop()
	}
}

// Shutdown gracefully stops all background services and releases resources.
// It must be called once during process exit to avoid leaked SQLite handles
// and orphaned goroutines.
func (n *Nexus) Shutdown() {
	n.StopQuotaMonitor()
	n.StopCapacityMonitor()

	// Stop the schedule loop goroutine.
	if n.scheduleCancel != nil {
		n.scheduleCancel()
	}

	// Cancel all active mission workers.
	n.workersMu.Lock()
	workers := make([]*missionWorker, 0, len(n.workers))
	for _, w := range n.workers {
		workers = append(workers, w)
	}
	n.workersMu.Unlock()
	for _, w := range workers {
		w.cancel()
	}

	// Close the SQLite store last, after all goroutines that reference it
	// have been asked to stop.
	if n.st != nil {
		_ = n.st.Close()
	}
}

// OpenStore opens the SQLite store at <DataDir>/nexus.db using the canonical
// cross-platform data directory authority (P1 canonical DataDir).
func OpenStore() (*store.Store, error) {
	dir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
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

// AskAgentResult confirms that a prompt was submitted to the requested
// persistent Agent identity. AskAgent never creates another Agent.
type AskAgentResult struct {
	AgentID       string `json:"agent_id"`
	RuntimeID     string `json:"runtime_id"`
	ExecutionRole string `json:"execution_role,omitempty"`
	Started       bool   `json:"started"`
	Accepted      bool   `json:"accepted"`
}

func submitPromptToRuntime(runtimeID, prompt string) error {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.SubmitPrompt(prompt)
}

// AskAgentOptions controls how AskAgent submits a prompt to a live Agent.
type AskAgentOptions struct {
	StartIfNeeded bool
	// ExecutionRoleID is a transient turn override. When empty, the runtime
	// binding is used. It never mutates AgentSpec or AgentRevision.
	ExecutionRoleID string
	// CompileSpecialization wraps the prompt in the WorkPackage envelope.
	// It is correct for headless/Mission execution; an interactive runtime
	// already carries the persona in-session, and re-sending the envelope
	// each turn floods the provider TUI.
	CompileSpecialization bool
}

// AskAgent submits work to an existing persistent Agent. A stopped Agent may
// be started first when explicitly requested, but Agent identity is preserved.
// Specialization compilation remains on by default for headless callers.
func (n *Nexus) AskAgent(ctx context.Context, agentID, prompt string, startIfNeeded bool) (*AskAgentResult, error) {
	return n.AskAgentWithOptions(ctx, agentID, prompt, AskAgentOptions{
		StartIfNeeded:         startIfNeeded,
		CompileSpecialization: true,
	})
}

// AskAgentWithOptions submits work with explicit control over specialization
// compilation. Interactive Lead Agents with a live runtime never receive the
// WorkPackage envelope, even when CompileSpecialization is requested.
func (n *Nexus) AskAgentWithOptions(ctx context.Context, agentID, prompt string, opts AskAgentOptions) (*AskAgentResult, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}
	agentCfg, cfgErr := currentAgentConfig(st, agent)
	if cfgErr != nil {
		return nil, cfgErr
	}

	runtimeID := ""
	started := false
	runtimeLive := false
	if gen, genErr := st.CurrentGeneration(agentID); genErr == nil && n.runtimeAlive(gen.RuntimeID) {
		runtimeID = gen.RuntimeID
		runtimeLive = true
	} else {
		if !opts.StartIfNeeded {
			return nil, fmt.Errorf("agent %s has no active runtime; use Start & Ask", agentID)
		}
		state, _ := n.EffectiveAgentState(agentID)
		var sess *registry.RuntimeSession
		if state == store.AgentRecoverable {
			sess, err = n.RecoverAgent(ctx, agentID)
		} else {
			provider, profile, resolveErr := n.ResolveStartParams(agentID, "", "")
			if resolveErr != nil {
				return nil, resolveErr
			}
			sess, err = n.StartAgent(ctx, agentID, provider, profile)
		}
		if err != nil {
			return nil, err
		}
		runtimeID = sess.RuntimeID
		started = true
		runtimeLive = true
	}

	executionRoleID := strings.TrimSpace(opts.ExecutionRoleID)
	executionRoleEffective := executionRoleID
	roleShouldAffectPrompt := executionRoleID != ""
	if executionRoleID == "" {
		if binding, bindErr := st.RuntimeExecutionRoleByRuntimeID(runtimeID); bindErr == nil {
			executionRoleID = binding.RoleID
			executionRoleEffective = binding.RoleID
			roleShouldAffectPrompt = true
		} else if bindErr == store.ErrNotFound {
			executionRoleEffective = "build"
		} else {
			return nil, fmt.Errorf("resolve runtime execution role: %w", bindErr)
		}
	}
	var executionRole *intelligence.ExecutionRoleContext
	if executionRoleID != "" {
		role, ok := ExecutionRoleForID(executionRoleID)
		if !ok {
			return nil, fmt.Errorf("unknown execution role %q", executionRoleID)
		}
		if roleShouldAffectPrompt && role.ID != "build" {
			executionRole = role.IntelligenceContext()
		}
	}

	compile := opts.CompileSpecialization
	// Interactive Lead sessions already host the persona in the live TUI;
	// never inject the full WorkPackage envelope into that path. A compact
	// execution-role overlay is still safe and intentional.
	if runtimeLive && strings.EqualFold(strings.TrimSpace(agent.Role), interactiveLeadRole) {
		compile = false
	}

	submit := n.submitPrompt
	if submit == nil {
		submit = submitPromptToRuntime
	}
	if compile && (agentCfg.AgentSpec.HasCustomBehavior() || executionRole != nil) {
		compiled, compileErr := intelligence.NewNexusEngine(nil).CompileExecutionContext(ctx, intelligence.ExecutionContextRequest{
			Agent:         agentCfg.AgentSpec,
			ExecutionRole: executionRole,
			Task:          intelligence.WorkPackageContext{Title: "Direct Agent request", Goal: prompt, Role: agentCfg.AgentSpec.Role},
		})
		if compileErr != nil {
			return nil, fmt.Errorf("compile agent specialization: %w", compileErr)
		}
		prompt = compiled.SystemInstructions + "\n\n" + compiled.TaskInstructions
	} else if executionRole != nil {
		prompt = renderExecutionRoleOverlay(executionRole, prompt)
	}
	if err := submit(runtimeID, prompt); err != nil {
		return nil, fmt.Errorf("submit prompt to existing agent: %w", err)
	}
	return &AskAgentResult{AgentID: agentID, RuntimeID: runtimeID, ExecutionRole: executionRoleEffective, Started: started, Accepted: true}, nil
}

func renderExecutionRoleOverlay(role *intelligence.ExecutionRoleContext, prompt string) string {
	if role == nil || strings.TrimSpace(role.ID) == "" {
		return prompt
	}
	return fmt.Sprintf("[Nexus execution role]\nID: %s\nRole: %s\nAccess intent: %s\nScope: %s\nRequired capabilities: %s\nBoundary: this role does not expand task/runtime authorization.\n\n%s",
		role.ID,
		role.Role,
		role.AccessIntent,
		role.Scope,
		strings.Join(role.RequiredCapabilities, ", "),
		prompt,
	)
}

// StartProjectShell launches an ordinary shell rooted at the Project canonical
// path. It does not create an Agent, ConfigRevision, lineage or RuntimeGeneration.
func (n *Nexus) StartProjectShell(ctx context.Context, projectID string) (*registry.RuntimeSession, error) {
	n.StartQuotaMonitor(ctx)
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(project.CanonicalPath) == "" {
		return nil, fmt.Errorf("project canonical path is empty")
	}
	info, err := os.Stat(project.CanonicalPath)
	if err != nil {
		return nil, fmt.Errorf("project canonical directory %q does not exist: %w", project.CanonicalPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project canonical path %q is not a directory", project.CanonicalPath)
	}
	return n.launcher.Launch(ctx, launcher.LaunchOptions{
		Title:       "Shell · " + project.Name,
		ProviderID:  "shell",
		ProfileID:   "local",
		Workspace:   project.CanonicalPath,
		ProjectID:   project.ID,
		ProjectName: project.Name,
	})
}

// StartAgent launches a supervised runtime for an agent, records a config
// revision and a runtime generation, and updates agent status. The runtime
// starts in the agent's project canonical path (P0-1). Empty provider is
// rejected (P0-6 — no silent fake fallback in production).
func (n *Nexus) StartAgent(ctx context.Context, agentID, provider, profile string) (*registry.RuntimeSession, error) {
	n.StartQuotaMonitor(ctx)
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
			agentCfg = NormalizeAgentSpec(agent, agentCfg)
		}
	}
	agentCfg = NormalizeAgentSpec(agent, agentCfg)
	configuredProvider := agentCfg.Provider
	configuredProfile := agentCfg.Profile
	agentCfg.Provider = provider
	agentCfg.Profile = profile
	if agentCfg.Profile == "" {
		agentCfg.Profile = "default"
	}
	accountScope := registeredAccountScope(provider, agentCfg.Profile)

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
	}
	continuityLaunch, err := n.continuityForNextGeneration(ctx, agentCfg, previousGen)
	if err != nil {
		return nil, fmt.Errorf("resolve start continuity: %w", err)
	}
	executionWorkspace, err := n.resolveExecutionWorkspace(ctx, proj, agent, agentCfg)
	if err != nil {
		return nil, fmt.Errorf("resolve execution workspace: %w", err)
	}
	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		AgentID:           agentID,
		RuntimeID:         "runtime_" + ids.NewRuntimeID(),
		ProjectID:         proj.ID,
		ProjectName:       proj.Name,
		ProviderID:        provider,
		ProfileID:         agentCfg.Profile,
		AccountScope:      accountScope,
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

	// Launcher already waits for the endpoint; re-ping so Start never returns a
	// session that looks live while the named pipe / socket is gone (Windows P0).
	if sess.Transport != "mock" {
		if err := protocol.WaitForEndpoint(ctx, sess.RuntimeID, 8*time.Second); err != nil || !n.runtimeHostReachable(sess.RuntimeID) {
			n.stopRuntime(sess.RuntimeID)
			if err == nil {
				err = fmt.Errorf("started runtime host did not accept attach")
			}
			return nil, fmt.Errorf("start agent runtime: %w", err)
		}
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
	if err := st.UpdateAgent(agent); err != nil {
		n.stopRuntime(sess.RuntimeID)
		return nil, fmt.Errorf("persist agent state after launch: %w", err)
	}

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

// registeredAccountScope resolves the persisted authenticated identity before
// a runtime is created. Failure is intentionally fail-closed: an unverified
// runtime can still launch, but it cannot consume account-owned quota data.
func registeredAccountScope(provider, profile string) model.AccountScope {
	p, err := profilepkg.Get(provider, profile)
	if err != nil {
		return model.AccountScope{}
	}
	return p.AccountScope
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
	if err := st.UpdateAgent(agent); err != nil {
		// TODO: emit alert metric
		_ = err
	}
	n.notifyAgentState(agentID, "STOPPING")

	gen, gerr := st.CurrentGeneration(agentID)
	if gerr != nil || gen.RuntimeID == "" {
		// No runtime generation — agent is already stopped.
		agent.Status = store.AgentStopped
		if err := st.UpdateAgent(agent); err != nil {
			// TODO: emit alert metric
			_ = err
		}
		n.notifyAgentState(agentID, "STOPPED")
		return nil
	}

	// Phase 2: send graceful stop and wait for process termination.
	if !n.runtimeAlive(gen.RuntimeID) {
		// Runtime already dead — proceed to finalize.
		agent.Status = store.AgentStopped
		if err := st.UpdateAgent(agent); err != nil {
			// TODO: emit alert metric
			_ = err
		}
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
			// Launcher.Stop is the sole process-termination authority and already
			// performs generation-validated force-stop after its grace period.
			// Never add a second raw-PID kill path here: if shutdown still has
			// not converged, fail closed and preserve the evidence for recovery.
			_ = os.Remove(protocol.EndpointPath(gen.RuntimeID))
			_ = n.runtimeRegistry().UpdateState(gen.RuntimeID, registry.StateFailed)
			agent.Status = store.AgentFailed
			agent.ContinuityStatus = "STOP_TIMEOUT"
			if err := st.UpdateAgent(agent); err != nil {
				// TODO: emit alert metric
				_ = err
			}
			n.notifyAgentState(agentID, "FAILED")
			return fmt.Errorf("stop timeout for runtime %s (agent marked FAILED)", gen.RuntimeID)
		case <-ticker.C:
			if !n.runtimeAlive(gen.RuntimeID) {
				// Phase 4: process confirmed dead — persist STOPPED.
				stopped := time.Now().UTC()
				_ = st.StopGeneration(gen.ID, stopped)
				agent.Status = store.AgentStopped
				if err := st.UpdateAgent(agent); err != nil {
					// TODO: emit alert metric
					_ = err
				}
				n.notifyAgentState(agentID, "STOPPED")
				return nil
			}
		}
	}
}

func (n *Nexus) stopRuntime(runtimeID string) {
	_ = n.launcher.Stop(runtimeID)
}

// runtimeAlive reports whether a runtime is currently usable: the process is
// alive (with host-generation protection against PID recycle) AND the control
// host still accepts a ping. A zombie PID or a dead host counts as not alive
// so Recover/Delete/Start can proceed. Mock transports (unit tests) skip the
// host ping because they never open a SessionHost endpoint.
func (n *Nexus) runtimeAlive(runtimeID string) bool {
	if runtimeID == "" {
		return false
	}
	sess, ok := n.runtimeRegistry().Get(runtimeID)
	if !ok {
		return false
	}
	// Registry state is authoritative for lifecycle. A stale/failed/stopped
	// generation must never be revived merely because an old host PID or pipe
	// still happens to exist.
	if !sess.IsActive() {
		return false
	}
	// HostGeneration belongs to the SessionHost, not the provider child.
	// Reuse the registry's canonical host-identity check so a surviving child
	// process cannot make a dead/recycled host generation look healthy.
	if !sess.HostLive() {
		return false
	}
	if sess.Transport == "mock" {
		return true
	}
	return n.runtimeHostReachable(runtimeID)
}

// runtimeHostReachable probes the SessionHost control endpoint with a short ping.
func (n *Nexus) runtimeHostReachable(runtimeID string) bool {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return false
	}
	defer client.Close()
	_ = client.RawConn().SetDeadline(time.Now().Add(800 * time.Millisecond))
	if err := client.Ping(); err != nil {
		return false
	}
	return true
}

// stopStaleRuntime best-effort stops a registry entry whose process or host is
// no longer usable, so a subsequent Recover/Start can launch a new generation.
func (n *Nexus) stopStaleRuntime(runtimeID string) {
	if runtimeID == "" {
		return
	}
	_ = n.launcher.Stop(runtimeID)
	_ = n.runtimeRegistry().Delete(runtimeID)
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
	if gerr == nil && gen.RuntimeID != "" {
		if n.runtimeAlive(gen.RuntimeID) {
			// Idempotent recover: return the live session so clients can rebind
			// the terminal without stop+start or an opaque 409.
			if sess, ok := n.runtimeRegistry().Get(gen.RuntimeID); ok {
				cp := sess
				return &cp, nil
			}
		}
		// Registry may still list a zombie/unreachable runtime — clear it so
		// Launch can register a new generation.
		if _, ok := n.runtimeRegistry().Get(gen.RuntimeID); ok {
			n.stopStaleRuntime(gen.RuntimeID)
		}
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
	continuity := store.ContinuityNewSession
	providerSessionID := ""

	// When recovering after a process termination / host reboot, start a clean
	// session unless the agent was explicitly configured with ContinuityPolicy == "native".
	if strings.EqualFold(agentCfg.ContinuityPolicy, "native") && sessionID != "" {
		if d, derr := n.controlDrivers().Get(provider); derr == nil {
			prof := model.Profile{Name: profile, Provider: provider}
			if can, _ := d.CanResume(ctx, prof, sessionID); can {
				if ra, rerr := d.BuildResumeArgs(ctx, prof, sessionID); rerr == nil {
					args = ra
					continuity = store.ContinuityNativeResumeUnverified
					providerSessionID = sessionID
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
		ProjectID:         proj.ID,
		ProjectName:       proj.Name,
		ProviderID:        provider,
		ProfileID:         profile,
		ProviderSessionID: providerSessionID,
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

	// Do not mark WORKING until the host accepts attach/ping (skip for mock transports).
	if sess.Transport != "mock" {
		if err := protocol.WaitForEndpoint(ctx, sess.RuntimeID, 8*time.Second); err != nil || !n.runtimeHostReachable(sess.RuntimeID) {
			n.stopRuntime(sess.RuntimeID)
			agent.Status = store.AgentRecoverable
			_ = st.UpdateAgent(agent)
			n.notifyAgentState(agentID, "RECOVERABLE")
			if err == nil {
				err = fmt.Errorf("recovered runtime host did not accept attach")
			}
			return nil, fmt.Errorf("recover agent runtime: %w", err)
		}
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
// deletion is refused to prevent orphaned provider processes. Zombie / host-
// unreachable runtimes are stopped first, then deleted.
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
	if gerr == nil && gen.RuntimeID != "" {
		if n.runtimeAlive(gen.RuntimeID) {
			return fmt.Errorf("cannot delete agent %q: runtime %s is live (stop the agent first)", agent.Name, gen.RuntimeID)
		}
		if _, ok := n.runtimeRegistry().Get(gen.RuntimeID); ok {
			n.stopStaleRuntime(gen.RuntimeID)
		}
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
