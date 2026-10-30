package nexus

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// The production Flow generator shares the existing Nexus Intelligence
// contract. This deterministic fake makes the persistence boundary testable
// without calling cloud APIs, spawning an Agent or starting a Mission.
type flowPlannerFixtureProvider struct {
	outlines []intelligence.WorkPackageOutline
}

func (p *flowPlannerFixtureProvider) Name() string { return "test-flow-planner" }
func (p *flowPlannerFixtureProvider) Available(context.Context) bool { return true }
func (p *flowPlannerFixtureProvider) AnalyzeIntent(context.Context, string, map[string]any) (*intelligence.IntentAnalysis, error) {
	return &intelligence.IntentAnalysis{Intent: "Ship login", Scope: "project"}, nil
}
func (p *flowPlannerFixtureProvider) EvaluateAmbiguities(context.Context, *intelligence.IntentAnalysis) ([]intelligence.AmbiguityItem, error) {
	return nil, nil
}
func (p *flowPlannerFixtureProvider) GeneratePlanOutline(context.Context, *intelligence.IntentAnalysis, map[string]string, map[string]any) ([]intelligence.WorkPackageOutline, error) {
	return p.outlines, nil
}
func (p *flowPlannerFixtureProvider) ExtractBrief(context.Context, string, map[string]any) (*intelligence.BriefExtraction, error) {
	return &intelligence.BriefExtraction{}, nil
}
func (p *flowPlannerFixtureProvider) PlanFromGoal(context.Context, string, map[string]any) (*intelligence.IntentAnalysis, []intelligence.AmbiguityItem, []intelligence.WorkPackageOutline, error) {
	return &intelligence.IntentAnalysis{Intent: "Ship login", Scope: "project", RiskLevel: "medium"}, nil, p.outlines, nil
}

func TestGenerateFlowFromPromptCreatesReviewableWorkPlanWithDependencies(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil { t.Fatal(err) }
	project, err := st.CreateProject(store.Project{Name: "SemanticFlow", CanonicalPath: t.TempDir()})
	if err != nil { t.Fatal(err) }
	n.intelligenceProviderForTest = &flowPlannerFixtureProvider{outlines: []intelligence.WorkPackageOutline{
		flowOutline("Contract", "Define authenticated session interface"),
		flowOutline("Backend", "Implement authenticated API endpoints", "Contract"),
		flowOutline("Frontend", "Implement React login and routing", "Contract"),
		flowOutline("Verification", "Test integration against acceptance conditions", "Backend", "Frontend"),
	}}
	plan, err := n.GeneratePlanFromIntent(context.Background(),project.ID,
		"Build backend API and React frontend with integration tests")
	if err != nil { t.Fatal(err) }
	if len(plan.Phases)!=1 || len(plan.Phases[0].Packages)!=4 {
		t.Fatalf("semantic plan lost activities: %+v",plan.Phases)
	}
	pkgs:=plan.Phases[0].Packages
	if len(pkgs[3].Dependencies)!=2 || pkgs[3].Dependencies[0]!=pkgs[1].ID ||
		pkgs[3].Dependencies[1]!=pkgs[2].ID {
		t.Fatalf("integration dependencies lost: %+v",pkgs[3].Dependencies)
	}
	if len(pkgs[3].AcceptanceCriteria)==0 { t.Fatal("acceptance evidence lost") }
	plans,err:=st.ListWorkPlans(project.ID)
	if err!=nil || len(plans)!=1 { t.Fatalf("expected one draft only: %v %+v",err,plans) }
}

func TestGenerateFlowFromPromptRejectsIncompleteAIPlanBeforePersistence(t *testing.T) {
	n:=openTestNexus(t)
	st,err:=n.OpenProject()
	if err!=nil {t.Fatal(err)}
	project,err:=st.CreateProject(store.Project{Name:"FailClosedFlow",CanonicalPath:t.TempDir()})
	if err!=nil {t.Fatal(err)}
	n.intelligenceProviderForTest=&flowPlannerFixtureProvider{outlines:[]intelligence.WorkPackageOutline{
		flowOutline("Everything", "Implement backend API and React frontend"),
	}}
	if _,err=n.GeneratePlanFromIntent(context.Background(),project.ID,
		"Implement backend API and React frontend with end-to-end tests");err==nil {
		t.Fatal("planner accepted a single generic activity")
	}
	plans,err:=st.ListWorkPlans(project.ID)
	if err!=nil || len(plans)!=0 {
		t.Fatalf("invalid response was persisted as a draft: %v %+v",err,plans)
	}
}
