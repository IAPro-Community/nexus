package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
)

func TestNexAssistantOverviewProjectsExistingHUDFacts(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	view := buildNexAssistantOverview(nexus.HUDState{
		GeneratedAt: now,
		Agents: []nexus.HUDAgent{
			{Status: "WORKING"},
			{Status: "IDLE"},
		},
		Missions: []nexus.HUDMission{
			{State: "BLOCKED_NEEDS_USER", NeedsUser: true},
			{State: "COMPLETED_VERIFIED"},
			{State: "CANCELED_BY_USER"},
			{State: "FAILED_NO_PROGRESS"},
			{State: "FAILED_BUDGET_EXCEEDED"},
			{State: "FAILED_VERIFICATION"},
		},
		Alerts: []nexus.HUDAlert{
			{Severity: "WARNING", SnoozedUntil: &future},
			{Severity: "WARNING", SnoozedUntil: &expired},
			{Severity: "CRITICAL", Acked: true},
		},
	}, now)
	if view.AgentCount != 1 || view.RunCount != 1 || view.NeedsUser != 1 || view.Alerts != 1 {
		t.Fatalf("unexpected projection: %+v", view)
	}
	if len(view.Suggestions) != 3 {
		t.Fatalf("expected three factual navigation options, got %+v", view.Suggestions)
	}
}

func TestNexAssistantDoesNotInventSuggestions(t *testing.T) {
	view := buildNexAssistantOverview(nexus.HUDState{}, time.Now())
	if len(view.Suggestions) != 0 || view.AgentCount != 0 || view.NeedsUser != 0 {
		t.Fatalf("empty snapshot must not create recommendations: %+v", view)
	}
	if _, ok := nexDeterministicReply("analise meu código", view); ok {
		t.Fatal("do not pretend code analysis without model/tool access")
	}
	answer, ok := nexDeterministicReply("status", view)
	if !ok || !strings.Contains(answer, "0 agentes ativos") {
		t.Fatalf("status must reflect actual snapshot: %q", answer)
	}
}

func TestNexProjectFactsAreBoundedAndAllowlisted(t *testing.T) {
	facts := []contextsnapshot.ProjectFact{
		{Category: "untrusted", Key: "ignore", Value: "do not include"},
		{Category: "commands", Key: "private.script", Value: "SECRET=not-a-key"},
	}
	for i := 0; i < 30; i++ {
		facts = append(facts, contextsnapshot.ProjectFact{
			Category: "stack", Key: fmt.Sprintf("stack.%d", i),
			Value:      strings.Repeat("a", 400),
			Basis:      contextsnapshot.FactObserved,
			Confidence: contextsnapshot.ConfidenceHigh,
		})
	}
	view := &nexus.ProjectIntelligenceView{
		CurrentSnapshot: &contextsnapshot.ProjectContextSnapshot{Facts: facts},
	}
	extracted := boundedNexProjectFacts(view)
	if len(extracted) != 12 {
		t.Fatalf("expected a bounded sample of 12 project facts, got %d", len(extracted))
	}
	for _, fact := range extracted {
		if fact.Category != "stack" || len(fact.Value) > 170 {
			t.Fatalf("project facts leaked an unexpected category or length: %+v", fact)
		}
	}
	if got := boundedNexProjectFacts(nil); len(got) != 0 {
		t.Fatal("missing snapshot must not fabricate context")
	}
}

func TestNexDeterministicQuickPromptsUseHUDState(t *testing.T) {
	rem := 72.0
	low := 10.0
	state := nexus.HUDState{
		GeneratedAt: time.Now().UTC(),
		Providers: []nexus.HUDProvider{
			{
				Provider:       "codex",
				Profile:        "work",
				Health:         "healthy",
				Availability:   "AVAILABLE",
				CapacityState:  "HEALTHY",
				LimitingWindow: &nexus.HUDWindow{Kind: "5h", RemainingPercent: &rem, Status: "LIVE"},
			},
			{
				Provider:       "claude",
				Profile:        "personal",
				Health:         "rate_limited",
				Availability:   "RATE_LIMITED",
				CapacityState:  "EXHAUSTED",
				RateLimited:    true,
				LimitingWindow: &nexus.HUDWindow{Kind: "5h", RemainingPercent: &low, Status: "RATE_LIMITED"},
			},
		},
		Agents: []nexus.HUDAgent{
			{ID: "ag-1", Name: "Worker Alpha", Provider: "codex", Status: "WORKING"},
			{ID: "ag-2", Name: "Idle Beta", Provider: "claude", Status: "IDLE"},
		},
		Missions: []nexus.HUDMission{
			{RunID: "run-1", State: "BLOCKED_NEEDS_USER", NeedsUser: true},
		},
		RuntimeAttention: []nexus.HUDRuntimeAttention{
			{RuntimeID: "rt-1"},
		},
	}
	view := buildNexAssistantOverview(state, state.GeneratedAt)

	quotaAns, ok := nexDeterministicReplyWithState("Como estão minhas cotas?", state, view)
	if !ok || !strings.Contains(quotaAns, "codex:work (72%)") || !strings.Contains(quotaAns, "Cotas de 2 conta(s)") {
		t.Fatalf("unexpected quota reply: ok=%v ans=%q", ok, quotaAns)
	}

	attAns, ok := nexDeterministicReplyWithState("O que precisa da minha atenção?", state, view)
	if !ok || !strings.Contains(attAns, "1 decisão(ões)") || !strings.Contains(attAns, "1 evento(s) de runtime") {
		t.Fatalf("unexpected attention reply: ok=%v ans=%q", ok, attAns)
	}

	agAns, ok := nexDeterministicReplyWithState("Resumo dos agentes ativos", state, view)
	if !ok || !strings.Contains(agAns, "1 ativo(s) de 2 registrado(s)") || !strings.Contains(agAns, "Worker Alpha") {
		t.Fatalf("unexpected agents reply: ok=%v ans=%q", ok, agAns)
	}

	if _, ok := nexDeterministicReplyWithState("analise meu código", state, view); ok {
		t.Fatal("open code analysis must still require a live read-only provider")
	}
}

func TestNexAssistantSetupAssistanceAndAutoConfigureFlow(t *testing.T) {
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("NEXUS_DISABLE_HUD", "")

	st, err := nexus.OpenStore()
	if err != nil {
		t.Fatalf("open isolated nexus store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	n := nexus.NewWithStoreAndExecutor(st, nil)
	handler := &NexusHandler{
		nexus: n,
		hud:   nexus.NewHUDApplicationService(n),
	}

	// Open question without configured provider returns 503 + structured setup_assistance.
	reqBody, _ := json.Marshal(map[string]any{"message": "Explique o motivo da fila atual"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nex/assistant", bytes.NewReader(reqBody))
	handler.handleNexAssistant(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when unconfigured, got %d body=%s", rec.Code, rec.Body.String())
	}
	var unavail struct {
		Error           string                        `json:"error"`
		SetupAssistance *nexus.NexChatSetupAssistance `json:"setup_assistance"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &unavail); err != nil {
		t.Fatalf("decode 503 response: %v", err)
	}
	if unavail.Error != "nex_model_unavailable" || unavail.SetupAssistance == nil || unavail.SetupAssistance.Reason == "" {
		t.Fatalf("expected structured setup_assistance, got %+v", unavail)
	}

	// Wire fake OpenAI server via /api/v1/nex/intelligence/auto-configure.
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Resposta via auto-configure."}}]}`))
	}))
	defer aiSrv.Close()

	autoBody, _ := json.Marshal(nexus.AutoConfigureIntelligenceRequest{
		Mode:    coreconfig.IntelligenceOpenAICompatible,
		BaseURL: aiSrv.URL,
		Model:   "local-qwen",
	})
	autoRec := httptest.NewRecorder()
	autoReq := httptest.NewRequest(http.MethodPost, "/api/v1/nex/intelligence/auto-configure", bytes.NewReader(autoBody))
	handler.handleNexIntelligenceAutoConfigure(autoRec, autoReq)
	if autoRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from auto-configure, got %d body=%s", autoRec.Code, autoRec.Body.String())
	}

	// Repeating the same open question now succeeds cleanly.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/nex/assistant", bytes.NewReader(reqBody))
	handler.handleNexAssistant(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 after auto-configure, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var okResp struct {
		Answer string `json:"answer"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &okResp); err != nil {
		t.Fatalf("decode 200 response: %v", err)
	}
	if okResp.Answer != "Resposta via auto-configure." || okResp.Source != "configured_model" {
		t.Fatalf("unexpected chat response after auto-configure: %+v", okResp)
	}
}
