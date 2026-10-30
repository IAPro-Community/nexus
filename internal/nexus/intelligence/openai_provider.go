package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAIIntelligenceProvider connects to OpenAI, DeepSeek, Ollama or any compatible /v1/chat/completions endpoint.
type OpenAIIntelligenceProvider struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewOpenAIProvider creates an intelligence provider from environment or explicit config.
func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIIntelligenceProvider {
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("DEEPSEEK_BASE_URL")
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		}
	}

	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = os.Getenv("DEEPSEEK_MODEL")
		}
		if model == "" {
			model = "gpt-4o-mini"
		}
	}

	return &OpenAIIntelligenceProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p *OpenAIIntelligenceProvider) Name() string {
	return "openai-compatible (" + p.Model + ")"
}

func (p *OpenAIIntelligenceProvider) Available(ctx context.Context) bool {
	return p.APIKey != "" || strings.Contains(p.BaseURL, "localhost") || strings.Contains(p.BaseURL, "127.0.0.1")
}

func (p *OpenAIIntelligenceProvider) AnalyzeIntent(ctx context.Context, input string, contextData map[string]any) (*IntentAnalysis, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}

	sysPrompt := `You are the Nexus product discovery and planning consultant for novices and experts. Help users who do not know the product or technology yet.
Extract meaningful goals, distinguish assumptions from verified facts, and avoid asking the user to supply engineering terminology. Never invent repository findings.
Analyze the user's objective and output ONLY valid JSON matching this schema:
{
  "intent": "summary of intent",
  "scope": "project" | "mission" | "package" | "task",
  "risk_level": "low" | "medium" | "high",
  "identified_goals": ["goal 1", "goal 2"],
  "constraints": ["constraint 1"],
  "assumptions": ["assumption 1"]
}`

	userContent, _ := json.Marshal(map[string]any{
		"objective":       input,
		"project_context": contextData,
	})

	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": string(userContent)},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("analyze intent with %s: %w", p.Name(), err)
	}

	var parsed IntentAnalysis
	if err := json.Unmarshal(resBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode intent analysis from %s: %w", p.Name(), err)
	}
	parsed.CreatedAt = time.Now().UTC()
	return &parsed, nil
}

func (p *OpenAIIntelligenceProvider) EvaluateAmbiguities(ctx context.Context, intent *IntentAnalysis) ([]AmbiguityItem, error) {
	return p.EvaluateAmbiguitiesWithContext(ctx, intent, nil)
}

func (p *OpenAIIntelligenceProvider) EvaluateAmbiguitiesWithContext(ctx context.Context, intent *IntentAnalysis, contextData map[string]any) ([]AmbiguityItem, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}

	sysPrompt := `Identify only concrete choices that materially change the solution and cannot be resolved from the user's goal and project context.
Return an empty unknowns array when safe proposed assumptions allow planning to proceed.
Never generate generic checklist questions about scope exclusions, tests, Definition of Done, deliverables, technology stack or risks.
Explain a real choice in approachable language with useful options and a recommendation.
Classify BLOCKING only for irreversible, high-impact or safety-critical choices, IMPORTANT for material design forks, or LOW_IMPACT for a safe default.
Output ONLY valid JSON matching:
{
  "unknowns": [
    {
      "key": "unique_key",
      "level": "BLOCKING" | "IMPORTANT" | "LOW_IMPACT",
      "question": "clear concise question",
      "rationale": "why this matters",
      "suggested_options": ["Option A", "Option B"],
      "default_choice": "Option A"
    }
  ]
}`

	intentBytes, _ := json.Marshal(map[string]any{"intent": intent, "project_context": contextData})
	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": string(intentBytes)},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("evaluate ambiguities with %s: %w", p.Name(), err)
	}

	var wrapper struct {
		Unknowns []AmbiguityItem `json:"unknowns"`
	}
	if err := json.Unmarshal(resBytes, &wrapper); err != nil {
		return nil, fmt.Errorf("decode ambiguity analysis from %s: %w", p.Name(), err)
	}
	return wrapper.Unknowns, nil
}

func (p *OpenAIIntelligenceProvider) GeneratePlanOutline(ctx context.Context, intent *IntentAnalysis, facts map[string]string, contextData map[string]any) ([]WorkPackageOutline, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}

	sysPrompt := `Decompose the intent into a series of structured, reviewable WorkPackages.
Each package must have clear, observable acceptance criteria. Do NOT invent Maestro skill identifiers; skills must be an empty array because Maestro is a separate authority.
For substantial prompts, decompose concrete backend/frontend/data/integration/verification deliverables with explicit dependencies and parallelizable workstreams; do not produce one generic activity or merely copy the same objective to fixed phases.
A genuinely atomic task may have one package. Never fabricate repository evidence or report unexecuted tests as passing.
dependencies must reference exact unique titles of other packages in this output; never invent or silently omit an unresolved dependency.
Output ONLY valid JSON matching:
{
  "packages": [
    {
      "title": "Package Title",
      "goal": "Specific measurable objective",
      "priority": "CRITICAL" | "HIGH" | "NORMAL" | "LOW",
      "dependencies": [],
      "role": "implementer" | "reviewer" | "tester" | "architect",
      "skills": [],
      "acceptance": ["criterion 1", "test passes"]
    }
  ]
}`

	userContent := map[string]any{
		"intent":          intent,
		"confirmed_facts": facts,
		"project_context": contextData,
	}
	userBytes, _ := json.Marshal(userContent)

	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": string(userBytes)},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("generate plan with %s: %w", p.Name(), err)
	}

	var wrapper struct {
		Packages []WorkPackageOutline `json:"packages"`
	}
	if err := json.Unmarshal(resBytes, &wrapper); err != nil {
		return nil, fmt.Errorf("decode plan outline from %s: %w", p.Name(), err)
	}
	for i := range wrapper.Packages {
		wrapper.Packages[i].Skills = nil
	}
	return wrapper.Packages, nil
}

func (p *OpenAIIntelligenceProvider) ExtractBrief(ctx context.Context, text string, contextData map[string]any) (*BriefExtraction, error) {
	if !p.Available(ctx) {
		return nil, ErrIntelligenceUnavailable
	}

	sysPrompt := `You are a product discovery and solution planning consultant, including for users with no technical background.
Extract grounded requirements only; leave unsupported factual arrays empty and never invent repository state or stack.
Separately propose 1–3 helpful approaches to meet the user's goal, each with an understandable benefit or trade-off. These are explicitly unverified recommendations, not approved scope.
Identify at most one material decision that cannot safely be inferred. An empty clarifications array is valid and preferred over a ritual interview.
Output ONLY valid JSON matching this schema:
{
  "discovery_complete": true,
  "clarifications": [{"key":"...","level":"BLOCKING|IMPORTANT|LOW_IMPACT","question":"...","rationale":"...","suggested_options":["..."],"default_choice":"..."}],
  "recommended_approaches": ["suggested solution and rationale (proposal, not fact)"],
  "in_scope": ["what must be built/changed"],
  "out_of_scope": ["explicit non-goals"],
  "constraints_technical": ["technical constraints"],
  "constraints_security": ["security/auth constraints"],
  "risks": ["risks or regressions to avoid"],
  "acceptance_criteria": ["observable done conditions"],
  "testing": ["tests expected"],
  "verification": ["how to verify (commands, checks)"],
  "prohibited_actions": ["actions the agent must NOT take"],
  "deliverables": ["expected artifacts"],
  "stack": ["languages/frameworks/services involved"],
  "existing_state": ["relevant current-state facts"]
}`

	userContent, _ := json.Marshal(map[string]any{
		"prompt":          text,
		"project_context": contextData,
	})

	body := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": sysPrompt},
			{"role": "user", "content": string(userContent)},
		},
		"temperature":     0.1,
		"response_format": map[string]string{"type": "json_object"},
	}

	resBytes, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("extract brief with %s: %w", p.Name(), err)
	}

	var parsed BriefExtraction
	if err := json.Unmarshal(resBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode brief extraction from %s: %w", p.Name(), err)
	}
	return &parsed, nil
}

func (p *OpenAIIntelligenceProvider) doRequest(ctx context.Context, body map[string]any) ([]byte, error) {
	reqData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := p.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBytes, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("malformed chat response")
	}

	return []byte(chatResp.Choices[0].Message.Content), nil
}
