package nexus

import "testing"

func TestIsSingleTurnDelegateIntentAllowsBoundedAnalysisOnly(t *testing.T) {
	for _, req := range []TaskRequirements{
		{TaskKind: "review"},
		{TaskKind: "security"},
		{TaskKind: "investigation"},
		{TaskKind: "architecture"},
		{TaskKind: "documentation"},
	} {
		if !IsSingleTurnDelegateIntent(req) {
			t.Fatalf("task kind %q should allow bounded single-turn delegation", req.TaskKind)
		}
	}
}

func TestIsSingleTurnDelegateIntentRejectsWritesAndDecomposition(t *testing.T) {
	if IsSingleTurnDelegateIntent(TaskRequirements{TaskKind: "coding"}) {
		t.Fatal("coding must stay on Direct/Mission path, not silent delegated writer")
	}
	if IsSingleTurnDelegateIntent(TaskRequirements{TaskKind: "review", RequiresDecomposition: true}) {
		t.Fatal("decomposed review must use Mission")
	}
}
