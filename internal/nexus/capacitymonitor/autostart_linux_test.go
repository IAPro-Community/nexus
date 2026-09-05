//go:build linux

package capacitymonitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxAutostartManager_GenerateUnit(t *testing.T) {
	mgr := NewLinuxAutostartManager()
	unit := mgr.GenerateUnitContent("/usr/local/bin/nexus")
	if !strings.Contains(unit, "ExecStart=/usr/local/bin/nexus capacity-monitor run") {
		t.Fatalf("expected ExecStart to contain binary path and capacity-monitor run, got: %s", unit)
	}
	if !strings.Contains(unit, "[Install]\nWantedBy=default.target") {
		t.Fatalf("expected unit to have default.target install section, got: %s", unit)
	}
}

func TestLinuxAutostartManager_InstallAndRemoveCustomDir(t *testing.T) {
	tempDir := t.TempDir()
	unitPath := filepath.Join(tempDir, "systemd", "user", "nexus-capacity-monitor.service")

	mgr := &LinuxAutostartManager{
		customUnitPath: unitPath,
		skipSystemctl:  true,
	}

	ctx := context.Background()

	// Initially not installed
	st, err := mgr.Status(ctx)
	if err != nil {
		t.Fatalf("unexpected error checking status: %v", err)
	}
	if st.Installed {
		t.Fatalf("expected not installed")
	}

	// Install
	if err := mgr.InstallAndEnable(ctx, "/opt/nexus/bin/nexus"); err != nil {
		t.Fatalf("failed to install: %v", err)
	}

	if _, err := os.Stat(unitPath); err != nil {
		t.Fatalf("expected unit file to exist at %s: %v", unitPath, err)
	}

	content, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("failed to read written unit file: %v", err)
	}
	if !strings.Contains(string(content), "ExecStart=/opt/nexus/bin/nexus capacity-monitor run") {
		t.Fatalf("unexpected unit content: %s", string(content))
	}

	// Remove
	if err := mgr.DisableAndRemove(ctx); err != nil {
		t.Fatalf("failed to remove: %v", err)
	}
	if _, err := os.Stat(unitPath); !os.IsNotExist(err) {
		t.Fatalf("expected unit file to be deleted, got err: %v", err)
	}
}
