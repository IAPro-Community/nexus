package nexus

import "testing"

func TestComposerInterviewPrioritizesWeakestDimension(t *testing.T) {
	brief := newComposerBrief("Improve provider routing", "")
	brief.Scope.InScope = []string{"provider routing"}
	brief.Context.Project = "nexus"
	brief.Context.Evidence = []string{"repository inspected"}
	brief.Environment.Stack = []string{"Go"}
	refreshComposerBrief(&brief)

	next, ok := selectComposerNextUnknownForBrief(brief)
	if !ok {
		t.Fatal("expected an interview question")
	}
	if next.Dimension != "acceptance" {
		t.Fatalf(
			"expected weakest acceptance dimension first, got %+v (interview=%+v)",
			next,
			brief.Interview,
		)
	}
	if next.PriorityScore <= 0 {
		t.Fatalf("expected information-gain priority, got %+v", next)
	}
}

func TestComposerInterviewDoesNotReaskInferredProjectFact(t *testing.T) {
	brief := newComposerBrief("Improve dashboard behavior", "")
	brief.Environment.Stack = []string{"React", "TypeScript"}
	refreshComposerBrief(&brief)

	for _, unknown := range brief.Unknowns {
		if unknown.Field == "environment.stack" && unknown.Status == PromptUnknownOpen {
			t.Fatalf("repository-covered stack must not remain open: %+v", unknown)
		}
	}
	next, ok := selectComposerNextUnknownForBrief(brief)
	if ok && next.Field == "environment.stack" {
		t.Fatalf("repository-covered stack was selected again: %+v", next)
	}
}

func TestComposerInterviewAddsSinglePressurePassNearConvergence(t *testing.T) {
	brief := newComposerBrief(
		"Choose the best provider automatically",
		"Use automatic provider routing with fallback; success means the request completes, tests cover routing, and verification runs go test.",
	)
	brief.Context.Project = "nexus"
	brief.Context.Evidence = []string{"repository inspected"}
	brief.Environment.Stack = []string{"Go"}
	brief.Scope.InScope = []string{"provider routing"}
	brief.Scope.OutOfScope = []string{"provider authentication"}
	brief.Quality.AcceptanceCriteria = []string{"request completes with an eligible provider"}
	brief.Quality.Testing = []string{"routing regression tests"}
	brief.Quality.Verification = []string{"go test ./..."}
	brief.Deliverable.ExpectedArtifacts = []string{"source changes", "tests"}
	brief.Decisions = []string{"route automatically across eligible providers"}

	refreshComposerBrief(&brief)
	count := 0
	for _, unknown := range brief.Unknowns {
		if unknown.Field == "interview.challenge" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one pressure-test question, got %d: %+v", count, brief.Unknowns)
	}
	if !brief.Interview.ChallengePending {
		t.Fatalf("expected pending pressure pass: %+v", brief.Interview)
	}

	refreshComposerBrief(&brief)
	count = 0
	for _, unknown := range brief.Unknowns {
		if unknown.Field == "interview.challenge" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("pressure pass must be idempotent, got %d", count)
	}
}

func TestComposerInterviewProfilesAreBounded(t *testing.T) {
	quickTarget, quickMax := composerInterviewProfile(ComposerInterviewQuick)
	standardTarget, standardMax := composerInterviewProfile(ComposerInterviewStandard)
	deepTarget, deepMax := composerInterviewProfile(ComposerInterviewDeep)

	if !(quickTarget > standardTarget && standardTarget > deepTarget) {
		t.Fatalf(
			"expected progressively stricter ambiguity targets: quick=%v standard=%v deep=%v",
			quickTarget,
			standardTarget,
			deepTarget,
		)
	}
	if quickMax >= standardMax || standardMax >= deepMax {
		t.Fatalf(
			"expected progressively larger question budgets: quick=%d standard=%d deep=%d",
			quickMax,
			standardMax,
			deepMax,
		)
	}
}

func TestComposerInterviewQuestionBudgetKeepsBlockingEscapeHatch(t *testing.T) {
	brief := LivingBrief{
		Interview: ComposerInterviewState{
			Mode:              ComposerInterviewQuick,
			QuestionsAnswered: 5,
			MaxQuestions:      5,
		},
		Unknowns: []PromptUnknown{
			{
				ID:            "recommended",
				Field:         "quality.testing",
				Status:        PromptUnknownOpen,
				Severity:      PromptUnknownRecommended,
				PriorityScore: 100,
			},
			{
				ID:            "blocking",
				Field:         "quality.acceptance_criteria",
				Status:        PromptUnknownOpen,
				Severity:      PromptUnknownBlocking,
				PriorityScore: 20,
			},
		},
	}
	next, ok := selectComposerNextUnknownForBrief(brief)
	if !ok || next.ID != "blocking" {
		t.Fatalf("question budget must suppress optional interview but keep blocking gap: %+v ok=%v", next, ok)
	}
}

func TestComposerInterviewModeIsAppliedBeforeInitialRefresh(t *testing.T) {
	brief := newComposerBriefWithInterviewMode(
		"Choose the best provider automatically",
		"Use automatic fallback with tests and verification.",
		ComposerInterviewQuick,
	)
	if brief.Interview.Mode != ComposerInterviewQuick {
		t.Fatalf("mode = %s, want QUICK", brief.Interview.Mode)
	}
	for _, unknown := range brief.Unknowns {
		if unknown.Field == "interview.challenge" {
			t.Fatalf("QUICK mode must not inherit a STANDARD pressure pass: %+v", unknown)
		}
	}
}

func TestComposerInterviewRefreshPreservesRealRoundCount(t *testing.T) {
	brief := newComposerBrief("Improve provider routing", "")
	brief.Interview.QuestionsAnswered = 3
	refreshComposerBrief(&brief)
	if brief.Interview.QuestionsAnswered != 3 {
		t.Fatalf("questions answered = %d, want 3", brief.Interview.QuestionsAnswered)
	}
}
