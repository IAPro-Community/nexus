package memoryrecovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LastEvent returns the most recent event of a journal operation. Recovery
// coordinators use the recorded digests to prove the source snapshot did not
// change since the interrupted operation.
func LastEvent(dataDir, operationID string) (Event, error) {
	if !safeOp.MatchString(operationID) {
		return Event{}, errors.New("invalid recovery operation id")
	}
	opDir := filepath.Join(dataDir, "memory-recovery-journal", operationID)
	events, err := readEvents(opDir, operationID)
	if err != nil {
		return Event{}, fmt.Errorf("recovery operation history is invalid: %w", err)
	}
	return events[len(events)-1], nil
}

// Resume reopens an existing, blocked journal operation so a recovery
// coordinator can finish it. It never creates a new operation and never
// rewrites history: the existing event chain is re-validated and the caller
// must continue from the recorded stage. The active lock must still belong
// to this operation, which proves the previous process did not release it.
func Resume(dataDir, operationID string) (*Journal, error) {
	if !safeOp.MatchString(operationID) {
		return nil, errors.New("invalid recovery operation id")
	}
	root := filepath.Join(dataDir, "memory-recovery-journal")
	lockPath := filepath.Join(root, ".active.lock")
	lock, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("recovery journal is not locked: %w", err)
	}
	if strings.TrimSpace(string(lock)) != operationID {
		return nil, errors.New("recovery journal lock belongs to another operation")
	}
	opDir := filepath.Join(root, operationID)
	info, err := os.Lstat(opDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("recovery operation directory is missing or unsafe")
	}
	events, err := readEvents(opDir, operationID)
	if err != nil {
		return nil, fmt.Errorf("recovery operation history is invalid: %w", err)
	}
	last := events[len(events)-1]
	if terminal(last.Stage) {
		return nil, errors.New("recovery operation already reached a terminal stage")
	}
	if last.Stage != NeedsRecovery {
		return nil, fmt.Errorf("recovery operation stage %s does not allow recovery resume", last.Stage)
	}
	return &Journal{directory: opDir, lockPath: lockPath, active: true, last: last}, nil
}
