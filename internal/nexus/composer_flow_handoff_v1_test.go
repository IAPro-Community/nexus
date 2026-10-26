package nexus

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerHandoffTracksDecisionsWithoutApprovingProposals(t *testing.T) {
	brief := LivingBrief{
		Goal: "Criar comunicação escolar",
		Unknowns: []PromptUnknown{
			{Field: "context.audience", Question: "Quem usa o app?", Status: PromptUnknownAnswered, Severity: PromptUnknownBlocking, Answer: "Pais e professores"},
			{Field: "constraints.security", Question: "Pode publicar dados reais?", Status: PromptUnknownOpen, Severity: PromptUnknownBlocking},
		},
		Recommendations: []ComposerFact{{
			Field: "recommendations", Value: "Avaliar agenda e comunicados",
			Status: ComposerFactProposed, SourceType: "INTELLIGENCE",
		}},
		Quality: BriefQuality{AcceptanceCriteria: []string{"Avisos com confirmação de leitura"}},
	}
	artifact := store.PromptArtifact{ID: "art_1", Version: 2, Hash: "hash-123"}
	session := store.ComposerSession{ID: "cmp_1"}
	handoff := composerHandoffFromArtifact(artifact, session, brief)
	if handoff.Version != 1 || handoff.ArtifactVersion != 2 || handoff.ArtifactHash != "hash-123" {
		t.Fatalf("frozen handoff missing lineage: %+v", handoff)
	}
	if handoff.ApprovedAnswers["context.audience"] != "Pais e professores" {
		t.Fatalf("approved user decision lost: %+v", handoff.ApprovedAnswers)
	}
	if len(handoff.Unresolved) != 1 || len(handoff.Recommendations) != 1 ||
		handoff.Recommendations[0].Status != ComposerFactProposed {
		t.Fatalf("proposed approach or pending decision lost provenance: %+v", handoff)
	}
	encoded, err := json.Marshal(handoff)
	if err != nil {
		t.Fatal(err)
	}
	facts := map[string]string{"composer.handoff.v1": string(encoded)}
	blockers := composerHandoffExecutionBlockers(facts)
	if len(blockers) != 1 || !strings.Contains(blockers[0], "Pode publicar") {
		t.Fatalf("unknown blocking approval must block admission: %+v", blockers)
	}
}

func TestComposerHandoffLegacyAndInvalidVersions(t *testing.T) {
	if blockers := composerHandoffExecutionBlockers(nil); len(blockers) != 0 {
		t.Fatalf("legacy plans must remain compatible: %+v", blockers)
	}
	if blockers := composerHandoffExecutionBlockers(map[string]string{
		"composer.handoff.v1": "{broken-json",
	}); len(blockers) == 0 {
		t.Fatal("invalid handoff must fail closed")
	}
	if blockers := composerHandoffExecutionBlockers(map[string]string{
		"composer.handoff.v1": `{"version":2}`,
	}); len(blockers) == 0 {
		t.Fatal("unsupported handoff version must fail closed")
	}
	if blockers := composerHandoffExecutionBlockers(map[string]string{
		"composer.handoff.v1": `{"version":1,"unresolved":[]}`,
	}); len(blockers) != 0 {
		t.Fatalf("resolved handoff must not create artificial admission blocker: %+v", blockers)
	}
}

func TestComposerSuitabilityUsesDependenciesAndDurability(t *testing.T) {
	for _, tc := range []struct {
		name string
		brief LivingBrief
		want FlowSuitability
	}{
		{"atomic", LivingBrief{Goal: "Corrigir erro de digitação na tela"}, FlowDirectFit},
		{"multi-agent", LivingBrief{Goal: "Executar com múltiplos agentes simultâneos"}, FlowBeneficial},
		{"full-stack", LivingBrief{Goal: "Implementar login", Scope: BriefScope{InScope: []string{"backend", "frontend"}}, Quality: BriefQuality{Review: []string{"review independente"}}}, FlowBeneficial},
		{"durable", LivingBrief{Goal: "Reestruturar sistema e garantir recovery", Constraints: BriefConstraints{Security: []string{"preservar credenciais"}}}, FlowStronglyRecommended},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assessment := AssessComposerFlowSuitability(tc.brief)
			if tc.want == FlowBeneficial {
				if assessment.Result == FlowDirectFit {
					t.Fatalf("expected flow recommendation: %+v", assessment)
				}
			} else if assessment.Result != tc.want {
				t.Fatalf("wanted %s, got %+v", tc.want, assessment)
			}
		})
	}
}
