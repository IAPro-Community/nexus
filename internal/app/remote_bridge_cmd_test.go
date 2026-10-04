package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
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

type fakeRemoteBridgeRuntimeClient struct {
	status      protocol.StatusData
	statusErr   error
	resizeRows  int
	resizeCols  int
	resizeErr   error
	attachResp  protocol.Response
	attachErr   error
	conn        net.Conn
	reader      *bufio.Reader
	closed      bool
}

func (f *fakeRemoteBridgeRuntimeClient) Status() (protocol.StatusData, error) {
	return f.status, f.statusErr
}

func (f *fakeRemoteBridgeRuntimeClient) Resize(rows, cols int) error {
	f.resizeRows, f.resizeCols = rows, cols
	return f.resizeErr
}

func (f *fakeRemoteBridgeRuntimeClient) Send(command protocol.CommandType, _ any) (protocol.Response, error) {
	if command != protocol.CmdAttach {
		return protocol.Response{}, errors.New("unexpected fake runtime command")
	}
	return f.attachResp, f.attachErr
}

func (f *fakeRemoteBridgeRuntimeClient) ClearDeadline() error { return nil }
func (f *fakeRemoteBridgeRuntimeClient) RawConn() net.Conn { return f.conn }
func (f *fakeRemoteBridgeRuntimeClient) Reader() *bufio.Reader { return f.reader }
func (f *fakeRemoteBridgeRuntimeClient) Close() error {
	f.closed = true
	if f.conn != nil {
		return f.conn.Close()
	}
	return nil
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
			ProfileID:         "work",
			ProviderSessionID: "provider-session-42",
			Model:             "gpt-5.6",
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
		Profile:           "work",
		Model:             "gpt-5.6",
		ProviderSessionID: "provider-session-42",
		Workspace:         "/srv/project",
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
		options.ProviderSessionID != payload.ProviderSessionID ||
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


func TestRemoteBridgeStatusReturnsSanitizedRuntimeProjection(t *testing.T) {
	fakeLauncher := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(true, fakeLauncher)
	session := registry.RuntimeSession{
		RuntimeID:          "runtime-1",
		AgentID:            "agent-1",
		ProjectID:          "project-1",
		ProviderID:         "codex",
		ProfileID:          "work",
		Model:              "gpt-5.6",
		ProviderSessionID:  "provider-session-old",
		Workspace:          "/srv/project",
		PID:                100,
		ProviderGeneration: 10,
		HostPID:            101,
		HostGeneration:     11,
		MachineID:          "machine-remote",
		Transport:          "mock",
		State:              registry.StateRunning,
	}
	runtimeClient := &fakeRemoteBridgeRuntimeClient{
		status: protocol.StatusData{
			RuntimeID:          session.RuntimeID,
			ProviderID:         session.ProviderID,
			ProfileID:          session.ProfileID,
			ProviderSessionID:  "provider-session-new",
			Workspace:          session.Workspace,
			PID:                session.PID,
			ProviderGeneration: session.ProviderGeneration,
			HostPID:            session.HostPID,
			HostGeneration:     session.HostGeneration,
			State:              string(registry.StateRunning),
			ControlLevel:       string(registry.ControlLevelTerminal),
			AttentionContext:   "must-not-cross-bridge",
			HostBinaryPath:     "/secret/nexus",
			ProviderBinaryPath: "/secret/codex",
		},
	}
	server.lookup = func(id string) (registry.RuntimeSession, bool) { return session, id == session.RuntimeID }
	server.dialRuntime = func(string) (remoteBridgeRuntimeClient, error) { return runtimeClient, nil }

	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"status-1",
		protocol.CmdRemoteBridgeStatus,
		protocol.RemoteBridgeRuntimePayload{RuntimeID: session.RuntimeID},
	))
	if !response.OK {
		t.Fatalf("status failed: %+v", response)
	}
	raw := string(response.Data)
	for _, forbidden := range []string{"must-not-cross-bridge", "/secret/nexus", "/secret/codex", "attention_context", "binary_path"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("remote status leaked %q: %s", forbidden, raw)
		}
	}
	var data protocol.RemoteBridgeRuntimeStatusData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.RuntimeID != session.RuntimeID || data.MachineID != session.MachineID ||
		data.Provider != session.ProviderID || data.Profile != session.ProfileID ||
		data.Model != session.Model || data.ProviderSessionID != "provider-session-new" ||
		data.Workspace != session.Workspace ||
		data.State != string(registry.StateRunning) || data.Transport != "nexus-remote-bridge" {
		t.Fatalf("unexpected remote status projection: %+v", data)
	}
}

func TestRemoteBridgeResizeUsesVerifiedRuntimeClient(t *testing.T) {
	fakeLauncher := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(true, fakeLauncher)
	session := registry.RuntimeSession{
		RuntimeID: "runtime-1",
		ProviderID: "codex",
		ProfileID: "work",
		Workspace: "/srv/project",
		PID: 100,
		HostPID: 101,
		Transport: "mock",
		State: registry.StateRunning,
	}
	runtimeClient := &fakeRemoteBridgeRuntimeClient{
		status: protocol.StatusData{
			RuntimeID: session.RuntimeID,
			PID: session.PID,
			HostPID: session.HostPID,
			State: string(registry.StateRunning),
		},
	}
	server.lookup = func(id string) (registry.RuntimeSession, bool) { return session, id == session.RuntimeID }
	server.dialRuntime = func(string) (remoteBridgeRuntimeClient, error) { return runtimeClient, nil }

	response := bridgeResponse(t, server, bridgeRequest(
		t,
		"resize-1",
		protocol.CmdRemoteBridgeResize,
		protocol.RemoteBridgeResizePayload{RuntimeID: session.RuntimeID, Rows: 50, Cols: 160},
	))
	if !response.OK {
		t.Fatalf("resize failed: %+v", response)
	}
	if runtimeClient.resizeRows != 50 || runtimeClient.resizeCols != 160 {
		t.Fatalf("resize did not reach runtime: rows=%d cols=%d", runtimeClient.resizeRows, runtimeClient.resizeCols)
	}
}

func TestRemoteBridgeAttachSwitchesFromJSONToByteTransparentStream(t *testing.T) {
	fakeLauncher := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(true, fakeLauncher)
	session := registry.RuntimeSession{
		RuntimeID: "runtime-1",
		ProviderID: "codex",
		ProfileID: "work",
		Workspace: "/srv/project",
		PID: 100,
		HostPID: 101,
		Transport: "mock",
		State: registry.StateRunning,
	}
	bridgeConn, runtimeConn := net.Pipe()
	defer runtimeConn.Close()
	attachResponse, err := protocol.NewResponse("history-line\r\n")
	if err != nil {
		t.Fatal(err)
	}
	runtimeClient := &fakeRemoteBridgeRuntimeClient{
		status: protocol.StatusData{
			RuntimeID: session.RuntimeID,
			PID: session.PID,
			HostPID: session.HostPID,
			State: string(registry.StateRunning),
		},
		attachResp: attachResponse,
		conn: bridgeConn,
		reader: bufio.NewReader(bridgeConn),
	}
	server.lookup = func(id string) (registry.RuntimeSession, bool) { return session, id == session.RuntimeID }
	server.dialRuntime = func(string) (remoteBridgeRuntimeClient, error) { return runtimeClient, nil }

	request := bridgeRequest(
		t,
		"attach-1",
		protocol.CmdRemoteBridgeAttach,
		protocol.RemoteBridgeRuntimePayload{RuntimeID: session.RuntimeID},
	)
	rawRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	input := bytes.NewBuffer(append(append(rawRequest, '\n'), []byte("typed-input")...))
	var output bytes.Buffer

	runtimeDone := make(chan error, 1)
	go func() {
		got := make([]byte, len("typed-input"))
		if _, err := io.ReadFull(runtimeConn, got); err != nil {
			runtimeDone <- err
			return
		}
		if string(got) != "typed-input" {
			runtimeDone <- errors.New("attach input was not byte transparent")
			return
		}
		if _, err := runtimeConn.Write([]byte("provider-output")); err != nil {
			runtimeDone <- err
			return
		}
		_ = runtimeConn.Close()
		runtimeDone <- nil
	}()

	if err := server.serveAttach(context.Background(), input, &output); err != nil {
		t.Fatalf("attach bridge failed: %v", err)
	}
	if err := <-runtimeDone; err != nil {
		t.Fatal(err)
	}

	wire := output.Bytes()
	newline := bytes.IndexByte(wire, '\n')
	if newline < 0 {
		t.Fatalf("attach did not emit JSON acknowledgement: %q", wire)
	}
	var response protocol.Response
	if err := json.Unmarshal(wire[:newline], &response); err != nil {
		t.Fatalf("decode attach acknowledgement: %v", err)
	}
	if !response.OK || response.RuntimeID != session.RuntimeID {
		t.Fatalf("unexpected attach acknowledgement: %+v", response)
	}
	var data protocol.RemoteBridgeAttachData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.History != "history-line\r\n" {
		t.Fatalf("history=%q", data.History)
	}
	if got := string(wire[newline+1:]); got != "provider-output" {
		t.Fatalf("live provider stream=%q", got)
	}
}

func TestRemoteBridgeAttachRejectsIdentityDriftBeforeStreaming(t *testing.T) {
	fakeLauncher := &fakeRemoteBridgeLauncher{}
	server := testRemoteBridgeServer(true, fakeLauncher)
	session := registry.RuntimeSession{
		RuntimeID: "runtime-1",
		ProviderID: "codex",
		ProfileID: "work",
		Workspace: "/srv/project",
		PID: 100,
		HostPID: 101,
		Transport: "mock",
		State: registry.StateRunning,
	}
	runtimeClient := &fakeRemoteBridgeRuntimeClient{
		status: protocol.StatusData{
			RuntimeID: session.RuntimeID,
			PID: 999,
			HostPID: session.HostPID,
			State: string(registry.StateRunning),
		},
	}
	server.lookup = func(id string) (registry.RuntimeSession, bool) { return session, id == session.RuntimeID }
	server.dialRuntime = func(string) (remoteBridgeRuntimeClient, error) { return runtimeClient, nil }

	request := bridgeRequest(
		t,
		"attach-drift",
		protocol.CmdRemoteBridgeAttach,
		protocol.RemoteBridgeRuntimePayload{RuntimeID: session.RuntimeID},
	)
	rawRequest, _ := json.Marshal(request)
	var output bytes.Buffer
	err := server.serveAttach(context.Background(), bytes.NewReader(append(rawRequest, '\n')), &output)
	if err == nil {
		t.Fatal("identity drift must reject attach")
	}
	if strings.Contains(output.String(), "provider-output") {
		t.Fatal("attach streamed bytes after identity rejection")
	}
}
