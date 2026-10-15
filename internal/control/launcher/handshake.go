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

// waitForRuntimeHandshake waits for the control endpoint and returns the host
// identity from the same responsive request. This avoids a second named-pipe
// connection during Windows startup.
func waitForRuntimeHandshake(ctx context.Context, runtimeID string, hostProc *os.Process, timeout time.Duration) (protocol.StatusData, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var hostDone <-chan struct{}
	if hostProc != nil && hostProc.Pid > 0 {
		done := make(chan struct{})
		hostDone = done
		go func() {
			_, _ = hostProc.Wait()
			close(done)
		}()
	}

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
		var status protocol.StatusData
		if err != nil {
			traceCaptureRuntimeHandshake("attempt=%d stage=dial-failed elapsed=%s err=%q", attempt, time.Since(attemptStarted), err)
		} else {
			traceCaptureRuntimeHandshake("attempt=%d stage=dial-ready elapsed=%s", attempt, time.Since(attemptStarted))
			statusStarted := time.Now()
			status, err = client.HandshakeStatusContext(attemptCtx)
			traceCaptureRuntimeHandshake("attempt=%d stage=status-finished elapsed=%s err=%q", attempt, time.Since(statusStarted), err)
			_ = client.Close()
		}
		cancelAttempt()
		if err == nil {
			traceCaptureRuntimeHandshake("stage=ready total=%s attempts=%d", time.Since(waitStarted), attempt)
			return status, nil
		}

		select {
		case <-hostDone:
			return protocol.StatusData{}, classifyDetachedHostFailure(runtimeID, hostProc.Pid)
		case <-ctx.Done():
			return protocol.StatusData{}, ctx.Err()
		case <-waitCtx.Done():
			traceCaptureRuntimeHandshake("stage=timeout total=%s attempts=%d ctx_err=%q", time.Since(waitStarted), attempt, waitCtx.Err())
			select {
			case <-hostDone:
				return protocol.StatusData{}, classifyDetachedHostFailure(runtimeID, hostProc.Pid)
			default:
			}
			if hostProc != nil && hostProc.Pid > 0 {
				if tail := readHostLogTail(runtimeID, 8192); strings.Contains(tail, codex.ErrTUIBusy.Error()) {
					return protocol.StatusData{}, classifyDetachedHostFailure(runtimeID, hostProc.Pid)
				}
			}
			return protocol.StatusData{}, fmt.Errorf("timed out waiting for endpoint %s", runtimeID)
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
