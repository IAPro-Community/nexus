package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxInstallMemoryCASConfigBytes = 16 << 20

// InstallMemoryCASReceipt is an operational checksum only. It contains no
// semantic content and is not a second memory authority.
type InstallMemoryCASReceipt struct {
	SHA256 string
	Bytes  int
}

// ReplaceInstallIntelligenceMemoriesIfUnchanged is a *single-process* CAS
// primitive. It preserves all unrelated config.json keys, including fields
// added in newer versions. It MUST NOT be called as a standalone "restore
// apply": another process using the legacy SaveConfig path does not honor
// this process's configMu, and SQLite cannot share this file transaction.
// A future coordinator must fence ALL writers and journal both sources.
func ReplaceInstallIntelligenceMemoriesIfUnchanged(
	expectedBefore, replacement []IntelligenceMemory,
) (InstallMemoryCASReceipt, error) {
	var zero InstallMemoryCASReceipt
	if err := validateInstallMemorySet(expectedBefore); err != nil {
		return zero, fmt.Errorf("invalid expected INSTALL memory snapshot: %w", err)
	}
	if err := validateInstallMemorySet(replacement); err != nil {
		return zero, fmt.Errorf("invalid replacement INSTALL memory snapshot: %w", err)
	}

	configMu.Lock()
	defer configMu.Unlock()

	dir, err := ConfigDir()
	if err != nil {
		return zero, err
	}
	filePath := filepath.Join(dir, "config.json")
	info, err := os.Lstat(filePath)
	if err != nil {
		return zero, fmt.Errorf("INSTALL restore requires existing config.json: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxInstallMemoryCASConfigBytes {
		return zero, errors.New("INSTALL restore requires a bounded regular config.json file")
	}
	currentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return zero, err
	}
	if len(currentBytes) > maxInstallMemoryCASConfigBytes {
		return zero, errors.New("config.json exceeds INSTALL restore size limit")
	}
	// RawMessage preserves forward-compatible values that Config might not
	// understand. Do not unmarshal into Config and write it back, because
	// that can drop unrelated or unknown settings.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(currentBytes, &fields); err != nil || fields == nil {
		return zero, errors.New("config.json is not a valid JSON object")
	}
	var current []IntelligenceMemory
	if value, ok := fields["intelligence_memories"]; ok && string(value) != "null" {
		if err := json.Unmarshal(value, &current); err != nil {
			return zero, errors.New("invalid existing INSTALL memory records")
		}
	}
	if err := validateInstallMemorySet(current); err != nil {
		return zero, fmt.Errorf("invalid existing INSTALL memory records: %w", err)
	}
	if !sameInstallMemorySet(current, expectedBefore) {
		return zero, errors.New("INSTALL memories changed since preview (CAS precondition failed)")
	}
	encoded, err := json.Marshal(replacement)
	if err != nil {
		return zero, err
	}
	if len(replacement) == 0 {
		// Absence and [] both mean empty INSTALL memories in the config.
		// Keep an explicit empty array to avoid ambiguities in read-back.
		fields["intelligence_memories"] = json.RawMessage("[]")
	} else {
		fields["intelligence_memories"] = encoded
	}
	nextBytes, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return zero, err
	}
	nextBytes = append(nextBytes, '\n')
	if len(nextBytes) > maxInstallMemoryCASConfigBytes {
		return zero, errors.New("resulting config.json exceeds INSTALL restore size limit")
	}

	tmp, err := os.CreateTemp(dir, ".nexus-install-memory-cas-*.tmp")
	if err != nil {
		return zero, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return zero, err
	}
	if _, err := tmp.Write(nextBytes); err != nil {
		tmp.Close()
		return zero, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return zero, err
	}
	if err := tmp.Close(); err != nil {
		return zero, err
	}
	// The rename itself is atomic on supported local file systems. This is
	// NOT a multi-process CAS or a transaction with SQLite.
	if err := os.Rename(tmpPath, filePath); err != nil {
		return zero, fmt.Errorf("replace INSTALL memories atomically: %w", err)
	}
	sum := sha256.Sum256(nextBytes)
	return InstallMemoryCASReceipt{SHA256: hex.EncodeToString(sum[:]), Bytes: len(nextBytes)}, nil
}

func validateInstallMemorySet(rows []IntelligenceMemory) error {
	ids := make(map[string]bool, len(rows))
	keys := make(map[string]bool, len(rows))
	for _, row := range rows {
		id, key := strings.TrimSpace(row.ID), strings.TrimSpace(row.Key)
		if id == "" || key == "" || strings.TrimSpace(row.Value) == "" ||
			id != row.ID || key != row.Key {
			return errors.New("INSTALL memory has invalid id, key or value")
		}
		if ids[id] || keys[key] {
			return errors.New("duplicate INSTALL memory id or key")
		}
		ids[id], keys[key] = true, true
	}
	return nil
}

func sameInstallMemorySet(a, b []IntelligenceMemory) bool {
	if len(a) != len(b) {
		return false
	}
	left := append([]IntelligenceMemory(nil), a...)
	right := append([]IntelligenceMemory(nil), b...)
	sort.Slice(left, func(i, j int) bool { return left[i].ID < left[j].ID })
	sort.Slice(right, func(i, j int) bool { return right[i].ID < right[j].ID })
	for i := range left {
		x, y := left[i], right[i]
		if x.ID != y.ID || x.Key != y.Key || x.Value != y.Value ||
			x.Source != y.Source || x.UpdatedAt != y.UpdatedAt {
			return false
		}
	}
	return true
}
