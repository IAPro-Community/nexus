package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/entitlement"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestRequireCapabilityUsesServerSideGrantBoundary(t *testing.T) {
	n := openTestNexus(t)
	n.SetEntitlementService(entitlement.StaticService{Grants: entitlement.Grants{
		Capabilities: map[entitlement.Capability]bool{
			entitlement.CapMissionsRun:  false,
			entitlement.CapRemoteAccess: true,
		},
	}})
	if err := n.RequireCapability(context.Background(), entitlement.CapRemoteAccess); err != nil {
		t.Fatalf("granted capability rejected: %v", err)
	}
	if err := n.RequireCapability(context.Background(), entitlement.CapMissionsRun); err == nil {
		t.Fatal("denied capability must fail closed")
	}
}

func TestMissionStartIsRejectedBeforeRunCreationWhenEntitlementDenied(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Entitlement", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Denied mission", "", []store.PlanPhase{{
		Title: "Phase",
		Packages: []store.WorkPackage{{
			Title: "A", Goal: "A", Status: "READY", Role: "implementer",
			AcceptanceCriteria: []string{"done"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	n.SetEntitlementService(entitlement.StaticService{Grants: entitlement.Grants{
		Capabilities: map[entitlement.Capability]bool{entitlement.CapMissionsRun: false},
	}})

	_, err = n.StartMissionRunApproved(
		context.Background(),
		plan.ID,
		plan.CurrentRevision,
		"",
		runner.DefaultAutonomyContract(),
		false,
	)
	if err == nil || !strings.Contains(err.Error(), "mission entitlement denied") {
		t.Fatalf("expected server-side entitlement denial, got %v", err)
	}
	runs, err := n.Runner().ListRuns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("denied Mission must not create durable runs: %+v", runs)
	}
}
