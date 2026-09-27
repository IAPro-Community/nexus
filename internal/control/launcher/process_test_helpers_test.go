package launcher

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/executil"
)

const launcherSleeperEnv = "NEXUS_TEST_LAUNCHER_SLEEPER"

func TestLauncherLongRunningProcessHelper(t *testing.T) {
	if os.Getenv(launcherSleeperEnv) != "1" {
		return
	}
	for {
		time.Sleep(time.Hour)
	}
}

func startLongRunningTestProcess(t *testing.T) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve test executable: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=^TestLauncherLongRunningProcessHelper$")
	cmd.Env = append(os.Environ(), launcherSleeperEnv+"=1")
	executil.HideConsoleWindow(cmd)
	return cmd
}
