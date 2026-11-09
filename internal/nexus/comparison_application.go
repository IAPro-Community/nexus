package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const comparisonDiffLimit = 16 * 1024

// ComparisonCheck is a bounded test/verification result without raw output.
type ComparisonCheck struct {
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
}

type ComparisonRunEvidence struct {
	RunID              string                    `json:"run_id"`
	State              runner.State              `json:"state"`
	BaseRevision       string                    `json:"base_revision"`
	ChangedFiles       []string                  `json:"changed_files"`
	Diff               string                    `json:"diff"`
	DiffTruncated      bool                      `json:"diff_truncated"`
	Checks             []ComparisonCheck         `json:"checks"`
	Cost               store.CostAnalyticsTotals `json:"cost"`
	CostEvidenceStatus string                    `json:"cost_evidence_status"`
}

type ComparisonLabView struct {
	store.ComparisonLabRecord
	RunA ComparisonRunEvidence `json:"run_a"`
	RunB ComparisonRunEvidence `json:"run_b"`
}

// ComparisonApplicationService compares two completed MissionRuns from the
// same frozen plan revision and records a human decision without applying it.
type ComparisonApplicationService struct{ nexus *Nexus }

func NewComparisonApplicationService(n *Nexus) *ComparisonApplicationService {
	return &ComparisonApplicationService{nexus: n}
}

func (s *ComparisonApplicationService) ready(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.nexus == nil || s.nexus.Store() == nil {
		return nil, fmt.Errorf("comparison service unavailable")
	}
	return s.nexus.Store(), nil
}

func (s *ComparisonApplicationService) Create(ctx context.Context, projectID, runAID, runBID string) (*ComparisonLabView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	runA, err := s.nexus.Runner().GetRun(ctx, strings.TrimSpace(runAID))
	if err != nil {
		return nil, fmt.Errorf("load first comparison run: %w", err)
	}
	runB, err := s.nexus.Runner().GetRun(ctx, strings.TrimSpace(runBID))
	if err != nil {
		return nil, fmt.Errorf("load second comparison run: %w", err)
	}
	if runA.ID == runB.ID || runA.ProjectID != projectID || runB.ProjectID != projectID {
		return nil, fmt.Errorf("comparison runs must be distinct and belong to the selected project")
	}
	if runA.PlanID != runB.PlanID || runA.PlanRevision != runB.PlanRevision {
		return nil, fmt.Errorf("comparison runs must use the same frozen plan revision")
	}
	if !isTerminalComparisonRun(runA.State) || !isTerminalComparisonRun(runB.State) {
		return nil, fmt.Errorf("both comparison runs must be finished")
	}
	if runA.DeliveryIntegration == nil || runB.DeliveryIntegration == nil ||
		runA.DeliveryIntegration.BaseRevision == "" ||
		runA.DeliveryIntegration.BaseRevision != runB.DeliveryIntegration.BaseRevision {
		return nil, fmt.Errorf("both runs require delivery evidence from the same base revision")
	}
	if runA.IntegratedWorkspace == "" || runB.IntegratedWorkspace == "" || runA.IntegratedWorkspace == runB.IntegratedWorkspace {
		return nil, fmt.Errorf("comparison requires two distinct isolated integration workspaces")
	}
	if err := validateComparisonWorkspace(projectID, runA.ID, runA.IntegratedWorkspace); err != nil {
		return nil, fmt.Errorf("validate first comparison workspace: %w", err)
	}
	if err := validateComparisonWorkspace(projectID, runB.ID, runB.IntegratedWorkspace); err != nil {
		return nil, fmt.Errorf("validate second comparison workspace: %w", err)
	}
	evidenceA, err := buildComparisonRunEvidence(ctx, st, runA)
	if err != nil {
		return nil, err
	}
	evidenceB, err := buildComparisonRunEvidence(ctx, st, runB)
	if err != nil {
		return nil, err
	}
	evidenceJSON, err := json.Marshal(struct {
		RunA ComparisonRunEvidence `json:"run_a"`
		RunB ComparisonRunEvidence `json:"run_b"`
	}{RunA: evidenceA, RunB: evidenceB})
	if err != nil {
		return nil, err
	}
	record, err := st.CreateComparisonLab(store.ComparisonLabRecord{
		ProjectID: projectID, PlanID: runA.PlanID, PlanRevision: runA.PlanRevision,
		RunAID: runA.ID, RunBID: runB.ID, EvidenceJSON: string(evidenceJSON),
	})
	if err != nil {
		return nil, err
	}
	return comparisonViewFromSnapshot(record)
}

func (s *ComparisonApplicationService) List(ctx context.Context, projectID string) ([]ComparisonLabView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	records, err := st.ListComparisonLabs(projectID)
	if err != nil {
		return nil, err
	}
	views := make([]ComparisonLabView, 0, len(records))
	for i := range records {
		view, buildErr := s.Get(ctx, records[i].ID)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, *view)
	}
	return views, nil
}

func (s *ComparisonApplicationService) Get(ctx context.Context, id string) (*ComparisonLabView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	record, err := st.GetComparisonLab(strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return comparisonViewFromSnapshot(record)
}

func (s *ComparisonApplicationService) Decide(ctx context.Context, id, decision, reason string) (*ComparisonLabView, error) {
	st, err := s.ready(ctx)
	if err != nil {
		return nil, err
	}
	record, err := st.DecideComparisonLab(id, decision, reason)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, record.ID)
}

func comparisonViewFromSnapshot(record *store.ComparisonLabRecord) (*ComparisonLabView, error) {
	var evidence struct {
		RunA ComparisonRunEvidence `json:"run_a"`
		RunB ComparisonRunEvidence `json:"run_b"`
	}
	if err := json.Unmarshal([]byte(record.EvidenceJSON), &evidence); err != nil {
		return nil, fmt.Errorf("decode comparison evidence snapshot: %w", err)
	}
	if evidence.RunA.RunID != record.RunAID || evidence.RunB.RunID != record.RunBID {
		return nil, fmt.Errorf("comparison evidence does not match its run identities")
	}
	return &ComparisonLabView{ComparisonLabRecord: *record, RunA: evidence.RunA, RunB: evidence.RunB}, nil
}

func buildComparisonRunEvidence(ctx context.Context, st *store.Store, run *runner.MissionRun) (ComparisonRunEvidence, error) {
	evidence := ComparisonRunEvidence{
		RunID: run.ID, State: run.State, Checks: make([]ComparisonCheck, 0),
		ChangedFiles: make([]string, 0), CostEvidenceStatus: "UNKNOWN",
	}
	if run.DeliveryIntegration != nil {
		evidence.BaseRevision = run.DeliveryIntegration.BaseRevision
		evidence.ChangedFiles = append(evidence.ChangedFiles, run.DeliveryIntegration.ChangedFiles...)
		sort.Strings(evidence.ChangedFiles)
		if run.IntegratedWorkspace != "" && evidence.BaseRevision != "" {
			var diff strings.Builder
			for _, file := range evidence.ChangedFiles {
				part, diffErr := comparisonFileDiff(ctx, run.IntegratedWorkspace, evidence.BaseRevision, file)
				if diffErr != nil {
					return ComparisonRunEvidence{}, fmt.Errorf("collect diff for %s: %w", file, diffErr)
				}
				remaining := comparisonDiffLimit - diff.Len()
				if remaining <= 0 {
					evidence.DiffTruncated = true
					break
				}
				if len(part) > remaining {
					part = part[:remaining]
					evidence.DiffTruncated = true
				}
				diff.Write(part)
				if evidence.DiffTruncated {
					break
				}
			}
			evidence.Diff = diff.String()
		}
	}
	for _, pkg := range run.PackageRuns {
		for _, check := range pkg.Verifications {
			evidence.Checks = append(evidence.Checks, ComparisonCheck{
				Name: pkg.Title + ": " + check.Command, Passed: check.Passed,
				ExitCode: check.ExitCode, DurationMS: check.DurationMs,
			})
		}
	}
	for _, check := range run.GlobalVerifications {
		evidence.Checks = append(evidence.Checks, ComparisonCheck{
			Name: "global: " + check.Command, Passed: check.Passed,
			ExitCode: check.ExitCode, DurationMS: check.DurationMs,
		})
	}
	if run.ProjectID != "" {
		cost, err := st.SummarizeCostEvidence(store.CostObservationFilter{ProjectID: run.ProjectID, MissionRunID: run.ID})
		if err != nil {
			return ComparisonRunEvidence{}, err
		}
		evidence.Cost = cost
		if cost.ObservationCount > 0 {
			evidence.CostEvidenceStatus = "PARTIAL"
			if cost.UnknownCostObservations == 0 {
				evidence.CostEvidenceStatus = "COMPLETE"
			}
		}
	}
	return evidence, nil
}

func comparisonFileDiff(ctx context.Context, workspace, baseRevision, file string) ([]byte, error) {
	if file == "" || filepath.IsAbs(file) || filepath.Clean(file) == ".." || strings.HasPrefix(filepath.Clean(file), ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid comparison path")
	}
	tracked, err := gitOutputBytes(ctx, workspace, "ls-tree", "-r", "--name-only", baseRevision, "--", file)
	if err == nil && strings.TrimSpace(string(tracked)) != "" {
		return gitOutputBytes(ctx, workspace, "diff", "--no-color", "--no-ext-diff", "--no-renames", "--unified=2", baseRevision, "--", file)
	}
	cmd := exec.CommandContext(ctx, "git", "-C", workspace, "diff", "--no-index", "--no-color", "--no-ext-diff", "--unified=2", "--", os.DevNull, file)
	output, diffErr := cmd.Output()
	if diffErr == nil {
		return output, nil
	}
	var exitErr *exec.ExitError
	if errors.As(diffErr, &exitErr) && exitErr.ExitCode() == 1 {
		return output, nil
	}
	return nil, diffErr
}

func validateComparisonWorkspace(projectID, runID, workspace string) error {
	if !safeComparisonPathSegment(projectID) || !safeComparisonPathSegment(runID) {
		return errors.New("project and run identifiers must be path-safe")
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}
	managedRoot, err := filepath.EvalSymlinks(filepath.Join(dataDir, "integrations"))
	if err != nil {
		return err
	}
	expected := filepath.Join(managedRoot, projectID, runID)
	expectedAbs, err := filepath.Abs(expected)
	if err != nil {
		return err
	}
	actualAbs, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	actual, err := filepath.EvalSymlinks(actualAbs)
	if err != nil {
		return err
	}
	want, err := filepath.EvalSymlinks(expectedAbs)
	if err != nil {
		return err
	}
	if filepath.Clean(actual) != filepath.Clean(want) {
		return errors.New("workspace is outside the Nexus-managed integration path")
	}
	relative, err := filepath.Rel(managedRoot, actual)
	if err != nil || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("integration workspace resolves outside the managed root")
	}
	info, err := os.Stat(actual)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("integration workspace is not a directory")
	}
	return nil
}

func safeComparisonPathSegment(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value &&
		!strings.ContainsAny(value, `/\`)
}

func isTerminalComparisonRun(state runner.State) bool {
	switch state {
	case runner.StateCompletedVerified, runner.StateFailed, runner.StateFailedNoProgress,
		runner.StateFailedBudgetExceeded, runner.StateFailedVerification, runner.StateCanceledByUser:
		return true
	default:
		return false
	}
}
