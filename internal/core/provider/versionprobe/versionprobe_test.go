package versionprobe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestResultForProbeFailureDoesNotExposeCapturedOutput(t *testing.T) {
	result := resultFromProbe("gemini", "SyntaxError: private diagnostic", errors.New("exit status 1"))
	if !result.Installed || result.BinaryPath != "gemini" || result.Version != "unknown" {
		t.Fatalf("unexpected installed provider result: %+v", result)
	}
	if result.Error == "" || strings.Contains(result.Error, "private diagnostic") {
		t.Fatalf("probe failure should be reported without copying combined output: %+v", result)
	}
}

func TestResultForProbeSuccessTrimsVersion(t *testing.T) {
	result := resultFromProbe("codex", "  codex-cli 1.2.3\r\n", nil)
	want := model.DetectionResult{Installed: true, Version: "codex-cli 1.2.3", BinaryPath: "codex"}
	if result != want {
		t.Fatalf("result=%+v, want %+v", result, want)
	}
}

func TestResultForEmptyProbeOutputIsUnknown(t *testing.T) {
	result := resultFromProbe("agy", " \r\n", nil)
	if !result.Installed || result.Version != "unknown" || result.Error == "" {
		t.Fatalf("empty output should remain installed but unverified: %+v", result)
	}
}

func TestDetectCachesCoalescesAndInvalidatesOnBinaryUpdate(t *testing.T) {
	InvalidateCache()
	origRun := runCommandFn
	origNow := nowFn
	t.Cleanup(func() {
		runCommandFn = origRun
		nowFn = origNow
		InvalidateCache()
	})

	binPath := filepath.Join(t.TempDir(), "mock-cli")
	if err := os.WriteFile(binPath, []byte("v1"), 0o700); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	gate := make(chan struct{})
	runCommandFn = func(ctx context.Context, binary string, args []string, env []string, dir string) (string, error) {
		n := calls.Add(1)
		if n == 1 {
			<-gate
			return "mock-cli 1.0.0\n", nil
		}
		return "mock-cli 2.0.0\n", nil
	}

	// Concurrent callers share a single probe execution (singleflight).
	var wg sync.WaitGroup
	results := make([]model.DetectionResult, 6)
	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = Detect(context.Background(), binPath)
		}(i)
	}
	time.Sleep(15 * time.Millisecond)
	close(gate)
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("expected concurrent probes to coalesce into 1 execution, got %d", calls.Load())
	}
	for _, r := range results {
		if r.Version != "mock-cli 1.0.0" {
			t.Fatalf("unexpected version result: %+v", r)
		}
	}

	// Subsequent call hits cache without executing binary.
	cached := Detect(context.Background(), binPath)
	if cached.Version != "mock-cli 1.0.0" || calls.Load() != 1 {
		t.Fatalf("expected cache hit (calls=1), got version=%q calls=%d", cached.Version, calls.Load())
	}

	// Binary upgrade changes file mtime/size and invalidates cache automatically.
	if err := os.WriteFile(binPath, []byte("v2-upgraded"), 0o700); err != nil {
		t.Fatal(err)
	}
	upgraded := Detect(context.Background(), binPath)
	if upgraded.Version != "mock-cli 2.0.0" || calls.Load() != 2 {
		t.Fatalf("expected upgraded binary to re-probe, got version=%q calls=%d", upgraded.Version, calls.Load())
	}
}

func BenchmarkDetectCached(b *testing.B) {
	InvalidateCache()
	origRun := runCommandFn
	b.Cleanup(func() {
		runCommandFn = origRun
		InvalidateCache()
	})
	binPath := filepath.Join(b.TempDir(), "bench-cli")
	if err := os.WriteFile(binPath, []byte("bin"), 0o700); err != nil {
		b.Fatal(err)
	}
	runCommandFn = func(ctx context.Context, binary string, args []string, env []string, dir string) (string, error) {
		return "bench-cli 1.0.0\n", nil
	}
	_ = Detect(context.Background(), binPath)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if res := Detect(context.Background(), binPath); res.Version != "bench-cli 1.0.0" {
			b.Fatalf("unexpected version: %+v", res)
		}
	}
}
