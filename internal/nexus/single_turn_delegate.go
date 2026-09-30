package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// SingleTurnDelegateResult is a bounded delegated execution result. It is
// deliberately not a WorkPlan or MissionRun: the interactive Lead owns the
// conversation and receives this result back as context.
type SingleTurnDelegateResult struct {
	AgentID   string   `json:"agent_id"`
	Provider  string   `json:"provider"`
	Profile   string   `json:"profile"`
	Model     string   `json:"model,omitempty"`
	SkillIDs  []string `json:"skill_ids,omitempty"`
	Output    string   `json:"output"`
	Fallback  bool     `json:"fallback"`
	Reason    string   `json:"reason"`
}

// IsSingleTurnDelegateRequest limits conversational delegation to bounded,
// analysis-oriented work. Any write language or decomposition requirement stays
// on the existing Mission path so Nexus never introduces a silent second writer
// into the user's checkout.
func IsSingleTurnDelegateRequest(prompt string, req TaskRequirements) bool {
	if req.RequiresDecomposition {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(prompt))
	if containsAny(text,
		"implemente", "implementar", "implement ", "implement\n",
		"corrija", "corrigir", "fix ", "fixe ",
		"altere", "alterar", "modifique", "modificar", "change ",
		"adicione", "adicionar", "add ", "remova", "remover", "remove ",
		"crie", "criar", "create ", "atualize", "atualizar", "update ",
		"edite", "editar", "edit ", "refatore", "refatorar", "refactor ",
		"escreva", "escrever", "write ", "aplique", "aplicar", "apply ",
	) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(req.TaskKind)) {
	case "review", "security", "investigation", "architecture":
		return true
	default:
		return false
	}
}

// ExecuteSingleTurnDelegate performs exactly one headless, read-only delegated
// turn through the canonical registered-resource scheduler. It creates no
// WorkPlan and no MissionRun. The specialist executes in an ephemeral
// Nexus-managed worktree after the provider's verified read-only review
// contract passes. The caller returns the textual result to the interactive Lead.
func (n *Nexus) ExecuteSingleTurnDelegate(ctx context.Context, projectID, prompt, preferredProvider string) (*SingleTurnDelegateResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	prompt = strings.TrimSpace(prompt)
	preferredProvider = strings.ToLower(strings.TrimSpace(preferredProvider))
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required for delegated turn")
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required for delegated turn")
	}

	req := ClassifyTaskRequirements(prompt)
	if !IsSingleTurnDelegateRequest(prompt, req) {
		return nil, fmt.Errorf("single-turn delegation is read-only; task %q requires Direct or Mission execution", req.TaskKind)
	}
	if preferredProvider != "" {
		req.PreferProvider = preferredProvider
	}

	skills, err := n.ResolveAutomaticSkillIDs(ctx, projectID, prompt)
	if err != nil {
		return nil, fmt.Errorf("resolve delegated skills: %w", err)
	}
	compiled, err := n.CompileAgentPromptForProject(projectID, prompt, skills)
	if err != nil {
		return nil, fmt.Errorf("compile delegated skill contracts: %w", err)
	}
	delegatedPrompt := compiled.CompiledPrompt

	accounts, err := n.ListResourcesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list delegated resources: %w", err)
	}
	if preferredProvider != "" {
		registered := false
		for _, account := range accounts {
			if strings.EqualFold(strings.TrimSpace(account.Provider), preferredProvider) {
				registered = true
				break
			}
		}
		if !registered {
			return nil, fmt.Errorf("preferred provider %q is not registered in Nexus resources", preferredProvider)
		}
	}
	contract := runner.DefaultAutonomyContract()
	resourceReq := req
	// Provider transport capabilities are distinct from semantic Agent skills.
	resourceReq.RequiredCapabilities = []string{"headless", "submit_prompt"}
	pool := accounts
	policy := PolicyBalanced
	if preferredProvider != "" {
		policy = PolicyPreferProvider
		preferred := filterFlowResourceAccounts(accounts, preferredProvider, "")
		eligiblePreferred := make([]ProviderAccount, 0, len(preferred))
		for _, account := range preferred {
			if !evaluateCandidate(account, resourceReq, policy).Eligible {
				continue
			}
			if ok, _ := providerSatisfiesAutonomyIsolation(account.Provider, contract, true); !ok {
				continue
			}
			eligiblePreferred = append(eligiblePreferred, account)
		}
		if len(eligiblePreferred) > 0 {
			pool = eligiblePreferred
		}
	}
	pool, isolationRejected := filterAccountsByAutonomyIsolation(pool, contract, true)
	if len(pool) == 0 {
		return nil, fmt.Errorf("no registered provider can satisfy read-only delegated execution: %s", strings.Join(isolationRejected, "; "))
	}
	recommendation := RecommendResources(pool, resourceReq, policy)
	if recommendation.Recommended == nil {
		return nil, fmt.Errorf("no eligible registered resource for delegated turn: %s", recommendation.Explanation)
	}
	selected := recommendation.Recommended.Account

	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	rawReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	pkg := &runner.PackageRun{
		PackageID:        fmt.Sprintf("delegate_%d", time.Now().UTC().UnixNano()),
		Title:            "Single-turn delegate · " + firstNonEmpty(req.Role, req.TaskKind, "review"),
		Goal:             prompt,
		Role:             firstNonEmpty(req.Role, "reviewer"),
		TaskRequirements: string(rawReq),
		SkillIDs:         append([]string(nil), skills...),
	}
	agent, err := createMissionAgent(st, projectID, pkg, "Delegate · ")
	if err != nil {
		return nil, fmt.Errorf("create delegated specialist: %w", err)
	}
	// DELEGATE is intentionally ephemeral. Keep audit/cost evidence from the
	// invocation, but do not accumulate a persistent Agent for every one-turn
	// consultation.
	defer func() { _ = n.DeleteAgent(agent.ID, projectID) }()

	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		return nil, err
	}
	cfg.Provider = selected.Provider
	cfg.Profile = selected.Profile
	cfg.Model = ""
	cfg.Isolation = "worktree"
	if _, err := n.SafeApply(ctx, agent.ID, cfg); err != nil {
		return nil, fmt.Errorf("configure delegated specialist: %w", err)
	}
	workspace, err := n.resolveExecutionWorkspace(ctx, project, agent, cfg)
	if err != nil {
		return nil, fmt.Errorf("prepare delegated worktree: %w", err)
	}
	defer func() { _ = removeAgentWorktree(context.Background(), project, agent, workspace) }()

	execution, err := n.executeAgentPrompt(ctx, agent.ID, workspace, delegatedPrompt, agentPromptPolicy{Contract: contract, Review: true})
	if err != nil {
		return nil, fmt.Errorf("execute delegated turn: %w", err)
	}
	return &SingleTurnDelegateResult{
		AgentID:  agent.ID,
		Provider: selected.Provider,
		Profile:  selected.Profile,
		Model:    cfg.Model,
		SkillIDs: append([]string(nil), skills...),
		Output:   execution.Output,
		Fallback: preferredProvider != "" && !strings.EqualFold(preferredProvider, selected.Provider),
		Reason:   recommendation.Explanation,
	}, nil
}
