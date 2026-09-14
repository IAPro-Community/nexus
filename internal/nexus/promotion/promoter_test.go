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
