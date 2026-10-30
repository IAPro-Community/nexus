package nexus

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

func flowOutline(title, goal string, deps ...string) intelligence.WorkPackageOutline {
	return intelligence.WorkPackageOutline{
		Title: title, Goal: goal, Role: "implementer",
		Acceptance: []string{"Deliverable meets the specified contract and can be verified"},
		Dependencies: deps,
	}
}

func TestValidateGeneratedFlowOutlinesAcceptsIndependentParallelWorkAndJoiningStep(t *testing.T) {
	prompt := "Build an API backend, React frontend, and run end-to-end tests."
	steps := []intelligence.WorkPackageOutline{
		flowOutline("Architecture", "Define approved API and session contracts"),
		flowOutline("Backend", "Implement handlers respecting approved contracts", "Architecture"),
		flowOutline("Frontend", "Implement React form and session handling", "step 1"),
		flowOutline("QA", "Verify both backend and frontend integration", "2", "#3"),
	}
	if err := validateGeneratedFlowOutlines(prompt, steps); err != nil {
		t.Fatalf("valid parallel Flow rejected: %v", err)
	}
	packages := packagesFromOutlines(prompt, steps)
	if len(packages) != 4 {
		t.Fatalf("want four packages, got %d", len(packages))
	}
	if len(packages[1].Dependencies) != 1 || packages[1].Dependencies[0] != packages[0].ID {
		t.Fatalf("backend dependencies not resolved: %+v", packages[1].Dependencies)
	}
	if len(packages[2].Dependencies) != 1 || packages[2].Dependencies[0] != packages[0].ID {
		t.Fatalf("frontend ordinal dependency not resolved: %+v", packages[2].Dependencies)
	}
	if len(packages[3].Dependencies) != 2 ||
		packages[3].Dependencies[0] != packages[1].ID ||
		packages[3].Dependencies[1] != packages[2].ID {
		t.Fatalf("QA join dependencies not resolved: %+v", packages[3].Dependencies)
	}
	if packages[0].Status != "READY" || packages[1].Status != "PENDING" ||
		packages[2].Status != "PENDING" || packages[3].Status != "PENDING" {
		t.Fatal("dependent activities must remain pending until prerequisites succeed")
	}
}

func TestValidateGeneratedFlowOutlinesRejectsUntrustworthyPlans(t *testing.T) {
	good := flowOutline("Backend", "Implement token session endpoints")
	cases := []struct {
		name string
		prompt string
		steps []intelligence.WorkPackageOutline
		want string
	}{
		{"empty", "create endpoint", nil, "expected 1"},
		{"multi-workstream collapsed", "Implement backend API and React frontend with tests",
			[]intelligence.WorkPackageOutline{good}, "one activity"},
		{"unknown dependency", "Build an auth feature", []intelligence.WorkPackageOutline{
			good, flowOutline("Review", "Review session security contracts", "Unobserved activity"),
		}, "unknown dependency"},
		{"cycle", "Build a feature", []intelligence.WorkPackageOutline{
			flowOutline("A", "Implement architectural model", "B"),
			flowOutline("B", "Review architectural model", "A"),
		}, "circular"},
		{"self dependency", "Fix API", []intelligence.WorkPackageOutline{
			flowOutline("A", "Implement the API", "#1"),
		}, "depends on itself"},
		{"duplicate title", "Implement feature", []intelligence.WorkPackageOutline{
			good, flowOutline("Backend", "Implement another API endpoint"),
		}, "duplicate"},
		{"placeholder", "Implement feature", []intelligence.WorkPackageOutline{
			flowOutline("...", "..."),
		}, "title/goal"},
		{"empty acceptance", "Fix API", []intelligence.WorkPackageOutline{
			{Title:"Backend", Goal:"Implement the API endpoint", Acceptance: []string{"measurable criterion"}},
		}, "acceptance criteria"},
		{"empty goal", "Fix API", []intelligence.WorkPackageOutline{
			flowOutline("Backend", ""),
		}, "title/goal"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGeneratedFlowOutlines(tt.prompt, tt.steps)
			if err == nil || !strings.Contains(err.Error(),tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestValidateGeneratedFlowOutlinesKeepsAtomicTaskDirect(t *testing.T) {
	if err := validateGeneratedFlowOutlines("Fix the typo in the footer",
		[]intelligence.WorkPackageOutline{flowOutline("Footer typo","Correct the footer label without changing layout")}); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFlowLimitsActivityCount(t *testing.T) {
	steps := make([]intelligence.WorkPackageOutline, maxGeneratedFlowActivities+1)
	for i := range steps {
		steps[i] = flowOutline("Independent activity", "Do an independently verifiable task")
	}
	if err := validateGeneratedFlowOutlines("Build a large application",steps); err==nil {
		t.Fatal("oversized AI work breakdown must be rejected")
	}
}
