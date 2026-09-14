//go:build !windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

func SpawnDetachedHost(exe string, runtimeID string) (*os.Process, error) {
	cmd := exec.Command(exe, "__control-host", "--runtime", runtimeID)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil

	dataDir, err := config.DataDir()
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), "ai-control")
	}
	logsDir := filepath.Join(dataDir, "logs")
	_ = os.MkdirAll(logsDir, 0700)

	logPath := filepath.Join(logsDir, fmt.Sprintf("%s.log", runtimeID))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		return nil, err
	}

	if logFile != nil {
		logFile.Close()
	}

	return cmd.Process, nil
}
