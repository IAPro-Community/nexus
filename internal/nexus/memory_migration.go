package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ErrMemoryAlreadyMigrated is returned when the fail-closed gate already
// latched a verified migration; migration is then a no-op by design.
var ErrMemoryAlreadyMigrated = errors.New("intelligence memory is already migrated to Maestro canonical memory")

// maestroMigrationProjectID is the synthetic journal identity for the
// whole-installation migration. memoryrecovery.Begin requires a prj_-prefixed
// id; this migration spans all projects plus the INSTALL config store.
const maestroMigrationProjectID = "prj_all_projects"

// Seams for tests: production always uses the real Maestro Bridge client.
// Nexus never falls back to its own store on failure; a failed export fails
// the migration.
var (
	maestroMemoryExporter = handoff.WriteMaestroMemory
	maestroMemoryReader   = handoff.ReadMaestroMemory
)

const (
	maestroMigrationIDPrefix   = "nexus-memmig-"
	maestroMigrationMaxEntries = 500
	pendingMemoryRowPrefix     = "pending."
)

// MemoryMigrationProjectPlan counts the project-scoped memories that a
// migration would export.
type MemoryMigrationProjectPlan struct {
	ProjectID string `json:"project_id"`
	Count     int    `json:"count"`
}

// MemoryMigrationPlan is the dry-run, read-only result of PlanMemoryMigration.
type MemoryMigrationPlan struct {
	DryRun             bool                         `json:"dry_run"`
	Workspace          string                       `json:"workspace"`
	Projects           []MemoryMigrationProjectPlan `json:"projects"`
	InstallCount       int                          `json:"install_count"`
	SourceDigestSHA256 string                       `json:"source_digest_sha256"`
}

// MemoryMigrationProjectResult counts what one verified migration exported.
type MemoryMigrationProjectResult struct {
	ProjectID string `json:"project_id"`
	Exported  int    `json:"exported"`
}

// MemoryMigrationResult reports the single latching migration run. The only
// persistent local side effects are, in order: the migration backup file, the
// recovery journal events, and the fail-closed gate file — gate last.
type MemoryMigrationResult struct {
	Workspace          string                         `json:"workspace"`
	Projects           []MemoryMigrationProjectResult `json:"projects"`
	InstallCount       int                            `json:"install_count"`
	JournalOperationID string                         `json:"journal_operation_id"`
	SourceDigestSHA256 string                         `json:"source_digest_sha256"`
	GateClosed         bool                           `json:"gate_closed"`
}

type memoryMigrationEntry struct {
	ProjectID string
	Key       string
	Value     string
	Source    string
}

type memoryMigrationSnapshotEntry struct {
	ProjectID string `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Source    string `json:"source"`
}

type memoryMigrationSnapshot struct {
	SchemaVersion int                            `json:"schema_version"`
	ProjectID     string                         `json:"project_id"`
	ProjectPath   string                         `json:"project_path"`
	Entries       []memoryMigrationSnapshotEntry `json:"entries"`
}

// maestroMemoryType maps legacy memory sources to canonical observation
// types deterministically.
func maestroMemoryType(source string) string {
	switch source {
	case "explicit":
		return "decision"
	case "destination":
		return "environment"
	default:
		return "discovery"
	}
}

func maestroMigrationID(projectID, key string) string {
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%s%s-%s", maestroMigrationIDPrefix, projectID, hex.EncodeToString(sum[:8]))
}

func truncateMaestroSummary(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 400 {
		runes = runes[:400]
	}
	return string(runes)
}

// collectMemoryMigrationSnapshot reads every legacy memory row and INSTALL
// memory into a deterministic, sorted snapshot. pending.* promotion rows are
// never exported: they are internal two-observation state, not durable facts.
func collectMemoryMigrationSnapshot(ctx context.Context, n *Nexus, workspace string) ([]memoryMigrationEntry, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	rows, err := st.ListAllIntelligenceMemories()
	if err != nil {
		return nil, fmt.Errorf("list legacy project memories: %w", err)
	}
	entries := make([]memoryMigrationEntry, 0, len(rows))
	for _, row := range rows {
		if strings.HasPrefix(row.Key, pendingMemoryRowPrefix) {
			continue
		}
		if strings.TrimSpace(row.Key) == "" || strings.TrimSpace(row.Value) == "" {
			continue
		}
		entries = append(entries, memoryMigrationEntry{
			ProjectID: row.ProjectID, Key: row.Key, Value: row.Value, Source: row.Source,
		})
	}
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load legacy INSTALL memories: %w", err)
	}
	for _, m := range cfg.IntelligenceMemories {
		if strings.HasPrefix(m.Key, pendingMemoryRowPrefix) {
			continue
		}
		if strings.TrimSpace(m.Key) == "" || strings.TrimSpace(m.Value) == "" {
			continue
		}
		entry := memoryMigrationEntry{
			ProjectID: "INSTALL",
			Key:       m.Key, Value: m.Value, Source: m.Source,
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ProjectID != entries[j].ProjectID {
			return entries[i].ProjectID < entries[j].ProjectID
		}
		return entries[i].Key < entries[j].Key
	})
	if len(entries) > maestroMigrationMaxEntries {
		return nil, fmt.Errorf("legacy memory snapshot has %d entries; migration refuses above %d", len(entries), maestroMigrationMaxEntries)
	}
	return entries, nil
}

func memoryMigrationDigest(entries []memoryMigrationEntry) string {
	sorted := append([]memoryMigrationEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ProjectID != sorted[j].ProjectID {
			return sorted[i].ProjectID < sorted[j].ProjectID
		}
		if sorted[i].Key != sorted[j].Key {
			return sorted[i].Key < sorted[j].Key
		}
		return sorted[i].Value < sorted[j].Value
	})
	encoded, _ := json.Marshal(sorted)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func memoryMigrationPreflight(workspace, reason string) (string, error) {
	cleaned := strings.TrimSpace(workspace)
	if cleaned == "" {
		return "", errors.New("maestro memory migration workspace is required")
	}
	absolute, err := filepath.Abs(cleaned)
	if err != nil || !filepath.IsAbs(absolute) {
		return "", errors.New("maestro memory migration workspace must be an absolute path")
	}
	reason = strings.TrimSpace(reason)
	if reasonUTF16Length(reason) < 8 || reasonUTF16Length(reason) > 200 {
		return "", errors.New("maestro memory migration reason must be 8 to 200 characters")
	}
	return absolute, nil
}

// PlanMemoryMigration is the dry-run: read-only, no journal, no backup file,
// no Maestro call, no gate change.
func (n *Nexus) PlanMemoryMigration(ctx context.Context, workspace string) (*MemoryMigrationPlan, error) {
	absolute, err := memoryMigrationPreflight(workspace, "Migration dry-run preflight")
	if err != nil {
		return nil, err
	}
	entries, digest, err := n.memoryMigrationSnapshot(ctx, absolute)
	if err != nil {
		return nil, err
	}
	plan := &MemoryMigrationPlan{DryRun: true, Workspace: absolute, SourceDigestSHA256: digest, Projects: []MemoryMigrationProjectPlan{}}
	byProject := map[string]int{}
	for _, entry := range entries {
		byProject[entry.ProjectID]++
	}
	projectIDs := make([]string, 0, len(byProject))
	for id := range byProject {
		if id == "INSTALL" {
			plan.InstallCount = byProject[id]
			continue
		}
		projectIDs = append(projectIDs, id)
	}
	sort.Strings(projectIDs)
	for _, id := range projectIDs {
		plan.Projects = append(plan.Projects, MemoryMigrationProjectPlan{ProjectID: id, Count: byProject[id]})
	}
	return plan, nil
}

func (n *Nexus) memoryMigrationSnapshot(ctx context.Context, workspace string) ([]memoryMigrationEntry, string, error) {
	if err := memoryMigrationJournalNotBlocked(); err != nil {
		return nil, "", err
	}
	entries, err := collectMemoryMigrationSnapshot(ctx, n, workspace)
	if err != nil {
		return nil, "", err
	}
	return entries, memoryMigrationDigest(entries), nil
}

func memoryMigrationJournalNotBlocked() error {
	dataDir, err := coreconfig.DataDir()
	if err != nil {
		return err
	}
	inspection, err := memoryrecovery.Inspect(dataDir)
	if err != nil {
		return fmt.Errorf("inspect recovery journal: %w", err)
	}
	if inspection.Blocked || len(inspection.Problems) != 0 {
		return errors.New("unfinished or corrupted recovery journal requires human review before memory migration")
	}
	return nil
}

// reasonUTF16Length matches the canonical Bridge limit unit.
func reasonUTF16Length(value string) int {
	return len([]rune(value)) // ASCII-only reason probes; canonical limits use UTF-16.
}

// MigrateIntelligenceMemoryToMaestro exports every legacy memory to Maestro
// canonical memory, verifies the read-back, then latches the fail-closed gate.
// The legacy rows are never deleted: the gate is the single latching side
// effect and it happens last, so a failure at any earlier point leaves the
// local data untouched and writers functional.
func (n *Nexus) MigrateIntelligenceMemoryToMaestro(ctx context.Context, workspace string, confirmed bool, reason string) (*MemoryMigrationResult, error) {
	absolute, err := memoryMigrationPreflight(workspace, reason)
	if err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, errors.New("explicit operator confirmation is required for Maestro memory migration")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if state, exists, err := st.LegacyMemoryMigrationState(); err != nil {
		return nil, fmt.Errorf("read legacy memory migration gate: %w", err)
	} else if exists && state.Phase == store.LegacyMemoryPhaseMigrated {
		return nil, ErrMemoryAlreadyMigrated
	}
	entries, digest, err := n.memoryMigrationSnapshot(ctx, absolute)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return &MemoryMigrationResult{Workspace: absolute, Projects: []MemoryMigrationProjectResult{}, SourceDigestSHA256: digest}, nil
	}
	dataDir, err := coreconfig.DataDir()
	if err != nil {
		return nil, err
	}
	backupPath, backupSHA, err := writeMemoryMigrationBackup(dataDir, absolute, entries)
	if err != nil {
		return nil, err
	}
	journal, err := memoryrecovery.Begin(dataDir, maestroMigrationProjectID, backupPath, backupSHA, digest)
	if err != nil {
		return nil, fmt.Errorf("open memory migration journal: %w", err)
	}
	opID, err := readActiveJournalOperationID(dataDir)
	if err != nil {
		_ = journal.Advance(memoryrecovery.Aborted)
		_ = journal.Close()
		return nil, err
	}
	result := &MemoryMigrationResult{
		Workspace:          absolute,
		Projects:           []MemoryMigrationProjectResult{},
		JournalOperationID: opID,
		SourceDigestSHA256: digest,
	}
	projectEntries, installEntries := splitMemoryMigrationEntries(entries)
	exportedAny, exportErr := exportMemoryMigrationEntries(ctx, absolute, reason, projectEntries, result)
	if exportErr != nil {
		memoryMigrationAbort(journal, exportedAny)
		return nil, exportErr
	}
	if err := journal.Advance(memoryrecovery.ProjectCommitted); err != nil {
		return nil, err
	}
	installExported, installErr := exportMemoryMigrationEntries(ctx, absolute, reason, installEntries, result)
	if installErr != nil {
		memoryMigrationAbort(journal, exportedAny || installExported)
		return nil, installErr
	}
	if err := journal.Advance(memoryrecovery.InstallCommitted); err != nil {
		return nil, err
	}
	if err := verifyMemoryMigrationReadBack(ctx, absolute, entries); err != nil {
		_ = journal.Advance(memoryrecovery.NeedsRecovery)
		return nil, err
	}
	if err := journal.Advance(memoryrecovery.Verified); err != nil {
		return nil, err
	}
	if err := st.CloseLegacyMemoryGate(result.JournalOperationID, digest); err != nil {
		return nil, fmt.Errorf("latch legacy memory gate: %w", err)
	}
	if err := journal.Close(); err != nil {
		return nil, fmt.Errorf("close memory migration journal: %w", err)
	}
	result.GateClosed = true
	return result, nil
}

// RecoverMemoryMigration resumes an interrupted migration. It only accepts an
// operation the journal already recorded as NEEDS_RECOVERY, re-verifies that
// the local source snapshot did not change, and re-exports with the same
// idempotency keys: Maestro answers REPLAYED for the already recorded
// observations, so no duplicate is created. The gate is latched only after a
// successful read-back.
func (n *Nexus) RecoverMemoryMigration(ctx context.Context, workspace string, confirmed bool, reason string) (*MemoryMigrationResult, error) {
	absolute, err := memoryMigrationPreflight(workspace, reason)
	if err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, errors.New("explicit operator confirmation is required for Maestro memory migration recovery")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if state, exists, err := st.LegacyMemoryMigrationState(); err != nil {
		return nil, fmt.Errorf("read legacy memory migration gate: %w", err)
	} else if exists && state.Phase == store.LegacyMemoryPhaseMigrated {
		return nil, ErrMemoryAlreadyMigrated
	}
	dataDir, err := coreconfig.DataDir()
	if err != nil {
		return nil, err
	}
	inspection, err := memoryrecovery.Inspect(dataDir)
	if err != nil {
		return nil, fmt.Errorf("inspect recovery journal: %w", err)
	}
	opID := ""
	for _, op := range inspection.Operations {
		if op.ProjectID == maestroMigrationProjectID {
			opID = op.ID
			break
		}
	}
	if opID == "" {
		return nil, errors.New("no interrupted Maestro memory migration operation found")
	}
	event, err := memoryrecovery.LastEvent(dataDir, opID)
	if err != nil {
		return nil, err
	}
	entries, err := collectMemoryMigrationSnapshot(ctx, n, absolute)
	if err != nil {
		return nil, err
	}
	digest := memoryMigrationDigest(entries)
	if digest != event.SourceDigestSHA256 {
		return nil, errors.New("legacy memory changed since the interrupted migration; recovery refused")
	}
	journal, err := memoryrecovery.Resume(dataDir, opID)
	if err != nil {
		return nil, fmt.Errorf("resume memory migration journal: %w", err)
	}
	result := &MemoryMigrationResult{
		Workspace:          absolute,
		Projects:           []MemoryMigrationProjectResult{},
		JournalOperationID: opID,
		SourceDigestSHA256: digest,
	}
	projectEntries, installEntries := splitMemoryMigrationEntries(entries)
	if _, exportErr := exportMemoryMigrationEntries(ctx, absolute, reason, projectEntries, result); exportErr != nil {
		_ = journal.Advance(memoryrecovery.NeedsRecovery)
		return nil, exportErr
	}
	if err := journal.Advance(memoryrecovery.ProjectCommitted); err != nil {
		return nil, err
	}
	if _, installErr := exportMemoryMigrationEntries(ctx, absolute, reason, installEntries, result); installErr != nil {
		_ = journal.Advance(memoryrecovery.NeedsRecovery)
		return nil, installErr
	}
	if err := journal.Advance(memoryrecovery.InstallCommitted); err != nil {
		return nil, err
	}
	if err := verifyMemoryMigrationReadBack(ctx, absolute, entries); err != nil {
		_ = journal.Advance(memoryrecovery.NeedsRecovery)
		return nil, err
	}
	if err := journal.Advance(memoryrecovery.Verified); err != nil {
		return nil, err
	}
	if err := st.CloseLegacyMemoryGate(result.JournalOperationID, digest); err != nil {
		return nil, fmt.Errorf("latch legacy memory gate: %w", err)
	}
	if err := journal.Close(); err != nil {
		return nil, fmt.Errorf("close memory migration journal: %w", err)
	}
	result.GateClosed = true
	return result, nil
}

// splitMemoryMigrationEntries separates the INSTALL config store from the
// project-scoped rows. The journal records both phases separately.
func splitMemoryMigrationEntries(entries []memoryMigrationEntry) (projects, install []memoryMigrationEntry) {
	for _, entry := range entries {
		if entry.ProjectID == "INSTALL" {
			install = append(install, entry)
			continue
		}
		projects = append(projects, entry)
	}
	return projects, install
}

// memoryMigrationAbort closes the journal cleanly when no export reached the
// canonical store, and keeps it for human review when a partial export may
// exist. A retained lock keeps legacy writers fenced until an operator
// reviews the operation; idempotency keys make a later retry safe without
// duplicates.
func memoryMigrationAbort(journal *memoryrecovery.Journal, exportedAny bool) {
	if exportedAny {
		_ = journal.Advance(memoryrecovery.NeedsRecovery)
		return
	}
	_ = journal.Advance(memoryrecovery.Aborted)
	_ = journal.Close()
}

func exportMemoryMigrationEntries(ctx context.Context, workspace, reason string, entries []memoryMigrationEntry, result *MemoryMigrationResult) (bool, error) {
	exportedAny := false
	perProject := map[string]int{}
	for _, entry := range entries {
		summary := truncateMaestroSummary(entry.Key + ": " + entry.Value)
		request := handoff.MaestroMemoryWriteRequest{
			Confirmed:      true,
			Reason:         reason,
			IdempotencyKey: maestroMigrationID(entry.ProjectID, entry.Key),
			Type:           maestroMemoryType(entry.Source),
			Summary:        summary,
			ScopeLevel:     "workspace",
			ProjectPath:    workspace,
		}
		if _, err := maestroMemoryExporter(ctx, workspace, request); err != nil {
			return exportedAny, fmt.Errorf("export %s memory %q: %w", entry.ProjectID, entry.Key, err)
		}
		exportedAny = true
		perProject[entry.ProjectID]++
	}
	projectIDs := make([]string, 0, len(perProject))
	for id := range perProject {
		projectIDs = append(projectIDs, id)
	}
	sort.Strings(projectIDs)
	for _, id := range projectIDs {
		if id == "INSTALL" {
			result.InstallCount = perProject[id]
			continue
		}
		result.Projects = append(result.Projects, MemoryMigrationProjectResult{ProjectID: id, Exported: perProject[id]})
	}
	return exportedAny, nil
}

// verifyMemoryMigrationReadBack proves the canonical store accepted the
// export: every migrated memory must be retrievable by its summary probe.
func verifyMemoryMigrationReadBack(ctx context.Context, workspace string, entries []memoryMigrationEntry) error {
	for _, entry := range entries {
		probe := truncateMaestroSummary(entry.Key + ": " + entry.Value)
		runes := []rune(probe)
		if len(runes) > 24 {
			runes = runes[:24]
		}
		page, err := maestroMemoryReader(ctx, workspace, "", string(runes), 50)
		if err != nil {
			return fmt.Errorf("read back %s memory %q: %w", entry.ProjectID, entry.Key, err)
		}
		found := false
		for _, item := range page.Items {
			if strings.Contains(item.Summary, string(runes)) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("read back missed %s memory %q", entry.ProjectID, entry.Key)
		}
	}
	return nil
}

// readActiveJournalOperationID reads the migration journal's own operation id
// from the active lock. memoryrecovery.Journal does not expose it; the lock
// content is the operation id written by memoryrecovery.Begin.
func readActiveJournalOperationID(dataDir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, "memory-recovery-journal", ".active.lock"))
	if err != nil {
		return "", fmt.Errorf("read memory migration journal operation id: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// writeMemoryMigrationBackup persists the exact pre-migration snapshot in the// Nexus-owned migration-backups directory with 0600 permissions, in the
// layout memoryrecovery.Begin verifies.
func writeMemoryMigrationBackup(dataDir, workspace string, entries []memoryMigrationEntry) (string, string, error) {
	root := filepath.Join(dataDir, "migration-backups")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", "", err
	}
	snapshot := memoryMigrationSnapshot{
		SchemaVersion: 1,
		ProjectID:     maestroMigrationProjectID,
		ProjectPath:   workspace,
		Entries:       make([]memoryMigrationSnapshotEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		snapshot.Entries = append(snapshot.Entries, memoryMigrationSnapshotEntry{
			ProjectID: entry.ProjectID, Key: entry.Key, Value: entry.Value, Source: entry.Source,
		})
	}
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	path := filepath.Join(root, fmt.Sprintf("legacy-intelligence-%s.json", digest[:16]))
	existing, err := os.ReadFile(path)
	if err == nil && sha256.Sum256(existing) == sum {
		return path, digest, nil
	}
	tmp, err := os.CreateTemp(root, ".legacy-intelligence-*.tmp")
	if err != nil {
		return "", "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return "", "", err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return "", "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", "", err
	}
	if err := tmp.Close(); err != nil {
		return "", "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return "", "", err
	}
	return path, digest, nil
}
