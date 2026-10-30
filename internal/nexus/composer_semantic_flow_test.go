package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func composerSemanticFixture(n *Nexus, atomic bool) {
	outlines := []intelligence.WorkPackageOutline{
		flowOutline("Architecture", "Define implementation interfaces and acceptance contracts"),
		flowOutline("Backend", "Implement backend endpoints against approved interfaces", "Architecture"),
		flowOutline("Frontend", "Implement frontend screens against approved interfaces", "Architecture"),
		flowOutline("Verification", "Run integration tests against the accepted backend and frontend contracts", "Backend", "Frontend"),
	}
	outlines[0].Role = "architect"
	outlines[3].Role = "tester"
	if atomic {
		outlines = []intelligence.WorkPackageOutline{
			flowOutline("Small feature", "Implement the requested small feature and verify its behavior"),
		}
	}
	n.intelligenceProviderForTest = &flowPlannerFixtureProvider{outlines: outlines}
}

func TestComposerSemanticFlowPreservesFrozenProvenanceAndGranularSteps(t *testing.T) {
	n:=openTestNexus(t)
	st,err:=n.OpenProject()
	if err!=nil {t.Fatal(err)}
	project,err:=st.CreateProject(store.Project{Name:"ComposerSemanticParity",CanonicalPath:t.TempDir()})
	if err!=nil {t.Fatal(err)}
	view,err:=n.CreateComposerSessionWithPrompt(context.Background(),project.ID,
		"Build secure authentication", "Implement backend API, React frontend and integration tests with independent review.")
	if err!=nil {t.Fatal(err)}
	artifact,err:=n.FinalizeComposerSession(context.Background(),view.Session.ID,nil,true)
	if err!=nil {t.Fatal(err)}
	composerSemanticFixture(n,false)

	plan,err:=n.MaterializePromptArtifactAsFlow(context.Background(),artifact.ID)
	if err!=nil {t.Fatal(err)}
	provider := n.intelligenceProviderForTest.(*flowPlannerFixtureProvider)
	if provider.lastGoal != artifact.Content || provider.lastContext["source_artifact_id"] != artifact.ID ||
		provider.lastContext["source_artifact_hash"] != artifact.Hash ||
		provider.lastContext["source"] != "FINALIZED_COMPOSER_ARTIFACT" {
		t.Fatalf("planner did not receive the exact immutable Composer prompt and artifact identity")
	}
	if provider.lastContext["intelligence_memories"] != nil {
		t.Fatal("a finalized Composer prompt must not be enriched with new mutable memories")
	}
	if len(plan.Phases)!=1 || len(plan.Phases[0].Packages)!=4 {
		t.Fatalf("prompt was not decomposed into specific activities: %+v",plan.Phases)
	}
	steps:=plan.Phases[0].Packages
	if len(steps[3].Dependencies)!=2 ||
		steps[3].Dependencies[0]!=steps[1].ID ||
		steps[3].Dependencies[1]!=steps[2].ID {
		t.Fatalf("verification lost join dependencies: %+v",steps[3].Dependencies)
	}
	for _,step:=range steps {
		if step.TaskRequirements=="" || len(step.AcceptanceCriteria)==0 {
			t.Fatalf("generated step lacks executable requirements: %+v",step)
		}
		if step.CompiledPrompt==artifact.Content {
			t.Fatal("scoped multi-step task incorrectly executes entire Composer prompt")
		}
	}
	facts:=plan.StructuredFacts
	if facts["nexus.source_artifact_id"]!=artifact.ID ||
		facts["nexus.source_artifact_hash"]!=artifact.Hash ||
		facts["composer.handoff.v1"]=="" || facts["composer.context"]=="" {
		t.Fatalf("immutable Composer provenance lost: %+v",facts)
	}
	plans,err:=st.ListWorkPlans(project.ID)
	if err!=nil||len(plans)!=1 {t.Fatalf("expected one reviewable Flow: %v %+v",err,plans)}
}

func TestComposerSemanticFlowRejectsInvalidProviderPlanWithoutFallback(t *testing.T) {
	n:=openTestNexus(t)
	st,err:=n.OpenProject()
	if err!=nil {t.Fatal(err)}
	project,err:=st.CreateProject(store.Project{Name:"ComposerFailClosed",CanonicalPath:t.TempDir()})
	if err!=nil {t.Fatal(err)}
	view,err:=n.CreateComposerSessionWithPrompt(context.Background(),project.ID,
		"Build secure login","Implement backend API and React frontend with integration tests")
	if err!=nil {t.Fatal(err)}
	artifact,err:=n.FinalizeComposerSession(context.Background(),view.Session.ID,nil,true)
	if err!=nil {t.Fatal(err)}
	// A deterministic invalid AI response demonstrates the same hard failure
	// path as an unavailable provider, without modifying global config.
	n.intelligenceProviderForTest=&flowPlannerFixtureProvider{outlines:[]intelligence.WorkPackageOutline{
		flowOutline("Everything","Implement backend API and React frontend and all tests"),
	}}
	if _,err=n.MaterializePromptArtifactAsFlow(context.Background(),artifact.ID);err==nil ||
		!strings.Contains(err.Error(),"one activity") {
		t.Fatalf("generic fallback was silently admitted: %v",err)
	}
	plans,err:=st.ListWorkPlans(project.ID)
	if err!=nil||len(plans)!=0 {t.Fatalf("invalid Composer Flow persisted: %v %+v",err,plans)}
	saved,err:=st.GetPromptArtifact(artifact.ID)
	if err!=nil||saved.Content!=artifact.Content||saved.Hash!=artifact.Hash {
		t.Fatalf("finalized source artifact changed during failed handoff: %v",err)
	}
}

func TestComposerSemanticFlowBlocksProviderClarificationWithoutLosingLineage(t *testing.T) {
	n:=openTestNexus(t)
	st,err:=n.OpenProject()
	if err!=nil {t.Fatal(err)}
	project,err:=st.CreateProject(store.Project{Name:"ComposerClarificationSafe",CanonicalPath:t.TempDir()})
	if err!=nil {t.Fatal(err)}
	view,err:=n.CreateComposerSessionWithPrompt(context.Background(),project.ID,
		"Build secure backend and React interface","Implement backend API, React frontend and integration tests with security review")
	if err!=nil {t.Fatal(err)}
	artifact,err:=n.FinalizeComposerSession(context.Background(),view.Session.ID,nil,true)
	if err!=nil {t.Fatal(err)}
	composerSemanticFixture(n,false)
	provider:=n.intelligenceProviderForTest.(*flowPlannerFixtureProvider)
	provider.unknowns=[]intelligence.AmbiguityItem{{
		Key:"credential_strategy",
		Level:intelligence.AmbiguityBlocking,
		Question:"Which identity provider is approved?",
		IsResolved:false,
	}}
	if _,err=n.MaterializePromptArtifactAsFlow(context.Background(),artifact.ID);err==nil ||
		!strings.Contains(err.Error(),"revise and finalize") {
		t.Fatalf("Composer must not create a generic continuation without frozen lineage: %v",err)
	}
	plans,err:=st.ListWorkPlans(project.ID)
	if err!=nil || len(plans)!=0 {t.Fatalf("blocked Composer Flow was persisted: %v %+v",err,plans)}
}
