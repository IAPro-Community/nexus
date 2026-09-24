package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	ActivationNexusOverlay = "NEXUS_OVERLAY"

	EnforcementEnforced   = "ENFORCED"
	EnforcementBestEffort = "BEST_EFFORT"
	EnforcementUnverified = "UNVERIFIED"
)

type RuntimeExecutionRoleView struct {
	Binding   store.RuntimeExecutionRoleBinding `json:"binding"`
	Role      ExecutionRole                     `json:"role"`
	Persisted bool                              `json:"persisted"`
}

// EffectiveRuntimeExecutionRole returns the persisted runtime role or the
// provider-independent build default without creating durable state.
func (n *Nexus) EffectiveRuntimeExecutionRole(runtimeID string) (RuntimeExecutionRoleView, error) {
	runtimeID = strings.TrimSpace(runtimeID)
	if runtimeID == "" {
		return RuntimeExecutionRoleView{}, fmt.Errorf("runtime_id is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return RuntimeExecutionRoleView{}, err
	}
	generation, err := st.GenerationByRuntimeID(runtimeID)
	if err != nil {
		return RuntimeExecutionRoleView{}, err
	}
	if binding, bindErr := st.RuntimeExecutionRole(generation.ID); bindErr == nil {
		role, ok := ExecutionRoleForID(binding.RoleID)
		if !ok {
			return RuntimeExecutionRoleView{}, fmt.Errorf("runtime %s references unknown execution role %q", runtimeID, binding.RoleID)
		}
		return RuntimeExecutionRoleView{Binding: binding, Role: role, Persisted: true}, nil
	} else if bindErr != store.ErrNotFound {
		return RuntimeExecutionRoleView{}, bindErr
	}

	role, _ := ExecutionRoleForID("build")
	return RuntimeExecutionRoleView{
		Binding: store.RuntimeExecutionRoleBinding{
			RuntimeGenerationID: generation.ID,
			AgentID:             generation.AgentID,
			RoleID:              role.ID,
			Scope:               string(ExecutionRoleRuntime),
			ActivationStrategy:  ActivationNexusOverlay,
			Enforcement:         EnforcementUnverified,
		},
		Role:      role,
		Persisted: false,
	}, nil
}

// SetRuntimeExecutionRole changes only the operational role binding of the
// concrete runtime generation. Agent identity and AgentRevision remain intact.
func (n *Nexus) SetRuntimeExecutionRole(ctx context.Context, runtimeID, roleID, activatedBy string) (RuntimeExecutionRoleView, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeExecutionRoleView{}, err
	}
	runtimeID = strings.TrimSpace(runtimeID)
	if runtimeID == "" {
		return RuntimeExecutionRoleView{}, fmt.Errorf("runtime_id is required")
	}
	role, ok := ExecutionRoleForID(roleID)
	if !ok {
		return RuntimeExecutionRoleView{}, fmt.Errorf("unknown execution role %q", roleID)
	}

	st, err := n.OpenProject()
	if err != nil {
		return RuntimeExecutionRoleView{}, err
	}
	generation, err := st.GenerationByRuntimeID(runtimeID)
	if err != nil {
		return RuntimeExecutionRoleView{}, err
	}

	enforcement := EnforcementUnverified
	if role.AccessIntent == AccessReadOnly || role.AccessIntent == AccessVerifyOnly {
		// M1 uses the existing prompt/context overlay only. Native/sandbox
		// enforcement is introduced by the projection/capability milestones.
		enforcement = EnforcementBestEffort
	}
	binding, err := st.SetRuntimeExecutionRole(store.RuntimeExecutionRoleBinding{
		RuntimeGenerationID: generation.ID,
		AgentID:             generation.AgentID,
		RoleID:              role.ID,
		Scope:               string(ExecutionRoleRuntime),
		ActivationStrategy:  ActivationNexusOverlay,
		Enforcement:         enforcement,
		ActivatedBy:         strings.TrimSpace(activatedBy),
	})
	if err != nil {
		return RuntimeExecutionRoleView{}, err
	}
	return RuntimeExecutionRoleView{Binding: binding, Role: role, Persisted: true}, nil
}
