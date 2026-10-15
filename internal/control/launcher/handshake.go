package launcher

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
)

func traceCaptureRuntimeHandshake(format string, args ...any) {
	if os.Getenv("NEXUS_DOCS_CAPTURE") == "1" {
		log.Printf("nexus_docs_capture_diagnostic=runtime_handshake "+format, args...)
	}
}

// waitForRuntimeHandshake waits for the control endpoint. When hostProc is set,
// an early host exit surfaces the host log immediately instead of timing out.
// Wait is required because a failed Setsid child becomes a zombie that still
// looks "alive" to kill(pid, 0) until the parent reaps it.
func waitForRuntimeHandshake(ctx context.Context, runtimeID string, hostProc *os.Process, timeout time.Duration) error {
	if hostProc == nil || hostProc.Pid <= 0 {
		return protocol.WaitForEndpoint(ctx, runtimeID, timeout)
	}

	hostDone := make(chan struct{})
	go func() {
		_, _ = hostProc.Wait()
		close(hostDone)
	}()

	waitStarted := time.Now()
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	attempt := 0
	for {
		attempt++
		attemptStarted := time.Now()
		attemptCtx, cancelAttempt := context.WithTimeout(waitCtx, 500*time.Millisecond)
		client, err := protocol.NewClientContext(attemptCtx, runtimeID)
		if err != nil {
			traceCaptureRuntimeHandshake("attempt=%d stage=dial-failed elapsed=%s err=%q", attempt, time.Since(attemptStarted), err)
		} else {
			traceCaptureRuntimeHandshake("attempt=%d stage=dial-ready elapsed=%s", attempt, time.Since(attemptStarted))
			pingStarted := time.Now()
			err = client.PingContext(attemptCtx)
			traceCaptureRuntimeHandshake("attempt=%d stage=ping-finished elapsed=%s err=%q", attempt, time.Since(pingStarted), err)
			_ = client.Close()
		}
		cancelAttempt()
		if err == nil {
			traceCaptureRuntimeHandshake("stage=ready total=%s attempts=%d", time.Since(waitStarted), attempt)
			return nil
		}

		select {
		case <-hostDone:
			return classifyDetachedHostFailure(runtimeID, hostProc.Pid)
		case <-ctx.Done():
			return ctx.Err()
		case <-waitCtx.Done():
			traceCaptureRuntimeHandshake("stage=timeout total=%s attempts=%d ctx_err=%q", time.Since(waitStarted), attempt, waitCtx.Err())
			select {
			case <-hostDone:
				return classifyDetachedHostFailure(runtimeID, hostProc.Pid)
			default:
			}
			if tail := readHostLogTail(runtimeID, 8192); strings.Contains(tail, codex.ErrTUIBusy.Error()) {
				return classifyDetachedHostFailure(runtimeID, hostProc.Pid)
			}
			return fmt.Errorf("timed out waiting for endpoint %s", runtimeID)
		case <-ticker.C:
		}
	}
}

func isEarlyHostExit(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, codex.ErrTUIBusy) {
		return true
	}
	return strings.Contains(err.Error(), "exited before handshake")
}
