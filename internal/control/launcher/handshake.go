package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
)

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

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	for {
		client, err := protocol.NewClient(runtimeID)
		if err == nil {
			err = client.Ping()
			_ = client.Close()
			if err == nil {
				return nil
			}
		}

		select {
		case <-hostDone:
			return classifyDetachedHostFailure(runtimeID, hostProc.Pid)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		if time.Now().After(deadline) {
			select {
			case <-hostDone:
				return classifyDetachedHostFailure(runtimeID, hostProc.Pid)
			default:
			}
			if tail := readHostLogTail(runtimeID, 8192); strings.Contains(tail, codex.ErrTUIBusy.Error()) {
				return classifyDetachedHostFailure(runtimeID, hostProc.Pid)
			}
			return fmt.Errorf("timed out waiting for endpoint %s", runtimeID)
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
