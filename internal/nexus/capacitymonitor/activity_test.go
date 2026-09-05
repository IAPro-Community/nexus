package capacitymonitor

import (
	"context"
	"testing"
)

func TestNormalizeProviderActivity(t *testing.T) {
	tests := []struct {
		name         string
		executable   string
		resolvedPath string
		wantProvider string
		wantActive   bool
	}{
		{
			name:         "codex binary name",
			executable:   "codex",
			resolvedPath: "/usr/local/bin/codex",
			wantProvider: "codex",
			wantActive:   true,
		},
		{
			name:         "agy binary in home directory",
			executable:   "agy",
			resolvedPath: "/home/user/.local/bin/agy",
			wantProvider: "agy",
			wantActive:   true,
		},
		{
			name:         "claude binary",
			executable:   "claude",
			resolvedPath: "/usr/bin/claude",
			wantProvider: "claude",
			wantActive:   true,
		},
		{
			name:         "opencode binary",
			executable:   "opencode",
			resolvedPath: "/opt/opencode/bin/opencode",
			wantProvider: "opencode",
			wantActive:   true,
		},
		{
			name:         "gemini binary",
			executable:   "gemini",
			resolvedPath: "/usr/bin/gemini",
			wantProvider: "gemini",
			wantActive:   true,
		},
		{
			name:         "cursor binary",
			executable:   "cursor",
			resolvedPath: "/usr/share/cursor/cursor",
			wantProvider: "cursor",
			wantActive:   true,
		},
		{
			name:         "windows exe suffix",
			executable:   "codex.exe",
			resolvedPath: "C:\\Program Files\\Codex\\codex.exe",
			wantProvider: "codex",
			wantActive:   true,
		},
		{
			name:         "unrelated process bash",
			executable:   "bash",
			resolvedPath: "/bin/bash",
			wantProvider: "",
			wantActive:   false,
		},
		{
			name:         "unrelated process node",
			executable:   "node",
			resolvedPath: "/usr/bin/node",
			wantProvider: "",
			wantActive:   false,
		},
		{
			name:         "process named with prefix like codex_helper not matching exact",
			executable:   "codex_helper",
			resolvedPath: "/usr/bin/codex_helper",
			wantProvider: "",
			wantActive:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProvider, gotActive := NormalizeProviderActivity(tt.executable, tt.resolvedPath)
			if gotProvider != tt.wantProvider || gotActive != tt.wantActive {
				t.Fatalf("NormalizeProviderActivity(%q, %q) = (%q, %v), want (%q, %v)",
					tt.executable, tt.resolvedPath, gotProvider, gotActive, tt.wantProvider, tt.wantActive)
			}
		})
	}
}

func TestActivityDetectorSnapshotDoesNotAttributeProfiles(t *testing.T) {
	detector := NewActivityDetector()
	if detector == nil {
		t.Fatal("expected non-nil detector")
	}

	snapshot, err := detector.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("detector snapshot failed: %v", err)
	}

	for providerKey := range snapshot {
		// Keys must only be normalized provider IDs
		if providerKey != "codex" && providerKey != "agy" && providerKey != "claude" &&
			providerKey != "opencode" && providerKey != "gemini" && providerKey != "cursor" {
			t.Fatalf("unexpected key in activity detector snapshot: %q", providerKey)
		}
	}
}
