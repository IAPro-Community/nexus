//go:build linux

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
)

func linuxAutoStartPath(m *AutoStartManager) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart", m.AppID+".desktop"), nil
}

func autostartIsEnabled(m *AutoStartManager) (bool, error) {
	path, err := linuxAutoStartPath(m)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func autostartSetEnabled(m *AutoStartManager, enabled bool) error {
	path, err := linuxAutoStartPath(m)
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	execLine := desktopExecArg(m.ExecPath)
	for _, arg := range m.Args {
		execLine += " " + desktopExecArg(arg)
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
Comment=Nexus desktop surface
Exec=%s
Terminal=false
Categories=Development;
`, m.AppName, execLine)
	return os.WriteFile(path, []byte(content), 0o644)
}
