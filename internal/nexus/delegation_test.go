package nexus

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestDecideDelegationKeepsAtomicWorkWithLead(t *testing.T) {
	decision := DecideDelegation("Troque o texto Login para Entrar.", DelegationAuto)
	if decision.Delegate || len(decision.Workstreams) != 0 {
		t.Fatalf("atomic task must remain direct: %+v", decision)
	}
}

func TestDecideDelegationCreatesBoundedIndependentWorkstreams(t *testing.T) {
	decision := DecideDelegation("Implemente backend API, frontend React e testes E2E", DelegationAuto)
	if !decision.Delegate || !decision.Parallelizable || len(decision.Workstreams) != 3 {
		t.Fatalf("expected bounded delegation: %+v", decision)
	}
	if decision.Workstreams[0].PreferredRoles[0] != "backend-engineer" || decision.Workstreams[1].PreferredRoles[0] != "frontend-engineer" || decision.Workstreams[2].PreferredRoles[0] != "qa-engineer" {
		t.Fatalf("workstreams must express roles, not concrete Agents: %+v", decision.Workstreams)
	}
	if len(decision.Workstreams[2].Dependencies) != 2 {
		t.Fatalf("QA must depend on implementation workstreams: %+v", decision.Workstreams[2])
	}
}

func TestDecideDelegationDispatchBrakeAndModes(t *testing.T) {
	for _, test := range []struct {
		name     string
		mode     DelegationMode
		goal     string
		delegate bool
		pending  bool
	}{
		{name: "off", mode: DelegationOff, goal: "backend frontend e2e", delegate: false},
		{name: "ask", mode: DelegationAsk, goal: "backend frontend e2e", delegate: true, pending: true},
		{name: "collision", mode: DelegationAuto, goal: "backend and frontend in the same file", delegate: false},
		{name: "no benefit", mode: DelegationAuto, goal: "refactor one function", delegate: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			decision := DecideDelegation(test.goal, test.mode)
			if decision.Delegate != test.delegate || decision.PendingApproval != test.pending {
				t.Fatalf("decision=%+v", decision)
			}
		})
	}
}

func TestDelegationDecisionFactsSurviveRoundTrip(t *testing.T) {
	original := DecideDelegation("backend frontend e2e", DelegationAuto)
	original.LeadAgentID = "lead-agent"
	facts, err := PersistDelegationDecisionFacts(map[string]string{"existing": "value"}, original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadDelegationDecisionFacts(facts)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.LeadAgentID != original.LeadAgentID || decoded.Delegate != original.Delegate || len(decoded.Workstreams) != 3 {
		t.Fatalf("round trip lost decision: got=%+v want=%+v", decoded, original)
	}
	if facts["existing"] != "value" || facts[LeadAgentFactKey] != "lead-agent" {
		t.Fatalf("existing facts or lead identity lost: %+v", facts)
	}
}

func TestDelegationPolicyIsStableForSameInput(t *testing.T) {
	first := DecideDelegation("backend frontend e2e", DelegationAuto)
	second := DecideDelegation("backend frontend e2e", DelegationAuto)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same policy inputs must produce the same decision: first=%+v second=%+v", first, second)
	}
}

func TestInteractiveLeadAutomaticallyDelegatesSpecializedWorkUnits(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Delegation", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{
		ProjectID: project.ID, Goal: "Implemente cadastro com backend API, frontend React e testes E2E",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(proposal.Flow.Steps); got != 5 {
		t.Fatalf("expected planning + backend/frontend + review + verification pipeline, got %d", got)
	}
	var decision DelegationDecision
	if err := json.Unmarshal([]byte(proposal.Flow.StructuredFacts[DelegationDecisionFactKey]), &decision); err != nil {
		t.Fatal(err)
	}
	if !decision.Delegate || decision.LeadAgentID != "" {
		t.Fatalf("delegation should not replace lead identity: %+v", decision)
	}
	if proposal.Flow.Steps[3].Status != "PENDING" || len(proposal.Flow.Steps[3].Dependencies) != 2 || proposal.Flow.Steps[3].PhaseType != string(PhaseReview) {
		t.Fatalf("review dependency gate missing: %+v", proposal.Flow.Steps[3])
	}
	if proposal.Flow.Steps[0].AssignmentStrategy != FlowAssignmentAuto || proposal.Flow.Steps[1].AssignmentStrategy != FlowAssignmentAuto {
		t.Fatal("delegated workstreams must use existing AUTO allocation")
	}
}

func TestCreateMissionAgentPersistsRichAgentSpecWithoutRuntimeAffinity(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "RichAgent", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	requirements := TaskRequirements{Role: "backend-engineer", PreferredRoles: []string{"backend-engineer"}, Domains: []string{"backend", "api"}, RequiredCapabilities: []string{"go", "api", "headless"}, DesiredStrengths: []string{"testing"}}
	raw, _ := json.Marshal(requirements)
	agent, err := createMissionAgent(st, project.ID, &runner.PackageRun{PackageID: "backend", Title: "Backend", Role: "implementer", TaskRequirements: string(raw)}, "Auto · ")
	if err != nil {
		t.Fatal(err)
	}
	if agent.CurrentRevisionID == "" || agent.Role != "backend-engineer" {
		t.Fatalf("rich persistent Agent was not created: %+v", agent)
	}
	revision, err := st.GetRevision(agent.CurrentRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	config, err := ParseAgentConfig(revision.Config)
	if err != nil {
		t.Fatal(err)
	}
	if config.Provider != "" || config.AgentSpec.Role != "backend-engineer" || !reflect.DeepEqual(config.AgentSpec.Domains, requirements.Domains) {
		t.Fatalf("AgentSpec must remain provider-independent and rich: %+v", config)
	}
	if config.Isolation != "worktree" {
		t.Fatalf("autonomous Mission Agent must use worktree isolation, got %q", config.Isolation)
	}
	if !config.AgentSpec.VerificationPolicy.RequireTests {
		t.Fatal("created specialist must carry verification policy")
	}
}
