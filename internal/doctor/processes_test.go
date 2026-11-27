package doctor

import (
	"os"
	"strings"
	"testing"
)

// pageSizeForTest mirrors os.Getpagesize() for the stat fixture.
func pageSizeForTest() int { return os.Getpagesize() }

func TestProcStatRSSParsesLinuxStat(t *testing.T) {
	// Fields after comm: state ppid pgrp session tty tpgid flags minflt cminflt
	// majflt cmajflt utime stime cutime cstime priority nice num_threads
	// itrealvalue starttime vsize rss ...
	line := "4242 (nexus-hud) S 1 4242 4242 0 -1 4194304 100 0 0 0 5 5 0 0 20 0 7 0 1234 999999 " +
		"2048 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 3 0 0 0 0 0"
	threads, rss, name, ppid, ok := procStatRSS([]byte(line))
	if !ok {
		t.Fatal("stat line should parse")
	}
	if name != "nexus-hud" {
		t.Fatalf("name = %q, want nexus-hud", name)
	}
	if ppid != 1 {
		t.Fatalf("ppid = %d, want 1", ppid)
	}
	if threads != 7 {
		t.Fatalf("threads = %d, want 7", threads)
	}
	wantRSS := int64(2048) * int64(pageSizeForTest())
	if rss != wantRSS {
		t.Fatalf("rss = %d, want %d", rss, wantRSS)
	}
}

func TestProcStatRSSRejectsTruncated(t *testing.T) {
	if _, _, _, _, ok := procStatRSS([]byte("1 (x) S 1")); ok {
		t.Fatal("truncated stat line must not parse")
	}
}

func TestRoleHintDistinguishesSurfacesAndShells(t *testing.T) {
	cases := []struct {
		name    string
		cmdline string
		want    string
	}{
		{"nexus-desktop", "/opt/nexus/nexus-desktop --linked", "desktop"},
		{"nexus-hud", "/opt/nexus/nexus-hud --linked", "hud"},
		{"nexus", "/usr/local/bin/nexus web --port=13000 --no-open", "core"},
		{"nexus", "/usr/local/bin/nexus __control-host --runtime abc", "control-host"},
		{"nexus", "/usr/local/bin/nexus __browser-mcp-stdio", "browser-mcp"},
		{"nexus", "/usr/local/bin/nexus doctor", "cli"},
		{"WebKitWebProcess", "/usr/lib/x86_64-linux-gnu/webkit2gtk/WebKitWebProcess 42 7", "webkit"},
		{"WebKitNetworkProcess", "WebKitNetworkProcess", "webkit"},
		{"bash", "/bin/bash -lc cd /home/user/nexus && npm run build", ""},
		{"node", "/usr/bin/node /home/user/nexus/web/scripts/build.mjs", ""},
	}
	for _, tc := range cases {
		if got := roleHint(tc.name, tc.cmdline); got != tc.want {
			t.Errorf("roleHint(%q, %q) = %q, want %q", tc.name, tc.cmdline, got, tc.want)
		}
	}
}

func TestSummarizeSurfacesRanksAndFormatsRoles(t *testing.T) {
	summary := summarizeSurfaces(map[string]int64{
		"desktop": 220 * 1024 * 1024,
		"webkit":  613 * 1024 * 1024,
	})
	if !strings.HasPrefix(summary, "desktop=220.0MiB webkit=613.0MiB") {
		t.Fatalf("unexpected summary %q", summary)
	}
	if summarizeSurfaces(nil) != "no local Nexus surface is running" {
		t.Fatalf("empty roles must report an idle surface")
	}
}

func TestInspectProcessesIsReadOnlyAndPlatformAware(t *testing.T) {
	report := InspectProcesses()
	if !report.Supported {
		if report.SurfaceSummary == "" {
			t.Fatal("unsupported platform must explain why")
		}
		return
	}
	if report.Source != "/proc" {
		t.Fatalf("source = %q, want /proc", report.Source)
	}
	if report.RoleRSSBytes == nil {
		t.Fatal("role map must be initialized")
	}
	var total int64
	for _, process := range report.Processes {
		if process.PID <= 0 || process.Role == "" {
			t.Fatalf("invalid sample %+v", process)
		}
		if process.Name == "bash" || process.Name == "sh" {
			t.Fatalf("shell processes must not be reported as Nexus surfaces: %+v", process)
		}
		total += process.RSSBytes
	}
	if total != report.TotalRSSBytes {
		t.Fatalf("total %d does not match samples %d", report.TotalRSSBytes, total)
	}
	for _, name := range report.DefunctNames {
		if strings.TrimSpace(name) == "" {
			t.Fatal("defunct names must identify the process")
		}
	}
}
