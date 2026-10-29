package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func legacyTestSnapshot() LegacyMemorySnapshot {
	return LegacyMemorySnapshot{
		SchemaVersion: legacyMemoryBackupVersion,
		CreatedAt:     time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC),
		ProjectID:     "prj_test",
		ProjectPath:   "/example/project",
		Install: []coreconfig.IntelligenceMemory{
			{ID: "install-1", Key: "quality.testing", Value: "run unit tests"},
			{ID: "install-2", Key: "execution.autonomy", Value: "never push without approval"},
		},
		Project: []store.IntelligenceMemory{
			{ID: "project-1", ProjectID: "prj_test", Key: "quality.testing", Value: "run all tests"},
			{ID: "project-2", ProjectID: "prj_test", Key: "pending.quality.verification", Value: "not promoted"},
			{ID: "project-3", ProjectID: "prj_test", Key: "verification", Value: "api_key=secret-value"},
		},
	}
}

func TestBuildLegacyMemoryMigrationPreviewIsMetadataOnly(t *testing.T) {
	snapshot := legacyTestSnapshot()
	original := snapshot.Project[0].Value
	got, err := BuildLegacyMemoryMigrationPreview(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if got.InstallCount != 2 || got.ProjectCount != 3 ||
		got.PendingCount != 1 || got.ConflictCount != 1 || got.ReviewCount != 4 ||
		!got.RequiresApproval {
		t.Fatalf("unexpected migration gates: %+v", got)
	}
	if len(got.Candidates) != 5 {
		t.Fatalf("expected all legacy records including pending for audit: %+v", got)
	}
	dispositions := map[string]string{}
	for _, candidate := range got.Candidates {
		dispositions[candidate.ID] = candidate.Disposition
	}
	if dispositions["install-1"] != "REVIEW_CONFLICT" ||
		dispositions["project-1"] != "REVIEW_CONFLICT" ||
		dispositions["install-2"] != "REVIEW_GLOBAL_TO_PROJECT" ||
		dispositions["project-2"] != "SKIP_PENDING" ||
		dispositions["project-3"] != "BLOCKED_SENSITIVE_VALUE" {
		t.Fatalf("incorrect scope/sensitivity classification: %#v", dispositions)
	}
	if snapshot.Project[0].Value != original {
		t.Fatal("preview mutated original memory")
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{
		"run all tests", "run unit tests", "never push without approval",
		"not promoted", "api_key=secret-value", "/example/project",
	} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("preview leaked memory text or source path: %q", secret)
		}
	}
}

func TestLegacyMemoryMigrationPreviewDeterministicAndFailsClosed(t *testing.T) {
	snapshot := legacyTestSnapshot()
	a, err := BuildLegacyMemoryMigrationPreview(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.CreatedAt = snapshot.CreatedAt.Add(24 * time.Hour)
	b, err := BuildLegacyMemoryMigrationPreview(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) || len(a.SourceDigestSHA256) != 64 {
		t.Fatalf("preview digest depends on preview timestamp or ordering: %v %v", a, b)
	}
	snapshot.Project[0].Value = "different verification"
	c, err := BuildLegacyMemoryMigrationPreview(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if c.SourceDigestSHA256 == a.SourceDigestSHA256 {
		t.Fatal("source digest must change when memory text changes")
	}
	_, err = BuildLegacyMemoryMigrationPreview(LegacyMemorySnapshot{SchemaVersion: 2, ProjectID: "prj"})
	if err == nil {
		t.Fatal("unsupported snapshot schema accepted")
	}
}

func TestLegacyMemoryBackupPrivateExactSnapshotAndTamperDetection(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dir)
	snapshot := legacyTestSnapshot()
	receipt, err := saveLegacyMemoryBackup(snapshot)
	if err != nil {
		t.Fatalf("private backup failed: %v", err)
	}
	if filepath.Dir(receipt.Path) != filepath.Join(dir, "migration-backups") {
		t.Fatalf("unexpected backup destination: %q", receipt.Path)
	}
	if err := VerifyLegacyMemoryBackup(receipt.Path, receipt.SHA256); err != nil {
		t.Fatalf("backup verification failed: %v", err)
	}
	data, err := os.ReadFile(receipt.Path)
	if err != nil {
		t.Fatal(err)
	}
	var restored LegacyMemorySnapshot
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot, restored) {
		t.Fatalf("backup changed data required for eventual rollback")
	}
	hash := sha256.Sum256(data)
	if receipt.SHA256 != hex.EncodeToString(hash[:]) {
		t.Fatal("receipt checksum differs from backup content")
	}
	info, err := os.Stat(receipt.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("backup readable by group/others: %o", info.Mode().Perm())
	}
	if err := os.WriteFile(receipt.Path, append(data, 'x'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLegacyMemoryBackup(receipt.Path, receipt.SHA256); err == nil {
		t.Fatal("tampered backup accepted")
	}
}

func TestLegacyMemoryBackupRejectsInvalidSchemaAndDigest(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	if _, err := saveLegacyMemoryBackup(LegacyMemorySnapshot{SchemaVersion: 2, ProjectID: "prj"}); err == nil {
		t.Fatal("invalid backup schema accepted")
	}
	if err := VerifyLegacyMemoryBackup(filepath.Join(t.TempDir(), "not-existing"), strings.Repeat("z", 64)); err == nil {
		t.Fatal("invalid digest accepted")
	}
}
