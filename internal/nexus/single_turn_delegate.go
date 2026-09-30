package nexus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// SingleTurnDelegateRequest is a bounded provider/specialist handoff that does
// not create a WorkPlan or MissionRun. It is intended for one synchronous turn
// whose result is returned to the interactive Lead.
type SingleTurnDelegateRequest struct {
	ProjectID      string
	Prompt         string
	PreferProvider string
	SkillIDs       []string
}

// SingleTurnDelegateResult records the actual resource used and the bounded
// output returned by the delegated turn.
type SingleTurnDelegateResult struct {
	AgentID  string   `json:"agent_id"`
	Provider string   `json:"provider"`
	Profile  string   `json:"profile"`
	SkillIDs []string `json:"skill_ids,omitempty"`
	Output   string   `json:"output"`
}

// DelegateSingleTurn selects a registered resource through the canonical
// scheduler, creates/reuses a dedicated specialist Agent, executes exactly one
// headless turn, and returns its output. It never creates a Mission.
func (n *Nexus) DelegateSingleTurn(ctx context.Context, req SingleTurnDelegateRequest) (*SingleTurnDelegateResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(req.ProjectID)
	if err != nil {
		return nil, err
	}

	preferProvider := strings.ToLower(strings.TrimSpace(req.PreferProvider))
	if preferProvider == "" {
		if provider, ok, resolveErr := n.ResolveRegisteredProviderPreference(req.Prompt); resolveErr != nil {
			return nil, resolveErr
		} else if ok {
			preferProvider = provider
		}
	}

	skills := append([]string(nil), req.SkillIDs...)
	if len(skills) == 0 {
		skills, err = n.ResolveAutomaticSkillIDs(ctx, req.ProjectID, req.Prompt)
		if err != nil {
			return nil, fmt.Errorf("resolve automatic skills: %w", err)
		}
	}
	compiled, err := n.CompileAgentPromptForProject(req.ProjectID, req.Prompt, skills)
	if err != nil {
		return nil, fmt.Errorf("compile delegated prompt: %w", err)
	}

	taskReq := ClassifyTaskRequirements(req.Prompt)
	taskReq.PreferProvider = preferProvider
	// Agent specialization owns semantic capabilities. Provider allocation only
	// hard-gates the transport/runtime capabilities needed for one headless turn.
	resourceReq := taskReq
	resourceReq.RequiredCapabilities = []string{"headless", "submit_prompt"}

	accounts, err := n.ListResourcesContext(ctx)
	if err != nil {
		return nil, err
	}
	policy := PolicyBalanced
	if preferProvider != "" {
		policy = PolicyPreferProvider
	}
	recommendation := RecommendResources(accounts, resourceReq, policy)
	if recommendation.Recommended == nil {
		return nil, fmt.Errorf("no eligible registered resource for delegated turn: %s", recommendation.Explanation)
	}
	selected := recommendation.Recommended.Account
	if preferProvider != "" && !strings.EqualFold(selected.Provider, preferProvider) {
		// PREFER may fall back when the requested provider is unavailable. Keep
		// that behavior explicit rather than pretending the requested provider ran.
		if !recommendation.Recommended.Eligible {
			return nil, fmt.Errorf("preferred provider %s is unavailable and no eligible fallback exists", preferProvider)
		}
	}

	agent, err := ensureSingleTurnDelegateAgent(st, project, taskReq)
	if err != nil {
		return nil, err
	}
	cfg, _ := currentAgentConfig(st, agent)
	cfg.Provider = selected.Provider
	cfg.Profile = selected.Profile
	cfg.Isolation = "project"
	cfg.AgentSpec = missionAgentSpec(defaultRole(taskReq.Role, "implementer"), taskReq)
	if _, err := n.SafeApply(ctx, agent.ID, cfg); err != nil {
		return nil, fmt.Errorf("configure delegated specialist: %w", err)
	}

	execution, err := n.executeAgentPrompt(ctx, agent.ID, project.CanonicalPath, compiled.CompiledPrompt, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
	})
	if err != nil {
		return nil, fmt.Errorf("execute delegated turn with %s:%s: %w", selected.Provider, selected.Profile, err)
	}
	if execution.Failure.Kind != "" {
		return nil, fmt.Errorf("delegated provider failure (%s): %s", execution.Failure.Kind, execution.Failure.Message)
	}
	return &SingleTurnDelegateResult{
		AgentID: agent.ID, Provider: selected.Provider, Profile: selected.Profile,
		SkillIDs: append([]string(nil), skills...), Output: strings.TrimSpace(execution.Output),
	}, nil
}

func ensureSingleTurnDelegateAgent(st *store.Store, project store.Project, req TaskRequirements) (store.Agent, error) {
	role := defaultRole(req.Role, "implementer")
	name := "Delegate · " + role
	agents, err := st.ListAgents(project.ID)
	if err != nil {
		return store.Agent{}, err
	}
	for _, agent := range agents {
		if agent.Name == name && (agent.Status == store.AgentStopped || agent.Status == store.AgentRecoverable || agent.Status == "") {
			return agent, nil
		}
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: name, Role: role})
	if err != nil {
		return store.Agent{}, err
	}
	cfg := AgentConfig{
		Isolation: "project",
		AgentSpec: missionAgentSpec(role, req),
		ContinuityPolicy: "new_session",
	}
	rev, err := st.AddRevision(agent.ID, cfg.ConfigJSON())
	if err != nil {
		_ = st.DeleteAgent(agent.ID, project.ID)
		return store.Agent{}, err
	}
	agent.CurrentRevisionID = rev.ID
	if err := st.UpdateAgent(agent); err != nil {
		_ = st.DeleteAgent(agent.ID, project.ID)
		return store.Agent{}, err
	}
	return agent, nil
}

// keep time imported in the same dependency graph as execution receipts when
// builds use selective file lists.
var _ = time.Time{}
