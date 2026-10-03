package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	remotecontrol "github.com/IAPro-Community/nexus/internal/control/remote"
	"github.com/IAPro-Community/nexus/internal/core/flags"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

var (
	ErrManagedRemoteExecutionDisabled = errors.New("managed remote execution is disabled")
	ErrExecutionTargetIdentityMismatch = errors.New("execution target machine identity mismatch")
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
	ApprovalToken          string                            `json:"approval_token,omitempty"`
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
// catalog for one Project. It never performs SSH or returns credentials. The
// effective target comes only from an explicit persisted binding; repository
// defaultTarget remains declarative until the user probes and activates it.
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
	binding, err := managedExecutionTargetBindingFromProject(project)
	if err != nil {
		return ProjectExecutionTargetCatalog{}, err
	}

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
			Effective:        binding == nil,
			Selectable:       true,
			ActivationStatus: "READY",
		}},
	}
	if binding != nil {
		catalog.EffectiveDefaultTarget = binding.TargetID
	}

	manifest, _, err := LoadProjectManifest(project.CanonicalPath)
	if errors.Is(err, os.ErrNotExist) {
		if binding != nil {
			catalog.Targets = append(catalog.Targets, ProjectExecutionTargetPreview{
				ID:               binding.TargetID,
				Kind:             runner.ExecutionTargetSSH,
				MachineID:        binding.MachineID,
				Transport:        "ssh",
				Source:           "project_binding",
				Effective:        true,
				Selectable:       false,
				ActivationStatus: "STALE_BINDING",
				Reason:           "the active target binding is no longer declared by nexus.yaml; deactivate or restore the target",
			})
		}
		return catalog, nil
	}
	if err != nil {
		return ProjectExecutionTargetCatalog{}, err
	}
	catalog.ManifestPresent = true
	catalog.DeclaredDefaultTarget = manifest.Execution.DefaultTarget
	catalog.Targets[0].DeclaredDefault = manifest.Execution.DefaultTarget == "local"

	bindingMatched := false
	for _, declared := range manifest.Execution.Targets {
		isBound := binding != nil && binding.TargetID == declared.ID
		if isBound {
			bindingMatched = true
		}
		preview := ProjectExecutionTargetPreview{
			ID:              declared.ID,
			Kind:            runner.ExecutionTargetSSH,
			Location:        declared.Location,
			MachineID:       declared.MachineID,
			Workspace:       declared.Workspace,
			Transport:       "ssh",
			Source:          "project_manifest",
			DeclaredDefault: declared.ID == manifest.Execution.DefaultTarget,
			Effective:       isBound,
		}
		switch {
		case isBound && !remoteEnabled:
			preview.Selectable = false
			preview.ActivationStatus = "FEATURE_DISABLED"
			preview.Reason = "this target is explicitly bound, but remote_execution is disabled; launches fail closed"
		case isBound:
			preview.Selectable = true
			preview.ActivationStatus = "ACTIVE"
			preview.Reason = "explicitly approved project binding; every launch revalidates the remote fingerprint"
		case !remoteEnabled:
			preview.Selectable = false
			preview.ActivationStatus = "FEATURE_DISABLED"
			preview.Reason = "managed remote execution is disabled by the remote_execution feature flag"
		default:
			preview.Selectable = true
			preview.ActivationStatus = "PROBE_REQUIRED"
			preview.Reason = "test the SSH bridge and explicitly activate the returned approval fingerprint"
		}
		catalog.Targets = append(catalog.Targets, preview)
	}

	if binding != nil && !bindingMatched {
		catalog.Targets = append(catalog.Targets, ProjectExecutionTargetPreview{
			ID:               binding.TargetID,
			Kind:             runner.ExecutionTargetSSH,
			MachineID:        binding.MachineID,
			Transport:        "ssh",
			Source:           "project_binding",
			Effective:        true,
			Selectable:       false,
			ActivationStatus: "STALE_BINDING",
			Reason:           "the active target binding is no longer declared by nexus.yaml; deactivate or restore the target",
		})
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
		return ProjectExecutionTargetProbe{}, fmt.Errorf("%w: remote_execution feature flag is off", ErrManagedRemoteExecutionDisabled)
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
			"%w: target %q expected machine %q but remote bridge reported %q",
			ErrExecutionTargetIdentityMismatch,
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
	if probe.Ready {
		probe.ApprovalToken = projectExecutionTargetApprovalToken(project, target, hello)
	}
	if !hello.RemoteExecutionEnabled {
		probe.Reason = "remote bridge is reachable but remote_execution is disabled on the target host"
	}
	return probe, nil
}


func projectExecutionTargetApprovalToken(
	project store.Project,
	target ProjectManifestTarget,
	hello protocol.RemoteBridgeHelloData,
) string {
	capabilities := make([]string, 0, len(hello.Capabilities))
	for _, capability := range hello.Capabilities {
		capabilities = append(capabilities, string(capability))
	}
	sort.Strings(capabilities)
	fingerprint := strings.Join([]string{
		"nexus-managed-execution-target/v1",
		strings.TrimSpace(project.ID),
		strings.TrimSpace(target.ID),
		strings.ToLower(strings.TrimSpace(target.Kind)),
		strings.TrimSpace(target.Location),
		strings.TrimSpace(target.MachineID),
		strings.TrimSpace(target.Workspace),
		strings.TrimSpace(hello.MachineID),
		strings.TrimSpace(hello.OS),
		strings.TrimSpace(hello.Arch),
		strings.TrimSpace(hello.NexusVersion),
		fmt.Sprintf("%t", hello.RemoteExecutionEnabled),
		strings.Join(capabilities, ","),
	}, "\x00")
	sum := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(sum[:])
}
