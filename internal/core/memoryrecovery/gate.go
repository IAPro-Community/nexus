package memoryrecovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ErrLegacyMemoryWritesDisabled is returned by legacy intelligence-memory
// writers after a verified Maestro canonical memory migration latched the
// gate. Reads stay allowed for historical visibility.
var ErrLegacyMemoryWritesDisabled = errors.New("legacy intelligence memory writes are disabled after Maestro canonical migration")

// LegacyMemoryPhaseMigrated marks the single latching side effect of a
// verified migration: legacy writers fail closed from this point on.
const LegacyMemoryPhaseMigrated = "MIGRATED"

const legacyMemoryGateSchemaVersion = 1

var legacyMemoryDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var legacyMemoryOpIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

// LegacyMemoryMigrationState is the durable fail-closed gate read before
// every legacy intelligence-memory write. The file lives in the Nexus data
// directory, next to the SQLite store.
type LegacyMemoryMigrationState struct {
	SchemaVersion      int    `json:"schemaVersion"`
	Phase              string `json:"phase"`
	MigratedAt         string `json:"migratedAt,omitempty"`
	SourceDigestSHA256 string `json:"sourceDigestSHA256,omitempty"`
}

// LegacyMemoryGatePath returns the gate file location inside dataDir.
func LegacyMemoryGatePath(dataDir string) (string, error) {
	dir := strings.TrimSpace(dataDir)
	if dir == "" {
		return "", errors.New("legacy memory gate requires the data directory")
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve legacy memory gate directory: %w", err)
	}
	return filepath.Join(absolute, "memory-migration-state.json"), nil
}

// ReadLegacyMemoryMigrationState returns the gate state and whether the gate
// file exists. A malformed or unreadable state file fails closed: callers
// must never assume legacy writes are allowed when the gate is damaged.
func ReadLegacyMemoryMigrationState(dataDir string) (LegacyMemoryMigrationState, bool, error) {
	path, err := LegacyMemoryGatePath(dataDir)
	if err != nil {
		return LegacyMemoryMigrationState{}, false, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return LegacyMemoryMigrationState{}, false, nil
	}
	if err != nil {
		return LegacyMemoryMigrationState{}, false, fmt.Errorf("read legacy memory migration gate: %w", err)
	}
	var state LegacyMemoryMigrationState
	if err := json.Unmarshal(raw, &state); err != nil {
		return LegacyMemoryMigrationState{}, true, fmt.Errorf("legacy memory migration gate is malformed: %w", err)
	}
	if state.SchemaVersion != legacyMemoryGateSchemaVersion {
		return LegacyMemoryMigrationState{}, true, fmt.Errorf("legacy memory migration gate has unsupported schema version %d", state.SchemaVersion)
	}
	if state.Phase != "" && state.Phase != LegacyMemoryPhaseMigrated {
		return LegacyMemoryMigrationState{}, true, fmt.Errorf("legacy memory migration gate has unknown phase %q", state.Phase)
	}
	if state.Phase == LegacyMemoryPhaseMigrated {
		if state.MigratedAt == "" || !legacyMemoryDigestPattern.MatchString(state.SourceDigestSHA256) {
			return LegacyMemoryMigrationState{}, true, errors.New("legacy memory migration gate is missing migration identity")
		}
	}
	return state, true, nil
}

// LegacyMemoryWritesDisabled reports whether the fail-closed gate is latched.
// A damaged gate file is an error, never an implicit "writes allowed".
func LegacyMemoryWritesDisabled(dataDir string) (bool, error) {
	state, _, err := ReadLegacyMemoryMigrationState(dataDir)
	if err != nil {
		return false, err
	}
	return state.Phase == LegacyMemoryPhaseMigrated, nil
}

// CloseLegacyMemoryGate latches the fail-closed state permanently. It records
// the source digest of the migrated snapshot and the recovery-journal
// operation id that reached a terminal Verified stage. The lock content is
// re-verified from disk so the gate can only be closed by the holder of the
// active recovery journal.
func CloseLegacyMemoryGate(dataDir, operationID, sourceDigestSHA256 string) error {
	if !legacyMemoryOpIDPattern.MatchString(operationID) {
		return errors.New("invalid legacy memory migration operation id")
	}
	if !legacyMemoryDigestPattern.MatchString(sourceDigestSHA256) {
		return errors.New("invalid legacy memory migration source digest")
	}
	lockPath := filepath.Join(dataDir, "memory-recovery-journal", ".active.lock")
	lock, err := os.ReadFile(lockPath)
	if err != nil {
		return fmt.Errorf("legacy memory gate requires the active recovery journal lock: %w", err)
	}
	if strings.TrimSpace(string(lock)) != operationID {
		return errors.New("legacy memory gate closed under a different recovery journal operation")
	}
	path, err := LegacyMemoryGatePath(dataDir)
	if err != nil {
		return err
	}
	if _, exists, err := ReadLegacyMemoryMigrationState(dataDir); err != nil {
		return err
	} else if exists {
		return errors.New("legacy memory gate is already closed")
	}
	state := LegacyMemoryMigrationState{
		SchemaVersion:      legacyMemoryGateSchemaVersion,
		Phase:              LegacyMemoryPhaseMigrated,
		MigratedAt:         time.Now().UTC().Format(time.RFC3339Nano),
		SourceDigestSHA256: sourceDigestSHA256,
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(dataDir, ".memory-migration-state-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// ReopenLegacyMemoryGate removes the gate file. It exists for the rollback
// path of the migration coordinator and must never be called while a journal
// operation is active: callers first prove there is no active recovery lock.
func ReopenLegacyMemoryGate(dataDir string) error {
	lockPath := filepath.Join(dataDir, "memory-recovery-journal", ".active.lock")
	if _, err := os.Lstat(lockPath); err == nil {
		return errors.New("legacy memory gate cannot be reopened while a recovery journal lock exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	path, err := LegacyMemoryGatePath(dataDir)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reopen legacy memory gate: %w", err)
	}
	return nil
}
