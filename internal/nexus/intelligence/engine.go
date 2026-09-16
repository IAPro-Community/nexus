package intelligence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// NexusEngine implements the high-level IntelligenceEngine interface.
type NexusEngine struct {
	provider    IntelligenceProvider
	contextData map[string]any
}

// ErrIntelligenceUnavailable is returned when Composer analysis/planning is requested without a real provider.
var ErrIntelligenceUnavailable = errors.New("nexus intelligence unavailable")

// IsTransientIntelligenceFailure reports CLI/provider failures that should soft-fail
// back to heuristics without dumping raw process errors into the chat transcript
// (OOM SIGKILL, context deadline, canceled run).
func IsTransientIntelligenceFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"signal: killed",
		"signal:killed",
		"killed",
		"out of memory",
		"oom",
		"context deadline exceeded",
		"context canceled",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// NewNexusEngine creates an engine wrapping the explicitly configured provider.
// A nil provider is valid for provider-independent operations such as prompt compilation,
// but Analyze/GeneratePlan fail closed instead of fabricating intelligence output.
func NewNexusEngine(p IntelligenceProvider) *NexusEngine {
	return &NexusEngine{provider: p}
}

// WithContextData binds a bounded project context envelope to subsequent
// analysis and plan-generation calls. The engine does not read the repository
// itself and therefore cannot accidentally expand the context surface.
func (e *NexusEngine) WithContextData(data map[string]any) *NexusEngine {
	if e == nil {
		return e
	}
	copyData := make(map[string]any, len(data))
	for key, value := range data {
		copyData[key] = value
	}
	e.contextData = copyData
	return e
}

func (e *NexusEngine) mergedContext(projectID string) map[string]any {
	merged := make(map[string]any, len(e.contextData)+1)
	for key, value := range e.contextData {
		merged[key] = value
	}
	merged["project_id"] = projectID
	return merged
}

func (e *NexusEngine) Analyze(ctx context.Context, goal string, projectID string) (*IntentAnalysis, []AmbiguityItem, error) {
	if e.provider == nil || !e.provider.Available(ctx) {
		return nil, nil, ErrIntelligenceUnavailable
	}
	intent, err := e.provider.AnalyzeIntent(ctx, goal, e.mergedContext(projectID))
	if err != nil {
		return nil, nil, err
	}
	var unknowns []AmbiguityItem
	var unknownErr error
	if contextual, ok := e.provider.(ContextualAmbiguityEvaluator); ok {
		unknowns, unknownErr = contextual.EvaluateAmbiguitiesWithContext(ctx, intent, e.mergedContext(projectID))
	} else {
		unknowns, unknownErr = e.provider.EvaluateAmbiguities(ctx, intent)
	}
	if unknownErr != nil {
		return intent, nil, unknownErr
	}
	return intent, unknowns, nil
}

// PlanFromGoal prefers a single-shot provider call when available (CLI), otherwise
// falls back to Analyze + GeneratePlan (OpenAI-compatible multi-call).
func (e *NexusEngine) PlanFromGoal(ctx context.Context, goal string, projectID string) (*IntentAnalysis, []AmbiguityItem, []WorkPackageOutline, error) {
	if e.provider == nil || !e.provider.Available(ctx) {
		return nil, nil, nil, ErrIntelligenceUnavailable
	}
	if oneshot, ok := e.provider.(OneshotPlanner); ok {
		intent, unknowns, packages, err := oneshot.PlanFromGoal(ctx, goal, e.mergedContext(projectID))
		if err != nil {
			return nil, nil, nil, err
		}
		if len(packages) == 0 {
			return intent, unknowns, nil, fmt.Errorf("intelligence provider %s returned an empty work plan", e.provider.Name())
		}
		return intent, unknowns, packages, nil
	}
	intent, unknowns, err := e.Analyze(ctx, goal, projectID)
	if err != nil {
		return nil, nil, nil, err
	}
	return intent, unknowns, nil, nil
}

// GeneratePlan asks the configured provider for a structured outline. It never
// falls back to locally fabricated work packages.
func (e *NexusEngine) GeneratePlan(ctx context.Context, intent *IntentAnalysis, facts map[string]string) ([]WorkPackageOutline, error) {
	if e.provider == nil || !e.provider.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}
	packages, err := e.provider.GeneratePlanOutline(ctx, intent, facts, e.contextData)
	if err != nil {
		return nil, err
	}
	if len(packages) == 0 {
		return nil, fmt.Errorf("intelligence provider %s returned an empty work plan", e.provider.Name())
	}
	return packages, nil
}

// Probe runs a minimal AnalyzeIntent round-trip to prove the provider responds.
func (e *NexusEngine) Probe(ctx context.Context) error {
	if e.provider == nil || !e.provider.Available(ctx) {
		return ErrIntelligenceUnavailable
	}
	probeCtx, cancel := context.WithTimeout(ctx, IntelligenceProbeTimeout)
	defer cancel()
	_, err := e.provider.AnalyzeIntent(probeCtx, "ping", map[string]any{"probe": true})
	return err
}

func (e *NexusEngine) ResolveClarification(state *ClarificationState, key string, answer string) {
	if state.StructuredFacts == nil {
		state.StructuredFacts = make(map[string]string)
	}
	state.StructuredFacts[key] = answer

	allDone := true
	for i := range state.Unknowns {
		if state.Unknowns[i].Key == key {
			state.Unknowns[i].Answer = answer
			state.Unknowns[i].IsResolved = true
		}
		if state.Unknowns[i].Level == AmbiguityBlocking && !state.Unknowns[i].IsResolved {
			allDone = false
		}
	}
	state.AllBlockingDone = allDone
}

func (e *NexusEngine) CompilePrompt(
	ctx context.Context,
	pkg WorkPackageOutline,
	facts map[string]string,
	skillIDs []string,
) (*PromptCompilationResult, error) {
	factsList := formatFacts(facts)
	compiled, err := e.CompileExecutionContext(ctx, ExecutionContextRequest{
		Agent:   AgentSpec{Role: pkg.Role},
		Project: ProjectContext{Facts: facts},
		Task:    WorkPackageContext{Title: pkg.Title, Goal: pkg.Goal, Priority: pkg.Priority, Role: pkg.Role, AcceptanceCriteria: append([]string(nil), pkg.Acceptance...)},
		Skills:  append([]string(nil), skillIDs...),
	})
	if err != nil {
		return nil, err
	}
	sysPrompt := compiled.SystemInstructions
	userPrompt := compiled.TaskInstructions

	estTokens := (len(sysPrompt) + len(userPrompt)) / 4

	return &PromptCompilationResult{
		PackageTitle:    pkg.Title,
		SystemPrompt:    sysPrompt,
		UserPrompt:      userPrompt,
		Skills:          append([]string(nil), skillIDs...),
		AcceptanceGates: pkg.Acceptance,
		Constraints:     factsList,
		EstimatedTokens: estTokens,
		CompiledAt:      time.Now().UTC(),
	}, nil
}

func (e *NexusEngine) CompileExecutionContext(_ context.Context, req ExecutionContextRequest) (*CompiledExecutionContext, error) {
	if strings.TrimSpace(req.Task.Title) == "" && strings.TrimSpace(req.Task.Goal) == "" {
		return nil, errors.New("execution task is required")
	}
	facts := formatFacts(req.Project.Facts)
	sections := []ContextSection{
		{Source: "agent", Name: "persistent specialization", Content: formatAgentSpec(req.Agent)},
		{Source: "task", Name: "work package role", Content: fmt.Sprintf("Role: %s\nTitle: %s\nGoal: %s", req.Task.Role, req.Task.Title, req.Task.Goal)},
	}
	if len(facts) > 0 {
		sections = append(sections, ContextSection{Source: "project", Name: "project facts", Content: strings.Join(facts, "\n")})
	}
	guidance := req.Guidance
	skillIDs := req.Skills
	if len(skillIDs) == 0 {
		skillIDs = guidance.Skills
	}
	if len(skillIDs) > 0 || guidance.Enabled && len(guidance.Instructions) > 0 {
		instructions := append([]string{}, guidance.Instructions...)
		content := append(instructions, skillIDs...)
		sections = append(sections, ContextSection{Source: "guidance", Name: "execution guidance", Content: strings.Join(content, "\n")})
	}
	if req.Runtime.Provider != "" || req.Runtime.Model != "" || req.Runtime.Workspace != "" || req.Runtime.Isolation != "" || len(req.Runtime.Capabilities) > 0 {
		runtimeContent := fmt.Sprintf("Provider: %s\nModel: %s\nWorkspace: %s\nIsolation: %s\nCapabilities: %s", req.Runtime.Provider, req.Runtime.Model, req.Runtime.Workspace, req.Runtime.Isolation, strings.Join(req.Runtime.Capabilities, ", "))
		sections = append(sections, ContextSection{Source: "runtime", Name: "runtime constraints", Content: runtimeContent})
	}

	sectionText := make([]string, 0, len(sections))
	for _, section := range sections {
		sectionText = append(sectionText, fmt.Sprintf("[%s]\n%s", section.Source, section.Content))
	}
	system := fmt.Sprintf("You are an autonomous AI engineering agent executing a structured WorkPackage.\nPersistent specialization: %s\nTask role: %s\nPackage: %s\nPriority: %s\n\n%s\n\nProduce reproducible verification evidence.", req.Agent.Role, req.Task.Role, req.Task.Title, req.Task.Priority, strings.Join(sectionText, "\n\n"))
	acceptance := make([]string, 0, len(req.Task.AcceptanceCriteria))
	for _, item := range req.Task.AcceptanceCriteria {
		acceptance = append(acceptance, "- [ ] "+item)
	}
	task := fmt.Sprintf("## Objective\n%s\n\n## Acceptance Criteria\n%s\n\nExecute the required changes step-by-step. Validate with automated tests before completing.", req.Task.Goal, strings.Join(acceptance, "\n"))
	return &CompiledExecutionContext{
		SystemInstructions: system,
		TaskInstructions:   task,
		Context:            facts,
		Constraints:        append([]string(nil), req.Task.Constraints...),
		Skills:             append([]string(nil), skillIDs...),
		AcceptanceCriteria: append([]string(nil), req.Task.AcceptanceCriteria...),
		Sections:           sections,
	}, nil
}

func formatFacts(facts map[string]string) []string {
	keys := make([]string, 0, len(facts))
	for key := range facts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("- **%s**: %s", key, facts[key]))
	}
	return out
}

func formatAgentSpec(spec AgentSpec) string {
	return fmt.Sprintf("Role: %s\nInstructions: %s\nResponsibilities: %s\nCapabilities: %s\nConstraints: %s\nDomains: %s\nStrengths: %s\nTags: %s\nVerification policy: require_evidence=%t; require_tests=%t", spec.Role, strings.Join(spec.Instructions, "; "), strings.Join(spec.Responsibilities, "; "), strings.Join(spec.Capabilities, "; "), strings.Join(spec.Constraints, "; "), strings.Join(spec.Domains, "; "), strings.Join(spec.Strengths, "; "), strings.Join(spec.Tags, "; "), spec.VerificationPolicy.RequireEvidence, spec.VerificationPolicy.RequireTests)
}
