package nexus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// PlanExecutionPreview is a side-effect-free projection of the decisions Nexus
// would attempt for one WorkPlan revision. It is advisory: Mission admission
// revalidates live state immediately before dispatch.
type PlanExecutionPreview struct {
	PlanID                   string                 `json:"plan_id"`
	ProjectID                string                 `json:"project_id"`
	Revision                 int                    `json:"revision"`
	Ready                    bool                   `json:"ready"`
	Steps                    []StepExecutionPreview `json:"steps"`
	Blockers                 []string               `json:"blockers,omitempty"`
	Warnings                 []string               `json:"warnings,omitempty"`
	CostBudget               CostBudgetAssessment   `json:"cost_budget"`
	CostAnomaly              CostAnomalyAssessment  `json:"cost_anomaly"`
	CostConfirmationRequired bool                   `json:"cost_confirmation_required"`
	CostConfirmationToken    string                 `json:"cost_confirmation_token,omitempty"`
	GeneratedAt              time.Time              `json:"generated_at"`
}

// StepExecutionPreview explains WHO and WITH WHAT Nexus would currently use
// for one WorkPackage without allocating or mutating any runtime state.
type StepExecutionPreview struct {
	PackageID        string                  `json:"package_id"`
	PhaseID          string                  `json:"phase_id,omitempty"`
	Title            string                  `json:"title"`
	Role             string                  `json:"role,omitempty"`
	Assignment       string                  `json:"assignment_strategy,omitempty"`
	ResourcePolicy   string                  `json:"resource_policy,omitempty"`
	Requirements     TaskRequirements        `json:"task_requirements"`
	Agent            PreviewAgentDecision    `json:"agent"`
	Resource         PreviewResourceDecision `json:"resource"`
	Model            PreviewModelDecision    `json:"model"`
	Skills           []string                `json:"skills,omitempty"`
	AcceptanceGates  []string                `json:"acceptance_criteria,omitempty"`
	Verification     []string                `json:"verification_requirements,omitempty"`
	RequiredEvidence []string                `json:"required_evidence,omitempty"`
	Ready            bool                    `json:"ready"`
	Blockers         []string                `json:"blockers,omitempty"`
}

type PreviewAgentDecision struct {
	AgentID        string  `json:"agent_id,omitempty"`
	Name           string  `json:"name,omitempty"`
	Role           string  `json:"role,omitempty"`
	Score          float64 `json:"score,omitempty"`
	Confidence     string  `json:"confidence,omitempty"`
	Reason         string  `json:"reason"`
	WillCreate     bool    `json:"will_create,omitempty"`
	CandidateCount int     `json:"candidate_count,omitempty"`
}

type PreviewResourceDecision struct {
	AccountID      string  `json:"account_id,omitempty"`
	Provider       string  `json:"provider,omitempty"`
	Profile        string  `json:"profile,omitempty"`
	DisplayName    string  `json:"display_name,omitempty"`
	Policy         string  `json:"policy,omitempty"`
	Score          float64 `json:"score,omitempty"`
	Confidence     string  `json:"confidence,omitempty"`
	Reason         string  `json:"reason"`
	Health         string  `json:"health,omitempty"`
	Freshness      string  `json:"freshness,omitempty"`
	QuotaRemaining float64 `json:"quota_remaining,omitempty"`
	CandidateCount int     `json:"candidate_count,omitempty"`
}

type PreviewModelDecision struct {
	Model    string `json:"model,omitempty"`
	Fallback bool   `json:"fallback,omitempty"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence"`
}

// PreviewPlanExecution evaluates the current plan using the canonical
// AgentMatcher, ResourceScheduler recommendation and model resolver. It starts
// no Agent/runtime/Mission and does not persist allocation changes.
func (n *Nexus) PreviewPlanExecution(ctx context.Context, planID string) (*PlanExecutionPreview, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	accounts, resourceErr := n.ListResourcesContext(ctx)
	return n.previewPlanExecution(ctx, st, plan, accounts, resourceErr)
}

func (n *Nexus) previewPlanExecution(ctx context.Context, st *store.Store, plan *store.WorkPlan, accounts []ProviderAccount, resourceErr error) (*PlanExecutionPreview, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if st == nil || plan == nil {
		return nil, fmt.Errorf("plan execution preview requires store and plan")
	}
	agents, err := st.ListAgents(plan.ProjectID)
	if err != nil {
		return nil, err
	}
	preview := &PlanExecutionPreview{
		PlanID:      plan.ID,
		ProjectID:   plan.ProjectID,
		Revision:    plan.CurrentRevision,
		Ready:       true,
		Steps:       []StepExecutionPreview{},
		GeneratedAt: time.Now().UTC(),
	}
	costSummary, costErr := NewCostApplicationService(st).Summary(ctx, plan.ProjectID, 0)
	if costErr != nil {
		prefs, prefsErr := st.GetCostPreferences()
		if prefsErr != nil || (prefs.Enabled && (prefs.HardLimitMicroUSD > 0 || prefs.EmergencyLimitMicroUSD > 0)) {
			preview.Ready = false
			preview.Blockers = append(preview.Blockers, "cost budget admission unavailable: "+costErr.Error())
		} else {
			preview.Warnings = append(preview.Warnings, "Cost Intelligence unavailable: "+costErr.Error())
		}
	} else {
		preview.CostBudget = costSummary.Budget
		preview.CostAnomaly = costSummary.Anomaly
		preview.CostConfirmationRequired = costSummary.Budget.RequiresConfirmation
		preview.CostConfirmationToken = costBudgetConfirmationToken(plan.ID, plan.CurrentRevision, costSummary)
		switch costSummary.Budget.Level {
		case CostBudgetSoft:
			preview.Warnings = append(preview.Warnings, costSummary.Budget.Message)
		case CostBudgetHard, CostBudgetEmergency, CostBudgetUnknown:
			if costSummary.Budget.RequiresConfirmation {
				preview.Warnings = append(preview.Warnings, costSummary.Budget.Message)
			}
		}
		if costSummary.Anomaly.Triggered {
			preview.Warnings = append(preview.Warnings, costSummary.Anomaly.Message)
		}
	}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			step := n.previewWorkPackageExecution(st, plan.ProjectID, phase.ID, pkg, agents, accounts, resourceErr)
			preview.Steps = append(preview.Steps, step)
			if !step.Ready {
				preview.Ready = false
				for _, blocker := range step.Blockers {
					preview.Blockers = append(preview.Blockers, fmt.Sprintf("%s: %s", firstNonEmpty(pkg.Title, pkg.ID), blocker))
				}
			}
		}
	}
	return preview, nil
}

func (n *Nexus) previewWorkPackageExecution(st *store.Store, projectID, phaseID string, pkg store.WorkPackage, agents []store.Agent, accounts []ProviderAccount, resourceErr error) StepExecutionPreview {
	runPkg := &runner.PackageRun{
		PackageID:          pkg.ID,
		PhaseID:            phaseID,
		Title:              pkg.Title,
		Goal:               pkg.Goal,
		Role:               pkg.Role,
		TaskRequirements:   pkg.TaskRequirements,
		AssignmentStrategy: pkg.AssignmentStrategy,
		ResourcePolicy:     pkg.ResourcePolicy,
		Provider:           pkg.Provider,
		Profile:            pkg.Profile,
		DesiredProvider:    pkg.DesiredProvider,
		DesiredProfile:     pkg.DesiredProfile,
	}
	req := missionTaskRequirements(runPkg)
	policy := flowResourcePolicy(pkg.ResourcePolicy)
	req.ProjectPolicy = string(policy)
	if strings.TrimSpace(pkg.Provider) != "" {
		req.PreferProvider = strings.TrimSpace(pkg.Provider)
	}

	step := StepExecutionPreview{
		PackageID:        pkg.ID,
		PhaseID:          phaseID,
		Title:            pkg.Title,
		Role:             pkg.Role,
		Assignment:       pkg.AssignmentStrategy,
		ResourcePolicy:   string(policy),
		Requirements:     req,
		Skills:           append([]string(nil), packageSkillIDs(pkg)...),
		AcceptanceGates:  append([]string(nil), pkg.AcceptanceCriteria...),
		Verification:     append([]string(nil), pkg.VerificationRequirements...),
		RequiredEvidence: append([]string(nil), pkg.RequiredEvidence...),
		Ready:            true,
		Agent: PreviewAgentDecision{
			Reason: "Agent decision not evaluated",
		},
		Resource: PreviewResourceDecision{
			Policy: string(policy),
			Reason: "Resource decision not evaluated",
		},
		Model: PreviewModelDecision{
			Reason:   "Model inventory not evaluated",
			Evidence: "UNKNOWN",
		},
	}

	selectedAgent, agentCfg, agentDecision, agentBlocker := previewAgentForPackage(st, projectID, pkg, req, agents)
	step.Agent = agentDecision
	if agentBlocker != "" {
		step.Ready = false
		step.Blockers = append(step.Blockers, agentBlocker)
	}

	if resourceErr != nil {
		step.Ready = false
		step.Blockers = append(step.Blockers, "resource discovery unavailable: "+resourceErr.Error())
		step.Resource.Reason = "Resource discovery failed"
		return step
	}
	if len(accounts) == 0 {
		step.Ready = false
		step.Blockers = append(step.Blockers, "no provider accounts discovered")
		step.Resource.Reason = "No provider accounts discovered"
		return step
	}

	if selectedAgent.ID == "" && !agentDecision.WillCreate && agentBlocker != "" {
		return step
	}

	desiredProvider := strings.TrimSpace(pkg.DesiredProvider)
	desiredProfile := strings.TrimSpace(pkg.DesiredProfile)
	if desiredProvider == "" {
		desiredProvider = strings.TrimSpace(pkg.Provider)
	}
	if desiredProfile == "" {
		desiredProfile = strings.TrimSpace(pkg.Profile)
	}
	runPkg.DesiredProvider = desiredProvider
	runPkg.DesiredProfile = desiredProfile
	preferenceMode := missionAffinityMode(policy, desiredProvider, desiredProfile)
	candidateAccounts, candidateErr := selectMissionCandidateAccounts(accounts, runPkg, agentCfg, preferenceMode, false)
	if candidateErr != nil {
		step.Ready = false
		step.Blockers = append(step.Blockers, candidateErr.Error())
		step.Resource.Reason = candidateErr.Error()
		return step
	}
	candidateAccounts, isolationRejected := filterAccountsByAutonomyIsolation(candidateAccounts, runner.DefaultAutonomyContract(), false)
	if len(candidateAccounts) == 0 && len(isolationRejected) > 0 {
		step.Ready = false
		step.Blockers = append(step.Blockers, "no provider satisfies the default Mission isolation contract: "+strings.Join(isolationRejected, "; "))
		step.Resource.Reason = "Provider isolation contract is not verified"
		return step
	}
	if len(candidateAccounts) == 0 {
		step.Ready = false
		step.Blockers = append(step.Blockers, "no eligible provider account for this step")
		step.Resource.Reason = "No eligible provider account"
		return step
	}

	var selected ProviderAccount
	var selectedCandidate *ResourceCandidate
	var keepCurrent bool
	if selectedAgent.ID != "" {
		selected, keepCurrent = selectCurrentResource(candidateAccounts, agentCfg, req, policy)
	}
	if keepCurrent {
		step.Resource = previewResourceFromAccount(selected, policy, "Keep current healthy compatible resource", len(candidateAccounts), nil)
	} else {
		recommendation := RecommendResources(candidateAccounts, req, policy)
		if recommendation.Recommended == nil {
			step.Ready = false
			step.Blockers = append(step.Blockers, firstNonEmpty(recommendation.Explanation, "no eligible provider account for this step"))
			step.Resource.Reason = firstNonEmpty(recommendation.Explanation, "No eligible provider account")
			step.Resource.CandidateCount = len(recommendation.Candidates)
			return step
		}
		selectedCandidate = recommendation.Recommended
		selected = selectedCandidate.Account
		step.Resource = previewResourceFromAccount(selected, policy, recommendation.Explanation, len(recommendation.Candidates), selectedCandidate)
	}

	modelPreference := AffinityPreference{Mode: AffinityAuto}
	if req.RuntimeAffinity != nil {
		modelPreference = req.RuntimeAffinity.Model
	}
	modelCandidates := configuredModelCandidates(agentCfg, selected, req.ModelCandidates...)
	if len(modelCandidates) == 0 {
		if modelPreference.Mode == AffinityPin {
			step.Ready = false
			step.Blockers = append(step.Blockers, "pinned model requested but model inventory is unavailable")
			step.Model = PreviewModelDecision{Reason: "Pinned model cannot be verified against model inventory", Evidence: "UNKNOWN"}
			return step
		}
		step.Model = PreviewModelDecision{
			Reason:   "No evidence-based model inventory is available for this resource; Mission will not invent a model choice",
			Evidence: "UNKNOWN",
		}
		return step
	}

	model, fallback, reason, modelErr := ResolveTaskModel(req, modelPreference, selected, modelCandidates)
	if modelErr != nil {
		step.Ready = false
		step.Blockers = append(step.Blockers, "model selection: "+modelErr.Error())
		step.Model = PreviewModelDecision{Reason: modelErr.Error(), Evidence: "LIVE_CONFIGURATION"}
		return step
	}
	step.Model = PreviewModelDecision{
		Model:    model.Model,
		Fallback: fallback,
		Reason:   reason,
		Evidence: "LIVE_CONFIGURATION",
	}
	return step
}

func previewAgentForPackage(st *store.Store, projectID string, pkg store.WorkPackage, req TaskRequirements, agents []store.Agent) (store.Agent, AgentConfig, PreviewAgentDecision, string) {
	strategy := strings.ToUpper(strings.TrimSpace(pkg.AssignmentStrategy))
	if strategy == "" && strings.TrimSpace(pkg.AgentAllocation) != "" {
		strategy = string(FlowAssignmentExisting)
	}

	switch strategy {
	case string(FlowAssignmentExisting):
		if strings.TrimSpace(pkg.AgentAllocation) == "" {
			return store.Agent{}, AgentConfig{}, PreviewAgentDecision{Reason: "EXISTING requires an Agent"}, "existing assignment requires an Agent"
		}
		agent, err := st.GetAgent(pkg.AgentAllocation, projectID)
		if err != nil {
			return store.Agent{}, AgentConfig{}, PreviewAgentDecision{AgentID: pkg.AgentAllocation, Reason: "Pinned Agent is unavailable"}, "pinned Agent is unavailable"
		}
		cfg, err := currentAgentConfig(st, agent)
		if err != nil {
			return store.Agent{}, AgentConfig{}, PreviewAgentDecision{AgentID: agent.ID, Name: agent.Name, Reason: "Pinned Agent config is unavailable"}, "pinned Agent config is unavailable"
		}
		return agent, cfg, PreviewAgentDecision{
			AgentID:    agent.ID,
			Name:       agent.Name,
			Role:       firstNonEmpty(cfg.AgentSpec.Role, agent.Role),
			Confidence: "PINNED",
			Reason:     "Existing Agent explicitly assigned by the Flow",
		}, ""
	case string(FlowAssignmentCreate):
		return store.Agent{}, AgentConfig{}, PreviewAgentDecision{
			Role:       firstNonEmpty(req.Role, pkg.Role, "implementer"),
			Confidence: "PLANNED",
			Reason:     "Flow requires creation of a dedicated specialist at execution time",
			WillCreate: true,
		}, ""
	}

	candidates := make([]AgentMatchCandidate, 0, len(agents))
	for _, agent := range agents {
		cfg, err := currentAgentConfig(st, agent)
		if err != nil {
			continue
		}
		candidates = append(candidates, AgentMatchCandidate{Agent: agent, Spec: cfg.AgentSpec})
	}
	if len(req.PreferredRoles) == 0 && strings.TrimSpace(pkg.Role) != "" {
		role := strings.TrimSpace(pkg.Role)
		req.PreferredRoles = []string{role}
		req.AcceptableRoles = []string{role, "fullstack-engineer", "generalist"}
	}
	match := MatchAgents(candidates, AgentMatchRequirements{
		PreferredRoles:        req.PreferredRoles,
		AcceptableRoles:       req.AcceptableRoles,
		Domains:               req.Domains,
		RequiredCapabilities:  agentRequiredCapabilities(req.RequiredCapabilities),
		PreferredCapabilities: req.PreferredCapabilities,
		DesiredStrengths:      req.DesiredStrengths,
		Constraints:           req.Constraints,
	})
	if match.Recommended == nil {
		return store.Agent{}, AgentConfig{}, PreviewAgentDecision{
			Role:           firstNonEmpty(req.Role, pkg.Role, "implementer"),
			Confidence:     "PLANNED",
			Reason:         firstNonEmpty(match.Explanation, "No reusable Agent matches; Mission will create a specialist"),
			WillCreate:     true,
			CandidateCount: len(match.Candidates),
		}, ""
	}
	agent := match.Recommended.Agent
	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		return store.Agent{}, AgentConfig{}, PreviewAgentDecision{AgentID: agent.ID, Name: agent.Name, Reason: "Matched Agent config is unavailable"}, "matched Agent config is unavailable"
	}
	return agent, cfg, PreviewAgentDecision{
		AgentID:        agent.ID,
		Name:           agent.Name,
		Role:           firstNonEmpty(cfg.AgentSpec.Role, agent.Role),
		Score:          match.Recommended.Score,
		Confidence:     match.Recommended.Confidence,
		Reason:         match.Explanation,
		CandidateCount: len(match.Candidates),
	}, ""
}

func previewResourceFromAccount(account ProviderAccount, policy SchedulerPolicy, reason string, candidateCount int, candidate *ResourceCandidate) PreviewResourceDecision {
	freshness := "UNKNOWN"
	if account.QuotaView != nil && strings.TrimSpace(account.QuotaView.Status) != "" {
		freshness = account.QuotaView.Status
	}
	out := PreviewResourceDecision{
		AccountID:      account.ID,
		Provider:       account.Provider,
		Profile:        account.Profile,
		DisplayName:    account.DisplayName,
		Policy:         string(policy),
		Reason:         firstNonEmpty(reason, "Selected by the existing Nexus resource scheduler"),
		Health:         account.Health,
		Freshness:      freshness,
		QuotaRemaining: account.QuotaRemaining,
		CandidateCount: candidateCount,
	}
	if candidate != nil {
		out.Score = candidate.TotalScore
		out.Confidence = candidate.Confidence
	}
	return out
}
