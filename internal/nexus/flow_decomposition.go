package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// FlowDecompositionRequest defines the contract for decomposing a prompt or artifact into a Flow proposal (PLAN 04).
type FlowDecompositionRequest struct {
	ProjectID                   string                           `json:"project_id"`
	ArtifactID                  string                           `json:"artifact_id,omitempty"`
	Goal                        string                           `json:"goal"`
	SourcePrompt                string                           `json:"source_prompt,omitempty"`
	SkillIDs                    []string                         `json:"skill_ids,omitempty"`
	MaestroSkills               []string                         `json:"maestro_skills,omitempty"`
	Simple                      bool                             `json:"simple,omitempty"`
	DelegationMode              DelegationMode                   `json:"delegation_mode,omitempty"`
	MissionTeamPreset           MissionTeamPreset                `json:"mission_team_preset,omitempty"`
	MissionTeamRuntimeOverrides map[string]RuntimeAffinityPolicy `json:"mission_team_runtime_overrides,omitempty"`
}

// resolvedSkillIDs prefers the canonical skill_ids field and falls back to the
// legacy maestro_skills transport alias.
func (req FlowDecompositionRequest) resolvedSkillIDs() []string {
	if len(req.SkillIDs) > 0 {
		return append([]string(nil), req.SkillIDs...)
	}
	return append([]string(nil), req.MaestroSkills...)
}

// FlowDecompositionProposal represents an inspectable candidate flow DAG before final persistence (PLAN 04).
type FlowDecompositionProposal struct {
	Title                  string         `json:"title"`
	Description            string         `json:"description"`
	Archetype              string         `json:"archetype"`
	SourceArtifactID       string         `json:"source_artifact_id,omitempty"`
	SourceArtifactRevision int            `json:"source_artifact_revision,omitempty"`
	Flow                   FlowDefinition `json:"flow"`
	Reasoning              string         `json:"reasoning"`
	MaestroAdvice          string         `json:"maestro_advice,omitempty"`
}

// FlowPreflightCheck represents a single preflight verification item (PLAN 05).
type FlowPreflightCheck struct {
	Key     string `json:"key"` // "dag_validity", "agent_allocation", "resources", "worktree_isolation", "security"
	Label   string `json:"label"`
	Status  string `json:"status"` // "PASS", "WARN", "FAIL"
	Summary string `json:"summary"`
}

// FlowPreflightReport represents the full preflight report before starting a run (PLAN 05).
type FlowPreflightReport struct {
	PlanID      string               `json:"plan_id"`
	Revision    int                  `json:"revision"`
	Strict      bool                 `json:"strict,omitempty"`
	Ready       bool                 `json:"ready"`
	Checks      []FlowPreflightCheck `json:"checks"`
	GeneratedAt time.Time            `json:"generated_at"`
}

// DecomposePromptIntoFlowProposal creates an intelligent, honest DAG proposal from a prompt or goal.
// If the goal represents an atomic, simple task, it produces a clean 1-step Flow without artificial bloat.
func (n *Nexus) DecomposePromptIntoFlowProposal(ctx context.Context, req FlowDecompositionRequest) (*FlowDecompositionProposal, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	project, err := st.GetProject(req.ProjectID)
	if err != nil {
		return nil, err
	}
	goal := strings.TrimSpace(req.Goal)
	sourcePrompt := strings.TrimSpace(req.SourcePrompt)
	var sourceArtifactRevision int
	if req.ArtifactID != "" {
		artifact, err := st.GetPromptArtifact(req.ArtifactID)
		if err != nil {
			return nil, fmt.Errorf("resolve source prompt artifact: %w", err)
		}
		session, err := st.GetComposerSession(artifact.SessionID)
		if err != nil {
			return nil, fmt.Errorf("resolve source composer session: %w", err)
		}
		if session.ProjectID != req.ProjectID {
			return nil, fmt.Errorf("source prompt artifact belongs to another project")
		}
		sourceArtifactRevision = artifact.Version
		if sourcePrompt == "" {
			sourcePrompt = strings.TrimSpace(artifact.Content)
		}
		if goal == "" {
			goal = strings.TrimSpace(artifact.Content)
		}
	}

	planningText := strings.TrimSpace(strings.Join([]string{goal, sourcePrompt}, "\n"))

	// Automatic skills are selected only from the canonical Nexus catalog. This
	// happens before Flow construction so SkillIDs survive every WorkPlan/Flow/
	// Mission snapshot and the receiving provider never needs to rediscover them.
	if len(req.resolvedSkillIDs()) == 0 {
		if automaticSkills, skillErr := n.ResolveAutomaticSkillIDs(ctx, req.ProjectID, planningText); skillErr == nil {
			req.SkillIDs = automaticSkills
		}
	}

	// Provider intent is registration-owned: a provider name mentioned by the
	// user is actionable only when it already exists in the same profile registry
	// that backs Usage. Resource eligibility and quota balancing remain downstream.
	registeredProvider := ""
	if provider, ok, providerErr := n.ResolveRegisteredProviderPreference(planningText); providerErr != nil {
		return nil, fmt.Errorf("resolve registered provider preference: %w", providerErr)
	} else if ok {
		registeredProvider = provider
	}

	archetype := string(classifyPromptArchetype(planningText))
	delegation := DecideDelegation(planningText, req.DelegationMode)

	// Atomicity must be decided from the complete execution intent, not only
	// from a short display goal. A complete Composer artifact can have a concise
	// title while still containing independent backend/frontend/QA workstreams.
	lowerGoal := strings.ToLower(planningText)
	explicitAtomic := strings.Contains(lowerGoal, "fix typo") ||
		strings.Contains(lowerGoal, "ajuste de digitação") ||
		strings.Contains(lowerGoal, "quick fix") ||
		strings.Contains(lowerGoal, "simple fix")
	isAtomic := req.Simple || explicitAtomic ||
		(!delegation.Delegate &&
			len(strings.Fields(planningText)) <= 12 &&
			!strings.Contains(lowerGoal, "refactor") &&
			!strings.Contains(lowerGoal, "e2e") &&
			!strings.Contains(lowerGoal, "system"))

	title := firstNonEmpty(goal, "Flow Proposal")
	verification := projectVerificationCommands(project.CanonicalPath)
	phaseID := "phase_" + ids.NewRuntimeID()
	flow := FlowDefinition{
		ID:          "flow_draft_" + ids.NewRuntimeID(),
		ProjectID:   req.ProjectID,
		Title:       title,
		Description: fmt.Sprintf("Proposta de decomposição inteligente para: %s", title),
		Status:      "DRAFT",
		Revision:    1,
		Policy:      FlowGuided,
		Phases: []FlowPhase{
			{ID: phaseID, Title: "Execução", Order: 1},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if facts, factErr := PersistDelegationDecisionFacts(nil, delegation); factErr == nil {
		flow.StructuredFacts = facts
	}

	if isAtomic {
		skills := req.resolvedSkillIDs()
		flow.Steps = []FlowStep{
			{
				ID:                       "step_" + ids.NewRuntimeID(),
				PhaseID:                  phaseID,
				Order:                    1,
				Title:                    title,
				Goal:                     firstNonEmpty(sourcePrompt, goal),
				Priority:                 "HIGH",
				Status:                   "READY",
				Role:                     "implementer",
				TaskRequirements:         taskRequirementsJSON(firstNonEmpty(sourcePrompt, goal), "implementer"),
				AssignmentStrategy:       FlowAssignmentAuto,
				AcceptanceCriteria:       []string{"Ajuste concluído com validação dos testes"},
				VerificationRequirements: verification,
				SkillIDs:                 skills,
				MaestroSkills:            skills,
				CompiledPrompt:           sourcePrompt,
			},
		}
	} else if delegation.Delegate {
		flow.Steps = delegatedFlowSteps(delegation.Workstreams, phaseID, title, verification, req.resolvedSkillIDs())
	} else {
		// Multi-step structured decomposition (Implementer -> Tester/Reviewer)
		skills := req.resolvedSkillIDs()
		step1ID := "step_" + ids.NewRuntimeID()
		step2ID := "step_" + ids.NewRuntimeID()
		flow.Steps = []FlowStep{
			{
				ID:                       step1ID,
				PhaseID:                  phaseID,
				Order:                    1,
				Title:                    "Implementação · " + title,
				Goal:                     fmt.Sprintf("Executar implementação conforme objetivo: %s", goal),
				Priority:                 "HIGH",
				Status:                   "READY",
				Role:                     "implementer",
				TaskRequirements:         taskRequirementsJSON(goal, "implementer"),
				AssignmentStrategy:       FlowAssignmentAuto,
				AcceptanceCriteria:       []string{"Funcionalidade implementada sem regressões de build"},
				VerificationRequirements: verification,
				SkillIDs:                 skills,
				MaestroSkills:            skills,
				CompiledPrompt:           sourcePrompt,
			},
			{
				ID:                       step2ID,
				PhaseID:                  phaseID,
				Order:                    2,
				Title:                    "Verificação & Testes · " + title,
				Goal:                     "Verificar comportamento e suite de testes",
				Priority:                 "HIGH",
				Status:                   "PENDING",
				Role:                     "reviewer",
				TaskRequirements:         taskRequirementsJSON("Verificar comportamento e suite de testes", "reviewer"),
				AssignmentStrategy:       FlowAssignmentAuto,
				Dependencies:             []string{step1ID},
				AcceptanceCriteria:       []string{"Todos os testes passando e suite de qualidade aprovada"},
				VerificationRequirements: verification,
			},
		}
	}
	intent := DecideIntent(planningText, nil)
	if registeredProvider != "" {
		intent.TaskRequirements.PreferProvider = registeredProvider
		intent.Evidence = append(intent.Evidence, "registered-provider-request:"+registeredProvider)
	}
	if isAtomic {
		intent.Strategy = IntentDirect
		intent.Confidence = "HIGH"
		intent.RecommendedAction = "execute_single_work_unit"
	}
	selection := SelectMissionTeam(planningText, intent)
	if NormalizeDelegationMode(req.DelegationMode) == DelegationOff {
		selection = MissionTeamSelection{Version: MissionTeamVersion, Strategy: intent.Strategy, Direct: true, CostClass: "DIRECT", Reason: "delegation mode OFF keeps execution with the Lead", CreatedAt: time.Now().UTC()}
	}
	if strings.TrimSpace(string(req.MissionTeamPreset)) != "" {
		if selection.Direct {
			return nil, fmt.Errorf("mission_team_preset cannot be used for a direct task")
		}
		definition, presetErr := MissionTeamPresetDefinitionFor(req.MissionTeamPreset)
		if presetErr != nil {
			return nil, presetErr
		}
		selection.Preset = definition.Preset
		selection.CostClass = "EXPLICIT"
		selection.Reason = "explicit mission team preset override"
		selection.Signals = append(selection.Signals, "explicit_preset")
		selection.ReviewRequired = true
	}
	if facts, factErr := PersistIntentDecisionFacts(flow.StructuredFacts, intent); factErr != nil {
		return nil, factErr
	} else {
		flow.StructuredFacts = facts
	}
	if facts, factErr := PersistMissionTeamFacts(flow.StructuredFacts, selection); factErr != nil {
		return nil, factErr
	} else {
		flow.StructuredFacts = facts
	}
	if err := BuildMissionTeamFlow(&flow, selection, firstNonEmpty(sourcePrompt, goal), verification, req.resolvedSkillIDs(), delegation.Workstreams, req.MissionTeamRuntimeOverrides); err != nil {
		return nil, err
	}
	if registeredProvider != "" {
		applyRegisteredProviderPreferenceToFlow(&flow, registeredProvider)
	}

	proposal := &FlowDecompositionProposal{
		Title:                  title,
		Description:            flow.Description,
		Archetype:              archetype,
		SourceArtifactID:       req.ArtifactID,
		SourceArtifactRevision: sourceArtifactRevision,
		Flow:                   flow,
		Reasoning:              fmt.Sprintf("Decomposição baseada no arquétipo %s com %d passos.", archetype, len(flow.Steps)),
	}
	return proposal, nil
}

func delegatedFlowSteps(workstreams []DelegatedWorkstream, phaseID, title string, verification, skills []string) []FlowStep {
	steps := make([]FlowStep, 0, len(workstreams))
	stepIDs := make(map[string]string, len(workstreams))
	for _, workstream := range workstreams {
		stepIDs[workstream.ID] = "step_" + ids.NewRuntimeID()
	}
	for order, workstream := range workstreams {
		dependencies := make([]string, 0, len(workstream.Dependencies))
		for _, dependency := range workstream.Dependencies {
			if id := stepIDs[dependency]; id != "" {
				dependencies = append(dependencies, id)
			}
		}
		role := "implementer"
		if len(workstream.PreferredRoles) > 0 {
			role = workstream.PreferredRoles[0]
		}
		status := "READY"
		if len(dependencies) > 0 {
			status = "PENDING"
		}
		label := workstream.ID
		if len(label) > 0 {
			label = strings.ToUpper(label[:1]) + label[1:]
		}
		steps = append(steps, FlowStep{
			ID: stepIDs[workstream.ID], PhaseID: phaseID, Order: order + 1,
			Title: label + " · " + title, Goal: workstream.Goal,
			Priority: "HIGH", Status: status, Dependencies: dependencies,
			ParallelGroup: workstream.ParallelGroup, Role: role,
			TaskRequirements: taskRequirementsForWorkstream(workstream), AssignmentStrategy: FlowAssignmentAuto,
			AcceptanceCriteria: []string{"Workstream concluído sem regressões"}, VerificationRequirements: verification,
			SkillIDs: append([]string(nil), skills...), MaestroSkills: append([]string(nil), skills...),
			RelevantPaths: append([]string(nil), workstream.OwnedPaths...), SharedArtifacts: append([]string(nil), workstream.SharedContracts...),
		})
	}
	return steps
}

func projectVerificationCommands(root string) []string {
	commands := make([]string, 0, 2)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		commands = append(commands, "go test ./...")
	}
	if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		commands = append(commands, "npm test")
	}
	return commands
}

// PreflightFlow validates a Flow or WorkPlan before execution, generating an honest readiness report (PLAN 05).
func (n *Nexus) PreflightFlow(ctx context.Context, planID string) (*FlowPreflightReport, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}

	flow := FlowFromWorkPlan(*plan)
	report := &FlowPreflightReport{
		PlanID:      planID,
		Revision:    plan.CurrentRevision,
		Ready:       true,
		GeneratedAt: time.Now().UTC(),
		Checks:      []FlowPreflightCheck{},
	}

	// 1. DAG Validity
	if err := ValidateFlowDAG(flow); err != nil {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "dag_validity",
			Label:   "Validação do Grafo (DAG)",
			Status:  "FAIL",
			Summary: fmt.Sprintf("Ciclo ou dependência inválida detectada: %s", err.Error()),
		})
	} else {
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "dag_validity",
			Label:   "Validação do Grafo (DAG)",
			Status:  "PASS",
			Summary: fmt.Sprintf("Topologia válida sem ciclos (%d passos ordenados).", len(flow.Steps)),
		})
	}

	// 2. Resource availability check
	resources, resErr := n.ListResources()
	if resErr != nil || len(resources) == 0 {
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "resources",
			Label:   "Recursos & Provedores",
			Status:  "WARN",
			Summary: "Nenhum recurso de provedor autenticado detectado. Execução dependerá do ambiente padrão.",
		})
	} else {
		report.Checks = append(report.Checks, FlowPreflightCheck{
			Key:     "resources",
			Label:   "Recursos & Provedores",
			Status:  "PASS",
			Summary: fmt.Sprintf("%d provedores/perfis disponíveis no ambiente.", len(resources)),
		})
	}

	// 3. Worktree isolation check
	report.Checks = append(report.Checks, FlowPreflightCheck{
		Key:     "worktree_isolation",
		Label:   "Isolamento de Worktree Git",
		Status:  "WARN",
		Summary: "O preflight não consegue provar isolamento por step; a execução autônoma exige configuração explícita de worktree.",
	})

	// 4. Security & Autonomy permissions
	report.Checks = append(report.Checks, FlowPreflightCheck{
		Key:     "security",
		Label:   "Contrato de Autonomia & Segurança",
		Status:  "WARN",
		Summary: "O preflight confirma apenas o contrato declarado; enforcement adicional depende do runtime e não é simulado como PASS.",
	})

	return report, nil
}

// PreflightFlowStrict performs the admission checks required before an
// autonomous mission may dispatch work. PreflightFlow remains advisory for
// legacy callers and intentionally keeps WARN checks non-blocking.
func (n *Nexus) PreflightFlowStrict(ctx context.Context, planID string, autonomous bool) (*FlowPreflightReport, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	report, err := n.PreflightFlow(ctx, planID)
	if err != nil {
		return nil, err
	}
	report.Strict = true
	return n.strictAdmissionForPlan(ctx, *plan, report, autonomous)
}

func (n *Nexus) strictAdmissionForPlan(ctx context.Context, plan store.WorkPlan, report *FlowPreflightReport, autonomous bool) (*FlowPreflightReport, error) {
	if report == nil {
		return nil, fmt.Errorf("strict admission report is required")
	}
	// Admission checks only turn Ready off when a check fails. Callers may
	// revalidate an immutable snapshot with a fresh report, so initialize the
	// optimistic state here instead of relying on PreflightFlow's constructor.
	report.Ready = true
	if err := validateFlowExecutionContract(plan); err != nil {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "execution_contract", Label: "Contrato de Execução", Status: "FAIL", Summary: err.Error()})
	}
	for _, problem := range validatePlanReadiness(plan) {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "plan_readiness", Label: "Plan readiness", Status: "FAIL", Summary: problem})
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	projectRecord, err := st.GetProject(plan.ProjectID)
	if err != nil {
		return nil, err
	}
	if autonomous {
		worktreeProblems := make([]string, 0)
		if baseErr := validateWorktreeBase(ctx, projectRecord); baseErr != nil {
			worktreeProblems = append(worktreeProblems, baseErr.Error())
		}
		for _, phase := range plan.Phases {
			for _, pkg := range phase.Packages {
				strategy := strings.ToUpper(strings.TrimSpace(pkg.AssignmentStrategy))
				if strategy == "" && strings.TrimSpace(pkg.AgentAllocation) != "" {
					strategy = string(FlowAssignmentExisting)
				}
				if strategy != string(FlowAssignmentExisting) {
					continue
				}
				agentID := strings.TrimSpace(pkg.AgentAllocation)
				agent, agentErr := st.GetAgent(agentID, plan.ProjectID)
				if agentErr != nil {
					worktreeProblems = append(worktreeProblems, fmt.Sprintf("step %s pins unavailable Agent %s", pkg.ID, agentID))
					continue
				}
				cfg, cfgErr := currentAgentConfig(st, agent)
				if cfgErr != nil {
					worktreeProblems = append(worktreeProblems, fmt.Sprintf("step %s cannot resolve Agent %s isolation: %v", pkg.ID, agentID, cfgErr))
					continue
				}
				if !autonomousAgentUsesWorktree(projectRecord, cfg) {
					worktreeProblems = append(worktreeProblems, fmt.Sprintf("step %s pins Agent %s without effective Isolation=worktree", pkg.ID, agentID))
				}
			}
		}
		status := "PASS"
		summary := "AUTO/CREATE steps use worktree-safe Agent allocation; pinned Agents satisfy effective Isolation=worktree"
		if len(worktreeProblems) > 0 {
			status = "FAIL"
			summary = strings.Join(worktreeProblems, "; ")
			report.Ready = false
		}
		setFlowPreflightCheck(report, FlowPreflightCheck{
			Key:     "worktree_isolation",
			Label:   "Isolamento de Worktree Git",
			Status:  status,
			Summary: summary,
		})
	}
	resources, resourceErr := n.ListResources()
	if resourceErr != nil || len(resources) == 0 {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "resources", Label: "Recursos & Provedores", Status: "FAIL", Summary: "nenhum recurso de provedor disponível para admissão strict"})
	} else {
		for _, phase := range plan.Phases {
			for _, pkg := range phase.Packages {
				if strings.TrimSpace(pkg.Provider) == "" && strings.TrimSpace(pkg.Profile) == "" {
					continue
				}
				if len(filterFlowResourceAccounts(resources, pkg.Provider, pkg.Profile)) == 0 {
					report.Ready = false
					report.Checks = append(report.Checks, FlowPreflightCheck{Key: "resources", Label: "Recursos & Provedores", Status: "FAIL", Summary: fmt.Sprintf("nenhum recurso satisfaz as restrições do step %s", pkg.ID)})
				}
			}
		}
	}
	var skillErr error
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			requested := packageSkillIDs(pkg)
			_, skillErr = validateGenericSkillIDs(n, plan.ProjectID, requested)
			if skillErr != nil {
				break
			}
		}
		if skillErr != nil {
			break
		}
	}
	if skillErr != nil {
		report.Ready = false
		report.Checks = append(report.Checks, FlowPreflightCheck{Key: "skills", Label: "Nexus Skills", Status: "FAIL", Summary: skillErr.Error()})
	}
	return report, nil
}

func setFlowPreflightCheck(report *FlowPreflightReport, check FlowPreflightCheck) {
	if report == nil {
		return
	}
	for i := range report.Checks {
		if report.Checks[i].Key == check.Key {
			report.Checks[i] = check
			return
		}
	}
	report.Checks = append(report.Checks, check)
}

// validatePlanReadiness is deterministic admission for data that an LLM may
// omit. It runs after the compatibility façade has been loaded, so Flow
// layout/order cannot substitute for executable task semantics.
func validatePlanReadiness(plan store.WorkPlan) []string {
	problems := make([]string, 0)
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			prefix := fmt.Sprintf("flow step %s", pkg.ID)
			if strings.TrimSpace(pkg.Title) == "" {
				problems = append(problems, prefix+" is missing a title")
			}
			if strings.TrimSpace(pkg.Goal) == "" {
				problems = append(problems, prefix+" is missing an executable goal")
			}
			if len(nonEmptyStrings(pkg.AcceptanceCriteria)) == 0 {
				problems = append(problems, prefix+" is missing acceptance criteria / Definition of Done")
			}
			if len(nonEmptyStrings(pkg.VerificationRequirements)) == 0 {
				problems = append(problems, prefix+" is missing verification requirements")
			}
		}
	}
	return problems
}

func nonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
