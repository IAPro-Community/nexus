package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/executil"
)

var ErrMissingContextCapsule = errors.New("flow step context capsule is missing")
var ErrEvidenceRepositoryUnavailable = errors.New("mission evidence repository is unavailable")

// ErrInfrastructureFailure marks failures that must not consume implementation
// retries (for example an unavailable SQLite evidence table).
var ErrInfrastructureFailure = errors.New("mission infrastructure failure")

const (
	maxCapsuleStrings = 64
	maxReceiptFiles   = 256
)

func capStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, min(limit, len(values)))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func gitText(ctx context.Context, workspace string, args ...string) string {
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	all := append([]string{"-C", workspace}, args...)
	out, err := executil.CommandContext(ctx, "git", all...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func mutationPaths(ctx context.Context, workspace string) []string {
	commands := [][]string{
		{"-C", workspace, "diff", "--name-only", "-z"},
		{"-C", workspace, "diff", "--cached", "--name-only", "-z"},
		{"-C", workspace, "ls-files", "--others", "--exclude-standard", "-z"},
	}
	seen := map[string]struct{}{}
	for _, args := range commands {
		out, err := executil.CommandContext(ctx, "git", args...).Output()
		if err != nil {
			continue
		}
		for _, raw := range bytes.Split(out, []byte{0}) {
			name := strings.TrimSpace(string(raw))
			if name != "" {
				seen[strings.ReplaceAll(name, "\\", "/")] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	if len(out) > maxReceiptFiles {
		out = out[:maxReceiptFiles]
	}
	return out
}

func workspaceMutationSnapshot(ctx context.Context, workspace string) map[string]string {
	paths := mutationPaths(ctx, workspace)
	if len(paths) == 0 {
		return map[string]string{}
	}
	root := gitText(ctx, workspace, "rev-parse", "--show-toplevel")
	if root == "" {
		root = workspace
	}
	out := make(map[string]string, len(paths))
	for _, rel := range paths {
		clean := filepath.Clean(rel)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, clean))
		if err != nil {
			out[rel] = "deleted-or-unreadable"
			continue
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:])
	}
	return out
}

func snapshotFingerprint(snapshot map[string]string) string {
	if len(snapshot) == 0 {
		return ""
	}
	keys := make([]string, 0, len(snapshot))
	for key := range snapshot {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, key := range keys {
		_, _ = h.Write([]byte(key))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(snapshot[key]))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func revisionChangedPaths(ctx context.Context, workspace, baseRevision, resultRevision string) []string {
	if strings.TrimSpace(workspace) == "" || strings.TrimSpace(baseRevision) == "" || strings.TrimSpace(resultRevision) == "" || baseRevision == resultRevision {
		return []string{}
	}
	out, err := executil.CommandContext(ctx, "git", "-C", workspace, "diff", "--name-only", "-z", baseRevision, resultRevision, "--").Output()
	if err != nil {
		return []string{}
	}
	paths := make([]string, 0)
	for _, raw := range bytes.Split(out, []byte{0}) {
		name := strings.TrimSpace(string(raw))
		if name == "" {
			continue
		}
		paths = append(paths, strings.ReplaceAll(name, "\\", "/"))
	}
	return capStrings(paths, maxReceiptFiles)
}

func changedSinceSnapshot(before, after map[string]string) []string {
	seen := map[string]struct{}{}
	for k := range before {
		seen[k] = struct{}{}
	}
	for k := range after {
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for key := range seen {
		if before[key] != after[key] {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	if len(out) > maxReceiptFiles {
		out = out[:maxReceiptFiles]
	}
	return out
}

func receiptChangedFiles(ctx context.Context, workspace, baseRevision, resultRevision string, before, after map[string]string) []string {
	if strings.TrimSpace(baseRevision) != "" && strings.TrimSpace(resultRevision) != "" && baseRevision != resultRevision {
		if paths := revisionChangedPaths(ctx, workspace, baseRevision, resultRevision); len(paths) > 0 {
			return capStrings(paths, maxReceiptFiles)
		}
	}
	return changedSinceSnapshot(before, after)
}

func durableContextRefs(workspace string) []string {
	refs := []string{}
	if info, err := os.Stat(filepath.Join(workspace, "AGENTS.md")); err == nil && !info.IsDir() {
		refs = append(refs, "AGENTS.md")
	}
	if info, err := os.Stat(filepath.Join(workspace, "DEV")); err == nil && info.IsDir() {
		refs = append(refs, "DEV/")
	}
	return refs
}

func capsuleConstraints(contract AutonomyContract) []string {
	out := []string{"Work only inside the approved Flow Step and acceptance criteria."}
	if contract.DisallowDestructiveGit {
		out = append(out, "Destructive git operations are prohibited.")
	}
	if !contract.AllowGitPush {
		out = append(out, "Git push is not authorized.")
	}
	if !contract.AllowDeploy {
		out = append(out, "Deploy/publish/release is not authorized.")
	}
	if len(contract.AllowedFilePatterns) > 0 {
		out = append(out, "Allowed file patterns: "+strings.Join(capStrings(contract.AllowedFilePatterns, maxCapsuleStrings), ", "))
	}
	return out
}

func evidenceRepository(repo RunRepository) (EvidenceRepository, error) {
	evidence, ok := repo.(EvidenceRepository)
	if !ok {
		return nil, ErrEvidenceRepositoryUnavailable
	}
	return evidence, nil
}

func packageByID(run *MissionRun, id string) *PackageRun {
	for i := range run.PackageRuns {
		if run.PackageRuns[i].PackageID == id {
			return &run.PackageRuns[i]
		}
	}
	return nil
}

func (r *MissionRunner) hydrateEvidence(ctx context.Context, run *MissionRun) {
	evidence, err := evidenceRepository(r.repo)
	if err != nil {
		return
	}
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.ContextCapsule == nil && !pkg.ContextRefreshRequired {
			if capsule, getErr := evidence.GetContextCapsule(ctx, run.ID, pkg.PackageID); getErr == nil {
				pkg.ContextCapsule = capsule
			}
		}
		if pkg.WorkReceipt == nil {
			if receipt, getErr := evidence.GetWorkReceipt(ctx, run.ID, pkg.PackageID); getErr == nil {
				pkg.WorkReceipt = receipt
				if receipt.Status == "VERIFIED" && pkg.State == StateReviewing {
					pkg.State = StateVerified
					finished := receipt.CompletedAt
					pkg.FinishedAt = &finished
				}
				if receipt.Status == "FAILED" && pkg.State == StateRemediating {
					pkg.State = StateFailed
				}
			}
		}
	}
}

func (r *MissionRunner) directDependencyReceipts(ctx context.Context, run *MissionRun, pkg *PackageRun) []WorkReceipt {
	evidence, _ := evidenceRepository(r.repo)
	out := []WorkReceipt{}
	for _, depID := range pkg.Dependencies {
		dep := packageByID(run, depID)
		if dep == nil {
			continue
		}
		receipt := dep.WorkReceipt
		if receipt == nil && evidence != nil {
			if stored, err := evidence.GetWorkReceipt(ctx, run.ID, depID); err == nil {
				receipt = stored
				dep.WorkReceipt = stored
			}
		}
		if receipt != nil {
			out = append(out, *receipt)
		}
	}
	return out
}

func (r *MissionRunner) ensureContextCapsule(ctx context.Context, run *MissionRun, pkg *PackageRun) error {
	if pkg.ContextCapsule != nil && !pkg.ContextRefreshRequired {
		return nil
	}
	if pkg.ContextRefreshRequired {
		pkg.ContextCapsule = nil
	}
	evidence, err := evidenceRepository(r.repo)
	if err != nil {
		return err
	}
	branch := gitText(ctx, pkg.Workspace, "branch", "--show-current")
	head := gitText(ctx, pkg.Workspace, "rev-parse", "HEAD")
	baseline := workspaceMutationSnapshot(ctx, pkg.Workspace)
	skillIDs := append([]string(nil), pkg.SkillIDs...)
	if len(skillIDs) == 0 {
		skillIDs = append(skillIDs, pkg.MaestroSkills...)
	}
	capsule := &ContextCapsule{ID: "capsule_" + ids.NewRuntimeID(), RunID: run.ID, ProjectID: run.ProjectID, FlowID: run.PlanID, FlowRevision: run.PlanRevision, Branch: branch, Head: head, DirtyFingerprint: snapshotFingerprint(baseline), HandoffJSON: pkg.HandoffJSON,
		Step:          ContextCapsuleStep{ID: pkg.PackageID, Title: pkg.Title, Goal: pkg.Goal, Role: pkg.Role, Dependencies: capStrings(pkg.Dependencies, maxCapsuleStrings), AssignmentStrategy: pkg.AssignmentStrategy, VerificationRequirements: capStrings(pkg.VerificationRequirements, maxCapsuleStrings)},
		RelevantPaths: capStrings(pkg.RelevantPaths, maxCapsuleStrings), DurableContextRefs: durableContextRefs(pkg.Workspace), DependencyReceipts: r.directDependencyReceipts(ctx, run, pkg), SkillIDs: capStrings(skillIDs, maxCapsuleStrings), MaestroSkills: capStrings(pkg.MaestroSkills, maxCapsuleStrings), AcceptanceCriteria: capStrings(pkg.AcceptanceCriteria, maxCapsuleStrings), Constraints: capsuleConstraints(run.Contract), BaselineWorkspaceSnapshot: baseline, CreatedAt: time.Now().UTC()}
	if err := evidence.SaveContextCapsule(ctx, capsule); err != nil {
		// Keep storage/SQL details out of the user-facing error while retaining
		// an errors.Is classification for the runner and telemetry.
		return fmt.Errorf("%w: SCHEMA_UNAVAILABLE", ErrInfrastructureFailure)
	}
	pkg.ContextCapsule = capsule
	pkg.ContextRefreshRequired = false
	return nil
}

func receiptCommands(results []VerificationResult) []string {
	out := []string{}
	for _, result := range results {
		if result.Command != "" {
			out = append(out, result.Command)
		}
	}
	return capStrings(out, maxCapsuleStrings)
}

func (r *MissionRunner) ensureWorkReceipt(ctx context.Context, run *MissionRun, pkg *PackageRun) error {
	if pkg.WorkReceipt != nil {
		return nil
	}
	if pkg.State != StateVerified && pkg.State != StateFailed {
		return nil
	}
	evidence, err := evidenceRepository(r.repo)
	if err != nil {
		return err
	}
	status := "FAILED"
	if pkg.State == StateVerified {
		status = "VERIFIED"
	}
	completed := time.Now().UTC()
	if pkg.FinishedAt != nil {
		completed = *pkg.FinishedAt
	}
	base := ""
	baseline := map[string]string{}
	if pkg.ContextCapsule != nil {
		base = pkg.ContextCapsule.Head
		baseline = pkg.ContextCapsule.BaselineWorkspaceSnapshot
	}
	after := workspaceMutationSnapshot(ctx, pkg.Workspace)
	resultRevision := gitText(ctx, pkg.Workspace, "rev-parse", "HEAD")
	changedFiles := receiptChangedFiles(ctx, pkg.Workspace, base, resultRevision, baseline, after)
	remaining := []string{}
	if status == "FAILED" && strings.TrimSpace(pkg.ErrorMessage) != "" {
		remaining = []string{strings.TrimSpace(pkg.ErrorMessage)}
	}
	summary := strings.TrimSpace(pkg.ResultSummary)
	if summary == "" {
		summary = fmt.Sprintf("%s ended in %s", pkg.Title, status)
	} else {
		summary = boundedCoordinationSummary(summary)
	}
	receipt := &WorkReceipt{ID: "receipt_" + ids.NewRuntimeID(), RunID: run.ID, StepID: pkg.PackageID, Status: status, Summary: summary, ChangedFiles: changedFiles, Commands: receiptCommands(pkg.Verifications), Tests: append([]VerificationResult(nil), pkg.Verifications...), Decisions: []string{}, Artifacts: []string{}, RemainingIssues: remaining, Verification: append([]VerificationResult(nil), pkg.Verifications...), HandoffJSON: pkg.HandoffJSON, AgentID: pkg.AssignedAgent, BaseRevision: base, ResultRevision: resultRevision, StartedAt: pkg.StartedAt, CompletedAt: completed}
	if err := evidence.SaveWorkReceipt(ctx, receipt); err != nil {
		return fmt.Errorf("persist work receipt: %w", err)
	}
	pkg.WorkReceipt = receipt
	return nil
}

func (r *MissionRunner) saveRun(ctx context.Context, run *MissionRun) error {
	refreshPhaseRuns(run)
	if err := validateMissionRunLifecycle(run); err != nil {
		return err
	}
	if run.watchFingerprint != "" {
		current := progressFingerprint(run)
		if current != run.watchFingerprint {
			run.LastProgressAt = time.Now().UTC()
			run.watchFingerprint = current
		}
	}
	for i := range run.PackageRuns {
		if err := r.ensureWorkReceipt(ctx, run, &run.PackageRuns[i]); err != nil {
			return err
		}
	}
	reconcileCoordinationRequests(run)
	return r.repo.SaveRun(ctx, run)
}

func validateMissionRunLifecycle(run *MissionRun) error {
	if run == nil {
		return fmt.Errorf("mission run is required")
	}
	if run.State != StateBlockedNeedsUser {
		return nil
	}
	if run.NeedsHuman == nil || run.NeedsHuman.ID == "" || run.NeedsHuman.Version < 1 || len(run.NeedsHuman.Options) == 0 {
		return fmt.Errorf("BLOCKED_NEEDS_USER requires an actionable versioned human intervention")
	}
	for _, option := range run.NeedsHuman.Options {
		if strings.TrimSpace(option.ID) == "" || strings.TrimSpace(string(option.Operation)) == "" {
			return fmt.Errorf("BLOCKED_NEEDS_USER intervention contains an invalid option")
		}
		switch option.Operation {
		case InterventionRetrySafePackage, InterventionReplanPackage, InterventionConfirmExternalOutcome, InterventionVerifyCompletedWithoutReplay:
		default:
			return fmt.Errorf("BLOCKED_NEEDS_USER intervention contains unsupported option %q", option.Operation)
		}
	}
	return nil
}

// RenderContextCapsule produces the bounded prompt section supplied to the
// provider for one Flow Step. It intentionally renders receipts/evidence only;
// there is no raw conversation/provider transcript input.
func RenderContextCapsule(capsule *ContextCapsule) string {
	if capsule == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Nexus Context Capsule\nFlow %s revision %d\nStep %s — %s\n", capsule.FlowID, capsule.FlowRevision, capsule.Step.ID, capsule.Step.Title)
	if capsule.Step.Goal != "" {
		fmt.Fprintf(&b, "Goal: %s\n", capsule.Step.Goal)
	}
	if capsule.Branch != "" || capsule.Head != "" {
		fmt.Fprintf(&b, "Source: branch=%s head=%s\n", capsule.Branch, capsule.Head)
	}
	if len(capsule.DurableContextRefs) > 0 {
		b.WriteString("Durable context refs:\n- " + strings.Join(capsule.DurableContextRefs, "\n- ") + "\n")
	}
	if len(capsule.RelevantPaths) > 0 {
		b.WriteString("Relevant paths:\n- " + strings.Join(capsule.RelevantPaths, "\n- ") + "\n")
	}
	if len(capsule.SkillIDs) > 0 {
		b.WriteString("Selected skills:\n- " + strings.Join(capsule.SkillIDs, "\n- ") + "\n")
	} else if len(capsule.MaestroSkills) > 0 {
		// Compatibility for capsules created before the generic SkillIDs field.
		b.WriteString("Selected skills:\n- " + strings.Join(capsule.MaestroSkills, "\n- ") + "\n")
	}
	if len(capsule.DependencyReceipts) > 0 {
		b.WriteString("Dependency Work Receipts:\n")
		for _, receipt := range capsule.DependencyReceipts {
			fmt.Fprintf(&b, "- %s [%s]: %s\n", receipt.StepID, receipt.Status, receipt.Summary)
			if len(receipt.ChangedFiles) > 0 {
				b.WriteString("  changed files: " + strings.Join(receipt.ChangedFiles, ", ") + "\n")
			}
			if len(receipt.Commands) > 0 {
				b.WriteString("  commands: " + strings.Join(receipt.Commands, "; ") + "\n")
			}
			if len(receipt.RemainingIssues) > 0 {
				b.WriteString("  remaining issues: " + strings.Join(receipt.RemainingIssues, "; ") + "\n")
			}
		}
	}
	if len(capsule.AcceptanceCriteria) > 0 {
		b.WriteString("Acceptance criteria:\n- " + strings.Join(capsule.AcceptanceCriteria, "\n- ") + "\n")
	}
	if len(capsule.Constraints) > 0 {
		b.WriteString("Constraints:\n- " + strings.Join(capsule.Constraints, "\n- ") + "\n")
	}
	return strings.TrimSpace(b.String())
}
