package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/autonomyguard"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// missionExecutionSnapshot is immutable input for one approved MissionRun.
// The live WorkPlan may continue evolving without changing an active run.
type missionExecutionSnapshot struct {
	Plan      store.WorkPlan          `json:"plan"`
	Contract  runner.AutonomyContract `json:"contract"`
	Preflight *FlowPreflightReport    `json:"preflight,omitempty"`
	Goal      *GoalContract           `json:"goal,omitempty"`
}

type missionWorker struct {
	id     string
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

func planToRunnerSpec(plan *store.WorkPlan, snapshotID string) (runner.PlanSpec, error) {
	if plan == nil || plan.ID == "" || plan.ProjectID == "" {
		return runner.PlanSpec{}, fmt.Errorf("valid work plan is required")
	}
	spec := runner.PlanSpec{ID: plan.ID, ProjectID: plan.ProjectID, Revision: plan.CurrentRevision, ExecutionSnapshotID: snapshotID}
	if decision, err := ReadDelegationDecisionFacts(plan.StructuredFacts); err == nil {
		spec.LeadAgentID = decision.LeadAgentID
	}
	seen := map[string]bool{}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			if strings.TrimSpace(pkg.ID) == "" {
				return runner.PlanSpec{}, fmt.Errorf("work package id is required in phase %s", phase.ID)
			}
			if seen[pkg.ID] {
				return runner.PlanSpec{}, fmt.Errorf("duplicate work package id %s", pkg.ID)
			}
			seen[pkg.ID] = true
			spec.Packages = append(spec.Packages, runner.PackageSpec{
				ID: pkg.ID, PhaseID: phase.ID, PhaseType: pkg.PhaseType, Title: pkg.Title, Goal: pkg.Goal, Priority: pkg.Priority,
				Dependencies: append([]string(nil), pkg.Dependencies...), ParallelGroup: pkg.ParallelGroup,
				Role: pkg.Role, ReviewRequired: pkg.ReviewRequired, TaskRequirements: pkg.TaskRequirements, AgentAllocation: pkg.AgentAllocation,
				AssignmentStrategy: pkg.AssignmentStrategy, ResourcePolicy: pkg.ResourcePolicy, Provider: pkg.Provider, Profile: pkg.Profile, DesiredProvider: pkg.DesiredProvider, DesiredProfile: pkg.DesiredProfile,
				SkillIDs: append([]string(nil), packageSkillIDs(pkg)...), MaestroSkills: append([]string(nil), pkg.MaestroSkills...), RelevantPaths: append([]string(nil), pkg.RelevantPaths...),
				AcceptanceCriteria: append([]string(nil), pkg.AcceptanceCriteria...), VerificationRequirements: append([]string(nil), pkg.VerificationRequirements...), EntryCriteria: append([]string(nil), pkg.EntryCriteria...), ExitCriteria: append([]string(nil), pkg.ExitCriteria...), RequiredEvidence: append([]string(nil), pkg.RequiredEvidence...), HandoffJSON: pkg.HandoffJSON, RoutingDecisionJSON: pkg.RoutingDecisionJSON,
			})
		}
	}
	if len(spec.Packages) == 0 {
		return runner.PlanSpec{}, fmt.Errorf("work plan %s has no packages", plan.ID)
	}
	return spec, nil
}

func freezePlanForExecution(n *Nexus, plan store.WorkPlan) (store.WorkPlan, error) {
	if err := validateFlowExecutionContract(plan); err != nil {
		return store.WorkPlan{}, err
	}
	for pi := range plan.Phases {
		for wi := range plan.Phases[pi].Packages {
			pkg := &plan.Phases[pi].Packages[wi]
			requested := packageSkillIDs(*pkg)
			validated, err := validateGenericSkillIDs(n, plan.ProjectID, requested)
			if err != nil {
				return store.WorkPlan{}, fmt.Errorf("freeze skills for package %s (%s): %w", pkg.ID, skillIDsErrorContext(requested), err)
			}
			pkg.SkillIDs = append([]string(nil), validated...)
			// The immutable execution snapshot carries one validated Skill set in
			// the canonical and legacy Skill fields. Process gates remain separate
			// metadata and are never reinterpreted as Skills.
			pkg.MaestroSkills = append([]string(nil), validated...)
		}
	}
	return plan, nil
}

func validateFlowExecutionContract(plan store.WorkPlan) error {
	if err := ValidateFlowDAG(FlowFromWorkPlan(plan)); err != nil {
		return fmt.Errorf("invalid Flow DAG: %w", err)
	}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			strategy := strings.ToUpper(strings.TrimSpace(pkg.AssignmentStrategy))
			switch strategy {
			case "": // legacy WorkPlan compatibility
			case string(FlowAssignmentExisting):
				if strings.TrimSpace(pkg.AgentAllocation) == "" {
					return fmt.Errorf("flow step %s uses EXISTING assignment without agent ID", pkg.ID)
				}
			case string(FlowAssignmentCreate):
				if strings.TrimSpace(pkg.AgentAllocation) != "" {
					return fmt.Errorf("flow step %s uses CREATE assignment with an existing agent ID", pkg.ID)
				}
			case string(FlowAssignmentAuto):
				if strings.TrimSpace(pkg.AgentAllocation) != "" {
					return fmt.Errorf("flow step %s uses AUTO assignment with a fixed agent ID", pkg.ID)
				}
			default:
				return fmt.Errorf("flow step %s has unsupported assignment strategy %q", pkg.ID, pkg.AssignmentStrategy)
			}
			policy := strings.ToUpper(strings.TrimSpace(pkg.ResourcePolicy))
			switch policy {
			case "", string(PolicyBalanced), string(PolicyPreserveQuota), string(PolicyPreferProvider), string(PolicyManual):
			default:
				return fmt.Errorf("flow step %s has unsupported resource policy %q", pkg.ID, pkg.ResourcePolicy)
			}
			if policy == string(PolicyManual) && strings.TrimSpace(pkg.Provider) == "" && strings.TrimSpace(pkg.Profile) == "" {
				return fmt.Errorf("flow step %s MANUAL resource policy requires provider and/or profile restriction", pkg.ID)
			}
		}
	}
	return nil
}

// StartMissionRun binds an immutable plan revision/snapshot and starts the
// durable state machine. autonomous=true starts a background worker immediately.
func (n *Nexus) StartMissionRun(ctx context.Context, planID, defaultAgentID string, contract runner.AutonomyContract, autonomous bool) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	if err := ensureDelegationApproved(plan); err != nil {
		return nil, err
	}
	project, err := st.GetProject(plan.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("resolve mission project: %w", err)
	}
	contract = normalizeAutonomyContract(contract, project.CanonicalPath)
	revision, err := st.GetPlanRevision(plan.ID, plan.CurrentRevision)
	if err != nil {
		return nil, fmt.Errorf("resolve immutable plan revision %d: %w", plan.CurrentRevision, err)
	}

	preflight, err := n.PreflightFlow(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("run preflight before mission: %w", err)
	}
	if !preflight.Ready {
		return nil, fmt.Errorf("mission blocked by preflight: invalid flow graph or checks failed")
	}
	if autonomous {
		preflight, err = n.PreflightFlowStrict(ctx, planID, true)
		if err != nil {
			return nil, fmt.Errorf("run strict admission before mission: %w", err)
		}
		if !preflight.Ready {
			return nil, fmt.Errorf("mission blocked by strict admission: worktree, resources, agent or Maestro requirements are not satisfied")
		}
	}

	frozenPlan, err := freezePlanForExecution(n, *plan)
	if err != nil {
		return nil, err
	}
	bindLeadAgent(&frozenPlan, defaultAgentID)
	envelope, err := json.Marshal(missionExecutionSnapshot{Plan: frozenPlan, Contract: contract, Preflight: preflight})
	if err != nil {
		return nil, fmt.Errorf("encode mission execution snapshot: %w", err)
	}
	snapshot, err := st.CreateExecutionSnapshot(plan.ID, revision.ID, string(envelope))
	if err != nil {
		return nil, fmt.Errorf("create immutable execution snapshot: %w", err)
	}
	spec, err := planToRunnerSpec(&frozenPlan, snapshot.ID)
	if err != nil {
		return nil, err
	}
	spec.Autonomous = autonomous
	run, err := n.Runner().StartMissionRun(ctx, spec, project.CanonicalPath, contract, defaultAgentID)
	if err != nil {
		return nil, err
	}
	if autonomous {
		n.StartMissionWorker(run.ID)
	}
	return run, nil
}

type MissionAdmissionOptions struct {
	CostConfirmationToken string
}

func (n *Nexus) StartMissionRunApproved(ctx context.Context, planID string, approvedRevision int, defaultAgentID string, contract runner.AutonomyContract, autonomous bool) (*runner.MissionRun, error) {
	return n.StartMissionRunApprovedWithOptions(
		ctx,
		planID,
		approvedRevision,
		defaultAgentID,
		contract,
		autonomous,
		MissionAdmissionOptions{},
	)
}

func (n *Nexus) StartMissionRunApprovedWithOptions(ctx context.Context, planID string, approvedRevision int, defaultAgentID string, contract runner.AutonomyContract, autonomous bool, options MissionAdmissionOptions) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	if err := ensureDelegationApproved(plan); err != nil {
		return nil, err
	}
	if plan.CurrentRevision != approvedRevision {
		return nil, fmt.Errorf("approved plan revision %d is stale; current is %d", approvedRevision, plan.CurrentRevision)
	}
	preflight, err := n.PreflightFlowStrict(ctx, planID, autonomous)
	if err != nil {
		return nil, fmt.Errorf("run preflight before approved mission: %w", err)
	}
	if !preflight.Ready {
		return nil, fmt.Errorf("approved mission blocked by preflight")
	}
	if err := validateCostBudgetAdmission(ctx, st, plan, options.CostConfirmationToken); err != nil {
		return nil, err
	}
	return n.startMissionRunAtRevision(ctx, plan, approvedRevision, defaultAgentID, contract, autonomous, preflight)
}

func validateCostBudgetAdmission(ctx context.Context, st *store.Store, plan *store.WorkPlan, confirmationToken string) error {
	if st == nil || plan == nil {
		return fmt.Errorf("cost budget admission requires store and plan")
	}
	prefs, err := st.GetCostPreferences()
	if err != nil {
		return fmt.Errorf("read cost budget policy: %w", err)
	}
	if !prefs.Enabled || (prefs.HardLimitMicroUSD == 0 && prefs.EmergencyLimitMicroUSD == 0) {
		return nil
	}
	summary, err := NewCostApplicationService(st).Summary(ctx, plan.ProjectID, 0)
	if err != nil {
		return fmt.Errorf("cost budget admission unavailable: %w", err)
	}
	expected := costBudgetConfirmationToken(plan.ID, plan.CurrentRevision, summary)
	provided := strings.TrimSpace(confirmationToken)
	if !summary.Budget.RequiresConfirmation {
		// A previously-issued token is revision/evidence-bound. If cost evidence
		// changed after preview (even toward a safer measured state), require the
		// caller to discard the stale acknowledgement and re-enter with the fresh
		// admission state. This keeps GO tied to the preview the user actually saw.
		if provided != "" {
			return fmt.Errorf("cost budget confirmation is stale; run Execution Preview again for the current cost state")
		}
		return nil
	}
	if expected == "" || provided != expected {
		return fmt.Errorf(
			"cost budget confirmation required: %s run Execution Preview again and explicitly confirm the current cost state",
			summary.Budget.Message,
		)
	}
	return nil
}

func (n *Nexus) startMissionRunAtRevision(ctx context.Context, plan *store.WorkPlan, revision int, defaultAgentID string, contract runner.AutonomyContract, autonomous bool, admission ...*FlowPreflightReport) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(plan.ProjectID)
	if err != nil {
		return nil, err
	}
	rev, err := st.GetPlanRevision(plan.ID, revision)
	if err != nil {
		return nil, err
	}
	var frozen store.WorkPlan
	if err := json.Unmarshal([]byte(rev.SnapshotJSON), &frozen); err != nil {
		return nil, err
	}
	bindLeadAgent(&frozen, defaultAgentID)
	contract = normalizeAutonomyContract(contract, project.CanonicalPath)
	var preflight *FlowPreflightReport
	if len(admission) > 0 {
		preflight = admission[0]
	}
	env, err := json.Marshal(missionExecutionSnapshot{Plan: frozen, Contract: contract, Preflight: preflight})
	if err != nil {
		return nil, err
	}
	snapshot, err := st.CreateExecutionSnapshot(plan.ID, rev.ID, string(env))
	if err != nil {
		return nil, err
	}
	spec, err := planToRunnerSpec(&frozen, snapshot.ID)
	if err != nil {
		return nil, err
	}
	spec.Autonomous = autonomous
	run, err := n.Runner().StartMissionRun(ctx, spec, project.CanonicalPath, contract, defaultAgentID)
	if err != nil {
		return nil, err
	}
	if autonomous {
		n.StartMissionWorker(run.ID)
	}
	return run, nil
}

func bindLeadAgent(plan *store.WorkPlan, leadAgentID string) {
	if plan == nil || strings.TrimSpace(leadAgentID) == "" {
		return
	}
	decision, err := ReadDelegationDecisionFacts(plan.StructuredFacts)
	if err != nil {
		return
	}
	decision = WithLeadAgent(decision, leadAgentID)
	if facts, factErr := PersistDelegationDecisionFacts(plan.StructuredFacts, decision); factErr == nil {
		plan.StructuredFacts = facts
	}
}

func normalizeAutonomyContract(contract runner.AutonomyContract, workspace string) runner.AutonomyContract {
	defaults := runner.DefaultAutonomyContract()
	if contract.MaxRetries <= 0 {
		contract.MaxRetries = defaults.MaxRetries
	}
	if contract.MaxTotalIterations <= 0 {
		contract.MaxTotalIterations = defaults.MaxTotalIterations
	}
	if contract.PackageTimeoutSeconds <= 0 {
		contract.PackageTimeoutSeconds = defaults.PackageTimeoutSeconds
	}
	if len(contract.VerificationCommands) == 0 && contract.RequireVerification {
		contract.VerificationCommands = detectVerificationCommands(workspace)
	}
	if len(contract.GlobalVerificationCommands) == 0 && contract.RequireVerification {
		contract.GlobalVerificationCommands = append([]string(nil), contract.VerificationCommands...)
	}
	return contract
}

// detectVerificationCommands picks only commands backed by files/scripts that
// actually exist in the project. No Go-only verification is assumed.
func detectVerificationCommands(workspace string) []string {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return nil
	}
	var commands []string
	if fileExists(filepath.Join(workspace, "go.mod")) {
		commands = append(commands, "go test ./...")
	}
	if fileExists(filepath.Join(workspace, "package.json")) {
		commands = append(commands, packageVerificationCommands(workspace)...)
	}
	if fileExists(filepath.Join(workspace, "pyproject.toml")) || fileExists(filepath.Join(workspace, "pytest.ini")) || fileExists(filepath.Join(workspace, "requirements.txt")) {
		commands = append(commands, "python -m pytest")
	}
	if fileExists(filepath.Join(workspace, "Cargo.toml")) {
		commands = append(commands, "cargo test")
	}
	return uniqueStrings(commands)
}

func packageVerificationCommands(workspace string) []string {
	data, err := os.ReadFile(filepath.Join(workspace, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	manager := "npm"
	if fileExists(filepath.Join(workspace, "yarn.lock")) {
		manager = "yarn"
	} else if fileExists(filepath.Join(workspace, "pnpm-lock.yaml")) {
		manager = "pnpm"
	} else if fileExists(filepath.Join(workspace, "bun.lock")) || fileExists(filepath.Join(workspace, "bun.lockb")) {
		manager = "bun"
	}
	var out []string
	for _, name := range []string{"typecheck", "lint", "test", "build"} {
		if _, ok := pkg.Scripts[name]; !ok {
			continue
		}
		switch manager {
		case "yarn", "bun":
			out = append(out, manager+" "+name)
		default:
			out = append(out, manager+" run "+name)
		}
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func (n *Nexus) loadMissionSnapshot(run *runner.MissionRun) (*missionExecutionSnapshot, error) {
	if run == nil || strings.TrimSpace(run.ExecutionSnapshotID) == "" {
		return nil, fmt.Errorf("mission run has no immutable execution snapshot")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	snapshot, err := st.GetExecutionSnapshot(run.ExecutionSnapshotID)
	if err != nil {
		return nil, fmt.Errorf("load execution snapshot %s: %w", run.ExecutionSnapshotID, err)
	}
	var envelope missionExecutionSnapshot
	if err := json.Unmarshal([]byte(snapshot.StateJSON), &envelope); err != nil {
		return nil, fmt.Errorf("decode execution snapshot %s: %w", snapshot.ID, err)
	}
	if envelope.Plan.ID != run.PlanID || envelope.Plan.CurrentRevision != run.PlanRevision {
		return nil, fmt.Errorf("execution snapshot does not match mission run plan revision")
	}
	return &envelope, nil
}

// StartMissionWorker starts at most one autonomous worker per run in this process.
func (n *Nexus) StartMissionWorker(runID string) {
	_ = n.startMissionWorker(runID, false)
}

// restartMissionWorker performs an ownership handoff for a resolved run. The
// old worker is canceled and joined before the new generation is registered;
// its defer can therefore never remove the new owner.
func (n *Nexus) restartMissionWorker(runID string) {
	_ = n.startMissionWorker(runID, true)
}

func (n *Nexus) startMissionWorker(runID string, replace bool) *missionWorker {
	if strings.TrimSpace(runID) == "" {
		return nil
	}
	for {
		n.workersMu.Lock()
		if n.workers == nil {
			n.workers = map[string]*missionWorker{}
		}
		existing := n.workers[runID]
		if existing == nil {
			ctx, cancel := context.WithCancel(context.Background())
			worker := &missionWorker{id: "worker_" + runnerID(), cancel: cancel, done: make(chan struct{})}
			n.workers[runID] = worker
			n.workersMu.Unlock()
			n.runMissionWorker(runID, ctx, worker)
			return worker
		}
		if !replace {
			n.workersMu.Unlock()
			return existing
		}
		existing.cancel()
		n.workersMu.Unlock()
		select {
		case <-existing.done:
		case <-time.After(15 * time.Second):
			return nil
		}
		n.workersMu.Lock()
		current := n.workers[runID]
		n.workersMu.Unlock()
		if current == existing {
			continue
		}
		// A different replacement already completed the handoff.
		return current
	}
}

func runnerID() string {
	return ids.NewRuntimeID()
}

func (n *Nexus) runMissionWorker(runID string, ctx context.Context, worker *missionWorker) {
	go func() {
		defer func() {
			n.releaseMissionWorker(runID, worker)
		}()
		_, worker.err = n.Runner().RunToTerminal(ctx, runID)
	}()
}

func (n *Nexus) releaseMissionWorker(runID string, worker *missionWorker) {
	n.workersMu.Lock()
	if current := n.workers[runID]; current == worker {
		delete(n.workers, runID)
	}
	n.workersMu.Unlock()
	close(worker.done)
}

// stopMissionWorker requests cancellation and waits briefly for the running
// provider/verification step to acknowledge it. This prevents a late worker
// save from overwriting PAUSED/CANCELED state.
func (n *Nexus) stopMissionWorker(runID string) error {
	n.workersMu.Lock()
	worker := n.workers[runID]
	n.workersMu.Unlock()
	if worker == nil {
		return nil
	}
	worker.cancel()
	select {
	case <-worker.done:
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("mission worker did not stop within 15s")
	}
}

// RecoverMissionRuns resumes durable non-terminal runs after Nexus restart.
func (n *Nexus) RecoverMissionRuns(ctx context.Context) error {
	_, err := n.recoverMissionRuns(ctx)
	return err
}

func (n *Nexus) recoverMissionRuns(ctx context.Context) ([]*missionWorker, error) {
	runs, err := n.Runner().ListRuns(ctx)
	if err != nil {
		return nil, err
	}
	workers := make([]*missionWorker, 0, len(runs))
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].StartedAt.Before(runs[j].StartedAt) })
	for _, run := range runs {
		if run == nil || runner.IsTerminalState(run.State) || run.State == runner.StatePaused || run.State == runner.StateBlockedNeedsUser {
			continue
		}
		if worker := n.startMissionWorker(run.ID, false); worker != nil {
			workers = append(workers, worker)
		}
	}
	return workers, nil
}

func waitMissionWorker(ctx context.Context, worker *missionWorker) error {
	if worker == nil {
		return fmt.Errorf("mission worker was not started")
	}
	select {
	case <-worker.done:
		return worker.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *Nexus) PauseMissionRun(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := n.stopMissionWorker(runID); err != nil {
		return nil, err
	}
	return n.Runner().PauseRun(ctx, runID, reason)
}

func (n *Nexus) ResumeMissionRun(ctx context.Context, runID string) (*runner.MissionRun, error) {
	run, err := n.Runner().ResumeRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	n.StartMissionWorker(runID)
	return run, nil
}

// TakeControlMissionRun pauses autonomous execution and starts an interactive
// supervised runtime for the package's persistent implementer Agent on the same
// isolated worktree. Mission state remains PAUSED until ReturnMissionRun.
func (n *Nexus) TakeControlMissionRun(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	run, err := n.PauseMissionRun(ctx, runID, reason)
	if err != nil {
		return nil, err
	}
	agentID := runner.ManualControlAgentID(run)
	if agentID == "" {
		return run, nil
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, fmt.Errorf("resolve takeover agent: %w", err)
	}
	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		return nil, fmt.Errorf("resolve takeover agent config: %w", err)
	}
	if cfg.Provider == "" || cfg.Profile == "" {
		return nil, fmt.Errorf("takeover agent %s has no provider/profile allocation", agentID)
	}
	if _, err := n.StartAgent(ctx, agentID, cfg.Provider, cfg.Profile); err != nil {
		return nil, fmt.Errorf("start interactive takeover runtime: %w", err)
	}
	return n.Runner().GetRun(ctx, runID)
}

// ReturnMissionRun stops the interactive takeover runtime before resuming the
// autonomous worker. This prevents the headless executor from racing a live
// human-controlled provider process on the same Agent/worktree.
func (n *Nexus) ReturnMissionRun(ctx context.Context, runID string) (*runner.MissionRun, error) {
	run, err := n.Runner().GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	agentID := runner.ManualControlAgentID(run)
	if agentID != "" {
		if err := n.StopAgent(ctx, agentID); err != nil {
			return nil, fmt.Errorf("stop interactive takeover runtime: %w", err)
		}
	}
	if len(run.Contract.AllowedFilePatterns) > 0 {
		snapshot, snapErr := n.loadMissionSnapshot(run)
		if snapErr != nil {
			return nil, snapErr
		}
		_ = snapshot // Contract is already bound to run; loading proves immutable snapshot still exists.
		pkgWorkspace := run.Workspace
		for i := range run.PackageRuns {
			if run.PackageRuns[i].AssignedAgent == agentID && run.PackageRuns[i].Workspace != "" {
				pkgWorkspace = run.PackageRuns[i].Workspace
				break
			}
		}
		changed, guardErr := autonomyguard.GitChangedPaths(ctx, pkgWorkspace)
		if guardErr != nil {
			return nil, fmt.Errorf("validate manual takeover changes: %w", guardErr)
		}
		if guardErr := autonomyguard.ValidateAllowedChanges(changed, run.Contract.AllowedFilePatterns); guardErr != nil {
			return nil, fmt.Errorf("manual takeover violates autonomy contract: %w", guardErr)
		}
	}
	return n.ResumeMissionRun(ctx, runID)
}

func (n *Nexus) CancelMissionRun(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := n.stopMissionWorker(runID); err != nil {
		return nil, err
	}
	return n.Runner().CancelRun(ctx, runID, reason)
}
