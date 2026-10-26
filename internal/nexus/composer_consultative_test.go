package nexus

import (
    "strings"
    "testing"

    "github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

// The Composer must not turn every empty checklist field into a user interview.
func TestComposerConsultativeDraftDoesNotManufactureSevenQuestions(t *testing.T) {
    brief := newComposerBrief("Quero organizar melhor o frontend do meu projeto Diário", "")
    for i := 0; i < 3; i++ {
        refreshComposerBrief(&brief)
    }
    for _, unknown := range brief.Unknowns {
        if unknown.Status == PromptUnknownOpen && (unknown.Source == "ARCHETYPE" || unknown.Source == "REFINERY") {
            t.Fatalf("ritual question should not be asked: %+v", unknown)
        }
    }
    if next, ok := selectComposerNextUnknownForBrief(brief); ok {
        t.Fatalf("a useful plan should start without a mandatory checklist: %+v", next)
    }
}

// Model suggestions are proposals; they must not silently turn into scope,
// approved decisions or invented repo findings.
func TestComposerConsultativeRecommendationsKeepProvenanceAndAreIdempotent(t *testing.T) {
    brief := newComposerBrief("Preciso de um aplicativo para os pais da escola", "")
    extraction := &intelligence.BriefExtraction{RecommendedApproaches: []string{
        "Começar com comunicados e confirmação de leitura; validar a necessidade de mensagens privadas.",
        "Experimentar um protótipo de agenda escolar antes de ampliar o escopo.",
    }}
    applyBriefExtraction(&brief, extraction)
    applyBriefExtraction(&brief, extraction)
    refreshComposerBrief(&brief)

    if len(brief.Recommendations) != 2 {
        t.Fatalf("expected two distinct proposed approaches, got %+v", brief.Recommendations)
    }
    for _, recommendation := range brief.Recommendations {
        if recommendation.SourceType != "INTELLIGENCE" || recommendation.Status != ComposerFactProposed {
            t.Fatalf("recommendation must not be an approved fact: %+v", recommendation)
        }
    }
    if len(brief.Scope.InScope) != 0 {
        t.Fatalf("a recommendation silently entered approved scope: %+v", brief.Scope.InScope)
    }
    if !strings.Contains(brief.Preview, "Suggested approaches") {
        t.Fatalf("prompt artifact lost proposals: %s", brief.Preview)
    }
    reply := composeAssistantReply(brief)
    if !strings.Contains(reply, "Caminhos sugeridos") {
        t.Fatalf("initial assistant reply did not surface useful options: %s", reply)
    }
}

func TestComposerConsultativeDoesNotReaskGroundedProviderField(t *testing.T) {
    brief := newComposerBrief("Melhorar a organização do frontend", "")
    brief.Scope.InScope = []string{"Eliminar duplicação no módulo de apresentação"}
    MergeIntelligenceUnknowns(&brief, []intelligence.AmbiguityItem{{
        Key: "scope.in_scope",
        Level: intelligence.AmbiguityBlocking,
        Question: "Qual módulo do frontend está no escopo?",
    }})
    refreshComposerBrief(&brief)
    for _, unknown := range brief.Unknowns {
        if unknown.Field == "scope.in_scope" && unknown.Status == PromptUnknownOpen {
            t.Fatalf("must not ask question already answered by evidence: %+v", unknown)
        }
    }
}
