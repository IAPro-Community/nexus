package nexus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestBuildRoutingDecisionReportProjectsPersistedSelection(t *testing.T) {
	raw, err := json.Marshal(RuntimeRoutingDecision{
		TaskID: "step-1", AgentID: "agent-1",
		SelectedProvider: "opencode", SelectedProfile: "qa", SelectedModel: "mimo",
		Reason: "QA capability match; preferred runtime available",
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildRoutingDecisionReport(&runner.MissionRun{
		ID: "run-1", PlanID: "plan-1", PlanRevision: 4,
		PackageRuns: []runner.PackageRun{{PackageID: "step-1", Title: "Run QA", State: runner.StateVerified, RoutingDecisionJSON: string(raw)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.RunID != "run-1" || report.PlanRevision != 4 || len(report.Decisions) != 1 {
		t.Fatalf("unexpected routing report: %+v", report)
	}
	decision := report.Decisions[0]
	if decision.PackageID != "step-1" || decision.Decision.SelectedProvider != "opencode" || decision.Decision.SelectedProfile != "qa" || decision.Decision.SelectedModel != "mimo" {
		t.Fatalf("routing selection was not projected: %+v", decision)
	}
}

func TestBuildRoutingDecisionReportRejectsCorruptPersistedDecision(t *testing.T) {
	_, err := BuildRoutingDecisionReport(&runner.MissionRun{
		ID: "run-corrupt", PackageRuns: []runner.PackageRun{{PackageID: "step-1", RoutingDecisionJSON: "not-json"}},
	})
	if err == nil {
		t.Fatal("corrupt persisted routing decision must not be silently hidden")
	}
}

func TestBuildRoutingDecisionReportProjectsPersistedIntent(t *testing.T) {
	decision := IntentDecision{Strategy: IntentPlan, Confidence: "HIGH", RecommendedAction: "create_work_plan"}
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildRoutingDecisionReportWithPlan(&runner.MissionRun{ID: "run-intent", PlanID: "plan-1"}, &store.WorkPlan{ID: "plan-1", StructuredFacts: map[string]string{IntentDecisionFactKey: string(raw)}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Intent == nil || report.Intent.Strategy != IntentPlan || report.Intent.Confidence != "HIGH" {
		t.Fatalf("persisted intent was not projected: %+v", report.Intent)
	}
}

func TestBuildRoutingDecisionReportRejectsCorruptPersistedIntent(t *testing.T) {
	_, err := BuildRoutingDecisionReportWithPlan(&runner.MissionRun{ID: "run-intent-corrupt"}, &store.WorkPlan{StructuredFacts: map[string]string{IntentDecisionFactKey: "not-json"}})
	if err == nil {
		t.Fatal("corrupt persisted intent must not be silently hidden")
	}
}

func TestBuildRoutingDecisionReportProjectsPersistedDelegation(t *testing.T) {
	decision := DecideDelegation("backend frontend e2e", DelegationAuto)
	facts, err := PersistDelegationDecisionFacts(nil, decision)
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildRoutingDecisionReportWithPlan(&runner.MissionRun{ID: "run-delegation", PlanID: "plan-delegation"}, &store.WorkPlan{ID: "plan-delegation", StructuredFacts: facts})
	if err != nil {
		t.Fatal(err)
	}
	if report.Delegation == nil || !report.Delegation.Delegate || len(report.Delegation.Workstreams) != 3 {
		t.Fatalf("persisted delegation was not projected: %+v", report.Delegation)
	}
}

func TestBuildRoutingDecisionReportComparesApprovedExecutionPreview(t *testing.T) {
	actual := RuntimeRoutingDecision{
		TaskID: "step-1", AgentID: "agent-1",
		SelectedProvider: "opencode", SelectedProfile: "qa", SelectedModel: "mimo",
		Fallback: true, SwitchReason: SwitchReasonRateLimited,
		Reason: "previewed resource became rate limited; scheduler selected a healthy alternative",
	}
	raw, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	preview := &PlanExecutionPreview{
		PlanID: "plan-1", ProjectID: "project-1", Revision: 4,
		GeneratedAt: time.Date(2026, 9, 21, 18, 0, 0, 0, time.UTC),
		Steps: []StepExecutionPreview{{
			PackageID: "step-1",
			Agent:     PreviewAgentDecision{AgentID: "agent-1"},
			Resource:  PreviewResourceDecision{Provider: "codex", Profile: "main"},
			Model:     PreviewModelDecision{Model: "strong"},
		}},
	}
	report, err := BuildRoutingDecisionReportWithPlanAndPreview(&runner.MissionRun{
		ID: "run-1", PlanID: "plan-1", PlanRevision: 4,
		PackageRuns: []runner.PackageRun{{PackageID: "step-1", RoutingDecisionJSON: string(raw)}},
	}, nil, preview)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Decisions) != 1 {
		t.Fatalf("expected one routing decision, got %d", len(report.Decisions))
	}
	entry := report.Decisions[0]
	if entry.Preview == nil || entry.Preview.Provider != "codex" || entry.Preview.Profile != "main" || entry.Preview.Model != "strong" {
		t.Fatalf("approved preview baseline missing: %+v", entry.Preview)
	}
	if !entry.ChangedSincePreview || entry.ChangeReason == "" {
		t.Fatalf("preview divergence must remain explainable: %+v", entry)
	}
}

func TestBuildRoutingDecisionReportDoesNotInventPreviewDivergence(t *testing.T) {
	actual := RuntimeRoutingDecision{
		TaskID: "step-1", AgentID: "agent-1",
		SelectedProvider: "codex", SelectedProfile: "main", SelectedModel: "strong",
		Reason: "selected by scheduler",
	}
	raw, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	preview := &PlanExecutionPreview{
		PlanID: "plan-1", Revision: 2, GeneratedAt: time.Now().UTC(),
		Steps: []StepExecutionPreview{{
			PackageID: "step-1",
			Agent:     PreviewAgentDecision{AgentID: "agent-1"},
			Resource:  PreviewResourceDecision{Provider: "codex", Profile: "main"},
			Model:     PreviewModelDecision{Model: "strong"},
		}},
	}
	report, err := BuildRoutingDecisionReportWithPlanAndPreview(&runner.MissionRun{
		ID: "run-1", PlanID: "plan-1", PlanRevision: 2,
		PackageRuns: []runner.PackageRun{{PackageID: "step-1", RoutingDecisionJSON: string(raw)}},
	}, nil, preview)
	if err != nil {
		t.Fatal(err)
	}
	entry := report.Decisions[0]
	if entry.ChangedSincePreview || entry.ChangeReason != "" {
		t.Fatalf("matching live route must not be labeled as changed: %+v", entry)
	}
}
