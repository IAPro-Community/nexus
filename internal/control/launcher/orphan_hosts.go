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
)

// HostProcess is a live SessionHost process discovered on this machine.
type HostProcess struct {
	PID        int
	RuntimeID  string
	Executable string
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
	if resolved, resolveErr := filepath.EvalSymlinks(selfExecutable); resolveErr == nil {
		selfExecutable = resolved
	}
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
		executable, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue
		}
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		if filepath.Clean(executable) != filepath.Clean(selfExecutable) {
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
		hosts = append(hosts, HostProcess{PID: pid, RuntimeID: runtimeID, Executable: executable})
	}
	return hosts
}
