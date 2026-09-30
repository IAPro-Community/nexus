package nexus

import "testing"

func TestIsSingleTurnDelegateRequestAllowsBoundedAnalysisOnly(t *testing.T) {
	cases := []struct {
		prompt string
		req    TaskRequirements
	}{
		{prompt: "revise isto", req: TaskRequirements{TaskKind: "review"}},
		{prompt: "faça uma auditoria de segurança", req: TaskRequirements{TaskKind: "security"}},
		{prompt: "investigue a causa raiz", req: TaskRequirements{TaskKind: "investigation"}},
		{prompt: "avalie a arquitetura e os trade-offs", req: TaskRequirements{TaskKind: "architecture"}},
	}
	for _, tc := range cases {
		if !IsSingleTurnDelegateRequest(tc.prompt, tc.req) {
			t.Fatalf("prompt %q task kind %q should allow bounded single-turn delegation", tc.prompt, tc.req.TaskKind)
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
