package nexus

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompactComposerIntelligenceBriefOmitsHeavyFields(t *testing.T) {
	brief := LivingBrief{
		Goal:         "Documentar API",
		SourcePrompt: strings.Repeat("x", 800),
		Preview:      strings.Repeat("PREVIEW-", 500),
		Facts:        []ComposerFact{{ID: "f1", Field: "stack", Value: "go"}},
		Context: BriefContext{
			ReferencedFiles: []ContextFileRef{{Path: "web/src/App.tsx", Excerpt: strings.Repeat("code", 200)}},
		},
		Unknowns: []PromptUnknown{{
			ID: "u1", Field: "quality.acceptance_criteria", Question: "Pronto quando?", Status: PromptUnknownOpen, Severity: PromptUnknownRecommended,
		}},
		Readiness: PromptReadiness{State: "READY_WITH_ASSUMPTIONS"},
	}
	compact := compactComposerIntelligenceBrief(brief)
	raw, err := json.Marshal(compact)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "PREVIEW-") {
		t.Fatal("compact brief must omit preview text")
	}
	if strings.Contains(string(raw), `"excerpt"`) || strings.Contains(string(raw), "codecode") {
		t.Fatal("compact brief must omit file excerpts")
	}
	if len(raw) > 4_000 {
		t.Fatalf("compact brief still too large: %d bytes", len(raw))
	}
	if got, _ := compact["referenced_files"].([]string); len(got) != 1 || got[0] != "web/src/App.tsx" {
		t.Fatalf("expected path-only referenced files, got %#v", compact["referenced_files"])
	}
}

func TestLowSignalComposerTurnSkipsEnrichmentWhenBriefIsDeep(t *testing.T) {
	deep := LivingBrief{
		Goal: "Documentar API REST com exemplos e testes",
		Intent: BriefIntent{
			Objective:      "Documentar API REST com exemplos e testes",
			DesiredOutcome: "UI + testes verdes no caminho alterado",
		},
		Scope:   BriefScope{InScope: []string{"OpenAPI", "exemplos curl"}},
		Quality: BriefQuality{AcceptanceCriteria: []string{"UI + testes verdes no caminho alterado"}},
	}
	if PromptIsShallow(deep) {
		t.Fatal("fixture must be deep enough to skip low-signal enrichment")
	}
	if shouldEnrichComposerTurn(deep, "continue?") {
		t.Fatal("continue? on a deep brief must not launch CLI intelligence")
	}
	if !shouldEnrichComposerTurn(deep, "Adicionar autenticação OAuth2 na documentação") {
		t.Fatal("substantive turns must still enrich")
	}
	shallow := LivingBrief{Goal: "melhorar ui"}
	if !shouldEnrichComposerTurn(shallow, "ok") {
		t.Fatal("shallow briefs still need enrichment even for short replies")
	}
}

func TestTurnIntelligenceNoticeHidesKilledCLI(t *testing.T) {
	if got := turnIntelligenceNotice(nil); got != "" {
		t.Fatalf("nil err => empty, got %q", got)
	}
	if got := turnIntelligenceNotice(errString("run cli:agy:x analyze intent: signal: killed")); got != "" {
		t.Fatalf("killed must be silent, got %q", got)
	}
	if got := turnIntelligenceNotice(errString("provider misconfigured")); got == "" {
		t.Fatal("non-transient failures need a short notice")
	}
	if strings.Contains(turnIntelligenceNotice(errString("provider misconfigured")), "misconfigured") {
		t.Fatal("notice must not leak raw provider errors")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
