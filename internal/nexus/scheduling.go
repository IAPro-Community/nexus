package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	ScheduleAt            = "AT"
	ScheduleAfterRun      = "AFTER_RUN"
	ScheduleWhenResources = "WHEN_RESOURCES"
)

type missionSchedulePayload struct {
	AgentID      string                  `json:"agent_id,omitempty"`
	PlanRevision int                     `json:"plan_revision"`
	Contract     runner.AutonomyContract `json:"contract"`
}

func (n *Nexus) ScheduleMission(ctx context.Context, planID string, args ...interface{}) (*store.MissionSchedule, error) {
	var approved int
	var mode string
	var scheduledFor *time.Time
	var afterRunID, agentID string
	var contract runner.AutonomyContract
	if len(args) == 6 {
		approved, _ = args[0].(int)
		mode, _ = args[1].(string)
		scheduledFor, _ = args[2].(*time.Time)
		afterRunID, _ = args[3].(string)
		agentID, _ = args[4].(string)
		contract, _ = args[5].(runner.AutonomyContract)
	} else if len(args) == 5 {
		mode, _ = args[0].(string)
		scheduledFor, _ = args[1].(*time.Time)
		afterRunID, _ = args[2].(string)
		agentID, _ = args[3].(string)
		contract, _ = args[4].(runner.AutonomyContract)
	} else {
		return nil, fmt.Errorf("invalid schedule arguments")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	mode = strings.ToUpper(strings.TrimSpace(mode))
	switch mode {
	case ScheduleAt:
		if scheduledFor == nil {
			return nil, fmt.Errorf("scheduled_for is required for AT schedule")
		}
		t := scheduledFor.UTC()
		scheduledFor = &t
	case ScheduleAfterRun:
		if strings.TrimSpace(afterRunID) == "" {
			return nil, fmt.Errorf("after_run_id is required for AFTER_RUN schedule")
		}
		if _, err := n.Runner().GetRun(ctx, afterRunID); err != nil {
			return nil, fmt.Errorf("dependency mission run not found: %w", err)
		}
	case ScheduleWhenResources:
	default:
		return nil, fmt.Errorf("unsupported schedule mode %q", mode)
	}
	if approved == 0 {
		approved = plan.CurrentRevision
	}
	if _, _, err := loadWorkPlanRevision(st, plan.ID, approved); err != nil {
		return nil, fmt.Errorf("approved schedule revision %d is unavailable: %w", approved, err)
	}
	payload, err := json.Marshal(missionSchedulePayload{AgentID: agentID, PlanRevision: approved, Contract: normalizeAutonomyContract(contract, "")})
	if err != nil {
		return nil, err
	}
	return st.CreateMissionSchedule(store.MissionSchedule{
		PlanID: plan.ID, ProjectID: plan.ProjectID, Mode: mode, ScheduledFor: scheduledFor,
		AfterRunID: afterRunID, ContractJSON: string(payload),
	})
}

func (n *Nexus) ListMissionSchedules(ctx context.Context, projectID string) ([]store.MissionSchedule, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListMissionSchedules(projectID)
}

func (n *Nexus) CancelMissionSchedule(ctx context.Context, id string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	return st.UpdateMissionScheduleStatus(id, store.ScheduleCanceled)
}

func (n *Nexus) StartScheduleLoop() {
	n.schedulerOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		n.scheduleCancel = cancel
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_ = n.processMissionSchedules(ctx)
				}
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

func (n *Nexus) startScheduledMissionRun(ctx context.Context, planID string, revision int, agentID string, contract runner.AutonomyContract, options MissionAdmissionOptions) (*runner.MissionRun, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	frozen, _, err := loadWorkPlanRevision(st, planID, revision)
	if err != nil {
		return nil, fmt.Errorf("load scheduled plan revision: %w", err)
	}
	accounts, resourceErr := n.ListResourcesContext(ctx)
	preview, err := n.previewPlanExecution(ctx, st, frozen, accounts, resourceErr)
	if err != nil {
		return nil, fmt.Errorf("refresh scheduled execution preview: %w", err)
	}
	if err := validateApprovedExecutionPreview(preview, planID, revision); err != nil {
		return nil, err
	}
	preflight := &FlowPreflightReport{
		PlanID:      planID,
		Revision:    revision,
		Ready:       true,
		Strict:      true,
		GeneratedAt: time.Now().UTC(),
		Checks:      []FlowPreflightCheck{},
	}
	preflight, err = n.strictAdmissionForPlan(ctx, *frozen, preflight, true)
	if err != nil {
		return nil, fmt.Errorf("run strict preflight for scheduled mission: %w", err)
	}
	if !preflight.Ready {
		return nil, fmt.Errorf("scheduled mission blocked by strict preflight")
	}
	if err := validateCostBudgetAdmission(ctx, st, frozen, options.CostConfirmationToken); err != nil {
		return nil, err
	}
	return n.startMissionRunAtRevisionWithAdmission(ctx, frozen, revision, agentID, contract, true, preflight, preview)
}

func (n *Nexus) processMissionSchedules(ctx context.Context) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	all, err := st.ListMissionSchedules("")
	if err != nil {
		return err
	}
	for _, schedule := range all {
		if schedule.Status == store.SchedulePending && schedule.Mode == ScheduleAfterRun && schedule.AfterRunID != "" {
			dependency, depErr := n.Runner().GetRun(ctx, schedule.AfterRunID)
			if depErr == nil && runner.IsTerminalState(dependency.State) && dependency.State != runner.StateCompletedVerified {
				_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
				continue
			}
		}
		if schedule.Status != store.ScheduleRunning || schedule.RunID == "" {
			continue
		}
		run, getErr := n.Runner().GetRun(ctx, schedule.RunID)
		if getErr != nil {
			continue
		}
		if run.State == runner.StateCompletedVerified {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleCompleted)
		} else if runner.IsTerminalState(run.State) {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
		}
	}

	ready, err := st.ListReadyMissionSchedules(time.Now().UTC())
	if err != nil {
		return err
	}
	for _, schedule := range ready {
		if schedule.Mode == ScheduleWhenResources && !n.hasAutonomousResource() {
			continue
		}
		var payload missionSchedulePayload
		if err := json.Unmarshal([]byte(schedule.ContractJSON), &payload); err != nil {
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
			continue
		}
		claimed, claimErr := st.ClaimMissionSchedule(schedule.ID)
		if claimErr != nil {
			return claimErr
		}
		if !claimed {
			// Another scheduler worker won the atomic claim after the ready list
			// was read. It owns responsibility for creating this run.
			continue
		}
		run, startErr := n.startScheduledMissionRun(ctx, schedule.PlanID, payload.PlanRevision, payload.AgentID, payload.Contract, MissionAdmissionOptions{})
		if startErr != nil {
			// WHEN_RESOURCES waits without claiming while no autonomous resource is
			// available. After a schedule is atomically claimed, admission failures
			// are durable failures: silently returning to PENDING can loop forever
			// on stale budget/policy/plan errors every scheduler tick.
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
			continue
		}
		if err := st.BindMissionScheduleRun(schedule.ID, run.ID); err != nil {
			_, _ = n.CancelMissionRun(ctx, run.ID, "schedule binding failed")
			_ = st.UpdateMissionScheduleStatus(schedule.ID, store.ScheduleFailed)
		}
	}
	return nil
}

func (n *Nexus) hasAutonomousResource() bool {
	accounts, err := n.ListResources()
	if err != nil || len(accounts) == 0 {
		return false
	}
	rec := RecommendResources(accounts, TaskRequirements{TaskKind: "coding", Role: "implementer", RequiredCapabilities: []string{"headless", "submit_prompt"}}, PolicyBalanced)
	return rec.Recommended != nil
}
