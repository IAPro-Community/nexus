package nexus

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/contextsnapshot"
)

func TestDecideIntentRoutesAtomicFixDirectly(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	if decision.Strategy != IntentDirect || decision.RecommendedAction != "execute_single_work_unit" {
		t.Fatalf("unexpected direct decision: %+v", decision)
	}
}

func TestDecideIntentUsesProjectFactsBeforeClarifying(t *testing.T) {
	snapshot := &contextsnapshot.ProjectContextSnapshot{
		Completeness: contextsnapshot.CompletenessComplete,
		Identity:     contextsnapshot.CodeIdentity{IdentityDigest: "sha"},
		Facts: []contextsnapshot.ProjectFact{{
			Key: "commands.test", Value: "go test ./...",
		}},
	}
	// Short goal without plan verbs: facts about tests cover the unknown.
	decision := DecideIntent("rodar testes", snapshot)
	if decision.Strategy == IntentClarify {
		t.Fatalf("covering repository facts should prevent ritual clarification: %+v", decision)
	}
	if len(decision.KnownFacts) != 1 || decision.SnapshotIdentity != "sha" {
		t.Fatalf("expected bounded fact grounding: %+v", decision)
	}
}

func TestDecideIntentUnrelatedFactsDoNotSkipClarification(t *testing.T) {
	snapshot := &contextsnapshot.ProjectContextSnapshot{
		Completeness: contextsnapshot.CompletenessPartial,
		Facts: []contextsnapshot.ProjectFact{{
			Key: "stack.go", Value: true,
		}},
	}
	decision := DecideIntent("faz algo", snapshot)
	if decision.Strategy != IntentClarify {
		t.Fatalf("unrelated facts must not silence material clarification: %+v", decision)
	}
}

func TestDecideIntentRoutesCompleteMultiStepGoalToPlan(t *testing.T) {
	decision := DecideIntent("implemente a próxima etapa com testes, revisão e critérios de aceitação", nil)
	if decision.Strategy != IntentPlan || decision.RecommendedAction != "create_work_plan" {
		t.Fatalf("unexpected plan decision: %+v", decision)
	}
}

func TestDecideIntentC2DoesNotAskKnownTestTooling(t *testing.T) {
	snapshot := &contextsnapshot.ProjectContextSnapshot{
		Completeness: contextsnapshot.CompletenessComplete,
		Identity:     contextsnapshot.CodeIdentity{IdentityDigest: "repo"},
		Facts: []contextsnapshot.ProjectFact{
			{Key: "commands.test", Value: "go test ./..."},
			{Key: "commands.build", Value: "go build ./..."},
			{Key: "stack.test_framework", Value: "go-test"},
		},
	}
	decision := DecideIntent("quero melhorar os testes", snapshot)
	if ComposerIsRequired(decision) {
		t.Fatalf("contextual improvement must not require Composer: %+v", decision)
	}
	if FlowIsRequired(decision) {
		t.Fatal("Flow must remain optional")
	}
	for _, question := range decision.BlockingQuestions {
		lower := strings.ToLower(question)
		if strings.Contains(lower, "framework") || strings.Contains(lower, "onde") || strings.Contains(lower, "build command") {
			t.Fatalf("must not ask known repository facts: %q", question)
		}
	}
	if len(decision.KnownFacts) == 0 {
		t.Fatal("project intelligence facts must be recorded")
	}
}

func TestDecideIntentC3CompleteSpecificationAsksNothing(t *testing.T) {
	goal := "Implement user session timeout of 30 minutes on the existing auth middleware, keep current cookie names, add tests in internal/control/web, run go test ./internal/control/web, and do not change the login UI."
	decision := DecideIntent(goal, nil)
	if decision.Strategy != IntentPlan {
		t.Fatalf("complete specification must PLAN: %+v", decision)
	}
	if len(decision.BlockingQuestions) != 0 {
		t.Fatalf("complete specification must not ask ritual questions: %+v", decision)
	}
	if ComposerIsRequired(decision) {
		t.Fatal("Composer must not be required for a complete specification")
	}
}

func TestDecideIntentC4ConflictingProductBehaviorAsksOneQuestion(t *testing.T) {
	decision := DecideIntent("o botão salvar deve persistir o rascunho automaticamente e não deve persistir o rascunho automaticamente", nil)
	if decision.Strategy != IntentClarify {
		t.Fatalf("conflicting product behavior must CLARIFY: %+v", decision)
	}
	if !ComposerIsRequired(decision) {
		t.Fatal("real ambiguity must require Composer")
	}
	if len(decision.BlockingQuestions) != 1 {
		t.Fatalf("expected exactly one material question, got %+v", decision.BlockingQuestions)
	}
}

func TestIntentDecisionJSONContractRoundTrip(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	var decoded IntentDecision
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Strategy != IntentDirect || decoded.RecommendedAction != decision.RecommendedAction {
		t.Fatalf("intent contract drifted: %+v", decoded)
	}
}

func TestDecideIntentOnlyClarifiesMaterialMissingObjective(t *testing.T) {
	decision := DecideIntent("construa", nil)
	if decision.Strategy != IntentClarify || len(decision.BlockingQuestions) != 1 {
		t.Fatalf("expected one material clarification: %+v", decision)
	}
}

func TestPersistIntentDecisionFactsUsesExistingWorkPlanMetadata(t *testing.T) {
	decision := DecideIntent("corrija este teste quebrado", nil)
	facts, err := PersistIntentDecisionFacts(map[string]string{"framework": "go"}, decision)
	if err != nil {
		t.Fatal(err)
	}
	if facts[IntentDecisionFactKey] == "" || facts[IntentStrategyFactKey] != string(IntentDirect) || facts[IntentConfidenceFactKey] != "HIGH" {
		t.Fatalf("intent decision metadata was not persisted: %#v", facts)
	}
	if facts["framework"] != "go" {
		t.Fatalf("existing structured facts were lost: %#v", facts)
	}
}
