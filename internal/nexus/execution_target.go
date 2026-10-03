package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	remotecontrol "github.com/IAPro-Community/nexus/internal/control/remote"
	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func ResolveExecutionTarget(project store.Project, cfg AgentConfig) runner.ExecutionTarget {
	options := cfg.Options
	adapter := strings.ToLower(strings.TrimSpace(agentModeStringOption(options, "execution_adapter")))
	kindRaw := strings.ToLower(strings.TrimSpace(agentModeStringOption(options, "target_kind")))
	if adapter != "command_template" {
		return runner.ExecutionTarget{
			ID:        "local",
			Kind:      runner.ExecutionTargetLocal,
			Adapter:   "native",
			MachineID: registry.LocalMachineID(),
			Location:  "local",
			Transport: "ipc",
			Workspace: project.CanonicalPath,
			Source:    "default",
		}
	}

	kind := runner.ExecutionTargetCustom
	switch kindRaw {
	case "ssh":
		kind = runner.ExecutionTargetSSH
	case "nexus-node", "nexus_node":
		kind = runner.ExecutionTargetNexusNode
	}
	targetWorkspace := strings.TrimSpace(agentModeStringOption(options, "target_workspace"))
	if targetWorkspace == "" {
		targetWorkspace = project.CanonicalPath
	}
	location := strings.TrimSpace(agentModeStringOption(options, "target_location"))
	if location == "" {
		location = firstNonEmpty(kindRaw, "custom")
	}
	transport := "command-template"
	if kind == runner.ExecutionTargetSSH {
		transport = "ssh"
	}
	if kind == runner.ExecutionTargetNexusNode {
		transport = "nexus-node"
	}

	fingerprint := strings.Join([]string{
		string(kind),
		kindRaw,
		strings.TrimSpace(agentModeStringOption(options, "command_template")),
		targetWorkspace,
		strings.TrimSpace(agentModeStringOption(options, "target_profile_home")),
	}, "\x00")
	sum := sha256.Sum256([]byte(fingerprint))
	id := "target_" + hex.EncodeToString(sum[:10])

	return runner.ExecutionTarget{
		ID:        id,
		Kind:      kind,
		Adapter:   "command_template",
		MachineID: strings.TrimSpace(agentModeStringOption(options, "target_machine_id")),
		Location:  location,
		Transport: transport,
		Workspace: targetWorkspace,
		Source:    "agent_config",
	}
}


type ProjectExecutionTargetPreview struct {
	ID               string                     `json:"id"`
	Kind             runner.ExecutionTargetKind `json:"kind"`
	Location         string                     `json:"location"`
	MachineID        string                     `json:"machine_id,omitempty"`
	Workspace        string                     `json:"workspace"`
	Transport        string                     `json:"transport"`
	Source           string                     `json:"source"`
	DeclaredDefault  bool                       `json:"declared_default"`
	Effective        bool                       `json:"effective"`
	Selectable       bool                       `json:"selectable"`
	ActivationStatus string                     `json:"activation_status"`
	Reason           string                     `json:"reason,omitempty"`
}

type ProjectExecutionTargetProbe struct {
	ProjectID              string                            `json:"project_id"`
	TargetID               string                            `json:"target_id"`
	Kind                   runner.ExecutionTargetKind        `json:"kind"`
	Location               string                            `json:"location"`
	Workspace              string                            `json:"workspace"`
	MachineID              string                            `json:"machine_id"`
	OS                     string                            `json:"os"`
	Arch                   string                            `json:"arch"`
	NexusVersion           string                            `json:"nexus_version"`
	RemoteExecutionEnabled bool                              `json:"remote_execution_enabled"`
	Capabilities           []protocol.RemoteBridgeCapability `json:"capabilities"`
	Ready                  bool                              `json:"ready"`
	Reason                 string                            `json:"reason,omitempty"`
}

type projectSSHBridge interface {
	Close() error
}

var openProjectSSHBridge = func(
	ctx context.Context,
	targetID string,
	location string,
) (projectSSHBridge, protocol.RemoteBridgeHelloData, error) {
	client, hello, err := remotecontrol.OpenSSHBridge(ctx, targetID, location)
	if err != nil {
		return nil, protocol.RemoteBridgeHelloData{}, err
	}
	return client, hello, nil
}

type ProjectExecutionTargetCatalog struct {
	ProjectID              string                          `json:"project_id"`
	ManifestPresent        bool                            `json:"manifest_present"`
	DeclaredDefaultTarget  string                          `json:"declared_default_target"`
	EffectiveDefaultTarget string                          `json:"effective_default_target"`
	RemoteExecutionEnabled bool                            `json:"remote_execution_enabled"`
	Targets                []ProjectExecutionTargetPreview `json:"targets"`
}

// PreviewProjectExecutionTargets projects the safe, non-secret execution-target
// catalog for one Project. It never returns command/probe templates and never
// activates a remote target. Until managed target activation has its own
// explicit approval boundary, LOCAL remains the effective default.
func (n *Nexus) PreviewProjectExecutionTargets(projectID string) (ProjectExecutionTargetCatalog, error) {
	st, err := n.OpenProject()
	if err != nil {
		return ProjectExecutionTargetCatalog{}, err
	}
	project, err := st.GetProject(strings.TrimSpace(projectID))
	if err != nil {
		return ProjectExecutionTargetCatalog{}, err
	}

	remoteEnabled := flags.Enabled(flags.RemoteExecution)
	catalog := ProjectExecutionTargetCatalog{
		ProjectID:              project.ID,
		DeclaredDefaultTarget:  "local",
		EffectiveDefaultTarget: "local",
		RemoteExecutionEnabled: remoteEnabled,
		Targets: []ProjectExecutionTargetPreview{{
			ID:               "local",
			Kind:             runner.ExecutionTargetLocal,
			Location:         "local",
			MachineID:        registry.LocalMachineID(),
			Workspace:        project.CanonicalPath,
			Transport:        "ipc",
			Source:           "default",
			DeclaredDefault:  true,
			Effective:        true,
			Selectable:       true,
			ActivationStatus: "READY",
		}},
	}

	manifest, _, err := LoadProjectManifest(project.CanonicalPath)
	if errors.Is(err, os.ErrNotExist) {
		return catalog, nil
	}
	if err != nil {
		return ProjectExecutionTargetCatalog{}, err
	}
	catalog.ManifestPresent = true
	catalog.DeclaredDefaultTarget = manifest.Execution.DefaultTarget
	catalog.Targets[0].DeclaredDefault = manifest.Execution.DefaultTarget == "local"

	for _, declared := range manifest.Execution.Targets {
		preview := ProjectExecutionTargetPreview{
			ID:              declared.ID,
			Kind:            runner.ExecutionTargetSSH,
			Location:        declared.Location,
			MachineID:       declared.MachineID,
			Workspace:       declared.Workspace,
			Transport:       "ssh",
			Source:          "project_manifest",
			DeclaredDefault: declared.ID == manifest.Execution.DefaultTarget,
			Effective:       false,
			Selectable:      false,
		}
		if !remoteEnabled {
			preview.ActivationStatus = "FEATURE_DISABLED"
			preview.Reason = "managed remote execution is disabled by the remote_execution feature flag"
		} else {
			preview.ActivationStatus = "PREVIEW_ONLY"
			preview.Reason = "managed SSH activation requires an explicit approval contract and is not active in this rollout"
		}
		catalog.Targets = append(catalog.Targets, preview)
	}

	return catalog, nil
}


// ProbeProjectExecutionTarget performs an explicit, read-only SSH handshake for
// one manifest-declared target. It never starts a provider runtime.
func (n *Nexus) ProbeProjectExecutionTarget(
	ctx context.Context,
	projectID string,
	targetID string,
) (ProjectExecutionTargetProbe, error) {
	if !flags.Enabled(flags.RemoteExecution) {
		return ProjectExecutionTargetProbe{}, fmt.Errorf("managed remote execution is disabled by the remote_execution feature flag")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	st, err := n.OpenProject()
	if err != nil {
		return ProjectExecutionTargetProbe{}, err
	}
	project, err := st.GetProject(strings.TrimSpace(projectID))
	if err != nil {
		return ProjectExecutionTargetProbe{}, err
	}
	manifest, _, err := LoadProjectManifest(project.CanonicalPath)
	if err != nil {
		return ProjectExecutionTargetProbe{}, err
	}
	target, ok := manifest.ProjectExecutionTarget(strings.TrimSpace(targetID))
	if !ok {
		return ProjectExecutionTargetProbe{}, store.ErrNotFound
	}
	if target.Kind != "ssh" {
		return ProjectExecutionTargetProbe{}, fmt.Errorf("execution target %q kind %q cannot be probed through SSH", target.ID, target.Kind)
	}

	client, hello, err := openProjectSSHBridge(ctx, target.ID, target.Location)
	if err != nil {
		return ProjectExecutionTargetProbe{}, err
	}
	defer client.Close()

	if target.MachineID != "" && hello.MachineID != target.MachineID {
		return ProjectExecutionTargetProbe{}, fmt.Errorf(
			"execution target %q expected machine %q but remote bridge reported %q",
			target.ID,
			target.MachineID,
			hello.MachineID,
		)
	}

	probe := ProjectExecutionTargetProbe{
		ProjectID:              project.ID,
		TargetID:               target.ID,
		Kind:                   runner.ExecutionTargetSSH,
		Location:               target.Location,
		Workspace:              target.Workspace,
		MachineID:              hello.MachineID,
		OS:                     hello.OS,
		Arch:                   hello.Arch,
		NexusVersion:           hello.NexusVersion,
		RemoteExecutionEnabled: hello.RemoteExecutionEnabled,
		Capabilities:           append([]protocol.RemoteBridgeCapability(nil), hello.Capabilities...),
		Ready:                  hello.RemoteExecutionEnabled,
	}
	if !hello.RemoteExecutionEnabled {
		probe.Reason = "remote bridge is reachable but remote_execution is disabled on the target host"
	}
	return probe, nil
}
