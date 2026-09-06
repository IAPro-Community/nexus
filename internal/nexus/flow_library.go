package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

const flowLeaderPolicyFact = "nexus.flow_leader_policy"

// FlowLeaderPolicy is a per-Flow recommendation, not a global Project binding.
type FlowLeaderPolicy struct {
	Role             string   `json:"role"`
	PreferredAgentID string   `json:"preferred_agent_id,omitempty"`
	Strategy         string   `json:"strategy"` // EXISTING | AUTO | NONE
	Skills           []string `json:"skills,omitempty"`
	Why              string   `json:"why,omitempty"`
}

func (n *Nexus) SetFlowLeaderPolicy(ctx context.Context, planID string, policy FlowLeaderPolicy) (*store.WorkPlan, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	policy.Role = strings.TrimSpace(policy.Role)
	policy.Strategy = strings.ToUpper(strings.TrimSpace(policy.Strategy))
	if policy.Strategy == "" {
		policy.Strategy = "AUTO"
	}
	if policy.Strategy != "AUTO" && policy.Strategy != "EXISTING" && policy.Strategy != "NONE" {
		return nil, fmt.Errorf("unsupported leader strategy %q", policy.Strategy)
	}
	if policy.Strategy != "NONE" && policy.Role == "" {
		return nil, fmt.Errorf("leader role is required")
	}
	if policy.Strategy == "EXISTING" {
		if policy.PreferredAgentID == "" {
			return nil, fmt.Errorf("existing leader requires an agent")
		}
		if _, err := st.GetAgent(policy.PreferredAgentID, plan.ProjectID); err != nil {
			return nil, err
		}
	} else {
		policy.PreferredAgentID = ""
	}
	encoded, _ := json.Marshal(policy)
	if plan.StructuredFacts == nil {
		plan.StructuredFacts = map[string]string{}
	}
	plan.StructuredFacts[flowLeaderPolicyFact] = string(encoded)
	updated, _, err := st.UpdateWorkPlan(*plan, "Flow leader policy updated")
	return updated, err
}

func (n *Nexus) GetFlowLeaderPolicy(_ context.Context, planID string) (FlowLeaderPolicy, error) {
	st, err := n.OpenProject()
	if err != nil {
		return FlowLeaderPolicy{}, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return FlowLeaderPolicy{}, err
	}
	raw := plan.StructuredFacts[flowLeaderPolicyFact]
	if raw == "" {
		return FlowLeaderPolicy{Role: "orchestrator", Strategy: "AUTO"}, nil
	}
	var policy FlowLeaderPolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return FlowLeaderPolicy{}, fmt.Errorf("decode flow leader policy: %w", err)
	}
	return policy, nil
}

// CloneFlowToProject creates an independent portable Draft. Agent bindings and
// contextual snapshots cannot cross Project boundaries and are deliberately removed.
func (n *Nexus) CloneFlowToProject(ctx context.Context, planID, destinationProjectID string) (*store.WorkPlan, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	source, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	if _, err := st.GetProject(destinationProjectID); err != nil {
		return nil, err
	}
	clone := *source
	clone.ID = ""
	clone.ProjectID = destinationProjectID
	clone.MissionID = ""
	clone.Status = "DRAFT"
	clone.CurrentRevision = 1
	clone.CreatedAt = source.CreatedAt
	clone.UpdatedAt = source.UpdatedAt
	clone.Phases = append([]store.PlanPhase(nil), source.Phases...)
	for pi := range clone.Phases {
		clone.Phases[pi].Packages = append([]store.WorkPackage(nil), source.Phases[pi].Packages...)
		for wi := range clone.Phases[pi].Packages {
			pkg := &clone.Phases[pi].Packages[wi]
			pkg.AgentAllocation = ""
			pkg.AssignmentStrategy = "AUTO"
			pkg.Provider = ""
			pkg.Profile = ""
			pkg.CompiledPrompt = ""
		}
	}
	clone.StructuredFacts = map[string]string{}
	for key, value := range source.StructuredFacts {
		clone.StructuredFacts[key] = value
	}
	if raw := clone.StructuredFacts[flowLeaderPolicyFact]; raw != "" {
		var policy FlowLeaderPolicy
		if json.Unmarshal([]byte(raw), &policy) == nil {
			policy.PreferredAgentID = ""
			policy.Strategy = "AUTO"
			encoded, _ := json.Marshal(policy)
			clone.StructuredFacts[flowLeaderPolicyFact] = string(encoded)
		}
	}
	return st.CreateWorkPlan(clone)
}
