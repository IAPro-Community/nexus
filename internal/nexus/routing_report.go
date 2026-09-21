package nexus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// RoutingDecisionReport is the read model for the decisions made while a
// MissionRun allocated its packages. It projects the persisted decision rather
// than recomputing a different explanation after execution.
type RoutingDecisionReport struct {
	RunID        string                       `json:"run_id"`
	PlanID       string                       `json:"plan_id,omitempty"`
	PlanRevision int                          `json:"work_plan_revision,omitempty"`
	Intent       *IntentDecision              `json:"intent,omitempty"`
	Delegation   *DelegationDecision          `json:"delegation,omitempty"`
	MissionTeam  *MissionTeamSelection        `json:"mission_team,omitempty"`
	Decisions    []RoutingDecisionReportEntry `json:"decisions"`
}

type RoutingDecisionReportEntry struct {
	PackageID string                 `json:"package_id"`
	Title     string                 `json:"title,omitempty"`
	State     runner.State           `json:"state,omitempty"`
	Decision  RuntimeRoutingDecision `json:"decision"`
}

// RoutingExplainability is the transport-facing explainability projection for
// one persisted decision: requested/selected/fallback/why plus alternatives,
// rejected and the real typed switch cause. It never recomputes routing.
type RoutingExplainability struct {
	RequestedProvider string              `json:"requested_provider,omitempty"`
	RequestedProfile  string              `json:"requested_profile,omitempty"`
	RequestedModel    string              `json:"requested_model,omitempty"`
	SelectedProvider  string              `json:"selected_provider,omitempty"`
	SelectedProfile   string              `json:"selected_profile,omitempty"`
	SelectedModel     string              `json:"selected_model,omitempty"`
	Fallback          bool                `json:"fallback"`
	Why               string              `json:"why,omitempty"`
	SwitchReason      RoutingSwitchReason `json:"switch_reason,omitempty"`
	SwitchDetail      string              `json:"switch_detail,omitempty"`
	Alternatives      []ModelCandidate    `json:"alternatives,omitempty"`
	Rejected          []string            `json:"rejected,omitempty"`
}

// ExplainRoutingDecision projects the persisted explainability fields without
// re-running routing or inventing Engine≠Backend≠Transport detail.
func ExplainRoutingDecision(decision RuntimeRoutingDecision) RoutingExplainability {
	return RoutingExplainability{
		RequestedProvider: decision.RequestedProvider,
		RequestedProfile:  decision.RequestedProfile,
		RequestedModel:    decision.RequestedModel,
		SelectedProvider:  decision.SelectedProvider,
		SelectedProfile:   decision.SelectedProfile,
		SelectedModel:     decision.SelectedModel,
		Fallback:          decision.Fallback,
		Why:               decision.Reason,
		SwitchReason:      decision.SwitchReason,
		SwitchDetail:      decision.SwitchDetail,
		Alternatives:      append([]ModelCandidate(nil), decision.Alternatives...),
		Rejected:          append([]string(nil), decision.Rejected...),
	}
}

// BuildRoutingDecisionReport reads only durable package decisions. A corrupt
// decision is an integrity error, not an invitation to synthesize an answer.
func BuildRoutingDecisionReport(run *runner.MissionRun) (RoutingDecisionReport, error) {
	return BuildRoutingDecisionReportWithPlan(run, nil)
}

// BuildRoutingDecisionReportWithPlan adds the persisted intent decision when
// the WorkPlan is available. It never re-runs routing or intent classification.
func BuildRoutingDecisionReportWithPlan(run *runner.MissionRun, plan *store.WorkPlan) (RoutingDecisionReport, error) {
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return RoutingDecisionReport{}, fmt.Errorf("mission run is required")
	}
	report := RoutingDecisionReport{
		RunID: run.ID, PlanID: run.PlanID, PlanRevision: run.PlanRevision,
		Decisions: make([]RoutingDecisionReportEntry, 0),
	}
	if plan != nil {
		if raw := strings.TrimSpace(plan.StructuredFacts[IntentDecisionFactKey]); raw != "" {
			var decision IntentDecision
			if err := json.Unmarshal([]byte(raw), &decision); err != nil {
				return RoutingDecisionReport{}, fmt.Errorf("decode persisted intent decision: %w", err)
			}
			report.Intent = &decision
		}
		if raw := strings.TrimSpace(plan.StructuredFacts[DelegationDecisionFactKey]); raw != "" {
			var decision DelegationDecision
			if err := json.Unmarshal([]byte(raw), &decision); err != nil {
				return RoutingDecisionReport{}, fmt.Errorf("decode persisted delegation decision: %w", err)
			}
			report.Delegation = &decision
		}
		if raw := strings.TrimSpace(plan.StructuredFacts[MissionTeamSelectionFactKey]); raw != "" {
			var selection MissionTeamSelection
			if err := json.Unmarshal([]byte(raw), &selection); err != nil {
				return RoutingDecisionReport{}, fmt.Errorf("decode persisted mission team selection: %w", err)
			}
			report.MissionTeam = &selection
		}
	}
	for _, pkg := range run.PackageRuns {
		if strings.TrimSpace(pkg.RoutingDecisionJSON) == "" {
			continue
		}
		var decision RuntimeRoutingDecision
		if err := json.Unmarshal([]byte(pkg.RoutingDecisionJSON), &decision); err != nil {
			return RoutingDecisionReport{}, fmt.Errorf("decode routing decision for package %s: %w", pkg.PackageID, err)
		}
		report.Decisions = append(report.Decisions, RoutingDecisionReportEntry{
			PackageID: pkg.PackageID, Title: pkg.Title, State: pkg.State, Decision: decision,
		})
	}
	return report, nil
}
