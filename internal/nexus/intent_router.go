package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
)

const (
	// IntentDecisionFactKey is a reserved structured-facts key in WorkPlan.
	// Keeping the decision in the existing plan metadata avoids a second store.
	IntentDecisionFactKey   = "nexus.intent_decision"
	IntentStrategyFactKey   = "nexus.intent_strategy"
	IntentConfidenceFactKey = "nexus.intent_confidence"
)

type IntentStrategy string

const (
	IntentDirect  IntentStrategy = "DIRECT"
	IntentClarify IntentStrategy = "CLARIFY"
	IntentPlan    IntentStrategy = "PLAN"
)

// IntentDecision is the bounded, explainable decision made before execution.
// It records what was known from the repository separately from assumptions
// and questions that still require a human decision.
type IntentDecision struct {
	Strategy             IntentStrategy                       `json:"strategy"`
	Confidence           string                               `json:"confidence"`
	KnownFacts           []string                             `json:"known_facts,omitempty"`
	Assumptions          []string                             `json:"assumptions,omitempty"`
	Unknowns             []string                             `json:"unknowns,omitempty"`
	BlockingQuestions    []string                             `json:"blocking_questions,omitempty"`
	Evidence             []string                             `json:"evidence,omitempty"`
	RecommendedAction    string                               `json:"recommended_next_action"`
	TaskRequirements     TaskRequirements                     `json:"task_requirements"`
	SnapshotIdentity     string                               `json:"snapshot_identity,omitempty"`
	SnapshotCompleteness contextsnapshot.SnapshotCompleteness `json:"snapshot_completeness,omitempty"`
}

// PersistIntentDecisionFacts records the bounded intent decision in the
// existing WorkPlan structured-facts map. It preserves caller-owned facts and
// stores the exact decision used for routing, rather than asking a provider to
// reconstruct it later.
func PersistIntentDecisionFacts(existing map[string]string, decision IntentDecision) (map[string]string, error) {
	raw, err := json.Marshal(decision)
	if err != nil {
		return nil, fmt.Errorf("encode intent decision: %w", err)
	}
	facts := make(map[string]string, len(existing)+3)
	for key, value := range existing {
		facts[key] = value
	}
	facts[IntentDecisionFactKey] = string(raw)
	facts[IntentStrategyFactKey] = string(decision.Strategy)
	facts[IntentConfidenceFactKey] = decision.Confidence
	return facts, nil
}

// DecideIntent applies the autonomy boundary without asking for facts that
// Project Intelligence already observed. It is deterministic and provider-
// independent; an intelligence provider may enrich the result later.
func DecideIntent(goal string, snapshot *contextsnapshot.ProjectContextSnapshot) IntentDecision {
	goal = strings.TrimSpace(goal)
	decision := IntentDecision{
		Strategy:          IntentPlan,
		Confidence:        "MEDIUM",
		TaskRequirements:  ClassifyTaskRequirements(goal),
		RecommendedAction: "create_work_plan",
	}
	if snapshot != nil {
		decision.SnapshotIdentity = snapshot.Identity.IdentityDigest
		decision.SnapshotCompleteness = snapshot.Completeness
		for _, fact := range snapshot.Facts {
			if len(decision.KnownFacts) >= 32 {
				break
			}
			decision.KnownFacts = append(decision.KnownFacts, fact.Key)
			if len(fact.Provenance) > 0 && len(decision.Evidence) < 32 {
				decision.Evidence = append(decision.Evidence, fact.Provenance[0].SourcePath+":"+fact.Key)
			}
		}
		if snapshot.Completeness == contextsnapshot.CompletenessPartial {
			decision.Assumptions = append(decision.Assumptions, "project intelligence is partial and bounded")
		}
	}

	if goal == "" {
		decision.Strategy = IntentClarify
		decision.Confidence = "HIGH"
		decision.Unknowns = append(decision.Unknowns, "objective")
		decision.BlockingQuestions = append(decision.BlockingQuestions, "What concrete outcome should Nexus deliver?")
		decision.RecommendedAction = "request_material_clarification"
		return decision
	}

	lower := strings.ToLower(goal)
	if isDirectIntent(lower) {
		decision.Strategy = IntentDirect
		decision.Confidence = "HIGH"
		decision.RecommendedAction = "execute_single_work_unit"
		decision.Evidence = append(decision.Evidence, "atomic-action-language")
		return decision
	}
	if hasConflictingProductBehavior(lower) {
		decision.Strategy = IntentClarify
		decision.Confidence = "HIGH"
		decision.Unknowns = append(decision.Unknowns, "conflicting_product_behavior")
		decision.BlockingQuestions = append(decision.BlockingQuestions, "Which of the conflicting behaviors should win?")
		decision.RecommendedAction = "request_material_clarification"
		return decision
	}
	if isMateriallyUnderspecified(lower, snapshot) {
		decision.Strategy = IntentClarify
		decision.Confidence = "MEDIUM"
		decision.Unknowns = append(decision.Unknowns, "material_scope_or_acceptance")
		decision.BlockingQuestions = append(decision.BlockingQuestions, "What scope and acceptance outcome should define success?")
		decision.RecommendedAction = "request_material_clarification"
		return decision
	}
	if decision.TaskRequirements.RequiresDecomposition || hasPlanSignals(lower) {
		decision.Strategy = IntentPlan
		decision.Confidence = "HIGH"
		decision.RecommendedAction = "create_work_plan"
		return decision
	}
	decision.Strategy = IntentPlan
	decision.RecommendedAction = "create_work_plan"
	return decision
}

func (n *Nexus) DecideIntentForProject(ctx context.Context, projectID, goal string) (IntentDecision, error) {
	if err := ctx.Err(); err != nil {
		return IntentDecision{}, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return IntentDecision{}, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return IntentDecision{}, err
	}
	snapshot := (*contextsnapshot.ProjectContextSnapshot)(nil)
	if view, viewErr := n.GetProjectIntelligence(ctx, projectID); viewErr == nil && view.CurrentSnapshot != nil {
		snapshot = view.CurrentSnapshot
	} else if discovered, discoverErr := contextsnapshot.Discover(project.CanonicalPath, contextsnapshot.Metadata{ProjectID: projectID}); discoverErr == nil {
		snapshot = &discovered
	}
	decision := DecideIntent(goal, snapshot)
	// Provider names are actionable only when they already exist in Nexus'
	// registered profile authority (the same source used by Usage/ListResources).
	// Never mint routing candidates from PATH discovery or arbitrary prompt text.
	if provider, ok, providerErr := n.ResolveRegisteredProviderPreference(goal); providerErr == nil && ok {
		decision.TaskRequirements.PreferProvider = provider
		decision.Evidence = append(decision.Evidence, "registered-provider-request:"+provider)
	}
	if decision.Strategy == IntentClarify && len(decision.BlockingQuestions) == 0 {
		return IntentDecision{}, fmt.Errorf("intent clarification has no blocking question")
	}
	return decision, nil
}

// ComposerIsRequired reports whether the bounded intent needs interactive
// refinement. DIRECT and PLAN never require Composer.
func ComposerIsRequired(decision IntentDecision) bool {
	return decision.Strategy == IntentClarify
}

// FlowIsRequired is always false: Flow is an optional WorkPlan projection.
func FlowIsRequired(IntentDecision) bool {
	return false
}

func isDirectIntent(goal string) bool {
	if containsAny(goal, "fix this broken test", "corrija este teste", "corrija esse teste", "fix typo", "ajuste de digitação", "quick fix") {
		return true
	}
	if containsAny(goal, "bug", "erro", "quebrado", "broken", "produção", "producao", "production", "arquitetura", "architecture") {
		return false
	}
	words := strings.Fields(goal)
	return len(words) <= 8 && containsAny(goal, "corrija", "corrigir", "fix", "ajuste", "update", "troque", "rename", "renomeie", "typo") && !hasPlanSignals(goal)
}

func hasConflictingProductBehavior(goal string) bool {
	must := strings.Contains(goal, "deve ") || strings.Contains(goal, "must ")
	mustNot := strings.Contains(goal, "não deve") || strings.Contains(goal, "nao deve") || strings.Contains(goal, "must not")
	return must && mustNot
}

func hasPlanSignals(goal string) bool {
	return containsAny(goal, "implemente", "implement", "melhorar", "improve", "refactor", "refatore", "projeto", "sistema", "feature", "e2e", "arquitetura", "architecture", "multiple", "and ")
}

func isMateriallyUnderspecified(goal string, snapshot *contextsnapshot.ProjectContextSnapshot) bool {
	if hasPlanSignals(goal) {
		return false
	}
	if len(strings.Fields(goal)) >= 4 {
		return false
	}
	// Facts only prevent clarification when they cover the concrete unknown.
	// A random stack fact must not turn a vague goal into PLAN.
	if snapshotFactsCoverGoal(goal, snapshot) {
		return false
	}
	return true
}

// snapshotFactsCoverGoal reports whether Project Intelligence already observed
// evidence for the topic named in the goal. Unrelated facts do not count.
func snapshotFactsCoverGoal(goal string, snapshot *contextsnapshot.ProjectContextSnapshot) bool {
	if snapshot == nil || len(snapshot.Facts) == 0 {
		return false
	}
	needed := make([]string, 0, 4)
	if containsAny(goal, "teste", "test") {
		needed = append(needed, "test", "vitest", "go-test", "jest")
	}
	if containsAny(goal, "build", "compile", "compilar") {
		needed = append(needed, "build", "commands")
	}
	if containsAny(goal, "lint", "format") {
		needed = append(needed, "lint", "format", "eslint")
	}
	if len(needed) == 0 {
		return false
	}
	for _, fact := range snapshot.Facts {
		key := strings.ToLower(fact.Key + " " + fmt.Sprint(fact.Value))
		for _, needle := range needed {
			if strings.Contains(key, needle) {
				return true
			}
		}
	}
	return false
}
