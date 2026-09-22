package nexus

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

// RunApplicationService is the transport-facing application boundary for
// MissionRun control. Lifecycle rules remain in Nexus and MissionRunner; this
// service prevents HTTP handlers from assembling those calls independently.
type RunApplicationService struct {
	nexus *Nexus
	bus   *events.Bus
}

// RunStartRequest contains the approved inputs needed to create a MissionRun.
type RunStartRequest struct {
	PlanID                string
	Revision              int
	AgentID               string
	Contract              runner.AutonomyContract
	Autonomous            bool
	CostConfirmationToken string
	ExecutionPreviewToken string
}

// RunStepResult is the stable application result for one diagnostic step.
type RunStepResult struct {
	Run       *runner.MissionRun
	Completed bool
}

func NewRunApplicationService(n *Nexus) *RunApplicationService {
	return NewRunApplicationServiceWithBus(n, events.DefaultBus())
}

func NewRunApplicationServiceWithBus(n *Nexus, bus *events.Bus) *RunApplicationService {
	return &RunApplicationService{nexus: n, bus: bus}
}

func (s *RunApplicationService) ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.nexus == nil {
		return fmt.Errorf("nexus run service unavailable")
	}
	return nil
}

func (s *RunApplicationService) Start(ctx context.Context, req RunStartRequest) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, err := s.nexus.StartMissionRunApprovedWithOptions(
		ctx,
		req.PlanID,
		req.Revision,
		req.AgentID,
		req.Contract,
		req.Autonomous,
		MissionAdmissionOptions{
			CostConfirmationToken: req.CostConfirmationToken,
			ExecutionPreviewToken: req.ExecutionPreviewToken,
		},
	)
	if err != nil {
		return nil, err
	}
	s.publishRunEvent(run, events.EventMissionStarted, "Mission run started", map[string]any{
		"plan_id": run.PlanID, "revision": run.PlanRevision, "autonomous": run.Autonomous,
	})
	return run, nil
}

func (s *RunApplicationService) List(ctx context.Context) ([]*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	return s.nexus.Runner().ListRuns(ctx)
}

func (s *RunApplicationService) Get(ctx context.Context, runID string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	return s.nexus.Runner().GetRun(ctx, runID)
}

// Routing returns the persisted, explainable allocation decisions for a run.
// It never recomputes a decision from current provider state.
func (s *RunApplicationService) Routing(ctx context.Context, runID string) (RoutingDecisionReport, error) {
	if err := s.ready(ctx); err != nil {
		return RoutingDecisionReport{}, err
	}
	run, err := s.nexus.Runner().GetRun(ctx, runID)
	if err != nil {
		return RoutingDecisionReport{}, err
	}
	if run.ExecutionSnapshotID != "" {
		snapshot, snapshotErr := s.nexus.loadMissionSnapshot(run)
		if snapshotErr != nil {
			return RoutingDecisionReport{}, snapshotErr
		}
		return BuildRoutingDecisionReportWithPlanAndPreview(run, &snapshot.Plan, snapshot.ExecutionPreview)
	}
	if plan, planErr := s.nexus.GetWorkPlan(ctx, run.PlanID); planErr == nil {
		return BuildRoutingDecisionReportWithPlan(run, plan)
	}
	return BuildRoutingDecisionReport(run)
}

func (s *RunApplicationService) Step(ctx context.Context, runID string) (*RunStepResult, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	before, _ := s.nexus.Runner().GetRun(ctx, runID)
	if before != nil {
		s.publishRunEvent(before, events.EventMissionStepStarted, "Mission step started", map[string]any{"run_id": runID})
	}
	run, completed, err := s.nexus.Runner().ExecuteNextStep(ctx, runID)
	if err != nil {
		if run != nil {
			if run.State == runner.StateBlockedNeedsUser && run.NeedsHuman != nil && (before == nil || before.NeedsHuman == nil || before.NeedsHuman.ID != run.NeedsHuman.ID) {
				s.publishRunEvent(run, events.EventHumanInterventionCreated, "Human intervention created", map[string]any{
					"intervention_id": run.NeedsHuman.ID, "version": run.NeedsHuman.Version, "reason": run.NeedsHuman.ReasonCode,
				})
			}
			s.publishRunEvent(run, events.EventMissionFailed, "Mission step failed", map[string]any{
				"run_id": runID, "error": security.Redact(err.Error()),
			})
		}
		return nil, err
	}
	if before != nil && before.ResumeRequest != nil && before.ResumeRequest.Status == "PENDING" {
		s.publishRunEvent(run, events.EventMissionResumed, "Mission resume started", map[string]any{"idempotency_key": before.ResumeRequest.IdempotencyKey})
	}
	s.publishCoordinationTransitions(before, run)
	if completed {
		s.publishRunEvent(run, events.EventMissionCompleted, "Mission run completed", map[string]any{"run_id": runID})
	} else {
		s.publishRunEvent(run, events.EventMissionStepCompleted, "Mission step completed", map[string]any{"run_id": runID})
	}
	return &RunStepResult{Run: run, Completed: completed}, nil
}

func (s *RunApplicationService) Pause(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, err := s.nexus.PauseMissionRun(ctx, runID, reason)
	if err == nil {
		s.publishRunEvent(run, events.EventMissionPaused, "Mission run paused", map[string]any{"reason": reason})
	}
	return run, err
}

func (s *RunApplicationService) TakeControl(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, err := s.nexus.TakeControlMissionRun(ctx, runID, reason)
	if err == nil {
		s.publishRunEvent(run, events.EventMissionPaused, "Mission run taken under manual control", map[string]any{"reason": reason})
	}
	return run, err
}

func (s *RunApplicationService) Resume(ctx context.Context, runID string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, err := s.nexus.ResumeMissionRun(ctx, runID)
	if err == nil {
		s.publishRunEvent(run, events.EventMissionResumed, "Mission run resumed", nil)
	}
	return run, err
}

func (s *RunApplicationService) ReturnToMission(ctx context.Context, runID string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, err := s.nexus.ReturnMissionRun(ctx, runID)
	if err == nil {
		s.publishRunEvent(run, events.EventMissionResumed, "Mission run returned to mission control", nil)
	}
	return run, err
}

func (s *RunApplicationService) Cancel(ctx context.Context, runID, reason string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, err := s.nexus.CancelMissionRun(ctx, runID, reason)
	if err == nil {
		s.publishRunEvent(run, events.EventMissionCanceled, "Mission run canceled", map[string]any{"reason": reason})
	}
	return run, err
}

func (s *RunApplicationService) ResolveIntervention(ctx context.Context, runID, interventionID string, version int, optionID, resolvedBy string) (*runner.MissionRun, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	run, created, err := s.nexus.ResolveMissionInterventionWithOutcome(ctx, runID, interventionID, version, optionID, resolvedBy)
	if err != nil {
		if errors.Is(err, runner.ErrStaleIntervention) {
			if current, getErr := s.nexus.Runner().GetRun(ctx, runID); getErr == nil {
				s.publishRunEvent(current, events.EventHumanInterventionStaleRejected, "Stale human intervention decision rejected", map[string]any{
					"intervention_id": interventionID, "version": version, "option_id": security.Redact(optionID),
				})
			}
		}
		return nil, err
	}
	if err == nil && created {
		s.publishRunEvent(run, events.EventHumanInterventionResolved, "Mission intervention resolved", map[string]any{
			"intervention_id": interventionID, "version": version, "option_id": security.Redact(optionID),
		})
		if run.ResumeRequest != nil {
			s.publishRunEvent(run, events.EventMissionResumeRequested, "Mission resume requested", map[string]any{"idempotency_key": run.ResumeRequest.IdempotencyKey})
		}
	}
	return run, err
}

type RunCoordinationResult struct {
	Request *runner.CoordinationRequest `json:"request"`
	Child   *runner.PackageRun          `json:"child,omitempty"`
}

func (s *RunApplicationService) SubmitCoordination(ctx context.Context, runID string, req runner.CoordinationRequest, materialize bool) (*RunCoordinationResult, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	saved, err := s.nexus.Runner().SubmitCoordinationRequest(ctx, runID, req)
	if err != nil {
		return nil, err
	}
	run, _ := s.nexus.Runner().GetRun(ctx, runID)
	s.publishRunEvent(run, events.EventMissionStepCompleted, "Mission coordination requested", map[string]any{
		"request_id":         saved.ID,
		"operation":          saved.Operation,
		"source_package_id":  saved.SourcePackageID,
		"coordination_phase": "REQUESTED",
	})
	result := &RunCoordinationResult{Request: saved}
	if !materialize {
		return result, nil
	}
	if resolved, source, ok := s.resolveCoordinationFromDurableContext(ctx, run, saved); ok {
		result.Request = resolved
		s.publishRunEvent(run, events.EventMissionStepCompleted, "Mission coordination resolved from durable context", map[string]any{
			"request_id":         saved.ID,
			"status":             resolved.Status,
			"resolution_source":  source,
			"coordination_phase": "RESOLVED",
		})
		return result, nil
	}
	child, err := s.nexus.Runner().MaterializeCoordinationWork(ctx, runID, saved.ID)
	if err != nil {
		return nil, err
	}
	result.Child = child
	refreshed, _ := s.nexus.Runner().GetRun(ctx, runID)
	if refreshed != nil {
		for i := range refreshed.CoordinationRequests {
			if refreshed.CoordinationRequests[i].ID == saved.ID {
				copy := refreshed.CoordinationRequests[i]
				result.Request = &copy
				break
			}
		}
	}
	s.publishRunEvent(refreshed, events.EventMissionStepCompleted, "Mission coordination materialized as dynamic work", map[string]any{
		"request_id":         saved.ID,
		"child_package_id":   child.PackageID,
		"source_package_id":  saved.SourcePackageID,
		"coordination_phase": "MATERIALIZED",
	})
	return result, nil
}

func (s *RunApplicationService) MaterializeCoordination(ctx context.Context, runID, requestID string) (*RunCoordinationResult, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	current, err := s.nexus.Runner().GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	var pending *runner.CoordinationRequest
	for i := range current.CoordinationRequests {
		if current.CoordinationRequests[i].ID == requestID {
			copy := current.CoordinationRequests[i]
			pending = &copy
			break
		}
	}
	if pending == nil {
		return nil, fmt.Errorf("coordination request %q not found in mission run %s", requestID, runID)
	}
	if resolved, source, ok := s.resolveCoordinationFromDurableContext(ctx, current, pending); ok {
		s.publishRunEvent(current, events.EventMissionStepCompleted, "Mission coordination resolved from durable context", map[string]any{
			"request_id":         requestID,
			"status":             resolved.Status,
			"resolution_source":  source,
			"coordination_phase": "RESOLVED",
		})
		return &RunCoordinationResult{Request: resolved}, nil
	}
	child, err := s.nexus.Runner().MaterializeCoordinationWork(ctx, runID, requestID)
	if err != nil {
		return nil, err
	}
	run, err := s.nexus.Runner().GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	var request *runner.CoordinationRequest
	for i := range run.CoordinationRequests {
		if run.CoordinationRequests[i].ID == requestID {
			copy := run.CoordinationRequests[i]
			request = &copy
			break
		}
	}
	if request == nil {
		return nil, fmt.Errorf("coordination request %q not found after materialization", requestID)
	}
	s.publishRunEvent(run, events.EventMissionStepCompleted, "Mission coordination materialized as dynamic work", map[string]any{
		"request_id":         requestID,
		"child_package_id":   child.PackageID,
		"source_package_id":  request.SourcePackageID,
		"coordination_phase": "MATERIALIZED",
	})
	return &RunCoordinationResult{Request: request, Child: child}, nil
}

func (s *RunApplicationService) ResolveCoordination(ctx context.Context, runID, requestID string, status runner.CoordinationStatus, summary, receiptID string) (*runner.CoordinationRequest, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	resolved, err := s.nexus.Runner().ResolveCoordinationRequest(ctx, runID, requestID, status, summary, receiptID)
	if err != nil {
		return nil, err
	}
	run, _ := s.nexus.Runner().GetRun(ctx, runID)
	s.publishRunEvent(run, events.EventMissionStepCompleted, "Mission coordination resolved", map[string]any{
		"request_id":         requestID,
		"status":             status,
		"receipt_id":         receiptID,
		"coordination_phase": "RESOLVED",
	})
	return resolved, nil
}

func (s *RunApplicationService) resolveCoordinationFromDurableContext(ctx context.Context, run *runner.MissionRun, req *runner.CoordinationRequest) (*runner.CoordinationRequest, string, bool) {
	if s == nil || s.nexus == nil || run == nil || req == nil || req.Status != runner.CoordinationPending {
		return nil, "", false
	}
	switch req.Operation {
	case runner.CoordinationNeedInfo:
		recall, err := s.nexus.RecallIntelligenceMemories(ctx, MemoryRecallRequest{
			ProjectID: run.ProjectID,
			Keys:      []string{req.Subject},
			Limit:     1,
		})
		if err != nil || len(recall.Items) == 0 {
			return nil, "", false
		}
		item := recall.Items[0]
		summary := fmt.Sprintf("%s=%s", item.Memory.Key, item.Memory.Value)
		resolved, err := s.nexus.Runner().ResolveCoordinationRequest(
			ctx,
			run.ID,
			req.ID,
			runner.CoordinationResolved,
			summary,
			"",
		)
		if err != nil {
			return nil, "", false
		}
		return resolved, "MEMORY", true
	case runner.CoordinationArtifactRequest:
		if len(req.ArtifactRefs) == 0 {
			return nil, "", false
		}
		requested := make(map[string]struct{}, len(req.ArtifactRefs))
		for _, ref := range req.ArtifactRefs {
			if ref = strings.TrimSpace(ref); ref != "" {
				requested[ref] = struct{}{}
			}
		}
		if len(requested) == 0 {
			return nil, "", false
		}
		for i := range run.PackageRuns {
			receipt := run.PackageRuns[i].WorkReceipt
			if receipt == nil {
				continue
			}
			found := map[string]struct{}{}
			for _, artifact := range receipt.Artifacts {
				if _, ok := requested[strings.TrimSpace(artifact)]; ok {
					found[strings.TrimSpace(artifact)] = struct{}{}
				}
			}
			if len(found) != len(requested) {
				continue
			}
			resolved, err := s.nexus.Runner().ResolveCoordinationRequest(
				ctx,
				run.ID,
				req.ID,
				runner.CoordinationResolved,
				receipt.Summary,
				receipt.ID,
			)
			if err != nil {
				return nil, "", false
			}
			return resolved, "WORK_RECEIPT", true
		}
	}
	return nil, "", false
}

func (s *RunApplicationService) publishCoordinationTransitions(before, after *runner.MissionRun) {
	if after == nil {
		return
	}
	previous := map[string]runner.CoordinationStatus{}
	if before != nil {
		for _, request := range before.CoordinationRequests {
			previous[request.ID] = request.Status
		}
	}
	for _, request := range after.CoordinationRequests {
		if request.Status == runner.CoordinationPending || previous[request.ID] == request.Status {
			continue
		}
		s.publishRunEvent(after, events.EventMissionStepCompleted, "Mission coordination state changed", map[string]any{
			"request_id":         request.ID,
			"status":             request.Status,
			"receipt_id":         request.ResultReceiptID,
			"child_package_id":   request.ChildPackageID,
			"coordination_phase": "RESOLVED",
		})
	}
}

func (s *RunApplicationService) Attention(ctx context.Context) (*runner.AttentionGroup, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	return s.nexus.AttentionCenter().ListAttention(ctx)
}

func (s *RunApplicationService) publishRunEvent(run *runner.MissionRun, eventType events.EventType, summary string, data map[string]any) {
	if s == nil || s.bus == nil || run == nil {
		return
	}
	runtimeID, provider, profile := "", "", ""
	eventData := make(map[string]any, len(data)+3)
	for key, value := range data {
		eventData[key] = value
	}
	if run.ProjectID != "" {
		eventData["project_id"] = run.ProjectID
	}
	if run.CurrentPkgIndex >= 0 && run.CurrentPkgIndex < len(run.PackageRuns) {
		pkg := run.PackageRuns[run.CurrentPkgIndex]
		runtimeID, provider, profile = pkg.AssignedRuntime, pkg.Provider, pkg.Profile
		if pkg.AssignedAgent != "" {
			eventData["agent_id"] = pkg.AssignedAgent
		}
	}
	s.bus.Publish(events.NewEventWithCorrelation(run.ID, runtimeID, provider, profile, eventType, summary, eventData))
}
