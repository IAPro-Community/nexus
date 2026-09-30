package nexus

import (
	"math"
	"strings"
)

type ComposerInterviewMode string

const (
	ComposerInterviewQuick    ComposerInterviewMode = "QUICK"
	ComposerInterviewStandard ComposerInterviewMode = "STANDARD"
	ComposerInterviewDeep     ComposerInterviewMode = "DEEP"
)

type ComposerClarity struct {
	Intent      float64 `json:"intent"`
	Scope       float64 `json:"scope"`
	Context     float64 `json:"context"`
	Constraints float64 `json:"constraints"`
	Acceptance  float64 `json:"acceptance"`
	Risk        float64 `json:"risk"`
	Decisions   float64 `json:"decisions"`
}

type ComposerInterviewState struct {
	Mode              ComposerInterviewMode `json:"mode"`
	Clarity           ComposerClarity       `json:"clarity"`
	Ambiguity         float64               `json:"ambiguity"`
	TargetAmbiguity   float64               `json:"target_ambiguity"`
	WeakestDimension  string                `json:"weakest_dimension,omitempty"`
	QuestionsAnswered int                   `json:"questions_answered"`
	MaxQuestions      int                   `json:"max_questions"`
	ChallengePending  bool                  `json:"challenge_pending,omitempty"`
}

func normalizeComposerInterviewMode(mode ComposerInterviewMode) ComposerInterviewMode {
	switch ComposerInterviewMode(strings.ToUpper(strings.TrimSpace(string(mode)))) {
	case ComposerInterviewQuick:
		return ComposerInterviewQuick
	case ComposerInterviewDeep:
		return ComposerInterviewDeep
	default:
		return ComposerInterviewStandard
	}
}

func composerInterviewProfile(mode ComposerInterviewMode) (target float64, maxQuestions int) {
	switch normalizeComposerInterviewMode(mode) {
	case ComposerInterviewQuick:
		return 0.35, 5
	case ComposerInterviewDeep:
		return 0.15, 18
	default:
		return 0.25, 10
	}
}

func refreshComposerInterview(brief *LivingBrief) {
	if brief == nil {
		return
	}
	mode := normalizeComposerInterviewMode(brief.Interview.Mode)
	target, maxQuestions := composerInterviewProfile(mode)
	clarity, ambiguity, weakest := computeComposerClarity(*brief)
	answered := composerInterviewAnsweredCount(brief.Unknowns)
	state := ComposerInterviewState{
		Mode: mode, Clarity: clarity, Ambiguity: ambiguity, TargetAmbiguity: target,
		WeakestDimension: weakest, QuestionsAnswered: answered, MaxQuestions: maxQuestions,
	}
	if shouldAddComposerPressurePass(*brief, state) {
		brief.Unknowns = append(brief.Unknowns, composerPressureUnknown(*brief))
	}
	for i := range brief.Unknowns {
		brief.Unknowns[i].Dimension = composerInterviewDimension(brief.Unknowns[i].Field)
		if brief.Unknowns[i].Status == PromptUnknownOpen {
			brief.Unknowns[i].PriorityScore = composerInterviewPriority(brief.Unknowns[i], clarity)
			if brief.Unknowns[i].Field == "interview.challenge" {
				state.ChallengePending = true
			}
		} else {
			brief.Unknowns[i].PriorityScore = 0
		}
	}
	brief.Interview = state
}

func computeComposerClarity(brief LivingBrief) (ComposerClarity, float64, string) {
	text := strings.ToLower(strings.Join([]string{brief.Goal, brief.SourcePrompt, brief.Intent.Objective, brief.Intent.DesiredOutcome}, " "))
	intent := 0.0
	if strings.TrimSpace(brief.Goal) != "" || strings.TrimSpace(brief.Intent.Objective) != "" { intent += 0.45 }
	if strings.TrimSpace(brief.Intent.DesiredOutcome) != "" { intent += 0.35 }
	if brief.Intent.Archetype != "" && brief.Intent.Archetype != PromptArchetypeGeneric { intent += 0.20 }

	scope := 0.10
	if len(brief.Scope.InScope) > 0 { scope += 0.55 }
	if len(brief.Scope.OutOfScope) > 0 { scope += 0.15 }
	if len(brief.Deliverable.ExpectedArtifacts) > 0 || strings.TrimSpace(brief.Deliverable.Type) != "" { scope += 0.20 }

	contextScore := 0.15
	if strings.TrimSpace(brief.Context.Project) != "" { contextScore += 0.30 }
	if len(brief.Context.ExistingState)+len(brief.Context.Evidence)+len(brief.Context.ReferencedFiles) > 0 { contextScore += 0.25 }
	if len(brief.Environment.Stack) > 0 || strings.TrimSpace(brief.Environment.Repository) != "" { contextScore += 0.30 }

	constraintSignal := composerContainsAny(text, "constraint", "restri", "must", "deve", "without", "sem ", "compat", "legacy", "security", "seguran", "deadline", "prazo", "budget", "orçamento")
	constraints := 0.65
	if constraintSignal {
		constraints = 0.15
		if len(brief.Constraints.Technical)+len(brief.Constraints.Product)+len(brief.Constraints.Security)+len(brief.Constraints.Compliance)+len(brief.Constraints.Schedule)+len(brief.Constraints.Budget) > 0 { constraints += 0.55 }
		if len(brief.Execution.ProhibitedActions) > 0 { constraints += 0.30 }
	}

	acceptance := 0.05
	if len(brief.Quality.AcceptanceCriteria) > 0 { acceptance += 0.45 }
	if len(brief.Quality.Testing) > 0 { acceptance += 0.20 }
	if len(brief.Quality.Verification) > 0 { acceptance += 0.20 }
	if len(brief.Quality.DefinitionOfDone) > 0 { acceptance += 0.10 }

	riskSignal := composerContainsAny(text, "risk", "risco", "regression", "regress", "security", "seguran", "auth", "quota", "limite", "deploy", "migration", "migra")
	risk := 0.75
	if riskSignal || len(brief.Risks)+len(brief.Constraints.Security) > 0 {
		risk = 0.10
		if len(brief.Risks) > 0 { risk += 0.45 }
		if len(brief.Constraints.Security) > 0 { risk += 0.25 }
		if len(brief.Execution.ProhibitedActions) > 0 { risk += 0.20 }
	}

	decisionSignal := composerContainsAny(text, "automatic", "automático", "auto ", "best", "melhor", "prefer", "prioridade", "tradeoff", "trade-off", "fallback", "custo", "cost", "quota", "sempre", "never", "nunca")
	decisions := 0.80
	if decisionSignal || len(brief.Decisions)+len(brief.Alternatives) > 0 {
		decisions = 0.10
		if len(brief.Decisions) > 0 { decisions += 0.45 }
		if len(brief.Alternatives) > 0 { decisions += 0.20 }
		for _, assumption := range brief.Assumptions {
			status := strings.ToUpper(strings.TrimSpace(assumption.Status))
			if status == "CONFIRMED" || status == "DISMISSED" { decisions += 0.25; break }
		}
	}

	clarity := ComposerClarity{
		Intent: clampComposerUnit(intent), Scope: clampComposerUnit(scope), Context: clampComposerUnit(contextScore),
		Constraints: clampComposerUnit(constraints), Acceptance: clampComposerUnit(acceptance),
		Risk: clampComposerUnit(risk), Decisions: clampComposerUnit(decisions),
	}
	type weightedDimension struct { name string; score, weight float64 }
	dimensions := []weightedDimension{
		{"intent", clarity.Intent, 1.0}, {"scope", clarity.Scope, 1.4}, {"context", clarity.Context, 1.0},
		{"constraints", clarity.Constraints, 0.8}, {"acceptance", clarity.Acceptance, 1.4},
		{"risk", clarity.Risk, 0.7}, {"decisions", clarity.Decisions, 0.7},
	}
	weakest, weakestScore := dimensions[0].name, dimensions[0].score
	weightedClarity, totalWeight := 0.0, 0.0
	for _, dimension := range dimensions {
		weightedClarity += dimension.score * dimension.weight
		totalWeight += dimension.weight
		if dimension.score < weakestScore { weakest, weakestScore = dimension.name, dimension.score }
	}
	ambiguity := 1.0
	if totalWeight > 0 { ambiguity = clampComposerUnit(1 - weightedClarity/totalWeight) }
	return clarity, math.Round(ambiguity*1000) / 1000, weakest
}

func composerInterviewDimension(field string) string {
	field = strings.ToLower(strings.TrimSpace(field))
	switch {
	case field == "interview.challenge", strings.HasPrefix(field, "decision"), strings.HasPrefix(field, "alternative"): return "decisions"
	case strings.HasPrefix(field, "scope."), strings.HasPrefix(field, "deliverable."): return "scope"
	case strings.HasPrefix(field, "quality."): return "acceptance"
	case strings.HasPrefix(field, "risk"), strings.HasPrefix(field, "constraints.security"): return "risk"
	case strings.HasPrefix(field, "constraints."), strings.HasPrefix(field, "execution."): return "constraints"
	case strings.HasPrefix(field, "context."), strings.HasPrefix(field, "environment."): return "context"
	case strings.HasPrefix(field, "intent."): return "intent"
	default: return "context"
	}
}

func composerDimensionScore(clarity ComposerClarity, dimension string) float64 {
	switch dimension {
	case "intent": return clarity.Intent
	case "scope": return clarity.Scope
	case "constraints": return clarity.Constraints
	case "acceptance": return clarity.Acceptance
	case "risk": return clarity.Risk
	case "decisions": return clarity.Decisions
	default: return clarity.Context
	}
}

func composerInterviewPriority(unknown PromptUnknown, clarity ComposerClarity) int {
	score := int(math.Round((1 - composerDimensionScore(clarity, composerInterviewDimension(unknown.Field))) * 100))
	switch unknown.Severity {
	case PromptUnknownBlocking: score += 30
	case PromptUnknownRecommended: score += 15
	case PromptUnknownOptional: score += 5
	}
	if strings.EqualFold(unknown.Source, "INTELLIGENCE") { score += 12 }
	if len(unknown.SuggestedOptions) > 0 { score += 4 }
	if unknown.Field == "interview.challenge" { score += 8 }
	if score < 0 { return 0 }
	return score
}

func selectComposerNextUnknownForBrief(brief LivingBrief) (PromptUnknown, bool) {
	mode := normalizeComposerInterviewMode(brief.Interview.Mode)
	_, maxQuestions := composerInterviewProfile(mode)
	budgetReached := brief.Interview.QuestionsAnswered >= maxQuestions
	var best PromptUnknown
	found := false
	for _, unknown := range brief.Unknowns {
		if unknown.Status != PromptUnknownOpen { continue }
		if budgetReached && unknown.Severity != PromptUnknownBlocking { continue }
		if !found || unknown.PriorityScore > best.PriorityScore ||
			(unknown.PriorityScore == best.PriorityScore && unknown.Severity == PromptUnknownBlocking && best.Severity != PromptUnknownBlocking) {
			best, found = unknown, true
		}
	}
	return best, found
}

func composerInterviewAnsweredCount(unknowns []PromptUnknown) int {
	count := 0
	for _, unknown := range unknowns {
		switch unknown.Status {
		case PromptUnknownAnswered, PromptUnknownConfirmed, PromptUnknownDismissed: count++
		}
	}
	return count
}

func shouldAddComposerPressurePass(brief LivingBrief, state ComposerInterviewState) bool {
	if state.Mode == ComposerInterviewQuick || state.QuestionsAnswered >= state.MaxQuestions { return false }
	for _, unknown := range brief.Unknowns {
		if unknown.Field == "interview.challenge" || unknown.ID == "interview-pressure" { return false }
		if unknown.Status == PromptUnknownOpen && unknown.Severity == PromptUnknownBlocking { return false }
	}
	if state.Ambiguity > state.TargetAmbiguity+0.10 { return false }
	text := strings.ToLower(brief.Goal + " " + brief.SourcePrompt)
	return len(brief.Decisions)+len(brief.Alternatives)+len(brief.Risks) > 0 ||
		composerContainsAny(text, "automatic", "automático", "best", "melhor", "prefer", "prioridade", "tradeoff", "trade-off", "fallback", "custo", "cost", "quota", "limite", "sempre", "never", "nunca")
}

func composerPressureUnknown(brief LivingBrief) PromptUnknown {
	text := strings.ToLower(brief.Goal + " " + brief.SourcePrompt)
	question := "Qual cenário faria a decisão principal deixar de ser válida, e qual alternativa deve prevalecer nesse caso?"
	options := []string{
		"Preservar compatibilidade/segurança mesmo com menor desempenho",
		"Priorizar qualidade mesmo com maior custo ou consumo",
		"Priorizar custo/quota mesmo com alguma perda de qualidade",
		"Não há exceção relevante; manter a decisão atual",
	}
	if composerContainsAny(text, "quota", "limite", "custo", "cost") {
		question = "Se qualidade, custo e quota entrarem em conflito, qual prioridade deve vencer e em que condição o Nexus deve pedir confirmação?"
	} else if composerContainsAny(text, "automatic", "automático", "best", "melhor", "fallback") {
		question = "Qual condição deve invalidar a escolha automática ou forçar confirmação antes de usar outra alternativa?"
	}
	return PromptUnknown{
		ID: "interview-pressure", Field: "interview.challenge", Question: question,
		Rationale: "Pressure-test final: expõe trade-offs escondidos antes de cristalizar o PromptArtifact.",
		Severity: PromptUnknownRecommended, Status: PromptUnknownOpen, Source: "INTERVIEW", SuggestedOptions: options,
	}
}

func composerContainsAny(text string, terms ...string) bool {
	for _, term := range terms { if strings.Contains(text, term) { return true } }
	return false
}

func clampComposerUnit(value float64) float64 {
	if value < 0 { return 0 }
	if value > 1 { return 1 }
	return value
}
