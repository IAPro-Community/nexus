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


func TestPromoterDecideAUTOPlanPromotesWithoutDelegation(t *testing.T) {
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
	if decision.Action != Promote {
		t.Fatalf("PLAN must promote to durable Mission even without fan-out: %+v", decision)
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

func TestPromoterDecideASKPlanWithoutDelegationRequiresConfirmation(t *testing.T) {
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
	if decision.Action != Ask {
		t.Fatalf("ASK must guard Mission promotion independently of delegation: %+v", decision)
	}
}
