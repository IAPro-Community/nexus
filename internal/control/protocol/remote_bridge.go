package protocol

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	CmdRemoteBridgeHello CommandType = "remote_bridge_hello"
	CmdRemoteBridgeSpawn CommandType = "remote_bridge_spawn"
)

type RemoteBridgeCapability string

const (
	RemoteBridgeCapabilitySpawn    RemoteBridgeCapability = "SPAWN"
	RemoteBridgeCapabilityCWD      RemoteBridgeCapability = "CWD"
	RemoteBridgeCapabilityPTY      RemoteBridgeCapability = "PTY"
	RemoteBridgeCapabilityResize   RemoteBridgeCapability = "RESIZE"
	RemoteBridgeCapabilityLiveness RemoteBridgeCapability = "LIVENESS"
)

const (
	maxRemoteBridgeIdentifierBytes = 240
	maxRemoteBridgeWorkspaceBytes  = 4096
	maxRemoteBridgeArgs            = 256
	maxRemoteBridgeArgBytes        = 8192
	maxRemoteBridgeArgsBytes       = 64 * 1024
)

var remoteBridgeIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@/+\\-]{0,239}$`)

// RemoteBridgeHelloPayload starts a structured remote-runtime handshake over a
// transport such as SSH stdio. The transport authenticates the host; this
// payload negotiates Nexus protocol/capabilities only.
type RemoteBridgeHelloPayload struct {
	ProtocolVersion int    `json:"protocol_version"`
	TargetID        string `json:"target_id"`
}

// RemoteBridgeHelloData is returned by the remote Nexus runtime bridge.
// Provider credentials remain owned by the remote host and are never returned.
type RemoteBridgeHelloData struct {
	ProtocolVersion int                      `json:"protocol_version"`
	MachineID       string                   `json:"machine_id"`
	OS              string                   `json:"os"`
	Arch            string                   `json:"arch"`
	NexusVersion           string                   `json:"nexus_version"`
	RemoteExecutionEnabled bool                     `json:"remote_execution_enabled"`
	Capabilities           []RemoteBridgeCapability `json:"capabilities"`
}

// RemoteBridgeSpawnPayload asks the remote Nexus runtime bridge to start one
// supervised provider runtime. Workspace and arguments remain structured data:
// implementations must never concatenate them into shell text.
//
// Authentication/profile material is resolved on the remote host. There is
// deliberately no password, private-key, token, arbitrary environment map or
// command-template field in this envelope.
type RemoteBridgeSpawnPayload struct {
	RuntimeID string   `json:"runtime_id"`
	AgentID   string   `json:"agent_id"`
	ProjectID string   `json:"project_id"`
	Provider  string   `json:"provider"`
	Profile   string   `json:"profile"`
	Model     string   `json:"model,omitempty"`
	Workspace string   `json:"workspace"`
	Args      []string `json:"args,omitempty"`
	Rows      int      `json:"rows,omitempty"`
	Cols      int      `json:"cols,omitempty"`
}

// RemoteBridgeSpawnData is the durable acknowledgement from the remote bridge.
// HostPID belongs to the remote Nexus supervisor, while ProviderPID belongs to
// the provider child and must not be used as a cross-host kill authority.
type RemoteBridgeSpawnData struct {
	RuntimeID   string `json:"runtime_id"`
	MachineID   string `json:"machine_id"`
	HostPID     int    `json:"host_pid,omitempty"`
	ProviderPID int    `json:"provider_pid,omitempty"`
	Transport   string `json:"transport"`
}

func ValidateRemoteBridgeHello(payload RemoteBridgeHelloPayload) error {
	if payload.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("remote bridge protocol version %d is unsupported", payload.ProtocolVersion)
	}
	if err := validateRemoteBridgeIdentifier("target_id", payload.TargetID); err != nil {
		return err
	}
	return nil
}

func ValidateRemoteBridgeHelloData(data RemoteBridgeHelloData) error {
	if data.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("remote bridge protocol version %d is unsupported", data.ProtocolVersion)
	}
	if err := validateRemoteBridgeIdentifier("machine_id", data.MachineID); err != nil {
		return err
	}
	if err := validateRemoteBridgeMetadata("os", data.OS, 64); err != nil {
		return err
	}
	if err := validateRemoteBridgeMetadata("arch", data.Arch, 64); err != nil {
		return err
	}
	if err := validateRemoteBridgeMetadata("nexus_version", data.NexusVersion, 128); err != nil {
		return err
	}
	required := map[RemoteBridgeCapability]bool{
		RemoteBridgeCapabilitySpawn:    false,
		RemoteBridgeCapabilityCWD:      false,
		RemoteBridgeCapabilityPTY:      false,
		RemoteBridgeCapabilityResize:   false,
		RemoteBridgeCapabilityLiveness: false,
	}
	for _, capability := range data.Capabilities {
		if _, tracked := required[capability]; tracked {
			required[capability] = true
		}
	}
	for capability, present := range required {
		if !present {
			return fmt.Errorf("remote bridge is missing required capability %s", capability)
		}
	}
	return nil
}

func ValidateRemoteBridgeSpawn(payload RemoteBridgeSpawnPayload) error {
	for name, value := range map[string]string{
		"runtime_id": payload.RuntimeID,
		"agent_id":   payload.AgentID,
		"project_id": payload.ProjectID,
		"provider":   payload.Provider,
		"profile":    payload.Profile,
	} {
		if err := validateRemoteBridgeIdentifier(name, value); err != nil {
			return err
		}
	}
	if payload.Model != "" {
		if strings.TrimSpace(payload.Model) != payload.Model || len(payload.Model) > maxRemoteBridgeIdentifierBytes || strings.ContainsAny(payload.Model, "\x00\r\n") {
			return fmt.Errorf("remote bridge model is invalid")
		}
	}
	if strings.TrimSpace(payload.Workspace) == "" || len(payload.Workspace) > maxRemoteBridgeWorkspaceBytes {
		return fmt.Errorf("remote bridge workspace is required and must not exceed %d bytes", maxRemoteBridgeWorkspaceBytes)
	}
	if strings.ContainsAny(payload.Workspace, "\x00\r\n") {
		return fmt.Errorf("remote bridge workspace contains control characters")
	}
	if len(payload.Args) > maxRemoteBridgeArgs {
		return fmt.Errorf("remote bridge argument list exceeds %d entries", maxRemoteBridgeArgs)
	}
	total := 0
	for index, arg := range payload.Args {
		if len(arg) > maxRemoteBridgeArgBytes {
			return fmt.Errorf("remote bridge argument %d exceeds %d bytes", index, maxRemoteBridgeArgBytes)
		}
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("remote bridge argument %d contains NUL", index)
		}
		total += len(arg)
		if total > maxRemoteBridgeArgsBytes {
			return fmt.Errorf("remote bridge argument payload exceeds %d bytes", maxRemoteBridgeArgsBytes)
		}
	}
	if payload.Rows < 0 || payload.Rows > 1000 || payload.Cols < 0 || payload.Cols > 1000 {
		return fmt.Errorf("remote bridge terminal dimensions are invalid")
	}
	return nil
}

func validateRemoteBridgeMetadata(name, value string, maxBytes int) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxBytes || strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("remote bridge %s is invalid", name)
	}
	return nil
}

func validateRemoteBridgeIdentifier(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxRemoteBridgeIdentifierBytes || !remoteBridgeIdentifierPattern.MatchString(value) {
		return fmt.Errorf("remote bridge %s is invalid", name)
	}
	return nil
}
