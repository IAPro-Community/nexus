package entitlement

import (
	"context"
	"testing"
)

func TestBetaGrantsGateCapabilitiesNotPlanNames(t *testing.T) {
	svc := StaticService{}
	g, err := svc.GrantsFor(context.Background(), "any-account")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []Capability{CapMissionsRun, CapRemoteAccess, CapRoutingAdvanced} {
		if !Require(g, c) {
			t.Errorf("beta must grant %q", c)
		}
	}
	if Require(g, CapTeamSharedContext) {
		t.Error("team context must stay gated during beta")
	}
	if g.PlanLabel == "" {
		t.Error("plan label must be set for display")
	}
}

func TestStaticServiceFallsBackToBetaGrants(t *testing.T) {
	g, err := StaticService{}.GrantsFor(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !g.Has(CapMissionsRun) || g.MaxMonthlyMissions != -1 {
		t.Fatalf("zero StaticService must yield beta grants: %+v", g)
	}
}
