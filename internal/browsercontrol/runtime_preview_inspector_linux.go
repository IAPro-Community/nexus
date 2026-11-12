//go:build linux

package browsercontrol

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

type linuxRuntimePreviewInspector struct {
	lookup func(string) (registry.RuntimeSession, bool)
}

// NewSystemRuntimePreviewInspector returns the Linux /proc-backed process and
// listener inspector. Other operating systems fail closed until their
// equivalent ownership check is implemented.
func NewSystemRuntimePreviewInspector() RuntimePreviewInspector {
	return linuxRuntimePreviewInspector{lookup: func(runtimeID string) (registry.RuntimeSession, bool) {
		return registry.DefaultRegistry().Get(runtimeID)
	}}
}

func newLinuxRuntimePreviewInspector(lookup func(string) (registry.RuntimeSession, bool)) RuntimePreviewInspector {
	return linuxRuntimePreviewInspector{lookup: lookup}
}

func (i linuxRuntimePreviewInspector) RuntimeSession(runtimeID string) (registry.RuntimeSession, bool) {
	if i.lookup == nil {
		return registry.RuntimeSession{}, false
	}
	session, ok := i.lookup(runtimeID)
	if !ok || session.RuntimeID != runtimeID || !session.IsActive() || !session.HostLive() {
		return registry.RuntimeSession{}, false
	}
	if session.HostGeneration <= 0 || session.ProviderGeneration <= 0 {
		return registry.RuntimeSession{}, false
	}
	if !linuxProcessMatchesGeneration(session.HostPID, session.HostGeneration) || !linuxProcessMatchesGeneration(session.PID, session.ProviderGeneration) {
		return registry.RuntimeSession{}, false
	}
	return session, true
}

func (linuxRuntimePreviewInspector) ProcessGeneration(processID int) (string, bool) {
	stat, err := readLinuxProcStat(processID)
	if err != nil {
		return "", false
	}
	bootID, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(bootID)) + ":" + strconv.FormatUint(stat.startTicks, 10), true
}

func (linuxRuntimePreviewInspector) IsRuntimeDescendant(session registry.RuntimeSession, processID int) bool {
	if processID <= 0 || session.HostPID <= 0 || session.PID <= 0 {
		return false
	}
	if !linuxProcessMatchesGeneration(session.HostPID, session.HostGeneration) || !linuxProcessMatchesGeneration(session.PID, session.ProviderGeneration) {
		return false
	}
	provider, err := readLinuxProcStat(session.PID)
	if err != nil {
		return false
	}
	host, err := readLinuxProcStat(session.HostPID)
	if err != nil {
		return false
	}
	candidate := processID
	type lineageNode struct {
		pid  int
		stat linuxProcStat
	}
	lineage := make([]lineageNode, 0, 16)
	reachedRoot := false
	for depth := 0; depth < 128; depth++ {
		if candidate == session.PID || candidate == session.HostPID {
			reachedRoot = true
			break
		}
		stat, err := readLinuxProcStat(candidate)
		if err != nil || stat.parentPID <= 0 || stat.parentPID == candidate {
			return false
		}
		lineage = append(lineage, lineageNode{pid: candidate, stat: stat})
		candidate = stat.parentPID
	}
	if !reachedRoot {
		return false
	}
	// Re-read the complete chain and roots so PID reuse or reparenting during
	// traversal cannot forge a supervised descendant relationship.
	currentProvider, err := readLinuxProcStat(session.PID)
	if err != nil || currentProvider.startTicks != provider.startTicks {
		return false
	}
	currentHost, err := readLinuxProcStat(session.HostPID)
	if err != nil || currentHost.startTicks != host.startTicks {
		return false
	}
	for _, node := range lineage {
		current, err := readLinuxProcStat(node.pid)
		if err != nil || current.startTicks != node.stat.startTicks || current.parentPID != node.stat.parentPID {
			return false
		}
	}
	return true
}

func (linuxRuntimePreviewInspector) OwnsListeningPort(processID, port int) bool {
	if processID <= 0 || port <= 0 || port > 65535 {
		return false
	}
	stat, err := readLinuxProcStat(processID)
	if err != nil {
		return false
	}
	inodes, err := linuxSocketInodes(processID)
	if err != nil || len(inodes) == 0 {
		return false
	}
	for _, table := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if linuxListenerTableHas(table, port, inodes) {
			current, err := readLinuxProcStat(processID)
			return err == nil && current.startTicks == stat.startTicks
		}
	}
	return false
}

func (i linuxRuntimePreviewInspector) FindRuntimeListener(session registry.RuntimeSession, port int) (int, string, bool) {
	if port <= 0 || port > 65535 || session.RuntimeID == "" || !session.IsActive() || !session.HostLive() {
		return 0, "", false
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, "", false
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || !i.IsRuntimeDescendant(session, pid) || !i.OwnsListeningPort(pid, port) {
			continue
		}
		generation, ok := i.ProcessGeneration(pid)
		if ok && generation != "" {
			return pid, generation, true
		}
	}
	return 0, "", false
}

type linuxProcStat struct {
	parentPID  int
	startTicks uint64
}

func readLinuxProcStat(pid int) (linuxProcStat, error) {
	if pid <= 0 {
		return linuxProcStat{}, os.ErrInvalid
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return linuxProcStat{}, err
	}
	// comm is parenthesized but may itself contain spaces and parentheses.
	rParen := strings.LastIndexByte(string(b), ')')
	if rParen < 0 || rParen+1 >= len(b) {
		return linuxProcStat{}, errors.New("malformed proc stat")
	}
	fields := strings.Fields(string(b[rParen+1:]))
	// The suffix starts at field 3 (state); PPID is field 4 and starttime 22.
	if len(fields) < 20 {
		return linuxProcStat{}, errors.New("short proc stat")
	}
	parentPID, err := strconv.Atoi(fields[1])
	if err != nil {
		return linuxProcStat{}, err
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return linuxProcStat{}, err
	}
	return linuxProcStat{parentPID: parentPID, startTicks: startTicks}, nil
}

func linuxProcessMatchesGeneration(pid int, recordedGeneration int64) bool {
	if recordedGeneration <= 0 {
		return false
	}
	stat, err := readLinuxProcStat(pid)
	if err != nil {
		return false
	}
	bootNS, err := linuxBootTimeNano()
	if err != nil {
		return false
	}
	// /proc/uptime has centisecond precision and Linux USER_HZ is 100 on the
	// supported distributions. A 250ms allowance covers sampling/tick error but
	// rejects a PID recycled after the runtime recorded its generation.
	startNS := bootNS + int64(stat.startTicks)*10_000_000
	return startNS <= recordedGeneration+250_000_000
}

func linuxBootTimeNano() (int64, error) {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0, errors.New("uptime unavailable")
	}
	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || uptime < 0 {
		return 0, errors.New("invalid uptime")
	}
	return time.Now().UnixNano() - int64(uptime*1_000_000_000), nil
}

func linuxSocketInodes(pid int) (map[string]struct{}, error) {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return nil, err
	}
	inodes := make(map[string]struct{})
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "fd", entry.Name()))
		if err != nil {
			continue
		}
		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			inodes[strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")] = struct{}{}
		}
	}
	return inodes, nil
}

func linuxListenerTableHas(path string, port int, inodes map[string]struct{}) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() { // header
		return false
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 || fields[3] != "0A" { // TCP_LISTEN
			continue
		}
		local := strings.SplitN(fields[1], ":", 2)
		if len(local) != 2 {
			continue
		}
		portValue, err := strconv.ParseUint(local[1], 16, 16)
		if err != nil || int(portValue) != port {
			continue
		}
		ip, err := decodeLinuxProcAddress(local[0])
		if err != nil || !ip.IsLoopback() {
			continue
		}
		if _, ok := inodes[fields[9]]; ok {
			return true
		}
	}
	return false
}

func decodeLinuxProcAddress(raw string) (net.IP, error) {
	b, err := hex.DecodeString(raw)
	if err != nil || (len(b) != net.IPv4len && len(b) != net.IPv6len) {
		return nil, errors.New("invalid proc socket address")
	}
	// /proc/net encodes each native-endian word in little-endian byte order.
	for start := 0; start < len(b); start += 4 {
		b[start], b[start+3] = b[start+3], b[start]
		b[start+1], b[start+2] = b[start+2], b[start+1]
	}
	return net.IP(b), nil
}
