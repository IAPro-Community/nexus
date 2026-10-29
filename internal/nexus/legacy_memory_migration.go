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
	"sort"
	"strings"
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	legacyMemoryBackupVersion = 1
	legacyMemoryBackupMaxSize = 16 << 20
)

// LegacyMemorySnapshot contains RAW legacy data strictly for a private local
// backup. It must NEVER be returned from an HTTP handler, logged, or used as
// another semantic memory authority. No import or deletion happens here.
type LegacyMemorySnapshot struct {
	SchemaVersion int                             `json:"schema_version"`
	CreatedAt     time.Time                       `json:"created_at"`
	ProjectID     string                          `json:"project_id"`
	ProjectPath   string                          `json:"project_path"`
	Install       []coreconfig.IntelligenceMemory `json:"install"`
	Project       []store.IntelligenceMemory      `json:"project"`
}

type LegacyMemoryMigrationCandidate struct {
	ID          string `json:"id"`
	LegacyScope string `json:"legacy_scope"`
	Key         string `json:"key"`
	Disposition string `json:"disposition"`
}

// LegacyMemoryMigrationPreview is intentionally metadata-only. It never
// exposes value, source, raw project path or user-authored memory text.
type LegacyMemoryMigrationPreview struct {
	SchemaVersion      int                              `json:"schema_version"`
	ProjectID          string                           `json:"project_id"`
	SourceDigestSHA256 string                           `json:"source_digest_sha256"`
	InstallCount       int                              `json:"install_count"`
	ProjectCount       int                              `json:"project_count"`
	PendingCount       int                              `json:"pending_count"`
	ConflictCount      int                              `json:"conflict_count"`
	ReviewCount        int                              `json:"review_count"`
	Candidates         []LegacyMemoryMigrationCandidate `json:"candidates"`
	RequiresApproval   bool                             `json:"requires_approval"`
}

// LegacyMemoryBackupReceipt is safe to show only to the authenticated local
// operator. The path points to a sensitive local-only 0600 backup.
type LegacyMemoryBackupReceipt struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	Bytes         int    `json:"bytes"`
	SchemaVersion int    `json:"schema_version"`
}

// captureLegacyMemorySnapshot reads both owners' *legacy* stores without
// writing anything. Pending promotion rows are included for a lossless backup
// but explicitly excluded as migration candidates.
func (n *Nexus) captureLegacyMemorySnapshot(ctx context.Context, projectID string) (LegacyMemorySnapshot, error) {
	if err := ctx.Err(); err != nil {
		return LegacyMemorySnapshot{}, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return LegacyMemorySnapshot{}, errors.New("project ID required for legacy memory inventory")
	}
	st, err := n.OpenProject()
	if err != nil {
		return LegacyMemorySnapshot{}, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return LegacyMemorySnapshot{}, fmt.Errorf("legacy memory project not found: %w", err)
	}
	if project.ID != projectID || strings.TrimSpace(project.CanonicalPath) == "" {
		return LegacyMemorySnapshot{}, errors.New("legacy memory project identity is incomplete")
	}
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		return LegacyMemorySnapshot{}, err
	}
	// Unlike ListIntelligenceMemories, the raw Store query preserves pending.*
	// records for an exact rollback snapshot.
	projectRows, err := st.ListIntelligenceMemories(projectID)
	if err != nil {
		return LegacyMemorySnapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return LegacyMemorySnapshot{}, err
	}
	return LegacyMemorySnapshot{
		SchemaVersion: legacyMemoryBackupVersion,
		CreatedAt:     time.Now().UTC(),
		ProjectID:     projectID,
		ProjectPath:   project.CanonicalPath,
		Install:       append([]coreconfig.IntelligenceMemory{}, cfg.IntelligenceMemories...),
		Project:       append([]store.IntelligenceMemory{}, projectRows...),
	}, nil
}

// BuildLegacyMemoryMigrationPreview performs deterministic read-only analysis.
// Every non-pending entry needs explicit owner/scope mapping before conversion.
// In particular INSTALL is global and must never silently become repository
// scope; PROJECT's Nexus ID cannot be assumed equal to Maestro repository ID.
func BuildLegacyMemoryMigrationPreview(snapshot LegacyMemorySnapshot) (LegacyMemoryMigrationPreview, error) {
	if snapshot.SchemaVersion != legacyMemoryBackupVersion || snapshot.ProjectID == "" {
		return LegacyMemoryMigrationPreview{}, errors.New("unsupported legacy memory snapshot")
	}
	basis := struct {
		ProjectID string                          `json:"project_id"`
		Install   []coreconfig.IntelligenceMemory `json:"install"`
		Project   []store.IntelligenceMemory      `json:"project"`
	}{snapshot.ProjectID, snapshot.Install, snapshot.Project}
	payload, err := json.Marshal(basis)
	if err != nil {
		return LegacyMemoryMigrationPreview{}, err
	}
	digest := sha256.Sum256(payload)
	out := LegacyMemoryMigrationPreview{
		SchemaVersion: legacyMemoryBackupVersion, ProjectID: snapshot.ProjectID,
		SourceDigestSHA256: hex.EncodeToString(digest[:]),
		InstallCount: len(snapshot.Install), ProjectCount: len(snapshot.Project),
		Candidates: []LegacyMemoryMigrationCandidate{}, RequiresApproval: true,
	}
	installValues := make(map[string]string)
	projectValues := make(map[string]string)
	for _, m := range snapshot.Install {
		installValues[normalizeMemoryKey(m.Key)] = m.Value
	}
	for _, m := range snapshot.Project {
		if !strings.HasPrefix(m.Key, pendingMemoryPrefix) {
			projectValues[normalizeMemoryKey(m.Key)] = m.Value
		}
	}
	conflictKeys := map[string]bool{}
	for key, value := range installValues {
		if other, ok := projectValues[key]; ok && other != value {
			conflictKeys[key] = true
		}
	}
	out.ConflictCount = len(conflictKeys)
	appendCandidate := func(id, scope, key, value string) {
		normalized := normalizeMemoryKey(key)
		disposition := "REVIEW_SCOPE_MAPPING"
		switch {
		case strings.HasPrefix(normalized, pendingMemoryPrefix):
			disposition = "SKIP_PENDING"
			out.PendingCount++
		case normalized == "" || !IsHabitMemoryKey(normalized):
			disposition = "REVIEW_UNRECOGNIZED_KEY"
		case LooksLikeSecret(value):
			disposition = "BLOCKED_SENSITIVE_VALUE"
		case conflictKeys[normalized]:
			disposition = "REVIEW_CONFLICT"
		case scope == MemoryScopeInstall:
			disposition = "REVIEW_GLOBAL_TO_PROJECT"
		default:
			disposition = "REVIEW_PROJECT_TO_REPOSITORY"
		}
		if disposition != "SKIP_PENDING" {
			out.ReviewCount++
		}
		out.Candidates = append(out.Candidates, LegacyMemoryMigrationCandidate{
			ID: security.Redact(id), LegacyScope: scope,
			Key: security.Redact(normalized), Disposition: disposition,
		})
	}
	for _, m := range snapshot.Install {
		appendCandidate(m.ID, MemoryScopeInstall, m.Key, m.Value)
	}
	for _, m := range snapshot.Project {
		appendCandidate(m.ID, MemoryScopeProject, m.Key, m.Value)
	}
	sort.Slice(out.Candidates, func(i, j int) bool {
		a, b := out.Candidates[i], out.Candidates[j]
		if a.LegacyScope != b.LegacyScope {
			return a.LegacyScope < b.LegacyScope
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		return a.ID < b.ID
	})
	return out, nil
}

// PreviewLegacyIntelligenceMemoryMigration is safe to call without modifying
// either store and is the mandatory first step before an explicit backup.
func (n *Nexus) PreviewLegacyIntelligenceMemoryMigration(ctx context.Context, projectID string) (LegacyMemoryMigrationPreview, error) {
	snapshot, err := n.captureLegacyMemorySnapshot(ctx, projectID)
	if err != nil {
		return LegacyMemoryMigrationPreview{}, err
	}
	return BuildLegacyMemoryMigrationPreview(snapshot)
}

// BackupLegacyIntelligenceMemories is an explicit local-only action. The
// caller must guard access to it; it is deliberately not connected to an
// HTTP route or triggered by startup/install/update. No Maestro writes occur.
func (n *Nexus) BackupLegacyIntelligenceMemories(ctx context.Context, projectID string) (LegacyMemoryBackupReceipt, error) {
	snapshot, err := n.captureLegacyMemorySnapshot(ctx, projectID)
	if err != nil {
		return LegacyMemoryBackupReceipt{}, err
	}
	return saveLegacyMemoryBackup(snapshot)
}

func saveLegacyMemoryBackup(snapshot LegacyMemorySnapshot) (LegacyMemoryBackupReceipt, error) {
	var zero LegacyMemoryBackupReceipt
	if snapshot.SchemaVersion != legacyMemoryBackupVersion || snapshot.ProjectID == "" {
		return zero, errors.New("invalid legacy backup snapshot")
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return zero, err
	}
	if len(data) > legacyMemoryBackupMaxSize {
		return zero, errors.New("legacy memory backup exceeds size limit")
	}
	dir, err := coreconfig.DataDir()
	if err != nil {
		return zero, err
	}
	dir = filepath.Join(dir, "migration-backups")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return zero, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return zero, errors.New("legacy backup destination is not a regular directory")
	}
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return zero, err
	}
	name := "legacy-intelligence-" + hex.EncodeToString(nonce) + ".json"
	target := filepath.Join(dir, name)
	tmp := target + ".tmp"
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return zero, err
	}
	defer os.Remove(tmp)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return zero, err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return zero, err
	}
	if err = file.Close(); err != nil {
		return zero, err
	}
	if err = os.Rename(tmp, target); err != nil {
		return zero, err
	}
	sum := sha256.Sum256(data)
	receipt := LegacyMemoryBackupReceipt{
		Path: target, SHA256: hex.EncodeToString(sum[:]),
		Bytes: len(data), SchemaVersion: legacyMemoryBackupVersion,
	}
	// Read back from disk before claiming a usable backup.
	if err = VerifyLegacyMemoryBackup(receipt.Path, receipt.SHA256); err != nil {
		return zero, fmt.Errorf("legacy memory backup verification failed: %w", err)
	}
	return receipt, nil
}

// VerifyLegacyMemoryBackup checks a private snapshot without importing it or
// changing source records. It guards future rollback tooling from using a
// truncated, tampered or unsafe backup.
func VerifyLegacyMemoryBackup(filePath, expectedSHA256 string) error {
	if !regexpSHA256(expectedSHA256) {
		return errors.New("invalid legacy backup digest")
	}
	info, err := os.Lstat(filePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > legacyMemoryBackupMaxSize {
		return errors.New("legacy backup must be a bounded regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return errors.New("legacy backup permissions are too broad")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != expectedSHA256 {
		return errors.New("legacy backup checksum mismatch")
	}
	var snapshot LegacyMemorySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("legacy backup is not valid JSON: %w", err)
	}
	if snapshot.SchemaVersion != legacyMemoryBackupVersion ||
		snapshot.ProjectID == "" || snapshot.ProjectPath == "" {
		return errors.New("legacy backup schema invalid")
	}
	return nil
}

func regexpSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
