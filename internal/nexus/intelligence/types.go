package intelligence

import (
	"context"
	"time"
)

// AmbiguityLevel classifies unknown requirements based on autonomy boundaries (Phase C).
type AmbiguityLevel string

const (
	AmbiguityBlocking  AmbiguityLevel = "BLOCKING"   // Must ask user before proceeding
	AmbiguityImportant AmbiguityLevel = "IMPORTANT"  // Material scope/risk change
	AmbiguityLowImpact AmbiguityLevel = "LOW_IMPACT" // Sensible default can be chosen autonomously
)

// IntentAnalysis represents structured decomposition of a user or system objective.
type IntentAnalysis struct {
	Intent          string    `json:"intent"`
	Scope           string    `json:"scope"`      // "project" | "mission" | "package" | "task"
	RiskLevel       string    `json:"risk_level"` // "low" | "medium" | "high"
	IdentifiedGoals []string  `json:"identified_goals"`
	Constraints     []string  `json:"constraints"`
	Assumptions     []string  `json:"assumptions"`
	SuggestedStack  []string  `json:"suggested_stack,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AmbiguityItem is an identified unknown that needs classification or user input.
type AmbiguityItem struct {
	Key              string         `json:"key"`
	Level            AmbiguityLevel `json:"level"`
	Question         string         `json:"question"`
	Rationale        string         `json:"rationale"`
	SuggestedOptions []string       `json:"suggested_options,omitempty"`
	DefaultChoice    string         `json:"default_choice,omitempty"`
	Answer           string         `json:"answer,omitempty"`
	IsResolved       bool           `json:"is_resolved"`
}

// ClarificationState tracks unknowns converted into durable facts and constraints.
type ClarificationState struct {
	Unknowns        []AmbiguityItem   `json:"unknowns"`
	StructuredFacts map[string]string `json:"structured_facts"`
	AllBlockingDone bool              `json:"all_blocking_resolved"`
}

// PromptCompilationResult is the compiled, scoped prompt ready for agent execution.
type PromptCompilationResult struct {
	MissionID       string    `json:"mission_id"`
	WorkPackageID   string    `json:"work_package_id"`
	PackageTitle    string    `json:"package_title"`
	SystemPrompt    string    `json:"system_prompt"`
	UserPrompt      string    `json:"user_prompt"`
	Skills          []string  `json:"skills,omitempty"`
	AcceptanceGates []string  `json:"acceptance_gates"`
	SharedArtifacts []string  `json:"shared_artifacts"`
	Constraints     []string  `json:"constraints"`
	EstimatedTokens int       `json:"estimated_tokens"`
	CompiledAt      time.Time `json:"compiled_at"`
}

// VerificationPolicy describes evidence requirements for an AgentSpec without
// coupling the Agent identity to a provider or to a prompt string.
type VerificationPolicy struct {
	RequireEvidence bool `json:"require_evidence,omitempty"`
	RequireTests    bool `json:"require_tests,omitempty"`
}

// AgentSpec is the durable behavioral specialization of an Agent. Provider
// and runtime settings intentionally remain outside this type.
type AgentSpec struct {
	Role               string             `json:"role,omitempty"`
	Instructions       []string           `json:"instructions,omitempty"`
	Responsibilities   []string           `json:"responsibilities,omitempty"`
	Capabilities       []string           `json:"capabilities,omitempty"`
	Constraints        []string           `json:"constraints,omitempty"`
	Domains            []string           `json:"domains,omitempty"`
	Strengths          []string           `json:"strengths,omitempty"`
	Tags               []string           `json:"tags,omitempty"`
	VerificationPolicy VerificationPolicy `json:"verification_policy,omitempty"`
}

func (s AgentSpec) HasCustomBehavior() bool {
	return len(s.Instructions) > 0 || len(s.Responsibilities) > 0 || len(s.Capabilities) > 0 || len(s.Constraints) > 0 || len(s.Domains) > 0 || len(s.Strengths) > 0 || len(s.Tags) > 0 || s.VerificationPolicy.RequireEvidence || s.VerificationPolicy.RequireTests
}

type ProjectContext struct {
	ProjectID string            `json:"project_id,omitempty"`
	Facts     map[string]string `json:"facts,omitempty"`
}

type WorkPackageContext struct {
	Title              string   `json:"title,omitempty"`
	Goal               string   `json:"goal,omitempty"`
	Priority           string   `json:"priority,omitempty"`
	Role               string   `json:"role,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Constraints        []string `json:"constraints,omitempty"`
}

// ExecutionGuidance is source-agnostic operational guidance attached to an
// execution context. Sources such as Nexus or Maestro may provide it, but
// consumers only depend on the generic contract.
type ExecutionGuidance struct {
	Enabled      bool     `json:"enabled,omitempty"`
	Instructions []string `json:"instructions,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Source       string   `json:"source,omitempty"`
	Reference    string   `json:"reference,omitempty"`
}

type RuntimeConstraints struct {
	Provider     string   `json:"provider,omitempty"`
	Model        string   `json:"model,omitempty"`
	Workspace    string   `json:"workspace,omitempty"`
	Isolation    string   `json:"isolation,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type ExecutionContextRequest struct {
	Agent    AgentSpec          `json:"agent"`
	Project  ProjectContext     `json:"project"`
	Task     WorkPackageContext `json:"task"`
	Skills   []string           `json:"skills,omitempty"`
	Guidance ExecutionGuidance  `json:"guidance,omitempty"`
	Runtime  RuntimeConstraints `json:"runtime"`
}

type ContextSection struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

// CompiledExecutionContext keeps sections and provenance inspectable while
// retaining rendered strings for provider adapters that accept text prompts.
type CompiledExecutionContext struct {
	SystemInstructions string           `json:"system_instructions"`
	TaskInstructions   string           `json:"task_instructions"`
	Context            []string         `json:"context,omitempty"`
	Constraints        []string         `json:"constraints,omitempty"`
	Skills             []string         `json:"skills,omitempty"`
	AcceptanceCriteria []string         `json:"acceptance_criteria,omitempty"`
	Sections           []ContextSection `json:"sections"`
}

// DecisionExplanation provides transparent reasoning for intelligence routing and plan optimizations.
type DecisionExplanation struct {
	Action     string    `json:"action"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	Confidence float64   `json:"confidence"`
	Reasoning  string    `json:"reasoning"`
	Timestamp  time.Time `json:"timestamp"`
}

// BriefExtraction is the structured decomposition of a free-form prompt into
// the durable fields of a Composer LivingBrief. Every field is optional; the
// provider returns only what the text actually supports so heuristics stay the
// primary source and intelligence merely enriches gaps.
type BriefExtraction struct {
	InScope              []string `json:"in_scope,omitempty"`
	OutOfScope           []string `json:"out_of_scope,omitempty"`
	ConstraintsTechnical []string `json:"constraints_technical,omitempty"`
	ConstraintsSecurity  []string `json:"constraints_security,omitempty"`
	Risks                []string `json:"risks,omitempty"`
	AcceptanceCriteria   []string `json:"acceptance_criteria,omitempty"`
	Testing              []string `json:"testing,omitempty"`
	Verification         []string `json:"verification,omitempty"`
	ProhibitedActions    []string `json:"prohibited_actions,omitempty"`
	Deliverables         []string `json:"deliverables,omitempty"`
	Stack                []string `json:"stack,omitempty"`
	ExistingState        []string `json:"existing_state,omitempty"`
}

// IntelligenceProvider defines the provider contract (OpenAI-compatible API or CLI).
type IntelligenceProvider interface {
	Name() string
	Available(ctx context.Context) bool
	AnalyzeIntent(ctx context.Context, input string, contextData map[string]any) (*IntentAnalysis, error)
	EvaluateAmbiguities(ctx context.Context, intent *IntentAnalysis) ([]AmbiguityItem, error)
	GeneratePlanOutline(ctx context.Context, intent *IntentAnalysis, facts map[string]string, contextData map[string]any) ([]WorkPackageOutline, error)
	// ExtractBrief maps a free-form prompt into structured brief fields. Providers
	// that cannot extract (e.g. non-capability-validated CLIs) return an empty
	// extraction with a nil error so callers fall back to heuristics.
	ExtractBrief(ctx context.Context, text string, contextData map[string]any) (*BriefExtraction, error)
}

// ContextualAmbiguityEvaluator is an optional extension that preserves the
// original provider contract while allowing ambiguity analysis to use the same
// bounded project envelope as intent analysis.
type ContextualAmbiguityEvaluator interface {
	EvaluateAmbiguitiesWithContext(context.Context, *IntentAnalysis, map[string]any) ([]AmbiguityItem, error)
}

// OneshotPlanner collapses intent + ambiguities + packages into a single model call.
// CLI providers implement this to avoid three sequential headless execs.
type OneshotPlanner interface {
	PlanFromGoal(ctx context.Context, goal string, contextData map[string]any) (*IntentAnalysis, []AmbiguityItem, []WorkPackageOutline, error)
}

// PlanGenerationTimeout is the hard ceiling for Composer auto_plan (including CLI).
const PlanGenerationTimeout = 90 * time.Second

// IntelligenceProbeTimeout bounds the preflight round-trip before Refinar.
const IntelligenceProbeTimeout = 20 * time.Second

// WorkPackageOutline is the raw outline produced by an intelligence provider before optimization.
type WorkPackageOutline struct {
	Title        string   `json:"title"`
	Goal         string   `json:"goal"`
	Priority     string   `json:"priority"` // "CRITICAL" | "HIGH" | "NORMAL" | "LOW"
	Dependencies []string `json:"dependencies"`
	Role         string   `json:"role"`
	Skills       []string `json:"skills"`
	Acceptance   []string `json:"acceptance"`
}

// IntelligenceEngine is the high-level orchestrator for intelligence tasks.
type IntelligenceEngine interface {
	Analyze(ctx context.Context, goal string, projectID string) (*IntentAnalysis, []AmbiguityItem, error)
	ResolveClarification(state *ClarificationState, key string, answer string)
	CompilePrompt(ctx context.Context, pkg WorkPackageOutline, facts map[string]string, skillIDs []string) (*PromptCompilationResult, error)
}
