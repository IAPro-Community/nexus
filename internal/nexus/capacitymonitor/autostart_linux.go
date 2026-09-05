//go:build linux

package capacitymonitor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const unitFileName = "nexus-capacity-monitor.service"

// LinuxAutostartManager manages user-scoped systemd services for Linux.
type LinuxAutostartManager struct {
	customUnitPath string
	skipSystemctl  bool
}

// NewLinuxAutostartManager creates a new LinuxAutostartManager targeting the user systemd config directory.
func NewLinuxAutostartManager() *LinuxAutostartManager {
	return &LinuxAutostartManager{}
}

// NewAutostartManager returns the platform-specific autostart manager.
func NewAutostartManager() AutostartManager {
	return NewLinuxAutostartManager()
}

func (m *LinuxAutostartManager) unitPath() string {
	if m.customUnitPath != "" {
		return m.customUnitPath
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join("/tmp", "systemd", "user", unitFileName)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "systemd", "user", unitFileName)
}

// GenerateUnitContent generates the INI unit configuration with the specified executable path.
func (m *LinuxAutostartManager) GenerateUnitContent(execPath string) string {
	if execPath == "" {
		execPath = "nexus"
	}
	return fmt.Sprintf(`[Unit]
Description=Nexus Capacity Monitor Daemon
Documentation=https://github.com/kivervinicius/ai-cli
After=network.target

[Service]
Type=simple
ExecStart=%s capacity-monitor run
Restart=on-failure
RestartSec=10s
KillMode=process
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
`, execPath)
}

// Status inspects the unit file and systemd state.
func (m *LinuxAutostartManager) Status(ctx context.Context) (AutostartStatus, error) {
	path := m.unitPath()
	st := AutostartStatus{
		Supported: true,
		UnitPath:  path,
	}

	if _, err := os.Stat(path); err == nil {
		st.Installed = true
	} else {
		return st, nil
	}

	if m.skipSystemctl {
		st.Enabled = true
		return st, nil
	}

	cmd := exec.CommandContext(ctx, "systemctl", "--user", "is-enabled", unitFileName)
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) == "enabled" {
		st.Enabled = true
	}

	activeCmd := exec.CommandContext(ctx, "systemctl", "--user", "is-active", unitFileName)
	actOut, _ := activeCmd.Output()
	if strings.TrimSpace(string(actOut)) == "active" {
		st.Active = true
	}

	return st, nil
}

// InstallAndEnable writes the unit file and enables/starts the service.
func (m *LinuxAutostartManager) InstallAndEnable(ctx context.Context, execPath string) error {
	path := m.unitPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	content := m.GenerateUnitContent(execPath)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write systemd unit file: %w", err)
	}

	if m.skipSystemctl {
		return nil
	}

	// Reload daemon
	_ = exec.CommandContext(ctx, "systemctl", "--user", "daemon-reload").Run()
	// Enable and start
	if err := exec.CommandContext(ctx, "systemctl", "--user", "enable", "--now", unitFileName).Run(); err != nil {
		return fmt.Errorf("failed to enable and start systemd user service: %w", err)
	}

	return nil
}

// DisableAndRemove stops, disables, and deletes the unit file.
func (m *LinuxAutostartManager) DisableAndRemove(ctx context.Context) error {
	path := m.unitPath()

	if !m.skipSystemctl {
		_ = exec.CommandContext(ctx, "systemctl", "--user", "stop", unitFileName).Run()
		_ = exec.CommandContext(ctx, "systemctl", "--user", "disable", unitFileName).Run()
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove systemd unit file: %w", err)
	}

	if !m.skipSystemctl {
		_ = exec.CommandContext(ctx, "systemctl", "--user", "daemon-reload").Run()
	}

	return nil
}
