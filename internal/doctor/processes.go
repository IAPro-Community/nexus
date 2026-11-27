package doctor

import (
	"os"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"sort"
	"strconv"
	"strings"
)

// ProcessSample is one locally observed Nexus/WebKit process with its resident
// set size. Read-only: the inspector never signals, kills or reaps anything.
type ProcessSample struct {
	PID       int    `json:"pid"`
	ParentPID int    `json:"ppid"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	RSSBytes  int64  `json:"rss_bytes"`
	Threads   int    `json:"threads"`
}

// ProcessReport summarizes local Nexus surfaces and their descendant renderers
// so memory regressions can be audited with one command (`nexus doctor --json`).
type ProcessReport struct {
	Supported      bool             `json:"supported"`
	Source         string           `json:"source"`
	Processes      []ProcessSample  `json:"processes"`
	TotalRSSBytes  int64            `json:"total_rss_bytes"`
	DefunctCount   int              `json:"defunct_count"`
	DefunctNames   []string         `json:"defunct_names,omitempty"`
	RoleRSSBytes   map[string]int64 `json:"role_rss_bytes"`
	SurfaceSummary string           `json:"surface_summary"`
}

var procStatRSS = func(data []byte) (threads int, rssBytes int64, name string, ppid int, ok bool) {
	text := string(data)
	open := strings.LastIndex(text, ")")
	if open < 0 || open+2 >= len(text) {
		return 0, 0, "", 0, false
	}
	nameField := text[strings.Index(text, "(")+1 : open]
	fields := strings.Fields(text[open+2:])
	// After the comm field the first field is state; ppid is fields[1].
	if len(fields) < 22 {
		return 0, 0, "", 0, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, "", 0, false
	}
	rssPages, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return 0, 0, "", 0, false
	}
	threads, err = strconv.Atoi(fields[17])
	if err != nil {
		threads = 0
	}
	return threads, rssPages * int64(os.Getpagesize()), nameField, ppid, true
}

var webKitWorker = regexp.MustCompile(`^WebKit(Web|Network)Process$`)

// roleHint classifies a process by its command line. Surfaces are matched by
// argv[0]/basename so a user shell containing the word "nexus" is not counted.
func roleHint(name, cmdline string) string {
	base := filepath.Base(strings.TrimSpace(strings.SplitN(cmdline, " ", 2)[0]))
	argv := strings.Split(cmdline, " ")
	switch {
	case base == "nexus-hud":
		return "hud"
	case base == "nexus-desktop":
		return "desktop"
	case base == "nexus":
		if strings.Contains(cmdline, "__control-host") {
			return "control-host"
		}
		if strings.Contains(cmdline, "__browser-mcp") {
			return "browser-mcp"
		}
		if len(argv) > 1 && argv[1] == "web" {
			return "core"
		}
		return "cli"
	case webKitWorker.MatchString(name):
		return "webkit"
	}
	return ""
}

// InspectProcesses reads /proc (Linux) and never modifies process state. On
// other platforms it reports supported=false with the reason instead of failing
// the whole diagnostic run.
func InspectProcesses() ProcessReport {
	report := ProcessReport{Source: "unavailable", RoleRSSBytes: map[string]int64{}}
	if stdruntime.GOOS != "linux" {
		report.SurfaceSummary = "process inspection is implemented for Linux /proc; use ps on " + stdruntime.GOOS
		return report
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		report.SurfaceSummary = "cannot read /proc: " + err.Error()
		return report
	}
	report.Supported = true
	report.Source = "/proc"
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		statData, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue // process exited between ReadDir and ReadFile
		}
		threads, rss, comm, ppid, ok := procStatRSS(statData)
		if !ok {
			continue
		}
		cmdline := ""
		if raw, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline")); err == nil {
			cmdline = strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " "))
		}
		if strings.Contains(cmdline, "<defunct>") || (cmdline == "" && strings.HasPrefix(rssName(statData), "[")) {
			continue
		}
		role := roleHint(comm, cmdline)
		if role == "" {
			continue
		}
		report.Processes = append(report.Processes, ProcessSample{
			PID:       pid,
			ParentPID: ppid,
			Name:      comm,
			Role:      role,
			RSSBytes:  rss,
			Threads:   threads,
		})
	}
	sort.Slice(report.Processes, func(i, j int) bool {
		if report.Processes[i].RSSBytes != report.Processes[j].RSSBytes {
			return report.Processes[i].RSSBytes > report.Processes[j].RSSBytes
		}
		return report.Processes[i].PID < report.Processes[j].PID
	})
	for _, process := range report.Processes {
		report.TotalRSSBytes += process.RSSBytes
		report.RoleRSSBytes[process.Role] += process.RSSBytes
	}
	report.DefunctNames, report.DefunctCount = countDefunct()
	report.SurfaceSummary = summarizeSurfaces(report.RoleRSSBytes)
	return report
}

// rssName extracts the comm field again for the defunct heuristic.
func rssName(data []byte) string {
	text := string(data)
	open := strings.Index(text, "(")
	close := strings.LastIndex(text, ")")
	if open < 0 || close <= open {
		return ""
	}
	return text[open+1 : close]
}

// countDefunct scans every process state and reports zombies by name. A
// non-zero count is the symptom of an unreaped launcher child.
func countDefunct() (names []string, count int) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, 0
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		text := string(data)
		close := strings.LastIndex(text, ")")
		if close < 0 || close+3 >= len(text) {
			continue
		}
		if strings.Fields(text[close+2:])[0] != "Z" {
			continue
		}
		count++
		name := rssName(data)
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, count
}

func summarizeSurfaces(byRole map[string]int64) string {
	if len(byRole) == 0 {
		return "no local Nexus surface is running"
	}
	roles := make([]string, 0, len(byRole))
	for role := range byRole {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	parts := make([]string, 0, len(roles))
	for _, role := range roles {
		parts = append(parts, role+"="+formatMiB(byRole[role]))
	}
	return strings.Join(parts, " ")
}

func formatMiB(bytes int64) string {
	// One decimal keeps the report readable without losing tens of MiB.
	return strconv.FormatFloat(float64(bytes)/1024/1024, 'f', 1, 64) + "MiB"
}
