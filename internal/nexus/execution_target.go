package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/registry"
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
