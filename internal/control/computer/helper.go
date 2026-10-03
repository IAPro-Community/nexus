package computer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type HelperProvider struct {
	path     string
	platform string
}

func NewHelperProvider(path, platform string) Provider {
	return &HelperProvider{path: path, platform: platform}
}

type helperRequest struct {
	ProtocolVersion int         `json:"protocol_version"`
	Operation       string      `json:"operation"`
	Payload         interface{} `json:"payload,omitempty"`
}

type helperEnvelope struct {
	OK          bool             `json:"ok"`
	Error       string           `json:"error,omitempty"`
	Capabilities *Capabilities   `json:"capabilities,omitempty"`
	Permissions *PermissionStatus `json:"permissions,omitempty"`
	Apps        []AppInfo        `json:"apps,omitempty"`
	Windows     []WindowInfo     `json:"windows,omitempty"`
	Snapshot    *Snapshot        `json:"snapshot,omitempty"`
	Action      *ActionResult    `json:"action,omitempty"`
}

func (p *HelperProvider) call(ctx context.Context, operation string, payload interface{}) (helperEnvelope, error) {
	body, err := json.Marshal(helperRequest{ProtocolVersion: helperProtocolVersion, Operation: operation, Payload: payload})
	if err != nil {
		return helperEnvelope{}, err
	}
	cmd := exec.CommandContext(ctx, p.path)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return helperEnvelope{}, fmt.Errorf("computer-use helper %s failed: %w: %s", operation, err, stderr.String())
	}
	var envelope helperEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		return helperEnvelope{}, fmt.Errorf("decode computer-use helper response: %w", err)
	}
	if !envelope.OK {
		if envelope.Error == "" {
			envelope.Error = "computer-use helper rejected the operation"
		}
		return helperEnvelope{}, fmt.Errorf("%s", envelope.Error)
	}
	return envelope, nil
}

func (p *HelperProvider) Capabilities(ctx context.Context) (Capabilities, error) {
	envelope, err := p.call(ctx, "capabilities", nil)
	if err != nil {
		return Capabilities{}, err
	}
	if envelope.Capabilities == nil {
		return Capabilities{}, fmt.Errorf("computer-use helper omitted capabilities")
	}
	if envelope.Capabilities.ProtocolVersion != helperProtocolVersion {
		return Capabilities{}, fmt.Errorf("computer-use helper protocol %d is incompatible with Nexus protocol %d", envelope.Capabilities.ProtocolVersion, helperProtocolVersion)
	}
	return *envelope.Capabilities, nil
}

func (p *HelperProvider) Permissions(ctx context.Context) (PermissionStatus, error) {
	envelope, err := p.call(ctx, "permissions", nil)
	if err != nil {
		return PermissionStatus{}, err
	}
	if envelope.Permissions == nil {
		return PermissionStatus{}, fmt.Errorf("computer-use helper omitted permission status")
	}
	return *envelope.Permissions, nil
}

func (p *HelperProvider) ListApps(ctx context.Context) ([]AppInfo, error) {
	envelope, err := p.call(ctx, "list_apps", nil)
	return envelope.Apps, err
}

func (p *HelperProvider) ListWindows(ctx context.Context, app string) ([]WindowInfo, error) {
	envelope, err := p.call(ctx, "list_windows", map[string]string{"app": app})
	return envelope.Windows, err
}

func (p *HelperProvider) Snapshot(ctx context.Context, req SnapshotRequest) (Snapshot, error) {
	envelope, err := p.call(ctx, "snapshot", req)
	if err != nil {
		return Snapshot{}, err
	}
	if envelope.Snapshot == nil {
		return Snapshot{}, fmt.Errorf("computer-use helper omitted snapshot")
	}
	return *envelope.Snapshot, nil
}

func (p *HelperProvider) Act(ctx context.Context, req ActionRequest) (ActionResult, error) {
	envelope, err := p.call(ctx, "act", req)
	if err != nil {
		return ActionResult{}, err
	}
	if envelope.Action == nil {
		return ActionResult{}, fmt.Errorf("computer-use helper omitted action result")
	}
	return *envelope.Action, nil
}
