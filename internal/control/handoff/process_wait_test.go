package handoff

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestWaitForProcessStopHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	if waitForProcessStop(ctx, os.Getpid(), time.Second) {
		t.Fatal("waitForProcessStop returned success for a canceled context")
	}
	if elapsed := time.Since(started); elapsed >= 100*time.Millisecond {
		t.Fatalf("cancellation took %s", elapsed)
	}
}
