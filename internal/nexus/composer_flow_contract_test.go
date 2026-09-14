package nexus

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestComposerAndFlowRemainOptionalAcrossIntentPipeline(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	if decision.Strategy != IntentDirect {
		t.Fatalf("C1 must be DIRECT: %+v", decision)
	}
	if ComposerIsRequired(decision) || FlowIsRequired(decision) {
		t.Fatal("DIRECT must not require Composer or Flow")
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := contextsnapshot.Discover(root, contextsnapshot.Metadata{ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	improve := DecideIntent("quero melhorar os testes", &snapshot)
	if ComposerIsRequired(improve) {
		t.Fatalf("C2 must not force Composer: %+v", improve)
	}

	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "optional-composer", CanonicalPath: root})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Fix test", "atomic", []store.PlanPhase{{
		ID: "phase", Title: "Fix", Order: 1, Packages: []store.WorkPackage{{
			ID: "pkg", Title: "Fix", Goal: "corrija este teste quebrado", Status: "READY",
			AcceptanceCriteria: []string{"tests pass"}, VerificationRequirements: []string{"true"},
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	flow := FlowFromWorkPlan(*plan)
	roundTrip := WorkPlanFromFlow(flow)
	if roundTrip.ID != plan.ID || roundTrip.Title != plan.Title {
		t.Fatalf("Flow projection mutated WorkPlan identity")
	}
	semantic := WorkPlanFromFlow(flow)
	flow.UpdatedAt = flow.UpdatedAt.Add(time.Second)
	if WorkPlanFromFlow(flow).Phases[0].Packages[0].Goal != semantic.Phases[0].Packages[0].Goal {
		t.Fatal("non-semantic Flow metadata must not change WorkPlan goal")
	}

	spec, err := planToRunnerSpec(plan, "snap")
	if err != nil {
		t.Fatal(err)
	}
	if spec.ID != plan.ID {
		t.Fatal("WorkPlan must remain the execution spec source")
	}
	_ = runner.DefaultAutonomyContract()
}

func TestFlowSemanticEditProducesWorkPlanRevisionPayload(t *testing.T) {
	plan := canonicalFlowPlanFixture()
	flow := FlowFromWorkPlan(plan)
	flow.Steps[0].Goal = "Plan work with an explicit acceptance gate"
	revised := WorkPlanFromFlow(flow)
	if revised.Phases[0].Packages[0].Goal == plan.Phases[0].Packages[0].Goal {
		t.Fatal("semantic Flow edit must change WorkPlan goal")
	}
	if revised.ID != plan.ID {
		t.Fatal("revision must keep WorkPlan identity")
	}
}
