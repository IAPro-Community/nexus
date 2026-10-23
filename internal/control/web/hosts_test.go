package web

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFormatHostsLine(t *testing.T) {
	got := formatHostsLine("127.0.0.1", "nexus.dev")
	if got != "127.0.0.1\tnexus.dev" {
		t.Fatalf("formatHostsLine: got %q", got)
	}
}

func TestNormalizedLine(t *testing.T) {
	cases := map[string]string{
		"  127.0.0.1\tnexus.dev  ": "127.0.0.1\tnexus.dev",
		"\n":                       "",
		"hello":                    "hello",
	}
	for input, want := range cases {
		if got := normalizedLine(input); got != want {
			t.Errorf("normalizedLine(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatHostsLineRoundTrip(t *testing.T) {
	line := formatHostsLine("127.0.0.1", "nexus.dev")
	if !strings.Contains(line, "127.0.0.1") || !strings.Contains(line, "nexus.dev") {
		t.Fatalf("formatHostsLine produced unexpected output: %q", line)
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(line, "\t") {
			t.Fatal("expected tab separator on Windows")
		}
	}
}

func TestIsHostsEntryPresentReadsFile(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root to read /etc/hosts")
	}
	present, err := IsHostsEntryPresent("localhost", "127.0.0.1")
	if err != nil {
		t.Fatalf("IsHostsEntryPresent: %v", err)
	}
	if !present {
		t.Fatal("localhost 127.0.0.1 should be present on every system")
	}
}

func TestEnsureAndRemoveIdempotency(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root to modify /etc/hosts")
	}

	// Ensure
	if err := EnsureNexusHostsEntry(); err != nil {
		t.Fatalf("EnsureNexusHostsEntry: %v", err)
	}
	present, err := IsHostsEntryPresent(nexusHostname, nexusIP)
	if err != nil {
		t.Fatalf("IsHostsEntryPresent: %v", err)
	}
	if !present {
		t.Fatal("nexus.dev should be present after Ensure")
	}

	// Second ensure must be idempotent
	if err := EnsureNexusHostsEntry(); err != nil {
		t.Fatalf("second EnsureNexusHostsEntry: %v", err)
	}

	// Remove
	if err := RemoveNexusHostsEntry(); err != nil {
		t.Fatalf("RemoveNexusHostsEntry: %v", err)
	}
	present, err = IsHostsEntryPresent(nexusHostname, nexusIP)
	if err != nil {
		t.Fatalf("IsHostsEntryPresent after remove: %v", err)
	}
	if present {
		t.Fatal("nexus.dev should be absent after Remove")
	}

	// Second remove must be idempotent
	if err := RemoveNexusHostsEntry(); err != nil {
		t.Fatalf("second RemoveNexusHostsEntry: %v", err)
	}
}

func TestHostsEntryWithCustomHostname(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root to modify /etc/hosts")
	}

	customHost := "nexus-test-local"
	if err := EnsureHostsEntry(customHost, "127.0.0.1"); err != nil {
		t.Fatalf("EnsureHostsEntry: %v", err)
	}
	present, err := IsHostsEntryPresent(customHost, "127.0.0.1")
	if err != nil {
		t.Fatalf("IsHostsEntryPresent: %v", err)
	}
	if !present {
		t.Fatal("custom entry should be present")
	}

	if err := RemoveHostsEntry(customHost, "127.0.0.1"); err != nil {
		t.Fatalf("RemoveHostsEntry: %v", err)
	}
}

func TestHostsEntryMatch(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		ip       string
		hostname string
		want     bool
	}{
		{"space separator", "127.0.0.1 localhost", "127.0.0.1", "localhost", true},
		{"tab separator", "127.0.0.1\tlocalhost", "127.0.0.1", "localhost", true},
		{"several spaces", "127.0.0.1    localhost", "127.0.0.1", "localhost", true},
		{"extra aliases", "127.0.0.1 localhost myhost", "127.0.0.1", "localhost", true},
		{"alias is second", "127.0.0.1 myhost localhost", "127.0.0.1", "localhost", true},
		{"inline comment", "127.0.0.1 localhost # loopback", "127.0.0.1", "localhost", true},
		{"different ip", "127.0.1.1 localhost", "127.0.0.1", "localhost", false},
		{"different hostname", "127.0.0.1 ip6-localhost", "127.0.0.1", "localhost", false},
		{"prefix must not match", "127.0.0.1 localhost2", "127.0.0.1", "localhost", false},
		{"comment line", "# 127.0.0.1 localhost", "127.0.0.1", "localhost", false},
		{"address only", "127.0.0.1", "127.0.0.1", "localhost", false},
		{"empty line", "", "127.0.0.1", "localhost", false},
		{"case insensitive", "127.0.0.1 LOCALHOST", "127.0.0.1", "localhost", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hostsEntryMatch(tc.line, tc.ip, tc.hostname); got != tc.want {
				t.Errorf("hostsEntryMatch(%q, %q, %q) = %v, want %v",
					tc.line, tc.ip, tc.hostname, got, tc.want)
			}
		})
	}
}

func TestStripHostnameFromLine(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		hostname string
		want     string
		wantKeep bool
	}{
		{"sole alias dropped", "127.0.0.1 nexus.dev", "nexus.dev", "", false},
		{"sole alias with tab dropped", "127.0.0.1\tnexus.dev", "nexus.dev", "", false},
		{"keeps other aliases", "127.0.0.1 nexus.dev myhost", "nexus.dev", "127.0.0.1\tmyhost", true},
		{"keeps first alias", "127.0.0.1 myhost nexus.dev", "nexus.dev", "127.0.0.1\tmyhost", true},
		{"unrelated line untouched", "127.0.1.1 myhost", "nexus.dev", "127.0.1.1 myhost", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, keep := stripHostnameFromLine(tc.line, "127.0.0.1", tc.hostname)
			if got != tc.want || keep != tc.wantKeep {
				t.Errorf("stripHostnameFromLine(%q, %q) = (%q, %v), want (%q, %v)",
					tc.line, tc.hostname, got, keep, tc.want, tc.wantKeep)
			}
		})
	}
}

func TestApplyHostsChangeIsIdempotent(t *testing.T) {
	base := []string{
		"# comment",
		"127.0.0.1 localhost",
		"::1 localhost ip6-localhost",
	}
	added := applyHostsChange(base, "nexus.dev", "127.0.0.1", true)
	if !hostsEntryPresent(added, "127.0.0.1", "nexus.dev") {
		t.Fatalf("entry missing after add: %v", added)
	}
	if !hostsEntryPresent(added, "127.0.0.1", "localhost") {
		t.Fatalf("existing alias lost after add: %v", added)
	}

	// Adding again must not duplicate the entry.
	twice := applyHostsChange(added, "nexus.dev", "127.0.0.1", true)
	count := 0
	for _, line := range twice {
		if hostsEntryMatch(line, "127.0.0.1", "nexus.dev") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("entry duplicated or lost after second add: %v", twice)
	}

	removed := applyHostsChange(added, "nexus.dev", "127.0.0.1", false)
	if hostsEntryPresent(removed, "127.0.0.1", "nexus.dev") {
		t.Fatalf("entry still present after remove: %v", removed)
	}
	if !hostsEntryPresent(removed, "127.0.0.1", "localhost") {
		t.Fatalf("unrelated alias lost after remove: %v", removed)
	}
}

func TestApplyHostsChangeKeepsSharedLine(t *testing.T) {
	base := []string{"127.0.0.1 localhost nexus.dev"}
	got := applyHostsChange(base, "nexus.dev", "127.0.0.1", false)
	if len(got) != 1 || got[0] != "127.0.0.1\tlocalhost" {
		t.Fatalf("shared line not preserved: %v", got)
	}
}

func TestHostsFileBackupNotNeeded(t *testing.T) {
	dir := t.TempDir()
	content := "127.0.0.1\tlocalhost\n::1\t\tlocalhost\n"
	path := filepath.Join(dir, "hosts")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "localhost") {
		t.Fatal("expected hosts content to contain localhost")
	}
}
