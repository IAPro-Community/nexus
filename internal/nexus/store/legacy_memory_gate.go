package store

import (
	"github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
)

// ErrLegacyMemoryWritesDisabled is returned by legacy intelligence-memory
// writers after a verified Maestro canonical memory migration latched the
// gate. Reads stay allowed for historical visibility.
var ErrLegacyMemoryWritesDisabled = memoryrecovery.ErrLegacyMemoryWritesDisabled

// LegacyMemoryPhaseMigrated marks the latched fail-closed state.
const LegacyMemoryPhaseMigrated = memoryrecovery.LegacyMemoryPhaseMigrated

// LegacyMemoryMigrationState is the durable fail-closed gate read before
// every legacy intelligence-memory write.
type LegacyMemoryMigrationState = memoryrecovery.LegacyMemoryMigrationState

// LegacyMemoryMigrationState returns the durable migration gate state and
// whether it exists. Package nexus uses it to coordinate migration recovery.
func (s *Store) LegacyMemoryMigrationState() (LegacyMemoryMigrationState, bool, error) {
	return memoryrecovery.ReadLegacyMemoryMigrationState(s.memoryFenceDir)
}

// CloseLegacyMemoryGate latches the fail-closed state permanently. The active
// recovery journal lock must still belong to the given operation id.
func (s *Store) CloseLegacyMemoryGate(operationID, sourceDigestSHA256 string) error {
	return memoryrecovery.CloseLegacyMemoryGate(s.memoryFenceDir, operationID, sourceDigestSHA256)
}

// ReopenLegacyMemoryGate removes the gate file. It refuses while a recovery
// journal lock exists.
func (s *Store) ReopenLegacyMemoryGate() error {
	return memoryrecovery.ReopenLegacyMemoryGate(s.memoryFenceDir)
}

// checkLegacyMemoryGate fails closed after a verified Maestro canonical
// memory migration latched the gate in this store directory.
func (s *Store) checkLegacyMemoryGate() error {
	disabled, err := memoryrecovery.LegacyMemoryWritesDisabled(s.memoryFenceDir)
	if err != nil {
		return err
	}
	if disabled {
		return ErrLegacyMemoryWritesDisabled
	}
	return nil
}
