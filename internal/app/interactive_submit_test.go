package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/promotion"
)

func TestDecideInteractiveSubmitBypassesProviderSlash(t *testing.T) {
	slow := promotion.Promoter{
		DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
			t.Fatal("provider slash must not call DecideIntent")
			return nexus.IntentDecision{}, nil
		},
	}
	decision := decideInteractiveSubmitDecision(context.Background(), slow, "project", "/status", nexus.DelegationAuto)
	action := decision.Action
	if action != promotion.KeepWithLead {
		t.Fatalf("action=%s want KeepWithLead", action)
	}
}

func TestDecideInteractiveSubmitTimesOutToKeepWithLead(t *testing.T) {
	slow := promotion.Promoter{
		DecideIntent: func(ctx context.Context, _, _ string) (nexus.IntentDecision, error) {
			select {
			case <-ctx.Done():
				return nexus.IntentDecision{}, ctx.Err()
			case <-time.After(2 * time.Second):
				return nexus.IntentDecision{Strategy: nexus.IntentDirect}, nil
			}
		},
	}
	start := time.Now()
	decision := decideInteractiveSubmitDecision(context.Background(), slow, "project", "implemente backend e frontend com testes e2e", nexus.DelegationAuto)
	action := decision.Action
	elapsed := time.Since(start)
	if action != promotion.KeepWithLead {
		t.Fatalf("action=%s want KeepWithLead on timeout", action)
	}
	if elapsed > time.Second {
		t.Fatalf("decide held Enter for %s", elapsed)
	}
	if elapsed < 100*time.Millisecond {
		t.Fatalf("expected timeout path, finished too fast: %s", elapsed)
	}
}

func TestDecideInteractiveSubmitPropagatesPromote(t *testing.T) {
	fast := promotion.Promoter{
		DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
			return nexus.IntentDecision{Strategy: nexus.IntentPlan}, nil
		},
	}
	action := decideInteractiveSubmit(context.Background(), fast, "project", "implemente backend API, frontend React e testes E2E", nexus.DelegationAuto)
	if action != promotion.Promote {
		t.Fatalf("action=%s want Promote (got err path?)", action)
	}
}

func TestDecideInteractiveSubmitKeepsOnIntentError(t *testing.T) {
	failing := promotion.Promoter{
		DecideIntent: func(context.Context, string, string) (nexus.IntentDecision, error) {
			return nexus.IntentDecision{}, errors.New("boom")
		},
	}
	action := decideInteractiveSubmit(context.Background(), failing, "project", "hello world", nexus.DelegationAuto)
	if action != promotion.KeepWithLead {
		t.Fatalf("action=%s", action)
	}
}

func TestShouldFallbackPromotionToLeadOnlyForHeuristicPromotions(t *testing.T) {
	heuristic := promotion.Decision{Action: promotion.Promote}
	if !shouldFallbackPromotionToLead(heuristic) {
		t.Fatal("heuristic promotion must fall back to Lead when Mission admission fails")
	}

	explicit := promotion.Decision{Action: promotion.Promote, ExplicitMissionRequest: true}
	if shouldFallbackPromotionToLead(explicit) {
		t.Fatal("explicit Mission request must remain blocked instead of silently executing as Direct")
	}

	keep := promotion.Decision{Action: promotion.KeepWithLead}
	if shouldFallbackPromotionToLead(keep) {
		t.Fatal("non-promotion decision must not be treated as a Mission fallback")
	}
}

func TestConfirmedInteractiveDelegationModeClearsAskPendingApproval(t *testing.T) {
	if got := confirmedInteractiveDelegationMode(nexus.DelegationAsk); got != nexus.DelegationAuto {
		t.Fatalf("confirmed ASK must dispatch as AUTO, got %s", got)
	}
	if got := confirmedInteractiveDelegationMode(nexus.DelegationOff); got != nexus.DelegationOff {
		t.Fatalf("OFF must remain OFF, got %s", got)
	}
	if got := confirmedInteractiveDelegationMode(nexus.DelegationAuto); got != nexus.DelegationAuto {
		t.Fatalf("AUTO must remain AUTO, got %s", got)
	}
}
