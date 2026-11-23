package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// EvidenceBundle is the bounded, read-only proof projection for one Mission.
// It reuses the canonical run, ContextCapsule/WorkReceipt, routing and
// ValidationEvidenceStream stores; it does not introduce a second evidence
// ledger and it never includes raw provider transcripts.
type EvidenceBundle struct {
	Version             string                              `json:"version"`
	RunID               string                              `json:"run_id"`
	PlanID              string                              `json:"plan_id,omitempty"`
	PlanRevision        int                                 `json:"plan_revision,omitempty"`
	State               runner.State                        `json:"state"`
	GeneratedAt         time.Time                           `json:"generated_at"`
	CompletedVerified   bool                                `json:"completed_verified"`
	EvidenceComplete    bool                                `json:"evidence_complete"`
	MissingEvidence     []string                            `json:"missing_evidence,omitempty"`
	IntegratedWorkspace string                              `json:"integrated_workspace,omitempty"`
	DeliveryIntegration *runner.DeliveryIntegrationEvidence `json:"delivery_integration,omitempty"`
	Capsules            []runner.ContextCapsule             `json:"capsules"`
	Receipts            []runner.WorkReceipt                `json:"receipts"`
	TestMatrix          []WorktreeTestMatrix                `json:"test_matrix"`
	Routing             RoutingDecisionReport               `json:"routing"`
	Validation          ValidationEvidenceReport            `json:"validation"`
	GlobalVerifications []runner.VerificationResult         `json:"global_verifications,omitempty"`
}

// EvidenceBundle returns one trustworthy projection of the durable evidence
// already owned by the Mission subsystem. Missing proof is reported explicitly
// rather than inferred from terminal state.
func (s *RunApplicationService) EvidenceBundle(ctx context.Context, runID string) (EvidenceBundle, error) {
	if err := s.ready(ctx); err != nil {
		return EvidenceBundle{}, err
	}
	run, err := s.nexus.Runner().GetRun(ctx, runID)
	if err != nil {
		return EvidenceBundle{}, err
	}
	st, err := s.nexus.OpenProject()
	if err != nil {
		return EvidenceBundle{}, err
	}

	capsuleRecords, err := st.ListFlowContextCapsules(runID)
	if err != nil {
		return EvidenceBundle{}, err
	}
	capsules := make([]runner.ContextCapsule, 0, len(capsuleRecords))
	for _, record := range capsuleRecords {
		var capsule runner.ContextCapsule
		if err := json.Unmarshal([]byte(record.ContentJSON), &capsule); err != nil {
			return EvidenceBundle{}, fmt.Errorf("decode persisted context capsule %s: %w", record.ID, err)
		}
		capsules = append(capsules, capsule)
	}

	receiptRecords, err := st.ListFlowWorkReceipts(runID)
	if err != nil {
		return EvidenceBundle{}, err
	}
	receipts := make([]runner.WorkReceipt, 0, len(receiptRecords))
	for _, record := range receiptRecords {
		var receipt runner.WorkReceipt
		if err := json.Unmarshal([]byte(record.ContentJSON), &receipt); err != nil {
			return EvidenceBundle{}, fmt.Errorf("decode persisted work receipt %s: %w", record.ID, err)
		}
		receipts = append(receipts, receipt)
	}

	routing, err := s.Routing(ctx, runID)
	if err != nil {
		return EvidenceBundle{}, fmt.Errorf("load routing evidence: %w", err)
	}
	validation, err := s.ValidationEvidence(ctx, runID)
	if err != nil {
		return EvidenceBundle{}, fmt.Errorf("load validation evidence: %w", err)
	}

	bundle := EvidenceBundle{
		Version:             "1",
		RunID:               run.ID,
		PlanID:              run.PlanID,
		PlanRevision:        run.PlanRevision,
		State:               run.State,
		GeneratedAt:         time.Now().UTC(),
		CompletedVerified:   run.State == runner.StateCompletedVerified,
		IntegratedWorkspace: run.IntegratedWorkspace,
		DeliveryIntegration: run.DeliveryIntegration,
		Capsules:            capsules,
		Receipts:            receipts,
		Routing:             routing,
		Validation:          validation,
		GlobalVerifications: append([]runner.VerificationResult(nil), run.GlobalVerifications...),
		MissingEvidence:     []string{},
	}

	receiptByStep := make(map[string]runner.WorkReceipt, len(receipts))
	for _, receipt := range receipts {
		receiptByStep[receipt.StepID] = receipt
	}
	for _, pkg := range run.PackageRuns {
		receipt, ok := receiptByStep[pkg.PackageID]
		if !ok {
			bundle.MissingEvidence = append(bundle.MissingEvidence, "work_receipt:"+pkg.PackageID)
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(receipt.Status), "VERIFIED") {
			bundle.MissingEvidence = append(bundle.MissingEvidence, "verified_work_receipt:"+pkg.PackageID)
		}
	}
	if !bundle.CompletedVerified {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "mission_completed_verified")
	}
	if run.NeedsHuman != nil && !run.NeedsHuman.Resolved {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "unresolved_human_intervention")
	}
	if !validation.ChainVerified {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "validation_chain_verified")
	}
	if len(validation.Entries) == 0 {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "validation_entries")
	} else {
		hasPass := false
		for _, entry := range validation.Entries {
			switch entry.Outcome {
			case store.EvidencePass:
				hasPass = true
			case store.EvidenceFail:
				bundle.MissingEvidence = append(bundle.MissingEvidence, "validation_failure")
			}
		}
		if !hasPass {
			bundle.MissingEvidence = append(bundle.MissingEvidence, "validation_pass")
		}
	}
	if len(run.GlobalVerifications) == 0 {
		bundle.MissingEvidence = append(bundle.MissingEvidence, "global_verification")
	} else {
		for _, result := range run.GlobalVerifications {
			if !result.Passed {
				bundle.MissingEvidence = append(bundle.MissingEvidence, "global_verification_passed")
				break
			}
		}
	}

	bundle.MissingEvidence = uniqueSortedStrings(bundle.MissingEvidence)
	bundle.EvidenceComplete = len(bundle.MissingEvidence) == 0
	bundle.TestMatrix = buildTestMatrix(run, receipts, capsules)
	return bundle, nil
}

// WorktreeTestResult is one real command result projected for a worktree step.
type WorktreeTestResult struct {
	Command    string    `json:"command"`
	ExitCode   int       `json:"exit_code"`
	Passed     bool      `json:"passed"`
	SHA        string    `json:"sha,omitempty"`
	VerifiedAt time.Time `json:"verified_at"`
	Logs       string    `json:"logs,omitempty"`
	Provenance string    `json:"provenance,omitempty"`
}

// WorktreeTestMatrix projects the real test commands recorded for one
// worktree step. A result is only PASS when it carries command execution
// provenance and a zero exit code; declarations without provenance never
// become PASS.
type WorktreeTestMatrix struct {
	StepID          string               `json:"step_id"`
	AgentID         string               `json:"agent_id,omitempty"`
	Branch          string               `json:"branch,omitempty"`
	Workspace       string               `json:"workspace,omitempty"`
	Results         []WorktreeTestResult `json:"results"`
	MissingEvidence []string             `json:"missing_evidence,omitempty"`
}

func buildTestMatrix(run *runner.MissionRun, receipts []runner.WorkReceipt, capsules []runner.ContextCapsule) []WorktreeTestMatrix {
	workspaceByStep := make(map[string]string, len(run.PackageRuns))
	for _, pkg := range run.PackageRuns {
		workspaceByStep[pkg.PackageID] = pkg.Workspace
	}
	branchByStep := make(map[string]string, len(capsules))
	shaByStep := make(map[string]string, len(capsules))
	for _, capsule := range capsules {
		stepID := capsule.Step.ID
		branchByStep[stepID] = capsule.Branch
		if capsule.Head != "" {
			shaByStep[stepID] = capsule.Head
		}
	}

	matrix := make([]WorktreeTestMatrix, 0, len(receipts))
	for _, receipt := range receipts {
		sha := shaByStep[receipt.StepID]
		if sha == "" {
			sha = strings.TrimSpace(receipt.ResultRevision)
		}
		if sha == "" {
			sha = strings.TrimSpace(receipt.BaseRevision)
		}
		entry := WorktreeTestMatrix{
			StepID:          receipt.StepID,
			AgentID:         receipt.AgentID,
			Branch:          branchByStep[receipt.StepID],
			Workspace:       workspaceByStep[receipt.StepID],
			Results:         []WorktreeTestResult{},
			MissingEvidence: []string{},
		}
		commands := receipt.Tests
		if len(commands) == 0 {
			commands = receipt.Verification
		}
		if len(commands) == 0 {
			entry.MissingEvidence = append(entry.MissingEvidence, "test_commands")
		}
		for _, command := range commands {
			result := WorktreeTestResult{
				Command:    command.Command,
				ExitCode:   command.ExitCode,
				Passed:     command.Source == runner.VerificationSourceCommandExecution && command.ExitCode == 0,
				SHA:        sha,
				VerifiedAt: command.VerifiedAt,
				Logs:       command.OutputSnippet,
				Provenance: command.Source,
			}
			if command.Source != runner.VerificationSourceCommandExecution {
				entry.MissingEvidence = append(entry.MissingEvidence, "command_execution_provenance")
			}
			if command.ExitCode < 0 {
				entry.MissingEvidence = append(entry.MissingEvidence, "exit_code")
			}
			if command.VerifiedAt.IsZero() {
				entry.MissingEvidence = append(entry.MissingEvidence, "verified_at")
			}
			if strings.TrimSpace(command.OutputSnippet) == "" {
				entry.MissingEvidence = append(entry.MissingEvidence, "logs")
			}
			entry.Results = append(entry.Results, result)
		}
		if sha == "" {
			entry.MissingEvidence = append(entry.MissingEvidence, "sha")
		}
		entry.MissingEvidence = uniqueSortedStrings(entry.MissingEvidence)
		matrix = append(matrix, entry)
	}
	return matrix
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
