package launcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestParseControlHostRuntime(t *testing.T) {
	id, ok := parseControlHostRuntime("/home/x/nexus __control-host --runtime codex-abc")
	if !ok || id != "codex-abc" {
		t.Fatalf("got %q ok=%v", id, ok)
	}
	if _, ok := parseControlHostRuntime("sleep 30"); ok {
		t.Fatal("expected non-host cmdline to be ignored")
	}
}

func TestOrphanHostProcessesKeepsActiveRuntime(t *testing.T) {
	hosts := []HostProcess{
		{PID: 11, RuntimeID: "keep-me"},
		{PID: 22, RuntimeID: "orphan-me"},
	}
	sessions := []registry.RuntimeSession{
		{RuntimeID: "keep-me", State: registry.StateRunning},
		{RuntimeID: "stopped", State: registry.StateStopped},
	}
	got := orphanHostProcesses(hosts, sessions)
	if len(got) != 1 || got[0].RuntimeID != "orphan-me" {
		t.Fatalf("got %#v", got)
	}
}

func TestReapOrphanHostsFromKillsUnregisteredHost(t *testing.T) {
	cmd := startLongRunningTestProcess(t)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	n := ReapOrphanHostsFrom([]HostProcess{{PID: pid, RuntimeID: "orphan-rt"}}, nil)
	if n != 1 {
		t.Fatalf("reaped=%d", n)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("orphan host was not killed")
	}
}

func TestReapOrphanHostsFromLeavesRunningRegistryHost(t *testing.T) {
	cmd := startLongRunningTestProcess(t)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	n := ReapOrphanHostsFrom(
		[]HostProcess{{PID: pid, RuntimeID: "keep-rt"}},
		[]registry.RuntimeSession{{RuntimeID: "keep-rt", State: registry.StateRunning, HostPID: pid}},
	)
	if n != 0 {
		t.Fatalf("reaped=%d want 0", n)
	}
	if !registry.IsProcessAlive(pid) {
		t.Fatal("registered RUNNING host was killed")
	}
}

func TestMatchingHostExecutableHandlesDeletedAndSiblings(t *testing.T) {
	if !matchingHostExecutable("/usr/local/bin/nexus", "/usr/local/bin/nexus (deleted)") {
		t.Fatal("deleted self binary should match")
	}
	if !matchingHostExecutable("/usr/local/bin/nexus", "/usr/local/bin/nexus-desktop") {
		t.Fatal("sibling nexus-desktop binary should match")
	}
	if matchingHostExecutable("/usr/local/bin/nexus", "/usr/bin/python3") {
		t.Fatal("unrelated binary must not match")
	}
}

func TestHostScopeMatchesGuardsForeignLiveDirectories(t *testing.T) {
	curData := t.TempDir()
	foreignLiveData := t.TempDir()
	foreignDeadData := filepath.Join(t.TempDir(), "removed-e2e-dir")

	if hostScopeMatches(hostEnvScope{DataDir: foreignLiveData}, curData, "", true) {
		t.Fatal("must not match foreign existing NEXUS_DATA_DIR")
	}
	if !hostScopeMatches(hostEnvScope{DataDir: foreignDeadData}, curData, "", true) {
		t.Fatal("deleted foreign NEXUS_DATA_DIR should be reapable")
	}
	if !hostScopeMatches(hostEnvScope{DataDir: curData}, curData, "", true) {
		t.Fatal("matching NEXUS_DATA_DIR should match")
	}
	if hostScopeMatches(hostEnvScope{}, curData, "", true) {
		t.Fatal("explicitly isolated process must not match unlabeled host")
	}
	if !hostScopeMatches(hostEnvScope{}, curData, "", false) {
		t.Fatal("non-isolated host should match unlabeled sibling host")
	}
	_ = os.RemoveAll(foreignDeadData)
}
