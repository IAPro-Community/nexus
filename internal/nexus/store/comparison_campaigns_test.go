package store

import (
	"testing"
)

func TestComparisonCampaignPersistsPairedPlansAndOneWayLaunchOwnership(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "comparison campaign", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	source, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "source plan"})
	if err != nil {
		t.Fatal(err)
	}
	variantA := WorkPlan{ID: "plan-campaign-a", ProjectID: project.ID, Title: "source · A", Phases: []PlanPhase{{ID: "phase-a", Packages: []WorkPackage{{ID: "pkg-a"}}}}}
	variantB := WorkPlan{ID: "plan-campaign-b", ProjectID: project.ID, Title: "source · B", Phases: []PlanPhase{{ID: "phase-b", Packages: []WorkPackage{{ID: "pkg-b"}}}}}
	campaign, err := st.CreateComparisonCampaign(ComparisonCampaign{
		ProjectID: project.ID, SourcePlanID: source.ID, SourcePlanRevision: 1, BaseRevision: "base-sha",
		VariantAName: "A", VariantAInstructions: "Approach A", VariantBName: "B", VariantBInstructions: "Approach B",
	}, variantA, variantB)
	if err != nil {
		t.Fatal(err)
	}
	if campaign.State != ComparisonCampaignPrepared || campaign.VariantAPlanID != variantA.ID || campaign.VariantBPlanID != variantB.ID {
		t.Fatalf("campaign wasn't atomically prepared: %+v", campaign)
	}
	loadedA, err := st.GetWorkPlan(variantA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedA.CurrentRevision != 1 || loadedA.Status != "READY" {
		t.Fatalf("variant plan wasn't frozen at revision 1: %+v", loadedA)
	}
	if _, err := st.BeginComparisonCampaignLaunch(campaign.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.BeginComparisonCampaignLaunch(campaign.ID); err == nil {
		t.Fatal("second launch must not acquire the same campaign")
	}
	if err := st.UpsertMissionRun(MissionRunRecord{ID: "campaign-run-a", PlanID: variantA.ID, ProjectID: project.ID, State: "EXECUTING", PayloadJSON: `{}`}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMissionRun(MissionRunRecord{ID: "campaign-run-b", PlanID: variantB.ID, ProjectID: project.ID, State: "EXECUTING", PayloadJSON: `{}`}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AttachComparisonCampaignRun(campaign.ID, "A", "campaign-run-a"); err != nil {
		t.Fatal(err)
	}
	partial, err := st.FailComparisonCampaignLaunch(campaign.ID, "alternative B link interrupted")
	if err != nil || partial.State != ComparisonCampaignPartial {
		t.Fatalf("one-sided launch must be recoverable as partial: %+v %v", partial, err)
	}
	if _, err := st.BeginComparisonCampaignResume(campaign.ID); err != nil {
		t.Fatalf("explicit resume of missing alternative must be available: %v", err)
	}
	attached, err := st.AttachComparisonCampaignRun(campaign.ID, "B", "campaign-run-b")
	if err != nil {
		t.Fatal(err)
	}
	if attached.State != ComparisonCampaignRunning || attached.RunAID != "campaign-run-a" || attached.RunBID != "campaign-run-b" {
		t.Fatalf("paired run ownership wasn't persisted: %+v", attached)
	}
	found, err := st.FindComparisonCampaignByRuns("campaign-run-b", "campaign-run-a")
	if err != nil || found.ID != campaign.ID {
		t.Fatalf("campaign lookup by run pair = %+v, %v", found, err)
	}
}
