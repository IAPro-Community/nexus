package launcher

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
)

// HostLogPath returns the detached SessionHost log path for a runtime.
func HostLogPath(runtimeID string) string {
	dataDir, err := config.DataDir()
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), "ai-control")
	}
	return filepath.Join(dataDir, "logs", fmt.Sprintf("%s.log", runtimeID))
}

func readHostLogTail(runtimeID string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	path := HostLogPath(runtimeID)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	data = bytes.TrimSpace(data)
	if idx := bytes.LastIndexByte(data, '\n'); idx >= 0 && idx+1 < len(data) {
		data = data[idx+1:]
	}
	return string(bytes.TrimSpace(data))
}

func classifyDetachedHostFailure(runtimeID string, hostPID int) error {
	tail := readHostLogTail(runtimeID, 8192)
	base := fmt.Errorf("detached SessionHost exited before handshake (pid %d)", hostPID)
	if tail == "" {
		return base
	}
	detail := fmt.Errorf("%w: %s", base, tail)
	if strings.Contains(tail, codex.ErrTUIBusy.Error()) {
		return fmt.Errorf("%w: %v", codex.ErrTUIBusy, detail)
	}
	return detail
}
