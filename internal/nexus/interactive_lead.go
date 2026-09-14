package nexus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const interactiveLeadRole = "lead-engineer"

// InteractiveLeadBinding carries only the existing project/Agent identity
// needed by the terminal launcher; runtime/provider selection stays in the
// launcher and scheduler layers.
type InteractiveLeadBinding struct {
	AgentID     string
	ProjectID   string
	ProjectName string
}

// PrepareInteractiveLead returns a persistent Agent for a supervised provider
// session. The provider/profile are the lead's current resource preference;
// they are not copied into delegated workstream requirements.
func (n *Nexus) PrepareInteractiveLead(ctx context.Context, projectID, provider, profile string) (store.Agent, error) {
	if err := ctx.Err(); err != nil {
		return store.Agent{}, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return store.Agent{}, err
	}
	agents, err := st.ListAgents(projectID)
	if err != nil {
		return store.Agent{}, err
	}
	for _, candidate := range agents {
		if candidate.Role != interactiveLeadRole {
			continue
		}
		state, stateErr := n.EffectiveAgentState(candidate.ID)
		if stateErr != nil || (state != store.AgentStopped && state != store.AgentRecoverable && state != store.AgentDetached) {
			continue
		}
		if generation, generationErr := st.CurrentGeneration(candidate.ID); generationErr == nil && n.runtimeAlive(generation.RuntimeID) {
			continue
		}
		claimed, claimedOK, claimErr := st.ClaimInteractiveLead(projectID)
		if claimErr != nil {
			return store.Agent{}, claimErr
		}
		if !claimedOK || claimed.ID != candidate.ID {
			continue
		}
		configured, configureErr := n.configureInteractiveLead(st, claimed, provider, profile)
		if configureErr != nil {
			_ = st.ReleaseInteractiveLeadReservation(claimed.ID)
		}
		return configured, configureErr
	}

	lead, err := st.CreateAgent(store.Agent{
		ProjectID: projectID,
		Name:      "Nexus Interactive Lead",
		Role:      interactiveLeadRole,
		Status:    store.AgentStarting,
	})
	if err != nil {
		return store.Agent{}, err
	}
	configured, err := n.configureInteractiveLead(st, lead, provider, profile)
	if err != nil {
		_ = st.ReleaseInteractiveLeadReservation(lead.ID)
		_ = st.DeleteAgent(lead.ID, projectID)
		return store.Agent{}, err
	}
	return configured, nil
}

func (n *Nexus) configureInteractiveLead(st *store.Store, agent store.Agent, provider, profile string) (store.Agent, error) {
	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		return store.Agent{}, err
	}
	cfg.Provider = strings.TrimSpace(provider)
	cfg.Profile = strings.TrimSpace(profile)
	cfg.AgentSpec = interactiveLeadSpec(cfg.AgentSpec)
	if cfg.Isolation == "" {
		cfg.Isolation = "project"
	}
	revision, err := st.AddRevision(agent.ID, cfg.ConfigJSON())
	if err != nil {
		return store.Agent{}, fmt.Errorf("persist interactive lead revision: %w", err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		return store.Agent{}, fmt.Errorf("link interactive lead revision: %w", err)
	}
	return agent, nil
}

func interactiveLeadSpec(existing intelligence.AgentSpec) intelligence.AgentSpec {
	if strings.TrimSpace(existing.Role) == "" {
		existing.Role = interactiveLeadRole
	}
	if len(existing.Responsibilities) == 0 {
		existing.Responsibilities = []string{
			"Manter continuidade da conversa interativa e decidir quando delegar",
			"Integrar e verificar o resultado global da Mission",
		}
	}
	if len(existing.Capabilities) == 0 {
		existing.Capabilities = []string{"orchestration", "integration", "testing", "fullstack"}
	}
	if len(existing.Domains) == 0 {
		existing.Domains = []string{"fullstack", "integration", "verification"}
	}
	existing.VerificationPolicy.RequireEvidence = true
	existing.VerificationPolicy.RequireTests = true
	return existing
}

// BindInteractiveLeadRuntime records the live provider runtime as a normal
// RuntimeGeneration so the LeadAgentID is durable and matcher status excludes
// the foreground Agent from delegated work unless explicitly requested.
func (n *Nexus) BindInteractiveLeadRuntime(ctx context.Context, agentID string, sess *registry.RuntimeSession) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if sess == nil || strings.TrimSpace(agentID) == "" || strings.TrimSpace(sess.RuntimeID) == "" {
		return fmt.Errorf("interactive lead runtime identity is incomplete")
	}
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return err
	}
	revisionID := agent.CurrentRevisionID
	if revisionID == "" {
		return fmt.Errorf("interactive lead %s has no configuration revision", agentID)
	}
	if _, existingErr := st.GenerationByRuntimeID(sess.RuntimeID); existingErr != nil {
		if _, err := st.AddGeneration(store.RuntimeGeneration{
			ID:              "gen_" + ids.NewRuntimeID(),
			AgentID:         agentID,
			RevisionID:      revisionID,
			RuntimeID:       sess.RuntimeID,
			Provider:        sess.ProviderID,
			Profile:         sess.ProfileID,
			ProviderSession: sess.ProviderSessionID,
			Continuity:      store.ContinuityNewSession,
			StartedAt:       time.Now().UTC(),
			State:           "RUNNING",
		}); err != nil {
			return fmt.Errorf("persist interactive lead runtime generation: %w", err)
		}
	}
	agent.Status = store.AgentWorking
	agent.ContinuityStatus = store.ContinuityNewSession
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	if err := st.UpdateAgent(agent); err != nil {
		return err
	}
	return nil
}

// ReleaseInteractiveLeadRuntime closes the durable generation when the
// line-oriented interactive attachment ends. It does not change the Agent's
// provider/profile preference.
func (n *Nexus) ReleaseInteractiveLeadRuntime(ctx context.Context, agentID, runtimeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	generation, err := st.GenerationByRuntimeID(runtimeID)
	if err != nil {
		return err
	}
	if agentID != "" && generation.AgentID != agentID {
		return fmt.Errorf("runtime %s is not owned by Agent %s", runtimeID, agentID)
	}
	if runtime, ok := n.RuntimeRegistry().Get(runtimeID); ok && runtime.State != registry.StateStopped && runtime.State != registry.StateFailed {
		agent, agentErr := st.GetAgent(generation.AgentID, "")
		if agentErr != nil {
			return agentErr
		}
		agent.Status = store.AgentDetached
		return st.UpdateAgent(agent)
	}
	if generation.StoppedAt == nil {
		now := time.Now().UTC()
		if err := st.StopGeneration(generation.ID, now); err != nil {
			return err
		}
	}
	agent, err := st.GetAgent(generation.AgentID, "")
	if err != nil {
		return err
	}
	// A legacy or externally-created Agent may temporarily have more than one
	// generation. Releasing one runtime must not mark the Agent stopped while a
	// different generation is still live.
	if generations, listErr := st.ListGenerations(generation.AgentID); listErr == nil {
		for _, other := range generations {
			if other.RuntimeID == runtimeID || other.StoppedAt != nil || other.State == "STOPPED" {
				continue
			}
			if n.runtimeAlive(other.RuntimeID) {
				agent.Status = store.AgentWorking
				return st.UpdateAgent(agent)
			}
		}
	}
	agent.Status = store.AgentStopped
	return st.UpdateAgent(agent)
}
