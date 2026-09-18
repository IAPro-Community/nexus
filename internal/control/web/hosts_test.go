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
