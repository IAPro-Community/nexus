package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

type fakeRemoteBridgeLauncher struct {
	calls   int
	options launcher.LaunchOptions
	session *registry.RuntimeSession
	err     error
}

func (f *fakeRemoteBridgeLauncher) Launch(_ context.Context, options launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	f.calls++
	f.options = options
	if f.err != nil {
		return nil, f.err
	}
	return f.session, nil
}

func testRemoteBridgeServer(enabled bool, fake *fakeRemoteBridgeLauncher) remoteBridgeServer {
	return remoteBridgeServer{
		launcher:      fake,
		machineID:     func() string { return "machine-test" },
		remoteEnabled: func() bool { return enabled },
		nexusVersion:  "1.1.0-test",
		osName:        "linux",
		arch:          "amd64",
		lookup:        func(string) (registry.RuntimeSession, bool) { return registry.RuntimeSession{}, false },
	}
}

func bridgeRequest(t *testing.T, id string, command protocol.CommandType, payload any) protocol.Request {
	t.Helper()
	request, err := protocol.NewRequest(command, payload)
	if err != nil {
		t.Fatal(err)
	}
	request.ID = id
	return request
}

func bridgeResponse(t *testing.T, server remoteBridgeServer, request protocol.Request) protocol.Response {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	var output bytes.Buffer
	if err := server.serve(context.Background(), bytes.NewReader(raw), &output); err != nil {
		t.Fatal(err)
	}
	var response protocol.Response
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
		t.Fatalf("decode bridge response: %v; raw=%q", err, output.String())
	}
	return response
}

func TestRemoteBridgeHelloReportsCapabilityAndActivation(t *testing.T) {
	fake := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(false, fake)
	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"hello-1",
		protocol.CmdRemoteBridgeHello,
		protocol.RemoteBridgeHelloPayload{ProtocolVersion: protocol.ProtocolVersion, TargetID: "devbox"},
	))
	if !response.OK || response.ID != "hello-1" {
		t.Fatalf("unexpected hello response: %+v", response)
	}
	if fake.calls != 0 {
		t.Fatal("hello must not launch a runtime")
	}
	var data protocol.RemoteBridgeHelloData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.MachineID != "machine-test" || data.OS != "linux" || data.Arch != "amd64" || data.NexusVersion != "1.1.0-test" {
		t.Fatalf("unexpected hello data: %+v", data)
	}
	if data.RemoteExecutionEnabled {
		t.Fatal("hello must report remote_execution disabled")
	}
	if err := protocol.ValidateRemoteBridgeHelloData(data); err != nil {
		t.Fatalf("hello data must satisfy bridge contract: %v", err)
	}
}

func TestRemoteBridgeSpawnFailsClosedWhenFeatureDisabled(t *testing.T) {
	fake := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(false, fake)
	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"spawn-off",
		protocol.CmdRemoteBridgeSpawn,
		protocol.RemoteBridgeSpawnPayload{
			RuntimeID: "runtime-1",
			AgentID:   "agent-1",
			ProjectID: "project-1",
			Provider:  "codex",
			Profile:   "work",
			Workspace: "/srv/project",
		},
	))
	if response.OK || response.Code != "REMOTE_EXECUTION_DISABLED" {
		t.Fatalf("disabled spawn must fail closed: %+v", response)
	}
	if fake.calls != 0 {
		t.Fatal("disabled bridge must not invoke launcher")
	}
}

func TestRemoteBridgeSpawnMapsOnlyStructuredLaunchFields(t *testing.T) {
	fake := &fakeRemoteBridgeLauncher{
		session: &registry.RuntimeSession{
			RuntimeID: "runtime-1",
			AgentID:   "agent-1",
			ProjectID: "project-1",
			ProviderID: "codex",
			ProfileID: "work",
			Model: "gpt-5.6",
			Workspace: "/srv/project",
			PID: 100,
			HostPID: 101,
			MachineID: "machine-remote",
			State: registry.StateRunning,
		},
	}
	server := testRemoteBridgeServer(true, fake)
	var resizedRuntime string
	var resizedRows, resizedCols int
	server.resize = func(runtimeID string, rows, cols int) error {
		resizedRuntime, resizedRows, resizedCols = runtimeID, rows, cols
		return nil
	}
	payload := protocol.RemoteBridgeSpawnPayload{
		RuntimeID: "runtime-1",
		AgentID:   "agent-1",
		ProjectID: "project-1",
		Provider:  "codex",
		Profile:   "work",
		Model:     "gpt-5.6",
		Workspace: "/srv/project",
		Args: []string{
			"--prompt",
			"literal ; $(echo data) && still-one-argument",
		},
		Rows: 40,
		Cols: 120,
	}
	response := bridgeResponse(t, server, bridgeRequest(t, "spawn-1", protocol.CmdRemoteBridgeSpawn, payload))
	if !response.OK || response.ID != "spawn-1" || response.RuntimeID != "runtime-1" {
		t.Fatalf("unexpected spawn response: %+v", response)
	}
	if fake.calls != 1 {
		t.Fatalf("launcher calls=%d want=1", fake.calls)
	}
	options := fake.options
	if options.RuntimeID != payload.RuntimeID ||
		options.AgentID != payload.AgentID ||
		options.ProjectID != payload.ProjectID ||
		options.ProviderID != payload.Provider ||
		options.ProfileID != payload.Profile ||
		options.Model != payload.Model ||
		options.Workspace != payload.Workspace {
		t.Fatalf("launch options lost routing identity: %+v", options)
	}
	if len(options.Args) != 2 || options.Args[1] != payload.Args[1] {
		t.Fatalf("argv was not preserved: %#v", options.Args)
	}
	if options.Environment != nil || options.Options != nil || options.PathPrepend != nil {
		t.Fatalf("remote payload must not populate mutable environment/options: %+v", options)
	}
	if resizedRuntime != "runtime-1" || resizedRows != 40 || resizedCols != 120 {
		t.Fatalf("initial resize was not applied: runtime=%q rows=%d cols=%d", resizedRuntime, resizedRows, resizedCols)
	}
	var data protocol.RemoteBridgeSpawnData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.MachineID != "machine-remote" || data.HostPID != 101 || data.ProviderPID != 100 || data.Transport != "nexus-remote-bridge" {
		t.Fatalf("unexpected spawn data: %+v", data)
	}
}

func TestRemoteBridgeSpawnIsIdempotentForMatchingLiveRuntime(t *testing.T) {
	fake := &fakeRemoteBridgeLauncher{}
	existing := registry.RuntimeSession{
		RuntimeID: "runtime-1",
		AgentID: "agent-1",
		ProjectID: "project-1",
		ProviderID: "codex",
		ProfileID: "work",
		Model: "gpt-5.6",
		Workspace: "/srv/project",
		PID: 100,
		HostPID: 101,
		MachineID: "machine-remote",
		Transport: "mock",
		State: registry.StateRunning,
	}
	server := testRemoteBridgeServer(true, fake)
	server.lookup = func(id string) (registry.RuntimeSession, bool) {
		return existing, id == existing.RuntimeID
	}
	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"spawn-repeat",
		protocol.CmdRemoteBridgeSpawn,
		protocol.RemoteBridgeSpawnPayload{
			RuntimeID: existing.RuntimeID,
			AgentID: existing.AgentID,
			ProjectID: existing.ProjectID,
			Provider: existing.ProviderID,
			Profile: existing.ProfileID,
			Model: existing.Model,
			Workspace: existing.Workspace,
		},
	))
	if !response.OK {
		t.Fatalf("matching live runtime should be reused: %+v", response)
	}
	if fake.calls != 0 {
		t.Fatal("idempotent replay must not launch a second provider")
	}
}

func TestRemoteBridgeSpawnRejectsRuntimeIdentityConflict(t *testing.T) {
	fake := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(true, fake)
	server.lookup = func(string) (registry.RuntimeSession, bool) {
		return registry.RuntimeSession{
			RuntimeID: "runtime-1",
			AgentID: "agent-1",
			ProjectID: "project-1",
			ProviderID: "codex",
			ProfileID: "other-profile",
			Workspace: "/srv/project",
			Transport: "mock",
			State: registry.StateRunning,
		}, true
	}
	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"spawn-conflict",
		protocol.CmdRemoteBridgeSpawn,
		protocol.RemoteBridgeSpawnPayload{
			RuntimeID: "runtime-1",
			AgentID: "agent-1",
			ProjectID: "project-1",
			Provider: "codex",
			Profile: "work",
			Workspace: "/srv/project",
		},
	))
	if response.OK || response.Code != "RUNTIME_ID_CONFLICT" {
		t.Fatalf("runtime identity conflict must fail closed: %+v", response)
	}
	if fake.calls != 0 {
		t.Fatal("conflicting runtime id must not launch")
	}
}

func TestRemoteBridgeRejectsUnknownPayloadFieldsAndOversizedFrames(t *testing.T) {
	fake := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(true, fake)

	request := protocol.Request{
		Version: protocol.ProtocolVersion,
		ID: "strict-1",
		Command: protocol.CmdRemoteBridgeSpawn,
		Payload: json.RawMessage(`{"runtime_id":"runtime-1","agent_id":"agent-1","project_id":"project-1","provider":"codex","profile":"work","workspace":"/srv/project","command_template":"bad"}`),
	}
	response := bridgeResponse(t, server, request)
	if response.OK || response.Code != "INVALID_REQUEST" {
		t.Fatalf("unknown payload field must fail closed: %+v", response)
	}
	if fake.calls != 0 {
		t.Fatal("invalid payload must not launch")
	}

	oversized := strings.Repeat("x", remoteBridgeMaxFrameBytes+1) + "\n"
	var output bytes.Buffer
	if err := server.serve(context.Background(), strings.NewReader(oversized), &output); err == nil {
		t.Fatal("oversized bridge frame must terminate the transport")
	}
}

func TestRemoteBridgeLaunchErrorsAreRedacted(t *testing.T) {
	secret := "sk-proj-1234567890abcdef1234567890abcdef1234567890"
	fake := &fakeRemoteBridgeLauncher{err: errors.New("OPENAI_API_KEY=" + secret)}
	server := testRemoteBridgeServer(true, fake)
	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"spawn-redacted",
		protocol.CmdRemoteBridgeSpawn,
		protocol.RemoteBridgeSpawnPayload{
			RuntimeID: "runtime-1",
			AgentID: "agent-1",
			ProjectID: "project-1",
			Provider: "codex",
			Profile: "work",
			Workspace: "/srv/project",
		},
	))
	if response.OK || response.Code != "LAUNCH_FAILED" {
		t.Fatalf("launch failure must be structured: %+v", response)
	}
	if strings.Contains(response.Error, secret) || strings.Contains(response.Message, secret) {
		t.Fatalf("bridge error leaked secret: %+v", response)
	}
}
