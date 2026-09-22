package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/autonomyguard"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type nexusPackageExecutor struct {
	n *Nexus
}

func newNexusPackageExecutor(n *Nexus) runner.PackageExecutor {
	return &nexusPackageExecutor{n: n}
}

func (e *nexusPackageExecutor) Allocate(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.AllocationResult, error) {
	if e.n == nil {
		return runner.AllocationResult{}, fmt.Errorf("nexus executor unavailable")
	}
	st, err := e.n.OpenProject()
	if err != nil {
		return runner.AllocationResult{}, err
	}
	if strings.TrimSpace(run.ExecutionSnapshotID) != "" {
		snapshot, snapshotErr := e.n.loadMissionSnapshot(run)
		if snapshotErr != nil {
			return runner.AllocationResult{}, fmt.Errorf("revalidate execution admission: %w", snapshotErr)
		}
		if run.Autonomous || snapshot.Preflight != nil && snapshot.Preflight.Strict {
			// Strict admission mutates Ready only when a check fails. Start from
			// ready here, matching PreflightFlow, so a valid immutable snapshot is
			// not rejected merely because this revalidation report is fresh.
			report := &FlowPreflightReport{PlanID: snapshot.Plan.ID, Revision: snapshot.Plan.CurrentRevision, Strict: true, Ready: true, GeneratedAt: time.Now().UTC()}
			if _, admissionErr := e.n.strictAdmissionForPlan(ctx, snapshot.Plan, report, run.Autonomous); admissionErr != nil {
				return runner.AllocationResult{}, fmt.Errorf("revalidate execution admission: %w", admissionErr)
			} else if !report.Ready {
				return runner.AllocationResult{}, fmt.Errorf("execution admission failed before dispatch")
			}
		}
	}

	strategy := strings.ToUpper(strings.TrimSpace(pkg.AssignmentStrategy))
	var agent store.Agent
	var agentEvidence agentSelectionEvidence
	switch strategy {
	case "": // legacy WorkPlan behavior
		if strings.TrimSpace(pkg.AssignedAgent) != "" {
			agent, err = st.GetAgent(pkg.AssignedAgent, run.ProjectID)
			agentEvidence = agentSelectionEvidence{Confidence: "PINNED", Reason: "existing Agent assignment"}
		} else {
			agent, agentEvidence, err = e.selectReusableAgent(st, run, pkg)
			if err == store.ErrNotFound {
				agent, err = createMissionAgent(st, run.ProjectID, pkg, "Mission · ")
				agentEvidence = agentSelectionEvidence{Confidence: "CREATED", Reason: "created from task requirements"}
			}
		}
	case string(FlowAssignmentExisting):
		if strings.TrimSpace(pkg.AssignedAgent) == "" {
			return runner.AllocationResult{}, fmt.Errorf("EXISTING Flow Step %s has no AgentID", pkg.PackageID)
		}
		if agentReservedInRun(run, pkg.AssignedAgent, pkg.PackageID) {
			return runner.AllocationResult{}, fmt.Errorf("agent %s is already assigned to another active Flow Step", pkg.AssignedAgent)
		}
		agent, err = st.GetAgent(pkg.AssignedAgent, run.ProjectID)
		agentEvidence = agentSelectionEvidence{Confidence: "PINNED", Reason: "existing Flow Agent assignment"}
	case string(FlowAssignmentCreate):
		agent, err = createMissionAgent(st, run.ProjectID, pkg, "Flow · ")
		agentEvidence = agentSelectionEvidence{Confidence: "CREATED", Reason: "created from Flow task requirements"}
	case string(FlowAssignmentAuto):
		agent, agentEvidence, err = e.selectReusableAgent(st, run, pkg)
		if err == store.ErrNotFound {
			agent, err = createMissionAgent(st, run.ProjectID, pkg, "Auto · ")
			agentEvidence = agentSelectionEvidence{Confidence: "CREATED", Reason: "created from task requirements"}
		}
	default:
		return runner.AllocationResult{}, fmt.Errorf("unsupported Flow assignment strategy %q", pkg.AssignmentStrategy)
	}
	if err != nil {
		return runner.AllocationResult{}, fmt.Errorf("allocate persistent Agent: %w", err)
	}

	req := missionTaskRequirements(pkg)
	policy := flowResourcePolicy(pkg.ResourcePolicy)
	req.ProjectPolicy = string(policy)
	if strings.TrimSpace(pkg.Provider) != "" {
		req.PreferProvider = strings.TrimSpace(pkg.Provider)
	}
	accounts, err := e.n.ListResources()
	if err != nil {
		return runner.AllocationResult{}, err
	}

	current, _ := currentAgentConfig(st, agent)
	isFailover := pkg.Attempt > 1 && (pkg.RetryFrom == runner.StateAllocating || isQuotaOrRateLimitFailure(pkg.FailureKind))

	desiredProvider := strings.TrimSpace(pkg.DesiredProvider)
	desiredProfile := strings.TrimSpace(pkg.DesiredProfile)
	if desiredProvider == "" && pkg.Attempt <= 1 {
		desiredProvider = strings.TrimSpace(pkg.Provider)
	}
	if desiredProfile == "" && pkg.Attempt <= 1 {
		desiredProfile = strings.TrimSpace(pkg.Profile)
	}
	pkg.DesiredProvider, pkg.DesiredProfile = desiredProvider, desiredProfile
	preferenceMode := missionAffinityMode(policy, desiredProvider, desiredProfile)

	candidateAccounts, candidateErr := selectMissionCandidateAccounts(accounts, pkg, current, preferenceMode, isFailover)
	if candidateErr != nil {
		return runner.AllocationResult{}, candidateErr
	}
	candidateAccounts, isolationRejected := filterAccountsByAutonomyIsolation(candidateAccounts, run.Contract, false)
	if len(candidateAccounts) == 0 && len(isolationRejected) > 0 {
		return runner.AllocationResult{}, fmt.Errorf("%w: no provider satisfies the Mission isolation contract for Flow Step %s: %s", ErrBlockedResource, pkg.PackageID, strings.Join(isolationRejected, "; "))
	}
	if len(candidateAccounts) == 0 {
		if preferenceMode == AffinityPin {
			return runner.AllocationResult{}, fmt.Errorf("%w: pinned runtime has no eligible account for Flow Step %s", ErrBlockedResource, pkg.PackageID)
		}
		return runner.AllocationResult{}, explainNoEligibleMissionAccounts(accounts, pkg.PackageID, isFailover)
	}

	var selected ProviderAccount
	var keepCurrent bool
	routingReason := ""
	if !isFailover {
		selected, keepCurrent = selectCurrentResource(candidateAccounts, current, req, policy)
	}
	if !keepCurrent {
		recommendation := RecommendResources(candidateAccounts, req, policy)
		if recommendation.Recommended == nil {
			if preferenceMode == AffinityPin {
				return runner.AllocationResult{}, fmt.Errorf("%w: pinned runtime unavailable for Flow Step %s: %s", ErrBlockedResource, pkg.PackageID, recommendation.Explanation)
			}
			return runner.AllocationResult{}, explainNoEligibleMissionAccounts(candidateAccounts, pkg.PackageID, isFailover)
		}
		selected = recommendation.Recommended.Account
		routingReason = recommendation.Explanation
	}

	current.Provider, current.Profile = selected.Provider, selected.Profile
	if pkg.Attempt > 1 && pkg.RetryFrom == runner.StateAllocating && req.EscalationLevel < pkg.Attempt-1 {
		req.EscalationLevel = pkg.Attempt - 1
	}
	modelFallback := false
	modelReason := ""
	modelPreference := AffinityPreference{Mode: AffinityAuto}
	if req.RuntimeAffinity != nil {
		modelPreference = req.RuntimeAffinity.Model
	}
	modelCandidates := configuredModelCandidates(current, selected, req.ModelCandidates...)
	if len(modelCandidates) > 0 {
		model, fallback, reason, modelErr := ResolveTaskModel(req, modelPreference, selected, modelCandidates)
		if modelErr != nil {
			return runner.AllocationResult{}, fmt.Errorf("resolve task model: %w", modelErr)
		}
		current.Model = model.Model
		modelFallback, modelReason = fallback, reason
	} else if modelPreference.Mode == AffinityPin {
		return runner.AllocationResult{}, fmt.Errorf("resolve task model: %w: model inventory unavailable", ErrBlockedResource)
	} else {
		current.Model = ""
		if modelPreference.Mode == AffinityPrefer && modelPreference.Value != "" {
			modelFallback = true
			modelReason = fmt.Sprintf("preferred model %q unavailable in runtime inventory; no WorkUnit model selected", modelPreference.Value)
		}
	}
	pkg.Provider, pkg.Profile = selected.Provider, selected.Profile
	routingReason = strings.TrimSpace(strings.Join([]string{routingReason, modelReason}, "; "))
	routingDecision := buildMissionRoutingDecision(pkg, agent, req, selected, current, desiredProvider, desiredProfile, preferenceMode, isFailover || modelFallback, routingReason, agentEvidence)
	skillResolutions, skillErr := resolveSkillReferences(e.n, run.ProjectID, packageRunSkillIDs(pkg))
	if skillErr != nil {
		return runner.AllocationResult{}, fmt.Errorf("persist skill resolution evidence: %w", skillErr)
	}
	routingDecision.SkillResolutions = skillResolutions
	raw, marshalErr := json.Marshal(routingDecision)
	if marshalErr != nil {
		return runner.AllocationResult{}, fmt.Errorf("persist routing decision: %w", marshalErr)
	}
	pkg.RoutingDecisionJSON = string(raw)

	// Interactive and Flow agents run in the Project folder by default. Worktree
	// isolation stays opt-in via agent config or project.default_isolation.
	if !run.Autonomous && strings.TrimSpace(current.Workspace) == "" && strings.TrimSpace(current.Isolation) == "" {
		current.Isolation = "project"
	}
	if _, err := e.n.SafeApply(ctx, agent.ID, current); err != nil {
		return runner.AllocationResult{}, fmt.Errorf("persist mission resource allocation: %w", err)
	}

	project, err := st.GetProject(run.ProjectID)
	if err != nil {
		return runner.AllocationResult{}, err
	}
	var workspace string
	if run.Autonomous {
		workspace, err = e.n.resolveAutonomousExecutionWorkspace(ctx, project, agent, current)
	} else {
		workspace, err = e.n.resolveExecutionWorkspace(ctx, project, agent, current)
	}
	if err != nil {
		return runner.AllocationResult{}, err
	}
	if run.Autonomous && pkg.ContextCapsule == nil {
		if err := prepareAutonomousPackageWorkspace(ctx, run, pkg, project.CanonicalPath, workspace); err != nil {
			return runner.AllocationResult{}, fmt.Errorf("prepare autonomous package workspace: %w", err)
		}
	}
	return runner.AllocationResult{AgentID: agent.ID, Workspace: workspace}, nil
}

// buildMissionRoutingDecision projects the concrete allocation made by the
// existing resource scheduler into the durable, explainable routing contract.
// It intentionally does not mutate the desired affinity when a fallback is
// selected and leaves engine unset when the scheduler has no engine evidence.
func buildMissionRoutingDecision(pkg *runner.PackageRun, agent store.Agent, req TaskRequirements, selected ProviderAccount, current AgentConfig, desiredProvider, desiredProfile string, preferenceMode AffinityMode, isFailover bool, routingReason string, evidence ...agentSelectionEvidence) ExecutionRoutingDecision {
	modelPreference := AffinityPreference{Mode: AffinityAuto}
	if req.RuntimeAffinity != nil {
		modelPreference = req.RuntimeAffinity.Model
	}
	affinity := RuntimeAffinityPolicy{
		Provider: AffinityPreference{Mode: preferenceMode, Value: desiredProvider},
		Profile:  AffinityPreference{Mode: preferenceMode, Value: desiredProfile},
		Model:    modelPreference,
	}
	fallback := isFailover || (desiredProvider != "" && desiredProvider != selected.Provider) || (desiredProfile != "" && desiredProfile != selected.Profile)
	reason := firstNonEmpty(routingReason, "selected by existing Nexus resource scheduler")
	agentSelection := agentSelectionEvidence{Confidence: "UNKNOWN", Reason: "Agent selection evidence unavailable"}
	if len(evidence) > 0 {
		agentSelection = evidence[0]
	}
	switchReason := DeriveRoutingSwitchReason(fallback, reason, pkg.FailureKind, pkg.ErrorMessage)
	// Model-incapable fallback carries its own typed cause even when the
	// resource itself did not fail: preferred model unavailable/incapable.
	if switchReason == SwitchReasonPreferredUnavailable && modelPreference.Mode == AffinityPrefer && modelPreference.Value != "" && current.Model != modelPreference.Value {
		switchReason = SwitchReasonModelIncapable
	}
	rejected := []string{}
	if fallback && reason != "" {
		rejected = append(rejected, fmt.Sprintf("requested %s:%s rejected: %s [%s]",
			firstNonEmpty(desiredProvider, "(auto)"), firstNonEmpty(desiredProfile, "(auto)"), reason, switchReason))
	}
	return ExecutionRoutingDecision{
		TaskID:               pkg.PackageID,
		AgentID:              agent.ID,
		AgentScore:           agentSelection.Score,
		AgentConfidence:      agentSelection.Confidence,
		AgentReason:          agentSelection.Reason,
		Requirements:         req,
		Desired:              affinity,
		AffinityPolicy:       affinity,
		Actual:               ModelCandidate{Provider: selected.Provider, Profile: selected.Profile, Model: current.Model, Healthy: selected.Health == "healthy", Authenticated: selected.Authenticated, QuotaAvailable: selected.Available},
		SelectedProvider:     selected.Provider,
		SelectedProfile:      selected.Profile,
		SelectedAccountScope: selected.Scope,
		SelectedModel:        current.Model,
		SelectedReasoning:    reason,
		SkillRefs:            packageRunSkillIDs(pkg),
		MaestroGuidanceRef:   maestroGuidanceReference(req.Guidance),
		Fallback:             fallback,
		Reason:               reason,
		Rejected:             rejected,
		RejectedCandidates:   append([]string(nil), rejected...),
		TaskClass:            req.TaskKind,
		RequestedProvider:    desiredProvider,
		RequestedProfile:     desiredProfile,
		RequestedModel:       modelPreference.Value,
		SwitchReason:         switchReason,
		SwitchDetail:         reason,
		CreatedAt:            time.Now().UTC(),
	}
}

func maestroGuidanceReference(guidance *intelligence.ExecutionGuidance) string {
	if guidance == nil || !strings.EqualFold(strings.TrimSpace(guidance.Source), "maestro") {
		return ""
	}
	return strings.TrimSpace(guidance.Reference)
}

func missionAgentSpec(role string, requirements TaskRequirements) intelligence.AgentSpec {
	role = strings.TrimSpace(role)
	normalizedRole := normalizeAgentTaxonomy(role)
	spec := intelligence.AgentSpec{
		Role:               role,
		Responsibilities:   []string{fmt.Sprintf("Executar o workstream especializado %s dentro dos artefatos atribuídos", role)},
		Capabilities:       appendUniqueStrings(agentRequiredCapabilities(requirements.RequiredCapabilities), requirements.PreferredCapabilities...),
		Domains:            append([]string(nil), requirements.Domains...),
		Strengths:          append([]string(nil), requirements.DesiredStrengths...),
		Constraints:        append([]string(nil), requirements.Constraints...),
		VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
	}

	addInstruction := func(values ...string) {
		spec.Instructions = appendUniqueStrings(spec.Instructions, values...)
	}
	addResponsibility := func(values ...string) {
		spec.Responsibilities = appendUniqueStrings(spec.Responsibilities, values...)
	}
	addCapability := func(values ...string) {
		spec.Capabilities = appendUniqueStrings(spec.Capabilities, values...)
	}
	addConstraint := func(values ...string) {
		spec.Constraints = appendUniqueStrings(spec.Constraints, values...)
	}
	addStrength := func(values ...string) {
		spec.Strengths = appendUniqueStrings(spec.Strengths, values...)
	}

	switch normalizedRole {
	case "senior-developer":
		addInstruction("Understand system invariants and cross-cutting impact before changing code; prefer incremental, verifiable evolution.")
		addResponsibility("Executar mudanças complexas, orientar trade-offs e preservar compatibilidade entre áreas.")
		addCapability("implementation", "code_review", "architecture", "debugging")
		addStrength("systems-thinking", "refactoring", "risk-management")
	case "investigator":
		addInstruction(
			"Reproduza e isole a falha antes de propor correção.",
			"Colete evidência suficiente para sustentar a causa raiz e diferencie hipótese de fato.",
		)
		addResponsibility("Produzir hipótese, evidência, causa raiz e próximos passos reproduzíveis.")
		addCapability("debugging", "root_cause_analysis", "log_analysis")
		addConstraint("Investigue sem editar a implementação enquanto a causa raiz não estiver comprovada e a remediação não tiver sido atribuída.")
		addStrength("hypothesis-testing", "evidence-gathering")
		spec.VerificationPolicy.RequireTests = false
	case "architect":
		addInstruction("Modele limites, contratos, dependências e trade-offs antes de recomendar mudanças estruturais.")
		addResponsibility("Definir arquitetura evolutiva, registrar decisões e identificar riscos sistêmicos.")
		addCapability("architecture", "system_design", "tradeoff_analysis")
		addStrength("systems-thinking", "decision-records")
		spec.VerificationPolicy.RequireTests = false
	case "security-reviewer":
		addInstruction("Revise ameaças, fronteiras de confiança e regressões de segurança; ausência de evidência não é evidência de segurança.")
		addResponsibility("Identificar vulnerabilidades, severidade, evidência e mitigação recomendada.")
		addCapability("security", "code_review", "threat_modeling")
		addConstraint("Revisão é read-only; qualquer remediação deve ser atribuída a um workstream de implementação.")
		addStrength("threat-modeling", "risk-classification")
		spec.VerificationPolicy.RequireTests = false
	case "performance-investigator":
		addInstruction("Estabeleça baseline e reproduza o gargalo antes de otimizar; compare resultados depois da mudança.")
		addResponsibility("Measure bottlenecks, explain root cause, and define reproducible performance experiments.")
		addCapability("performance_analysis", "profiling", "benchmarking")
		addConstraint("Investigue sem editar a implementação até que o gargalo esteja medido e a remediação seja atribuída.")
		addStrength("measurement", "profiling")
		spec.VerificationPolicy.RequireTests = false
	case "reviewer":
		addInstruction("Procure riscos, regressões e claims sem evidência; classifique severidade objetivamente.")
		addResponsibility("Revisar diffs, compatibilidade, qualidade e evidência de verificação.")
		addCapability("code_review", "risk_analysis")
		addConstraint("Revisão é read-only; remediation pertence a um workstream de implementação.")
		spec.VerificationPolicy.RequireTests = false
	case "qa":
		addInstruction("Reproduza falhas, cubra regressões e exija evidência observável antes de concluir.")
		addResponsibility("Projetar e executar testes unitários, integração, E2E e smoke conforme o escopo.")
		addCapability("testing", "e2e", "regression_analysis")
		addStrength("reproduction", "regression-detection")
	case "devops":
		addInstruction("Prioritize reproducibility, safe rollback, and operational observability.")
		addCapability("ci_cd", "release", "operations")
	case "backend-engineer":
		addInstruction("Priorize contratos estáveis, validação de entrada e observabilidade no backend.")
	case "frontend-engineer":
		addInstruction("Preserve acessibilidade, responsividade, i18n e os tokens visuais existentes.")
	}
	return spec
}

func createMissionAgent(st *store.Store, projectID string, pkg *runner.PackageRun, prefix string) (store.Agent, error) {
	name := strings.TrimSpace(pkg.Title)
	if name == "" {
		name = pkg.PackageID
	}
	requirements := missionTaskRequirements(pkg)
	role := defaultRole(requirements.Role, defaultRole(pkg.Role, "implementer"))
	if len(requirements.PreferredRoles) > 0 {
		role = requirements.PreferredRoles[0]
	}
	spec := missionAgentSpec(role, requirements)
	agent, err := st.CreateAgent(store.Agent{ProjectID: projectID, Name: prefix + name, Role: role})
	if err != nil {
		return store.Agent{}, err
	}
	// Autonomous package Agents must execute in a dedicated worktree. Without
	// this explicit isolation, strict admission rejects the package before the
	// provider is ever dispatched, even when the project itself is worktree-safe.
	revision, err := st.AddRevision(agent.ID, (AgentConfig{Isolation: "worktree", AgentSpec: spec}).ConfigJSON())
	if err != nil {
		_ = st.DeleteAgent(agent.ID, projectID)
		return store.Agent{}, fmt.Errorf("persist AgentSpec revision: %w", err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		_ = st.DeleteAgent(agent.ID, projectID)
		return store.Agent{}, fmt.Errorf("link AgentSpec revision: %w", err)
	}
	return agent, nil
}

type agentSelectionEvidence struct {
	Score      float64
	Confidence string
	Reason     string
}

func (e *nexusPackageExecutor) selectReusableAgent(st *store.Store, run *runner.MissionRun, pkg *runner.PackageRun) (store.Agent, agentSelectionEvidence, error) {
	agents, err := st.ListAgents(run.ProjectID)
	if err != nil {
		return store.Agent{}, agentSelectionEvidence{}, err
	}
	reserved := reservedAgentsInRun(run, pkg.PackageID)
	candidates := make([]AgentMatchCandidate, 0, len(agents))
	for _, candidate := range agents {
		if _, used := reserved[candidate.ID]; used {
			continue
		}
		cfg, cfgErr := currentAgentConfig(st, candidate)
		if cfgErr != nil {
			continue
		}
		candidates = append(candidates, AgentMatchCandidate{Agent: candidate, Spec: cfg.AgentSpec})
	}
	requirements := missionTaskRequirements(pkg)
	// A package's legacy Role is often the generic "implementer" label. It
	// must not erase the richer classified roles persisted in TaskRequirements
	// (for example frontend-engineer for a React task).
	if len(requirements.PreferredRoles) == 0 && strings.TrimSpace(pkg.Role) != "" {
		role := strings.TrimSpace(pkg.Role)
		requirements.PreferredRoles = []string{role}
		requirements.AcceptableRoles = []string{role, "fullstack-engineer", "generalist"}
	}
	match := MatchAgents(candidates, AgentMatchRequirements{
		PreferredRoles:  requirements.PreferredRoles,
		AcceptableRoles: requirements.AcceptableRoles,
		Domains:         requirements.Domains,
		// headless/submit_prompt are provider-resource capabilities, not
		// persistent-Agent specializations. Passing them to the Agent matcher
		// would reject every legacy/custom Agent whose AgentSpec correctly
		// contains only behavioral capabilities.
		RequiredCapabilities:  agentRequiredCapabilities(requirements.RequiredCapabilities),
		PreferredCapabilities: requirements.PreferredCapabilities,
		DesiredStrengths:      requirements.DesiredStrengths,
		Constraints:           requirements.Constraints,
	})
	if match.Recommended == nil {
		return store.Agent{}, agentSelectionEvidence{}, store.ErrNotFound
	}
	return match.Recommended.Agent, agentSelectionEvidence{
		Score: match.Recommended.Score, Confidence: match.Recommended.Confidence, Reason: match.Explanation,
	}, nil
}

func agentRequiredCapabilities(capabilities []string) []string {
	result := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		switch normalizeAgentTaxonomy(capability) {
		case "headless", "submit-prompt":
			continue
		default:
			result = append(result, capability)
		}
	}
	return result
}

func reservedAgentsInRun(run *runner.MissionRun, exceptPackageID string) map[string]struct{} {
	reserved := make(map[string]struct{})
	if run == nil {
		return reserved
	}
	for _, pkg := range run.PackageRuns {
		if pkg.PackageID == exceptPackageID {
			continue
		}
		if pkg.AssignedAgent == "" {
			continue
		}
		switch pkg.State {
		case runner.StatePending, runner.StateVerified, runner.StateFailed, runner.StateCanceledByUser:
			continue
		default:
			reserved[pkg.AssignedAgent] = struct{}{}
		}
	}
	return reserved
}

func agentReservedInRun(run *runner.MissionRun, agentID, exceptPackageID string) bool {
	reserved := reservedAgentsInRun(run, exceptPackageID)
	_, ok := reserved[agentID]
	return ok
}

func flowResourcePolicy(raw string) SchedulerPolicy {
	switch SchedulerPolicy(strings.ToUpper(strings.TrimSpace(raw))) {
	case PolicyPreserveQuota:
		return PolicyPreserveQuota
	case PolicyPreferProvider:
		return PolicyPreferProvider
	case PolicyManual:
		return PolicyManual
	default:
		return PolicyBalanced
	}
}

func missionAffinityMode(policy SchedulerPolicy, desiredProvider, desiredProfile string) AffinityMode {
	if strings.TrimSpace(desiredProvider) == "" && strings.TrimSpace(desiredProfile) == "" {
		return AffinityAuto
	}
	if policy == PolicyManual {
		return AffinityPin
	}
	return AffinityPrefer
}

// selectMissionCandidateAccounts applies AUTO/PREFER/PIN before RecommendResources.
// PIN never opens the full pool on failover: a pinned runtime that fails stays blocked.
func selectMissionCandidateAccounts(accounts []ProviderAccount, pkg *runner.PackageRun, current AgentConfig, preferenceMode AffinityMode, isFailover bool) ([]ProviderAccount, error) {
	pinProvider := strings.TrimSpace(pkg.DesiredProvider)
	pinProfile := strings.TrimSpace(pkg.DesiredProfile)
	if pinProvider == "" {
		pinProvider = strings.TrimSpace(pkg.Provider)
	}
	if pinProfile == "" {
		pinProfile = strings.TrimSpace(pkg.Profile)
	}

	switch preferenceMode {
	case AffinityPin:
		if isFailover {
			return nil, fmt.Errorf("%w: pinned runtime %s:%s unavailable after quota/rate limit", ErrBlockedResource, pinProvider, pinProfile)
		}
		return filterFlowResourceAccounts(accounts, pinProvider, pinProfile), nil
	case AffinityPrefer:
		if !isFailover {
			preferred := filterFlowResourceAccounts(accounts, pinProvider, pinProfile)
			if len(preferred) > 0 {
				return preferred, nil
			}
			// Preferred identity unavailable on first attempt: open the pool for fallback.
			return accounts, nil
		}
		if current.Provider != "" {
			return filterOutFailingResource(accounts, current.Provider, current.Profile), nil
		}
		return accounts, nil
	default: // AUTO
		if !isFailover {
			return filterFlowResourceAccounts(accounts, pkg.Provider, pkg.Profile), nil
		}
		if current.Provider != "" {
			return filterOutFailingResource(accounts, current.Provider, current.Profile), nil
		}
		return accounts, nil
	}
}

func filterFlowResourceAccounts(accounts []ProviderAccount, provider, profile string) []ProviderAccount {
	provider, profile = strings.TrimSpace(provider), strings.TrimSpace(profile)
	if provider == "" && profile == "" {
		return accounts
	}
	out := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if provider != "" && account.Provider != provider {
			continue
		}
		if profile != "" && account.Profile != profile {
			continue
		}
		out = append(out, account)
	}
	return out
}

func filterOutFailingResource(accounts []ProviderAccount, failingProvider, failingProfile string) []ProviderAccount {
	out := make([]ProviderAccount, 0, len(accounts))
	for _, a := range accounts {
		if strings.EqualFold(a.Provider, failingProvider) && strings.EqualFold(a.Profile, failingProfile) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func explainNoEligibleMissionAccounts(accounts []ProviderAccount, pkgID string, isFailover bool) error {
	if len(accounts) == 0 {
		return fmt.Errorf("%w: no provider profiles registered for Flow Step %s", ErrAllocationUnregistered, pkgID)
	}
	unauth, unknown, exhausted, rateLimited, network := 0, 0, 0, 0, 0
	for _, acc := range accounts {
		switch {
		case !acc.Authenticated:
			unauth++
		case acc.NetworkError || (acc.QuotaView != nil && strings.EqualFold(strings.TrimSpace(acc.QuotaView.Status), "ERROR")):
			network++
		case acc.RateLimited:
			rateLimited++
		case (acc.QuotaView != nil && strings.EqualFold(strings.TrimSpace(acc.QuotaView.Status), "UNKNOWN")) ||
			(acc.AvailReasons != nil && acc.AvailReasons.UnknownQuota):
			unknown++
		case accountQuotaExhausted(acc):
			exhausted++
		}
	}
	n := len(accounts)
	switch {
	case unauth == n:
		return fmt.Errorf("%w: no authenticated provider profiles for Flow Step %s", ErrAllocationUnauthenticated, pkgID)
	case network == n:
		return fmt.Errorf("%w: provider network is unavailable for Flow Step %s", ErrAllocationNetwork, pkgID)
	case rateLimited == n:
		return fmt.Errorf("%w: all provider profiles are rate-limited for Flow Step %s", ErrAllocationRateLimited, pkgID)
	case exhausted == n:
		return fmt.Errorf("%w: all provider quotas are exhausted for Flow Step %s", ErrAllocationQuotaExhausted, pkgID)
	case unknown == n || (unauth+unknown == n && unknown > 0):
		return fmt.Errorf("%w: no provider profile with known quota for Flow Step %s", ErrAllocationUnknownQuota, pkgID)
	case isFailover:
		return fmt.Errorf("no eligible alternative provider profiles for Flow Step %s after quota/rate limit failure", pkgID)
	default:
		return fmt.Errorf("no eligible provider profiles for Flow Step %s (unauthenticated, unregistered, UNKNOWN, or rate-limited)", pkgID)
	}
}

func accountQuotaExhausted(acc ProviderAccount) bool {
	if acc.RateLimited || acc.NetworkError {
		return false
	}
	if acc.AvailReasons != nil && len(acc.AvailReasons.ExhaustedWindows) > 0 {
		return true
	}
	if acc.QuotaView != nil {
		for _, window := range acc.QuotaView.AllWindows() {
			if window.Kind != "unknown" && window.Remaining <= 0 &&
				strings.EqualFold(strings.TrimSpace(window.Status), "LIVE") {
				return true
			}
		}
	}
	return acc.QuotaTotal > 0 && acc.QuotaRemaining <= 0 && !acc.Available
}

func isQuotaOrRateLimitFailure(kind model.FailureKind) bool {
	return kind == model.FailureQuota || kind == model.FailureRateLimit
}

func (e *nexusPackageExecutor) Compile(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.PromptArtifact, error) {
	if pkg.ContextCapsule == nil {
		return runner.PromptArtifact{}, runner.ErrMissingContextCapsule
	}
	snapshot, err := e.n.loadMissionSnapshot(run)
	if err != nil {
		return runner.PromptArtifact{}, err
	}
	compiled, err := compilePackagePromptFromExecutionSnapshot(ctx, &snapshot.Plan, pkg.PhaseID, pkg.PackageID, snapshot.SkillContracts)
	if err != nil {
		return runner.PromptArtifact{}, err
	}
	if strings.TrimSpace(pkg.AssignedAgent) != "" {
		st, storeErr := e.n.OpenProject()
		if storeErr != nil {
			return runner.PromptArtifact{}, storeErr
		}
		agent, agentErr := st.GetAgent(pkg.AssignedAgent, run.ProjectID)
		if agentErr != nil {
			return runner.PromptArtifact{}, agentErr
		}
		cfg, cfgErr := currentAgentConfig(st, agent)
		if cfgErr != nil {
			return runner.PromptArtifact{}, cfgErr
		}
		for _, phase := range snapshot.Plan.Phases {
			for _, target := range phase.Packages {
				if target.ID == pkg.PackageID {
					compiledForAgent, compileErr := compileTargetPackagePromptForAgent(ctx, &snapshot.Plan, &target, cfg.AgentSpec, packageSkillIDs(target), snapshot.SkillContracts...)
					if compileErr != nil {
						return runner.PromptArtifact{}, compileErr
					}
					compiled = compiledForAgent
				}
			}
		}
	}
	content := strings.TrimSpace(compiled.SystemPrompt) + "\n\n" + strings.TrimSpace(compiled.UserPrompt)
	if strings.TrimSpace(content) == "" {
		return runner.PromptArtifact{}, fmt.Errorf("compiled prompt is empty")
	}
	if len(compiled.AcceptanceGates) > 0 {
		content += "\n\n## Acceptance gates\n- " + strings.Join(compiled.AcceptanceGates, "\n- ")
	}
	if len(pkg.RelevantPaths) > 0 {
		content += "\n\n## Relevant paths\n- " + strings.Join(pkg.RelevantPaths, "\n- ")
	}
	if len(pkg.VerificationRequirements) > 0 {
		content += "\n\n## Step verification requirements\n- " + strings.Join(pkg.VerificationRequirements, "\n- ")
	}
	// The typed ContextCapsule is persisted by MissionRunner before Compile.
	// Render only its bounded durable refs and WorkReceipts; never raw provider
	// output/transcripts. This is the actual provider handoff boundary.
	if capsule := strings.TrimSpace(runner.RenderContextCapsule(pkg.ContextCapsule)); capsule != "" {
		content += "\n\n" + capsule
	}
	if strings.TrimSpace(pkg.RemediationContext) != "" {
		content += "\n\n## Remediation evidence from the previous attempt\n" + strings.TrimSpace(pkg.RemediationContext) +
			"\n\nAddress the evidence above. Do not repeat the same failed approach without a concrete change."
	}
	content += autonomyPromptBoundaries(run.Contract)

	h := sha256.Sum256([]byte(content))
	created, err := e.n.st.CreatePromptVersion(store.PromptVersion{
		PlanID:       run.PlanID,
		PackageID:    pkg.PackageID,
		PlanRevision: run.PlanRevision,
		ContentHash:  hex.EncodeToString(h[:]),
		Content:      content,
	})
	if err != nil {
		return runner.PromptArtifact{}, fmt.Errorf("persist immutable prompt version: %w", err)
	}
	return runner.PromptArtifact{VersionID: created.ID, Content: content}, nil
}

func autonomyPromptBoundaries(contract runner.AutonomyContract) string {
	var rules []string
	if contract.DisallowDestructiveGit {
		rules = append(rules, "Do not use destructive git operations such as reset --hard, force push, or deleting branches.")
	}
	if !contract.AllowGitPush {
		rules = append(rules, "Do not push commits or branches to any remote.")
	}
	if !contract.AllowDeploy {
		rules = append(rules, "Do not deploy, publish releases, or modify external production systems.")
	}
	if len(contract.AllowedFilePatterns) > 0 {
		rules = append(rules, "Only modify files matching: "+strings.Join(contract.AllowedFilePatterns, ", "))
	}
	if len(rules) == 0 {
		return ""
	}
	return "\n\n## Autonomy boundaries\n- " + strings.Join(rules, "\n- ")
}

func (e *nexusPackageExecutor) Execute(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun, prompt string) (runner.ExecutionResult, error) {
	if strings.TrimSpace(pkg.AssignedAgent) == "" {
		return runner.ExecutionResult{}, fmt.Errorf("package has no assigned persistent agent")
	}
	result, err := e.n.executeAgentPrompt(ctx, pkg.AssignedAgent, pkg.Workspace, prompt, agentPromptPolicy{Contract: run.Contract, MissionRunID: run.ID, PackageID: pkg.PackageID})
	if err != nil {
		if result == nil {
			return runner.ExecutionResult{}, err
		}
		return runner.ExecutionResult{RuntimeID: result.RuntimeID, Output: result.Output, Failure: result.Failure}, err
	}
	if len(run.Contract.AllowedFilePatterns) > 0 {
		changed, guardErr := autonomyguard.GitChangedPaths(ctx, pkg.Workspace)
		if guardErr != nil {
			return runner.ExecutionResult{}, fmt.Errorf("enforce allowed file patterns: %w", guardErr)
		}
		if guardErr := autonomyguard.ValidateAllowedChanges(changed, run.Contract.AllowedFilePatterns); guardErr != nil {
			return runner.ExecutionResult{}, guardErr
		}
	}
	if pkg.Attempt > 1 && isQuotaOrRateLimitFailure(pkg.FailureKind) {
		events.DefaultBus().Publish(events.NewEventWithCorrelation(
			run.ID,
			run.ID,
			pkg.Provider,
			pkg.Profile,
			events.EventQuotaFailoverCompleted,
			fmt.Sprintf("Auto-failover confirmado para %s:%s após execução bem-sucedida no destino.", pkg.Provider, pkg.Profile),
			map[string]any{
				"package_id": pkg.PackageID,
				"provider":   pkg.Provider,
				"profile":    pkg.Profile,
				"attempt":    pkg.Attempt,
				"workspace":  pkg.Workspace,
				"runtime_id": result.RuntimeID,
			},
		))
	}
	return runner.ExecutionResult{RuntimeID: result.RuntimeID, Output: result.Output}, nil
}

func (e *nexusPackageExecutor) Review(ctx context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.ReviewVerdict, error) {
	st, err := e.n.OpenProject()
	if err != nil {
		return runner.ReviewVerdict{}, err
	}
	implementerID := pkg.AssignedAgent
	reviewWorkspace := pkg.Workspace
	if strings.EqualFold(strings.TrimSpace(pkg.PhaseType), "REVIEW") || strings.EqualFold(strings.TrimSpace(pkg.PhaseType), "SECURITY_REVIEW") {
		for _, dependencyID := range pkg.Dependencies {
			for i := range run.PackageRuns {
				dependency := &run.PackageRuns[i]
				if dependency.PackageID == dependencyID && dependency.AssignedAgent != "" {
					implementerID = dependency.AssignedAgent
					if dependency.Workspace != "" {
						reviewWorkspace = dependency.Workspace
					}
					break
				}
			}
			if implementerID != pkg.AssignedAgent {
				break
			}
		}
	}
	if implementerID == "" {
		return runner.ReviewVerdict{}, fmt.Errorf("cannot review package without implementer identity")
	}

	var reviewer store.Agent
	if (strings.EqualFold(strings.TrimSpace(pkg.PhaseType), "REVIEW") || strings.EqualFold(strings.TrimSpace(pkg.PhaseType), "SECURITY_REVIEW")) && pkg.AssignedAgent != "" {
		reviewer, err = st.GetAgent(pkg.AssignedAgent, run.ProjectID)
	} else {
		reviewer, err = e.ensureReviewerAgent(ctx, st, run, pkg, implementerID)
	}
	if err != nil {
		return runner.ReviewVerdict{}, err
	}

	reviewPrompt := buildReviewPrompt(pkg)
	before, _ := workspaceFingerprint(ctx, reviewWorkspace)
	result, err := e.n.executeAgentPrompt(ctx, reviewer.ID, reviewWorkspace, reviewPrompt, agentPromptPolicy{Contract: run.Contract, Review: true, MissionRunID: run.ID, PackageID: pkg.PackageID})
	if err != nil {
		return runner.ReviewVerdict{}, fmt.Errorf("reviewer execution failed: %w", err)
	}
	after, _ := workspaceFingerprint(ctx, reviewWorkspace)
	if before != "" && after != "" && before != after {
		return runner.ReviewVerdict{}, fmt.Errorf("independent reviewer modified the implementation workspace; review rejected")
	}

	verdict, err := parseReviewVerdict(result.Output)
	if err != nil {
		return runner.ReviewVerdict{}, err
	}
	verdict.ReviewerAgentID = reviewer.ID
	verdict.ReviewedAt = time.Now().UTC()
	verdict.ReviewIndependence = "DISTINCT_AGENT"
	if reviewer.ID == implementerID {
		verdict.ReviewIndependence = "DEGRADED_SELF_REVIEW"
	}
	return verdict, nil
}

func (e *nexusPackageExecutor) ensureReviewerAgent(ctx context.Context, st *store.Store, run *runner.MissionRun, pkg *runner.PackageRun, implementerID string) (store.Agent, error) {
	agents, err := st.ListAgents(run.ProjectID)
	if err != nil {
		return store.Agent{}, err
	}
	var reviewer store.Agent
	for _, candidate := range agents {
		if candidate.ID != implementerID && candidate.Role == "reviewer" && candidate.Name == "Nexus Independent Reviewer" {
			reviewer = candidate
			break
		}
	}
	if reviewer.ID == "" {
		reviewer, err = st.CreateAgent(store.Agent{ProjectID: run.ProjectID, Name: "Nexus Independent Reviewer", Role: "reviewer"})
		if err != nil {
			return store.Agent{}, err
		}
	}

	req := TaskRequirements{TaskKind: "review", Role: "reviewer", RequiredCapabilities: []string{"headless", "submit_prompt"}}
	accounts, err := e.n.ListResources()
	if err != nil {
		return store.Agent{}, err
	}
	reviewAccounts := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if !accountSupportsHeadlessReview(account) {
			continue
		}
		if ok, _ := providerSatisfiesAutonomyIsolation(account.Provider, run.Contract, true); ok {
			reviewAccounts = append(reviewAccounts, account)
		}
	}
	accounts = reviewAccounts

	// Independence means a distinct Agent identity and, whenever possible, a
	// distinct provider/profile from the implementer. Do not accidentally reward
	// same-provider affinity through the general scheduler for review work.
	implementerProvider := ""
	implementerProfile := ""
	if impl, getErr := st.GetAgent(implementerID, run.ProjectID); getErr == nil {
		if cfg, cfgErr := currentAgentConfig(st, impl); cfgErr == nil {
			implementerProvider = cfg.Provider
			implementerProfile = cfg.Profile
		}
	}
	if implementerProvider != "" {
		different := make([]ProviderAccount, 0, len(accounts))
		for _, account := range accounts {
			if account.Provider != implementerProvider || account.Profile != implementerProfile {
				different = append(different, account)
			}
		}
		if len(different) > 0 {
			accounts = different
		}
	}
	recommendation := RecommendResources(accounts, req, PolicyBalanced)
	if recommendation.Recommended == nil {
		return store.Agent{}, fmt.Errorf("no eligible reviewer provider/profile with verified read-only isolation")
	}
	selected := recommendation.Recommended.Account
	current, _ := currentAgentConfig(st, reviewer)
	current.AgentSpec = missionAgentSpec("reviewer", TaskRequirements{
		TaskKind:             "review",
		Role:                 "reviewer",
		PreferredRoles:       []string{"reviewer"},
		RequiredCapabilities: []string{"code_review"},
		Domains:              []string{"quality", "verification"},
	})
	current.Provider = selected.Provider
	current.Profile = selected.Profile
	// Review is executed against the implementer's concrete worktree and must
	// not create a second unrelated worktree.
	current.Workspace = pkg.Workspace
	current.Isolation = "project"
	if _, err := e.n.SafeApply(ctx, reviewer.ID, current); err != nil {
		return store.Agent{}, fmt.Errorf("configure independent reviewer: %w", err)
	}
	return reviewer, nil
}

// accountSupportsHeadlessReview returns true when the provider account
// advertises both "headless" and "submit_prompt" capabilities at SUPPORTED
// level. This replaces the old provider-name whitelist so that any driver
// (current or future) that declares these capabilities qualifies automatically.
func accountSupportsHeadlessReview(acc ProviderAccount) bool {
	required := []string{"headless", "submit_prompt"}
	for _, cap := range required {
		val := strings.ToLower(strings.TrimSpace(acc.Capabilities[cap]))
		if val != "supported" {
			return false
		}
	}
	return true
}

func missionTaskRequirements(pkg *runner.PackageRun) TaskRequirements {
	req := TaskRequirements{
		TaskKind:             "coding",
		Role:                 defaultRole(pkg.Role, "implementer"),
		RequiredCapabilities: []string{"headless", "submit_prompt"},
	}
	if strings.TrimSpace(pkg.TaskRequirements) != "" {
		var explicit TaskRequirements
		if json.Unmarshal([]byte(pkg.TaskRequirements), &explicit) == nil {
			if explicit.TaskKind != "" {
				req.TaskKind = explicit.TaskKind
			}
			if explicit.Role != "" {
				req.Role = explicit.Role
			}
			req.PreferredRoles = append([]string(nil), explicit.PreferredRoles...)
			req.AcceptableRoles = append([]string(nil), explicit.AcceptableRoles...)
			req.Domains = append([]string(nil), explicit.Domains...)
			req.EstimatedComplexity = explicit.EstimatedComplexity
			req.EstimatedTokens = explicit.EstimatedTokens
			req.RequiresDecomposition = explicit.RequiresDecomposition
			req.Confidence = explicit.Confidence
			req.Source = explicit.Source
			req.PreferProvider = explicit.PreferProvider
			req.AgentPreference = explicit.AgentPreference
			req.ProjectPolicy = explicit.ProjectPolicy
			req.RuntimeAffinity = explicit.RuntimeAffinity
			req.ModelCandidates = append([]ModelCandidate(nil), explicit.ModelCandidates...)
			req.EscalationLevel = explicit.EscalationLevel
			if explicit.Guidance != nil {
				guidance := *explicit.Guidance
				guidance.Instructions = append([]string(nil), explicit.Guidance.Instructions...)
				guidance.Skills = append([]string(nil), explicit.Guidance.Skills...)
				req.Guidance = &guidance
			}
			req.RequiredCapabilities = mergeRequiredCapabilities(req.RequiredCapabilities, explicit.RequiredCapabilities)
			req.PreferredCapabilities = append([]string(nil), explicit.PreferredCapabilities...)
			req.DesiredStrengths = append([]string(nil), explicit.DesiredStrengths...)
			req.Constraints = append([]string(nil), explicit.Constraints...)
		}
	}
	return req
}

// configuredModelCandidates binds declarative model metadata to the live
// account selected by the existing resource scheduler. Configuration can
// describe names, capabilities and cost/reasoning ranks, but it cannot assert
// health, authentication or quota; those fields always come from the account.
func configuredModelCandidates(cfg AgentConfig, account ProviderAccount, taskCandidates ...ModelCandidate) []ModelCandidate {
	configured := append([]ModelCandidate(nil), taskCandidates...)
	if len(configured) == 0 {
		configured = append(configured, cfg.ModelCandidates...)
	}
	result := make([]ModelCandidate, 0, len(configured))
	for _, candidate := range configured {
		if strings.TrimSpace(candidate.Model) == "" {
			continue
		}
		candidate.Provider = account.Provider
		candidate.Profile = account.Profile
		candidate.Healthy = strings.EqualFold(account.Health, "healthy") || strings.TrimSpace(account.Health) == ""
		candidate.Authenticated = account.Authenticated
		candidate.QuotaAvailable = account.Available && !account.RateLimited
		result = append(result, candidate)
	}
	return result
}

func mergeRequiredCapabilities(base, extra []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range append(append([]string(nil), base...), extra...) {
		key := normalizeCapabilityName(item)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func selectCurrentResource(accounts []ProviderAccount, cfg AgentConfig, req TaskRequirements, policy SchedulerPolicy) (ProviderAccount, bool) {
	if cfg.Provider == "" || cfg.Profile == "" {
		return ProviderAccount{}, false
	}
	for _, account := range accounts {
		if account.Provider != cfg.Provider || account.Profile != cfg.Profile {
			continue
		}
		candidate := evaluateCandidate(account, req, policy)
		return account, candidate.Eligible
	}
	return ProviderAccount{}, false
}

func currentAgentConfig(st *store.Store, agent store.Agent) (AgentConfig, error) {
	if agent.CurrentRevisionID == "" {
		return NormalizeAgentSpec(agent, AgentConfig{}), nil
	}
	rev, err := st.GetRevision(agent.CurrentRevisionID)
	if err != nil {
		return AgentConfig{}, err
	}
	cfg, err := ParseAgentConfig(rev.Config)
	if err != nil {
		return AgentConfig{}, err
	}
	return NormalizeAgentSpec(agent, cfg), nil
}

func defaultRole(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func buildReviewPrompt(pkg *runner.PackageRun) string {
	var evidence []string
	for _, v := range pkg.Verifications {
		evidence = append(evidence, fmt.Sprintf("- %s: passed=%t exit=%d\n%s", v.Command, v.Passed, v.ExitCode, v.OutputSnippet))
	}
	openFindings := make([]string, 0)
	for _, finding := range runner.OpenReviewFindings(pkg.ReviewFindings) {
		openFindings = append(openFindings, fmt.Sprintf("- %s [%s] %s (%s)", finding.ID, finding.Severity, finding.Description, finding.RequiredAction))
	}
	return fmt.Sprintf(`You are the independent code reviewer for an autonomous software-delivery mission.
Do NOT modify files, run destructive git commands, or approve based on assumptions.
Inspect the current workspace and implementation for the package below.

Package: %s
Goal: %s
Acceptance criteria:
- %s

Semantic handoff:
%s

Previously open findings (resolve only these or report new findings):
%s

Verification evidence:
%s

Return ONLY one valid JSON object. Use this example shape (replace values; do not emit the pipe characters):
{"approved":true,"findings":[],"structured_findings":[{"id":"","severity":"LOW","category":"","file":"","location":"","description":"","rationale":"","required_action":"","evidence":[]}],"remediation_tips":[]}
The approved field must be a JSON boolean. The severity field must be one of the strings HIGH, MEDIUM, or LOW.
Approve only when the implementation genuinely satisfies the goal and acceptance criteria and you found no blocking correctness, security, test, or integration issue.`, pkg.Title, pkg.Goal, strings.Join(pkg.AcceptanceCriteria, "\n- "), pkg.HandoffJSON, strings.Join(openFindings, "\n"), strings.Join(evidence, "\n"))
}

func parseReviewVerdict(output string) (runner.ReviewVerdict, error) {
	text := strings.TrimSpace(output)
	if text == "" {
		return runner.ReviewVerdict{}, fmt.Errorf("reviewer returned no output")
	}
	var lastErr error
	for start := strings.LastIndex(text, "{"); start >= 0; {
		var raw map[string]json.RawMessage
		decoder := json.NewDecoder(strings.NewReader(text[start:]))
		if err := decoder.Decode(&raw); err == nil {
			if _, hasApproved := raw["approved"]; hasApproved {
				payloadBytes, marshalErr := json.Marshal(raw)
				if marshalErr != nil {
					lastErr = marshalErr
				} else {
					var payload struct {
						Approved           bool                   `json:"approved"`
						Findings           []string               `json:"findings"`
						StructuredFindings []runner.ReviewFinding `json:"structured_findings"`
						RemediationTips    []string               `json:"remediation_tips"`
					}
					if err := json.Unmarshal(payloadBytes, &payload); err == nil {
						return runner.ReviewVerdict{Approved: payload.Approved, Findings: payload.Findings, StructuredFindings: payload.StructuredFindings, RemediationTips: payload.RemediationTips}, nil
					} else {
						lastErr = err
					}
				}
			}
		} else {
			lastErr = err
		}
		start = strings.LastIndex(text[:start], "{")
	}
	if lastErr == nil {
		return runner.ReviewVerdict{}, fmt.Errorf("reviewer did not return structured JSON evidence")
	}
	return runner.ReviewVerdict{}, fmt.Errorf("invalid reviewer JSON: %w", lastErr)
}
