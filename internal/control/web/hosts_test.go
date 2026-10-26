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

func TestHostsEntryMatchUsesTokensAndIgnoresInlineComments(t *testing.T) {
	cases := []struct {
		name string
		line string
		want bool
	}{
		{name: "spaces and aliases", line: "127.0.0.1 localhost nexus.dev", want: true},
		{name: "tabs and inline comment", line: "127.0.0.1\tlocalhost\tnexus.dev # local", want: true},
		{name: "case insensitive hostname", line: "127.0.0.1 NEXUS.DEV", want: true},
		{name: "other address", line: "127.0.1.1 localhost", want: false},
		{name: "prefix is not a match", line: "127.0.0.1 nexus.development", want: false},
		{name: "comment is not an entry", line: "# 127.0.0.1 nexus.dev", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hostsEntryMatch(tc.line, "127.0.0.1", "nexus.dev"); got != tc.want {
				t.Errorf("hostsEntryMatch(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestApplyHostsChangePreservesOtherAliasesAndInlineComment(t *testing.T) {
	lines := []string{
		"# user comment",
		"127.0.0.1 localhost nexus.dev # keep this note",
		"# IAPro Nexus — local development hostname",
		"127.0.0.1 nexus.dev",
	}

	got := applyHostsChange(lines, "nexus.dev", "127.0.0.1", false)
	want := []string{
		"# user comment",
		"127.0.0.1\tlocalhost\t# keep this note",
	}
	if len(got) != len(want) {
		t.Fatalf("removed lines = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("line %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestApplyHostsChangePreservesInlineCommentWhenRemovingOnlyAlias(t *testing.T) {
	lines := []string{
		hostsComment,
		"127.0.0.1 nexus.dev # user-owned note",
	}

	got := applyHostsChange(lines, "nexus.dev", "127.0.0.1", false)
	want := []string{"# user-owned note"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("removed lines = %#v, want %#v", got, want)
	}
}

func TestApplyHostsChangeRecognizesAliasesAndPreservesInput(t *testing.T) {
	lines := []string{"127.0.0.1 localhost\tnexus.dev # user note"}
	got := applyHostsChange(lines, "nexus.dev", "127.0.0.1", true)
	if len(got) != 1 || got[0] != lines[0] {
		t.Fatalf("adding an existing alias changed the file: %#v", got)
	}
	if !hostsEntryPresent(got, "127.0.0.1", "nexus.dev") {
		t.Fatal("existing alias was not detected")
	}
}

func TestApplyHostsChangeAddsMissingEntryOnce(t *testing.T) {
	lines := []string{"127.0.0.1 localhost"}
	added := applyHostsChange(lines, "nexus.dev", "127.0.0.1", true)
	want := []string{
		"127.0.0.1 localhost",
		hostsComment,
		"127.0.0.1\tnexus.dev",
	}
	if len(added) != len(want) {
		t.Fatalf("added lines = %#v, want %#v", added, want)
	}
	for index := range want {
		if added[index] != want[index] {
			t.Errorf("line %d = %q, want %q", index, added[index], want[index])
		}
	}
	if twice := applyHostsChange(added, "nexus.dev", "127.0.0.1", true); len(twice) != len(want) {
		t.Fatalf("second add duplicated the mapping: %#v", twice)
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
