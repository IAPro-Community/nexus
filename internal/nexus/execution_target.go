package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
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
