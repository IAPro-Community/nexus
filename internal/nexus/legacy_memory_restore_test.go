package nexus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func legacyRestoreFixture(t *testing.T) (*Nexus, string, LegacyMemoryBackupReceipt) {
	t.Helper()
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil { t.Fatal(err) }
	project, err := st.CreateProject(store.Project{Name: "Restore Test", CanonicalPath: t.TempDir()})
	if err != nil { t.Fatal(err) }
	if _, err := st.UpsertIntelligenceMemory(store.IntelligenceMemory{
		ProjectID: project.ID, Key: "quality.verification", Value: "make quality", Source: "user",
	}); err != nil { t.Fatal(err) }
	if _, err := st.UpsertIntelligenceMemory(store.IntelligenceMemory{
		ProjectID: project.ID, Key: "pending.quality.testing", Value: "proposal", Source: "composer",
	}); err != nil { t.Fatal(err) }
	backup, err := n.BackupLegacyIntelligenceMemories(context.Background(), project.ID)
	if err != nil { t.Fatal(err) }
	return n, project.ID, backup
}

func TestRestorePreviewChecksProjectIdentityDriftAndKeepsSourcesUntouched(t *testing.T) {
	n, projectID, backup := legacyRestoreFixture(t)
	ctx := context.Background()
	preview, err := n.PreviewLegacyMemoryRestore(ctx, projectID, backup.Path, backup.SHA256)
	if err != nil { t.Fatal(err) }
	if preview.ProjectCount != 2 || preview.PendingCount != 1 || preview.SourceDrift || !preview.RequiresApproval {
		t.Fatalf("unexpected restore preview: %+v", preview)
	}
	st, _ := n.OpenProject()
	if _, err := st.UpsertIntelligenceMemory(store.IntelligenceMemory{
		ProjectID: projectID, Key: "quality.verification", Value: "make regression", Source: "user",
	}); err != nil { t.Fatal(err) }
	drift, err := n.PreviewLegacyMemoryRestore(ctx, projectID, backup.Path, backup.SHA256)
	if err != nil { t.Fatal(err) }
	if !drift.SourceDrift || drift.CurrentSourceSHA == drift.BackupSourceSHA {
		t.Fatalf("drift not identified: %+v", drift)
	}
	rows, err := st.ListIntelligenceMemories(projectID)
	if err != nil || len(rows) != 2 { t.Fatalf("preview unexpectedly altered storage: %v", err) }
}

func TestRestoreStagingIsConfirmedAndMetadataOnly(t *testing.T) {
	n, projectID, backup := legacyRestoreFixture(t)
	ctx := context.Background()
	if _, err := n.StageLegacyMemoryRestore(ctx, projectID, backup.Path, backup.SHA256, false); err == nil {
		t.Fatal("unconfirmed staging accepted")
	}
	intent, err := n.StageLegacyMemoryRestore(ctx, projectID, backup.Path, backup.SHA256, true)
	if err != nil { t.Fatal(err) }
	if intent.Status != "STAGED_NO_APPLY" || intent.Preview.ProjectID != projectID {
		t.Fatalf("staging falsely claimed restoration: %+v", intent)
	}
	dataDir, err := os.ReadDir(filepath.Join(filepath.Dir(filepath.Dir(backup.Path)), "migration-restore-intents"))
	if err != nil || len(dataDir) != 1 { t.Fatalf("expected one audit intent: %v", err) }
	blob, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(backup.Path)), "migration-restore-intents", intent.IntentID+".json"))
	if err != nil { t.Fatal(err) }
	var parsed LegacyRestoreIntent
	if err := json.Unmarshal(blob, &parsed); err != nil { t.Fatal(err) }
	if parsed.Status != "STAGED_NO_APPLY" || parsed.IntentID != intent.IntentID {
		t.Fatal("audited restore intent was not durable")
	}
	if strings.Contains(string(blob), "make quality") || strings.Contains(string(blob), "proposal") {
		t.Fatal("restore intent leaked legacy semantic content")
	}
}

func TestRestoreRejectsWrongDigestOutsideDirectoryOrCrossProjectRows(t *testing.T) {
	n, projectID, backup := legacyRestoreFixture(t)
	ctx := context.Background()
	if _, err := n.PreviewLegacyMemoryRestore(ctx, projectID, backup.Path, strings.Repeat("a", 64)); err == nil {
		t.Fatal("wrong digest accepted")
	}
	other := filepath.Join(t.TempDir(), filepath.Base(backup.Path))
	data, err := os.ReadFile(backup.Path)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(other, data, 0600); err != nil { t.Fatal(err) }
	if _, err := n.PreviewLegacyMemoryRestore(ctx, projectID, other, backup.SHA256); err == nil {
		t.Fatal("unowned backup path accepted")
	}
	// Deliberately tamper a copy of raw backup to cross the project boundary
	// and verify the project-row guard independently from the checksum.
	var snapshot LegacyMemorySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil { t.Fatal(err) }
	snapshot.Project[0].ProjectID = "prj_attacker"
	edited, err := json.Marshal(snapshot)
	if err != nil { t.Fatal(err) }
	file := filepath.Join(t.TempDir(), "legacy-intelligence-tampered.json")
	if err := os.WriteFile(file, edited, 0600); err != nil { t.Fatal(err) }
	if _, err := readLocalLegacyRestoreBackup(file, strings.Repeat("a", 64)); err == nil {
		t.Fatal("tampered backup accepted")
	}
}
