package runner

import "testing"

func TestValidateCoordinationRequest(t *testing.T) {
	base := CoordinationRequest{
		ID:              "coord_1",
		RunID:           "run_1",
		SourcePackageID: "step_1",
		Operation:       CoordinationNeedInfo,
		Subject:         "refresh token persistence",
		Status:          CoordinationPending,
	}
	if err := ValidateCoordinationRequest(base); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	delegated := base
	delegated.Operation = CoordinationDelegate
	delegated.Objective = "inspect authentication persistence"
	delegated.PreferredRole = "backend-engineer"
	delegated.PreferredProvider = "codex"
	if err := ValidateCoordinationRequest(delegated); err != nil {
		t.Fatalf("valid delegation rejected: %v", err)
	}

	missingObjective := delegated
	missingObjective.Objective = ""
	if err := ValidateCoordinationRequest(missingObjective); err == nil {
		t.Fatal("DELEGATE without objective must fail")
	}

	unknown := base
	unknown.Operation = CoordinationOperation("CHAT")
	if err := ValidateCoordinationRequest(unknown); err == nil {
		t.Fatal("free-form/unknown coordination operation must fail closed")
	}
}
