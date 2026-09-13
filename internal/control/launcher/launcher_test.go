package launcher

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/control/registry"
)

func TestMain(m *testing.M) {
	testDir, err := os.MkdirTemp("", "ai-control-launcher-test-*")
	if err == nil {
		defer os.RemoveAll(testDir)
		_ = os.Setenv("AI_MANAGER_DATA_DIR", testDir)
		_ = os.Setenv("AI_CLI_DATA_DIR", testDir)
	}
	registry.ResetDefaultRegistryForTest()
	code := m.Run()
	registry.ResetDefaultRegistryForTest()
	os.Exit(code)
}

func TestRuntimeLauncher_StandaloneLaunchAndHandshake(t *testing.T) {
	l := NewLauncher()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	runtimeID := "rt-launcher-test"
	opts := LaunchOptions{
		RuntimeID:  runtimeID,
		ProviderID: "fake",
		ProfileID:  "default",
		Workspace:  os.TempDir(),
		Standalone: true,
		Timeout:    3 * time.Second,
		Labels:     map[string]string{"nexus.interactive_lead": "true"},
	}

	sess, err := l.Launch(ctx, opts)
	if err != nil {
		t.Fatalf("launcher failed: %v", err)
	}

	if sess.RuntimeID != runtimeID {
		t.Errorf("expected runtime ID %s, got %s", runtimeID, sess.RuntimeID)
	}
	if sess.State != "RUNNING" {
		t.Errorf("expected state RUNNING, got %s", sess.State)
	}
	if sess.Labels["nexus.interactive_lead"] != "true" {
		t.Fatalf("expected launcher labels to persist, got %#v", sess.Labels)
	}
}

func TestKillProcessTerminatesChild(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	KillProcess(pid)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("process was not killed")
	}
	if registry.IsProcessAlive(pid) {
		t.Fatal("killed PID still reports alive")
	}
}

func TestForceKillSessionRemovesAliveHost(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	sess := registry.RuntimeSession{RuntimeID: "rt-force-kill", HostPID: pid, State: registry.StateStarting}
	forceKillSession(sess.RuntimeID, sess, true)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("host PID %d still alive after forceKillSession", pid)
	}
	if registry.IsProcessAlive(pid) {
		t.Fatalf("reaped host PID %d still reports alive", pid)
	}
}
