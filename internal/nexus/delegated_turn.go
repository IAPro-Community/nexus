package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// DelegatedTurnRequest describes one bounded specialist turn requested from an
// interactive Lead. It deliberately does not create a WorkPlan or Mission.
type DelegatedTurnRequest struct {
	ProjectID         string   `json:"project_id"`
	LeadAgentID       string   `json:"lead_agent_id,omitempty"`
	Prompt            string   `json:"prompt"`
	PreferredProvider string   `json:"preferred_provider,omitempty"`
	SkillIDs          []string `json:"skill_ids,omitempty"`
}

// DelegatedTurnResult is the bounded result returned to the interactive Lead.
// The worker identity is ephemeral; provider/profile and skills are retained as
// explainability evidence.
type DelegatedTurnResult struct {
	Provider  string   `json:"provider"`
	Profile   string   `json:"profile"`
	RuntimeID string   `json:"runtime_id,omitempty"`
	Output    string   `json:"output"`
	SkillIDs  []string `json:"skill_ids,omitempty"`
	Fallback  bool     `json:"fallback"`
	Reason    string   `json:"reason"`
}

const maxDelegatedTurnOutputBytes = 32 * 1024

// ExecuteDelegatedTurn executes exactly one read-only headless specialist turn.
// It reuses the canonical registered resource inventory, ResourceScheduler,
// SkillCatalog and executeAgentPrompt path. It never mutates the Lead Agent and
// never creates a Mission.
func (n *Nexus) ExecuteDelegatedTurn(ctx context.Context, req DelegatedTurnRequest) (*DelegatedTurnResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.PreferredProvider = strings.ToLower(strings.TrimSpace(req.PreferredProvider))
	if req.ProjectID == "" {
		return nil, fmt.Errorf("delegated turn requires project_id")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("delegated turn requires prompt")
	}

	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(req.ProjectID)
	if err != nil {
		return nil, err
	}

	if req.PreferredProvider == "" {
		if provider, ok, resolveErr := n.ResolveRegisteredProviderPreference(req.Prompt); resolveErr != nil {
			return nil, fmt.Errorf("resolve delegated provider preference: %w", resolveErr)
		} else if ok {
			req.PreferredProvider = provider
		}
	}

	requirements := ClassifyTaskRequirements(req.Prompt)
	requirements.RequiresDecomposition = false
	requirements.PreferProvider = req.PreferredProvider
	if len(req.SkillIDs) == 0 {
		skills, skillErr := n.ResolveAutomaticSkillIDs(ctx, req.ProjectID, req.Prompt)
		if skillErr != nil {
			return nil, fmt.Errorf("resolve delegated skills: %w", skillErr)
		}
		req.SkillIDs = skills
	}
	req.SkillIDs = uniqueStrings(req.SkillIDs)

	compiled, err := n.CompileAgentPromptForProject(req.ProjectID, delegatedTurnPrompt(req.Prompt), req.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("compile delegated turn: %w", err)
	}

	accounts, err := n.ListResourcesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list delegated resources: %w", err)
	}
	selected, fallback, reason, err := selectDelegatedTurnResource(accounts, requirements, req.PreferredProvider)
	if err != nil {
		return nil, err
	}

	role := strings.TrimSpace(requirements.Role)
	if len(requirements.PreferredRoles) > 0 && strings.TrimSpace(requirements.PreferredRoles[0]) != "" {
		role = strings.TrimSpace(requirements.PreferredRoles[0])
	}
	if role == "" {
		role = "reviewer"
	}
	spec := missionAgentSpec(role, requirements)
	spec.Instructions = appendUniqueStrings(spec.Instructions,
		"Execute one bounded read-only delegated turn and return evidence to the interactive Lead.",
		"Do not modify files, commit, push, deploy, or perform external side effects.")
	spec.Constraints = appendUniqueStrings(spec.Constraints,
		"read-only delegated turn",
		"no workspace mutation",
		"no external side effects")

	agent, err := st.CreateAgent(store.Agent{
		ProjectID:    project.ID,
		Name:         "Delegate · " + role,
		Role:         role,
		IdentityKind: "EPHEMERAL_WORKER",
		Status:       store.AgentStopped,
	})
	if err != nil {
		return nil, fmt.Errorf("create delegated specialist: %w", err)
	}
	defer func() { _ = st.DeleteAgent(agent.ID, project.ID) }()

	cfg := AgentConfig{
		Provider:  selected.Provider,
		Profile:   selected.Profile,
		Workspace: project.CanonicalPath,
		Isolation: "project",
		AgentSpec: spec,
	}
	rev, err := st.AddRevision(agent.ID, cfg.ConfigJSON())
	if err != nil {
		return nil, fmt.Errorf("persist delegated specialist config: %w", err)
	}
	agent.CurrentRevisionID = rev.ID
	if err := st.UpdateAgent(agent); err != nil {
		return nil, fmt.Errorf("link delegated specialist config: %w", err)
	}

	execution, err := n.executeAgentPrompt(ctx, agent.ID, project.CanonicalPath, compiled.CompiledPrompt, agentPromptPolicy{
		Contract: runner.DefaultAutonomyContract(),
		Review:   true,
	})
	if err != nil {
		if execution != nil {
			return &DelegatedTurnResult{
				Provider: selected.Provider, Profile: selected.Profile, RuntimeID: execution.RuntimeID,
				Output: boundedDelegatedTurnOutput(execution.Output), SkillIDs: req.SkillIDs,
				Fallback: fallback, Reason: reason,
			}, fmt.Errorf("delegated turn failed: %w", err)
		}
		return nil, fmt.Errorf("delegated turn failed: %w", err)
	}
	return &DelegatedTurnResult{
		Provider: selected.Provider, Profile: selected.Profile, RuntimeID: execution.RuntimeID,
		Output: boundedDelegatedTurnOutput(execution.Output), SkillIDs: req.SkillIDs,
		Fallback: fallback, Reason: reason,
	}, nil
}

func delegatedTurnPrompt(prompt string) string {
	return `[Nexus bounded delegated turn]
Mode: READ_ONLY
Scope: this turn only.
Do not modify files or configuration. Do not commit, push, deploy, install, publish, or cause external side effects.
Inspect the available project context and answer the request with concise findings, evidence, and actionable recommendations.
If the request would normally require writing code, describe the exact recommended change instead of applying it.

User request:
` + strings.TrimSpace(prompt)
}

func selectDelegatedTurnResource(accounts []ProviderAccount, req TaskRequirements, preferredProvider string) (ProviderAccount, bool, string, error) {
	eligible := make([]ProviderAccount, 0, len(accounts))
	contract := runner.DefaultAutonomyContract()
	for _, account := range accounts {
		if !accountSupportsHeadlessReview(account) {
			continue
		}
		if ok, _ := providerSatisfiesAutonomyIsolation(account.Provider, contract, true); !ok {
			continue
		}
		eligible = append(eligible, account)
	}
	if len(eligible) == 0 {
		return ProviderAccount{}, false, "", fmt.Errorf("%w: no registered provider can satisfy a read-only delegated turn", ErrBlockedResource)
	}

	resourceReq := TaskRequirements{
		TaskKind:             req.TaskKind,
		Role:                 req.Role,
		PreferredRoles:       append([]string(nil), req.PreferredRoles...),
		AcceptableRoles:      append([]string(nil), req.AcceptableRoles...),
		Domains:              append([]string(nil), req.Domains...),
		RequiredCapabilities: []string{"headless", "submit_prompt"},
		PreferProvider:       strings.ToLower(strings.TrimSpace(preferredProvider)),
		ProviderPriorities:   req.ProviderPriorities,
	}

	if resourceReq.PreferProvider != "" {
		preferred := filterFlowResourceAccounts(eligible, resourceReq.PreferProvider, "")
		if len(preferred) > 0 {
			rec := RecommendResources(preferred, resourceReq, PolicyPreferProvider)
			if rec.Recommended != nil {
				return rec.Recommended.Account, false, rec.Explanation, nil
			}
		}
	}

	policy := PolicyBalanced
	fallback := false
	reasonPrefix := ""
	if resourceReq.PreferProvider != "" {
		policy = PolicyPreferProvider
		fallback = true
		reasonPrefix = fmt.Sprintf("preferred provider %s unavailable for safe delegated execution; ", resourceReq.PreferProvider)
	}
	rec := RecommendResources(eligible, resourceReq, policy)
	if rec.Recommended == nil {
		return ProviderAccount{}, fallback, "", fmt.Errorf("%w: %s", ErrBlockedResource, rec.Explanation)
	}
	return rec.Recommended.Account, fallback, reasonPrefix + rec.Explanation, nil
}

func boundedDelegatedTurnOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) <= maxDelegatedTurnOutputBytes {
		return output
	}
	return output[:maxDelegatedTurnOutputBytes] + "\n… [delegated output truncated]"
}
