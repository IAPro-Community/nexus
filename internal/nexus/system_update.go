package nexus

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
)

// UpdateResult summarizes explicit Maestro maintenance. NexusUpdated remains
// false because Nexus replacement belongs to the shared Update Service.
type UpdateResult struct {
	NexusUpdated   bool   `json:"nexus_updated"`
	NexusVersion   string `json:"nexus_version"`
	MaestroUpdated bool   `json:"maestro_updated"`
	MaestroVersion string `json:"maestro_version"`
	Error          string `json:"error,omitempty"`
}

// PerformSystemUpdate installs/updates the Maestro CLI library and syncs skills.
// It does not replace the Nexus binary; that remains a release-workflow concern.
func PerformSystemUpdate() UpdateResult {
	res := UpdateResult{
		NexusVersion: buildinfo.Version,
	}

	fmt.Println("\n[1/2] Updating Orquestrador Maestro...")
	if npmPath, err := exec.LookPath("npm"); err == nil {
		cmd := exec.Command(npmPath, "install", "-g", "@iapro/orquestrador-maestro-cli@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("  ⚠️ Failed to update @iapro/orquestrador-maestro-cli via npm: %v\n", err)
		} else {
			res.MaestroUpdated = true
			fmt.Println("  ✓ Updated @iapro/orquestrador-maestro-cli to latest release.")
		}
	}

	if maestroBin, err := exec.LookPath("orquestrador-maestro"); err == nil {
		cmdSync := exec.Command(maestroBin, "update", "--non-interactive")
		_ = cmdSync.Run()

		cmdVer := exec.Command(maestroBin, "version")
		if out, err := cmdVer.Output(); err == nil {
			res.MaestroVersion = strings.TrimSpace(string(out))
		}
	} else {
		res.MaestroVersion = "not installed"
	}

	targetDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	_ = os.MkdirAll(targetDir, 0755)

	if maestroBin, err := exec.LookPath("orquestrador-maestro"); err == nil {
		_ = os.Remove(filepath.Join(targetDir, "maestro"))
		_ = os.Remove(filepath.Join(targetDir, "orquestrador"))
		_ = os.Symlink(maestroBin, filepath.Join(targetDir, "maestro"))
		_ = os.Symlink(maestroBin, filepath.Join(targetDir, "orquestrador"))
	}

	fmt.Println("\n[2/2] Confirming IAPro Nexus binary was not changed...")
	status := NewMaestroClient().Status()
	if status.Capabilities != nil {
		res.MaestroVersion = status.Capabilities.Version
	}
	res.NexusUpdated = false
	res.Error = "Nexus binary update was not performed; use the release workflow to build and install a new binary"
	return res
}
