package remote

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

type fakeManagedBridge struct {
	mu          sync.Mutex
	status      protocol.RemoteBridgeRuntimeStatusData
	statusErr   error
	resizeRows  int
	resizeCols  int
	controls    []protocol.CommandType
	controlData map[protocol.CommandType]any
	controlErr  error
	closed      bool
}

func (f *fakeManagedBridge) Status(context.Context, string) (protocol.RemoteBridgeRuntimeStatusData, error) {
	return f.status, f.statusErr
}

func (f *fakeManagedBridge) Resize(_ context.Context, _ string, rows, cols int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resizeRows, f.resizeCols = rows, cols
	return nil
}

func (f *fakeManagedBridge) Control(_ context.Context, runtimeID string, command protocol.CommandType, payload any) (protocol.Response, error) {
	f.mu.Lock()
	f.controls = append(f.controls, command)
	f.mu.Unlock()
	if f.controlErr != nil {
		return protocol.Response{}, f.controlErr
	}
	data := any("ok")
	if f.controlData != nil {
		if configured, ok := f.controlData[command]; ok {
			data = configured
		}
	}
	response, err := protocol.NewResponse(data)
	if err != nil {
		return protocol.Response{}, err
	}
	response.RuntimeID = runtimeID
	response.Action = string(command)
	if command == protocol.CmdStop {
		response.Code = "STOP_REQUESTED"
		response.State = string(registry.StateStopping)
	}
	return response, nil
}

func (f *fakeManagedBridge) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func newProxyTestSession(runtimeID string) registry.RuntimeSession {
	return registry.RuntimeSession{
		RuntimeID:  runtimeID,
		AgentID:    "agent-1",
		ProjectID:  "project-1",
		ProviderID: "codex",
		ProfileID:  "work",
		Model:      "gpt-5.6",
		Workspace:  "/srv/project",
		PID:        424242, // remote provider PID only
		MachineID:  "machine-remote",
		State:      registry.StateRunning,
		StartedAt:  time.Now().UTC(),
	}
}

func startTestManagedProxy(
	t *testing.T,
	bridge *fakeManagedBridge,
	attach managedAttachOpener,
) (*ManagedRuntimeProxy, registry.RuntimeSession, *registry.Registry) {
	t.Helper()
	runtimeID := "remote-proxy-" + strings.ReplaceAll(t.Name(), "/", "-")
	reg := registry.NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	proxy, session, err := StartManagedRuntimeProxy(context.Background(), ManagedRuntimeProxyConfig{
		Session:      newProxyTestSession(runtimeID),
		TargetID:     "devbox",
		Location:     "devbox",
		Bridge:       bridge,
		Registry:     reg,
		AttachOpener: attach,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })
	return proxy, session, reg
}

func TestManagedRuntimeProxyLooksLikeLocalProtocolRuntime(t *testing.T) {
	bridge := &fakeManagedBridge{
		status: protocol.RemoteBridgeRuntimeStatusData{
			RuntimeID:    "ignored-by-proxy",
			MachineID:    "machine-remote",
			Provider:     "codex",
			Profile:      "work",
			Model:        "gpt-5.6",
			Workspace:    "/srv/project",
			State:        string(registry.StateRunning),
			ControlLevel: string(registry.ControlLevelTerminal),
			Transport:    "nexus-remote-bridge",
		},
		controlData: map[protocol.CommandType]any{
			protocol.CmdPing: "pong",
			protocol.CmdUsage: protocol.UsageData{
				RuntimeID:   "remote",
				ProviderID:  "codex",
				ProfileID:   "work",
				Status:      "AVAILABLE",
				PercentLeft: 75,
			},
			protocol.CmdEvents: []protocol.EventData{},
		},
	}
	_, session, reg := startTestManagedProxy(t, bridge, nil)

	persisted, ok := reg.Get(session.RuntimeID)
	if !ok {
		t.Fatal("proxy runtime was not registered")
	}
	if persisted.Transport != registry.TransportManagedSSHProxy ||
		persisted.MachineID != "machine-remote" ||
		persisted.Location != "devbox" ||
		persisted.HostPID <= 0 ||
		persisted.HostBinaryFingerprint == "" ||
		persisted.ProviderBinaryPath != "" {
		t.Fatalf("unexpected proxy registry session: %+v", persisted)
	}
	if err := registry.ValidateAttachCompatibility(persisted); err != nil {
		t.Fatalf("proxy registry session is not attach-compatible: %v", err)
	}

	client, err := protocol.NewClient(session.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Fatalf("proxy ping failed: %v", err)
	}
	status, err := client.Status()
	if err != nil {
		t.Fatalf("proxy status failed: %v", err)
	}
	if status.RuntimeID != session.RuntimeID ||
		status.ProviderID != "codex" ||
		status.ProfileID != "work" ||
		status.Workspace != "/srv/project" ||
		status.PID != 424242 ||
		status.HostPID != persisted.HostPID ||
		status.ControlLevel != string(registry.ControlLevelTerminal) {
		t.Fatalf("unexpected local status projection: %+v", status)
	}
	if err := client.Resize(55, 180); err != nil {
		t.Fatalf("proxy resize failed: %v", err)
	}
	bridge.mu.Lock()
	rows, cols := bridge.resizeRows, bridge.resizeCols
	bridge.mu.Unlock()
	if rows != 55 || cols != 180 {
		t.Fatalf("resize did not reach bridge: rows=%d cols=%d", rows, cols)
	}

	if err := client.SubmitPrompt("literal prompt"); err != nil {
		t.Fatalf("submit prompt did not traverse proxy: %v", err)
	}
	events, err := client.Events(10)
	if err != nil || events == nil {
		t.Fatalf("events did not traverse proxy: events=%v err=%v", events, err)
	}
	usage, err := client.Usage()
	if err != nil {
		t.Fatalf("usage did not traverse proxy: %v", err)
	}
	if usage.Status != "AVAILABLE" || usage.PercentLeft != 75 {
		t.Fatalf("unexpected usage through proxy: %+v", usage)
	}
}

func TestManagedRuntimeProxyAttachPreservesHistoryAndRawBytes(t *testing.T) {
	bridge := &fakeManagedBridge{
		status: protocol.RemoteBridgeRuntimeStatusData{
			State:        string(registry.StateRunning),
			ControlLevel: string(registry.ControlLevelTerminal),
		},
	}
	remoteBridgeSide, remoteProviderSide := net.Pipe()
	attach := func(_ context.Context, location, runtimeID string) (*managedAttachStream, error) {
		if location != "devbox" || runtimeID == "" {
			return nil, errors.New("unexpected attach target")
		}
		return &managedAttachStream{
			History: "history\r\n",
			Reader:  remoteBridgeSide,
			Writer:  remoteBridgeSide,
			Close:   remoteBridgeSide.Close,
		}, nil
	}
	_, session, _ := startTestManagedProxy(t, bridge, attach)

	client, err := protocol.NewClient(session.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	response, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ClearDeadline(); err != nil {
		t.Fatal(err)
	}
	var history string
	if err := json.Unmarshal(response.Data, &history); err != nil {
		t.Fatal(err)
	}
	if history != "history\r\n" {
		t.Fatalf("history=%q", history)
	}

	raw := client.RawConn()
	reader := client.Reader()
	if raw == nil || reader == nil {
		t.Fatal("attached proxy did not expose raw stream")
	}

	remoteDone := make(chan error, 1)
	go func() {
		remoteReader := bufio.NewReader(remoteProviderSide)
		line, err := remoteReader.ReadString('\n')
		if err != nil {
			remoteDone <- err
			return
		}
		var lease protocol.Request
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &lease); err != nil {
			remoteDone <- err
			return
		}
		if lease.Command != protocol.CmdLeaseAcquire {
			remoteDone <- errors.New("lease command was not forwarded byte-for-byte")
			return
		}
		if _, err := remoteProviderSide.Write([]byte("provider-output")); err != nil {
			remoteDone <- err
			return
		}
		remoteDone <- nil
	}()

	lease, err := protocol.NewRequest(protocol.CmdLeaseAcquire, nil)
	if err != nil {
		t.Fatal(err)
	}
	leaseRaw, _ := json.Marshal(lease)
	if _, err := raw.Write(append(leaseRaw, '\n')); err != nil {
		t.Fatal(err)
	}

	output := make([]byte, len("provider-output"))
	if _, err := io.ReadFull(reader, output); err != nil {
		t.Fatal(err)
	}
	if string(output) != "provider-output" {
		t.Fatalf("provider output=%q", output)
	}
	if err := <-remoteDone; err != nil {
		t.Fatal(err)
	}
	_ = remoteProviderSide.Close()
}

func TestManagedRuntimeProxyBlocksCrossMachineContinuityControls(t *testing.T) {
	bridge := &fakeManagedBridge{
		status: protocol.RemoteBridgeRuntimeStatusData{
			State:        string(registry.StateRunning),
			ControlLevel: string(registry.ControlLevelTerminal),
		},
	}
	_, session, _ := startTestManagedProxy(t, bridge, nil)

	client, err := protocol.NewClient(session.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_, err = client.Send(protocol.CmdHandoff, protocol.HandoffPayload{TargetProfile: "other"})
	if err == nil || !strings.Contains(err.Error(), "cross-machine") {
		t.Fatalf("remote handoff must fail closed, got %v", err)
	}
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	for _, command := range bridge.controls {
		if command == protocol.CmdHandoff {
			t.Fatal("blocked handoff reached remote bridge")
		}
	}
}

func TestManagedRuntimeProxyStopClosesEndpointAndMarksStopped(t *testing.T) {
	bridge := &fakeManagedBridge{
		status: protocol.RemoteBridgeRuntimeStatusData{
			State:        string(registry.StateRunning),
			ControlLevel: string(registry.ControlLevelTerminal),
		},
	}
	_, session, reg := startTestManagedProxy(t, bridge, nil)

	client, err := protocol.NewClient(session.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Stop(); err != nil {
		t.Fatalf("remote stop failed: %v", err)
	}
	_ = client.Close()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		persisted, ok := reg.Get(session.RuntimeID)
		if ok && persisted.State == registry.StateStopped {
			if _, err := protocol.NewClient(session.RuntimeID); err == nil {
				t.Fatal("proxy endpoint remained connectable after stop")
			}
			bridge.mu.Lock()
			closed := bridge.closed
			bridge.mu.Unlock()
			if !closed {
				t.Fatal("SSH control bridge remained open after proxy stop")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("proxy did not transition to STOPPED")
}

func TestManagedRuntimeProxyRedactsBridgeErrors(t *testing.T) {
	secret := "sk-proj-1234567890abcdef1234567890abcdef1234567890"
	bridge := &fakeManagedBridge{
		statusErr: errors.New("OPENAI_API_KEY=" + secret),
	}
	_, session, _ := startTestManagedProxy(t, bridge, nil)

	client, err := protocol.NewClient(session.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_, err = client.Status()
	if err == nil {
		t.Fatal("bridge status error expected")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("proxy leaked secret: %v", err)
	}
}
