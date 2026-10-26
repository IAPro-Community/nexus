package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

var ErrDispatchOutcomeUnknown = errors.New("provider dispatch outcome is unknown")
var ErrInterventionAlreadyResolved = errors.New("INTERVENTION_ALREADY_RESOLVED")
var ErrStaleIntervention = errors.New("STALE_INTERVENTION")
var ErrInvalidInterventionOption = errors.New("INVALID_INTERVENTION_OPTION")
var ErrInterventionPolicyDenied = errors.New("INTERVENTION_POLICY_DENIED")
var ErrDispatchNotSent = errors.New("external dispatch was not sent")

type dispatchOutcomeError struct {
	outcome DispatchState
	err     error
}

func (e *dispatchOutcomeError) Error() string { return e.err.Error() }
func (e *dispatchOutcomeError) Unwrap() error { return e.err }

// MarkDispatchNotSent lets an executor prove that the provider boundary was
// never crossed. All other execution errors fail closed as unknown.
func MarkDispatchNotSent(err error) error {
	if err == nil {
		return ErrDispatchNotSent
	}
	return &dispatchOutcomeError{outcome: DispatchFailedBeforeDispatch, err: fmt.Errorf("%w: %v", ErrDispatchNotSent, err)}
}

func dispatchOutcome(err error) DispatchState {
	var classified *dispatchOutcomeError
	if errors.As(err, &classified) && classified.outcome == DispatchFailedBeforeDispatch {
		return DispatchFailedBeforeDispatch
	}
	return DispatchUnknownExternalOutcome
}

func InterventionIdempotencyKey(runID, interventionID string, version int, optionID string) string {
	input := fmt.Sprintf("%s\x00%s\x00%d\x00%s", runID, interventionID, version, optionID)
	sum := sha256.Sum256([]byte(input))
	return "human-resolution-" + hex.EncodeToString(sum[:])
}

// MissionRunner is a deterministic, durable state machine. Provider-specific
// work is delegated to PackageExecutor and every transition is persisted.
type MissionRunner struct {
	repo     RunRepository
	executor PackageExecutor
	retry    *RetryController
	verifier *VerificationEngine
	owner    string
	leaseTTL time.Duration
}

func NewMissionRunner(repo RunRepository, executor PackageExecutor) *MissionRunner {
	if repo == nil {
		repo = NewMemoryRunRepository()
	}
	return &MissionRunner{
		repo: repo, executor: executor, retry: NewRetryController(), verifier: NewVerificationEngine(),
		owner: "runner_" + ids.NewRuntimeID(), leaseTTL: 30 * time.Second,
	}
}

func (r *MissionRunner) StartMissionRun(ctx context.Context, plan PlanSpec, workspace string, contract AutonomyContract, defaultAgentID string) (*MissionRun, error) {
	if r.executor == nil {
		return nil, fmt.Errorf("mission runner requires a real package executor")
	}
	if plan.ID == "" || plan.ProjectID == "" {
		return nil, fmt.Errorf("plan id and project id are required")
	}
	if len(plan.Packages) == 0 {
		return nil, fmt.Errorf("plan %s contains no work packages to execute", plan.ID)
	}
	if contract.MaxRetries <= 0 {
		contract.MaxRetries = 1
	}
	if contract.MaxTotalIterations <= 0 {
		contract.MaxTotalIterations = 1
	}
	if contract.MaxNoProgress <= 0 {
		contract.MaxNoProgress = 2
	}

	now := time.Now().UTC()
	leadAgentID := plan.LeadAgentID
	if leadAgentID == "" {
		leadAgentID = defaultAgentID
	}
	run := &MissionRun{ID: "run_" + ids.NewRuntimeID(), PlanID: plan.ID, PlanRevision: plan.Revision, ExecutionSnapshotID: plan.ExecutionSnapshotID, ProjectID: plan.ProjectID, LeadAgentID: leadAgentID,
		Workspace: workspace, State: StateExecuting, Contract: contract, Autonomous: plan.Autonomous, StartedAt: now, LastProgressAt: now, UpdatedAt: now}
	for _, spec := range plan.Packages {
		state := StatePending
		if len(spec.Dependencies) == 0 {
			state = StateReady
		}
		assignedAgent := spec.AgentAllocation
		// Legacy packages without an explicit Flow assignment retain the old
		// default-Agent fallback. CREATE/AUTO stay unassigned until allocation so
		// explicit approval cannot accidentally reuse one Agent/worktree.
		if assignedAgent == "" && spec.AssignmentStrategy == "" && spec.ParallelGroup == "" {
			assignedAgent = defaultAgentID
		}
		desiredProvider, desiredProfile := spec.DesiredProvider, spec.DesiredProfile
		if desiredProvider == "" {
			desiredProvider = spec.Provider
		}
		if desiredProfile == "" {
			desiredProfile = spec.Profile
		}
		skillIDs := append([]string(nil), spec.SkillIDs...)
		if len(skillIDs) == 0 {
			skillIDs = append(skillIDs, spec.MaestroSkills...)
		}
		run.PackageRuns = append(run.PackageRuns, PackageRun{
			ID: "pkgrun_" + ids.NewRuntimeID(), PackageID: spec.ID, PhaseID: spec.PhaseID, PhaseType: spec.PhaseType, Title: spec.Title, Goal: spec.Goal,
			Priority: spec.Priority, Role: spec.Role, ReviewRequired: reviewRequiredForPackage(spec), TaskRequirements: spec.TaskRequirements, Dependencies: append([]string(nil), spec.Dependencies...), ParallelGroup: spec.ParallelGroup,
			AssignmentStrategy: spec.AssignmentStrategy, ResourcePolicy: spec.ResourcePolicy, Provider: spec.Provider, Profile: spec.Profile,
			DesiredProvider: desiredProvider, DesiredProfile: desiredProfile,
			SkillIDs: skillIDs, MaestroSkills: append([]string(nil), spec.MaestroSkills...), RelevantPaths: append([]string(nil), spec.RelevantPaths...),
			AcceptanceCriteria: append([]string(nil), spec.AcceptanceCriteria...), VerificationRequirements: append([]string(nil), spec.VerificationRequirements...), EntryCriteria: append([]string(nil), spec.EntryCriteria...), ExitCriteria: append([]string(nil), spec.ExitCriteria...), RequiredEvidence: append([]string(nil), spec.RequiredEvidence...), HandoffJSON: spec.HandoffJSON, State: state, Attempt: 1,
			AssignedAgent: assignedAgent, RoutingDecisionJSON: spec.RoutingDecisionJSON, StartedAt: now,
		})
	}
	if err := validateDependencyGraph(run.PackageRuns); err != nil {
		return nil, err
	}
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (r *MissionRunner) GetRun(ctx context.Context, id string) (*MissionRun, error) {
	return r.repo.GetRun(ctx, id)
}
func (r *MissionRunner) ListRuns(ctx context.Context) ([]*MissionRun, error) {
	return r.repo.ListRuns(ctx)
}

func (r *MissionRunner) ExecuteNextStep(ctx context.Context, runID string) (*MissionRun, bool, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, false, err
	}
	r.hydrateEvidence(ctx, run)
	leaseToken := run.LeaseToken
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, leaseToken) }()

	// Keep fencing ownership alive while provider/test/review operations may run
	// for minutes. If renewal fails, cancel the in-flight operation and refuse
	// to persist stale state.
	leaseCtx, leaseCancel := context.WithCancel(ctx)
	defer leaseCancel()
	heartbeatErr := make(chan error, 1)
	heartbeatDone := make(chan struct{})
	heartbeatEvery := r.leaseTTL / 3
	if heartbeatEvery <= 0 {
		heartbeatEvery = 10 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(heartbeatEvery)
		defer ticker.Stop()
		defer close(heartbeatDone)
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticker.C:
				if renewErr := r.repo.RenewLease(context.Background(), runID, r.owner, leaseToken, r.leaseTTL); renewErr != nil {
					select {
					case heartbeatErr <- renewErr:
					default:
					}
					leaseCancel()
					return
				}
			}
		}
	}()
	defer func() { leaseCancel(); <-heartbeatDone }()

	if isTerminalRunState(run.State) {
		return run, run.State == StateCompletedVerified, nil
	}
	if run.State == StatePaused {
		return run, false, fmt.Errorf("mission run is paused: %s", run.PausedReason)
	}
	now := time.Now().UTC()
	if ApplyProgressWatchdog(run, now) {
		if err := r.saveRun(ctx, run); err != nil {
			return run, false, err
		}
		return run, false, fmt.Errorf("mission stalled: no durable progress within stall timeout")
	}
	run.watchFingerprint = progressFingerprint(run)
	if run.ResumeRequest != nil && run.ResumeRequest.Status == "PENDING" {
		now := time.Now().UTC()
		run.ResumeRequest.Status = "STARTED"
		run.ResumeRequest.StartedAt = &now
		run.UpdatedAt = now
		if err := r.saveRun(ctx, run); err != nil {
			return run, false, fmt.Errorf("claim durable mission resume: %w", err)
		}
	}

	run.TotalIterations++
	if run.TotalIterations > run.Contract.MaxTotalIterations {
		run.State = StateFailedBudgetExceeded
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, fmt.Errorf("mission iteration budget exceeded (%d/%d)", run.TotalIterations, run.Contract.MaxTotalIterations)
	}
	refreshDependencyStates(run)
	if allPackagesVerified(run) {
		return r.verifyGlobalDefinition(ctx, leaseCtx, run)
	}

	pkg := nextPackage(run)
	if pkg == nil {
		run.State = StateFailedNoProgress
		run.PausedReason = "No package can progress safely; dependency graph has no actionable continuation."
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, fmt.Errorf("no runnable package; dependency graph is blocked")
	}
	run.CurrentPkgIndex = packageIndex(run, pkg.PackageID)
	if pkg.State == StateExecuting && pkg.DispatchState == DispatchCompleted {
		if pkg.AssignedRuntime == "" {
			pkg.DispatchState = DispatchUnknownExternalOutcome
			pkg.ErrorMessage = fmt.Sprintf("package %s has completed dispatch %s without runtime evidence; refusing duplicate provider execution", pkg.PackageID, pkg.DispatchID)
			r.blockNeedsHuman(run, pkg, "DISPATCH_OUTCOME_UNKNOWN", pkg.ErrorMessage, "Reconcile the provider outcome before continuing.", []string{"Inspect provider runtime"})
		} else {
			// The provider operation is already known to have completed. Resume at
			// testing/review; never enter executeOne for a completed dispatch.
			pkg.State = StateTesting
		}
		run.UpdatedAt = time.Now().UTC()
		if err := r.saveRun(ctx, run); err != nil {
			return run, false, err
		}
		if run.State == StateBlockedNeedsUser {
			return run, false, ErrDispatchOutcomeUnknown
		}
		return run, false, nil
	}

	opCtx := leaseCtx
	cancelOperation := func() {}
	if run.Contract.PackageTimeoutSeconds > 0 {
		opCtx, cancelOperation = context.WithTimeout(leaseCtx, time.Duration(run.Contract.PackageTimeoutSeconds)*time.Second)
	}
	opCtx, cancelStallWatch := armStallWatchdog(opCtx, run)
	defer cancelStallWatch()
	defer cancelOperation()

	switch pkg.State {
	case StateReady:
		pkg.State = StateAllocating
	case StateAllocating:
		allocation, allocErr := r.executor.Allocate(opCtx, run, pkg)
		if allocErr != nil {
			if stalled, stallErr := r.finalizeStallIfNeeded(ctx, run, allocErr); stalled {
				return run, false, stallErr
			}
			return r.packageFailureFrom(ctx, run, pkg, StateAllocating, fmt.Errorf("allocate package: %w", allocErr))
		}
		if allocation.AgentID == "" || allocation.Workspace == "" {
			return r.packageFailureFrom(ctx, run, pkg, StateAllocating, fmt.Errorf("allocator returned incomplete assignment"))
		}
		pkg.AssignedAgent, pkg.AssignedAgentKind, pkg.Workspace = allocation.AgentID, allocation.Kind, allocation.Workspace
		pkg.State = StateCompiling
	case StateCompiling:
		if capsuleErr := r.ensureContextCapsule(opCtx, run, pkg); capsuleErr != nil {
			if errors.Is(capsuleErr, ErrInfrastructureFailure) {
				// Evidence persistence is an infrastructure boundary. Fail closed
				// without retrying implementation work or invoking a provider.
				pkg.State = StateFailed
				pkg.ErrorMessage = "SCHEMA_UNAVAILABLE"
				run.State = StateFailed
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, capsuleErr
			}
			return r.packageFailureFrom(ctx, run, pkg, StateCompiling, fmt.Errorf("prepare context capsule: %w", capsuleErr))
		}
		artifact, compileErr := r.executor.Compile(opCtx, run, pkg)
		if compileErr != nil {
			return r.packageFailureFrom(ctx, run, pkg, StateCompiling, fmt.Errorf("compile package prompt: %w", compileErr))
		}
		if artifact.VersionID == "" || artifact.Content == "" {
			return r.packageFailureFrom(ctx, run, pkg, StateCompiling, fmt.Errorf("prompt compiler returned no immutable artifact"))
		}
		pkg.PromptVersionID, pkg.CompiledPrompt = artifact.VersionID, artifact.Content
		pkg.State = StateExecuting
	case StateExecuting:
		if isReviewPhase(pkg) {
			// A review phase consumes its implementation dependency and invokes
			// Review without granting the reviewer a write-capable Execute turn.
			pkg.State = StateReviewing
		} else if isVerificationPhase(pkg) {
			// Verification phases execute canonical commands in StateTesting;
			// they do not ask a verifier to mutate source files.
			pkg.State = StateTesting
		} else if pkg.ParallelGroup != "" {
			if err := r.executeParallelGroup(opCtx, run, pkg.ParallelGroup); err != nil {
				if stalled, stallErr := r.finalizeStallIfNeeded(ctx, run, err); stalled {
					return run, false, stallErr
				}
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, err
			}
		} else {
			if err := r.executeOne(opCtx, run, pkg); err != nil {
				if stalled, stallErr := r.finalizeStallIfNeeded(ctx, run, err); stalled {
					return run, false, stallErr
				}
				if errors.Is(err, ErrDispatchOutcomeUnknown) {
					r.blockNeedsHuman(run, pkg, "DISPATCH_OUTCOME_UNKNOWN", err.Error(), "Provider outcome is unknown after the dispatch boundary.", []string{"Inspect provider runtime", "Retry only after confirming whether work completed"})
					run.UpdatedAt = time.Now().UTC()
					_ = r.saveRun(ctx, run)
					return run, false, err
				}
				if run.State == StateBlockedNeedsUser && run.NeedsHuman != nil && !run.NeedsHuman.Resolved {
					// executeOne already crossed a typed human-intervention boundary.
					// Do not erase that durable state by routing the same error into
					// automatic remediation/no-progress handling.
					run.UpdatedAt = time.Now().UTC()
					if saveErr := r.saveRun(ctx, run); saveErr != nil {
						return run, false, saveErr
					}
					return run, false, err
				}
				retryFrom := StateCompiling
				if isQuotaOrRateLimitFailure(pkg.FailureKind) {
					retryFrom = StateAllocating
				}
				if terminalErr := r.markRemediation(run, pkg, retryFrom, err.Error()); terminalErr != nil {
					run.UpdatedAt = time.Now().UTC()
					_ = r.saveRun(ctx, run)
					return run, false, terminalErr
				}
			}
		}
	case StateTesting:
		verificationCommands := run.Contract.VerificationCommands
		if len(pkg.VerificationRequirements) > 0 {
			verificationCommands = pkg.VerificationRequirements
		}
		if run.Contract.RequireVerification && len(verificationCommands) == 0 {
			if pkg.ReplayProhibited && pkg.DispatchState == DispatchCompleted {
				failure := "Verification is required but no verification commands are configured. The completed provider dispatch was preserved."
				pkg.ErrorMessage = failure
				r.blockNeedsHuman(run, pkg, "COMPLETED_DISPATCH_VERIFICATION_FAILED", failure, "Configure or repair the verification requirements, then retry verification. The provider will not be executed again.", nil)
				break
			}
			if terminalErr := r.markRemediation(run, pkg, StateCompiling, "Verification is required but no verification commands are configured."); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
			break
		}
		results := r.verifier.RunVerification(opCtx, pkg.Workspace, verificationCommands)
		pkg.Verifications = append(pkg.Verifications, results...)
		if recorder, ok := r.executor.(ValidationEvidenceRecorder); ok {
			if err := recorder.RecordValidationEvidence(opCtx, run, pkg, results); err != nil {
				pkg.State = StateFailed
				pkg.ErrorMessage = "VALIDATION_EVIDENCE_UNAVAILABLE"
				run.State = StateFailed
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, fmt.Errorf("persist validation evidence: %w", err)
			}
		}
		if verificationPassed(results, run.Contract.RequireVerification) {
			if pkg.ReviewRequired {
				pkg.State = StateReviewing
			} else {
				pkg.State = StateVerified
				now := time.Now().UTC()
				pkg.FinishedAt = &now
			}
		} else {
			if pkg.ReplayProhibited && pkg.DispatchState == DispatchCompleted {
				failure := verificationFailureContext(results)
				pkg.ErrorMessage = failure
				r.blockNeedsHuman(run, pkg, "COMPLETED_DISPATCH_VERIFICATION_FAILED", failure, "Review or repair the workspace, then retry verification. The completed provider dispatch will not be replayed.", nil)
				break
			}
			// A verification failure is evidence that the current execution
			// strategy did not satisfy the task. Re-enter allocation so the Nexus
			// runtime router can apply a different model/resource strategy on the
			// next attempt instead of merely replaying the same runtime.
			if terminalErr := r.markRemediation(run, pkg, StateAllocating, verificationFailureContext(results)); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
		}
	case StateReviewing:
		verdict, reviewErr := r.executor.Review(opCtx, run, pkg)
		if reviewErr != nil {
			if terminalErr := r.markRemediation(run, pkg, StateReviewing, "Independent review failed: "+reviewErr.Error()); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
			break
		}
		if verdict.ReviewerAgentID == "" {
			return r.packageFailureFrom(ctx, run, pkg, StateReviewing, fmt.Errorf("review verdict has no reviewer identity"))
		}
		if verdict.ReviewedAt.IsZero() {
			verdict.ReviewedAt = time.Now().UTC()
		}
		verdict = NormalizeReviewVerdict(verdict)
		verdict.StructuredFindings = DeduplicateReviewFindings(verdict.StructuredFindings)
		pkg.ReviewFindings = MergeReviewFindings(pkg.ReviewFindings, verdict.StructuredFindings, verdict.Approved, pkg.Attempt)
		pkg.Verdicts = append(pkg.Verdicts, verdict)
		if verdict.Approved {
			pkg.State = StateVerified
			now := time.Now().UTC()
			pkg.FinishedAt = &now
			pkg.RemediationContext = ""
			pkg.ReviewInvalidated = false
			pkg.ReviewInvalidationReason = ""
		} else {
			if terminalErr := r.markRemediation(run, pkg, StateCompiling, reviewFailureContext(verdict)); terminalErr != nil {
				run.UpdatedAt = time.Now().UTC()
				_ = r.saveRun(ctx, run)
				return run, false, terminalErr
			}
		}
	case StateRemediating:
		canRetry, reason := r.retry.ShouldRetry(pkg, run.Contract)
		if !canRetry {
			pkg.State = StateFailed
			pkg.ErrorMessage = reason
			if run.Contract.EscalateOnFailure {
				r.blockNeedsHuman(run, pkg, "NO_PROGRESS", reason, "The configured remediation budget was exhausted without verified progress.", []string{"Review failure evidence", "Choose a different strategy or agent", "Resume after updating the plan"})
			} else {
				run.State = StateFailedNoProgress
			}
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, fmt.Errorf("package %s failed: %s", pkg.PackageID, reason)
		}
		pkg.Attempt++
		resetDispatch(pkg)
		next := pkg.RetryFrom
		if next == "" || next == StateRemediating {
			next = StateExecuting
		}
		pkg.State = next
	case StatePending:
		return r.packageFailureFrom(ctx, run, pkg, StatePending, fmt.Errorf("package selected before dependencies were verified"))
	}

	refreshDependencyStates(run)
	if allPackagesVerified(run) {
		return r.verifyGlobalDefinition(ctx, leaseCtx, run)
	}
	select {
	case hbErr := <-heartbeatErr:
		return run, false, fmt.Errorf("mission lease heartbeat lost: %w", hbErr)
	default:
	}
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return run, false, err
	}
	return run, false, nil
}

func (r *MissionRunner) blockNeedsHuman(run *MissionRun, pkg *PackageRun, reasonCode, summary, question string, actions []string) {
	run.State = StateBlockedNeedsUser
	taskID := ""
	if pkg != nil {
		taskID = pkg.PackageID
	}
	options := interventionOptions(reasonCode, pkg)
	run.NeedsHuman = &HumanIntervention{
		ID:         "intervention_" + ids.NewRuntimeID(),
		ReasonCode: reasonCode, Summary: summary, Question: question,
		Context: interventionContext(pkg), RecommendedActions: append([]string(nil), actions...),
		Impact:    "Only the affected task is paused; independent verified work is preserved.",
		MissionID: run.ID, TaskID: taskID, Source: "mission_runner", Timestamp: time.Now().UTC(),
		Version: 1, Scope: "PACKAGE", Options: options,
	}
}

func interventionContext(pkg *PackageRun) string {
	if pkg == nil {
		return ""
	}
	return pkg.RemediationContext
}

func interventionOptions(reasonCode string, pkg *PackageRun) []InterventionOption {
	packageID := ""
	if pkg != nil {
		packageID = pkg.PackageID
		if pkg.ReplayProhibited && pkg.DispatchState == DispatchCompleted {
			return []InterventionOption{{ID: "verify-completed-without-replay", Operation: InterventionVerifyCompletedWithoutReplay, Label: "Verify completed work without replay", PackageID: packageID}}
		}
	}
	switch reasonCode {
	case "DISPATCH_OUTCOME_UNKNOWN":
		return []InterventionOption{{ID: "confirm-external-completion", Operation: InterventionConfirmExternalOutcome, Label: "Confirm external completion", PackageID: packageID}}
	case "NO_PROGRESS":
		return []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, Label: "Replan package", PackageID: packageID}}
	default:
		return []InterventionOption{{ID: "replan-package", Operation: InterventionReplanPackage, Label: "Replan package", PackageID: packageID}}
	}
}

func (r *MissionRunner) packageFailureFrom(ctx context.Context, run *MissionRun, pkg *PackageRun, retryFrom State, err error) (*MissionRun, bool, error) {
	if terminalErr := r.markRemediation(run, pkg, retryFrom, err.Error()); terminalErr != nil {
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, pkg.State == StateFailed, terminalErr
	}
	run.UpdatedAt = time.Now().UTC()
	_ = r.saveRun(ctx, run)
	return run, pkg.State == StateFailed, err
}

func (r *MissionRunner) markRemediation(run *MissionRun, pkg *PackageRun, retryFrom State, failure string) error {
	failure = strings.TrimSpace(failure)
	if failure == "" {
		failure = "unknown package failure"
	}
	if pkg.LastFailureSignature == failure {
		pkg.NoProgressCount++
	} else {
		pkg.LastFailureSignature = failure
		pkg.NoProgressCount = 1
	}
	pkg.ErrorMessage = failure
	if len(pkg.Verdicts) > 0 && pkg.Verdicts[len(pkg.Verdicts)-1].Approved {
		pkg.ReviewInvalidated = true
		pkg.ReviewInvalidationReason = "package changed or verification failed after review approval"
	}
	// Every remediation attempt must carry an explicit strategy change. This is
	// persisted with the PackageRun so a restart cannot silently repeat the same
	// approach or make the transition depend on in-memory state.
	switch pkg.NoProgressCount {
	case 1:
		pkg.StrategyVariant = "ALTERNATE_APPROACH"
	case 2:
		pkg.StrategyVariant = "REPLAN_DECOMPOSE"
	default:
		pkg.StrategyVariant = fmt.Sprintf("ESCALATE_%d", pkg.NoProgressCount)
	}
	pkg.RemediationContext = fmt.Sprintf("strategy=%s; failure=%s", pkg.StrategyVariant, failure)
	pkg.RetryFrom = retryFrom
	pkg.State = StateRemediating
	limit := run.Contract.MaxNoProgress
	if limit <= 0 {
		limit = 2
	}
	if pkg.NoProgressCount >= limit {
		pkg.State = StateFailed
		run.State = StateFailedNoProgress
		return fmt.Errorf("package %s made no progress after %d identical failures: %s", pkg.PackageID, pkg.NoProgressCount, failure)
	}
	return nil
}

func (r *MissionRunner) completeRun(ctx context.Context, run *MissionRun) (*MissionRun, bool, error) {
	if run == nil {
		return nil, false, fmt.Errorf("cannot complete a nil mission run")
	}
	if !allPackagesVerified(run) {
		return run, false, fmt.Errorf("cannot complete mission %s: not all packages are verified", run.ID)
	}
	if run.State == StateBlockedNeedsUser || (run.NeedsHuman != nil && !run.NeedsHuman.Resolved) {
		return run, false, fmt.Errorf("cannot complete mission %s: human intervention is still pending", run.ID)
	}
	now := time.Now().UTC()
	run.State = StateCompletedVerified
	if run.ResumeRequest != nil && run.ResumeRequest.Status != "COMPLETED" {
		run.ResumeRequest.Status = "COMPLETED"
		run.ResumeRequest.CompletedAt = &now
	}
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := r.saveRun(ctx, run); err != nil {
		return run, false, err
	}
	return run, true, nil
}

// verifyGlobalDefinition is the final fail-closed gate. Package review is not
// sufficient evidence that the integrated delivery satisfies the plan's
// Definition of Done. Product-created autonomous runs populate the explicit
// global command set during contract normalization.
func (r *MissionRunner) verifyGlobalDefinition(ctx, operationCtx context.Context, run *MissionRun) (*MissionRun, bool, error) {
	commands := run.Contract.GlobalVerificationCommands
	if len(commands) == 0 {
		if run.Autonomous {
			run.State = StateFailedVerification
			run.PausedReason = "autonomous Mission requires an explicit global Definition-of-Done command"
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, fmt.Errorf("autonomous Mission has no global Definition-of-Done verification")
		}
		// Keep direct legacy runner callers compatible; product admission still
		// requires a real global command set for autonomous runs.
		return r.completeRun(ctx, run)
	}
	if run.Autonomous {
		if _, ok := r.executor.(ValidationEvidenceRecorder); !ok {
			run.State = StateFailed
			run.PausedReason = "VALIDATION_EVIDENCE_UNAVAILABLE"
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, ErrEvidenceRepositoryUnavailable
		}
		if _, ok := r.executor.(GlobalValidationEvidenceRecorder); !ok {
			run.State = StateFailed
			run.PausedReason = "GLOBAL_VALIDATION_EVIDENCE_UNAVAILABLE"
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, ErrEvidenceRepositoryUnavailable
		}
	}
	workspace, err := resolveIntegratedVerificationWorkspace(run)
	if errors.Is(err, ErrDeliveryIntegrationRequired) {
		integrator, ok := r.executor.(DeliveryIntegrator)
		if !ok {
			run.State = StateFailedVerification
			run.PausedReason = err.Error()
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, err
		}
		integration, integrationErr := integrator.IntegrateDelivery(operationCtx, run)
		if integrationErr != nil {
			run.State = StateFailedVerification
			run.PausedReason = "delivery integration failed: " + integrationErr.Error()
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, fmt.Errorf("integrate Mission delivery: %w", integrationErr)
		}
		if strings.TrimSpace(integration.Workspace) == "" {
			run.State = StateFailedVerification
			run.PausedReason = "delivery integration returned no verification workspace"
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, fmt.Errorf("integrate Mission delivery: empty workspace")
		}
		run.IntegratedWorkspace = integration.Workspace
		run.DeliveryIntegration = &integration
		workspace = integration.Workspace
		err = nil
	}
	if err != nil {
		run.State = StateFailedVerification
		run.PausedReason = err.Error()
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, err
	}
	run.State = StateVerifying
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return run, false, err
	}
	results := r.verifier.RunVerification(operationCtx, workspace, commands)
	run.GlobalVerifications = append(run.GlobalVerifications, results...)
	if recorder, ok := r.executor.(GlobalValidationEvidenceRecorder); ok {
		if err := recorder.RecordGlobalValidationEvidence(operationCtx, run, results); err != nil {
			run.State = StateFailed
			run.PausedReason = "VALIDATION_EVIDENCE_UNAVAILABLE"
			run.UpdatedAt = time.Now().UTC()
			_ = r.saveRun(ctx, run)
			return run, false, fmt.Errorf("persist global validation evidence: %w", err)
		}
	}
	if verificationPassed(results, true) {
		return r.completeRun(ctx, run)
	}
	var target *PackageRun
	for i := range run.PackageRuns {
		if run.PackageRuns[i].State == StateVerified {
			target = &run.PackageRuns[i]
		}
	}
	if target == nil {
		run.State = StateFailedNoProgress
		run.PausedReason = verificationFailureContext(results)
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, fmt.Errorf("global Definition of Done failed without a verified package to reopen")
	}
	if err := r.markRemediation(run, target, StateCompiling, "Global Definition of Done failed: "+verificationFailureContext(results)); err != nil {
		run.UpdatedAt = time.Now().UTC()
		_ = r.saveRun(ctx, run)
		return run, false, err
	}
	run.State = StateExecuting
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return run, false, err
	}
	return run, false, fmt.Errorf("global Definition of Done failed; package %s reopened for remediation", target.PackageID)
}

// resolveIntegratedVerificationWorkspace picks the tree that actually holds
// agent delivery. Global DoD must not run against an untouched canonical
// baseline while packages wrote isolated worktrees. Multiple divergent
// worktrees without a merged integration tree fail closed.
func resolveIntegratedVerificationWorkspace(run *MissionRun) (string, error) {
	if run == nil {
		return "", fmt.Errorf("mission run is required for global verification")
	}
	canonical := strings.TrimSpace(run.Workspace)
	seen := map[string]string{}
	var order []string
	for _, pkg := range run.PackageRuns {
		include := pkg.State == StateVerified ||
			(pkg.WorkReceipt != nil && strings.EqualFold(pkg.WorkReceipt.Status, "COMPLETED"))
		if !include {
			continue
		}
		ws := strings.TrimSpace(pkg.Workspace)
		if ws == "" {
			continue
		}
		if _, ok := seen[ws]; ok {
			continue
		}
		seen[ws] = ws
		order = append(order, ws)
	}
	if len(order) == 0 {
		return canonical, nil
	}
	var nonCanonical []string
	for _, ws := range order {
		if ws != canonical {
			nonCanonical = append(nonCanonical, ws)
		}
	}
	if len(nonCanonical) == 0 {
		return canonical, nil
	}
	if len(nonCanonical) == 1 && len(order) == 1 {
		// Single agent worktree is the delivery tree for this run.
		return nonCanonical[0], nil
	}
	if len(nonCanonical) == 1 {
		// Packages mixed canonical + one worktree: verify the worktree that
		// holds mutations, never the untouched baseline alone.
		return nonCanonical[0], nil
	}
	return "", fmt.Errorf("%w: global Definition of Done requires an integrated tree; agent worktrees are not merged into %s", ErrDeliveryIntegrationRequired, canonical)
}

func (r *MissionRunner) PauseRun(ctx context.Context, runID, reason string) (*MissionRun, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()
	if isTerminalRunState(run.State) {
		return nil, fmt.Errorf("cannot pause terminal mission run %s", run.State)
	}
	run.State = StatePaused
	run.PausedReason = reason
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (r *MissionRunner) ResumeRun(ctx context.Context, runID string) (*MissionRun, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()
	if run.State != StatePaused {
		return nil, fmt.Errorf("mission run %s cannot resume from %s", runID, run.State)
	}
	run.State = StateExecuting
	run.PausedReason = ""
	run.UpdatedAt = time.Now().UTC()
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// ResolveIntervention validates and persists a typed option for a blocked
// mission. The optionID argument is deliberately closed over the intervention
// options; the final argument is an audit actor, not an executable command.
func (r *MissionRunner) ResolveIntervention(ctx context.Context, runID, interventionID string, version int, optionID, resolvedBy string) (*MissionRun, error) {
	run, _, err := r.resolveIntervention(ctx, runID, interventionID, version, optionID, resolvedBy)
	return run, err
}

// ResolveInterventionWithOutcome also tells the application layer whether a
// new durable resume request was created. Duplicate requests must not replace
// or restart a worker that already owns the continuation.
func (r *MissionRunner) ResolveInterventionWithOutcome(ctx context.Context, runID, interventionID string, version int, optionID, resolvedBy string) (*MissionRun, bool, error) {
	return r.resolveIntervention(ctx, runID, interventionID, version, optionID, resolvedBy)
}

func (r *MissionRunner) resolveIntervention(ctx context.Context, runID, interventionID string, version int, optionID, resolvedBy string) (*MissionRun, bool, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()

	if run.NeedsHuman == nil {
		return nil, false, fmt.Errorf("%w: mission run %s has no pending intervention", ErrStaleIntervention, runID)
	}
	if version < 1 || run.NeedsHuman.Version < 1 {
		return nil, false, fmt.Errorf("%w: intervention version must be positive", ErrStaleIntervention)
	}
	key := InterventionIdempotencyKey(runID, interventionID, version, optionID)
	if run.NeedsHuman.Resolved {
		if run.NeedsHuman.ID != interventionID || run.NeedsHuman.Version != version {
			return nil, false, fmt.Errorf("%w: current intervention is %s version %d", ErrStaleIntervention, run.NeedsHuman.ID, run.NeedsHuman.Version)
		}
		if run.NeedsHuman.Resolution != nil && run.NeedsHuman.Resolution.IdempotencyKey == key {
			return run, false, nil
		}
		return run, false, fmt.Errorf("%w: existing resolution=%s", ErrInterventionAlreadyResolved, resolutionKey(run.NeedsHuman))
	}
	if run.State != StateBlockedNeedsUser {
		return nil, false, fmt.Errorf("%w: mission run %s is not blocked (state=%s)", ErrStaleIntervention, runID, run.State)
	}
	if run.NeedsHuman.ID != interventionID {
		return nil, false, fmt.Errorf("%w: expected intervention %s, got %s", ErrStaleIntervention, run.NeedsHuman.ID, interventionID)
	}
	if run.NeedsHuman.Version != version {
		return nil, false, fmt.Errorf("%w: expected version %d, got %d", ErrStaleIntervention, run.NeedsHuman.Version, version)
	}
	option, ok := findInterventionOption(run.NeedsHuman.Options, optionID)
	if !ok {
		return nil, false, fmt.Errorf("%w: option %q is not offered by intervention %s", ErrInvalidInterventionOption, optionID, interventionID)
	}
	if !run.Contract.AllowsHumanOperation(option.Operation) {
		return nil, false, fmt.Errorf("%w: operation %s is not allowed by the autonomy contract", ErrInterventionPolicyDenied, option.Operation)
	}
	if option.PackageID != "" && option.PackageID != run.NeedsHuman.TaskID {
		return nil, false, fmt.Errorf("%w: option %q is outside intervention scope", ErrInvalidInterventionOption, optionID)
	}
	pkg := packageForIntervention(run, option)
	if err := applyInterventionOption(pkg, option); err != nil {
		return nil, false, err
	}
	if (option.Operation == InterventionRetrySafePackage || option.Operation == InterventionReplanPackage) && !packageDependenciesVerified(run, pkg) {
		// Queue the resumed package behind its dependencies. refreshDependencyStates
		// will make it runnable again after every prerequisite reaches VERIFIED.
		pkg.State = StatePending
	}

	now := time.Now().UTC()
	run.NeedsHuman.Resolved = true
	run.NeedsHuman.ResolvedAt = &now
	if strings.TrimSpace(resolvedBy) == "" {
		resolvedBy = "human"
	}
	run.NeedsHuman.ResolutionDecision = option.ID
	run.NeedsHuman.ResolutionChosen = string(option.Operation)
	run.NeedsHuman.Resolution = &InterventionResolution{
		InterventionID: interventionID, Version: version, OptionID: option.ID,
		IdempotencyKey: key, ResolvedAt: now, ResolvedBy: resolvedBy,
	}
	run.State = StateExecuting
	run.PausedReason = ""
	run.ResumeRequest = &MissionResumeRequest{ID: "resume_" + key, IdempotencyKey: key, Status: "PENDING", RequestedAt: now}
	run.UpdatedAt = now

	if committer, ok := r.repo.(InterventionCommitter); ok {
		if err := committer.CommitInterventionResolution(ctx, run, run.NeedsHuman.Resolution); err != nil {
			return nil, false, err
		}
	} else if err := r.saveRun(ctx, run); err != nil {
		return nil, false, err
	}
	return run, true, nil
}

func resolutionKey(intervention *HumanIntervention) string {
	if intervention != nil && intervention.Resolution != nil {
		return intervention.Resolution.IdempotencyKey
	}
	return ""
}

func findInterventionOption(options []InterventionOption, optionID string) (InterventionOption, bool) {
	for _, option := range options {
		if option.ID == optionID {
			return option, true
		}
	}
	return InterventionOption{}, false
}

func packageForIntervention(run *MissionRun, option InterventionOption) *PackageRun {
	packageID := option.PackageID
	if packageID == "" && run.NeedsHuman != nil {
		packageID = run.NeedsHuman.TaskID
	}
	for i := range run.PackageRuns {
		if run.PackageRuns[i].PackageID == packageID {
			return &run.PackageRuns[i]
		}
	}
	return nil
}

func packageDependenciesVerified(run *MissionRun, pkg *PackageRun) bool {
	if run == nil || pkg == nil {
		return false
	}
	if len(pkg.Dependencies) == 0 {
		return true
	}
	states := make(map[string]State, len(run.PackageRuns))
	for _, candidate := range run.PackageRuns {
		states[candidate.PackageID] = candidate.State
	}
	for _, dependencyID := range pkg.Dependencies {
		if states[dependencyID] != StateVerified {
			return false
		}
	}
	return true
}

func applyInterventionOption(pkg *PackageRun, option InterventionOption) error {
	if pkg == nil {
		return fmt.Errorf("%w: intervention package is missing", ErrInvalidInterventionOption)
	}
	switch option.Operation {
	case InterventionConfirmExternalOutcome:
		if pkg.DispatchState != DispatchUnknownExternalOutcome && pkg.DispatchState != DispatchIntent {
			return fmt.Errorf("%w: external outcome for package %s is not awaiting reconciliation", ErrInvalidInterventionOption, pkg.PackageID)
		}
		// Human reconciliation proves only that the external dispatch completed.
		// It is not package verification evidence. Resume at TESTING so package
		// verification and any required independent review still run normally.
		pkg.DispatchState = DispatchCompleted
		pkg.State = StateTesting
		pkg.WorkReceipt = nil
		pkg.FinishedAt = nil
		pkg.ErrorMessage = ""
		pkg.FailureKind = model.FailureNone
		pkg.ResultSummary = "External outcome explicitly confirmed by a human; package verification is still required."
	case InterventionVerifyCompletedWithoutReplay:
		if (pkg.State != StateExecuting && pkg.State != StateTesting) || !pkg.ReplayProhibited || pkg.DispatchState != DispatchCompleted || strings.TrimSpace(pkg.DispatchID) == "" || strings.TrimSpace(pkg.AssignedRuntime) == "" {
			return fmt.Errorf("%w: package %s has no completed dispatch eligible for verification", ErrInterventionPolicyDenied, pkg.PackageID)
		}
		// Preserve the dispatch identity and runtime evidence. A new provider
		// execution could repeat side effects that already happened.
		pkg.State = StateTesting
		pkg.WorkReceipt = nil
		pkg.FinishedAt = nil
		pkg.ErrorMessage = ""
		pkg.FailureKind = model.FailureNone
		pkg.ResultSummary = "Verifying completed work without replaying the provider dispatch."
	case InterventionRetrySafePackage:
		if pkg.State == StateVerified || pkg.ReplayProhibited || (pkg.DispatchState != DispatchNone && pkg.DispatchState != DispatchFailedBeforeDispatch) {
			return fmt.Errorf("%w: package %s is not proven safe to retry", ErrInterventionPolicyDenied, pkg.PackageID)
		}
		resetDispatch(pkg)
		pkg.PromptVersionID = ""
		pkg.CompiledPrompt = ""
		pkg.State = StateAllocating
	case InterventionReplanPackage:
		if pkg.State == StateVerified || pkg.ReplayProhibited || pkg.DispatchState == DispatchIntent || pkg.DispatchState == DispatchUnknownExternalOutcome || pkg.DispatchState == DispatchFailed {
			return fmt.Errorf("%w: package %s requires external outcome reconciliation before replanning", ErrInterventionPolicyDenied, pkg.PackageID)
		}
		resetDispatch(pkg)
		pkg.PromptVersionID = ""
		pkg.CompiledPrompt = ""
		pkg.State = StateAllocating
	default:
		return fmt.Errorf("%w: unsupported operation %s", ErrInvalidInterventionOption, option.Operation)
	}
	return nil
}

func (r *MissionRunner) CancelRun(ctx context.Context, runID, reason string) (*MissionRun, error) {
	run, err := r.repo.AcquireLease(ctx, runID, r.owner, r.leaseTTL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.repo.ReleaseLease(context.Background(), runID, r.owner, run.LeaseToken) }()
	if isTerminalRunState(run.State) {
		return run, nil
	}
	now := time.Now().UTC()
	run.State = StateCanceledByUser
	run.PausedReason = reason
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := r.saveRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

// RunToTerminal executes a durable mission autonomously until a terminal or
// human-blocked state is reached. Transient package errors are allowed to flow
// through the bounded remediation state machine; terminal failures are returned.
func (r *MissionRunner) RunToTerminal(ctx context.Context, runID string) (*MissionRun, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		run, done, stepErr := r.ExecuteNextStep(ctx, runID)
		if run == nil {
			return nil, stepErr
		}
		if done || run.State == StateCompletedVerified {
			return run, nil
		}
		if run.State == StatePaused || run.State == StateBlockedNeedsUser || isTerminalRunState(run.State) {
			if stepErr != nil {
				return run, stepErr
			}
			return run, fmt.Errorf("mission stopped in state %s", run.State)
		}
		// A package-level error is persisted as REMEDIATING; keep executing
		// while the contract still permits retries instead of aborting the run.
		if stepErr != nil {
			continue
		}
	}
}

func (r *MissionRunner) beginDispatch(ctx context.Context, run *MissionRun, pkg *PackageRun) error {
	if pkg.DispatchState == DispatchIntent || pkg.DispatchState == DispatchUnknownExternalOutcome {
		if pkg.DispatchState == DispatchIntent {
			pkg.DispatchState = DispatchUnknownExternalOutcome
			pkg.ErrorMessage = fmt.Sprintf("package %s has unresolved dispatch %s; refusing duplicate provider execution", pkg.PackageID, firstNonEmptyDispatchID(pkg.DispatchID))
			_ = r.saveRun(ctx, run)
		}
		return fmt.Errorf("%w: package %s has unresolved dispatch %s; refusing duplicate provider execution", ErrDispatchOutcomeUnknown, pkg.PackageID, firstNonEmptyDispatchID(pkg.DispatchID))
	}
	now := time.Now().UTC()
	pkg.DispatchID = "dispatch_" + ids.NewRuntimeID()
	pkg.DispatchState = DispatchIntent
	pkg.DispatchStartedAt = &now
	pkg.DispatchFinishedAt = nil
	return r.saveRun(ctx, run)
}

func firstNonEmptyDispatchID(dispatchID string) string {
	if dispatchID == "" {
		return "<missing-id>"
	}
	return dispatchID
}

func finishDispatch(pkg *PackageRun, state DispatchState) {
	now := time.Now().UTC()
	pkg.DispatchState = state
	pkg.DispatchFinishedAt = &now
}

func resetDispatch(pkg *PackageRun) {
	pkg.DispatchID = ""
	pkg.DispatchState = DispatchNone
	pkg.DispatchStartedAt = nil
	pkg.DispatchFinishedAt = nil
	pkg.AssignedRuntime = ""
}

func (r *MissionRunner) executeOne(ctx context.Context, run *MissionRun, pkg *PackageRun) error {
	if err := r.beginDispatch(ctx, run, pkg); err != nil {
		run.State = StateBlockedNeedsUser
		pkg.ErrorMessage = err.Error()
		return err
	}
	outcome, err := r.executor.Execute(ctx, run, pkg, pkg.CompiledPrompt)
	if err != nil {
		pkg.FailureKind = outcome.Failure.Kind
		if outcome.RuntimeID != "" && outcome.Failure.Kind != model.FailureNone {
			pkg.AssignedRuntime = outcome.RuntimeID
			finishDispatch(pkg, DispatchFailed)
			return fmt.Errorf("execution failed: %w", err)
		}
		finishDispatch(pkg, dispatchOutcome(err))
		if pkg.DispatchState == DispatchUnknownExternalOutcome {
			return fmt.Errorf("%w: %v", ErrDispatchOutcomeUnknown, err)
		}
		return fmt.Errorf("execution failed: %w", err)
	}
	if outcome.RuntimeID == "" {
		finishDispatch(pkg, DispatchUnknownExternalOutcome)
		return fmt.Errorf("%w: executor returned no runtime evidence", ErrDispatchOutcomeUnknown)
	}
	pkg.AssignedRuntime = outcome.RuntimeID
	finishDispatch(pkg, DispatchCompleted)
	pkg.ErrorMessage = ""
	pkg.FailureKind = model.FailureNone

	if reason := strings.TrimSpace(outcome.CoordinationUnsafeReason); reason != "" {
		pkg.ReplayProhibited = true
		r.blockNeedsHuman(run, pkg, "UNSAFE_COORDINATION_AFTER_SIDE_EFFECT", reason,
			"An Agent requested coordination after mutating its workspace. Review the changes before any replay or continuation.",
			[]string{"Inspect workspace changes", "Keep or revert partial work", "Resume with an explicit plan"})
		return fmt.Errorf("unsafe coordination boundary: %s", reason)
	}
	if len(outcome.CoordinationRequests) > 0 {
		if !outcome.CoordinationReplaySafe {
			pkg.ReplayProhibited = true
			r.blockNeedsHuman(run, pkg, "COORDINATION_REPLAY_UNPROVEN",
				"Agent requested coordination without positive proof that replay is side-effect safe.",
				"Review the partial execution before allowing the package to run again.",
				[]string{"Inspect workspace changes", "Confirm no external side effects occurred", "Resume with an explicit plan"})
			return fmt.Errorf("coordination replay safety was not proven")
		}
		sourceID := pkg.PackageID
		if err := r.acceptExecutionCoordination(run, sourceID, outcome.CoordinationRequests); err != nil {
			r.blockNeedsHuman(run, pkg, "COORDINATION_REJECTED", err.Error(),
				"The coordination request could not be materialized safely.",
				[]string{"Inspect coordination request", "Adjust the plan or request", "Resume after correction"})
			return err
		}
		return nil
	}

	pkg.ResumeAgentID = ""
	if pkg.DynamicWork && strings.TrimSpace(pkg.CoordinationRequestID) != "" {
		pkg.ResultSummary = boundedCoordinationSummary(outcome.Output)
	}
	if pkg.DynamicWork && coordinationOperationForPackage(run, pkg) == CoordinationNeedInfo {
		if pkg.ResultSummary == "" {
			return fmt.Errorf("NEED_INFO coordination returned an empty answer")
		}
		pkg.State = StateVerified
		now := time.Now().UTC()
		pkg.FinishedAt = &now
		return nil
	}
	pkg.State = StateTesting
	return nil
}

// executeParallelGroup runs provider turns concurrently only after every runnable
// member of the group has reached EXECUTING. Earlier lifecycle stages remain
// deterministic and persisted one transition at a time.
func (r *MissionRunner) executeParallelGroup(ctx context.Context, run *MissionRun, group string) error {
	indexes := make([]int, 0)
	for i := range run.PackageRuns {
		pkg := &run.PackageRuns[i]
		if pkg.ParallelGroup != group || pkg.State == StateVerified || pkg.State == StateFailed || pkg.State == StatePending {
			continue
		}
		if pkg.State != StateExecuting {
			return nil
		}
		indexes = append(indexes, i)
	}
	if len(indexes) == 0 {
		return nil
	}
	// Persist every dispatch intent before launching any provider goroutine.
	for _, idx := range indexes {
		pkg := &run.PackageRuns[idx]
		if pkg.DispatchState == DispatchIntent || pkg.DispatchState == DispatchUnknownExternalOutcome {
			pkg.ErrorMessage = fmt.Sprintf("package %s has unresolved dispatch %s; refusing duplicate provider execution", pkg.PackageID, firstNonEmptyDispatchID(pkg.DispatchID))
			r.blockNeedsHuman(run, pkg, "DISPATCH_OUTCOME_UNKNOWN", pkg.ErrorMessage, "Reconcile the provider outcome before continuing.", []string{"Inspect provider runtime"})
			return fmt.Errorf("%w: %s", ErrDispatchOutcomeUnknown, pkg.ErrorMessage)
		}
		now := time.Now().UTC()
		pkg.DispatchID = "dispatch_" + ids.NewRuntimeID()
		pkg.DispatchState = DispatchIntent
		pkg.DispatchStartedAt = &now
		pkg.DispatchFinishedAt = nil
	}
	if err := r.saveRun(ctx, run); err != nil {
		return err
	}
	type result struct {
		index   int
		outcome ExecutionResult
		err     error
	}
	results := make(chan result, len(indexes))
	for _, idx := range indexes {
		runCopy := *run
		runCopy.PackageRuns = append([]PackageRun(nil), run.PackageRuns...)
		go func() {
			pkg := &runCopy.PackageRuns[idx]
			outcome, err := r.executor.Execute(ctx, &runCopy, pkg, pkg.CompiledPrompt)
			results <- result{index: idx, outcome: outcome, err: err}
		}()
	}
	var firstErr error
	for range indexes {
		res := <-results
		pkg := &run.PackageRuns[res.index]
		if res.err != nil {
			pkg.FailureKind = res.outcome.Failure.Kind
			if res.outcome.RuntimeID != "" && res.outcome.Failure.Kind != model.FailureNone {
				pkg.AssignedRuntime = res.outcome.RuntimeID
				finishDispatch(pkg, DispatchFailed)
				retryFrom := StateCompiling
				if isQuotaOrRateLimitFailure(pkg.FailureKind) {
					retryFrom = StateAllocating
				}
				if err := r.markRemediation(run, pkg, retryFrom, "Execution failed: "+res.err.Error()); err != nil && firstErr == nil {
					firstErr = err
				}
				continue
			}
			finishDispatch(pkg, dispatchOutcome(res.err))
			if pkg.DispatchState == DispatchUnknownExternalOutcome {
				if firstErr == nil {
					r.blockNeedsHuman(run, pkg, "DISPATCH_OUTCOME_UNKNOWN", res.err.Error(), "Provider outcome is unknown after the dispatch boundary.", []string{"Inspect provider runtime"})
					firstErr = fmt.Errorf("%w: %v", ErrDispatchOutcomeUnknown, res.err)
				}
				continue
			}
			if err := r.markRemediation(run, pkg, StateCompiling, "Execution failed: "+res.err.Error()); err != nil && firstErr == nil {
				firstErr = err
			}
			continue
		}
		if res.outcome.RuntimeID == "" {
			finishDispatch(pkg, DispatchUnknownExternalOutcome)
			if firstErr == nil {
				r.blockNeedsHuman(run, pkg, "DISPATCH_OUTCOME_UNKNOWN", "executor returned no runtime evidence", "Provider outcome is unknown after the dispatch boundary.", []string{"Inspect provider runtime"})
				firstErr = fmt.Errorf("%w: executor returned no runtime evidence", ErrDispatchOutcomeUnknown)
			}
			continue
		}
		pkg.AssignedRuntime = res.outcome.RuntimeID
		finishDispatch(pkg, DispatchCompleted)
		pkg.ErrorMessage = ""
		pkg.FailureKind = model.FailureNone

		if reason := strings.TrimSpace(res.outcome.CoordinationUnsafeReason); reason != "" {
			pkg.ReplayProhibited = true
			r.blockNeedsHuman(run, pkg, "UNSAFE_COORDINATION_AFTER_SIDE_EFFECT", reason,
				"An Agent requested coordination after mutating its workspace. Review the changes before any replay or continuation.",
				[]string{"Inspect workspace changes", "Keep or revert partial work", "Resume with an explicit plan"})
			if firstErr == nil {
				firstErr = fmt.Errorf("unsafe coordination boundary: %s", reason)
			}
			continue
		}
		if len(res.outcome.CoordinationRequests) > 0 {
			if !res.outcome.CoordinationReplaySafe {
				pkg.ReplayProhibited = true
				r.blockNeedsHuman(run, pkg, "COORDINATION_REPLAY_UNPROVEN",
					"Agent requested coordination without positive proof that replay is side-effect safe.",
					"Review the partial execution before allowing the package to run again.",
					[]string{"Inspect workspace changes", "Confirm no external side effects occurred", "Resume with an explicit plan"})
				if firstErr == nil {
					firstErr = fmt.Errorf("coordination replay safety was not proven")
				}
				continue
			}
			sourceID := pkg.PackageID
			if err := r.acceptExecutionCoordination(run, sourceID, res.outcome.CoordinationRequests); err != nil {
				r.blockNeedsHuman(run, pkg, "COORDINATION_REJECTED", err.Error(),
					"The coordination request could not be materialized safely.",
					[]string{"Inspect coordination request", "Adjust the plan or request", "Resume after correction"})
				if firstErr == nil {
					firstErr = err
				}
			}
			continue
		}

		pkg.ResumeAgentID = ""
		if pkg.DynamicWork && strings.TrimSpace(pkg.CoordinationRequestID) != "" {
			pkg.ResultSummary = boundedCoordinationSummary(res.outcome.Output)
		}
		if pkg.DynamicWork && coordinationOperationForPackage(run, pkg) == CoordinationNeedInfo {
			if pkg.ResultSummary == "" {
				if firstErr == nil {
					firstErr = fmt.Errorf("NEED_INFO coordination returned an empty answer")
				}
				continue
			}
			pkg.State = StateVerified
			now := time.Now().UTC()
			pkg.FinishedAt = &now
			continue
		}
		pkg.State = StateTesting
	}
	return firstErr
}

func validateDependencyGraph(pkgs []PackageRun) error {
	known := map[string]bool{}
	for _, p := range pkgs {
		if p.PackageID == "" {
			return fmt.Errorf("package id is required")
		}
		if known[p.PackageID] {
			return fmt.Errorf("duplicate package id %s", p.PackageID)
		}
		known[p.PackageID] = true
	}
	for _, p := range pkgs {
		for _, dep := range p.Dependencies {
			if dep == p.PackageID {
				return fmt.Errorf("package %s depends on itself", p.PackageID)
			}
			if !known[dep] {
				return fmt.Errorf("package %s depends on unknown package %s", p.PackageID, dep)
			}
		}
	}
	// cycle check via DFS
	deps := map[string][]string{}
	for _, p := range pkgs {
		deps[p.PackageID] = p.Dependencies
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("dependency cycle includes %s", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dep := range deps[id] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range deps {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func refreshDependencyStates(run *MissionRun) {
	states := map[string]State{}
	for _, p := range run.PackageRuns {
		states[p.PackageID] = p.State
	}
	for i := range run.PackageRuns {
		p := &run.PackageRuns[i]
		if p.State != StatePending {
			continue
		}
		ready := true
		for _, dep := range p.Dependencies {
			if states[dep] != StateVerified {
				ready = false
				break
			}
		}
		if ready {
			p.State = StateReady
		}
	}
}
func nextPackage(run *MissionRun) *PackageRun {
	var selected *PackageRun
	selectedRank := int(^uint(0) >> 1)
	for i := range run.PackageRuns {
		p := &run.PackageRuns[i]
		if p.State == StateVerified || p.State == StatePending || p.State == StateFailed {
			continue
		}
		rank := packageStageRank(p.State)
		if selected == nil || rank < selectedRank {
			selected, selectedRank = p, rank
		}
	}
	return selected
}

func packageStageRank(state State) int {
	switch state {
	case StateReady:
		return 0
	case StateAllocating:
		return 1
	case StateCompiling:
		return 2
	case StateExecuting:
		return 3
	case StateTesting:
		return 4
	case StateReviewing:
		return 5
	case StateRemediating:
		return 6
	default:
		return 100
	}
}
func allPackagesVerified(run *MissionRun) bool {
	if len(run.PackageRuns) == 0 {
		return false
	}
	for _, p := range run.PackageRuns {
		if p.State != StateVerified {
			return false
		}
	}
	return true
}
func packageIndex(run *MissionRun, id string) int {
	for i := range run.PackageRuns {
		if run.PackageRuns[i].PackageID == id {
			return i
		}
	}
	return -1
}
func verificationPassed(results []VerificationResult, required bool) bool {
	if required && len(results) == 0 {
		return false
	}
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
func verificationFailureContext(results []VerificationResult) string {
	for i := len(results) - 1; i >= 0; i-- {
		if !results[i].Passed {
			return fmt.Sprintf("Verification failed: %s (exit %d)\n%s", results[i].Command, results[i].ExitCode, results[i].OutputSnippet)
		}
	}
	return "Verification produced no passing evidence."
}
func reviewFailureContext(v ReviewVerdict) string {
	v = NormalizeReviewVerdict(v)
	ids := make([]string, 0, len(v.StructuredFindings))
	for _, finding := range v.StructuredFindings {
		ids = append(ids, finding.ID)
	}
	return fmt.Sprintf("Review rejected. Finding IDs: %v. Findings: %v. Remediation: %v", ids, v.Findings, v.RemediationTips)
}

// IsTerminalState reports whether a mission cannot make further progress.
func IsTerminalState(s State) bool {
	return s == StateCompletedVerified || s == StateFailed || s == StateFailedNoProgress || s == StateFailedBudgetExceeded || s == StateFailedVerification || s == StateCanceledByUser
}

func isTerminalRunState(s State) bool { return IsTerminalState(s) }

func isQuotaOrRateLimitFailure(kind model.FailureKind) bool {
	return kind == model.FailureQuota || kind == model.FailureRateLimit
}
