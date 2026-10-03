package nexus

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const managedExecutionTargetSettingKey = "managed_execution_target"

var ErrExecutionTargetApprovalStale = errors.New("execution target approval is stale")

type ManagedExecutionTargetBinding struct {
	TargetID      string    `json:"target_id"`
	ApprovalToken string    `json:"approval_token"`
	MachineID     string    `json:"machine_id"`
	ActivatedAt   time.Time `json:"activated_at"`
}

func managedExecutionTargetBindingFromProject(
	project store.Project,
) (*ManagedExecutionTargetBinding, error) {
	raw := strings.TrimSpace(project.Settings)
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("decode project settings: %w", err)
	}
	value := settings[managedExecutionTargetSettingKey]
	if len(value) == 0 || string(value) == "null" {
		return nil, nil
	}
	var binding ManagedExecutionTargetBinding
	if err := json.Unmarshal(value, &binding); err != nil {
		return nil, fmt.Errorf("decode managed execution target binding: %w", err)
	}
	binding.TargetID = strings.TrimSpace(binding.TargetID)
	binding.ApprovalToken = strings.TrimSpace(binding.ApprovalToken)
	binding.MachineID = strings.TrimSpace(binding.MachineID)
	if binding.TargetID == "" || binding.ApprovalToken == "" || binding.MachineID == "" {
		return nil, fmt.Errorf("managed execution target binding is incomplete")
	}
	return &binding, nil
}

func (n *Nexus) ManagedExecutionTargetBinding(
	projectID string,
) (*ManagedExecutionTargetBinding, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(strings.TrimSpace(projectID))
	if err != nil {
		return nil, err
	}
	return managedExecutionTargetBindingFromProject(project)
}

// ActivateManagedExecutionTarget persists only a previously probed target
// fingerprint. It re-probes before writing, so stale/mutated target approvals
// cannot be activated.
func (n *Nexus) ActivateManagedExecutionTarget(
	ctx context.Context,
	projectID string,
	targetID string,
	approvalToken string,
) (*ManagedExecutionTargetBinding, error) {
	if !flags.Enabled(flags.RemoteExecution) {
		return nil, fmt.Errorf("%w: remote_execution feature flag is off", ErrManagedRemoteExecutionDisabled)
	}
	projectID = strings.TrimSpace(projectID)
	targetID = strings.TrimSpace(targetID)
	approvalToken = strings.TrimSpace(approvalToken)
	if projectID == "" || targetID == "" || approvalToken == "" {
		return nil, fmt.Errorf("project_id, target_id and approval_token are required")
	}

	probe, err := n.ProbeProjectExecutionTarget(ctx, projectID, targetID)
	if err != nil {
		return nil, err
	}
	if !probe.Ready || probe.ApprovalToken == "" {
		return nil, fmt.Errorf("%w: execution target is not ready", ErrExecutionTargetApprovalStale)
	}
	if subtle.ConstantTimeCompare([]byte(approvalToken), []byte(probe.ApprovalToken)) != 1 {
		return nil, fmt.Errorf("%w: probe fingerprint changed; test the target again", ErrExecutionTargetApprovalStale)
	}

	binding := ManagedExecutionTargetBinding{
		TargetID:      probe.TargetID,
		ApprovalToken: probe.ApprovalToken,
		MachineID:     probe.MachineID,
		ActivatedAt:   time.Now().UTC(),
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if err := st.SetProjectSetting(projectID, managedExecutionTargetSettingKey, binding); err != nil {
		return nil, err
	}
	return &binding, nil
}

func (n *Nexus) DeactivateManagedExecutionTarget(projectID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	return st.DeleteProjectSetting(strings.TrimSpace(projectID), managedExecutionTargetSettingKey)
}
