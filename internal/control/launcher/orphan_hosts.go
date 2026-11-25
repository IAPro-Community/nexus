package launcher

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/config"
)

// HostProcess is a live SessionHost process discovered on this machine.
type HostProcess struct {
	PID        int
	RuntimeID  string
	Executable string
}

var nexusHostBinaryNames = map[string]struct{}{
	"nexus":         {},
	"nexus.exe":     {},
	"nexus-desktop": {},
	"nexus-hud":     {},
	"ai-manager":    {},
	"ai-cli":        {},
}

func normalizeProcExePath(exe string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(exe), " (deleted)")
	if resolved, err := filepath.EvalSymlinks(trimmed); err == nil {
		trimmed = strings.TrimSuffix(strings.TrimSpace(resolved), " (deleted)")
	}
	return filepath.Clean(trimmed)
}

func matchingHostExecutable(selfExecutable, candidateExecutable string) bool {
	selfClean := normalizeProcExePath(selfExecutable)
	candClean := normalizeProcExePath(candidateExecutable)
	if selfClean == "" || candClean == "" || candClean == "." {
		return false
	}
	if candClean == selfClean {
		return true
	}
	_, candIsNexus := nexusHostBinaryNames[strings.ToLower(filepath.Base(candClean))]
	_, selfIsNexus := nexusHostBinaryNames[strings.ToLower(filepath.Base(selfClean))]
	return candIsNexus && (selfIsNexus || strings.HasSuffix(strings.ToLower(filepath.Base(selfClean)), ".test"))
}

type hostEnvScope struct {
	DataDir  string
	StateDir string
}

func parseHostEnvScope(environ []byte) hostEnvScope {
	var scope hostEnvScope
	for _, entry := range strings.Split(string(environ), "\x00") {
		k, v, ok := strings.Cut(entry, "=")
		if !ok || strings.TrimSpace(v) == "" {
			continue
		}
		switch k {
		case "NEXUS_DATA_DIR", "AI_MANAGER_DATA_DIR", "AI_CLI_DATA_DIR":
			if scope.DataDir == "" {
				scope.DataDir = filepath.Clean(v)
			}
		case "NEXUS_STATE_DIR", "AI_MANAGER_STATE_DIR", "AI_CLI_STATE_DIR":
			if scope.StateDir == "" {
				scope.StateDir = filepath.Clean(v)
			}
		}
	}
	return scope
}

func hostScopeMatches(target hostEnvScope, currentDataDir, currentStateDir string, explicitIsolated bool) bool {
	if target.DataDir != "" {
		if currentDataDir != "" && filepath.Clean(target.DataDir) == filepath.Clean(currentDataDir) {
			return true
		}
		_, err := os.Stat(target.DataDir)
		return os.IsNotExist(err)
	}
	if target.StateDir != "" {
		if currentStateDir != "" && filepath.Clean(target.StateDir) == filepath.Clean(currentStateDir) {
			return true
		}
		_, err := os.Stat(target.StateDir)
		return os.IsNotExist(err)
	}
	// Never reap unlabeled host processes when running inside an explicitly isolated tempdir scope.
	return !explicitIsolated
}

func parseControlHostRuntime(cmdline string) (string, bool) {
	fields := strings.Fields(strings.ReplaceAll(cmdline, "\x00", " "))
	for i, field := range fields {
		if field != "__control-host" {
			continue
		}
		if i+2 < len(fields) && fields[i+1] == "--runtime" {
			id := strings.TrimSpace(fields[i+2])
			if id != "" {
				return id, true
			}
		}
		return "", false
	}
	return "", false
}

func orphanHostProcesses(hosts []HostProcess, sessions []registry.RuntimeSession) []HostProcess {
	active := make(map[string]bool, len(sessions))
	for _, sess := range sessions {
		if sess.IsActive() && strings.TrimSpace(sess.RuntimeID) != "" {
			active[sess.RuntimeID] = true
		}
	}
	var orphans []HostProcess
	for _, host := range hosts {
		if host.PID <= 0 || strings.TrimSpace(host.RuntimeID) == "" {
			continue
		}
		if active[host.RuntimeID] {
			continue
		}
		orphans = append(orphans, host)
	}
	return orphans
}

func reapLiveHost(host HostProcess) {
	if host.PID <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if client, err := protocol.NewClient(host.RuntimeID); err == nil {
		_, _ = client.SendContext(ctx, protocol.CmdTerminate, nil)
		_ = client.Close()
	}
	killProcessGroup(host.PID)
	KillProcess(host.PID)
	_ = os.Remove(protocol.EndpointPath(host.RuntimeID))
}

// ReapOrphanHosts terminates SessionHost processes whose runtime ID is not in
// an active registry row. Registered RUNNING/STARTING/DETACHED hosts are kept.
func ReapOrphanHosts(sessions []registry.RuntimeSession) int {
	return ReapOrphanHostsFrom(listControlHostProcesses(), sessions)
}

// ReapOrphanHostsFrom is the testable entry point for orphan reaping.
func ReapOrphanHostsFrom(hosts []HostProcess, sessions []registry.RuntimeSession) int {
	orphans := orphanHostProcesses(hosts, sessions)
	for _, host := range orphans {
		reapLiveHost(host)
	}
	return len(orphans)
}

func listControlHostProcesses() []HostProcess {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	selfExecutable, err := os.Executable()
	if err != nil {
		return nil
	}
	curDataDir, _ := config.DataDir()
	curStateDir, _ := config.StateDir()
	explicitIsolated := strings.TrimSpace(os.Getenv("NEXUS_DATA_DIR")) != "" ||
		strings.TrimSpace(os.Getenv("NEXUS_STATE_DIR")) != "" ||
		strings.TrimSpace(os.Getenv("AI_MANAGER_DATA_DIR")) != "" ||
		strings.TrimSpace(os.Getenv("AI_CLI_DATA_DIR")) != ""

	self := os.Getpid()
	var hosts []HostProcess
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 1 || pid == self {
			continue
		}
		if !processOwnedByCurrentUID(pid) {
			continue
		}
		executable, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue
		}
		if !matchingHostExecutable(selfExecutable, executable) {
			continue
		}
		data, err := os.ReadFile("/proc/" + entry.Name() + "/cmdline")
		if err != nil {
			continue
		}
		runtimeID, ok := parseControlHostRuntime(string(data))
		if !ok {
			continue
		}
		if envBytes, envErr := os.ReadFile("/proc/" + entry.Name() + "/environ"); envErr == nil {
			scope := parseHostEnvScope(envBytes)
			if !hostScopeMatches(scope, curDataDir, curStateDir, explicitIsolated) {
				continue
			}
		} else if explicitIsolated {
			continue
		}
		hosts = append(hosts, HostProcess{PID: pid, RuntimeID: runtimeID, Executable: normalizeProcExePath(executable)})
	}
	return hosts
}
