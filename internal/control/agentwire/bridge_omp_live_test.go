package agentwire

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestOMPLiveRPCNoAuthTwoSessions is opt-in. The default test suite never
// downloads a provider, attempts a login, or requires paid model access.
// Set NEXUS_OMP_LIVE_BIN to a locally installed OMP executable to run this.
func TestOMPLiveRPCNoAuthTwoSessions(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("live OMP RPC sandbox scenario currently certified on Linux only")
	}
	binary := strings.TrimSpace(os.Getenv("NEXUS_OMP_LIVE_BIN"))
	if binary == "" {
		t.Skip("set NEXUS_OMP_LIVE_BIN to opt in to real OMP RPC verification")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("OMP executable unavailable: %v", err)
	}

	type instance struct {
		bridge *Bridge
		events chan WireSignal
	}
	model := strings.TrimSpace(os.Getenv("NEXUS_OMP_LIVE_MODEL"))
	if model == "" {
		model = "openai/gpt-5.2"
	}
	var instances []instance
	for i := 0; i < 2; i++ {
		root := t.TempDir()
		workspace := filepath.Join(root, "workspace")
		home := filepath.Join(root, "home")
		if err := os.MkdirAll(workspace, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(home, 0700); err != nil {
			t.Fatal(err)
		}
		// A minimal, explicit env prevents inherited API keys or user config.
		env := []string{
			"HOME=" + home,
			"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
			"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
			"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
			"PATH=/usr/bin:/bin",
			"TERM=dumb",
			"LANG=C.UTF-8",
			"CI=true",
		}
		procCtx, procCancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(procCtx, binary,
			"--mode", "rpc", "--no-session", "--no-extensions", "--no-ui",
			"--model", model)
		cmd.Env = env
		cmd.Dir = workspace
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			procCancel()
			t.Fatal(err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			procCancel()
			t.Fatal(err)
		}
		cmd.Stderr = io.Discard
		events := make(chan WireSignal, 128)
		bridge, err := NewBridge("pi_rpc", stdout, stdin, workspace,
			func(signal WireSignal) { events <- signal }, nil)
		if err != nil {
			procCancel()
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			procCancel()
			t.Fatalf("start real OMP instance %d: %v", i, err)
		}
		t.Cleanup(func() {
			bridge.Stop()
			procCancel()
			_ = stdin.Close()
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
		// Start keeps its context for the lifetime of the protocol pump.
		// Canceling a temporary handshake context would discard all later events.
		err = bridge.Start(procCtx)
		if err != nil {
			t.Fatalf("real OMP instance %d handshake failed: %v", i, err)
		}
		instances = append(instances, instance{bridge: bridge, events: events})
	}

	a, b := instances[0].bridge.SessionID(), instances[1].bridge.SessionID()
	if a == "" || b == "" || a == b {
		t.Fatalf("OMP sessions are not independently identified (empty=%t/%t equal=%t)",
			a == "", b == "", a == b)
	}
	// Without authentication, prompts must fail with a correlated provider
	// outcome, never with an invented successful DONE.
	for attempt := 0; attempt < 2; attempt++ {
		var group sync.WaitGroup
		errs := make(chan error, len(instances))
		for i, inst := range instances {
			group.Add(1)
			go func(i int, inst instance) {
				defer group.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				errs <- inst.bridge.Prompt(ctx, fmt.Sprintf("answer hello %d", i))
			}(i, inst)
		}
		group.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("OMP prompt admission failed: %v", err)
			}
		}
		for i, inst := range instances {
			timeout := time.NewTimer(15 * time.Second)
			foundError := false
			for !foundError {
				select {
				case signal := <-inst.events:
					if signal.State == "done" {
						t.Fatalf("instance %d fabricated DONE without auth", i)
					}
					if signal.State == "error" {
						if signal.Kind != "prompt_result" {
							t.Fatalf("instance %d reported non-correlated error: %+v", i, signal)
						}
						foundError = true
					}
				case <-timeout.C:
					t.Fatalf("instance %d missing correlated prompt_result error", i)
				}
			}
			timeout.Stop()
		}
	}
}
