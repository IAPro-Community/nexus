package nexus

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

const (
	MissionTeamPresetFactKey    = "nexus.mission_team.preset"
	MissionTeamSelectionFactKey = "nexus.mission_team.selection"
	MissionTeamVersion          = 1
)

type MissionPhaseType string

const (
	PhaseDiagnosis      MissionPhaseType = "DIAGNOSIS"
	PhasePlanning       MissionPhaseType = "PLANNING"
	PhaseArchitecture   MissionPhaseType = "ARCHITECTURE"
	PhaseImplementation MissionPhaseType = "IMPLEMENTATION"
	PhaseReview         MissionPhaseType = "REVIEW"
	PhaseSecurityReview MissionPhaseType = "SECURITY_REVIEW"
	PhaseVerification   MissionPhaseType = "VERIFICATION"
)

type PhaseFailurePolicy string

const (
	FailureRetryFallback PhaseFailurePolicy = "RETRY_FALLBACK"
	FailureRemediate     PhaseFailurePolicy = "REMEDIATE"
	FailureBlock         PhaseFailurePolicy = "BLOCK"
)

// MissionPhase is a role/capability request, not a provider-specific Agent.
// Runtime affinity remains owned by the existing scheduler.
type MissionPhase struct {
	ID                    string                `json:"id"`
	Type                  MissionPhaseType      `json:"type"`
	RequiredRole          string                `json:"required_role"`
	AcceptableRoles       []string              `json:"acceptable_roles,omitempty"`
	RequiredCapabilities  []string              `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string              `json:"preferred_capabilities,omitempty"`
	Dependencies          []string              `json:"dependencies,omitempty"`
	EntryCriteria         []string              `json:"entry_criteria,omitempty"`
	ExitCriteria          []string              `json:"exit_criteria,omitempty"`
	RequiredEvidence      []string              `json:"required_evidence,omitempty"`
	ResourcePreference    RuntimeAffinityPolicy `json:"resource_preference,omitempty"`
	FailurePolicy         PhaseFailurePolicy    `json:"failure_policy"`
}

type MissionTeamPreset string

const (
	PresetFeatureDelivery    MissionTeamPreset = "FEATURE_DELIVERY"
	PresetBugFix             MissionTeamPreset = "BUG_FIX"
	PresetRefactor           MissionTeamPreset = "REFACTOR"
	PresetArchitectureChange MissionTeamPreset = "ARCHITECTURE_CHANGE"
	PresetReleaseCandidate   MissionTeamPreset = "RELEASE_CANDIDATE"
)

type MissionTeamSelection struct {
	Version        int               `json:"version"`
	Strategy       IntentStrategy    `json:"strategy"`
	Preset         MissionTeamPreset `json:"preset,omitempty"`
	Direct         bool              `json:"direct"`
	Reason         string            `json:"reason"`
	Signals        []string          `json:"signals,omitempty"`
	CostClass      string            `json:"cost_class"`
	ReviewRequired bool              `json:"review_required"`
	CreatedAt      time.Time         `json:"created_at"`
}

type PhaseHandoff struct {
	MissionObjective    string   `json:"mission_objective"`
	PhaseObjective      string   `json:"phase_objective"`
	PreviousPhaseResult string   `json:"previous_phase_result,omitempty"`
	RelevantDecisions   []string `json:"relevant_decisions,omitempty"`
	ChangedArtifacts    []string `json:"changed_artifacts,omitempty"`
	Evidence            []string `json:"evidence,omitempty"`
	KnownFailures       []string `json:"known_failures,omitempty"`
	RejectedApproaches  []string `json:"rejected_approaches,omitempty"`
	OpenRisks           []string `json:"open_risks,omitempty"`
	AcceptanceCriteria  []string `json:"acceptance_criteria,omitempty"`
	NextExpectedAction  string   `json:"next_expected_action"`
	ContextBudgetBytes  int      `json:"context_budget_bytes"`
}

type MissionTeamPresetDefinition struct {
	Preset      MissionTeamPreset `json:"preset"`
	Description string            `json:"description"`
	Phases      []MissionPhase    `json:"phases"`
}

func MissionTeamPresetDefinitionFor(preset MissionTeamPreset) (MissionTeamPresetDefinition, error) {
	role := func(name string, acceptable ...string) []string {
		if len(acceptable) == 0 {
			acceptable = []string{name}
		}
		return acceptable
	}
	phase := func(id string, kind MissionPhaseType, required string, deps ...string) MissionPhase {
		return MissionPhase{ID: id, Type: kind, RequiredRole: required, AcceptableRoles: role(required), Dependencies: deps, FailurePolicy: FailureRemediate}
	}

	var phases []MissionPhase
	description := ""
	switch MissionTeamPreset(strings.ToUpper(strings.TrimSpace(string(preset)))) {
	case PresetFeatureDelivery:
		description = "Planejamento, implementação, revisão e verificação de uma feature."
		phases = []MissionPhase{
			phase("planning", PhasePlanning, "architect"),
			phase("implementation", PhaseImplementation, "implementer", "planning"),
			phase("review", PhaseReview, "reviewer", "implementation"),
			phase("verification", PhaseVerification, "qa-engineer", "review"),
		}
	case PresetBugFix:
		description = "Diagnóstico, correção, revisão e verificação de um defeito."
		phases = []MissionPhase{
			phase("diagnosis", PhaseDiagnosis, "diagnostician"),
			phase("fix", PhaseImplementation, "implementer", "diagnosis"),
			phase("review", PhaseReview, "reviewer", "fix"),
			phase("verification", PhaseVerification, "qa-engineer", "review"),
		}
	case PresetRefactor:
		description = "Planejamento, refatoração, revisão e verificação de regressão."
		phases = []MissionPhase{
			phase("planning", PhasePlanning, "architect"),
			phase("refactor", PhaseImplementation, "implementer", "planning"),
			phase("review", PhaseReview, "reviewer", "refactor"),
			phase("verification", PhaseVerification, "qa-engineer", "review"),
		}
	case PresetArchitectureChange:
		description = "Arquitetura, implementação, revisão arquitetural e verificação."
		phases = []MissionPhase{
			phase("architecture", PhaseArchitecture, "architect"),
			phase("implementation", PhaseImplementation, "implementer", "architecture"),
			phase("review", PhaseReview, "reviewer", "implementation"),
			phase("verification", PhaseVerification, "qa-engineer", "review"),
		}
	case PresetReleaseCandidate:
		description = "Revisão, validação de segurança/qualidade e verificação de release."
		phases = []MissionPhase{
			phase("review", PhaseReview, "reviewer"),
			phase("security", PhaseSecurityReview, "security-reviewer", "review"),
			phase("verification", PhaseVerification, "verifier", "security"),
		}
	default:
		return MissionTeamPresetDefinition{}, fmt.Errorf("invalid mission team preset %q", preset)
	}
	for i := range phases {
		if phases[i].Type == PhaseVerification {
			phases[i].AcceptableRoles = []string{"qa-engineer", "verifier"}
		}
		phases[i].EntryCriteria = []string{"previous phase completed"}
		phases[i].ExitCriteria = []string{"phase receipt persisted", "required evidence recorded"}
		phases[i].RequiredEvidence = []string{"WORKUNIT_COMPLETED"}
		switch phases[i].Type {
		case PhasePlanning, PhaseArchitecture, PhaseDiagnosis:
			phases[i].RequiredEvidence = []string{"PLANNING_COMPLETED"}
		case PhaseImplementation:
			phases[i].RequiredEvidence = []string{"WORKUNIT_IMPLEMENTED"}
		}
		if phases[i].Type == PhaseReview || phases[i].Type == PhaseSecurityReview {
			phases[i].RequiredEvidence = []string{"REVIEW_COMPLETED"}
		}
		if phases[i].Type == PhaseVerification {
			phases[i].RequiredEvidence = []string{"VERIFICATION_PASSED"}
		}
	}
	return MissionTeamPresetDefinition{Preset: MissionTeamPreset(strings.ToUpper(string(preset))), Description: description, Phases: phases}, nil
}

func SelectMissionTeam(goal string, decision IntentDecision) MissionTeamSelection {
	text := strings.ToLower(strings.TrimSpace(goal))
	complexDefect := containsAny(text, "bug", "erro", "quebrado", "broken", "regressão", "regressao") && len(strings.Fields(text)) > 4
	selection := MissionTeamSelection{Version: MissionTeamVersion, Strategy: decision.Strategy, Direct: decision.Strategy == IntentDirect && !complexDefect, CostClass: "DIRECT", CreatedAt: time.Now().UTC()}
	if selection.Direct {
		selection.Reason = "intent decision classified the request as atomic/direct"
		return selection
	}
	choose := func(preset MissionTeamPreset, cost string, reason string, signals ...string) {
		selection.Preset, selection.CostClass, selection.Reason, selection.Signals, selection.ReviewRequired = preset, cost, reason, signals, true
	}
	switch {
	case containsAny(text, "release candidate", "candidato a release", "pronto para release"):
		choose(PresetReleaseCandidate, "HIGH", "release validation explicitly requested", "release")
	case containsAny(text, "arquitetura", "architectural", "architecture", "novo subsistema", "storage strategy"):
		choose(PresetArchitectureChange, "HIGH", "architecture-impacting change detected", "architecture")
	case containsAny(text, "refactor", "refatore", "refatoração", "refatoracao"):
		choose(PresetRefactor, "MEDIUM", "refactoring intent detected", "refactor")
	case containsAny(text, "bug", "erro", "quebrado", "broken", "regressão", "regressao"):
		choose(PresetBugFix, "MEDIUM", "defect-remediation intent detected", "defect")
	default:
		choose(PresetFeatureDelivery, "MEDIUM", "non-atomic implementation requires durable delivery gates", "implementation", "verification")
	}
	return selection
}

func PersistMissionTeamFacts(existing map[string]string, selection MissionTeamSelection) (map[string]string, error) {
	raw, err := json.Marshal(selection)
	if err != nil {
		return nil, fmt.Errorf("encode mission team selection: %w", err)
	}
	facts := cloneFacts(existing)
	if facts == nil {
		facts = map[string]string{}
	}
	facts[MissionTeamSelectionFactKey] = string(raw)
	if selection.Preset != "" {
		facts[MissionTeamPresetFactKey] = string(selection.Preset)
	} else {
		delete(facts, MissionTeamPresetFactKey)
	}
	return facts, nil
}

func ReadMissionTeamSelectionFacts(facts map[string]string) (MissionTeamSelection, error) {
	selection := MissionTeamSelection{}
	raw := strings.TrimSpace(facts[MissionTeamSelectionFactKey])
	if raw == "" {
		return selection, fmt.Errorf("mission team selection is not persisted")
	}
	if err := json.Unmarshal([]byte(raw), &selection); err != nil {
		return MissionTeamSelection{}, fmt.Errorf("decode mission team selection: %w", err)
	}
	if selection.Direct {
		return selection, nil
	}
	if _, err := MissionTeamPresetDefinitionFor(selection.Preset); err != nil {
		return MissionTeamSelection{}, err
	}
	return selection, nil
}

// BuildMissionTeamFlow projects a preset onto the existing Flow/WorkPlan DAG.
// It does not allocate Agents or providers; allocation remains downstream.
func BuildMissionTeamFlow(flow *FlowDefinition, selection MissionTeamSelection, goal string, verification, skills []string, workstreams []DelegatedWorkstream, runtimeOverrides ...map[string]RuntimeAffinityPolicy) error {
	if flow == nil {
		return fmt.Errorf("flow is required")
	}
	if selection.Direct || selection.Preset == "" {
		return nil
	}
	definition, err := MissionTeamPresetDefinitionFor(selection.Preset)
	if err != nil {
		return err
	}
	if facts, factErr := PersistMissionTeamFacts(flow.StructuredFacts, selection); factErr == nil {
		flow.StructuredFacts = facts
	} else {
		return factErr
	}
	flow.Phases = make([]FlowPhase, 0, len(definition.Phases))
	flow.Steps = []FlowStep{}
	phaseIDs := make(map[string]string, len(definition.Phases))
	for order, phase := range definition.Phases {
		phaseID := "phase_" + phase.ID + "_" + ids.NewRuntimeID()
		phaseIDs[phase.ID] = phaseID
		flow.Phases = append(flow.Phases, FlowPhase{ID: phaseID, Title: humanPhaseTitle(phase.Type), Description: phase.RequiredRole, Order: order + 1, Type: string(phase.Type), RequiredRole: phase.RequiredRole, FailurePolicy: string(phase.FailurePolicy)})
	}
	phaseSteps := make(map[string][]string, len(definition.Phases))
	for _, phase := range definition.Phases {
		phase.ResourcePreference = applyMissionTeamRuntimeOverride(phase, firstRuntimeOverride(runtimeOverrides))
		steps := []FlowStep{}
		implementationStreams := implementationWorkstreams(workstreams)
		if phase.Type == PhaseImplementation && len(implementationStreams) > 0 {
			for index, stream := range implementationStreams {
				steps = append(steps, missionTeamStep(flow, phase, phaseIDs[phase.ID], goal, stream.Goal, verification, skills, phaseDependencyStepIDs(phase, phaseSteps), stream.OwnedPaths, stream.SharedContracts, index))
			}
		} else {
			phaseGoal := goal
			if phase.Type == PhaseReview || phase.Type == PhaseSecurityReview {
				phaseGoal = "Reviewar a implementação contra o objetivo, contratos e evidências."
			}
			if phase.Type == PhaseVerification {
				phaseGoal = "Executar o plano de verificação e confirmar a Definition of Done."
			}
			steps = append(steps, missionTeamStep(flow, phase, phaseIDs[phase.ID], goal, phaseGoal, verification, skills, phaseDependencyStepIDs(phase, phaseSteps), nil, nil, 0))
		}
		phaseSteps[phase.ID] = make([]string, 0, len(steps))
		for _, step := range steps {
			phaseSteps[phase.ID] = append(phaseSteps[phase.ID], step.ID)
			flow.Steps = append(flow.Steps, step)
		}
	}
	return nil
}

func firstRuntimeOverride(overrides []map[string]RuntimeAffinityPolicy) map[string]RuntimeAffinityPolicy {
	if len(overrides) == 0 {
		return nil
	}
	return overrides[0]
}

func applyMissionTeamRuntimeOverride(phase MissionPhase, overrides map[string]RuntimeAffinityPolicy) RuntimeAffinityPolicy {
	policy := phase.ResourcePreference
	for _, key := range []string{phase.ID, string(phase.Type), strings.ToLower(phase.ID), strings.ToLower(string(phase.Type))} {
		if override, ok := overrides[key]; ok {
			policy = mergeRuntimeAffinity(policy, override)
			break
		}
	}
	return policy
}

func mergeRuntimeAffinity(base, override RuntimeAffinityPolicy) RuntimeAffinityPolicy {
	merge := func(current, requested AffinityPreference) AffinityPreference {
		if requested.Mode == "" && strings.TrimSpace(requested.Value) == "" {
			return current
		}
		return requested
	}
	return RuntimeAffinityPolicy{
		Engine: merge(base.Engine, override.Engine), Provider: merge(base.Provider, override.Provider),
		Profile: merge(base.Profile, override.Profile), Model: merge(base.Model, override.Model),
	}
}

func implementationWorkstreams(workstreams []DelegatedWorkstream) []DelegatedWorkstream {
	out := make([]DelegatedWorkstream, 0, len(workstreams))
	for _, stream := range workstreams {
		if strings.EqualFold(strings.TrimSpace(stream.ID), "qa") || containsNormalized(stream.PreferredRoles, "qa-engineer") || containsNormalized(stream.PreferredRoles, "reviewer") {
			continue
		}
		out = append(out, stream)
	}
	return out
}

func missionTeamStep(flow *FlowDefinition, phase MissionPhase, phaseID, missionGoal, phaseGoal string, verification, skills, dependencies, ownedPaths, sharedArtifacts []string, order int) FlowStep {
	if len(dependencies) > 0 {
		dependencies = append([]string(nil), dependencies...)
	}
	stepID := "step_" + phase.ID + "_" + ids.NewRuntimeID()
	reviewRequired := phase.Type == PhaseReview || phase.Type == PhaseSecurityReview
	acceptance := append([]string{}, phase.ExitCriteria...)
	if len(acceptance) == 0 {
		acceptance = []string{"phase exit criteria satisfied"}
	}
	handoff, _ := json.Marshal(PhaseHandoff{MissionObjective: missionGoal, PhaseObjective: phaseGoal, RelevantDecisions: []string{"team_preset=" + string(flowPolicyPreset(flow)), "required_role=" + phase.RequiredRole}, AcceptanceCriteria: acceptance, NextExpectedAction: "continue to next dependency-ready phase", ContextBudgetBytes: 32768})
	return FlowStep{ID: stepID, PhaseID: phaseID, Order: order + 1, Title: humanPhaseTitle(phase.Type), Goal: phaseGoal, Priority: "HIGH", Status: statusForDependencies(dependencies), Dependencies: dependencies, ParallelGroup: implementationParallelGroup(phase.Type), Role: phase.RequiredRole, PhaseType: string(phase.Type), ReviewRequired: reviewRequired, TaskRequirements: missionTaskRequirementsJSON(phaseGoal, phase), AssignmentStrategy: FlowAssignmentAuto, AcceptanceCriteria: acceptance, VerificationRequirements: append([]string(nil), verification...), RequiredEvidence: append([]string(nil), phase.RequiredEvidence...), EntryCriteria: append([]string(nil), phase.EntryCriteria...), ExitCriteria: append([]string(nil), phase.ExitCriteria...), HandoffJSON: string(handoff), SkillIDs: append([]string(nil), skills...), MaestroSkills: append([]string(nil), skills...), RelevantPaths: append([]string(nil), ownedPaths...), SharedArtifacts: append([]string(nil), sharedArtifacts...), DesiredProvider: affinityValue(phase.ResourcePreference.Provider), DesiredProfile: affinityValue(phase.ResourcePreference.Profile)}
}

func missionTaskRequirementsJSON(prompt string, phase MissionPhase) string {
	req := ClassifyTaskRequirements(prompt)
	req.Role = phase.RequiredRole
	req.PreferredRoles = append([]string{phase.RequiredRole}, phase.AcceptableRoles...)
	req.AcceptableRoles = append([]string{phase.RequiredRole}, phase.AcceptableRoles...)
	req.RequiredCapabilities = appendMissionStrings(req.RequiredCapabilities, phase.RequiredCapabilities...)
	req.PreferredCapabilities = appendMissionStrings(req.PreferredCapabilities, phase.PreferredCapabilities...)
	if !isZeroRuntimeAffinity(phase.ResourcePreference) {
		req.RuntimeAffinity = &phase.ResourcePreference
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func isZeroRuntimeAffinity(policy RuntimeAffinityPolicy) bool {
	return policy.Engine.Mode == "" && policy.Engine.Value == "" && policy.Provider.Mode == "" && policy.Provider.Value == "" && policy.Profile.Mode == "" && policy.Profile.Value == "" && policy.Model.Mode == "" && policy.Model.Value == ""
}

func affinityValue(preference AffinityPreference) string {
	if preference.Mode == AffinityPrefer || preference.Mode == AffinityPin {
		return strings.TrimSpace(preference.Value)
	}
	return ""
}

func appendMissionStrings(base []string, values ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(values))
	out := make([]string, 0, len(base)+len(values))
	for _, value := range append(append([]string(nil), base...), values...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func phaseDependencyStepIDs(phase MissionPhase, previous map[string][]string) []string {
	var out []string
	for _, dependency := range phase.Dependencies {
		out = append(out, previous[dependency]...)
	}
	return out
}

func statusForDependencies(dependencies []string) string {
	if len(dependencies) == 0 {
		return "READY"
	}
	return "PENDING"
}

func implementationParallelGroup(kind MissionPhaseType) string {
	if kind == PhaseImplementation {
		return "implementation"
	}
	return ""
}

func humanPhaseTitle(kind MissionPhaseType) string {
	switch kind {
	case PhaseDiagnosis:
		return "Diagnosis"
	case PhasePlanning:
		return "Planning"
	case PhaseArchitecture:
		return "Architecture"
	case PhaseImplementation:
		return "Implementation"
	case PhaseReview:
		return "Review"
	case PhaseSecurityReview:
		return "Security Review"
	case PhaseVerification:
		return "Verification"
	default:
		return string(kind)
	}
}

func flowPolicyPreset(flow *FlowDefinition) MissionTeamPreset {
	if flow == nil {
		return ""
	}
	selection, err := ReadMissionTeamSelectionFacts(flow.StructuredFacts)
	if err != nil {
		return ""
	}
	return selection.Preset
}
