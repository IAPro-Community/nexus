package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CLIPromptRunner executes one non-interactive prompt using a capability-validated
// provider/profile adapter supplied by the Nexus runtime layer.
type CLIPromptRunner func(ctx context.Context, prompt string) (string, error)

// CLIProvider turns an existing coding-agent CLI into a Nexus Intelligence source.
// The runtime adapter is responsible for validating Headless + SubmitPrompt support
// before setting capabilityValidated=true.
type CLIProvider struct {
	providerID          string
	profile             string
	capabilityValidated bool
	run                 CLIPromptRunner
}

// HeadlessPromptArgs returns the verified non-interactive invocation contract for
// coding-agent CLIs that currently advertise both Headless and SubmitPrompt support.
func HeadlessPromptArgs(providerID, prompt string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "claude", "agy", "gemini", "cursor":
		return []string{"-p", prompt}, nil
	case "codex":
		return []string{"exec", prompt}, nil
	case "opencode":
		return []string{"run", prompt}, nil
	default:
		return nil, fmt.Errorf("provider %q has no verified headless prompt contract", providerID)
	}
}

func NewCLIProvider(providerID, profile string, capabilityValidated bool, run CLIPromptRunner) *CLIProvider {
	return &CLIProvider{providerID: strings.TrimSpace(providerID), profile: strings.TrimSpace(profile), capabilityValidated: capabilityValidated, run: run}
}

func (p *CLIProvider) Name() string {
	if p.profile == "" {
		return "cli:" + p.providerID
	}
	return "cli:" + p.providerID + ":" + p.profile
}

func (p *CLIProvider) Available(context.Context) bool {
	return p != nil && p.providerID != "" && p.profile != "" && p.capabilityValidated && p.run != nil
}

func (p *CLIProvider) AnalyzeIntent(ctx context.Context, input string, contextData map[string]any) (*IntentAnalysis, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose.
You are the Nexus product discovery and planning consultant for novices and experts.
Explain the underlying need as concrete goals. Recommend sensible steps from evidence and label assumptions.
Do not demand architecture, scope exclusions or test details from non-technical users.
Analyze this objective and produce:
{"intent":"...","scope":"project|mission|package|task","risk_level":"low|medium|high","identified_goals":["..."],"constraints":["..."],"assumptions":["..."]}
Objective: %s
Context: %s`, input, boundContextJSON(contextData))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s analyze intent: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s analyze intent: %w", p.Name(), err)
	}
	var result IntentAnalysis
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s analyze intent: %w", p.Name(), err)
	}
	result.CreatedAt = time.Now().UTC()
	return &result, nil
}

func (p *CLIProvider) EvaluateAmbiguities(ctx context.Context, intent *IntentAnalysis) ([]AmbiguityItem, error) {
	return p.EvaluateAmbiguitiesWithContext(ctx, intent, nil)
}

func (p *CLIProvider) EvaluateAmbiguitiesWithContext(ctx context.Context, intent *IntentAnalysis, contextData map[string]any) ([]AmbiguityItem, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	intentJSON, _ := json.Marshal(intent)
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose.
Identify only material choices which are genuinely impossible to infer from the goal and available context.
Return {"unknowns":[]} when the user can safely proceed with explicit proposed assumptions.
Never ask generic checklist questions (out of scope, Definition of Done, tech stack, risks, deliverables or tests).
Use simple language the user understands and provide recommended options for real choices.
BLOCKING is reserved for non-reversible, high-impact or safety-critical choices; IMPORTANT materially changes design; LOW_IMPACT has a safe default.
Schema: {"unknowns":[{"key":"...","level":"BLOCKING|IMPORTANT|LOW_IMPACT","question":"...","rationale":"...","suggested_options":["..."],"default_choice":"..."}]}
Intent: %s
Project context: %s`, string(intentJSON), boundContextJSON(contextData))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s ambiguity analysis: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s ambiguity analysis: %w", p.Name(), err)
	}
	var result struct {
		Unknowns []AmbiguityItem `json:"unknowns"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s ambiguity analysis: %w", p.Name(), err)
	}
	return result.Unknowns, nil
}

func (p *CLIProvider) GeneratePlanOutline(ctx context.Context, intent *IntentAnalysis, facts map[string]string, contextData map[string]any) ([]WorkPackageOutline, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	input := boundContextJSON(map[string]any{"intent": intent, "confirmed_facts": facts, "project_context": contextData})
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose.
Decompose the engineering objective into independently reviewable WorkPackages. Do NOT invent Maestro skill identifiers. The skills array must be empty; Maestro is a separate authority.
Use real titles and goals derived from the objective. Never copy schema punctuation or field names as values.
Break a complex prompt into distinct domain-specific deliverables, not generic Planning/Implementation/Review labels. Split independent backend, frontend, data, integration and verification work when the prompt requires them.
Preserve real prerequisites and explicit opportunities for parallel execution. Acceptance must be observable; do not invent a successful test or inspected repository evidence.
For an atomic task, one WorkPackage is appropriate. Do not invent work just to increase the count.
dependencies must be titles of OTHER packages in this same list, or [].
Schema: {"packages":[{"title":"short unique name","goal":"measurable objective","priority":"CRITICAL|HIGH|NORMAL|LOW","dependencies":[],"role":"implementer|reviewer|tester|architect","skills":[],"acceptance":["done when ..."]}]}
Input: %s`, input)
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s plan generation: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s plan generation: %w", p.Name(), err)
	}
	var result struct {
		Packages []WorkPackageOutline `json:"packages"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s plan generation: %w", p.Name(), err)
	}
	for i := range result.Packages {
		result.Packages[i].Skills = nil
	}
	return result.Packages, nil
}

// ExtractBrief performs a single headless JSON oneshot that maps the prompt
// into structured brief fields. When the CLI backend is not capability-validated
// it returns an empty extraction with a nil error so the caller falls back to
// deterministic heuristics instead of surfacing an error.
func (p *CLIProvider) ExtractBrief(ctx context.Context, text string, contextData map[string]any) (*BriefExtraction, error) {
	if !p.Available(ctx) {
		return &BriefExtraction{}, nil
	}
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose or markdown fences.
You are a consultative product planner for non-technical and expert users.
Extract only grounded requirements from the prompt/context; do not invent stack or repository facts.
Separately propose 1–3 concise solution approaches understandable to the user, explaining benefits or trade-offs.
Recommendations are proposals, NEVER facts or user-approved scope. Other arrays remain empty unless grounded.
Identify at most one material user decision that cannot safely be inferred; return an empty clarifications array otherwise. Never ask generic checklist questions.
Schema:
{"discovery_complete":true,"clarifications":[{"key":"...","level":"BLOCKING|IMPORTANT|LOW_IMPACT","question":"...","rationale":"...","suggested_options":["..."],"default_choice":"..."}],"recommended_approaches":["Proposed approach and why (not verified)"],"in_scope":["..."],"out_of_scope":["..."],"constraints_technical":["..."],"constraints_security":["..."],"risks":["..."],"acceptance_criteria":["..."],"testing":["..."],"verification":["..."],"prohibited_actions":["..."],"deliverables":["..."],"stack":["..."],"existing_state":["..."]}
Prompt: %s
Context: %s`, text, boundContextJSON(contextData))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("run %s extract brief: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, fmt.Errorf("decode %s extract brief: %w", p.Name(), err)
	}
	var result BriefExtraction
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("decode %s extract brief: %w", p.Name(), err)
	}
	return &result, nil
}

// PlanFromGoal performs a single headless CLI call that returns intent, unknowns, and packages.
func (p *CLIProvider) PlanFromGoal(ctx context.Context, goal string, contextData map[string]any) (*IntentAnalysis, []AmbiguityItem, []WorkPackageOutline, error) {
	if !p.Available(ctx) {
		return nil, nil, nil, ErrIntelligenceUnavailable
	}
	prompt := fmt.Sprintf(`Return ONLY a JSON object. Do not wrap it in prose or markdown fences.
You are the Nexus software architecture intelligence layer.
Given the engineering objective, produce intent analysis, requirement unknowns, and independently reviewable WorkPackages in ONE response.
Do NOT invent Maestro skill identifiers; packages.skills must be [].
BLOCKING unknowns mean execution cannot safely continue; IMPORTANT changes design; LOW_IMPACT has a safe default_choice.
Never copy schema examples, ellipses, or schema field names as real values.
packages[].title and packages[].goal must be concrete strings derived from the objective.
For multi-workstream prompts, break down distinct backend/frontend/data/integration/verification deliverables and explicit dependencies. Do not return one generic activity or a boilerplate fixed phase sequence.
Keep independent workstreams parallel when safe. Acceptance criteria must be measurable, not claims of tests that have already passed.
An atomic request can produce a single activity; never invent extra work solely for cardinality.
packages[].dependencies is an array of titles of OTHER packages in this same list. Use [] when there is no prerequisite.
Schema:
{
  "intent":"<restated objective>",
  "scope":"project|mission|package|task",
  "risk_level":"low|medium|high",
  "identified_goals":["<goal>"],
  "constraints":[],
  "assumptions":[],
  "unknowns":[{"key":"<id>","level":"BLOCKING|IMPORTANT|LOW_IMPACT","question":"<question>","rationale":"<why>","suggested_options":["<option>"],"default_choice":"<option>"}],
  "packages":[{"title":"<short unique name>","goal":"<measurable objective>","priority":"CRITICAL|HIGH|NORMAL|LOW","dependencies":[],"role":"implementer|reviewer|tester|architect","skills":[],"acceptance":["<done when>"]}]
}
Objective: %s
Context: %s`, goal, boundContextJSON(contextData))
	out, err := p.run(ctx, prompt)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("run %s plan-from-goal: %w", p.Name(), err)
	}
	payload, err := extractJSONObject(out)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decode %s plan-from-goal: %w", p.Name(), err)
	}
	var result struct {
		IntentAnalysis
		Unknowns []AmbiguityItem      `json:"unknowns"`
		Packages []WorkPackageOutline `json:"packages"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, nil, nil, fmt.Errorf("decode %s plan-from-goal: %w", p.Name(), err)
	}
	intent := result.IntentAnalysis
	intent.CreatedAt = time.Now().UTC()
	if strings.TrimSpace(intent.Intent) == "" {
		intent.Intent = strings.TrimSpace(goal)
	}
	for i := range result.Packages {
		result.Packages[i].Skills = nil
	}
	return &intent, result.Unknowns, result.Packages, nil
}

// boundContextJSON serializes context for CLI prompts with a hard byte ceiling.
func boundContextJSON(contextData map[string]any) string {
	if contextData == nil {
		return "{}"
	}
	raw, err := json.Marshal(contextData)
	if err != nil {
		return "{}"
	}
	if len(raw) <= MaxCLIContextBytes {
		return string(raw)
	}
	trimmed := append(append([]byte(nil), raw[:MaxCLIContextBytes]...), []byte("…[truncated]")...)
	return string(trimmed)
}

func extractJSONObject(out string) ([]byte, error) {
	trimmed := strings.TrimSpace(out)
	if strings.HasPrefix(trimmed, "```") {
		if nl := strings.Index(trimmed, "\n"); nl >= 0 {
			trimmed = trimmed[nl+1:]
		}
		if end := strings.LastIndex(trimmed, "```"); end >= 0 {
			trimmed = trimmed[:end]
		}
		trimmed = strings.TrimSpace(trimmed)
	}
	start := strings.Index(trimmed, "{")
	if start < 0 {
		return nil, fmt.Errorf("no JSON object in provider output")
	}
	// Decode the first top-level JSON value so trailing prose (e.g. "OK ... {hint}")
	// cannot poison Unmarshal with "invalid character 'O' after top-level value".
	dec := json.NewDecoder(strings.NewReader(trimmed[start:]))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("no JSON object in provider output: %w", err)
	}
	if len(raw) == 0 || raw[0] != '{' {
		return nil, fmt.Errorf("no JSON object in provider output")
	}
	return []byte(raw), nil
}
