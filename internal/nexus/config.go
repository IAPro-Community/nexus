package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// AgentConfig is the persistent, revisioned configuration for an Agent (Gate 3).
// It is stored as a JSON blob in agent_revisions.config.
type AgentConfig struct {
	Provider         string                 `json:"provider"`
	Profile          string                 `json:"profile"`
	Model            string                 `json:"model,omitempty"`
	ModelCandidates  []ModelCandidate       `json:"model_candidates,omitempty"`
	Options          map[string]any         `json:"options,omitempty"`
	Workspace        string                 `json:"workspace,omitempty"`
	Isolation        string                 `json:"isolation,omitempty"`         // "project" | "global" | "none"
	MaestroMode      string                 `json:"maestro_mode,omitempty"`      // "OFF" | "ASSIST" | "ORCHESTRATE"
	ContinuityPolicy string                 `json:"continuity_policy,omitempty"` // "auto" | "native" | "new_session"
	Environment      map[string]string      `json:"environment,omitempty"`
	Allocation       *AllocationPolicy      `json:"allocation,omitempty"`
	AgentSpec        intelligence.AgentSpec `json:"agent_spec,omitempty"`
}

// NormalizeAgentSpec preserves legacy agents whose only specialization was
// stored in store.Agent.Role. It never invents instructions or capabilities.
func NormalizeAgentSpec(agent store.Agent, cfg AgentConfig) AgentConfig {
	if strings.TrimSpace(cfg.AgentSpec.Role) == "" {
		cfg.AgentSpec.Role = strings.TrimSpace(agent.Role)
	}
	return cfg
}

// AllocationPolicy controls how provider resources are allocated to an agent.
type AllocationPolicy struct {
	PreferProvider  string `json:"prefer_provider,omitempty"`
	MaxConcurrent   int    `json:"max_concurrent,omitempty"`
	QuotaPreserve   bool   `json:"quota_preserve,omitempty"`
	CooldownSeconds int    `json:"cooldown_seconds,omitempty"`
}

// OptionSpec describes a single configuration option owned by a provider driver.
// The UI renders these generically — no provider-specific React conditionals.
type OptionSpec struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Description     string   `json:"description,omitempty"`
	Type            string   `json:"type"` // "string" | "number" | "boolean" | "select" | "multiselect" | "text"
	Default         any      `json:"default,omitempty"`
	Choices         []Choice `json:"choices,omitempty"`
	Validation      string   `json:"validation,omitempty"` // regex or rule name
	Sensitive       bool     `json:"sensitive,omitempty"`
	Advanced        bool     `json:"advanced,omitempty"`
	RestartRequired bool     `json:"restart_required,omitempty"` // continuity impact
	Group           string   `json:"group,omitempty"`            // UI section grouping
}

// Choice is a labeled value for select/multiselect options.
type Choice struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ImpactMode describes what happens when a config is applied.
type ImpactMode string

const (
	ImpactLiveSameRuntime ImpactMode = "LIVE_SAME_RUNTIME" // No restart needed
	ImpactRestartRuntime  ImpactMode = "RESTART_RUNTIME"   // New generation, same agent
	ImpactNewSession      ImpactMode = "NEW_SESSION"       // New provider session
)

// ConfigImpact describes the result of comparing current vs proposed config.
type ConfigImpact struct {
	Mode            ImpactMode   `json:"mode"`
	ChangedFields   []string     `json:"changed_fields"`
	RequiresRestart bool         `json:"requires_restart"`
	RequiresNewSess bool         `json:"requires_new_session"`
	CurrentConfig   *AgentConfig `json:"current_config,omitempty"`
	ProposedConfig  *AgentConfig `json:"proposed_config,omitempty"`
	Warnings        []string     `json:"warnings,omitempty"`
}

// ParseAgentConfig deserializes a JSON config string into AgentConfig.
func ParseAgentConfig(raw string) (AgentConfig, error) {
	var cfg AgentConfig
	if raw == "" || raw == "{}" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(raw), &cfg)
	return cfg, err
}

// ConfigJSON serializes AgentConfig to JSON string.
func (c AgentConfig) ConfigJSON() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// AnalyzeImpact compares current and proposed agent configs and returns the
// impact mode and changed fields. This drives the Safe Apply preview.
func AnalyzeImpact(current, proposed AgentConfig) ConfigImpact {
	impact := ConfigImpact{
		CurrentConfig:  &current,
		ProposedConfig: &proposed,
		ChangedFields:  []string{},
		Warnings:       []string{},
	}

	if reflect.DeepEqual(current, proposed) {
		return impact // no changes
	}

	// Detect which fields changed.
	if current.Provider != proposed.Provider {
		impact.ChangedFields = append(impact.ChangedFields, "provider")
		impact.RequiresNewSess = true
	}
	if current.Profile != proposed.Profile {
		impact.ChangedFields = append(impact.ChangedFields, "profile")
		impact.RequiresNewSess = true
	}
	if current.Model != proposed.Model {
		impact.ChangedFields = append(impact.ChangedFields, "model")
		impact.RequiresRestart = true
	}
	if current.Workspace != proposed.Workspace {
		impact.ChangedFields = append(impact.ChangedFields, "workspace")
		impact.RequiresRestart = true
	}
	if current.Isolation != proposed.Isolation {
		impact.ChangedFields = append(impact.ChangedFields, "isolation")
		impact.RequiresRestart = true
	}
	if current.MaestroMode != proposed.MaestroMode {
		impact.ChangedFields = append(impact.ChangedFields, "maestro_mode")
		// Maestro mode change is live — no restart needed.
	}
	if current.ContinuityPolicy != proposed.ContinuityPolicy {
		impact.ChangedFields = append(impact.ChangedFields, "continuity_policy")
	}
	if strings.EqualFold(proposed.ContinuityPolicy, "new_session") {
		impact.RequiresNewSess = true
		impact.RequiresRestart = true
	}
	if !reflect.DeepEqual(current.Options, proposed.Options) {
		impact.ChangedFields = append(impact.ChangedFields, "options")
		impact.RequiresRestart = true

		currentMode := ""
		if current.Options != nil {
			if m, ok := current.Options["mode"].(string); ok {
				currentMode = m
			}
		}
		proposedMode := ""
		if proposed.Options != nil {
			if m, ok := proposed.Options["mode"].(string); ok {
				proposedMode = m
			}
		}
		if currentMode != proposedMode {
			impact.RequiresNewSess = true
		}
	}
	if !reflect.DeepEqual(current.Environment, proposed.Environment) {
		impact.ChangedFields = append(impact.ChangedFields, "environment")
		impact.RequiresRestart = true
	}
	if !reflect.DeepEqual(current.AgentSpec, proposed.AgentSpec) {
		impact.ChangedFields = append(impact.ChangedFields, "agent_spec")
		impact.RequiresRestart = true
		impact.RequiresNewSess = true
	}
	if !reflect.DeepEqual(current.Allocation, proposed.Allocation) {
		impact.ChangedFields = append(impact.ChangedFields, "allocation")
	}

	// Determine impact mode.
	switch {
	case impact.RequiresNewSess:
		impact.Mode = ImpactNewSession
	case impact.RequiresRestart:
		impact.Mode = ImpactRestartRuntime
	default:
		impact.Mode = ImpactLiveSameRuntime
	}

	return impact
}

// SafeApply applies a config change transactionally: validates candidate config,
// analyzes impact, stops previous runtime gracefully if restart is needed,
// launches generation N+1 without creating duplicate revisions, commits
// revision atomically, and provides automatic rollback on launch failure.
func (n *Nexus) SafeApply(ctx context.Context, agentID string, proposed AgentConfig) (*ConfigImpact, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}

	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}

	// Resolve project canonical workspace (§14).
	proj, perr := st.GetProject(agent.ProjectID)
	if perr != nil {
		return nil, fmt.Errorf("resolve agent project: %w", perr)
	}

	// Load current config from the agent's current revision.
	var current AgentConfig
	oldRevID := agent.CurrentRevisionID
	if agent.CurrentRevisionID != "" {
		rev, rerr := st.GetRevision(agent.CurrentRevisionID)
		if rerr == nil {
			current, _ = ParseAgentConfig(rev.Config)
			current = NormalizeAgentSpec(agent, current)
		}
	}
	proposed = NormalizeAgentSpec(agent, proposed)
	if err := validateAgentModeConfiguration(proposed); err != nil {
		return nil, err
	}

	// Analyze impact.
	impact := AnalyzeImpact(current, proposed)
	if len(impact.ChangedFields) == 0 {
		return &impact, nil // no-op
	}

	// A stopped Agent may be configured before a concrete provider account is
	// selected. The resource picker persists the exact provider/profile before
	// the first start. Running Agents still require a concrete provider.
	if proposed.Profile == "" && proposed.Provider != "" {
		proposed.Profile = "default"
	}

	// Resolve runtime state before deciding whether a restart impact should be
	// executed immediately. Configuration is durable state; applying it to a
	// STOPPED agent must never have the surprising side effect of starting a
	// provider process. The impact still describes what the next live apply
	// would require, but the candidate revision is simply persisted for the
	// next explicit StartAgent call.
	oldGen, gerr := st.CurrentGeneration(agentID)
	wasAlive := gerr == nil && oldGen.RuntimeID != "" && n.runtimeAlive(oldGen.RuntimeID)
	if proposed.Provider == "" && wasAlive {
		return nil, fmt.Errorf("provider is required while the agent is running; select a resource first")
	}
	if !wasAlive && agent.Status == store.AgentStopped {
		rev, err := st.AddRevision(agentID, proposed.ConfigJSON())
		if err != nil {
			return nil, fmt.Errorf("create config revision: %w", err)
		}
		agent.CurrentRevisionID = rev.ID
		agent.Status = store.AgentStopped
		if err := st.UpdateAgent(agent); err != nil {
			return nil, fmt.Errorf("update stopped agent revision: %w", err)
		}
		impact.Warnings = append(impact.Warnings, "configuration persisted; agent remains stopped until explicitly started")
		return &impact, nil
	}

	// Case 1: Live change (e.g. MaestroMode, Allocation Policy) — no restart needed.
	if !impact.RequiresRestart && !impact.RequiresNewSess {
		rev, err := st.AddRevision(agentID, proposed.ConfigJSON())
		if err != nil {
			return nil, fmt.Errorf("create config revision: %w", err)
		}
		agent.CurrentRevisionID = rev.ID
		if err := st.UpdateAgent(agent); err != nil {
			return nil, fmt.Errorf("update agent revision: %w", err)
		}
		return &impact, nil
	}

	// Validate launch-only fields and workspace before stopping the current runtime.
	// This prevents a known-invalid candidate from taking a healthy Agent offline.
	if _, err := driver.ApplyLaunchConfiguration(proposed.Provider, proposed.Model, proposed.Options, nil); err != nil {
		return nil, fmt.Errorf("validate candidate launch configuration: %w", err)
	}
	executionWorkspace, err := n.resolveExecutionWorkspace(ctx, proj, agent, proposed)
	if err != nil {
		return nil, fmt.Errorf("resolve candidate execution workspace: %w", err)
	}
	var previousGen *store.RuntimeGeneration
	if gerr == nil {
		copyGen := oldGen
		previousGen = &copyGen
	}
	launchDecision, err := n.continuityForNextGeneration(ctx, proposed, previousGen)
	if err != nil {
		return nil, fmt.Errorf("validate candidate continuity: %w", err)
	}
	if impact.RequiresNewSess || strings.EqualFold(proposed.ContinuityPolicy, "new_session") {
		launchDecision = continuityLaunch{Status: store.ContinuityNewSession}
	}

	// Case 2: Restart required — execute ReconfigureTransaction (A5).
	// Step 1: Freeze writer & set RECONFIGURING state.
	agent.Status = store.AgentReconfig
	_ = st.UpdateAgent(agent)
	n.notifyAgentState(agentID, "RECONFIGURING")

	// Step 2: Stop old runtime if it was alive.
	if wasAlive {
		if err := n.StopAgent(ctx, agentID); err != nil {
			// Rollback to previous state
			agent.Status = store.AgentWorking
			_ = st.UpdateAgent(agent)
			n.notifyAgentState(agentID, "WORKING")
			return nil, fmt.Errorf("stop current runtime for config apply: %w", err)
		}
	}

	// Step 3: Create single immutable config revision (N+1).
	rev, err := st.AddRevision(agentID, proposed.ConfigJSON())
	if err != nil {
		agent.Status = store.AgentRecoverable
		_ = st.UpdateAgent(agent)
		return nil, fmt.Errorf("create config revision: %w", err)
	}

	// Step 4: Launch generation N+1 with the validated candidate.
	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		AgentID:           agent.ID,
		ProjectID:         proj.ID,
		ProjectName:       proj.Name,
		ProviderID:        proposed.Provider,
		ProfileID:         proposed.Profile,
		ProviderSessionID: launchDecision.ProviderSessionID,
		Args:              launchDecision.Args,
		Workspace:         executionWorkspace,
		Model:             proposed.Model,
		Environment:       proposed.Environment,
		Isolation:         proposed.Isolation,
		Options:           proposed.Options,
	})
	if err != nil {
		// Candidate launch failed. Restore the last known-good configuration when
		// the previous runtime had been live; only degrade to RECOVERABLE if the
		// rollback itself cannot be launched.
		rollbackErr := error(nil)
		if wasAlive {
			rollbackErr = n.restorePreviousRuntime(ctx, st, proj, agent, current, oldRevID, oldGen)
		}
		if rollbackErr == nil && wasAlive {
			return nil, fmt.Errorf("candidate config launch failed; previous runtime restored: %w", err)
		}
		agent.CurrentRevisionID = oldRevID
		agent.Status = store.AgentRecoverable
		_ = st.UpdateAgent(agent)
		n.notifyAgentState(agentID, "RECOVERABLE")
		if rollbackErr != nil {
			return nil, fmt.Errorf("candidate config launch failed: %v; rollback failed: %w", err, rollbackErr)
		}
		return nil, fmt.Errorf("launch runtime generation for config apply: %w", err)
	}

	// Step 5: Commit runtime generation N+1.
	newGen := store.RuntimeGeneration{
		AgentID:         agentID,
		RevisionID:      rev.ID,
		RuntimeID:       sess.RuntimeID,
		Provider:        proposed.Provider,
		Profile:         proposed.Profile,
		ProviderSession: sess.ProviderSessionID,
		Continuity:      launchDecision.Status,
		StartedAt:       time.Now().UTC(),
		State:           "RUNNING",
	}
	if _, err := st.AddGeneration(newGen); err != nil {
		n.stopRuntime(sess.RuntimeID)
		agent.CurrentRevisionID = oldRevID
		agent.Status = store.AgentRecoverable
		_ = st.UpdateAgent(agent)
		return nil, fmt.Errorf("commit runtime generation: %w", err)
	}

	// Step 6: Atomically switch Agent current revision and status to WORKING.
	agent.Status = store.AgentWorking
	agent.CurrentRevisionID = rev.ID
	agent.ContinuityStatus = launchDecision.Status
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	_ = st.UpdateAgent(agent)

	// Step 7: Record lineage edge if provider or session changed.
	if wasAlive && oldGen.RuntimeID != "" {
		_ = st.AddLineage(store.LineageEntry{
			AgentID:       agentID,
			Relation:      "RECONFIGURE_RESTART",
			SourceRuntime: oldGen.RuntimeID,
			SourceSession: oldGen.ProviderSession,
			TargetRuntime: sess.RuntimeID,
			TargetSession: sess.ProviderSessionID,
			CreatedAt:     time.Now().UTC(),
		})
	}

	// Step 8: Notify terminal broker and observers.
	oldRuntimeID := ""
	if wasAlive {
		oldRuntimeID = oldGen.RuntimeID
	}
	n.notifyRuntimeChanged(agentID, oldRuntimeID, sess.RuntimeID, proposed.Provider, proposed.Profile, launchDecision.Status)
	n.notifyAgentState(agentID, "WORKING")
	n.notifyContinuity(agentID, launchDecision.Status)

	return &impact, nil
}

// restorePreviousRuntime is the compensating half of SafeApply. It attempts to
// relaunch the old immutable config/revision after a candidate launch fails.
func (n *Nexus) restorePreviousRuntime(ctx context.Context, st *store.Store, project store.Project, agent store.Agent, previousCfg AgentConfig, previousRevisionID string, previousGen store.RuntimeGeneration) error {
	if previousCfg.Provider == "" {
		previousCfg.Provider = previousGen.Provider
	}
	if previousCfg.Profile == "" {
		previousCfg.Profile = previousGen.Profile
	}
	workspace, err := n.resolveExecutionWorkspace(ctx, project, agent, previousCfg)
	if err != nil {
		return err
	}
	continuity, err := n.continuityForNextGeneration(ctx, previousCfg, &previousGen)
	if err != nil {
		return err
	}
	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		AgentID:           agent.ID,
		ProjectID:         project.ID,
		ProjectName:       project.Name,
		ProviderID:        previousCfg.Provider,
		ProfileID:         previousCfg.Profile,
		ProviderSessionID: continuity.ProviderSessionID,
		Args:              continuity.Args,
		Workspace:         workspace,
		Model:             previousCfg.Model,
		Environment:       previousCfg.Environment,
		Isolation:         previousCfg.Isolation,
		Options:           previousCfg.Options,
	})
	if err != nil {
		return err
	}
	gen := store.RuntimeGeneration{
		AgentID:         agent.ID,
		RevisionID:      previousRevisionID,
		RuntimeID:       sess.RuntimeID,
		Provider:        previousCfg.Provider,
		Profile:         previousCfg.Profile,
		ProviderSession: sess.ProviderSessionID,
		Continuity:      continuity.Status,
		StartedAt:       time.Now().UTC(),
		State:           "RUNNING",
	}
	if _, err := st.AddGeneration(gen); err != nil {
		n.stopRuntime(sess.RuntimeID)
		return err
	}
	agent.CurrentRevisionID = previousRevisionID
	agent.Status = store.AgentWorking
	agent.ContinuityStatus = continuity.Status
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	if err := st.UpdateAgent(agent); err != nil {
		n.stopRuntime(sess.RuntimeID)
		return err
	}
	n.notifyRuntimeChanged(agent.ID, previousGen.RuntimeID, sess.RuntimeID, previousCfg.Provider, previousCfg.Profile, continuity.Status)
	n.notifyAgentState(agent.ID, store.AgentWorking)
	n.notifyContinuity(agent.ID, continuity.Status)
	return nil
}
