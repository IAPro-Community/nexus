package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type managedStartRouteLauncher struct {
	mu          sync.Mutex
	launchCalls int
	stopCalls   int
}

func (l *managedStartRouteLauncher) Launch(context.Context, launcher.LaunchOptions) (*registry.RuntimeSession, error) {
	l.mu.Lock()
	l.launchCalls++
	l.mu.Unlock()
	return nil, fmt.Errorf("local launcher must not be used for an approved managed target")
}

func (l *managedStartRouteLauncher) Stop(runtimeID string) error {
	l.mu.Lock()
	l.stopCalls++
	l.mu.Unlock()
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Stop()
}

type managedStartBridge struct {
	mu       sync.Mutex
	spawn    protocol.RemoteBridgeSpawnPayload
	controls []protocol.CommandType
	closed   bool
}

func (b *managedStartBridge) Spawn(
	_ context.Context,
	payload protocol.RemoteBridgeSpawnPayload,
) (protocol.RemoteBridgeSpawnData, error) {
	b.mu.Lock()
	b.spawn = payload
	b.mu.Unlock()
	return protocol.RemoteBridgeSpawnData{
		RuntimeID:   payload.RuntimeID,
		MachineID:   "machine-devbox",
		HostPID:     9001,
		ProviderPID: 9002,
		Transport:   "nexus-remote-bridge",
	}, nil
}

func (b *managedStartBridge) Status(
	_ context.Context,
	runtimeID string,
) (protocol.RemoteBridgeRuntimeStatusData, error) {
	b.mu.Lock()
	payload := b.spawn
	b.mu.Unlock()
	return protocol.RemoteBridgeRuntimeStatusData{
		RuntimeID:    runtimeID,
		MachineID:    "machine-devbox",
		Provider:     payload.Provider,
		Profile:      payload.Profile,
		Model:        payload.Model,
		Workspace:    payload.Workspace,
		State:        string(registry.StateRunning),
		ControlLevel: string(registry.ControlLevelTerminal),
		Transport:    "nexus-remote-bridge",
	}, nil
}

func (b *managedStartBridge) Resize(context.Context, string, int, int) error {
	return nil
}

func (b *managedStartBridge) Control(
	_ context.Context,
	runtimeID string,
	command protocol.CommandType,
	_ any,
) (protocol.Response, error) {
	b.mu.Lock()
	b.controls = append(b.controls, command)
	b.mu.Unlock()
	response, err := protocol.NewResponse("ok")
	if err != nil {
		return protocol.Response{}, err
	}
	response.RuntimeID = runtimeID
	response.Action = string(command)
	switch command {
	case protocol.CmdStop:
		response.Code = "STOP_REQUESTED"
		response.State = string(registry.StateStopping)
	case protocol.CmdTerminate:
		response.Code = "TERMINATE_REQUESTED"
		response.State = string(registry.StateStopping)
	}
	return response, nil
}

func (b *managedStartBridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

func openManagedStartTestNexus(t *testing.T) (*Nexus, *managedStartRouteLauncher) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dir)
	registry.ResetDefaultRegistryForTest()
	st, err := store.Open(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	routeLauncher := &managedStartRouteLauncher{}
	n := &Nexus{
		st:            st,
		launcher:      routeLauncher,
		runtimeReg:    registry.DefaultRegistry(),
		remoteProxies: map[string]runtimeProxy{},
	}
	t.Cleanup(func() {
		n.Shutdown()
	})
	return n, routeLauncher
}

func TestStartAgentUsesApprovedManagedTargetAndStopsThroughProxy(t *testing.T) {
	t.Setenv("NEXUS_ENABLE_REMOTE_EXECUTION", "1")

	n, routeLauncher := openManagedStartTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "nexus.yaml"), []byte("version: 1\nexecution:\n  defaultTarget: devbox\n  targets:\n    - id: devbox\n      kind: ssh\n      location: ssh-devbox\n      machineId: machine-devbox\n      workspace: /srv/project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{
		Name:          "Managed remote start",
		CanonicalPath: projectDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, _, err := LoadProjectManifest(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	target, ok := manifest.ProjectExecutionTarget("devbox")
	if !ok {
		t.Fatal("devbox target missing")
	}
	hello := protocol.RemoteBridgeHelloData{
		ProtocolVersion:        protocol.ProtocolVersion,
		MachineID:              "machine-devbox",
		OS:                     "linux",
		Arch:                   "amd64",
		NexusVersion:           "1.1.0-test",
		RemoteExecutionEnabled: true,
		Capabilities: []protocol.RemoteBridgeCapability{
			protocol.RemoteBridgeCapabilitySpawn,
			protocol.RemoteBridgeCapabilityCWD,
			protocol.RemoteBridgeCapabilityPTY,
			protocol.RemoteBridgeCapabilityResize,
			protocol.RemoteBridgeCapabilityLiveness,
		},
	}
	binding := ManagedExecutionTargetBinding{
		TargetID:      target.ID,
		ApprovalToken: projectExecutionTargetApprovalToken(project, target, hello),
		MachineID:     hello.MachineID,
	}
	if err := st.SetProjectSetting(project.ID, managedExecutionTargetSettingKey, binding); err != nil {
		t.Fatal(err)
	}

	agent, err := st.CreateAgent(store.Agent{
		ProjectID: project.ID,
		Name:      "Remote Dev",
	})
	if err != nil {
		t.Fatal(err)
	}

	bridge := &managedStartBridge{}
	previousOpener := openManagedRemoteBridge
	openManagedRemoteBridge = func(
		context.Context,
		string,
		string,
	) (managedRemoteBridge, protocol.RemoteBridgeHelloData, error) {
		return bridge, hello, nil
	}
	t.Cleanup(func() { openManagedRemoteBridge = previousOpener })

	session, err := n.StartAgent(context.Background(), agent.ID, "fake", "work")
	if err != nil {
		t.Fatal(err)
	}
	if session.Transport != registry.TransportManagedSSHProxy ||
		session.MachineID != "machine-devbox" ||
		session.Location != "ssh-devbox" ||
		session.Workspace != "/srv/project" {
		t.Fatalf("managed target was not projected into runtime: %+v", session)
	}
	routeLauncher.mu.Lock()
	localLaunchCalls := routeLauncher.launchCalls
	routeLauncher.mu.Unlock()
	if localLaunchCalls != 0 {
		t.Fatalf("approved remote binding called local launcher %d times", localLaunchCalls)
	}

	bridge.mu.Lock()
	spawn := bridge.spawn
	bridge.mu.Unlock()
	if spawn.RuntimeID != session.RuntimeID ||
		spawn.AgentID != agent.ID ||
		spawn.ProjectID != project.ID ||
		spawn.Provider != "fake" ||
		spawn.Profile != "work" ||
		spawn.Workspace != "/srv/project" {
		t.Fatalf("remote spawn lost routing identity: %+v", spawn)
	}

	generation, err := st.CurrentGeneration(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if generation.RuntimeID != session.RuntimeID || generation.Provider != "fake" || generation.Profile != "work" {
		t.Fatalf("runtime generation did not bind remote runtime: %+v", generation)
	}

	if err := n.StopAgent(context.Background(), agent.ID); err != nil {
		t.Fatalf("stop remote Agent: %v", err)
	}
	routeLauncher.mu.Lock()
	stopCalls := routeLauncher.stopCalls
	routeLauncher.mu.Unlock()
	if stopCalls != 1 {
		t.Fatalf("remote stop did not use transport-agnostic launcher stop: %d", stopCalls)
	}
	bridge.mu.Lock()
	controls := append([]protocol.CommandType(nil), bridge.controls...)
	closed := bridge.closed
	bridge.mu.Unlock()
	foundStop := false
	for _, command := range controls {
		if command == protocol.CmdStop {
			foundStop = true
			break
		}
	}
	if !foundStop {
		t.Fatalf("remote provider never received stop: %v", controls)
	}
	if !closed {
		t.Fatal("managed SSH bridge remained open after stop")
	}
	n.remoteProxiesMu.Lock()
	proxyCount := len(n.remoteProxies)
	n.remoteProxiesMu.Unlock()
	if proxyCount != 0 {
		t.Fatalf("remote proxy lifecycle leaked %d entries", proxyCount)
	}
}
