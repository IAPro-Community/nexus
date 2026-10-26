package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// PlanApplicationService owns the durable WorkPlan CRUD boundary shared by
// transport adapters. Intelligence, compilation and execution remain on the
// Nexus domain facade because they cross Composer, Flow and runtime concerns.
type PlanApplicationService struct {
	nexus *Nexus
}

func NewPlanApplicationService(n *Nexus) *PlanApplicationService {
	return &PlanApplicationService{nexus: n}
}

func (s *PlanApplicationService) store(ctx context.Context) (*store.Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.nexus.OpenProject()
}

func (s *PlanApplicationService) Create(ctx context.Context, projectID, title, description string, phases []store.PlanPhase, facts map[string]string) (*store.WorkPlan, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("plan title is required")
	}

	plan := store.WorkPlan{
		ProjectID:       projectID,
		Title:           title,
		Description:     description,
		Phases:          phases,
		StructuredFacts: s.ensureIntentDecisionFacts(ctx, projectID, title, description, phases, facts),
	}
	for pi := range plan.Phases {
		if plan.Phases[pi].ID == "" {
			plan.Phases[pi].ID = "phase_" + ids.NewRuntimeID()
		}
		for wi := range plan.Phases[pi].Packages {
			if plan.Phases[pi].Packages[wi].ID == "" {
				plan.Phases[pi].Packages[wi].ID = "pkg_" + ids.NewRuntimeID()
			}
		}
	}

	return st.CreateWorkPlan(plan)
}

// ensureIntentDecisionFacts persists a deterministic intent decision on every
// WorkPlan creation path (CLI, Web, Composer) unless the caller already did.
func (s *PlanApplicationService) ensureIntentDecisionFacts(ctx context.Context, projectID, title, description string, phases []store.PlanPhase, facts map[string]string) map[string]string {
	out := facts
	if out == nil {
		out = map[string]string{}
	}
	_, hasIntentDecision := out[IntentDecisionFactKey]
	goal := strings.TrimSpace(description)
	if goal == "" {
		goal = strings.TrimSpace(title)
	}
	for _, phase := range phases {
		for _, pkg := range phase.Packages {
			if g := strings.TrimSpace(pkg.Goal); g != "" {
				goal = g
				break
			}
		}
		if goal != "" && goal != strings.TrimSpace(title) {
			break
		}
	}
	if hasIntentDecision {
		return ensureDelegationFacts(out, goal)
	}
	decision, err := s.nexus.DecideIntentForProject(ctx, projectID, goal)
	if err != nil {
		out["intent_decision_status"] = "error"
		out["intent_decision_error"] = err.Error()
		return ensureDelegationFacts(out, goal)
	}
	enriched, err := PersistIntentDecisionFacts(out, decision)
	if err != nil {
		out["intent_decision_status"] = "error"
		out["intent_decision_error"] = err.Error()
		return ensureDelegationFacts(out, goal)
	}
	enriched["intent_decision_status"] = "ok"
	return ensureDelegationFacts(enriched, goal)
}

func ensureDelegationFacts(facts map[string]string, goal string) map[string]string {
	if _, ok := facts[DelegationDecisionFactKey]; ok {
		return facts
	}
	decision := DecideDelegation(goal, DelegationMode(facts[DelegationModeFactKey]))
	enriched, err := PersistDelegationDecisionFacts(facts, decision)
	if err != nil {
		return facts
	}
	return enriched
}

func (s *PlanApplicationService) Get(ctx context.Context, planID string) (*store.WorkPlan, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.GetWorkPlan(planID)
}

func (s *PlanApplicationService) List(ctx context.Context, projectID string) ([]store.WorkPlan, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListWorkPlans(projectID)
}

func (s *PlanApplicationService) Update(ctx context.Context, plan store.WorkPlan, changeSummary string) (*store.WorkPlan, *store.PlanRevision, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, nil, err
	}
	current, currentErr := st.GetWorkPlan(plan.ID)
	if currentErr == nil {
		if plan.StructuredFacts == nil {
			plan.StructuredFacts = map[string]string{}
		}
		// Composer provenance is an immutable part of the original approval.
		// A Flow edit may revise tasks, but may not delete or replace the
		// source artifact, original decision contract or its pending blockers.
		// Replanning requires a new Composer artifact and WorkPlan.
		for _, key := range []string{
			"composer.handoff.v1", "composer.handoff.schema_version",
			"nexus.source_artifact_id", "nexus.source_artifact_revision",
			"nexus.source_artifact_hash", "nexus.source_composer_session_id",
			"composer.context",
		} {
			if original, exists := current.StructuredFacts[key]; exists {
				if updated, present := plan.StructuredFacts[key]; !present || updated != original {
					return nil, nil, fmt.Errorf("immutable Composer plan provenance %q cannot be changed; create a new Composer artifact and WorkPlan", key)
				}
			}
		}
		for _, key := range []string{DelegationDecisionFactKey, DelegationModeFactKey, LeadAgentFactKey} {
			if _, present := plan.StructuredFacts[key]; !present {
				if value := current.StructuredFacts[key]; value != "" {
					plan.StructuredFacts[key] = value
				}
			}
		}
		if raw := plan.StructuredFacts[DelegationDecisionFactKey]; raw != "" {
			if _, decodeErr := ReadDelegationDecisionFacts(plan.StructuredFacts); decodeErr != nil {
				return nil, nil, decodeErr
			}
		}
	}
	return st.UpdateWorkPlan(plan, changeSummary)
}

func (s *PlanApplicationService) Delete(ctx context.Context, planID string) error {
	st, err := s.store(ctx)
	if err != nil {
		return err
	}
	return st.DeleteWorkPlan(planID)
}

func (s *PlanApplicationService) Revisions(ctx context.Context, planID string) ([]store.PlanRevision, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListPlanRevisions(planID)
}
