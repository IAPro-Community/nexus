package nexus

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
)

// LegacyRestorePreview contains only metadata; the raw snapshot never leaves
// the local restore reader. Staging an intent does NOT import semantic memory.
type LegacyRestorePreview struct {
	SchemaVersion     int       `json:"schema_version"`
	ProjectID         string    `json:"project_id"`
	BackupSHA256      string    `json:"backup_sha256"`
	BackupSourceSHA   string    `json:"backup_source_sha256"`
	CurrentSourceSHA  string    `json:"current_source_sha256"`
	SourceDrift       bool      `json:"source_drift"`
	InstallCount      int       `json:"install_count"`
	ProjectCount      int       `json:"project_count"`
	PendingCount      int       `json:"pending_count"`
	RequiresApproval  bool      `json:"requires_approval"`
	CheckedAt         time.Time `json:"checked_at"`
}

type LegacyRestoreIntent struct {
	SchemaVersion   int                  `json:"schema_version"`
	Status          string               `json:"status"`
	IntentID        string               `json:"intent_id"`
	Preview         LegacyRestorePreview `json:"preview"`
	CreatedAt       time.Time            `json:"created_at"`
}

// readLocalLegacyRestoreBackup only accepts local Nexus-created backup paths;
// callers cannot select an arbitrary file from another directory.
func readLocalLegacyRestoreBackup(filePath, expectedSHA256 string) (LegacyMemorySnapshot, error) {
	var zero LegacyMemorySnapshot
	dir, err := coreconfig.DataDir()
	if err != nil { return zero, err }
	backupDir := filepath.Join(dir, "migration-backups")
	absolute, err := filepath.Abs(filePath)
	if err != nil { return zero, err }
	if filepath.Dir(absolute) != filepath.Clean(backupDir) ||
		!strings.HasPrefix(filepath.Base(absolute), "legacy-intelligence-") ||
		filepath.Ext(absolute) != ".json" {
		return zero, errors.New("restore input must be an owned local Nexus memory backup")
	}
	if err := VerifyLegacyMemoryBackup(absolute, expectedSHA256); err != nil {
		return zero, err
	}
	data, err := os.ReadFile(absolute)
	if err != nil { return zero, err }
	var snapshot LegacyMemorySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil { return zero, err }
	if snapshot.ProjectID == "" || snapshot.ProjectPath == "" ||
		snapshot.SchemaVersion != legacyMemoryBackupVersion {
		return zero, errors.New("invalid restore snapshot identity")
	}
	ids := make(map[string]bool)
	for _, item := range snapshot.Install {
		if item.ID == "" || item.Key == "" || ids["INSTALL:"+item.ID] {
			return zero, errors.New("invalid or duplicate INSTALL memory ID")
		}
		ids["INSTALL:"+item.ID] = true
	}
	for _, item := range snapshot.Project {
		if item.ID == "" || item.Key == "" ||
			item.ProjectID != snapshot.ProjectID || ids["PROJECT:"+item.ID] {
			return zero, errors.New("invalid or cross-project memory in restore snapshot")
		}
		ids["PROJECT:"+item.ID] = true
	}
	return snapshot, nil
}

// PreviewLegacyMemoryRestore does not write to either memory owner. It binds
// the backup to the exact current Nexus project path and flags source drift.
func (n *Nexus) PreviewLegacyMemoryRestore(ctx context.Context, projectID, filePath, expectedSHA256 string) (LegacyRestorePreview, error) {
	var zero LegacyRestorePreview
	if err := ctx.Err(); err != nil { return zero, err }
	st, err := n.OpenProject()
	if err != nil { return zero, err }
	project, err := st.GetProject(projectID)
	if err != nil { return zero, err }
	snapshot, err := readLocalLegacyRestoreBackup(filePath, expectedSHA256)
	if err != nil { return zero, err }
	if snapshot.ProjectID != projectID || snapshot.ProjectPath != project.CanonicalPath {
		return zero, errors.New("restore backup project identity or path mismatch")
	}
	backed, err := BuildLegacyMemoryMigrationPreview(snapshot)
	if err != nil { return zero, err }
	current, err := n.captureLegacyMemorySnapshot(ctx, projectID)
	if err != nil { return zero, err }
	currentPreview, err := BuildLegacyMemoryMigrationPreview(current)
	if err != nil { return zero, err }
	return LegacyRestorePreview{
		SchemaVersion: 1, ProjectID: projectID, BackupSHA256: expectedSHA256,
		BackupSourceSHA: backed.SourceDigestSHA256,
		CurrentSourceSHA: currentPreview.SourceDigestSHA256,
		SourceDrift: backed.SourceDigestSHA256 != currentPreview.SourceDigestSHA256,
		InstallCount: backed.InstallCount, ProjectCount: backed.ProjectCount,
		PendingCount: backed.PendingCount, RequiresApproval: true,
		CheckedAt: time.Now().UTC(),
	}, nil
}

// StageLegacyMemoryRestore makes a small, metadata-only audit intent. It
// never copies raw observations into another memory store and never overwrites
// config.json, SQLite or the Maestro canonical store.
func (n *Nexus) StageLegacyMemoryRestore(ctx context.Context, projectID, backupPath, sha string, confirmed bool) (LegacyRestoreIntent, error) {
	var zero LegacyRestoreIntent
	if !confirmed {
		return zero, errors.New("explicit local restore staging confirmation required")
	}
	preview, err := n.PreviewLegacyMemoryRestore(ctx, projectID, backupPath, sha)
	if err != nil { return zero, err }
	dataDir, err := coreconfig.DataDir()
	if err != nil { return zero, err }
	dir := filepath.Join(dataDir, "migration-restore-intents")
	if err = os.MkdirAll(dir, 0700); err != nil { return zero, err }
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return zero, errors.New("restore intent directory must not be a symlink")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil { return zero, err }
	intent := LegacyRestoreIntent{
		SchemaVersion: 1, Status: "STAGED_NO_APPLY",
		IntentID: "restore-" + hex.EncodeToString(nonce),
		Preview: preview, CreatedAt: time.Now().UTC(),
	}
	body, err := json.MarshalIndent(intent, "", "  ")
	if err != nil { return zero, err }
	final := filepath.Join(dir, intent.IntentID+".json")
	tmp := final+".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil { return zero, err }
	defer os.Remove(tmp)
	if _, err = file.Write(body); err != nil { file.Close(); return zero, err }
	if err = file.Sync(); err != nil { file.Close(); return zero, err }
	if err = file.Close(); err != nil { return zero, err }
	if err = os.Rename(tmp, final); err != nil { return zero, err }
	check, err := os.ReadFile(final)
	if err != nil { return zero, err }
	if sha256.Sum256(check) != sha256.Sum256(body) {
		return zero, errors.New("restore intent verification failed")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(final)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			return zero, errors.New("restore intent permissions too broad")
		}
	}
	return intent, nil
}
