package nexus

import (
	"math"
	"regexp"
	"strings"
)

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

type BriefContext struct {
	Domain        string   `json:"domain,omitempty"`
	Project       string   `json:"project,omitempty"`
	Audience      string   `json:"audience,omitempty"`
	ExistingState []string `json:"existing_state,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
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
	ID            string                `json:"id"`
	Field         string                `json:"field"`
	Question      string                `json:"question"`
	Rationale     string                `json:"rationale"`
	Severity      PromptUnknownSeverity `json:"severity"`
	Status        PromptUnknownStatus   `json:"status"`
	Answer        string                `json:"answer,omitempty"`
	InferredValue string                `json:"inferred_value,omitempty"`
	Confidence    string                `json:"confidence,omitempty"`
	Source        string                `json:"source,omitempty"`
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
	Goal          string             `json:"goal"`
	SourcePrompt  string             `json:"source_prompt,omitempty"`
	Intent        BriefIntent        `json:"intent"`
	Context       BriefContext       `json:"context"`
	Deliverable   BriefDeliverable   `json:"deliverable"`
	Scope         BriefScope         `json:"scope"`
	Environment   BriefEnvironment   `json:"environment"`
	Constraints   BriefConstraints   `json:"constraints"`
	Execution     BriefExecution     `json:"execution"`
	Quality       BriefQuality       `json:"quality"`
	Decisions     []string           `json:"decisions,omitempty"`
	Assumptions   []PromptAssumption `json:"assumptions,omitempty"`
	Alternatives  []string           `json:"alternatives,omitempty"`
	Risks         []string           `json:"risks,omitempty"`
	Unknowns      []PromptUnknown    `json:"unknowns,omitempty"`
	OpenQuestions []string           `json:"open_questions,omitempty"`
	Readiness     PromptReadiness    `json:"readiness"`
}

type composerUnknownBlueprint struct {
	ID        string
	Field     string
	Question  string
	Rationale string
	Severity  PromptUnknownSeverity
}

func newComposerBrief(goal, sourcePrompt string) LivingBrief {
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
	}
	mergeTextIntoBrief(&brief, body, "USER")
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
	brief.OpenQuestions = nil
	for _, unknown := range brief.Unknowns {
		if unknown.Status == PromptUnknownOpen {
			brief.OpenQuestions = append(brief.OpenQuestions, unknown.Question)
		}
	}
	brief.Readiness = computePromptReadiness(*brief)
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
	blueprints := archetypeUnknowns(brief.Intent.Archetype)
	current := map[string]PromptUnknown{}
	for _, item := range brief.Unknowns {
		current[item.Field] = item
	}
	out := make([]PromptUnknown, 0, len(blueprints))
	for _, bp := range blueprints {
		item := current[bp.Field]
		if item.ID == "" {
			item = PromptUnknown{
				ID:        bp.ID,
				Field:     bp.Field,
				Question:  bp.Question,
				Rationale: bp.Rationale,
				Severity:  bp.Severity,
				Source:    "INTELLIGENCE",
			}
		}
		item.Question = bp.Question
		item.Rationale = bp.Rationale
		item.Severity = bp.Severity
		summary, status, confidence := evaluateBriefField(brief, bp.Field)
		switch status {
		case PromptUnknownConfirmed, PromptUnknownAnswered:
			item.Status = status
			item.Answer = summary
			item.InferredValue = ""
			item.Confidence = confidence
			item.Source = "USER"
		case PromptUnknownInferred:
			item.Status = PromptUnknownInferred
			item.Answer = ""
			item.InferredValue = summary
			item.Confidence = confidence
			item.Source = "PROJECT_CONTEXT"
		default:
			item.Status = PromptUnknownOpen
			item.Answer = ""
			item.InferredValue = ""
			item.Confidence = ""
		}
		out = append(out, item)
	}
	return out
}

func computePromptReadiness(brief LivingBrief) PromptReadiness {
	checks := []PromptReadinessCheck{}
	completed := 0.0
	blockingOpen := false
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
			if bp.Severity == PromptUnknownBlocking {
				blockingOpen = true
			}
		}
		checks = append(checks, check)
	}
	total := float64(len(checks))
	score := 0
	if total > 0 {
		score = int(math.Round((completed / total) * 100))
	}
	state := PromptReadinessNeedsInformation
	switch {
	case total == 0:
		state = PromptReadinessReady
	case blockingOpen:
		state = PromptReadinessBlocked
	case score >= 90 && !inferredOnly:
		state = PromptReadinessReady
	case score >= 80:
		state = PromptReadinessReadyWithAssumption
	}
	summary := "Composer still needs more requirements."
	switch state {
	case PromptReadinessReady:
		summary = "Prompt is ready to finalize."
	case PromptReadinessReadyWithAssumption:
		summary = "Prompt is usable, but some details were inferred."
	case PromptReadinessBlocked:
		summary = "Blocking gaps remain before the final prompt is trustworthy."
	}
	return PromptReadiness{Score: score, State: state, Summary: summary, Checks: checks}
}

func renderCanonicalPrompt(brief LivingBrief) string {
	var b strings.Builder
	writePromptSection(&b, "Role", roleForArchetype(brief.Intent.Archetype))
	writePromptSection(&b, "Objective", firstNonEmpty(strings.TrimSpace(brief.Intent.Objective), strings.TrimSpace(brief.Goal)))
	writePromptList(&b, "Context", append(append([]string(nil), brief.Context.Evidence...), brief.Context.ExistingState...))
	writePromptList(&b, "Requirements", brief.Scope.InScope)
	writePromptList(&b, "Out of scope", brief.Scope.OutOfScope)
	writePromptList(&b, "Constraints", append(append(append([]string(nil), brief.Constraints.Technical...), brief.Constraints.Product...), brief.Constraints.Security...))
	writePromptList(&b, "Architecture / Technical Context", brief.Environment.Stack)
	writePromptList(&b, "Skills", flattenAssumptionsToLines(brief.Assumptions))
	writePromptList(&b, "Acceptance Criteria", brief.Quality.AcceptanceCriteria)
	writePromptList(&b, "Testing", brief.Quality.Testing)
	writePromptList(&b, "Verification", brief.Quality.Verification)
	writePromptList(&b, "Deliverables", brief.Deliverable.ExpectedArtifacts)
	return strings.TrimSpace(b.String())
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
	case "environment.stack":
		if len(brief.Environment.Stack) > 0 {
			return strings.Join(brief.Environment.Stack, ", "), PromptUnknownConfirmed, "HIGH"
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
	case "context.existing_state":
		if len(brief.Context.ExistingState) > 0 {
			return brief.Context.ExistingState[0], PromptUnknownConfirmed, "HIGH"
		}
	}
	return "", PromptUnknownOpen, ""
}

func archetypeUnknowns(archetype PromptArchetype) []composerUnknownBlueprint {
	switch archetype {
	case PromptArchetypeBugFix:
		return []composerUnknownBlueprint{
			{ID: "current-bug", Field: "context.existing_state", Question: "Qual é o comportamento atual que está errado?", Rationale: "O prompt precisa descrever o bug real.", Severity: PromptUnknownBlocking},
			{ID: "fix-scope", Field: "scope.in_scope", Question: "Qual parte do sistema deve ser corrigida agora?", Rationale: "Escopo claro evita remediações amplas.", Severity: PromptUnknownRecommended},
			{ID: "regression-test", Field: "quality.testing", Question: "Qual teste de regressão deve provar a correção?", Rationale: "Bug fix sem teste regride com facilidade.", Severity: PromptUnknownBlocking},
			{ID: "verification", Field: "quality.verification", Question: "Como você quer verificar a correção ao final?", Rationale: "A entrega precisa de evidência final.", Severity: PromptUnknownRecommended},
		}
	case PromptArchetypeResearch:
		return []composerUnknownBlueprint{
			{ID: "research-scope", Field: "scope.in_scope", Question: "Qual recorte da pesquisa deve ser coberto?", Rationale: "Sem escopo a resposta fica genérica.", Severity: PromptUnknownBlocking},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Como você reconhecerá uma boa síntese?", Rationale: "Critérios evitam pesquisa extensa sem valor.", Severity: PromptUnknownRecommended},
			{ID: "verification", Field: "quality.verification", Question: "Que tipo de validação ou fontes você espera?", Rationale: "Pesquisa boa precisa de verificabilidade.", Severity: PromptUnknownBlocking},
		}
	default:
		return []composerUnknownBlueprint{
			{ID: "scope", Field: "scope.in_scope", Question: "Qual parte desta entrega está dentro do escopo imediato?", Rationale: "Escopo explícito evita expansão silenciosa.", Severity: PromptUnknownBlocking},
			{ID: "stack", Field: "environment.stack", Question: "Qual stack, serviço ou área técnica será alterada?", Rationale: "A estratégia muda com a stack real.", Severity: PromptUnknownRecommended},
			{ID: "acceptance", Field: "quality.acceptance_criteria", Question: "Qual resultado mensurável define que isso está pronto?", Rationale: "Sem critério mensurável o prompt final fica fraco.", Severity: PromptUnknownBlocking},
			{ID: "testing", Field: "quality.testing", Question: "Quais testes ou evidências técnicas devem acompanhar a entrega?", Rationale: "Composer precisa embutir qualidade e regressão.", Severity: PromptUnknownRecommended},
			{ID: "verification", Field: "quality.verification", Question: "Como a execução deve ser verificada no final?", Rationale: "Verificação final fecha o contrato do prompt.", Severity: PromptUnknownBlocking},
		}
	}
}

func classifyPromptArchetype(text string) PromptArchetype {
	text = strings.ToLower(text)
	switch {
	case strings.Contains(text, "bug") || strings.Contains(text, "fix") || strings.Contains(text, "erro") || strings.Contains(text, "corrigir"):
		return PromptArchetypeBugFix
	case strings.Contains(text, "research") || strings.Contains(text, "pesquisa") || strings.Contains(text, "analisar"):
		return PromptArchetypeResearch
	case strings.Contains(text, "oauth") || strings.Contains(text, "security") || strings.Contains(text, "csrf"):
		return PromptArchetypeSoftwareFeature
	case strings.Contains(text, "infra") || strings.Contains(text, "deploy") || strings.Contains(text, "docker"):
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
	case "environment.stack":
		return "Stack"
	case "quality.acceptance_criteria":
		return "Acceptance criteria"
	case "quality.testing":
		return "Testing"
	case "quality.verification":
		return "Verification"
	case "context.existing_state":
		return "Current state"
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
