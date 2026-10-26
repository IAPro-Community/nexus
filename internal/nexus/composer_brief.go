package nexus

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

type ComposerFactStatus string

const (
	ComposerFactProposed  ComposerFactStatus = "PROPOSED"
	ComposerFactConfirmed ComposerFactStatus = "CONFIRMED"
	ComposerFactInferred  ComposerFactStatus = "INFERRED"
	ComposerFactAssumed   ComposerFactStatus = "ASSUMED"
	ComposerFactDismissed ComposerFactStatus = "DISMISSED"
)

// ComposerFact is a traceable statement in the brief. It deliberately stores
// provenance, rather than treating model output as user-confirmed truth.
type ComposerFact struct {
	ID         string             `json:"id"`
	Field      string             `json:"field"`
	Value      string             `json:"value"`
	SourceType string             `json:"source_type"`
	SourceRef  string             `json:"source_ref,omitempty"`
	Confidence string             `json:"confidence,omitempty"`
	Status     ComposerFactStatus `json:"status"`
	Impact     string             `json:"impact,omitempty"`
	Revision   int                `json:"revision"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type MotivationMap struct {
	Problem        []ComposerFact `json:"problem,omitempty"`
	Affected       []ComposerFact `json:"affected,omitempty"`
	Impact         []ComposerFact `json:"impact,omitempty"`
	Result         []ComposerFact `json:"result,omitempty"`
	Urgency        []ComposerFact `json:"urgency,omitempty"`
	Tradeoffs      []ComposerFact `json:"tradeoffs,omitempty"`
	SuccessSignals []ComposerFact `json:"success_signals,omitempty"`
	FailureSignals []ComposerFact `json:"failure_signals,omitempty"`
}

type PromptArchetype string

const (
	PromptArchetypeSoftwareFeature PromptArchetype = "SOFTWARE_FEATURE"
	PromptArchetypeBugFix          PromptArchetype = "BUG_FIX"
	PromptArchetypeArchitecture    PromptArchetype = "ARCHITECTURE"
	PromptArchetypeDevOps          PromptArchetype = "DEVOPS"
	PromptArchetypeResearch        PromptArchetype = "RESEARCH"
	PromptArchetypeSecurity        PromptArchetype = "SECURITY"
	PromptArchetypeGeneric         PromptArchetype = "GENERIC"
)

type PromptUnknownSeverity string

const (
	PromptUnknownBlocking    PromptUnknownSeverity = "BLOCKING"
	PromptUnknownRecommended PromptUnknownSeverity = "RECOMMENDED"
	PromptUnknownOptional    PromptUnknownSeverity = "OPTIONAL"
)

type PromptUnknownStatus string

const (
	PromptUnknownOpen      PromptUnknownStatus = "OPEN"
	PromptUnknownAnswered  PromptUnknownStatus = "ANSWERED"
	PromptUnknownInferred  PromptUnknownStatus = "INFERRED"
	PromptUnknownConfirmed PromptUnknownStatus = "CONFIRMED"
	PromptUnknownDismissed PromptUnknownStatus = "DISMISSED"
)

type PromptReadinessState string

const (
	PromptReadinessReady               PromptReadinessState = "READY"
	PromptReadinessReadyWithAssumption PromptReadinessState = "READY_WITH_ASSUMPTIONS"
	PromptReadinessNeedsInformation    PromptReadinessState = "NEEDS_INFORMATION"
	PromptReadinessBlocked             PromptReadinessState = "BLOCKED"
)

type BriefIntent struct {
	Archetype      PromptArchetype `json:"archetype"`
	Objective      string          `json:"objective,omitempty"`
	DesiredOutcome string          `json:"desired_outcome,omitempty"`
}

// ContextFileRef is a durable, redacted reference to a project file that the
// user explicitly attached to the Composer brief. The excerpt is bounded and
// the hash lets the UI detect drift without re-reading the file.
type ContextFileRef struct {
	Path    string `json:"path"`
	Excerpt string `json:"excerpt,omitempty"`
	Hash    string `json:"hash,omitempty"`
}

type BriefContext struct {
	Domain          string           `json:"domain,omitempty"`
	Project         string           `json:"project,omitempty"`
	Audience        string           `json:"audience,omitempty"`
	ExistingState   []string         `json:"existing_state,omitempty"`
	Evidence        []string         `json:"evidence,omitempty"`
	ReferencedFiles []ContextFileRef `json:"referenced_files,omitempty"`
}

type BriefDeliverable struct {
	Type              string   `json:"type,omitempty"`
	Format            string   `json:"format,omitempty"`
	ExpectedArtifacts []string `json:"expected_artifacts,omitempty"`
}

type BriefScope struct {
	InScope    []string `json:"in_scope,omitempty"`
	OutOfScope []string `json:"out_of_scope,omitempty"`
}

type BriefEnvironment struct {
	Stack            []string `json:"stack,omitempty"`
	Repository       string   `json:"repository,omitempty"`
	OperatingSystems []string `json:"operating_systems,omitempty"`
	Infrastructure   []string `json:"infrastructure,omitempty"`
}

type BriefConstraints struct {
	Technical  []string `json:"technical,omitempty"`
	Product    []string `json:"product,omitempty"`
	Security   []string `json:"security,omitempty"`
	Compliance []string `json:"compliance,omitempty"`
	Schedule   []string `json:"schedule,omitempty"`
	Budget     []string `json:"budget,omitempty"`
}

type BriefExecution struct {
	Autonomy          string   `json:"autonomy,omitempty"`
	AllowedActions    []string `json:"allowed_actions,omitempty"`
	ProhibitedActions []string `json:"prohibited_actions,omitempty"`
}

type BriefQuality struct {
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Testing            []string `json:"testing,omitempty"`
	Review             []string `json:"review,omitempty"`
	Verification       []string `json:"verification,omitempty"`
	DefinitionOfDone   []string `json:"definition_of_done,omitempty"`
}

type PromptAssumption struct {
	ID         string `json:"id"`
	Value      string `json:"value"`
	Confidence string `json:"confidence"`
	Status     string `json:"status"`
}

type PromptUnknown struct {
	ID               string                `json:"id"`
	Field            string                `json:"field"`
	Question         string                `json:"question"`
	Rationale        string                `json:"rationale"`
	Severity         PromptUnknownSeverity `json:"severity"`
	Status           PromptUnknownStatus   `json:"status"`
	Answer           string                `json:"answer,omitempty"`
	InferredValue    string                `json:"inferred_value,omitempty"`
	Confidence       string                `json:"confidence,omitempty"`
	Source           string                `json:"source,omitempty"`
	SuggestedOptions []string              `json:"suggested_options,omitempty"`
	DefaultChoice    string                `json:"default_choice,omitempty"`
	Dimension        string                `json:"dimension,omitempty"`
	PriorityScore    int                   `json:"priority_score,omitempty"`
}

type PromptReadinessCheck struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

type PromptReadiness struct {
	Score   int                    `json:"score"`
	State   PromptReadinessState   `json:"state"`
	Summary string                 `json:"summary"`
	Checks  []PromptReadinessCheck `json:"checks,omitempty"`
}

// LivingBrief stores the structured, durable understanding assembled by Composer.
type LivingBrief struct {
	Goal          string                 `json:"goal"`
	SourcePrompt  string                 `json:"source_prompt,omitempty"`
	Intent        BriefIntent            `json:"intent"`
	Context       BriefContext           `json:"context"`
	Deliverable   BriefDeliverable       `json:"deliverable"`
	Scope         BriefScope             `json:"scope"`
	Environment   BriefEnvironment       `json:"environment"`
	Constraints   BriefConstraints       `json:"constraints"`
	Execution     BriefExecution         `json:"execution"`
	Quality       BriefQuality           `json:"quality"`
	Decisions     []string               `json:"decisions,omitempty"`
	Assumptions   []PromptAssumption     `json:"assumptions,omitempty"`
	Alternatives  []string               `json:"alternatives,omitempty"`
	Risks         []string               `json:"risks,omitempty"`
	Unknowns      []PromptUnknown        `json:"unknowns,omitempty"`
	OpenQuestions []string               `json:"open_questions,omitempty"`
	Readiness     PromptReadiness        `json:"readiness"`
	Interview     ComposerInterviewState `json:"interview"`
	Motivation    MotivationMap          `json:"motivation,omitempty"`
	Facts         []ComposerFact         `json:"facts,omitempty"`
	Preview       string                 `json:"preview,omitempty"`
}

type composerUnknownBlueprint struct {
	ID        string
	Field     string
	Question  string
	Rationale string
	Severity  PromptUnknownSeverity
}

func newComposerBrief(goal, sourcePrompt string) LivingBrief {
	return newComposerBriefWithInterviewMode(goal, sourcePrompt, ComposerInterviewStandard)
}

func newComposerBriefWithInterviewMode(goal, sourcePrompt string, interviewMode ComposerInterviewMode) LivingBrief {
	body := strings.TrimSpace(sourcePrompt)
	if body == "" {
		body = strings.TrimSpace(goal)
	}
	objective := strings.TrimSpace(goal)
	if objective == "" {
		objective = firstStatement(body)
	}
	brief := LivingBrief{
		Goal:         strings.TrimSpace(goal),
		SourcePrompt: strings.TrimSpace(sourcePrompt),
		Intent: BriefIntent{
			Archetype:      classifyPromptArchetype(body),
			Objective:      objective,
			DesiredOutcome: objective,
		},
		Interview: ComposerInterviewState{Mode: normalizeComposerInterviewMode(interviewMode)},
	}
	mergeTextIntoBrief(&brief, body, "USER")
	brief.Motivation = deriveMotivationMap(brief)
	refreshComposerBrief(&brief)
	return brief
}

func refreshComposerBrief(brief *LivingBrief) {
	if brief == nil {
		return
	}
	if strings.TrimSpace(brief.Goal) == "" {
		brief.Goal = strings.TrimSpace(brief.Intent.Objective)
	}
	if brief.Intent.Archetype == "" {
		brief.Intent.Archetype = classifyPromptArchetype(brief.Goal + "\n" + brief.SourcePrompt)
	}
	brief.Unknowns = refreshComposerUnknowns(*brief)
	refreshComposerInterview(brief)
	brief.OpenQuestions = nil
	for _, unknown := range brief.Unknowns {
		if unknown.Status == PromptUnknownOpen {
			brief.OpenQuestions = append(brief.OpenQuestions, unknown.Question)
		}
	}
	brief.Readiness = computePromptReadiness(*brief)
	brief.Motivation = deriveMotivationMap(*brief)
	brief.Preview = renderCanonicalPrompt(*brief)
}

// PromptIsShallow reports whether the brief still lacks an observable outcome
// that would produce a strong development prompt.
func PromptIsShallow(brief LivingBrief) bool {
	if strings.TrimSpace(brief.Goal) == "" {
		return true
	}
	words := strings.Fields(brief.Goal)
	if len(words) < 4 && len(brief.Quality.AcceptanceCriteria) == 0 && len(brief.Scope.InScope) == 0 {
		return true
	}
	if len(brief.Quality.AcceptanceCriteria) == 0 && len(brief.Scope.InScope) == 0 && len(brief.Decisions) == 0 {
		lower := strings.ToLower(brief.Goal)
		vague := []string{"melhorar", "improve", "ajustar", "update", "dashboard", "ui", "algo", "something"}
		hits := 0
		for _, v := range vague {
			if strings.Contains(lower, v) {
				hits++
			}
		}
		if hits > 0 && len(words) < 12 {
			return true
		}
	}
	return false
}

func deriveMotivationMap(brief LivingBrief) MotivationMap {
	now := time.Now().UTC()
	fact := func(id, field, value, source, impact string) ComposerFact {
		return ComposerFact{ID: id, Field: field, Value: strings.TrimSpace(value), SourceType: source, Confidence: "MEDIUM", Status: ComposerFactInferred, Impact: impact, Revision: 1, UpdatedAt: now}
	}
	out := MotivationMap{}
	if strings.TrimSpace(brief.Goal) != "" {
		out.Result = append(out.Result, fact("motivation-result", "motivation.result", brief.Goal, "USER", "defines the observable outcome"))
	}
	if len(brief.Context.ExistingState) > 0 {
		for i, value := range brief.Context.ExistingState {
			out.Problem = append(out.Problem, fact(fmt.Sprintf("motivation-problem-%d", i+1), "motivation.problem", value, "USER", "explains the current problem"))
		}
	}
	if len(brief.Quality.AcceptanceCriteria) > 0 {
		for i, value := range brief.Quality.AcceptanceCriteria {
			out.SuccessSignals = append(out.SuccessSignals, fact(fmt.Sprintf("motivation-success-%d", i+1), "motivation.success", value, "USER", "proves success"))
		}
	}
	return out
}

func mergeTextIntoBrief(brief *LivingBrief, text, source string) {
	text = strings.TrimSpace(text)
	if brief == nil || text == "" {
		return
	}
	if brief.Intent.Objective == "" {
		brief.Intent.Objective = firstStatement(text)
	}
	if brief.Intent.DesiredOutcome == "" {
		brief.Intent.DesiredOutcome = brief.Intent.Objective
	}
	brief.Context.ExistingState = appendUniqueAll(brief.Context.ExistingState, extractTaggedStatements(text, []string{
		"existing", "current", "today", "already", "atual", "hoje", "existente", "dashboard", "api", "backend", "frontend", "projeto",
	}))
	brief.Scope.InScope = appendUniqueAll(brief.Scope.InScope, extractTaggedStatements(text, []string{
		"include", "includes", "implement", "build", "create", "deliver", "scope", "escopo", "precisa", "deve", "fazer",
	}))
	brief.Scope.OutOfScope = appendUniqueAll(brief.Scope.OutOfScope, extractTaggedStatements(text, []string{
		"out of scope", "don't", "do not", "exclude", "without", "fora do escopo", "nao", "não", "evitar",
	}))
	brief.Quality.AcceptanceCriteria = appendUniqueAll(brief.Quality.AcceptanceCriteria, extractTaggedStatements(text, []string{
		"success means", "ready when", "done when", "acceptance", "criteria", "must", "should", "resultado", "sucesso", "pronto",
	}))
	brief.Quality.Testing = appendUniqueAll(brief.Quality.Testing, extractTaggedStatements(text, []string{
		"test", "tests", "coverage", "regression", "e2e", "integration", "unit", "teste", "testes", "cobertura",
	}))
	brief.Quality.Verification = appendUniqueAll(brief.Quality.Verification, extractTaggedStatements(text, []string{
		"verify", "verification", "validate", "validated", "go test", "npm test", "pnpm", "check", "verificar", "validação", "validar",
	}))
	brief.Constraints.Technical = appendUniqueAll(brief.Constraints.Technical, extractTaggedStatements(text, []string{
		"constraint", "limitation", "compatible", "compatibility", "technical", "restri", "compat", "legacy",
	}))
	brief.Constraints.Schedule = appendUniqueAll(brief.Constraints.Schedule, extractTaggedStatements(text, []string{
		"deadline", "today", "tomorrow", "this week", "schedule", "prazo", "semana", "dia",
	}))
	brief.Constraints.Security = appendUniqueAll(brief.Constraints.Security, extractTaggedStatements(text, []string{
		"security", "oauth", "auth", "token", "secret", "csrf", "seguran", "autentica",
	}))
	brief.Environment.Stack = appendUniqueAll(brief.Environment.Stack, inferStackFromText(text))
	brief.Environment.OperatingSystems = appendUniqueAll(brief.Environment.OperatingSystems, inferPlatformsFromText(text))
	if strings.Contains(strings.ToLower(text), "flow") {
		brief.Deliverable.ExpectedArtifacts = appendUnique(brief.Deliverable.ExpectedArtifacts, "Flow draft")
	}
	if strings.Contains(strings.ToLower(text), "prompt") {
		brief.Deliverable.Type = firstNonEmpty(brief.Deliverable.Type, "Prompt artifact")
	}
	if strings.Contains(strings.ToLower(text), "review") {
		brief.Quality.Review = appendUnique(brief.Quality.Review, firstStatement(text))
	}
	if source == "USER" {
		for _, assumed := range inferAssumptions(*brief) {
			brief.Assumptions = upsertAssumption(brief.Assumptions, assumed)
		}
	}
}

func refreshComposerUnknowns(brief LivingBrief) []PromptUnknown {
	// Preserve intelligence / user unknowns keyed by field. Archetype checklists
	// only fill gaps that are still missing — they never wipe intelligence items.
	current := map[string]PromptUnknown{}
	order := []string{}
	for _, item := range brief.Unknowns {
		// Legacy sessions may carry open questions produced by the static
		// checklist. They are not user decisions and must not be replayed.
		if (item.Source == "ARCHETYPE" || item.Source == "REFINERY") &&
			item.Status != PromptUnknownAnswered && item.Status != PromptUnknownConfirmed && item.Status != PromptUnknownDismissed {
			continue
		}
		field := item.Field
		if field == "" {
			field = item.ID
		}
		if _, exists := current[field]; !exists {
			order = append(order, field)
		}
		current[field] = item
	}

	shallow := PromptIsShallow(brief)
	blueprints := archetypeUnknowns(brief.Intent.Archetype)
	for _, bp := range blueprints {
		item, exists := current[bp.Field]
		if !exists {
			// A missing field is not by itself a reason to interrogate a
			// non-technical user. The intelligence evaluator supplies material
			// questions; the archetype remains a non-blocking quality checklist.
			_, status, _ := evaluateBriefField(brief, bp.Field)
			if status == PromptUnknownOpen {
				continue
			}
			item = PromptUnknown{
				ID: bp.ID, Field: bp.Field, Question: bp.Question,
				Rationale: bp.Rationale, Severity: bp.Severity, Source: "ARCHETYPE",
			}
			order = append(order, bp.Field)
		} else if item.Source == "INTELLIGENCE" || item.Source == "USER" || item.Source == "MEMORY" {
			// Keep intelligence/user/memory questions intact; only re-evaluate status.
		} else {
			item.Question = bp.Question
			item.Rationale = bp.Rationale
			if item.Severity == "" {
				item.Severity = bp.Severity
			}
		}

		if item.Status == PromptUnknownAnswered || item.Status == PromptUnknownConfirmed || item.Status == PromptUnknownDismissed {
			current[bp.Field] = item
			continue
		}

		summary, status, confidence := evaluateBriefField(brief, bp.Field)
		switch status {
		case PromptUnknownConfirmed, PromptUnknownAnswered:
			item.Status = status
			item.Answer = summary
			item.InferredValue = ""
			item.Confidence = confidence
			if item.Source == "" || item.Source == "ARCHETYPE" {
				item.Source = "USER"
			}
		case PromptUnknownInferred:
			item.Status = PromptUnknownInferred
			item.Answer = ""
			item.InferredValue = summary
			item.Confidence = confidence
			if item.Source == "" || item.Source == "ARCHETYPE" {
				item.Source = "PROJECT_CONTEXT"
			}
			// Ritual archetype BLOCKING becomes RECOMMENDED when not shallow and
			// the field was only inferred from context/memory.
			if item.Source != "INTELLIGENCE" && !shallow && bp.Severity == PromptUnknownBlocking {
				item.Severity = PromptUnknownRecommended
			}
		default:
			item.Status = PromptUnknownOpen
			item.Answer = ""
			item.InferredValue = ""
			item.Confidence = ""
			// Ritual ARCHETYPE never stays BLOCKING: shallow prompts extract via
			// intelligence (options); deep prompts unlock destinations without a form.
			if item.Source != "INTELLIGENCE" && item.Source != "USER" && item.Source != "MEMORY" &&
				bp.Severity == PromptUnknownBlocking {
				item.Severity = PromptUnknownRecommended
				if !shallow {
					item.Status = PromptUnknownInferred
					item.InferredValue = "deferred — prompt already has actionable depth"
					item.Confidence = "LOW"
					item.Source = "REFINERY"
				}
			}
		}
		current[bp.Field] = item
	}

	out := make([]PromptUnknown, 0, len(order))
	seen := map[string]bool{}
	for _, field := range order {
		item := current[field]
		if seen[field] {
			continue
		}
		seen[field] = true
		out = append(out, item)
	}
	return out
}

// MergeIntelligenceUnknowns folds provider AmbiguityItems into the brief without
// discarding existing resolved answers.
func MergeIntelligenceUnknowns(brief *LivingBrief, items []intelligence.AmbiguityItem) {
	if brief == nil || len(items) == 0 {
		return
	}
	byField := map[string]int{}
	for i, u := range brief.Unknowns {
		byField[u.Field] = i
		if u.ID != "" {
			byField[u.ID] = i
		}
	}
	for _, item := range items {
		field := strings.TrimSpace(item.Key)
		if field == "" || strings.TrimSpace(item.Question) == "" {
			continue
		}
		// If already evidenced in the brief, do not ask for it again.
		if _, status, _ := evaluateBriefField(*brief, field); status == PromptUnknownConfirmed || status == PromptUnknownAnswered {
			continue
		}
		severity := PromptUnknownRecommended
		switch item.Level {
		case intelligence.AmbiguityBlocking:
			severity = PromptUnknownBlocking
		case intelligence.AmbiguityLowImpact:
			severity = PromptUnknownOptional
		}
		next := PromptUnknown{
			ID: field, Field: field, Question: item.Question, Rationale: item.Rationale,
			Severity: severity, Status: PromptUnknownOpen, Source: "INTELLIGENCE",
			SuggestedOptions: append([]string{}, item.SuggestedOptions...),
			DefaultChoice:    item.DefaultChoice,
		}
		if item.IsResolved && strings.TrimSpace(item.Answer) != "" {
			next.Status = PromptUnknownAnswered
			next.Answer = item.Answer
		} else if severity == PromptUnknownOptional && item.DefaultChoice != "" {
			next.Status = PromptUnknownInferred
			next.InferredValue = item.DefaultChoice
			next.Confidence = "MEDIUM"
		}
		if idx, ok := byField[field]; ok {
			prev := brief.Unknowns[idx]
			if prev.Status == PromptUnknownAnswered || prev.Status == PromptUnknownConfirmed || prev.Status == PromptUnknownDismissed {
				continue
			}
			if len(next.SuggestedOptions) == 0 {
				next.SuggestedOptions = prev.SuggestedOptions
			}
			if next.DefaultChoice == "" {
				next.DefaultChoice = prev.DefaultChoice
			}
			brief.Unknowns[idx] = next
			continue
		}
		brief.Unknowns = append(brief.Unknowns, next)
	}
}

// applyBriefExtraction merges provider-extracted brief fields into the living
// brief. Heuristics remain the primary source (they run first); intelligence
// only fills gaps. Every newly added value is recorded as a traceable
// ComposerFact with SourceType="INTELLIGENCE" and Status=INFERRED so the brief
// never presents inferred content as user-confirmed truth.
func applyBriefExtraction(brief *LivingBrief, ext *intelligence.BriefExtraction) {
	if brief == nil || ext == nil {
		return
	}
	now := time.Now().UTC()
	merge := func(field string, target *[]string, values []string) {
		for _, raw := range values {
			value := strings.TrimSpace(raw)
			if value == "" {
				continue
			}
			before := len(*target)
			*target = appendUnique(*target, value)
			if len(*target) == before {
				continue
			}
			brief.Facts = append(brief.Facts, ComposerFact{
				ID:         fmt.Sprintf("intelligence-%s-%d", field, len(brief.Facts)+1),
				Field:      field,
				Value:      value,
				SourceType: "INTELLIGENCE",
				Confidence: "MEDIUM",
				Status:     ComposerFactInferred,
				Revision:   1,
				UpdatedAt:  now,
			})
		}
	}
	merge("scope.in_scope", &brief.Scope.InScope, ext.InScope)
	merge("scope.out_of_scope", &brief.Scope.OutOfScope, ext.OutOfScope)
	merge("constraints.technical", &brief.Constraints.Technical, ext.ConstraintsTechnical)
	merge("constraints.security", &brief.Constraints.Security, ext.ConstraintsSecurity)
	merge("risks", &brief.Risks, ext.Risks)
	merge("quality.acceptance_criteria", &brief.Quality.AcceptanceCriteria, ext.AcceptanceCriteria)
	merge("quality.testing", &brief.Quality.Testing, ext.Testing)
	merge("quality.verification", &brief.Quality.Verification, ext.Verification)
	merge("execution.prohibited_actions", &brief.Execution.ProhibitedActions, ext.ProhibitedActions)
	merge("deliverable.expected_artifacts", &brief.Deliverable.ExpectedArtifacts, ext.Deliverables)
	merge("environment.stack", &brief.Environment.Stack, ext.Stack)
	merge("context.existing_state", &brief.Context.ExistingState, ext.ExistingState)
}

func computePromptReadiness(brief LivingBrief) PromptReadiness {
	checks := []PromptReadinessCheck{}
	completed := 0.0
	inferredOnly := false
	for _, bp := range archetypeUnknowns(brief.Intent.Archetype) {
		summary, status, _ := evaluateBriefField(brief, bp.Field)
		check := PromptReadinessCheck{Key: bp.Field, Label: readinessLabel(bp.Field), Summary: summary}
		switch status {
		case PromptUnknownConfirmed, PromptUnknownAnswered:
			check.Status = "COMPLETE"
			completed += 1
		case PromptUnknownInferred:
			check.Status = "PARTIAL"
			completed += 0.6
			inferredOnly = true
		default:
			check.Status = "MISSING"
		}
		checks = append(checks, check)
	}

	legacyScore := 0
	if total := float64(len(checks)); total > 0 {
		legacyScore = int(math.Round((completed / total) * 100))
	}

	blockingOpen := false
	for _, unknown := range brief.Unknowns {
		if unknown.Status == PromptUnknownOpen && unknown.Severity == PromptUnknownBlocking {
			blockingOpen = true
			break
		}
	}

	score := legacyScore
	target := 0.25
	ambiguity := 1.0
	if brief.Interview.Mode != "" {
		ambiguity = clampComposerUnit(brief.Interview.Ambiguity)
		target = clampComposerUnit(brief.Interview.TargetAmbiguity)
		interviewScore := int(math.Round((1 - ambiguity) * 100))
		if interviewScore > score {
			score = interviewScore
		}
	}

	state := PromptReadinessNeedsInformation
	switch {
	case len(checks) == 0:
		state = PromptReadinessReady
	case blockingOpen:
		state = PromptReadinessBlocked
	case !PromptIsShallow(brief) && len(brief.OpenQuestions) == 0:
		// Missing optional checklist fields are not pending user decisions.
		// The draft remains reviewable with explicit assumptions.
		state = PromptReadinessReadyWithAssumption
	case brief.Interview.Mode != "" && ambiguity <= target && !brief.Interview.ChallengePending && !inferredOnly:
		state = PromptReadinessReady
	case brief.Interview.Mode != "" && ambiguity <= target+0.10:
		state = PromptReadinessReadyWithAssumption
	case legacyScore >= 90 && !inferredOnly:
		state = PromptReadinessReady
	case legacyScore >= 80:
		state = PromptReadinessReadyWithAssumption
	}

	summary := "Composer still needs more requirements."
	switch state {
	case PromptReadinessReady:
		summary = "Prompt is ready to finalize."
	case PromptReadinessReadyWithAssumption:
		if brief.Interview.ChallengePending {
			summary = "Prompt is usable; one pressure-test question can still expose a hidden tradeoff."
		} else {
			summary = "Prompt is usable, but some details were inferred."
		}
	case PromptReadinessBlocked:
		summary = "Blocking gaps remain before the final prompt is trustworthy."
	}
	return PromptReadiness{Score: score, State: state, Summary: summary, Checks: checks}
}

func renderCanonicalPrompt(brief LivingBrief) string {
	var b strings.Builder
	writePromptSection(&b, "Role", roleForArchetype(brief.Intent.Archetype))
	writePromptSection(&b, "Objective", firstNonEmpty(strings.TrimSpace(brief.Intent.Objective), strings.TrimSpace(brief.Goal)))
	writePromptList(&b, "Motivation — Problem", factsToLines(brief.Motivation.Problem))
	writePromptList(&b, "Motivation — Impact", factsToLines(brief.Motivation.Impact))
	writePromptList(&b, "Motivation — Result", factsToLines(brief.Motivation.Result))
	writePromptSection(&b, "Domain", brief.Context.Domain)
	writePromptSection(&b, "Audience", brief.Context.Audience)
	writePromptList(&b, "Context", append(append([]string(nil), brief.Context.Evidence...), brief.Context.ExistingState...))
	writeReferencedFiles(&b, brief.Context.ReferencedFiles)
	writePromptList(&b, "Requirements", brief.Scope.InScope)
	writePromptList(&b, "Out of scope", brief.Scope.OutOfScope)
	writePromptList(&b, "Decisions", brief.Decisions)
	writePromptList(&b, "Assumptions", flattenAssumptionsToLines(brief.Assumptions))
	writePromptList(&b, "Constraints — Technical", brief.Constraints.Technical)
	writePromptList(&b, "Constraints — Product", brief.Constraints.Product)
	writePromptList(&b, "Constraints — Security", brief.Constraints.Security)
	writePromptList(&b, "Constraints — Compliance", brief.Constraints.Compliance)
	writePromptList(&b, "Constraints — Schedule", brief.Constraints.Schedule)
	writePromptList(&b, "Constraints — Budget", brief.Constraints.Budget)
	writePromptList(&b, "Environment — Stack", brief.Environment.Stack)
	writePromptSection(&b, "Environment — Repository", brief.Environment.Repository)
	writePromptList(&b, "Environment — Operating Systems", brief.Environment.OperatingSystems)
	writePromptList(&b, "Environment — Infrastructure", brief.Environment.Infrastructure)
	writePromptSection(&b, "Execution — Autonomy", brief.Execution.Autonomy)
	writePromptList(&b, "Execution — Allowed actions", brief.Execution.AllowedActions)
	writePromptList(&b, "Execution — Prohibited actions", brief.Execution.ProhibitedActions)
	writePromptList(&b, "Risks", brief.Risks)
	writePromptList(&b, "Alternatives rejected", brief.Alternatives)
	writePromptList(&b, "Acceptance Criteria", brief.Quality.AcceptanceCriteria)
	writePromptList(&b, "Testing", brief.Quality.Testing)
	writePromptList(&b, "Verification", brief.Quality.Verification)
	writePromptList(&b, "Review", brief.Quality.Review)
	writePromptList(&b, "Definition of Done", brief.Quality.DefinitionOfDone)
	writePromptSection(&b, "Deliverable — Type", brief.Deliverable.Type)
	writePromptSection(&b, "Deliverable — Format", brief.Deliverable.Format)
	writePromptList(&b, "Deliverables", brief.Deliverable.ExpectedArtifacts)
	writePromptList(&b, "Accepted gaps", acceptedGapLines(brief))
	return strings.TrimSpace(b.String())
}

// writeReferencedFiles renders explicitly attached project files with a short,
// already-redacted excerpt so the canonical prompt carries real context.
func writeReferencedFiles(b *strings.Builder, files []ContextFileRef) {
	if len(files) == 0 {
		return
	}
	b.WriteString("# Context — Referenced files\n")
	for _, file := range files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(path)
		if hash := strings.TrimSpace(file.Hash); hash != "" {
			b.WriteString(" (sha256:")
			b.WriteString(hash)
			b.WriteString(")")
		}
		b.WriteString("\n")
		excerpt := strings.TrimSpace(file.Excerpt)
		if excerpt == "" {
			continue
		}
		b.WriteString("```\n")
		b.WriteString(excerpt)
		if !strings.HasSuffix(excerpt, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n")
	}
	b.WriteString("\n")
}

func factsToLines(facts []ComposerFact) []string {
	out := make([]string, 0, len(facts))
	for _, fact := range facts {
		if strings.TrimSpace(fact.Value) != "" {
			out = append(out, strings.TrimSpace(fact.Value))
		}
	}
	return out
}

func acceptedGapLines(brief LivingBrief) []string {
	out := []string{}
	for _, unknown := range brief.Unknowns {
		switch unknown.Status {
		case PromptUnknownDismissed:
			out = append(out, firstNonEmpty(unknown.Field, unknown.ID)+": dismissed — "+firstNonEmpty(unknown.Answer, unknown.Question))
		case PromptUnknownConfirmed, PromptUnknownAnswered:
			if unknown.Severity == PromptUnknownOptional || unknown.Severity == PromptUnknownRecommended {
				continue
			}
		}
	}
	return out
}

func writePromptSection(b *strings.Builder, title, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(value)
	b.WriteString("\n\n")
}

func firstStatement(text string) string {
	for _, part := range splitStatements(text) {
		if part != "" {
			return part
		}
	}
	return ""
}

func splitStatements(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, ";", ".")
	raw := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '.' || r == '!' || r == '?' })
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func extractTaggedStatements(text string, tags []string) []string {
	lower := strings.ToLower(text)
	statements := splitStatements(text)
	out := []string{}
	for i, statement := range statements {
		candidate := strings.ToLower(statement)
		for _, tag := range tags {
			if strings.Contains(candidate, tag) || strings.Contains(lower, tag) && len(statements) == 1 {
				out = appendUnique(out, strings.TrimSpace(statements[i]))
				break
			}
		}
	}
	return out
}

func inferStackFromText(text string) []string {
	patterns := map[string]*regexp.Regexp{
		"Go":         regexp.MustCompile(`(?i)\bgo(lang)?\b`),
		"React":      regexp.MustCompile(`(?i)\breact\b`),
		"TypeScript": regexp.MustCompile(`(?i)\btypescript\b|\bts\b`),
		"PostgreSQL": regexp.MustCompile(`(?i)\bpostgres(ql)?\b`),
		"SQLite":     regexp.MustCompile(`(?i)\bsqlite\b`),
		"Docker":     regexp.MustCompile(`(?i)\bdocker\b`),
	}
	out := []string{}
	for label, pattern := range patterns {
		if pattern.MatchString(text) {
			out = append(out, label)
		}
	}
	return out
}

func inferPlatformsFromText(text string) []string {
	patterns := map[string]*regexp.Regexp{
		"Linux":   regexp.MustCompile(`(?i)\blinux\b`),
		"macOS":   regexp.MustCompile(`(?i)\bmacos\b|\bmac\b`),
		"Windows": regexp.MustCompile(`(?i)\bwindows\b`),
	}
	out := []string{}
	for label, pattern := range patterns {
		if pattern.MatchString(text) {
			out = append(out, label)
		}
	}
	return out
}

func inferAssumptions(brief LivingBrief) []PromptAssumption {
	out := []PromptAssumption{}
	if len(brief.Environment.Stack) > 0 {
		out = append(out, PromptAssumption{
			ID:         "environment.stack",
			Value:      "Stack appears to be " + strings.Join(brief.Environment.Stack, ", "),
			Confidence: "HIGH",
			Status:     "OPEN",
		})
	}
	return out
}

func upsertAssumption(items []PromptAssumption, candidate PromptAssumption) []PromptAssumption {
	if candidate.ID == "" || strings.TrimSpace(candidate.Value) == "" {
		return items
	}
	for i, item := range items {
		if item.ID == candidate.ID {
			items[i] = candidate
			return items
		}
	}
	return append(items, candidate)
}

func evaluateBriefField(brief LivingBrief, field string) (string, PromptUnknownStatus, string) {
	switch field {
	case "scope.in_scope":
		if len(brief.Scope.InScope) > 0 {
			return brief.Scope.InScope[0], PromptUnknownConfirmed, "HIGH"
		}
	case "scope.out_of_scope":
		if len(brief.Scope.OutOfScope) > 0 {
			return brief.Scope.OutOfScope[0], PromptUnknownConfirmed, "HIGH"
		}
	case "environment.stack":
		if len(brief.Environment.Stack) > 0 {
			return strings.Join(brief.Environment.Stack, ", "), PromptUnknownConfirmed, "HIGH"
		}
	case "environment.infrastructure":
		if len(brief.Environment.Infrastructure) > 0 {
			return strings.Join(brief.Environment.Infrastructure, ", "), PromptUnknownConfirmed, "HIGH"
		}
	case "quality.acceptance_criteria":
		if len(brief.Quality.AcceptanceCriteria) > 0 {
			return brief.Quality.AcceptanceCriteria[0], PromptUnknownConfirmed, "HIGH"
		}
	case "quality.testing":
		if len(brief.Quality.Testing) > 0 {
			return brief.Quality.Testing[0], PromptUnknownConfirmed, "HIGH"
		}
	case "quality.verification":
		if len(brief.Quality.Verification) > 0 {
			return brief.Quality.Verification[0], PromptUnknownConfirmed, "HIGH"
		}
	case "quality.definition_of_done":
		if len(brief.Quality.DefinitionOfDone) > 0 {
			return brief.Quality.DefinitionOfDone[0], PromptUnknownConfirmed, "HIGH"
		}
	case "context.existing_state":
		if len(brief.Context.ExistingState) > 0 {
			return brief.Context.ExistingState[0], PromptUnknownConfirmed, "HIGH"
		}
	case "risks":
		if len(brief.Risks) > 0 {
			return brief.Risks[0], PromptUnknownConfirmed, "HIGH"
		}
	case "execution.prohibited_actions":
		if len(brief.Execution.ProhibitedActions) > 0 {
			return brief.Execution.ProhibitedActions[0], PromptUnknownConfirmed, "HIGH"
		}
	case "deliverable.expected_artifacts":
		if len(brief.Deliverable.ExpectedArtifacts) > 0 {
			return brief.Deliverable.ExpectedArtifacts[0], PromptUnknownConfirmed, "HIGH"
		}
	case "constraints.security":
		if len(brief.Constraints.Security) > 0 {
			return brief.Constraints.Security[0], PromptUnknownConfirmed, "HIGH"
		}
	}
	return "", PromptUnknownOpen, ""
}

func commonDeliveryUnknowns() []composerUnknownBlueprint {
	return []composerUnknownBlueprint{
		{ID: "out-of-scope", Field: "scope.out_of_scope", Question: "O que está explicitamente fora do escopo desta entrega?", Rationale: "Fora de escopo evita expansão silenciosa.", Severity: PromptUnknownRecommended},
		{ID: "risks", Field: "risks", Question: "Quais riscos ou regressões devem ser evitados?", Rationale: "Riscos explícitos melhoram o prompt de execução.", Severity: PromptUnknownRecommended},
		{ID: "prohibited", Field: "execution.prohibited_actions", Question: "Quais ações o agente NÃO deve executar (commit, push, deploy, etc.)?", Rationale: "Ações proibidas protegem o repositório.", Severity: PromptUnknownRecommended},
		{ID: "dod", Field: "quality.definition_of_done", Question: "Qual é a definição de pronto (Definition of Done)?", Rationale: "DoD fecha o contrato da entrega.", Severity: PromptUnknownRecommended},
		{ID: "deliverables", Field: "deliverable.expected_artifacts", Question: "Quais artefatos devem ser entregues ao final?", Rationale: "Artefatos esperados tornam o resultado verificável.", Severity: PromptUnknownRecommended},
	}
}

func archetypeUnknowns(archetype PromptArchetype) []composerUnknownBlueprint {
	appendCommon := func(base []composerUnknownBlueprint) []composerUnknownBlueprint {
		return append(base, commonDeliveryUnknowns()...)
	}
	switch archetype {
	case PromptArchetypeBugFix:
		return appendCommon([]composerUnknownBlueprint{
			{ID: "current-bug", Field: "context.existing_state", Question: "Qual é o comportamento atual que está errado?", Rationale: "O prompt precisa descrever o bug real.", Severity: PromptUnknownBlocking},
			{ID: "fix-scope", Field: "scope.in_scope", Question: "Qual parte do sistema deve ser corrigida agora?", Rationale: "Escopo claro evita remediações amplas.", Severity: PromptUnknownRecommended},
			{ID: "regression-test", Field: "quality.testing", Question: "Qual teste de regressão deve provar a correção?", Rationale: "Bug fix sem teste regride com facilidade.", Severity: PromptUnknownBlocking},
			{ID: "verification", Field: "quality.verification", Question: "Como você quer verificar a correção ao final?", Rationale: "A entrega precisa de evidência final.", Severity: PromptUnknownRecommended},
		})
	case PromptArchetypeResearch:
		return appendCommon([]composerUnknownBlueprint{
			{ID: "research-scope", Field: "scope.in_scope", Question: "Qual recorte da pesquisa deve ser coberto?", Rationale: "Sem escopo a resposta fica genérica.", Severity: PromptUnknownBlocking},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Como você reconhecerá uma boa síntese?", Rationale: "Critérios evitam pesquisa extensa sem valor.", Severity: PromptUnknownRecommended},
			{ID: "verification", Field: "quality.verification", Question: "Que tipo de validação ou fontes você espera?", Rationale: "Pesquisa boa precisa de verificabilidade.", Severity: PromptUnknownBlocking},
		})
	case PromptArchetypeDevOps:
		return appendCommon([]composerUnknownBlueprint{
			{ID: "scope", Field: "scope.in_scope", Question: "Qual mudança de infraestrutura ou deploy está no escopo?", Rationale: "Escopo DevOps evita mudanças amplas.", Severity: PromptUnknownBlocking},
			{ID: "infra", Field: "environment.infrastructure", Question: "Qual infraestrutura, ambiente ou pipeline será alterado?", Rationale: "Infra concreta evita prompts genéricos.", Severity: PromptUnknownBlocking},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Como você saberá que o deploy/infra está correto?", Rationale: "Critério mensurável fecha a entrega.", Severity: PromptUnknownBlocking},
			{ID: "verification", Field: "quality.verification", Question: "Quais checks de saúde ou smoke tests validam o resultado?", Rationale: "Verificação operacional é obrigatória.", Severity: PromptUnknownRecommended},
		})
	case PromptArchetypeArchitecture:
		return appendCommon([]composerUnknownBlueprint{
			{ID: "scope", Field: "scope.in_scope", Question: "Qual decisão ou módulo arquitetural está no escopo?", Rationale: "Escopo evita redesign amplo.", Severity: PromptUnknownBlocking},
			{ID: "stack", Field: "environment.stack", Question: "Quais componentes ou camadas técnicas entram na análise?", Rationale: "Contexto técnico ancora a decisão.", Severity: PromptUnknownRecommended},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Como uma boa decisão arquitetural será reconhecida?", Rationale: "Critérios evitam parecer genérico.", Severity: PromptUnknownBlocking},
			{ID: "verification", Field: "quality.verification", Question: "Que evidências ou ADRs devem documentar a decisão?", Rationale: "Arquitetura precisa de registro verificável.", Severity: PromptUnknownRecommended},
		})
	case PromptArchetypeSecurity:
		return appendCommon([]composerUnknownBlueprint{
			{ID: "scope", Field: "scope.in_scope", Question: "Qual superfície de ataque ou controle de segurança será tratado?", Rationale: "Escopo de segurança precisa ser preciso.", Severity: PromptUnknownBlocking},
			{ID: "security", Field: "constraints.security", Question: "Quais restrições de segurança, auth ou compliance são obrigatórias?", Rationale: "Restrições de segurança não podem ser inferidas.", Severity: PromptUnknownBlocking},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Qual resultado de segurança define sucesso?", Rationale: "Aceite de segurança precisa ser observável.", Severity: PromptUnknownBlocking},
			{ID: "testing", Field: "quality.testing", Question: "Quais testes ou scans de segurança devem acompanhar a entrega?", Rationale: "Segurança sem teste é regressão.", Severity: PromptUnknownRecommended},
			{ID: "verification", Field: "quality.verification", Question: "Como a correção de segurança será verificada?", Rationale: "Verificação fecha o contrato.", Severity: PromptUnknownRecommended},
		})
	default:
		return appendCommon([]composerUnknownBlueprint{
			{ID: "scope", Field: "scope.in_scope", Question: "Qual parte desta entrega está dentro do escopo imediato?", Rationale: "Escopo explícito evita expansão silenciosa.", Severity: PromptUnknownBlocking},
			{ID: "stack", Field: "environment.stack", Question: "Qual stack, serviço ou área técnica será alterada?", Rationale: "A estratégia muda com a stack real.", Severity: PromptUnknownRecommended},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Qual resultado mensurável define que isso está pronto?", Rationale: "Sem critério mensurável o prompt final fica fraco.", Severity: PromptUnknownBlocking},
			{ID: "testing", Field: "quality.testing", Question: "Quais testes ou evidências técnicas devem acompanhar a entrega?", Rationale: "Composer precisa embutir qualidade e regressão.", Severity: PromptUnknownRecommended},
			{ID: "verification", Field: "quality.verification", Question: "Como a execução deve ser verificada no final?", Rationale: "Verificação final fecha o contrato do prompt.", Severity: PromptUnknownBlocking},
		})
	}
}

func classifyPromptArchetype(text string) PromptArchetype {
	text = strings.ToLower(text)
	switch {
	case strings.Contains(text, "bug") || strings.Contains(text, "fix") || strings.Contains(text, "erro") || strings.Contains(text, "corrigir"):
		return PromptArchetypeBugFix
	case strings.Contains(text, "research") || strings.Contains(text, "pesquisa") || strings.Contains(text, "analisar"):
		return PromptArchetypeResearch
	case strings.Contains(text, "oauth") || strings.Contains(text, "security") || strings.Contains(text, "csrf") || strings.Contains(text, "seguran"):
		return PromptArchetypeSecurity
	case strings.Contains(text, "infra") || strings.Contains(text, "deploy") || strings.Contains(text, "docker") || strings.Contains(text, "kubern"):
		return PromptArchetypeDevOps
	case strings.Contains(text, "architecture") || strings.Contains(text, "arquitet"):
		return PromptArchetypeArchitecture
	default:
		return PromptArchetypeSoftwareFeature
	}
}

func readinessLabel(field string) string {
	switch field {
	case "scope.in_scope":
		return "Scope"
	case "scope.out_of_scope":
		return "Out of scope"
	case "environment.stack":
		return "Stack"
	case "environment.infrastructure":
		return "Infrastructure"
	case "quality.acceptance_criteria":
		return "Acceptance criteria"
	case "quality.testing":
		return "Testing"
	case "quality.verification":
		return "Verification"
	case "quality.definition_of_done":
		return "Definition of Done"
	case "context.existing_state":
		return "Current state"
	case "risks":
		return "Risks"
	case "execution.prohibited_actions":
		return "Prohibited actions"
	case "deliverable.expected_artifacts":
		return "Deliverables"
	case "constraints.security":
		return "Security constraints"
	default:
		return field
	}
}

func roleForArchetype(archetype PromptArchetype) string {
	switch archetype {
	case PromptArchetypeBugFix:
		return "Act as a senior software engineer focused on root-cause fixes and regression safety."
	case PromptArchetypeResearch:
		return "Act as a rigorous research assistant and synthesize only what the evidence supports."
	case PromptArchetypeDevOps:
		return "Act as a senior DevOps engineer and keep infrastructure changes minimal, reversible, and verified."
	case PromptArchetypeArchitecture:
		return "Act as a software architect and document tradeoffs with concrete evidence."
	case PromptArchetypeSecurity:
		return "Act as a security-focused engineer and never weaken authentication, authorization, or secret handling."
	default:
		return "Act as a senior software engineer and keep the work within the confirmed scope."
	}
}

func flattenAssumptionsToLines(items []PromptAssumption) []string {
	out := []string{}
	for _, item := range items {
		if strings.TrimSpace(item.Value) != "" {
			out = append(out, item.Value+" (confidence: "+strings.ToLower(item.Confidence)+")")
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
