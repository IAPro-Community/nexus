package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestValidatePlanReadinessRejectsUnverifiableTask(t *testing.T) {
	problems := validatePlanReadiness(store.WorkPlan{Phases: []store.PlanPhase{{Packages: []store.WorkPackage{{ID: "integration", Title: "Integration", Goal: "", AcceptanceCriteria: nil}}}}})
	if len(problems) != 3 {
		t.Fatalf("expected missing goal, DoD and evidence blockers, got %v", problems)
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "executable goal") || !strings.Contains(joined, "verification") {
		t.Fatalf("unexpected readiness blockers: %v", problems)
	}
}

func TestValidatePlanReadinessAcceptsCompleteTask(t *testing.T) {
	problems := validatePlanReadiness(store.WorkPlan{Phases: []store.PlanPhase{{Packages: []store.WorkPackage{{ID: "integration", Title: "Integration", Goal: "Ship the integration", AcceptanceCriteria: []string{"API and UI work together"}, VerificationRequirements: []string{"go test ./..."}}}}}})
	if len(problems) != 0 {
		t.Fatalf("complete task should be ready, got %v", problems)
	}
}
