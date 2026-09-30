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

func TestIsSingleTurnDelegateRequestRejectsWritesAndDecomposition(t *testing.T) {
	if IsSingleTurnDelegateRequest("corrija o código", TaskRequirements{TaskKind: "coding"}) {
		t.Fatal("coding must stay on Direct/Mission path, not silent delegated writer")
	}
	if IsSingleTurnDelegateRequest("revise backend e frontend em paralelo", TaskRequirements{TaskKind: "review", RequiresDecomposition: true}) {
		t.Fatal("decomposed review must use Mission")
	}
	if IsSingleTurnDelegateRequest("use codex para refatorar a arquitetura", TaskRequirements{TaskKind: "architecture"}) {
		t.Fatal("write language must force the controlled Mission path even for an analysis-class task")
	}
	if IsSingleTurnDelegateRequest("use codex para atualizar o README", TaskRequirements{TaskKind: "documentation"}) {
		t.Fatal("documentation changes must not become a read-only delegated turn")
	}
}
