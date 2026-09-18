package intelligence

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIProviderUnavailableReturnsErrorInsteadOfFallback(t *testing.T) {
	p := NewOpenAIProvider("https://api.example.invalid/v1", "", "test-model")
	_, err := p.AnalyzeIntent(context.Background(), "build product", nil)
	if !errors.Is(err, ErrIntelligenceUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestOpenAIProviderHTTPFailureIsNotConvertedToFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	_, err := p.AnalyzeIntent(context.Background(), "build product", nil)
	if err == nil {
		t.Fatal("expected provider HTTP error, got nil")
	}
}

func TestOpenAIPlanGenerationCannotMintMaestroSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"packages\":[{\"title\":\"Backend\",\"goal\":\"Build API\",\"priority\":\"HIGH\",\"dependencies\":[],\"role\":\"implementer\",\"skills\":[\"skill-fabricated\"],\"acceptance\":[\"tests pass\"]}]}"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	pkgs, err := p.GeneratePlanOutline(context.Background(), &IntentAnalysis{Intent: "Build API"}, nil, nil)
	if err != nil {
		t.Fatalf("generate plan: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected one package, got %d", len(pkgs))
	}
	if len(pkgs[0].Skills) != 0 {
		t.Fatalf("intelligence must not mint Maestro skills, got %#v", pkgs[0].Skills)
	}
}

func TestOpenAIAnalyzeIncludesBoundedProjectContext(t *testing.T) {
	var requestBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		requestBody = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"intent\":\"ship auth\",\"scope\":\"project\",\"risk_level\":\"high\",\"identified_goals\":[\"auth\"],\"constraints\":[],\"assumptions\":[]}"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	_, err := p.AnalyzeIntent(context.Background(), "ship auth", map[string]any{
		"project_context": map[string]any{
			"branch":   "main",
			"excerpts": []map[string]string{{"path": "AGENTS.md", "content": "Architecture constraints"}},
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeIntent: %v", err)
	}
	if !strings.Contains(requestBody, "AGENTS.md") || !strings.Contains(requestBody, "Architecture constraints") {
		t.Fatalf("OpenAI-compatible analysis did not receive project context: %s", requestBody)
	}
}

func TestOpenAIExtractBriefParsesStructuredFields(t *testing.T) {
	var requestBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		requestBody = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"in_scope\":[\"backend\",\"frontend\"],\"acceptance_criteria\":[\"tests pass\"],\"stack\":[\"Go\"],\"verification\":[\"go test\"]}"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	ext, err := p.ExtractBrief(context.Background(), "Ship auth with backend and frontend", map[string]any{"project_context": map[string]any{"head": "abc123"}})
	if err != nil {
		t.Fatalf("ExtractBrief: %v", err)
	}
	if len(ext.InScope) != 2 || ext.InScope[0] != "backend" {
		t.Fatalf("unexpected in_scope: %#v", ext.InScope)
	}
	if len(ext.AcceptanceCriteria) != 1 || len(ext.Stack) != 1 || len(ext.Verification) != 1 {
		t.Fatalf("unexpected extraction: %#v", ext)
	}
	if !strings.Contains(requestBody, "Ship auth") {
		t.Fatalf("extract request missing prompt text: %s", requestBody)
	}
}

func TestOpenAIExtractBriefUnavailableReturnsError(t *testing.T) {
	p := NewOpenAIProvider("https://api.example.invalid/v1", "", "test-model")
	if _, err := p.ExtractBrief(context.Background(), "build product", nil); !errors.Is(err, ErrIntelligenceUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestOpenAIPlanIncludesBoundedProjectContext(t *testing.T) {
	var requestBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		requestBody = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"packages\":[{\"title\":\"Build\",\"goal\":\"ship\",\"priority\":\"NORMAL\",\"dependencies\":[],\"role\":\"implementer\",\"skills\":[],\"acceptance\":[\"done\"]}]}"}}]}`))
	}))
	defer srv.Close()
	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	_, err := p.GeneratePlanOutline(context.Background(), &IntentAnalysis{Intent: "ship"}, nil, map[string]any{"project_context": map[string]any{"head": "abc123"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(requestBody, "abc123") || !strings.Contains(requestBody, "project_context") {
		t.Fatalf("OpenAI plan request missing bounded project context: %s", requestBody)
	}
}
