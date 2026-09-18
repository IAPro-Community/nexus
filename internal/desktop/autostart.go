package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// AutoStartManager configures desktop application startup at user login.
type AutoStartManager struct {
	AppName  string
	ExecPath string
}

// NewAutoStartManager creates a manager for login startup.
func NewAutoStartManager(appName, execPath string) *AutoStartManager {
	if appName == "" {
		appName = "IAPro Nexus"
	}
	if execPath == "" {
		execPath, _ = os.Executable()
	}
	return &AutoStartManager{
		AppName:  appName,
		ExecPath: execPath,
	}
}

// IsEnabled checks if autostart is currently configured.
func (m *AutoStartManager) IsEnabled() bool {
	switch runtime.GOOS {
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		desktopFile := filepath.Join(home, ".config", "autostart", "iapro-nexus.desktop")
		_, err = os.Stat(desktopFile)
		return err == nil
	}
	return false
}

// SetEnabled enables or disables autostart.
func (m *AutoStartManager) SetEnabled(enabled bool) error {
	switch runtime.GOOS {
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		autoDir := filepath.Join(home, ".config", "autostart")
		desktopFile := filepath.Join(autoDir, "iapro-nexus.desktop")

		if !enabled {
			_ = os.Remove(desktopFile)
			return nil
		}

		if err := os.MkdirAll(autoDir, 0755); err != nil {
			return err
		}

		content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
Comment=Autonomous AI Orchestrator & Workspace OS
Exec=%s --minimized
Icon=nexus
Terminal=false
Categories=Development;
`, m.AppName, desktopExecArg(m.ExecPath))

		return os.WriteFile(desktopFile, []byte(content), 0644)
	}
	return nil
}

// desktopExecArg quotes a single executable argument according to the
// freedesktop Desktop Entry Exec grammar. It prevents spaces or quotes in a
// user-selected installation path from becoming additional arguments.
func desktopExecArg(value string) string {
	var b strings.Builder
	b.Grow(len(value) + 2)
	b.WriteByte('"')
	for _, r := range value {
		if r == '\\' || r == '"' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}
