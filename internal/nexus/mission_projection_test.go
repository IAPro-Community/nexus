package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestCanonicalMissionStagePipeline(t *testing.T) {
	cases := map[runner.State]string{
		runner.StateReady:      "READY",
		runner.StateAllocating: "ALLOCATE",
		runner.StateCompiling:  "COMPILE_PROMPT",
		runner.StateExecuting:  "EXECUTE",
		runner.StateTesting:    "TEST",
		runner.StateReviewing:  "REVIEW",
		runner.StateVerifying:  "VERIFY",
	}
	for state, want := range cases {
		if got := CanonicalMissionStage(state); got != want {
			t.Fatalf("state %q: got %q want %q", state, got, want)
		}
	}
	if got := CanonicalMissionStage(""); got != "UNKNOWN" {
		t.Fatalf("empty state must be UNKNOWN, got %q", got)
	}
	if got := CanonicalMissionStage("SOME_FUTURE_STATE"); got != "UNKNOWN" {
		t.Fatalf("unrecognized state must be UNKNOWN, got %q", got)
	}
}

func TestBuildMissionStageProjectionFromRunner(t *testing.T) {
	run := &runner.MissionRun{
		ID: "run-1", State: runner.StateExecuting, CurrentPkgIndex: 0,
		PackageRuns: []runner.PackageRun{{
			PackageID: "pkg-1", Title: "Build API", State: runner.StateExecuting,
			AssignedAgent: "agent-1", Provider: "codex", Profile: "main",
			Attempt: 2, RetryFrom: runner.StateAllocating, FailureKind: model.FailureRateLimit,
			Verifications: []runner.VerificationResult{
				{Command: "go test ./...", Passed: true},
				{Command: "npm test", Passed: false},
			},
		}},
	}
	proj := BuildMissionStageProjection(run)
	if proj.RunID != "run-1" || proj.CurrentStage != "EXECUTE" {
		t.Fatalf("stage projection wrong: %+v", proj)
	}
	if proj.AgentID != "agent-1" || proj.TaskID != "pkg-1" || proj.TaskTitle != "Build API" {
		t.Fatalf("agent/task projection wrong: %+v", proj)
	}
	if proj.Provider != "codex" || proj.Profile != "main" || proj.Resource != "codex:main" {
		t.Fatalf("resource projection wrong: %+v", proj)
	}
	if !proj.IsFallbackRetry || proj.Attempt != 2 || proj.RetryFrom != string(runner.StateAllocating) {
		t.Fatalf("fallback-retry projection wrong: %+v", proj)
	}
	if proj.FailureKind != string(model.FailureRateLimit) {
		t.Fatalf("failure kind missing: %+v", proj)
	}
	if proj.Verification.Count != 2 || proj.Verification.Passed != 1 || proj.Verification.Failed != 1 || !proj.Verification.HasEvidence {
		t.Fatalf("verification projection wrong: %+v", proj.Verification)
	}
}

func TestBuildMissionStageProjectionUnknownWhenMissing(t *testing.T) {
	if got := BuildMissionStageProjection(nil); got.CurrentStage != "UNKNOWN" {
		t.Fatalf("nil run must be UNKNOWN, got %+v", got)
	}
	if got := BuildMissionStageProjection(&runner.MissionRun{}); got.CurrentStage != "UNKNOWN" {
		t.Fatalf("empty run must be UNKNOWN, got %+v", got)
	}
	run := &runner.MissionRun{ID: "run-x", State: runner.StateExecuting, PackageRuns: []runner.PackageRun{{
		PackageID: "pkg-x", State: "",
	}}}
	if got := BuildMissionStageProjection(run); got.CurrentStage != "UNKNOWN" {
		t.Fatalf("unknown package state must be UNKNOWN, got %+v", got)
	} else if got.AgentID != "" || got.Resource != "" || got.IsFallbackRetry {
		t.Fatalf("unknown must not invent agent/resource/retry: %+v", got)
	} else if got.Verification.HasEvidence {
		t.Fatalf("unknown must not invent verification: %+v", got)
	}
}
