package agentwire

import (
	"context"
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

// TestOMPLiveAuthenticatedRPCTwoSessions is an explicitly paid, opt-in E2E.
// It is never enabled by CI or inherited from a developer's environment:
// the operator must select the model, binary and ONE accepted API-key name.
// Credentials and provider output are never written into test diagnostics.
func TestOMPLiveAuthenticatedRPCTwoSessions(t *testing.T) {
	if os.Getenv("NEXUS_OMP_AUTH_E2E") != "1" {
		t.Skip("authenticated OMP E2E requires explicit NEXUS_OMP_AUTH_E2E=1")
	}
	if runtime.GOOS != "linux" {
		t.Skip("authenticated OMP RPC smoke is currently Linux-scoped")
	}
	binary := strings.TrimSpace(os.Getenv("NEXUS_OMP_AUTH_BIN"))
	model := strings.TrimSpace(os.Getenv("NEXUS_OMP_AUTH_MODEL"))
	keyName := os.Getenv("NEXUS_OMP_AUTH_KEY_ENV")
	allowed := map[string]bool{
		"OPENAI_API_KEY": true,
		"ANTHROPIC_API_KEY": true,
		"GEMINI_API_KEY": true,
		"OPENROUTER_API_KEY": true,
	}
	if binary == "" || model == "" || !allowed[keyName] || os.Getenv(keyName) == "" {
		t.Fatal("set NEXUS_OMP_AUTH_BIN, NEXUS_OMP_AUTH_MODEL, NEXUS_OMP_AUTH_KEY_ENV and its provider key before running paid E2E")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatal("configured OMP binary does not exist")
	}

	type instance struct {
		bridge *Bridge
		events chan WireSignal
		mu     *sync.Mutex
		text   *strings.Builder
	}
	instances := make([]instance, 0, 2)
	for i := 0; i < 2; i++ {
		root := t.TempDir()
		home := filepath.Join(root, "home")
		workspace := filepath.Join(root, "workspace")
		for _, dir := range []string{home, workspace} {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal("unable to create isolated E2E workspace")
			}
		}
		// Never pass os.Environ() to a third-party agent in CI: only the
		// explicitly selected provider credential is forwarded.
		env := []string{
			"HOME=" + home,
			"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
			"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
			"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
			"PATH=/usr/bin:/bin",
			"TERM=dumb",
			"CI=true",
			keyName + "=" + os.Getenv(keyName),
		}
		ctx, cancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(ctx, binary,
			"--mode", "rpc", "--no-session", "--no-extensions", "--no-ui",
			"--model", model)
		cmd.Env = env
		cmd.Dir = workspace
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			t.Fatal("unable to open OMP stdout")
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			cancel()
			t.Fatal("unable to open OMP stdin")
		}
		cmd.Stderr = io.Discard // May contain provider-supplied sensitive data.
		events := make(chan WireSignal, 256)
		mu := &sync.Mutex{}
		output := &strings.Builder{}
		bridge, err := NewBridge("pi_rpc", stdout, stdin, workspace,
			func(s WireSignal) { events <- s },
			func(fragment string) {
				mu.Lock()
				defer mu.Unlock()
				// The smoke only checks nonempty model output: no need to
				// retain an unbounded answer or print it in CI logs.
				if output.Len() < 65536 {
					output.WriteString(fragment)
				}
			})
		if err != nil {
			cancel()
			t.Fatal("unable to initialize OMP RPC bridge")
		}
		if err := cmd.Start(); err != nil {
			cancel()
			t.Fatal("unable to start OMP child")
		}
		t.Cleanup(func() {
			bridge.Stop()
			cancel()
			_ = stdin.Close()
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
		// Start owns the long-lived RPC context. Never cancel it after
		// the initial get_state handshake.
		if err := bridge.Start(ctx); err != nil {
			t.Fatal("OMP authenticated RPC handshake failed")
		}
		instances = append(instances, instance{bridge: bridge, events: events, mu: mu, text: output})
	}
	if a, b := instances[0].bridge.SessionID(), instances[1].bridge.SessionID(); a == "" || b == "" || a == b {
		t.Fatal("two OMP authenticated sessions did not obtain independent identities")
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, inst := range instances {
		wg.Add(1)
		go func(b *Bridge) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			results <- b.Prompt(ctx, "Reply with one plain text sentence containing the word NEXUS.")
		}(inst.bridge)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal("an authenticated prompt was not admitted")
		}
	}
	for index, inst := range instances {
		timeout := time.NewTimer(90 * time.Second)
		settled := false
		for !settled {
			select {
			case ev := <-inst.events:
				switch ev.State {
				case "done":
					if ev.Kind != "prompt_result" {
						t.Fatalf("session %d completed without correlated prompt result", index)
					}
					settled = true
				case "error", "cancelled":
					t.Fatalf("session %d failed to complete authenticated model execution (state=%s)", index, ev.State)
				}
			case <-timeout.C:
				t.Fatalf("session %d timed out waiting for authenticated provider completion", index)
			}
		}
		timeout.Stop()
		inst.mu.Lock()
		hasText := strings.TrimSpace(inst.text.String()) != ""
		inst.mu.Unlock()
		if !hasText {
			t.Fatalf("session %d reported completion but no model text was observed", index)
		}
	}
}
