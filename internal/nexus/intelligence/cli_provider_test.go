package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCLIProviderUnavailableWhenBackendIsNotCapabilityValidated(t *testing.T) {
	p := NewCLIProvider("claude", "work", false, func(context.Context, string) (string, error) {
		return "", nil
	})
	if p.Available(context.Background()) {
		t.Fatal("CLI intelligence must be unavailable until the Nexus adapter validates headless + submit_prompt")
	}
	_, err := p.AnalyzeIntent(context.Background(), "ship auth", nil)
	if !errors.Is(err, ErrIntelligenceUnavailable) {
		t.Fatalf("expected ErrIntelligenceUnavailable, got %v", err)
	}
}

func TestCLIProviderAnalyzeParsesJSONEnvelope(t *testing.T) {
	var gotPrompt string
	p := NewCLIProvider("claude", "work", true, func(_ context.Context, prompt string) (string, error) {
		gotPrompt = prompt
		return "prefix\n```json\n{\"intent\":\"ship auth\",\"scope\":\"project\",\"risk_level\":\"high\",\"identified_goals\":[\"auth\"],\"constraints\":[],\"assumptions\":[]}\n```\nsuffix", nil
	})
	got, err := p.AnalyzeIntent(context.Background(), "ship auth", nil)
	if err != nil {
		t.Fatalf("AnalyzeIntent: %v", err)
	}
	if got.Intent != "ship auth" || got.RiskLevel != "high" {
		t.Fatalf("unexpected analysis: %+v", got)
	}
	if gotPrompt == "" {
		t.Fatal("expected a compiled CLI prompt")
	}
}

func TestExtractJSONObjectIgnoresTrailingProseWithBraces(t *testing.T) {
	payload, err := extractJSONObject(`{"intent":"quero ideias","scope":"project","risk_level":"low","identified_goals":["ideas"],"constraints":[],"assumptions":[]}
OK. Prefer {option A} over {option B}.`)
	if err != nil {
		t.Fatal(err)
	}
	var result IntentAnalysis
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Intent != "quero ideias" {
		t.Fatalf("unexpected intent %q", result.Intent)
	}
}

func TestExtractJSONObjectRejectsNonObject(t *testing.T) {
	if _, err := extractJSONObject(`OK no json here`); err == nil {
		t.Fatal("expected error")
	}
}

func TestHeadlessPromptArgsMatchSupportedProviderCLIs(t *testing.T) {
	cases := []struct {
		provider string
		want0    string
	}{
		{"claude", "-p"}, {"agy", "-p"}, {"gemini", "-p"}, {"cursor", "-p"}, {"opencode", "run"}, {"codex", "exec"},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			args, err := HeadlessPromptArgs(tc.provider, "hello")
			if err != nil {
				t.Fatal(err)
			}
			if len(args) != 2 || args[0] != tc.want0 || args[1] != "hello" {
				t.Fatalf("unexpected args %#v", args)
			}
		})
	}
}

func TestCLIProviderPlanIncludesProjectContext(t *testing.T) {
	var prompt string
	p := NewCLIProvider("claude", "work", true, func(_ context.Context, input string) (string, error) {
		prompt = input
		return `{"packages":[{"title":"Build","goal":"ship","priority":"NORMAL","dependencies":[],"role":"implementer","skills":[],"acceptance":["done"]}]}`, nil
	})
	_, err := p.GeneratePlanOutline(context.Background(), &IntentAnalysis{Intent: "ship"}, nil, map[string]any{"project_context": map[string]any{"head": "abc123"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "abc123") || !strings.Contains(prompt, "project_context") {
		t.Fatalf("CLI plan prompt missing bounded project context: %s", prompt)
	}
}

func TestCLIProviderExtractBriefEmptyWhenUnavailable(t *testing.T) {
	p := NewCLIProvider("claude", "work", false, func(context.Context, string) (string, error) {
		t.Fatal("runner must not be invoked when provider is unavailable")
		return "", nil
	})
	ext, err := p.ExtractBrief(context.Background(), "ship auth", nil)
	if err != nil {
		t.Fatalf("unavailable CLI must fall back to empty extraction, got err %v", err)
	}
	if ext == nil || len(ext.InScope) != 0 {
		t.Fatalf("expected empty extraction, got %#v", ext)
	}
}

func TestCLIProviderExtractBriefParsesJSONEnvelope(t *testing.T) {
	var gotPrompt string
	p := NewCLIProvider("claude", "work", true, func(_ context.Context, prompt string) (string, error) {
		gotPrompt = prompt
		return "```json\n{\"in_scope\":[\"login\"],\"risks\":[\"token leak\"],\"testing\":[\"unit\"]}\n```", nil
	})
	ext, err := p.ExtractBrief(context.Background(), "ship auth", map[string]any{"probe": false})
	if err != nil {
		t.Fatalf("ExtractBrief: %v", err)
	}
	if len(ext.InScope) != 1 || ext.InScope[0] != "login" || len(ext.Risks) != 1 || len(ext.Testing) != 1 {
		t.Fatalf("unexpected extraction: %#v", ext)
	}
	if !strings.Contains(gotPrompt, "ship auth") {
		t.Fatalf("extract prompt missing text: %s", gotPrompt)
	}
}

func TestCLIProviderPlanFromGoalOneshot(t *testing.T) {
	var calls int
	p := NewCLIProvider("codex", "default", true, func(_ context.Context, prompt string) (string, error) {
		calls++
		if !strings.Contains(prompt, `"unknowns"`) || !strings.Contains(prompt, `"packages"`) {
			t.Fatalf("oneshot prompt missing combined schema: %s", prompt)
		}
		if strings.Contains(prompt, `"dependencies":["package title"]`) || strings.Contains(prompt, `"title":"..."`) {
			t.Fatalf("schema placeholders teach the model to emit invalid packages: %s", prompt)
		}
		return `{
			"intent":"ship auth",
			"scope":"project",
			"risk_level":"medium",
			"identified_goals":["auth"],
			"constraints":[],
			"assumptions":[],
			"unknowns":[{"key":"stack","level":"LOW_IMPACT","question":"stack?","rationale":"default ok","suggested_options":["go"],"default_choice":"go"}],
			"packages":[{"title":"Auth","goal":"login","priority":"HIGH","dependencies":[],"role":"implementer","skills":["fake"],"acceptance":["ok"]}]
		}`, nil
	})
	intent, unknowns, packages, err := p.PlanFromGoal(context.Background(), "ship auth", map[string]any{"probe": false})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 CLI call, got %d", calls)
	}
	if intent == nil || intent.Intent != "ship auth" {
		t.Fatalf("unexpected intent %+v", intent)
	}
	if len(unknowns) != 1 || unknowns[0].Key != "stack" {
		t.Fatalf("unexpected unknowns %+v", unknowns)
	}
	if len(packages) != 1 || packages[0].Title != "Auth" || packages[0].Skills != nil {
		t.Fatalf("unexpected packages %+v", packages)
	}
}

func TestBoundContextJSONCapsOversizedPayload(t *testing.T) {
	huge := map[string]any{"blob": strings.Repeat("Z", MaxCLIContextBytes+2048)}
	out := boundContextJSON(huge)
	if len(out) > MaxCLIContextBytes+32 {
		t.Fatalf("expected capped context, got %d bytes", len(out))
	}
	if !strings.Contains(out, "[truncated]") {
		t.Fatal("expected truncation marker")
	}
}

func TestIsTransientIntelligenceFailure(t *testing.T) {
	if !IsTransientIntelligenceFailure(errors.New("run cli:agy:x analyze intent: signal: killed")) {
		t.Fatal("signal: killed must be transient")
	}
	if IsTransientIntelligenceFailure(errors.New("provider misconfigured")) {
		t.Fatal("config errors are not transient")
	}
}
