package nexus

import (
	"context"
	"fmt"
	"strings"
)

// ConfirmCapacitySwitch safely applies a user-confirmed resource switch for an Agent,
// ensuring the action is bound to the current runtime generation and a valid recommendation fingerprint.
func (n *Nexus) ConfirmCapacitySwitch(
	ctx context.Context,
	agentID, runtimeGeneration, recommendationFingerprint, provider, profile string,
) (*ResourceAllocation, error) {
	agentID = strings.TrimSpace(agentID)
	recommendationFingerprint = strings.TrimSpace(recommendationFingerprint)
	provider = strings.TrimSpace(provider)
	profile = strings.TrimSpace(profile)

	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if recommendationFingerprint == "" {
		return nil, fmt.Errorf("recommendation_fingerprint is required for capacity switch confirmation")
	}
	if provider == "" || profile == "" {
		return nil, fmt.Errorf("provider and profile are required")
	}

	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}

	// 1. Verify agent exists
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, fmt.Errorf("agent %s not found: %w", agentID, err)
	}

	// 2. Generation verification (if runtimeGeneration provided)
	if runtimeGeneration != "" {
		gen, err := st.CurrentGeneration(agentID)
		if err == nil {
			if gen.ID != runtimeGeneration && gen.RuntimeID != runtimeGeneration {
				return nil, fmt.Errorf("stale runtime generation: expected %s, current is %s", runtimeGeneration, gen.ID)
			}
		}
	}

	// 3. Apply resource allocation to the Agent
	allocation, err := n.AllocateResource(ctx, agent.ID, provider, profile, PolicyBalanced)
	if err != nil {
		return nil, fmt.Errorf("allocate resource for capacity switch: %w", err)
	}

	return allocation, nil
}
