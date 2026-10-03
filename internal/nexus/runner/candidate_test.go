package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func verifiedCandidate(id string, index int, independence string, severity string, failedChecks int) PackageRun {
	now := time.Now().UTC()
	verifications := make([]VerificationResult, 0, failedChecks+1)
	for i := 0; i < failedChecks; i++ {
		verifications = append(verifications, VerificationResult{Command: "test", Passed: false, VerifiedAt: now})
	}
	verifications = append(verifications, VerificationResult{Command: "test", Passed: true, VerifiedAt: now})
	findings := []ReviewFinding{}
	if severity != "" {
		findings = append(findings, ReviewFinding{
			ID: "finding-" + id, Severity: severity, Description: "candidate finding", Status: FindingOpen,
		})
	}
	return PackageRun{
		PackageID:            id,
		Title:                id,
		State:                StateVerified,
		CandidateGroupID:     "candidate:root",
		CandidateIndex:       index,
		CandidateDisposition: CandidatePending,
		AssignedAgent:        "agent-" + id,
		Workspace:            "/tmp/" + id,
		WorkReceipt: &WorkReceipt{
			ID: "receipt-" + id, RunID: "run-candidate", StepID: id, Status: "VERIFIED",
			Summary: id + " verified", ChangedFiles: []string{"file.txt"}, StartedAt: now, CompletedAt: now,
		},
		Verdicts: []ReviewVerdict{{
			Approved: true, ReviewerAgentID: "reviewer-" + id, ReviewIndependence: independence,
			StructuredFindings: findings, ReviewedAt: now,
		}},
		Verifications: verifications,
		StartedAt:     now,
	}
}

func TestFinalizeCandidateCoordinatorSelectsLowestRiskIndependentCandidate(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, nil)
	run := &MissionRun{
		ID: "run-candidate", ProjectID: "project", PlanID: "plan", PlanRevision: 1,
		PackageRuns: []PackageRun{
			verifiedCandidate("root__candidate_1", 1, "DISTINCT_AGENT", "high", 0),
			verifiedCandidate("root__candidate_2", 2, "DISTINCT_AGENT", "low", 1),
			{
				PackageID: "root", Title: "Root", State: StateReady,
				CandidateGroupID: "candidate:root", CandidateCoordinator: true,
				CandidateDisposition: CandidateCoordinator, StartedAt: time.Now().UTC(),
			},
		},
	}
	coordinator := &run.PackageRuns[2]
	if err := r.finalizeCandidateCoordinator(context.Background(), run, coordinator); err != nil {
		t.Fatal(err)
	}
	if coordinator.State != StateVerified || coordinator.CandidateEvaluation == nil {
		t.Fatalf("coordinator not verified with evaluation: %+v", coordinator)
	}
	if got := coordinator.CandidateEvaluation.WinnerPackageID; got != "root__candidate_2" {
		t.Fatalf("winner=%q, want candidate 2", got)
	}
	if run.PackageRuns[0].CandidateDisposition != CandidateRejected {
		t.Fatalf("candidate 1 disposition=%s", run.PackageRuns[0].CandidateDisposition)
	}
	if run.PackageRuns[1].CandidateDisposition != CandidateSelected {
		t.Fatalf("candidate 2 disposition=%s", run.PackageRuns[1].CandidateDisposition)
	}
	if coordinator.WorkReceipt == nil || coordinator.WorkReceipt.StepID != "root" {
		t.Fatalf("coordinator receipt missing: %+v", coordinator.WorkReceipt)
	}
	if !strings.Contains(coordinator.WorkReceipt.Summary, "APPLY selected root__candidate_2") {
		t.Fatalf("coordinator receipt does not explain selection: %q", coordinator.WorkReceipt.Summary)
	}
}

func TestFinalizeCandidateCoordinatorRejectsSelfReviewOnly(t *testing.T) {
	repo := NewMemoryRunRepository()
	r := NewMissionRunner(repo, nil)
	run := &MissionRun{
		ID: "run-candidate", ProjectID: "project", PlanID: "plan", PlanRevision: 1,
		PackageRuns: []PackageRun{
			verifiedCandidate("root__candidate_1", 1, "DEGRADED_SELF_REVIEW", "", 0),
			verifiedCandidate("root__candidate_2", 2, "DEGRADED_SELF_REVIEW", "", 0),
			{
				PackageID: "root", Title: "Root", State: StateReady,
				CandidateGroupID: "candidate:root", CandidateCoordinator: true,
				CandidateDisposition: CandidateCoordinator, StartedAt: time.Now().UTC(),
			},
		},
	}
	err := r.finalizeCandidateCoordinator(context.Background(), run, &run.PackageRuns[2])
	if err == nil || !strings.Contains(err.Error(), "no independently approved candidate") {
		t.Fatalf("self-review-only candidate set must fail closed, got %v", err)
	}
}

func TestCandidateDeliveryEligible(t *testing.T) {
	normal := &PackageRun{PackageID: "normal"}
	selected := &PackageRun{PackageID: "selected", CandidateGroupID: "g", CandidateDisposition: CandidateSelected}
	rejected := &PackageRun{PackageID: "rejected", CandidateGroupID: "g", CandidateDisposition: CandidateRejected}
	coordinator := &PackageRun{PackageID: "root", CandidateGroupID: "g", CandidateCoordinator: true, CandidateDisposition: CandidateCoordinator}
	if !candidateDeliveryEligible(normal) || !candidateDeliveryEligible(selected) {
		t.Fatal("normal and selected candidate must participate in delivery")
	}
	if candidateDeliveryEligible(rejected) || candidateDeliveryEligible(coordinator) {
		t.Fatal("rejected candidate/coordinator must not participate in delivery")
	}
}
