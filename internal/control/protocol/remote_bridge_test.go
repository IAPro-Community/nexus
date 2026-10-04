package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func fullRemoteBridgeCapabilities() []RemoteBridgeCapability {
	return []RemoteBridgeCapability{
		RemoteBridgeCapabilitySpawn,
		RemoteBridgeCapabilityCWD,
		RemoteBridgeCapabilityPTY,
		RemoteBridgeCapabilityResize,
		RemoteBridgeCapabilityLiveness,
	}
}

func TestValidateRemoteBridgeHandshake(t *testing.T) {
	hello := RemoteBridgeHelloPayload{ProtocolVersion: ProtocolVersion, TargetID: "devbox"}
	if err := ValidateRemoteBridgeHello(hello); err != nil {
		t.Fatalf("valid hello rejected: %v", err)
	}

	data := RemoteBridgeHelloData{
		ProtocolVersion: ProtocolVersion,
		MachineID:       "machine-devbox",
		OS:              "linux",
		Arch:            "amd64",
		NexusVersion:    "1.1.0-beta",
		Capabilities:    fullRemoteBridgeCapabilities(),
	}
	if err := ValidateRemoteBridgeHelloData(data); err != nil {
		t.Fatalf("valid hello response rejected: %v", err)
	}

	badVersion := hello
	badVersion.ProtocolVersion++
	if err := ValidateRemoteBridgeHello(badVersion); err == nil {
		t.Fatal("protocol version mismatch must fail closed")
	}

	missingPTY := data
	missingPTY.Capabilities = []RemoteBridgeCapability{
		RemoteBridgeCapabilitySpawn,
		RemoteBridgeCapabilityCWD,
		RemoteBridgeCapabilityResize,
		RemoteBridgeCapabilityLiveness,
	}
	if err := ValidateRemoteBridgeHelloData(missingPTY); err == nil || !strings.Contains(err.Error(), "PTY") {
		t.Fatalf("missing PTY capability must be rejected, got %v", err)
	}

	missingVersion := data
	missingVersion.NexusVersion = ""
	if err := ValidateRemoteBridgeHelloData(missingVersion); err == nil {
		t.Fatal("remote Nexus version is required")
	}
}

func TestValidateRemoteBridgeSpawnKeepsArgvStructured(t *testing.T) {
	payload := RemoteBridgeSpawnPayload{
		RuntimeID: "runtime_123",
		AgentID:   "agent_123",
		ProjectID: "project_123",
		Provider:  "codex",
		Profile:           "work@example.com",
		ProviderSessionID: "provider-session-42",
		Workspace:         "/srv/project with spaces",
		Args: []string{
			"--prompt",
			"literal ; $(echo this-is-data) && not-a-shell-command",
		},
		Rows: 40,
		Cols: 120,
	}
	if err := ValidateRemoteBridgeSpawn(payload); err != nil {
		t.Fatalf("structured argv with shell-looking data must remain valid: %v", err)
	}

	req, err := NewRequest(CmdRemoteBridgeSpawn, payload)
	if err != nil {
		t.Fatal(err)
	}
	if req.Command != CmdRemoteBridgeSpawn {
		t.Fatalf("command=%q", req.Command)
	}
	var decoded RemoteBridgeSpawnPayload
	if err := json.Unmarshal(req.Payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Args) != 2 || decoded.Args[1] != payload.Args[1] {
		t.Fatalf("argv was not preserved as structured data: %#v", decoded.Args)
	}
	if decoded.ProviderSessionID != payload.ProviderSessionID {
		t.Fatalf("provider session id was not preserved: %q", decoded.ProviderSessionID)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"password", "private_key", "token", "environment", "command_template"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("spawn envelope unexpectedly contains credential/shell field %q: %s", forbidden, raw)
		}
	}
}

func TestValidateRemoteBridgeSpawnRejectsMalformedPayload(t *testing.T) {
	base := RemoteBridgeSpawnPayload{
		RuntimeID: "runtime_123",
		AgentID:   "agent_123",
		ProjectID: "project_123",
		Provider:  "codex",
		Profile:   "default",
		Workspace: "/srv/project",
	}

	tests := []struct {
		name   string
		mutate func(*RemoteBridgeSpawnPayload)
	}{
		{
			name: "invalid runtime id",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.RuntimeID = "runtime id"
			},
		},
		{
			name: "workspace control characters",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.Workspace = "/srv/project\nother"
			},
		},
		{
			name: "provider session control characters",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.ProviderSessionID = "session\nother"
			},
		},
		{
			name: "nul argument",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.Args = []string{"ok", "bad\x00arg"}
			},
		},
		{
			name: "too many arguments",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.Args = make([]string, maxRemoteBridgeArgs+1)
			},
		},
		{
			name: "oversized argument",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.Args = []string{strings.Repeat("x", maxRemoteBridgeArgBytes+1)}
			},
		},
		{
			name: "invalid terminal dimensions",
			mutate: func(payload *RemoteBridgeSpawnPayload) {
				payload.Rows = 1001
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := base
			tc.mutate(&payload)
			if err := ValidateRemoteBridgeSpawn(payload); err == nil {
				t.Fatal("malformed remote bridge spawn was accepted")
			}
		})
	}
}


func TestValidateRemoteBridgeRuntimeAndResize(t *testing.T) {
	if err := ValidateRemoteBridgeRuntime(RemoteBridgeRuntimePayload{RuntimeID: "runtime-1"}); err != nil {
		t.Fatalf("valid runtime reference rejected: %v", err)
	}
	for _, runtimeID := range []string{"", "runtime id", "-runtime"} {
		if err := ValidateRemoteBridgeRuntime(RemoteBridgeRuntimePayload{RuntimeID: runtimeID}); err == nil {
			t.Fatalf("invalid runtime id %q was accepted", runtimeID)
		}
	}
	if err := ValidateRemoteBridgeResize(RemoteBridgeResizePayload{
		RuntimeID: "runtime-1",
		Rows:      40,
		Cols:      120,
	}); err != nil {
		t.Fatalf("valid resize rejected: %v", err)
	}
	for _, payload := range []RemoteBridgeResizePayload{
		{RuntimeID: "runtime-1", Rows: 0, Cols: 120},
		{RuntimeID: "runtime-1", Rows: 40, Cols: 0},
		{RuntimeID: "runtime-1", Rows: 1001, Cols: 120},
		{RuntimeID: "runtime-1", Rows: 40, Cols: 1001},
	} {
		if err := ValidateRemoteBridgeResize(payload); err == nil {
			t.Fatalf("invalid resize was accepted: %+v", payload)
		}
	}
}
