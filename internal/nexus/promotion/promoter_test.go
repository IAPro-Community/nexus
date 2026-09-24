package promotion

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestPromoterDecideOFFKeepsLead(t *testing.T) {
	p := Promoter{}
	decision, err := p.Decide(context.Background(), "", "implemente backend e frontend", ModeOff)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != KeepWithLead || decision.Mode != ModeOff {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestPromoterDecideAUTOUsesExistingNexusPolicies(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "implemente backend e frontend com testes", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != Promote || !decision.Delegation.Delegate {
		t.Fatalf("expected promotion from deterministic delegation policy: %+v", decision)
	}
}

func TestPromoterDecideASKRequiresConfirmation(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "implemente backend e frontend", ModeAsk)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != Ask {
		t.Fatalf("expected ASK action, got %+v", decision)
	}
}

func TestPromoterDecideAUTOPlanWithoutDelegationStaysWithLead(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "refatore o módulo de cache e valide o resultado", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Delegation.Delegate {
		t.Fatalf("fixture must remain single-topology, got %+v", decision.Delegation)
	}
	if decision.Action != KeepWithLead {
		t.Fatalf("single-topology PLAN must stay with the interactive Lead: %+v", decision)
	}
}

func TestPromoterDecideDirectStaysWithLead(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentDirect}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "renomeie este botão", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != KeepWithLead {
		t.Fatalf("DIRECT intent must stay with the interactive Lead: %+v", decision)
	}
}

func TestPromoterDecideASKPlanWithoutDelegationStaysWithLead(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "refatore o módulo de cache e valide o resultado", ModeAsk)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Delegation.Delegate {
		t.Fatalf("fixture must remain single-topology, got %+v", decision.Delegation)
	}
	if decision.Action != KeepWithLead {
		t.Fatalf("ASK must not interrupt ordinary interactive plans: %+v", decision)
	}
}

func TestPromoterDecideAUTORequiresDecompositionPromotes(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{
			Strategy:         nexus.IntentPlan,
			TaskRequirements: nexus.TaskRequirements{RequiresDecomposition: true},
		}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "implemente uma plataforma completa", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != Promote {
		t.Fatalf("decomposition-worthy PLAN must promote: %+v", decision)
	}
}

func TestPromoterDecideAUTOExplicitMissionPromotes(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "crie uma missão para revisar este módulo", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != Promote {
		t.Fatalf("explicit Mission request must promote: %+v", decision)
	}
	if !decision.ExplicitMissionRequest {
		t.Fatalf("explicit Mission request must be marked as user-owned: %+v", decision)
	}
}

func TestPromoterHeuristicPromotionIsNotMarkedExplicit(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "implemente backend e frontend com testes", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != Promote || !decision.Delegation.Delegate {
		t.Fatalf("fixture must be heuristically promoted: %+v", decision)
	}
	if decision.ExplicitMissionRequest {
		t.Fatalf("heuristic promotion must remain eligible for Lead fallback: %+v", decision)
	}
}

func TestPromoterExplicitMissionOverridesDirectHeuristic(t *testing.T) {
	p := Promoter{DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
		return nexus.IntentDecision{Strategy: nexus.IntentDirect}, nil
	}}
	decision, err := p.Decide(context.Background(), "project", "execute esta correção como Mission", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != Promote || !decision.ExplicitMissionRequest {
		t.Fatalf("explicit Mission request must override DIRECT heuristic: %+v", decision)
	}
}

func TestExplicitMissionMatcherDoesNotTreatPermissionWordsAsMission(t *testing.T) {
	for _, input := range []string{
		"ajuste a permission policy deste provider",
		"corrija a submission do formulário",
		"revise a transmission layer",
		"configure permissions no runtime",
		"documente o comportamento mission-critical do scheduler",
		"explique nossa mission statement para o README",
	} {
		if IsExplicitMissionRequest(input) {
			t.Fatalf("ordinary word containing mission substring must not request Mission: %q", input)
		}
	}
}

func TestExplicitMissionMatcherAcceptsStandaloneLocalizedForms(t *testing.T) {
	for _, input := range []string{
		"execute como Mission",
		"crie uma missão para isso",
		"rode como missao",
		"execute de forma autônoma",
		"faça uma execução autonoma",
		"quero uma missão para revisar o módulo",
	} {
		if !IsExplicitMissionRequest(input) {
			t.Fatalf("expected explicit Mission intent for %q", input)
		}
	}
}
