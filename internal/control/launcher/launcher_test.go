package launcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
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

type failBuildCommandDriver struct {
	*driver.FakeDriver
}

func (d *failBuildCommandDriver) ProviderID() string { return "fake" }

func (d *failBuildCommandDriver) BuildCommand(context.Context, model.Profile, []string) (string, []string, []string, error) {
	return "", nil, nil, fmt.Errorf("logical provider binary is intentionally unavailable on host")
}

func TestCustomTargetProviderProcessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_CUSTOM_TARGET_PROVIDER") != "1" {
		return
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func TestRuntimeLauncherCustomTargetDoesNotRequireProviderBinaryOnHost(t *testing.T) {
	l := NewLauncher()
	drivers := driver.NewRegistry()
	drivers.Register(&failBuildCommandDriver{FakeDriver: driver.NewFakeDriver()})
	l.drivers = drivers

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	template := fmt.Sprintf(`"%s" -test.run=TestCustomTargetProviderProcessHelper {args}`, filepath.ToSlash(exe))

	sess, err := l.Launch(ctx, LaunchOptions{
		RuntimeID:  "rt-custom-target-no-provider-bin",
		ProviderID: "fake",
		ProfileID:  "default",
		Workspace:  t.TempDir(),
		Standalone: true,
		Timeout:    3 * time.Second,
		Environment: map[string]string{
			"GO_WANT_CUSTOM_TARGET_PROVIDER": "1",
		},
		Options: map[string]any{
			"execution_adapter": "command_template",
			"command_template":  template,
			"target_kind":       "custom",
		},
	})
	if err != nil {
		t.Fatalf("custom target unexpectedly required provider BuildCommand: %v", err)
	}
	if sess.ProviderID != "fake" || sess.Binary == "" {
		t.Fatalf("logical provider/session metadata lost: %#v", sess)
	}
	if sess.Labels["nexus.execution_adapter"] != "command_template" ||
		sess.Labels["nexus.target_kind"] != "custom" {
		t.Fatalf("custom target metadata missing from runtime session: %#v", sess.Labels)
	}
	if err := l.Stop(sess.RuntimeID); err != nil {
		// Standalone stop may report the protocol close as part of teardown;
		// the important assertion is that launch succeeded without BuildCommand.
		t.Logf("custom target stop: %v", err)
	}
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
	if sess.PID <= 0 {
		t.Errorf("expected provider PID from status handshake, got %d", sess.PID)
	}
	if sess.HostPID <= 0 {
		t.Errorf("expected host PID from status handshake, got %d", sess.HostPID)
	}
	if sess.HostGeneration <= 0 {
		t.Errorf("expected non-zero host generation from status handshake, got %d", sess.HostGeneration)
	}
	if sess.Labels["nexus.interactive_lead"] != "true" {
		t.Fatalf("expected launcher labels to persist, got %#v", sess.Labels)
	}

	persisted, ok := l.reg.Get(runtimeID)
	if !ok {
		t.Fatalf("expected launched runtime %s in registry", runtimeID)
	}
	if persisted.HostPID != sess.HostPID || persisted.HostGeneration != sess.HostGeneration {
		t.Fatalf("registry lost host identity: persisted host=%d generation=%d, returned host=%d generation=%d", persisted.HostPID, persisted.HostGeneration, sess.HostPID, sess.HostGeneration)
	}
}

func TestKillProcessTerminatesChild(t *testing.T) {
	cmd := startLongRunningTestProcess(t)
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

func TestForceKillSessionDoesNotSignalCurrentProcess(t *testing.T) {
	currentPID := os.Getpid()
	sess := registry.RuntimeSession{
		RuntimeID: "rt-self-protection",
		PID:       currentPID,
		HostPID:   currentPID,
		State:     registry.StateRunning,
	}
	forceKillSession(sess.RuntimeID, sess, true)
	if sessionProcessAlive(sess, true) {
		t.Fatal("current Nexus process identity must not be treated as a killable runtime child")
	}
}

func TestForceKillSessionRemovesAliveHost(t *testing.T) {
	cmd := startLongRunningTestProcess(t)
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

func TestLaunchMissingWorkspaceDirectoryFails(t *testing.T) {
	l := NewLauncher()
	drivers := driver.NewRegistry()
	drivers.Register(driver.NewFakeDriver())
	l.drivers = drivers

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	missingDir := filepath.Join(t.TempDir(), "non-existent-workspace")
	_, err := l.Launch(ctx, LaunchOptions{
		RuntimeID:  "rt-missing-ws",
		ProviderID: "fake",
		ProfileID:  "default",
		Workspace:  missingDir,
	})
	if err == nil {
		t.Fatal("expected Launch to fail for non-existent workspace directory, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected 'does not exist' error, got: %v", err)
	}
}
