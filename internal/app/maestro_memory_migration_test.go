package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestMaestroMemoryPreviewDoesNotCreateMissingProjectDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dir)
	err := maestroMemoryMigrationCmd([]string{"memory-preview", "--project-id", "prj_missing"})
	if err == nil || !strings.Contains(err.Error(), "existing Nexus project database required") {
		t.Fatalf("missing database was not rejected: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "nexus.db")); !os.IsNotExist(err) {
		t.Fatalf("inventory unexpectedly created a database: %v", err)
	}
}


func TestParseMaestroRestoreSubcommandsRequireProvenanceAndConfirmation(t *testing.T) {
	sum := strings.Repeat("a", 64)
	valid := [][]string{
		{"memory-restore-preview", "--project-id", "prj_example", "--backup-path", "/tmp/legacy-intelligence-example.json", "--backup-sha256", sum},
		{"memory-restore-stage", "--project-id", "prj_example", "--backup-path", "/tmp/legacy-intelligence-example.json", "--backup-sha256", sum, "--confirm-restore-stage"},
	}
	for _, argv := range valid {
		got, err := parseMaestroMemoryMigrationArgs(argv)
		if err != nil || got.ProjectID != "prj_example" {
			t.Fatalf("valid restore args rejected: %v", err)
		}
	}
	invalid := [][]string{
		{"memory-restore-preview", "--project-id", "prj_example"},
		{"memory-restore-preview", "--project-id", "prj_example", "--backup-path", "/tmp/test.json", "--backup-sha256", "bad"},
		{"memory-restore-preview", "--project-id", "prj_example", "--backup-path", "/tmp/test.json", "--backup-sha256", sum, "--confirm-restore-stage"},
		{"memory-restore-stage", "--project-id", "prj_example", "--backup-path", "/tmp/test.json", "--backup-sha256", sum},
		{"memory-restore-stage", "--project-id", "prj_example", "--backup-path", "/tmp/test.json", "--backup-sha256", sum, "--confirm-local-backup", "--confirm-restore-stage"},
	}
	for _, argv := range invalid {
		if _, err := parseMaestroMemoryMigrationArgs(argv); err == nil {
			t.Fatalf("unsafe restore args accepted: %v", argv)
		}
	}
}
