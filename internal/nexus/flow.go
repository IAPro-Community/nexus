package nexus

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

const FlowPolicyFactKey = "nexus.flow_policy"

type FlowPolicy string

const (
	FlowGuided     FlowPolicy = "GUIDED"
	FlowAutonomous FlowPolicy = "AUTONOMOUS"
)

type FlowAssignmentStrategy string

const (
	FlowAssignmentExisting FlowAssignmentStrategy = "EXISTING"
	FlowAssignmentCreate   FlowAssignmentStrategy = "CREATE"
	FlowAssignmentAuto     FlowAssignmentStrategy = "AUTO"
)

type FlowPhase struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description,omitempty"`
	Order         int    `json:"order"`
	Type          string `json:"type,omitempty"`
	RequiredRole  string `json:"required_role,omitempty"`
	FailurePolicy string `json:"failure_policy,omitempty"`
}

type FlowStep struct {
	ID                       string                 `json:"id"`
	PhaseID                  string                 `json:"phase_id"`
	Order                    int                    `json:"order"`
	Title                    string                 `json:"title"`
	Goal                     string                 `json:"goal"`
	Priority                 string                 `json:"priority"`
	Status                   string                 `json:"status"`
	Dependencies             []string               `json:"dependencies"`
	ParallelGroup            string                 `json:"parallel_group,omitempty"`
	Role                     string                 `json:"role"`
	PhaseType                string                 `json:"phase_type,omitempty"`
	ReviewRequired           bool                   `json:"review_required,omitempty"`
	TaskRequirements         string                 `json:"task_requirements,omitempty"`
	AssignmentStrategy       FlowAssignmentStrategy `json:"assignment_strategy"`
	AgentID                  string                 `json:"agent_id,omitempty"`
	ResourcePolicy           string                 `json:"resource_policy,omitempty"`
	Provider                 string                 `json:"provider,omitempty"`
	Profile                  string                 `json:"profile,omitempty"`
	DesiredProvider          string                 `json:"desired_provider,omitempty"`
	DesiredProfile           string                 `json:"desired_profile,omitempty"`
	SkillIDs                 []string               `json:"skill_ids,omitempty"`
	MaestroGates             []string               `json:"maestro_gates,omitempty"`
	MaestroSkills            []string               `json:"maestro_skills,omitempty"`
	RelevantPaths            []string               `json:"relevant_paths,omitempty"`
	AcceptanceCriteria       []string               `json:"acceptance_criteria"`
	VerificationRequirements []string               `json:"verification_requirements,omitempty"`
	SharedArtifacts          []string               `json:"shared_artifacts,omitempty"`
	EntryCriteria            []string               `json:"entry_criteria,omitempty"`
	ExitCriteria             []string               `json:"exit_criteria,omitempty"`
	RequiredEvidence         []string               `json:"required_evidence,omitempty"`
	HandoffJSON              string                 `json:"handoff,omitempty"`
	CompiledPrompt           string                 `json:"compiled_prompt,omitempty"`
}

type FlowDefinition struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	MissionID       string            `json:"mission_id,omitempty"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Status          string            `json:"status"`
	Revision        int               `json:"revision"`
	Policy          FlowPolicy        `json:"policy"`
	PolicyStored    bool              `json:"-"`
	Phases          []FlowPhase       `json:"phases"`
	Steps           []FlowStep        `json:"steps"`
	StructuredFacts map[string]string `json:"structured_facts,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	return append([]string(nil), in...)
}

func cloneFacts(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func normalizeFlowPolicy(raw string) FlowPolicy {
	if strings.EqualFold(strings.TrimSpace(raw), string(FlowAutonomous)) {
		return FlowAutonomous
	}
	return FlowGuided
}

func assignmentForPackage(pkg store.WorkPackage) FlowAssignmentStrategy {
	switch strings.ToUpper(strings.TrimSpace(pkg.AssignmentStrategy)) {
	case string(FlowAssignmentExisting):
		return FlowAssignmentExisting
	case string(FlowAssignmentCreate):
		return FlowAssignmentCreate
	case string(FlowAssignmentAuto):
		return FlowAssignmentAuto
	}
	if strings.TrimSpace(pkg.AgentAllocation) != "" {
		return FlowAssignmentExisting
	}
	return FlowAssignmentAuto
}

func FlowFromWorkPlan(plan store.WorkPlan) FlowDefinition {
	facts := cloneFacts(plan.StructuredFacts)
	rawPolicy, stored := "", false
	if facts != nil {
		rawPolicy, stored = facts[FlowPolicyFactKey]
	}
	flow := FlowDefinition{
		ID: plan.ID, ProjectID: plan.ProjectID, MissionID: plan.MissionID,
		Title: plan.Title, Description: plan.Description, Status: plan.Status,
		Revision: plan.CurrentRevision, Policy: normalizeFlowPolicy(rawPolicy), PolicyStored: stored,
		StructuredFacts: facts, CreatedAt: plan.CreatedAt, UpdatedAt: plan.UpdatedAt,
	}
	for _, phase := range plan.Phases {
		flow.Phases = append(flow.Phases, FlowPhase{ID: phase.ID, Title: phase.Title, Description: phase.Description, Order: phase.Order, Type: phase.Type, RequiredRole: phase.RequiredRole, FailurePolicy: phase.FailurePolicy})
		for order, pkg := range phase.Packages {
			flow.Steps = append(flow.Steps, FlowStep{
				ID: pkg.ID, PhaseID: phase.ID, Order: order, Title: pkg.Title, Goal: pkg.Goal,
				Priority: pkg.Priority, Status: pkg.Status, Dependencies: cloneStrings(pkg.Dependencies), ParallelGroup: pkg.ParallelGroup,
				Role: pkg.Role, PhaseType: pkg.PhaseType, ReviewRequired: pkg.ReviewRequired, TaskRequirements: pkg.TaskRequirements, AssignmentStrategy: assignmentForPackage(pkg), AgentID: pkg.AgentAllocation,
				ResourcePolicy: pkg.ResourcePolicy, Provider: pkg.Provider, Profile: pkg.Profile, DesiredProvider: pkg.DesiredProvider, DesiredProfile: pkg.DesiredProfile,
				SkillIDs: cloneStrings(pkg.SkillIDs), MaestroGates: cloneStrings(pkg.MaestroGates), MaestroSkills: cloneStrings(pkg.MaestroSkills), RelevantPaths: cloneStrings(pkg.RelevantPaths),
				AcceptanceCriteria: cloneStrings(pkg.AcceptanceCriteria), VerificationRequirements: cloneStrings(pkg.VerificationRequirements),
				SharedArtifacts: cloneStrings(pkg.SharedArtifacts), EntryCriteria: cloneStrings(pkg.EntryCriteria), ExitCriteria: cloneStrings(pkg.ExitCriteria), RequiredEvidence: cloneStrings(pkg.RequiredEvidence), HandoffJSON: pkg.HandoffJSON, CompiledPrompt: pkg.CompiledPrompt,
			})
		}
	}
	return flow
}

func WorkPlanFromFlow(flow FlowDefinition) store.WorkPlan {
	facts := cloneFacts(flow.StructuredFacts)
	if facts == nil && (flow.PolicyStored || flow.Policy != FlowGuided) {
		facts = map[string]string{}
	}
	if flow.PolicyStored || flow.Policy != FlowGuided {
		facts[FlowPolicyFactKey] = string(normalizeFlowPolicy(string(flow.Policy)))
	} else if facts != nil {
		delete(facts, FlowPolicyFactKey)
	}
	phases := make([]store.PlanPhase, len(flow.Phases))
	phaseIndex := make(map[string]int, len(flow.Phases))
	for i, phase := range flow.Phases {
		phases[i] = store.PlanPhase{ID: phase.ID, Title: phase.Title, Description: phase.Description, Order: phase.Order, Type: phase.Type, RequiredRole: phase.RequiredRole, FailurePolicy: phase.FailurePolicy}
		phaseIndex[phase.ID] = i
	}
	steps := append([]FlowStep(nil), flow.Steps...)
	sort.SliceStable(steps, func(i, j int) bool {
		pi, pj := flowPhaseOrder(flow, steps[i].PhaseID), flowPhaseOrder(flow, steps[j].PhaseID)
		if pi != pj {
			return pi < pj
		}
		if steps[i].Order != steps[j].Order {
			return steps[i].Order < steps[j].Order
		}
		return steps[i].ID < steps[j].ID
	})
	for _, step := range steps {
		idx, ok := phaseIndex[step.PhaseID]
		if !ok {
			idx = len(phases)
			phaseIndex[step.PhaseID] = idx
			phases = append(phases, store.PlanPhase{ID: step.PhaseID, Title: "Flow", Order: len(phases) + 1})
		}
		phases[idx].Packages = append(phases[idx].Packages, store.WorkPackage{
			ID: step.ID, Title: step.Title, Goal: step.Goal, Priority: step.Priority, Status: step.Status,
			Dependencies: cloneStrings(step.Dependencies), ParallelGroup: step.ParallelGroup, Role: step.Role, PhaseType: step.PhaseType, ReviewRequired: step.ReviewRequired, TaskRequirements: step.TaskRequirements,
			AgentAllocation: step.AgentID, AssignmentStrategy: string(step.AssignmentStrategy), ResourcePolicy: step.ResourcePolicy,
			Provider: step.Provider, Profile: step.Profile, DesiredProvider: step.DesiredProvider, DesiredProfile: step.DesiredProfile, SkillIDs: cloneStrings(step.SkillIDs), MaestroGates: cloneStrings(step.MaestroGates), MaestroSkills: cloneStrings(step.MaestroSkills),
			RelevantPaths: cloneStrings(step.RelevantPaths), AcceptanceCriteria: cloneStrings(step.AcceptanceCriteria),
			VerificationRequirements: cloneStrings(step.VerificationRequirements), SharedArtifacts: cloneStrings(step.SharedArtifacts), EntryCriteria: cloneStrings(step.EntryCriteria), ExitCriteria: cloneStrings(step.ExitCriteria), RequiredEvidence: cloneStrings(step.RequiredEvidence), HandoffJSON: step.HandoffJSON, CompiledPrompt: step.CompiledPrompt,
		})
	}
	return store.WorkPlan{ID: flow.ID, ProjectID: flow.ProjectID, MissionID: flow.MissionID, Title: flow.Title, Description: flow.Description,
		Status: flow.Status, CurrentRevision: flow.Revision, Phases: phases, StructuredFacts: facts, CreatedAt: flow.CreatedAt, UpdatedAt: flow.UpdatedAt}
}

func flowPhaseOrder(flow FlowDefinition, phaseID string) int {
	for _, p := range flow.Phases {
		if p.ID == phaseID {
			return p.Order
		}
	}
	return 1 << 30
}

func stepLess(flow FlowDefinition, a, b FlowStep) bool {
	pa, pb := flowPhaseOrder(flow, a.PhaseID), flowPhaseOrder(flow, b.PhaseID)
	if pa != pb {
		return pa < pb
	}
	if a.Order != b.Order {
		return a.Order < b.Order
	}
	return a.ID < b.ID
}

func validateFlowReferences(flow FlowDefinition) (map[string]FlowStep, error) {
	phaseIDs := make(map[string]struct{}, len(flow.Phases))
	for _, phase := range flow.Phases {
		if strings.TrimSpace(phase.ID) == "" {
			return nil, fmt.Errorf("flow phase id is required")
		}
		if _, exists := phaseIDs[phase.ID]; exists {
			return nil, fmt.Errorf("duplicate flow phase %s", phase.ID)
		}
		phaseIDs[phase.ID] = struct{}{}
	}
	byID := make(map[string]FlowStep, len(flow.Steps))
	for _, step := range flow.Steps {
		if _, exists := phaseIDs[step.PhaseID]; !exists {
			return nil, fmt.Errorf("flow step %s references unknown phase %s", step.ID, step.PhaseID)
		}
		if strings.TrimSpace(step.ID) == "" {
			return nil, fmt.Errorf("flow step id is required")
		}
		if _, exists := byID[step.ID]; exists {
			return nil, fmt.Errorf("duplicate flow step %s", step.ID)
		}
		byID[step.ID] = step
	}
	for _, step := range flow.Steps {
		for _, dep := range step.Dependencies {
			if dep == step.ID {
				return nil, fmt.Errorf("flow step %s depends on itself", step.ID)
			}
			if _, exists := byID[dep]; !exists {
				return nil, fmt.Errorf("flow step %s depends on unknown step %s", step.ID, dep)
			}
		}
	}
	return byID, nil
}

func ValidateFlowDAG(flow FlowDefinition) error { _, err := TopologicalOrder(flow); return err }

func TopologicalOrder(flow FlowDefinition) ([]string, error) {
	byID, err := validateFlowReferences(flow)
	if err != nil {
		return nil, err
	}
	indegree := make(map[string]int, len(byID))
	children := make(map[string][]string, len(byID))
	for id := range byID {
		indegree[id] = 0
	}
	for _, step := range flow.Steps {
		for _, dep := range step.Dependencies {
			indegree[step.ID]++
			children[dep] = append(children[dep], step.ID)
		}
	}
	ready := make([]FlowStep, 0)
	for _, step := range flow.Steps {
		if indegree[step.ID] == 0 {
			ready = append(ready, step)
		}
	}
	sort.SliceStable(ready, func(i, j int) bool { return stepLess(flow, ready[i], ready[j]) })
	order := make([]string, 0, len(flow.Steps))
	for len(ready) > 0 {
		step := ready[0]
		ready = ready[1:]
		order = append(order, step.ID)
		ids := append([]string(nil), children[step.ID]...)
		sort.SliceStable(ids, func(i, j int) bool { return stepLess(flow, byID[ids[i]], byID[ids[j]]) })
		for _, child := range ids {
			indegree[child]--
			if indegree[child] == 0 {
				ready = append(ready, byID[child])
				sort.SliceStable(ready, func(i, j int) bool { return stepLess(flow, ready[i], ready[j]) })
			}
		}
	}
	if len(order) != len(flow.Steps) {
		return nil, fmt.Errorf("flow contains a dependency cycle")
	}
	return order, nil
}

func ExecutionWaves(flow FlowDefinition) ([][]string, error) {
	byID, err := validateFlowReferences(flow)
	if err != nil {
		return nil, err
	}
	remaining := make(map[string]int, len(byID))
	children := make(map[string][]string, len(byID))
	for _, step := range flow.Steps {
		remaining[step.ID] = len(step.Dependencies)
		for _, dep := range step.Dependencies {
			children[dep] = append(children[dep], step.ID)
		}
	}
	waves := [][]string{}
	for len(remaining) > 0 {
		ready := make([]FlowStep, 0)
		for id, count := range remaining {
			if count == 0 {
				ready = append(ready, byID[id])
			}
		}
		if len(ready) == 0 {
			return nil, fmt.Errorf("flow contains a dependency cycle")
		}
		sort.SliceStable(ready, func(i, j int) bool { return stepLess(flow, ready[i], ready[j]) })
		wave := make([]string, 0, len(ready))
		for _, step := range ready {
			wave = append(wave, step.ID)
			delete(remaining, step.ID)
		}
		for _, id := range wave {
			for _, child := range children[id] {
				if _, ok := remaining[child]; ok {
					remaining[child]--
				}
			}
		}
		waves = append(waves, wave)
	}
	return waves, nil
}

func DependentsOf(flow FlowDefinition, stepID string) []string {
	byID := make(map[string]FlowStep, len(flow.Steps))
	for _, s := range flow.Steps {
		byID[s.ID] = s
	}
	out := []string{}
	for _, step := range flow.Steps {
		for _, dep := range step.Dependencies {
			if dep == stepID {
				out = append(out, step.ID)
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return stepLess(flow, byID[out[i]], byID[out[j]]) })
	return out
}
