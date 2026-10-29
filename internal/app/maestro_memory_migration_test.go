package app

import "testing"

func TestParseMaestroMemoryMigrationArgsRequiresExplicitProjectAndBackupApproval(t *testing.T) {
	valid := []struct {
		args []string
		cmd  string
	}{
		{[]string{"memory-preview", "--project-id", "prj_abc123"}, "memory-preview"},
		{[]string{"memory-preview", "--json", "--project-id=prj_abc123"}, "memory-preview"},
		{[]string{"memory-backup", "--project-id", "prj_abc123", "--confirm-local-backup"}, "memory-backup"},
	}
	for _, tc := range valid {
		got, err := parseMaestroMemoryMigrationArgs(tc.args)
		if err != nil || got.Command != tc.cmd || got.ProjectID != "prj_abc123" {
			t.Fatalf("valid memory args rejected: %+v %v", got, err)
		}
	}
	invalid := [][]string{
		{},
		{"memory-backup", "--project-id", "prj_abc123"},
		{"memory-preview", "--confirm-local-backup", "--project-id", "prj_abc123"},
		{"memory-preview"},
		{"memory-preview", "--project-id", "../escape"},
		{"memory-preview", "--project-id", "prj_abc", "extra"},
		{"memory-preview", "--project-id", "prj_abc", "--unknown"},
		{"memory-preview", "--project-id", " prj_abc"},
		{"memory-migrate", "--project-id", "prj_abc123"},
	}
	for _, args := range invalid {
		if got, err := parseMaestroMemoryMigrationArgs(args); err == nil {
			t.Fatalf("unsafe or implicit migration CLI invocation accepted: %+v", got)
		}
	}
}
